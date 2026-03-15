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

func TestOvernightNext(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("22:00", "02:00"),
	)

	tests := []struct {
		name string
		t    time.Time
		want time.Time
	}{
		// Saturday 10:00 -> Monday 22:00 (skip weekend)
		{"sat midday", dt("2026-03-14 10:00"), dt("2026-03-16 22:00")},
		// Saturday 02:00 (just ended) -> Monday 22:00
		{"sat 02:00", dt("2026-03-14 02:00"), dt("2026-03-16 22:00")},
		// Friday 10:00 (daytime gap) -> Friday 22:00
		{"fri daytime", dt("2026-03-13 10:00"), dt("2026-03-13 22:00")},
		// Already active: returns same minute
		{"fri 23:00", dt("2026-03-13 23:00"), dt("2026-03-13 23:00")},
		// Post-midnight active: returns same minute
		{"sat 01:00 from fri", dt("2026-03-14 01:00"), dt("2026-03-14 01:00")},
	}

	for _, tt := range tests {
		got, ok := s.Next(tt.t)
		if !ok {
			t.Errorf("%s: Next returned false", tt.name)
			continue
		}
		if !got.Equal(tt.want) {
			t.Errorf("%s: Next(%v) = %v, want %v", tt.name, tt.t, got, tt.want)
		}
	}
}

func TestOvernightWindowBoundaries(t *testing.T) {
	s := New(
		Weekdays(),
		TimeRange("22:00", "02:00"),
	)

	// Exact start: Friday 22:00
	start, end, ok := s.Window(dt("2026-03-13 22:00"))
	if !ok {
		t.Fatal("Window returned false at exact start")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 02:00")) {
		t.Errorf("end = %v, want 2026-03-14 02:00", end)
	}

	// Post-midnight 00:00 (from Friday)
	start, end, ok = s.Window(dt("2026-03-14 00:00"))
	if !ok {
		t.Fatal("Window returned false at midnight")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 02:00")) {
		t.Errorf("end = %v, want 2026-03-14 02:00", end)
	}

	// Just before end: 01:59 (from Friday)
	start, end, ok = s.Window(dt("2026-03-14 01:59"))
	if !ok {
		t.Fatal("Window returned false at 01:59")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 02:00")) {
		t.Errorf("end = %v, want 2026-03-14 02:00", end)
	}

	// At end: 02:00 -> not active (exclusive end)
	_, _, ok = s.Window(dt("2026-03-14 02:00"))
	if ok {
		t.Error("Window should return false at exclusive end 02:00")
	}

	// Sunday 01:00 (from Saturday) -> not active
	_, _, ok = s.Window(dt("2026-03-15 01:00"))
	if ok {
		t.Error("Window should return false on Sunday post-midnight from Saturday")
	}
}

func TestOvernightNoWeekdayRule(t *testing.T) {
	// Pure overnight range, no day restriction
	s := New(TimeRange("22:00", "06:00"))

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"22:00", dt("2026-03-14 22:00"), true},
		{"23:59", dt("2026-03-14 23:59"), true},
		{"00:00", dt("2026-03-14 00:00"), true},
		{"05:59", dt("2026-03-14 05:59"), true},
		{"06:00", dt("2026-03-14 06:00"), false},
		{"12:00", dt("2026-03-14 12:00"), false},
		{"21:59", dt("2026-03-14 21:59"), false},
	}

	for _, tt := range tests {
		if got := s.Match(tt.t); got != tt.want {
			t.Errorf("%s: Match(%v) = %v, want %v", tt.name, tt.t, got, tt.want)
		}
	}

	// Window at 23:00: start today 22:00, end tomorrow 06:00
	start, end, ok := s.Window(dt("2026-03-14 23:00"))
	if !ok {
		t.Fatal("Window returned false")
	}
	if !start.Equal(dt("2026-03-14 22:00")) {
		t.Errorf("start = %v, want 2026-03-14 22:00", start)
	}
	if !end.Equal(dt("2026-03-15 06:00")) {
		t.Errorf("end = %v, want 2026-03-15 06:00", end)
	}

	// Window at 03:00 (post-midnight)
	start, end, ok = s.Window(dt("2026-03-14 03:00"))
	if !ok {
		t.Fatal("Window returned false at post-midnight")
	}
	if !start.Equal(dt("2026-03-13 22:00")) {
		t.Errorf("start = %v, want 2026-03-13 22:00", start)
	}
	if !end.Equal(dt("2026-03-14 06:00")) {
		t.Errorf("end = %v, want 2026-03-14 06:00", end)
	}
}

