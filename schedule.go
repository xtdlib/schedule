// Package schedule provides composable rules for determining whether
// a given time falls within a defined schedule.
package schedule

import (
	"log/slog"
	"os"
	"syscall"
	"time"
)

// Rule determines whether a given time is active.
type Rule interface {
	Match(t time.Time) bool
}

// RuleFunc adapts a plain function to the Rule interface.
type RuleFunc func(time.Time) bool

func (f RuleFunc) Match(t time.Time) bool { return f(t) }

// timeRangeRule is a concrete type so Schedule can detect overnight ranges.
type timeRangeRule struct {
	fromMin   int
	toMin     int
	overnight bool
}

func (r *timeRangeRule) Match(t time.Time) bool {
	m := t.Hour()*60 + t.Minute()
	if !r.overnight {
		return m >= r.fromMin && m < r.toMin
	}
	return m >= r.fromMin || m < r.toMin
}

// inPostMidnight returns true if t is in the after-midnight portion of an overnight range.
func (r *timeRangeRule) inPostMidnight(t time.Time) bool {
	if !r.overnight {
		return false
	}
	m := t.Hour()*60 + t.Minute()
	return m < r.toMin
}

// Schedule is a set of rules that must all match for a time to be active.
type Schedule struct {
	rules []Rule
}

// New creates a schedule from the given rules. All rules must match.
func New(rules ...Rule) *Schedule {
	return &Schedule{rules: rules}
}

func (s *Schedule) findTimeRange() *timeRangeRule {
	for _, r := range s.rules {
		if tr, ok := r.(*timeRangeRule); ok {
			return tr
		}
	}
	return nil
}

// Match returns true if all rules match t.
// For overnight time ranges (e.g. "22:00"-"02:00"), day/date rules are
// evaluated against the previous day when t is in the post-midnight portion.
func (s *Schedule) Match(t time.Time) bool {
	tr := s.findTimeRange()
	evalTime := t
	if tr != nil && tr.inPostMidnight(t) {
		evalTime = t.AddDate(0, 0, -1)
	}
	for _, r := range s.rules {
		if r == tr {
			if !r.Match(t) {
				return false
			}
		} else {
			if !r.Match(evalTime) {
				return false
			}
		}
	}
	return true
}

// Window returns the start and end time of the active window containing t.
// Returns zero times and false if t is not active.
func (s *Schedule) Window(t time.Time) (start, end time.Time, ok bool) {
	if !s.Match(t) {
		return time.Time{}, time.Time{}, false
	}
	tr := s.findTimeRange()
	if tr == nil {
		y, m, d := t.Date()
		loc := t.Location()
		return time.Date(y, m, d, 0, 0, 0, 0, loc),
			time.Date(y, m, d, 23, 59, 0, 0, loc), true
	}
	loc := t.Location()
	if !tr.overnight {
		y, m, d := t.Date()
		start = time.Date(y, m, d, tr.fromMin/60, tr.fromMin%60, 0, 0, loc)
		end = time.Date(y, m, d, tr.toMin/60, tr.toMin%60, 0, 0, loc)
		return start, end, true
	}
	if tr.inPostMidnight(t) {
		yesterday := t.AddDate(0, 0, -1)
		y, m, d := yesterday.Date()
		start = time.Date(y, m, d, tr.fromMin/60, tr.fromMin%60, 0, 0, loc)
		y, m, d = t.Date()
		end = time.Date(y, m, d, tr.toMin/60, tr.toMin%60, 0, 0, loc)
	} else {
		y, m, d := t.Date()
		start = time.Date(y, m, d, tr.fromMin/60, tr.fromMin%60, 0, 0, loc)
		tomorrow := t.AddDate(0, 0, 1)
		y, m, d = tomorrow.Date()
		end = time.Date(y, m, d, tr.toMin/60, tr.toMin%60, 0, 0, loc)
	}
	return start, end, true
}

// Next returns the next minute at or after t where the schedule becomes active.
// It scans forward up to 400 days. Returns zero time and false if none found.
func (s *Schedule) Next(after time.Time) (time.Time, bool) {
	t := after.Truncate(time.Minute)
	limit := t.Add(400 * 24 * time.Hour)
	for t.Before(limit) {
		if s.Match(t) {
			return t, true
		}
		t = t.Add(time.Minute)
	}
	return time.Time{}, false
}

// ActiveUntil returns the last contiguous minute where the schedule remains
// active starting from t. Returns zero time and false if t is not active.
func (s *Schedule) ActiveUntil(t time.Time) (time.Time, bool) {
	cur := t.Truncate(time.Minute)
	if !s.Match(cur) {
		return time.Time{}, false
	}
	limit := cur.Add(400 * 24 * time.Hour)
	for cur.Before(limit) {
		next := cur.Add(time.Minute)
		if !s.Match(next) {
			return cur, true
		}
		cur = next
	}
	return cur, true
}

