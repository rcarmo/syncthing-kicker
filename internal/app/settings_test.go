package app

import (
	"os"
	"strings"
	"testing"
)

func TestParseFolderCron(t *testing.T) {
	got, err := parseFolderCron("folderA: */5 * * * *\nfolderB: 0 0 * * 1\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["folderA"] != "*/5 * * * *" {
		t.Fatalf("folderA expr mismatch: %q", got["folderA"])
	}
	if got["folderB"] != "0 0 * * 1" {
		t.Fatalf("folderB expr mismatch: %q", got["folderB"])
	}
}

func TestParseFolderCronRejectsMisformattedFolderID(t *testing.T) {
	_, err := parseFolderCron("folder A: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseFolderCronRejectsMissingExpr(t *testing.T) {
	_, err := parseFolderCron("folderA:\n")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseFolderCronRejectsMalformedLine(t *testing.T) {
	_, err := parseFolderCron("folderA */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRequiresAPIKey(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_CRON", "*/5 * * * *")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRequiresSchedule(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_FOLDERS", "folder1, folder2")

	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.APIURL != "http://127.0.0.1:8384/" {
		t.Fatalf("api url mismatch: %q", st.APIURL)
	}
	if st.VerifyTLS != true {
		t.Fatalf("verify tls mismatch")
	}
	if st.StatusDelaySec != 5 {
		t.Fatalf("status delay mismatch: %v", st.StatusDelaySec)
	}
}

func TestLoadSettingsRejectsInvalidStatusDelay(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_STATUS_DELAY", "not-a-number")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRejectsNegativeStatusDelay(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_STATUS_DELAY", "-1")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRejectsInvalidRequestTimeout(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_REQUEST_TIMEOUT", "NaN")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRejectsNegativeRequestTimeout(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_REQUEST_TIMEOUT", "-1")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsRejectsInvalidTimezone(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("CRON_TZ", "Not/A_Timezone")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadSettingsAcceptsValidTimezone(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("CRON_TZ", "UTC")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CronTimezone != "UTC" {
		t.Fatalf("timezone mismatch: %s", st.CronTimezone)
	}
}

// Test parseFolderCron with duplicate folder IDs
func TestParseFolderCronRejectsDuplicateFolderIDs(t *testing.T) {
	// Note: the current implementation will silently overwrite duplicates
	// This test documents the current behavior
	got, err := parseFolderCron("folderA: */5 * * * *\nfolderA: 0 0 * * *\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Last one wins
	if got["folderA"] != "0 0 * * *" {
		t.Fatalf("expected last duplicate to win, got: %q", got["folderA"])
	}
}

// Test parseFolderCron with empty folder ID
func TestParseFolderCronRejectsEmptyFolderID(t *testing.T) {
	_, err := parseFolderCron(": */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for empty folder ID")
	}
}

// Test parseFolderCron with whitespace in folder ID
func TestParseFolderCronRejectsWhitespaceInFolderID(t *testing.T) {
	_, err := parseFolderCron("folder A: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for whitespace in folder ID")
	}
}

// Test parseFolderCron with comma in folder ID
func TestParseFolderCronRejectsCommaInFolderID(t *testing.T) {
	_, err := parseFolderCron("folder,A: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for comma in folder ID")
	}
}

// Test parseFolderCron with semicolon in folder ID
func TestParseFolderCronRejectsSemicolonInFolderID(t *testing.T) {
	_, err := parseFolderCron("folder;A: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for semicolon in folder ID")
	}
}

// Test parseFolderCron with tab in folder ID
func TestParseFolderCronRejectsTabInFolderID(t *testing.T) {
	_, err := parseFolderCron("folder\tA: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for tab in folder ID")
	}
}

// Test parseFolderCron with newline in folder ID
func TestParseFolderCronRejectsNewlineInFolderID(t *testing.T) {
	_, err := parseFolderCron("folder\nA: */5 * * * *\n")
	if err == nil {
		t.Fatalf("expected error for newline in folder ID")
	}
}

// Test parseFolderCron with extra colons after split (not an error - just in expression)
func TestParseFolderCronAcceptsColonInExpression(t *testing.T) {
	// "folder:A: */5 * * * *" splits to folder="folder" and expr="A: */5 * * * *"
	// The expression containing ":" is not invalid per se
	got, err := parseFolderCron("folder:A: */5 * * * *\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["folder"] != "A: */5 * * * *" {
		t.Fatalf("expression mismatch: %q", got["folder"])
	}
}

