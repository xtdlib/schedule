package schedule

import "testing"

func TestParseWeekdays(t *testing.T) {
	s, err := Parse([]byte(`{
		"days": "weekdays",
		"time_range": ["07:00", "19:00"],
		"except_dates": ["2026-01-01"]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if !s.Match(dt("2026-03-16 10:00")) { // Monday
		t.Error("should match weekday in range")
	}
	if s.Match(dt("2026-03-14 10:00")) { // Saturday
		t.Error("should not match weekend")
	}
	if s.Match(dt("2026-01-01 10:00")) { // holiday
		t.Error("should not match holiday")
	}
}

func TestParseSpecificDays(t *testing.T) {
	s, err := Parse([]byte(`{
		"days": ["monday", "friday"],
		"time_range": ["09:00", "17:00"]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if !s.Match(dt("2026-03-16 10:00")) { // Monday
		t.Error("should match Monday")
	}
	if s.Match(dt("2026-03-17 10:00")) { // Tuesday
		t.Error("should not match Tuesday")
	}
}

func TestParseOvernight(t *testing.T) {
	s, err := Parse([]byte(`{
		"days": "weekdays",
		"time_range": ["22:00", "02:00"]
	}`))
	if err != nil {
		t.Fatal(err)
	}

	if !s.Match(dt("2026-03-13 23:00")) { // Friday night
		t.Error("should match Friday night")
	}
	if !s.Match(dt("2026-03-14 01:00")) { // Saturday early (from Friday)
		t.Error("should match Saturday early morning from Friday")
	}
	if s.Match(dt("2026-03-14 22:00")) { // Saturday night
		t.Error("should not match Saturday night")
	}
}

func TestParseError(t *testing.T) {
	_, err := Parse([]byte(`{"days": "invalid"}`))
	if err == nil {
		t.Error("expected error for invalid days")
	}

	_, err = Parse([]byte(`{"days": ["noday"]}`))
	if err == nil {
		t.Error("expected error for invalid day name")
	}
}
