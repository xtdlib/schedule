package schedule

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Config is a JSON-serializable schedule definition.
type Config struct {
	// Days: "weekdays", "weekends", or list like ["monday","wednesday","friday"]
	Days json.RawMessage `json:"days,omitempty"`
	// TimeRange: ["HH:MM", "HH:MM"]
	TimeRange [2]string `json:"time_range,omitempty"`
	// ExceptDates: ["YYYY-MM-DD", ...]
	ExceptDates []string `json:"except_dates,omitempty"`
	// OnlyDates: ["YYYY-MM-DD", ...]
	OnlyDates []string `json:"only_dates,omitempty"`
	// DateRange: ["YYYY-MM-DD", "YYYY-MM-DD"]
	DateRange [2]string `json:"date_range,omitempty"`
	// Not: inverts the entire schedule
	Not bool `json:"not,omitempty"`
}

var weekdayNames = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

// Parse parses JSON bytes into a Schedule.
func Parse(data []byte) (*Schedule, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("schedule: %w", err)
	}
	return cfg.Schedule()
}

// Schedule converts a Config into a Schedule.
func (c *Config) Schedule() (*Schedule, error) {
	var rules []Rule

	if len(c.Days) > 0 {
		r, err := parseDaysRule(c.Days)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	if c.TimeRange[0] != "" && c.TimeRange[1] != "" {
		rules = append(rules, TimeRange(c.TimeRange[0], c.TimeRange[1]))
	}

	if len(c.ExceptDates) > 0 {
		rules = append(rules, ExceptDates(c.ExceptDates...))
	}

	if len(c.OnlyDates) > 0 {
		rules = append(rules, OnlyDates(c.OnlyDates...))
	}

	if c.DateRange[0] != "" && c.DateRange[1] != "" {
		rules = append(rules, DateRange(c.DateRange[0], c.DateRange[1]))
	}

	s := New(rules...)

	if c.Not {
		return New(Not(RuleFunc(s.Match))), nil
	}

	return s, nil
}

func parseDaysRule(raw json.RawMessage) (Rule, error) {
	// Try as string first: "weekdays" or "weekends"
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch strings.ToLower(s) {
		case "weekdays":
			return Weekdays(), nil
		case "weekends":
			return Weekends(), nil
		default:
			return nil, fmt.Errorf("schedule: unknown days value %q, want \"weekdays\", \"weekends\", or [\"monday\", ...]", s)
		}
	}

	// Try as array: ["monday", "wednesday", "friday"]
	var names []string
	if err := json.Unmarshal(raw, &names); err != nil {
		return nil, fmt.Errorf("schedule: days must be a string or array of strings")
	}

	days := make([]time.Weekday, 0, len(names))
	for _, name := range names {
		d, ok := weekdayNames[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("schedule: unknown day %q", name)
		}
		days = append(days, d)
	}
	return Days(days...), nil
}