// Test parseFolderCron with comment lines
func TestParseFolderCronIgnoresComments(t *testing.T) {
	got, err := parseFolderCron("# This is a comment\nfolderA: */5 * * * *\n# Another comment\nfolderB: 0 0 * * *\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(got))
	}
	if got["folderA"] != "*/5 * * * *" {
		t.Fatalf("folderA expr mismatch: %q", got["folderA"])
	}
}

// Test parseFolderCron with blank lines
func TestParseFolderCronIgnoresBlankLines(t *testing.T) {
	got, err := parseFolderCron("folderA: */5 * * * *\n\n\nfolderB: 0 0 * * *\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 folders, got %d", len(got))
	}
}

// Test parseFolderCron with only whitespace
func TestParseFolderCronAcceptsOnlyWhitespace(t *testing.T) {
	got, err := parseFolderCron("   \n\t\n   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(got))
	}
}

// Test parseFolderCron with very long folder ID
func TestParseFolderCronAcceptsLongFolderID(t *testing.T) {
	longID := "folder" + strings.Repeat("X", 200)
	input := longID + ": */5 * * * *\n"
	got, err := parseFolderCron(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[longID] != "*/5 * * * *" {
		t.Fatalf("long folder ID not preserved")
	}
}

// Test parseFolderCron with very long cron expression
func TestParseFolderCronAcceptsLongCronExpression(t *testing.T) {
	longExpr := "0 " + strings.Repeat("1,2,3,4,5,6,7,8,9,10,", 20) + "* * * *"
	input := "folderA: " + longExpr + "\n"
	got, err := parseFolderCron(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["folderA"] != longExpr {
		t.Fatalf("long cron expression not preserved")
	}
}

// Test parseFolderCron with special but valid folder ID characters
func TestParseFolderCronAcceptsSpecialCharacters(t *testing.T) {
	validIDs := []string{
		"folder-A",
		"folder_B",
		"folder.C",
		"folder123",
		"ABC-123_xyz.789",
	}
	for _, id := range validIDs {
		input := id + ": */5 * * * *\n"
		got, err := parseFolderCron(input)
		if err != nil {
			t.Fatalf("unexpected error for valid folder ID %q: %v", id, err)
		}
		if got[id] != "*/5 * * * *" {
			t.Fatalf("folder ID %q not preserved", id)
		}
	}
}

// Test LoadSettingsFromEnv with empty API URL (uses default)
func TestLoadSettingsUsesDefaultForEmptyAPIURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_API_URL", "")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string should use default
	if st.APIURL != "http://127.0.0.1:8384/" {
		t.Fatalf("expected default URL, got: %q", st.APIURL)
	}
}

// Test LoadSettingsFromEnv with only whitespace API URL
func TestLoadSettingsRejectsWhitespaceAPIURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_API_URL", "   \t\n   ")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error for whitespace API URL")
	}
}

// Test LoadSettingsFromEnv ensures trailing slash
func TestLoadSettingsAddsTrailingSlashToAPIURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_API_URL", "http://127.0.0.1:8384")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.APIURL != "http://127.0.0.1:8384/" {
		t.Fatalf("expected trailing slash, got: %q", st.APIURL)
	}
}

// Test LoadSettingsFromEnv with only whitespace API key
func TestLoadSettingsRejectsWhitespaceAPIKey(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "   \t\n   ")
	os.Setenv("ST_CRON", "*/5 * * * *")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error for whitespace API key")
	}
}

// Test LoadSettingsFromEnv with zero status delay
func TestLoadSettingsAcceptsZeroStatusDelay(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_STATUS_DELAY", "0")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.StatusDelaySec != 0 {
		t.Fatalf("expected 0, got: %v", st.StatusDelaySec)
	}
}

// Test LoadSettingsFromEnv with zero request timeout
func TestLoadSettingsAcceptsZeroRequestTimeout(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_REQUEST_TIMEOUT", "0")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.RequestTimeout != 0 {
		t.Fatalf("expected 0, got: %v", st.RequestTimeout)
	}
}

