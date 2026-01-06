package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/rcarmo/syncthing-kicker/internal/syncthing"
)

type Service struct {
	Settings Settings
	Client   *syncthing.Client
	Logger   *log.Logger
}

func (s *Service) CheckOnce(ctx context.Context) error {
	folders := foldersFromEnv()
	return s.checkSyncStatus(ctx, folders, 0)
}

func (s *Service) Run(ctx context.Context) error {
	pending := make(chan struct{}, 1024)
	defer close(pending)

	if s.Settings.ScanOnStartup {
		s.Logger.Printf("Triggering scan on startup")
		folders := foldersFromEnv()
		if err := s.triggerScans(ctx, folders, pending); err != nil {
			return err
		}
		for folder := range s.Settings.FolderCron {
			if err := s.triggerScans(ctx, []string{folder}, pending); err != nil {
				return err
			}
		}
		if s.Settings.RunOnce {
			return nil
		}
	}

	sched, err := s.buildCronScheduler(pending)
	if err != nil {
		return err
	}
	defer sched.Stop()

	s.Logger.Printf("Scheduler starting")
	sched.Start()

	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) buildCronScheduler(pending chan struct{}) (*cron.Cron, error) {
	opts := []cron.Option{}
	if tz := strings.TrimSpace(s.Settings.CronTimezone); tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
		}
		opts = append(opts, cron.WithLocation(loc))
		s.Logger.Printf("Scheduler timezone: %s", tz)
	}

	// 5-field cron (min hour dom mon dow)
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	c := cron.New(append(opts, cron.WithParser(parser))...)

	if s.Settings.CronExpr != "" {
		folders := foldersFromEnv()
		if _, err := c.AddFunc(s.Settings.CronExpr, func() {
			ctx := context.Background()
			_ = s.triggerScans(ctx, folders, pending)
		}); err != nil {
			return nil, fmt.Errorf("invalid ST_CRON: %w", err)
		}
	}

	for folder, expr := range s.Settings.FolderCron {
		folder := folder
		expr := expr
		if _, err := c.AddFunc(expr, func() {
			ctx := context.Background()
			_ = s.triggerScans(ctx, []string{folder}, pending)
		}); err != nil {
			return nil, fmt.Errorf("invalid ST_FOLDER_CRON expr for %s: %w", folder, err)
		}
	}

	if len(c.Entries()) == 0 {
		return nil, errors.New("No schedules configured (check ST_CRON / ST_FOLDER_CRON).")
	}
	return c, nil
}

func (s *Service) triggerScans(ctx context.Context, folders []string, pending chan struct{}) error {
	for _, folder := range folders {
		folder = strings.TrimSpace(folder)
		if folder == "" {
			continue
		}

		if s.Settings.DryRun {
			s.Logger.Printf("[dry-run] Would trigger scan for folder '%s'", folder)
		} else {
			// Syncthing may hold POST open; keep timeout low and treat timeouts as success.
			_, err := s.Client.PostScan(ctx, folder, 5*time.Second)
			if err != nil {
				// If the context timed out, treat it as non-fatal.
				if errors.Is(err, context.DeadlineExceeded) {
					s.Logger.Printf("Scan trigger for folder '%s' timed out; Syncthing may still be processing", folder)
				} else {
					s.Logger.Printf("Scan trigger failed for folder '%s': %v", folder, err)
				}
			} else {
				s.Logger.Printf("Triggered scan for folder '%s'", folder)
			}
		}

		// Fire-and-forget status check.
		select {
		case pending <- struct{}{}:
		default:
		}
		go func(folder string) {
			defer func() {
				<-pending
			}()
			_ = s.checkSyncStatus(context.Background(), []string{folder}, s.Settings.StatusDelaySec)
		}(folder)
	}
	return nil
}

func (s *Service) checkSyncStatus(ctx context.Context, folders []string, delaySec float64) error {
	if delaySec > 0 {
		t := time.NewTimer(time.Duration(delaySec * float64(time.Second)))
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}

	wantAll := false
	for _, f := range folders {
		if strings.TrimSpace(f) == "*" {
			wantAll = true
			break
		}
	}

	folderIDs := []string{}
	if wantAll {
		cfg, _, err := s.Client.SystemConfig(ctx, 15*time.Second)
		if err != nil {
			s.Logger.Printf("Failed to fetch folder list for wildcard status check: %v", err)
			return nil
		}
		for _, f := range cfg.Folders {
			if strings.TrimSpace(f.ID) != "" {
				folderIDs = append(folderIDs, f.ID)
			}
		}
		if len(folderIDs) == 0 {
			s.Logger.Printf("No folders returned by Syncthing config; nothing to report")
			return nil
		}
	} else {
		for _, f := range folders {
			f = strings.TrimSpace(f)
			if f != "" && f != "*" {
				folderIDs = append(folderIDs, f)
			}
		}
	}

	for _, id := range folderIDs {
		st, _, err := s.Client.FolderStatus(ctx, id, 10*time.Second)
		if err != nil {
			s.Logger.Printf("Folder %s status check failed: %v", id, err)
			continue
		}
		s.Logger.Printf("Folder %s status: state=%s needBytes=%d inSyncBytes=%d", id, st.State, st.NeedBytes, st.InSyncBytes)
	}
	return nil
}

func foldersFromEnv() []string {
	raw := os.Getenv("ST_FOLDERS")
	if strings.TrimSpace(raw) == "" {
		return []string{"*"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
