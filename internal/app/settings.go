package app

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type Settings struct {
	APIURL         string
	APIKey         string
	ScanOnStartup  bool
	VerifyTLS      bool
	RequestTimeout float64 // seconds; 0 means default
	RunOnce        bool
	DryRun         bool
	CronExpr       string
	FolderCron     map[string]string
	CronTimezone   string
	StatusDelaySec float64
}

func LoadSettingsFromEnv() (Settings, error) {
	apiURL := os.Getenv("ST_API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:8384"
	}
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		return Settings{}, errors.New("ST_API_URL must not be empty")
	}
	apiURL = strings.TrimRight(apiURL, "/") + "/"

	apiKey := strings.TrimSpace(os.Getenv("ST_API_KEY"))
	if apiKey == "" {
		return Settings{}, errors.New("ST_API_KEY environment variable is required")
	}

	cronExpr := strings.TrimSpace(os.Getenv("ST_CRON"))
	folderCron, err := parseFolderCron(os.Getenv("ST_FOLDER_CRON"))
	if err != nil {
		return Settings{}, err
	}

	if cronExpr == "" && len(folderCron) == 0 {
		return Settings{}, errors.New("Set ST_CRON (global cron schedule) and/or ST_FOLDER_CRON (per-folder schedules).")
	}

	cronTZ := strings.TrimSpace(os.Getenv("CRON_TZ"))
	if cronTZ == "" {
		cronTZ = strings.TrimSpace(os.Getenv("TZ"))
	}
	if cronTZ != "" {
		if _, err := time.LoadLocation(cronTZ); err != nil {
			return Settings{}, fmt.Errorf("invalid CRON_TZ/TZ value: %w", err)
		}
	}

	statusDelaySec := 5.0
	if raw := strings.TrimSpace(getenv("ST_STATUS_DELAY", "5")); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Settings{}, fmt.Errorf("invalid ST_STATUS_DELAY: %w", err)
		}
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return Settings{}, errors.New("ST_STATUS_DELAY must be >= 0 and not NaN or Inf")
		}
		statusDelaySec = v
	}

	verifyTLS := parseBool(getenv("ST_TLS_VERIFY", "true"), true)
	requestTimeout := 0.0
	if raw := strings.TrimSpace(os.Getenv("ST_REQUEST_TIMEOUT")); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return Settings{}, fmt.Errorf("invalid ST_REQUEST_TIMEOUT: %w", err)
		}
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return Settings{}, errors.New("ST_REQUEST_TIMEOUT must be >= 0 and not NaN or Inf")
		}
		requestTimeout = v
	}

	return Settings{
		APIURL:         apiURL,
		APIKey:         apiKey,
		ScanOnStartup:  parseBool(getenv("SCAN_ON_STARTUP", "false"), false),
		VerifyTLS:      verifyTLS,
		RequestTimeout: requestTimeout,
		RunOnce:        parseBool(getenv("RUN_ONCE", "false"), false),
		DryRun:         parseBool(getenv("DRY_RUN", "false"), false),
		CronExpr:       cronExpr,
		FolderCron:     folderCron,
		CronTimezone:   cronTZ,
		StatusDelaySec: statusDelaySec,
	}, nil
}

func getenv(name, def string) string {
	v := os.Getenv(name)
	if v == "" {
		return def
	}
	return v
}

func parseBool(raw string, def bool) bool {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return def
	}
	switch s {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func parseFolderCron(raw string) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, errors.New("Invalid ST_FOLDER_CRON line. Expected 'folderId: <cron expr>'")
		}
		folder := strings.TrimSpace(parts[0])
		expr := strings.TrimSpace(parts[1])
		if folder == "" || expr == "" {
			return nil, errors.New("Invalid ST_FOLDER_CRON line. Expected 'folderId: <cron expr>'")
		}
		if err := validateFolderID(folder); err != nil {
			return nil, err
		}
		out[folder] = expr
	}
	return out, nil
}

func validateFolderID(folder string) error {
	// Syncthing folder IDs are generally simple slugs; reject whitespace and separators
	// that are likely user mistakes or unsafe to pass around.
	if strings.ContainsAny(folder, " \t\r\n,;") {
		return errors.New("Invalid folder ID in ST_FOLDER_CRON")
	}
	if strings.Contains(folder, ":") {
		return errors.New("Invalid folder ID in ST_FOLDER_CRON")
	}
	return nil
}