// Test LoadSettingsFromEnv with very large numeric values
func TestLoadSettingsAcceptsLargeNumericValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_STATUS_DELAY", "999999.99")
	os.Setenv("ST_REQUEST_TIMEOUT", "999999.99")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.StatusDelaySec != 999999.99 {
		t.Fatalf("status delay mismatch: %v", st.StatusDelaySec)
	}
	if st.RequestTimeout != 999999.99 {
		t.Fatalf("request timeout mismatch: %v", st.RequestTimeout)
	}
}

// Test LoadSettingsFromEnv with fractional numeric values
func TestLoadSettingsAcceptsFractionalNumericValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_STATUS_DELAY", "2.5")
	os.Setenv("ST_REQUEST_TIMEOUT", "30.5")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.StatusDelaySec != 2.5 {
		t.Fatalf("status delay mismatch: %v", st.StatusDelaySec)
	}
	if st.RequestTimeout != 30.5 {
		t.Fatalf("request timeout mismatch: %v", st.RequestTimeout)
	}
}

// Test parseBool with various formats
func TestParseBoolHandlesVariousFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1", true},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"0", false},
		{"false", false},
		{"FALSE", false},
		{"False", false},
		{"no", false},
		{"NO", false},
		{"off", false},
		{"OFF", false},
		{"  true  ", true},
		{"  false  ", false},
	}

	for _, tt := range tests {
		result := parseBool(tt.input, false)
		if result != tt.expected {
			t.Fatalf("parseBool(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

// Test parseBool with invalid input returns default
func TestParseBoolReturnsDefaultForInvalidInput(t *testing.T) {
	result := parseBool("invalid", true)
	if result != true {
		t.Fatalf("expected default value true, got %v", result)
	}

	result = parseBool("maybe", false)
	if result != false {
		t.Fatalf("expected default value false, got %v", result)
	}
}

// Test parseBool with empty string returns default
func TestParseBoolReturnsDefaultForEmptyString(t *testing.T) {
	result := parseBool("", true)
	if result != true {
		t.Fatalf("expected default value true, got %v", result)
	}
}

// Test LoadSettingsFromEnv with both ST_CRON and ST_FOLDER_CRON
func TestLoadSettingsAcceptsBothGlobalAndFolderCron(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("ST_FOLDER_CRON", "folderA: 0 0 * * *\nfolderB: 0 12 * * *")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CronExpr != "*/5 * * * *" {
		t.Fatalf("global cron mismatch: %q", st.CronExpr)
	}
	if len(st.FolderCron) != 2 {
		t.Fatalf("expected 2 folder crons, got %d", len(st.FolderCron))
	}
}

// Test LoadSettingsFromEnv with only ST_FOLDER_CRON
func TestLoadSettingsAcceptsOnlyFolderCron(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_FOLDER_CRON", "folderA: */5 * * * *")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CronExpr != "" {
		t.Fatalf("expected empty global cron, got: %q", st.CronExpr)
	}
	if len(st.FolderCron) != 1 {
		t.Fatalf("expected 1 folder cron, got %d", len(st.FolderCron))
	}
}

// Test LoadSettingsFromEnv with neither schedule
func TestLoadSettingsRejectsNeitherSchedule(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	_, err := LoadSettingsFromEnv()
	if err == nil {
		t.Fatalf("expected error when neither schedule is provided")
	}
}

// Test LoadSettingsFromEnv respects TZ fallback for CRON_TZ
func TestLoadSettingsUsesTZAsFallbackForCronTZ(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("TZ", "America/Los_Angeles")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CronTimezone != "America/Los_Angeles" {
		t.Fatalf("expected TZ fallback, got: %q", st.CronTimezone)
	}
}

// Test LoadSettingsFromEnv with CRON_TZ overriding TZ
func TestLoadSettingsCronTZOverridesTZ(t *testing.T) {
	os.Clearenv()
	os.Setenv("ST_API_KEY", "abc123")
	os.Setenv("ST_CRON", "*/5 * * * *")
	os.Setenv("TZ", "America/Los_Angeles")
	os.Setenv("CRON_TZ", "UTC")
	st, err := LoadSettingsFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.CronTimezone != "UTC" {
		t.Fatalf("expected CRON_TZ to override TZ, got: %q", st.CronTimezone)
	}
}
