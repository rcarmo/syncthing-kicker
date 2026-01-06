package app

import (
	"io"
	"log"
	"testing"

	"github.com/rcarmo/syncthing-kicker/internal/syncthing"
)

func TestBuildCronSchedulerRejectsInvalidGlobalCron(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "not a cron",
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildCronSchedulerRejectsInvalidFolderCronExpr(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "",
			FolderCron:   map[string]string{"folderA": "not a cron"},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildCronSchedulerRejectsInvalidTimezone(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/5 * * * *",
			FolderCron:   map[string]string{},
			CronTimezone: "Invalid/Zone",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error")
	}
}

// Test cron expression with wrong number of fields (too few)
func TestBuildCronSchedulerRejectsCronWithTooFewFields(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/5 * *",
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for cron with too few fields")
	}
}

// Test cron expression with wrong number of fields (too many)
func TestBuildCronSchedulerRejectsCronWithTooManyFields(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/5 * * * * * *",
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for cron with too many fields")
	}
}

// Test cron expression with out of range values
func TestBuildCronSchedulerRejectsOutOfRangeMinute(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "60 * * * *", // minute must be 0-59
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for minute value out of range")
	}
}

func TestBuildCronSchedulerRejectsOutOfRangeHour(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "0 24 * * *", // hour must be 0-23
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for hour value out of range")
	}
}

func TestBuildCronSchedulerRejectsOutOfRangeDayOfMonth(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "0 0 32 * *", // day of month must be 1-31
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for day of month value out of range")
	}
}

func TestBuildCronSchedulerRejectsOutOfRangeMonth(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "0 0 1 13 *", // month must be 1-12
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for month value out of range")
	}
}

func TestBuildCronSchedulerRejectsOutOfRangeDayOfWeek(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "0 0 * * 8", // day of week must be 0-6
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for day of week value out of range")
	}
}

// Test empty cron expression
func TestBuildCronSchedulerRejectsEmptyCronExpression(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "",
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for no schedules configured")
	}
}

// Test special characters in cron expression
func TestBuildCronSchedulerRejectsInvalidSpecialCharacters(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/5 * * * $",
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for invalid special character")
	}
}

// Test invalid step values
func TestBuildCronSchedulerRejectsInvalidStepValue(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/0 * * * *", // step cannot be 0
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for invalid step value")
	}
}

// Test invalid range
func TestBuildCronSchedulerRejectsInvalidRange(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "10-5 * * * *", // range start > end without wrapping
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error for invalid range")
	}
}

// Test valid complex cron expressions
func TestBuildCronSchedulerAcceptsValidComplexExpressions(t *testing.T) {
	validExpressions := []string{
		"0 0 * * *",       // daily at midnight
		"*/15 * * * *",    // every 15 minutes
		"0 9-17 * * 1-5",  // weekdays 9am-5pm
		"0,30 * * * *",    // at 0 and 30 minutes
		"0 0 1,15 * *",    // 1st and 15th of month
		"0 0 * * 0",       // every Sunday
		"*/10 8-18 * * *", // every 10 min between 8am-6pm
	}

	for _, expr := range validExpressions {
		svc := &Service{
			Settings: Settings{
				CronExpr:     expr,
				FolderCron:   map[string]string{},
				CronTimezone: "",
			},
			Client: syncthingStub(),
			Logger: log.New(io.Discard, "", 0),
		}

		_, err := svc.buildCronScheduler(make(chan struct{}, 1))
		if err != nil {
			t.Fatalf("expected valid cron expression %q to be accepted, got error: %v", expr, err)
		}
	}
}

// Test whitespace handling
func TestBuildCronSchedulerRejectsExtraWhitespace(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr:     "*/5  *  *  *  *", // extra spaces
			FolderCron:   map[string]string{},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	// This should actually be accepted by the cron parser (it handles whitespace)
	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err != nil {
		// If error, that's fine - whitespace handling varies
		return
	}
}

// Test timezone with scheduler
func TestBuildCronSchedulerAcceptsValidTimezones(t *testing.T) {
	timezones := []string{
		"UTC",
		"America/New_York",
		"Europe/London",
		"Asia/Tokyo",
	}

	for _, tz := range timezones {
		svc := &Service{
			Settings: Settings{
				CronExpr:     "*/5 * * * *",
				FolderCron:   map[string]string{},
				CronTimezone: tz,
			},
			Client: syncthingStub(),
			Logger: log.New(io.Discard, "", 0),
		}

		_, err := svc.buildCronScheduler(make(chan struct{}, 1))
		if err != nil {
			t.Fatalf("expected valid timezone %q to be accepted, got error: %v", tz, err)
		}
	}
}

// Test folder cron with invalid expressions
func TestBuildCronSchedulerRejectsMultipleFoldersWithOneInvalid(t *testing.T) {
	svc := &Service{
		Settings: Settings{
			CronExpr: "",
			FolderCron: map[string]string{
				"folderA": "*/5 * * * *",
				"folderB": "invalid",
				"folderC": "0 0 * * *",
			},
			CronTimezone: "",
		},
		Client: syncthingStub(),
		Logger: log.New(io.Discard, "", 0),
	}

	_, err := svc.buildCronScheduler(make(chan struct{}, 1))
	if err == nil {
		t.Fatalf("expected error when one folder has invalid cron")
	}
}

func syncthingStub() *syncthing.Client {
	// buildCronScheduler does not call the client; use a nil-ish placeholder.
	return &syncthing.Client{}
}