func TestOvernightParse(t *testing.T) {
	s, err := Parse([]byte(`{"days": "weekdays", "time_range": ["22:00", "02:00"]}`))
	if err != nil {
		t.Fatal(err)
	}

	// Friday 23:00: active
	if !s.Match(dt("2026-03-13 23:00")) {
		t.Error("should match Friday 23:00")
	}
	// Saturday 01:00 (from Friday): active
	if !s.Match(dt("2026-03-14 01:00")) {
		t.Error("should match Saturday 01:00 from Friday")
	}
	// Saturday 22:00: inactive
	if s.Match(dt("2026-03-14 22:00")) {
		t.Error("should not match Saturday 22:00")
	}
	// Friday daytime: inactive
	if s.Match(dt("2026-03-13 10:00")) {
		t.Error("should not match Friday daytime")
	}
}

func TestWeekdaysOnly(t *testing.T) {
	s := New(Weekdays())

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"monday morning", dt("2026-03-16 00:00"), true},
		{"monday midday", dt("2026-03-16 12:00"), true},
		{"monday late", dt("2026-03-16 23:59"), true},
		{"friday", dt("2026-03-20 15:00"), true},
		{"saturday", dt("2026-03-14 10:00"), false},
		{"sunday", dt("2026-03-15 10:00"), false},
	}

	for _, tt := range tests {
		if got := s.Match(tt.t); got != tt.want {
			t.Errorf("%s: Match(%v) = %v, want %v", tt.name, tt.t, got, tt.want)
		}
	}
}

func TestWeekdaysOnlyWindow(t *testing.T) {
	s := New(Weekdays())

	// Monday: window is midnight to 23:59
	start, end, ok := s.Window(dt("2026-03-16 10:00"))
	if !ok {
		t.Fatal("Window returned false on Monday")
	}
	if !start.Equal(dt("2026-03-16 00:00")) {
		t.Errorf("start = %v, want 2026-03-16 00:00", start)
	}
	if !end.Equal(dt("2026-03-16 23:59")) {
		t.Errorf("end = %v, want 2026-03-16 23:59", end)
	}

	// Saturday: not active
	_, _, ok = s.Window(dt("2026-03-14 10:00"))
	if ok {
		t.Error("Window should return false on Saturday")
	}

	// Sunday: not active
	_, _, ok = s.Window(dt("2026-03-15 10:00"))
	if ok {
		t.Error("Window should return false on Sunday")
	}
}

func TestWeekdaysOnlyNext(t *testing.T) {
	s := New(Weekdays())

	// Saturday -> next Monday 00:00
	got, ok := s.Next(dt("2026-03-14 10:00"))
	if !ok {
		t.Fatal("Next returned false")
	}
	want := dt("2026-03-16 00:00")
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}

	// Already active: returns same minute
	got, ok = s.Next(dt("2026-03-16 10:00"))
	if !ok || !got.Equal(dt("2026-03-16 10:00")) {
		t.Errorf("Next when active = %v (ok=%v), want 2026-03-16 10:00", got, ok)
	}
}

func TestWeekdaysOnlyActiveUntil(t *testing.T) {
	s := New(Weekdays())

	// Monday 10:00 -> active until Friday 23:59
	got, ok := s.ActiveUntil(dt("2026-03-16 10:00"))
	if !ok {
		t.Fatal("ActiveUntil returned false")
	}
	want := dt("2026-03-20 23:59")
	if !got.Equal(want) {
		t.Errorf("ActiveUntil = %v, want %v", got, want)
	}

	// Saturday: not active
	_, ok = s.ActiveUntil(dt("2026-03-14 10:00"))
	if ok {
		t.Error("ActiveUntil should return false on Saturday")
	}
}

func TestWeekdaysOnlyParse(t *testing.T) {
	s, err := Parse([]byte(`{"days": "weekdays"}`))
	if err != nil {
		t.Fatal(err)
	}

	if !s.Match(dt("2026-03-16 10:00")) { // Monday
		t.Error("should match Monday")
	}
	if s.Match(dt("2026-03-14 10:00")) { // Saturday
		t.Error("should not match Saturday")
	}

	start, end, ok := s.Window(dt("2026-03-16 10:00"))
	if !ok {
		t.Fatal("Window returned false")
	}
	if !start.Equal(dt("2026-03-16 00:00")) {
		t.Errorf("start = %v, want 2026-03-16 00:00", start)
	}
	if !end.Equal(dt("2026-03-16 23:59")) {
		t.Errorf("end = %v, want 2026-03-16 23:59", end)
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
