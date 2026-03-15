package schedule

import (
	"testing"
	"time"
)

func dt(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestWeekdayBusinessHoursExceptHoliday(t *testing.T) {
	s := New(
		Weekdays(),
		ExceptDates("2026-01-01", "2026-12-25"),
		TimeRange("07:00", "19:00"),
	)

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"weekday in range", dt("2026-03-16 10:00"), true},      // Monday
		{"weekday before range", dt("2026-03-16 06:59"), false},  // Monday
		{"weekday at start", dt("2026-03-16 07:00"), true},       // Monday
		{"weekday at end", dt("2026-03-16 19:00"), false},        // Monday
		{"weekend", dt("2026-03-14 10:00"), false},               // Saturday
		{"holiday", dt("2026-01-01 10:00"), false},               // Thursday but holiday
		{"holiday weekend", dt("2026-12-25 10:00"), false},       // Friday but holiday
		{"weekday end of range", dt("2026-03-16 18:59"), true},   // Monday
	}

	for _, tt := range tests {
		if got := s.Match(tt.t); got != tt.want {
			t.Errorf("%s: Match(%v) = %v, want %v", tt.name, tt.t, got, tt.want)
		}
	}
}

func TestTimeRangeOverMidnight(t *testing.T) {
	s := New(TimeRange("22:00", "06:00"))

	tests := []struct {
		t    time.Time
		want bool
	}{
		{dt("2026-01-01 23:00"), true},
		{dt("2026-01-01 05:59"), true},
		{dt("2026-01-01 06:00"), false},
		{dt("2026-01-01 21:59"), false},
		{dt("2026-01-01 22:00"), true},
	}

	for _, tt := range tests {
		if got := s.Match(tt.t); got != tt.want {
			t.Errorf("Match(%v) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestNext(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("07:00", "19:00"),
	)

	// Saturday 10:00 -> next Monday 07:00
	got, ok := s.Next(dt("2026-03-14 10:00"))
	if !ok {
		t.Fatal("Next returned false")
	}
	want := dt("2026-03-16 07:00")
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}

	// Already active: returns same minute
	got, ok = s.Next(dt("2026-03-16 10:00"))
	if !ok || !got.Equal(dt("2026-03-16 10:00")) {
		t.Errorf("Next when active = %v (ok=%v), want 2026-03-16 10:00", got, ok)
	}
}

func TestActiveUntil(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("07:00", "19:00"),
	)

	got, ok := s.ActiveUntil(dt("2026-03-16 10:00")) // Monday
	if !ok {
		t.Fatal("ActiveUntil returned false")
	}
	want := dt("2026-03-16 18:59")
	if !got.Equal(want) {
		t.Errorf("ActiveUntil = %v, want %v", got, want)
	}

	// Not active
	_, ok = s.ActiveUntil(dt("2026-03-14 10:00")) // Saturday
	if ok {
		t.Error("ActiveUntil should return false on Saturday")
	}
}

func TestDays(t *testing.T) {
	s := New(Days(time.Monday, time.Wednesday, time.Friday))

	if !s.Match(dt("2026-03-16 12:00")) { // Monday
		t.Error("should match Monday")
	}
	if s.Match(dt("2026-03-17 12:00")) { // Tuesday
		t.Error("should not match Tuesday")
	}
}

func TestDateRange(t *testing.T) {
	s := New(DateRange("2026-03-01", "2026-03-15"))

	if !s.Match(dt("2026-03-01 00:00")) {
		t.Error("should match start")
	}
	if !s.Match(dt("2026-03-15 23:59")) {
		t.Error("should match end")
	}
	if s.Match(dt("2026-03-16 00:00")) {
		t.Error("should not match after end")
	}
}

func TestNot(t *testing.T) {
	s := New(Not(Weekends()))
	if !s.Match(dt("2026-03-16 12:00")) { // Monday
		t.Error("Not(Weekends) should match Monday")
	}
	if s.Match(dt("2026-03-14 12:00")) { // Saturday
		t.Error("Not(Weekends) should not match Saturday")
	}
}

func TestOvernightWeekdays(t *testing.T) {
	// 2026-03-13 is Friday, 2026-03-14 is Saturday, 2026-03-15 is Sunday
	s := New(
		Weekdays(),
		ExceptDates("2026-03-12"), // Thursday holiday
		TimeRange("22:00", "02:00"),
	)

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		// Friday night → Saturday early morning: active (started on Friday)
		{"fri 22:00", dt("2026-03-13 22:00"), true},
		{"fri 23:59", dt("2026-03-13 23:59"), true},
		{"sat 00:00 (from fri)", dt("2026-03-14 00:00"), true},
		{"sat 01:59 (from fri)", dt("2026-03-14 01:59"), true},
		{"sat 02:00 (ended)", dt("2026-03-14 02:00"), false},

		// Saturday night: inactive (Saturday is not a weekday)
		{"sat 22:00", dt("2026-03-14 22:00"), false},
		{"sun 01:00 (from sat)", dt("2026-03-15 01:00"), false},

		// Sunday night: inactive
		{"sun 22:00", dt("2026-03-15 22:00"), false},
		// Monday early morning from Sunday: inactive
		{"mon 01:00 (from sun)", dt("2026-03-16 01:00"), false},

		// Monday night: active
		{"mon 22:00", dt("2026-03-16 22:00"), true},
		{"tue 01:00 (from mon)", dt("2026-03-17 01:00"), true},

		// Thursday is a holiday
		{"thu 22:00 (holiday)", dt("2026-03-12 22:00"), false},
		{"fri 01:00 (from thu holiday)", dt("2026-03-13 01:00"), false},

		// Daytime: inactive
		{"fri 10:00", dt("2026-03-13 10:00"), false},
	}

	for _, tt := range tests {
		if got := s.Match(tt.t); got != tt.want {
			t.Errorf("%s: Match(%v) = %v, want %v", tt.name, tt.t, got, tt.want)
		}
	}
}

func TestOvernightWindow(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("22:00", "02:00"),
	)

	// Friday 23:00 → window is Fri 22:00 to Sat 02:00
	start, end, ok := s.Window(dt("2026-03-13 23:00"))
	if !ok {
		t.Fatal("Window returned false")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 02:00")) {
		t.Errorf("end = %v, want 2026-03-14 02:00", end)
	}

	// Saturday 01:00 (post-midnight, started Friday) → same window
	start, end, ok = s.Window(dt("2026-03-14 01:00"))
	if !ok {
		t.Fatal("Window returned false for post-midnight")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 02:00")) {
		t.Errorf("end = %v, want 2026-03-14 02:00", end)
	}

	// Saturday 22:00 → not active
	_, _, ok = s.Window(dt("2026-03-14 22:00"))
	if ok {
		t.Error("Window should return false on Saturday night")
	}
}

func TestOvernightActiveUntil(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("22:00", "02:00"),
	)

	got, ok := s.ActiveUntil(dt("2026-03-13 23:00")) // Friday
	if !ok {
		t.Fatal("ActiveUntil returned false")
	}
	want := dt("2026-03-14 01:59")
	if !got.Equal(want) {
		t.Errorf("ActiveUntil = %v, want %v", got, want)
	}
}

func TestOnlyDates(t *testing.T) {
	s := New(OnlyDates("2026-12-25"))
	if !s.Match(dt("2026-12-25 12:00")) {
		t.Error("should match")
	}
	if s.Match(dt("2026-12-26 12:00")) {
		t.Error("should not match")
	}
}