// Watch sleeps until the schedule ends, then sends SIGTERM to the process.
// If still alive after 30 seconds, it panics.
// Call once at startup; it runs in the background.
func (s *Schedule) Watch() {
	go func() {
		for {
			now := time.Now()
			until, ok := s.ActiveUntil(now)
			if !ok {
				slog.Info("schedule: outside of active schedule, sending SIGTERM")
				syscall.Kill(os.Getpid(), syscall.SIGTERM)
				time.Sleep(30 * time.Second)
				panic("schedule: process did not exit within 30s after SIGTERM")
			}
			remaining := time.Until(until) / 2
			if remaining < 5*time.Second {
				remaining = 5 * time.Second
			}
			time.Sleep(remaining)
		}
	}()
}

// Wait blocks until the schedule becomes active, checking every second at minimum.
func (s *Schedule) Wait() {
	for !s.Match(time.Now()) {
		next, ok := s.Next(time.Now())
		if !ok {
			time.Sleep(time.Second)
			continue
		}
		wait := time.Until(next) / 2
		if wait < time.Second {
			wait = time.Second
		}
		time.Sleep(wait)
	}
}

// --- Rules ---

// Weekdays matches Monday through Friday.
func Weekdays() Rule {
	return RuleFunc(func(t time.Time) bool {
		d := t.Weekday()
		return d >= time.Monday && d <= time.Friday
	})
}

// Weekends matches Saturday and Sunday.
func Weekends() Rule {
	return RuleFunc(func(t time.Time) bool {
		d := t.Weekday()
		return d == time.Saturday || d == time.Sunday
	})
}

// Days matches only the specified weekdays.
func Days(days ...time.Weekday) Rule {
	set := make(map[time.Weekday]bool, len(days))
	for _, d := range days {
		set[d] = true
	}
	return RuleFunc(func(t time.Time) bool {
		return set[t.Weekday()]
	})
}

// TimeRange matches when the clock time is between from and to (inclusive start,
// exclusive end). Both are in "HH:MM" format. Supports midnight wrap
// (e.g. "22:00" to "02:00"). Panics on invalid format.
func TimeRange(from, to string) Rule {
	fh, fm := parseHHMM(from)
	th, tm := parseHHMM(to)
	fromMin := fh*60 + fm
	toMin := th*60 + tm
	return &timeRangeRule{
		fromMin:   fromMin,
		toMin:     toMin,
		overnight: fromMin > toMin,
	}
}

func parseHHMM(s string) (int, int) {
	if len(s) != 5 || s[2] != ':' {
		panic("schedule: invalid time format " + s + ", want HH:MM")
	}
	h := int(s[0]-'0')*10 + int(s[1]-'0')
	m := int(s[3]-'0')*10 + int(s[4]-'0')
	if h > 23 || m > 59 {
		panic("schedule: invalid time " + s)
	}
	return h, m
}

// ExceptDates excludes specific dates. Accepts "YYYY-MM-DD" strings.
// Panics on invalid format.
func ExceptDates(dates ...string) Rule {
	set := parseDateSet(dates)
	return RuleFunc(func(t time.Time) bool {
		return !set[t.Format(time.DateOnly)]
	})
}

// OnlyDates matches only the specified dates. Accepts "YYYY-MM-DD" strings.
// Panics on invalid format.
func OnlyDates(dates ...string) Rule {
	set := parseDateSet(dates)
	return RuleFunc(func(t time.Time) bool {
		return set[t.Format(time.DateOnly)]
	})
}

func parseDateSet(dates []string) map[string]bool {
	set := make(map[string]bool, len(dates))
	for _, d := range dates {
		if _, err := time.Parse(time.DateOnly, d); err != nil {
			panic("schedule: invalid date " + d + ", want YYYY-MM-DD")
		}
		set[d] = true
	}
	return set
}

// DateRange matches dates from start to end inclusive. Accepts "YYYY-MM-DD" strings.
// Panics on invalid format.
func DateRange(start, end string) Rule {
	if _, err := time.Parse(time.DateOnly, start); err != nil {
		panic("schedule: invalid date " + start + ", want YYYY-MM-DD")
	}
	if _, err := time.Parse(time.DateOnly, end); err != nil {
		panic("schedule: invalid date " + end + ", want YYYY-MM-DD")
	}
	return RuleFunc(func(t time.Time) bool {
		d := t.Format(time.DateOnly)
		return d >= start && d <= end
	})
}

// Not inverts a rule.
func Not(r Rule) Rule {
	return RuleFunc(func(t time.Time) bool {
		return !r.Match(t)
	})
}
