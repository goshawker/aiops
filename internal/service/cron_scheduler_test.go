package service

import (
	"testing"
	"time"
)

func TestShouldRun_Hourly(t *testing.T) {
	s := &CronScheduler{}
	// @hourly should match when minute is 0
	now := time.Date(2026, 6, 3, 14, 0, 0, 0, time.UTC)
	if !s.shouldRun("@hourly", now) {
		t.Error("@hourly should match at minute 0")
	}
	// Should not match when minute is not 0
	now = time.Date(2026, 6, 3, 14, 30, 0, 0, time.UTC)
	if s.shouldRun("@hourly", now) {
		t.Error("@hourly should not match at minute 30")
	}
}

func TestShouldRun_Daily(t *testing.T) {
	s := &CronScheduler{}
	now := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	if !s.shouldRun("@daily", now) {
		t.Error("@daily should match at midnight")
	}
	now = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	if s.shouldRun("@daily", now) {
		t.Error("@daily should not match at noon")
	}
}

func TestShouldRun_Weekly(t *testing.T) {
	s := &CronScheduler{}
	// Sunday = 0
	now := time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC) // Sunday
	if !s.shouldRun("@weekly", now) {
		t.Error("@weekly should match on Sunday at midnight")
	}
	now = time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC) // Wednesday
	if s.shouldRun("@weekly", now) {
		t.Error("@weekly should not match on Wednesday")
	}
}

func TestShouldRun_StandardCron(t *testing.T) {
	s := &CronScheduler{}
	// "*/5 * * * *" should match every 5 minutes
	now := time.Date(2026, 6, 3, 14, 0, 0, 0, time.UTC)
	if !s.shouldRun("*/5 * * * *", now) {
		t.Error("*/5 should match minute 0")
	}
	now = time.Date(2026, 6, 3, 14, 10, 0, 0, time.UTC)
	if !s.shouldRun("*/5 * * * *", now) {
		t.Error("*/5 should match minute 10")
	}
	now = time.Date(2026, 6, 3, 14, 7, 0, 0, time.UTC)
	if s.shouldRun("*/5 * * * *", now) {
		t.Error("*/5 should not match minute 7")
	}
}

func TestShouldRun_WeekdayRange(t *testing.T) {
	s := &CronScheduler{}
	// "0 9 * * 1-5" at 9:00 on weekdays
	// Monday=1
	now := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC) // Monday
	if !s.shouldRun("0 9 * * 1-5", now) {
		t.Error("should match Monday at 9:00")
	}
	// Sunday=0
	now = time.Date(2026, 6, 7, 9, 0, 0, 0, time.UTC) // Sunday
	if s.shouldRun("0 9 * * 1-5", now) {
		t.Error("should not match Sunday at 9:00")
	}
}

func TestShouldRun_InvalidExpression(t *testing.T) {
	s := &CronScheduler{}
	now := time.Now()
	if s.shouldRun("invalid", now) {
		t.Error("invalid expression should return false")
	}
	if s.shouldRun("* * *", now) {
		t.Error("3-field expression should return false")
	}
}

func TestMatchCronField_Wildcard(t *testing.T) {
	if !matchCronField("*", 42, 0, 59) {
		t.Error("* should match any value")
	}
}

func TestMatchCronField_Step(t *testing.T) {
	if !matchCronField("*/5", 0, 0, 59) {
		t.Error("*/5 should match 0")
	}
	if !matchCronField("*/5", 15, 0, 59) {
		t.Error("*/5 should match 15")
	}
	if matchCronField("*/5", 7, 0, 59) {
		t.Error("*/5 should not match 7")
	}
}

func TestMatchCronField_Range(t *testing.T) {
	if !matchCronField("1-5", 3, 0, 59) {
		t.Error("1-5 should match 3")
	}
	if matchCronField("1-5", 7, 0, 59) {
		t.Error("1-5 should not match 7")
	}
}

func TestMatchCronField_List(t *testing.T) {
	if !matchCronField("1,3,5", 3, 0, 59) {
		t.Error("1,3,5 should match 3")
	}
	if matchCronField("1,3,5", 4, 0, 59) {
		t.Error("1,3,5 should not match 4")
	}
}

func TestMatchCronField_Exact(t *testing.T) {
	if !matchCronField("42", 42, 0, 59) {
		t.Error("42 should match 42")
	}
	if matchCronField("42", 43, 0, 59) {
		t.Error("42 should not match 43")
	}
}

func TestMatchCronField_Invalid(t *testing.T) {
	if matchCronField("abc", 0, 0, 59) {
		t.Error("invalid field should return false")
	}
}
