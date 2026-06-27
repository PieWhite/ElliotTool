package market

import (
	"testing"
	"time"
)

func TestNormalizeRTHPreservesOriginalTimes(t *testing.T) {
	t.Parallel()
	calendar, err := NewUSCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	at := func(hour, minute int) int64 {
		return time.Date(2026, time.June, 26, hour, minute, 0, 0, location).Unix()
	}
	input := []Candle{
		{Time: at(8, 0), Open: 10, High: 11, Low: 9, Close: 10, Volume: 1},
		{Time: at(9, 30), Open: 10, High: 12, Low: 10, Close: 11, Volume: 2},
		{Time: at(15, 59), Open: 11, High: 13, Low: 11, Close: 12, Volume: 3},
		{Time: at(16, 0), Open: 12, High: 12, Low: 11, Close: 11, Volume: 4},
	}
	result := calendar.Normalize(input, Timeframe1m, SessionRTH)
	if len(result) != 2 {
		t.Fatalf("got %d RTH bars, want 2", len(result))
	}
	if result[0].Time != input[1].Time || result[1].Time != input[2].Time {
		t.Fatalf("timestamps changed: %+v", result)
	}
	if result[0].BarIndex != 0 || result[1].BarIndex != 1 {
		t.Fatalf("unexpected bar indices: %+v", result)
	}
}

func TestFutureBarsSkipWeekend(t *testing.T) {
	t.Parallel()
	calendar, err := NewUSCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	fridayClose := time.Date(2026, time.June, 26, 15, 59, 0, 0, location)
	future := calendar.FutureBarTimes(fridayClose, Timeframe1m, SessionRTH, 2)
	if len(future) != 2 {
		t.Fatalf("got %d bars, want 2", len(future))
	}
	first := time.Unix(future[0], 0).In(location)
	if first.Weekday() != time.Monday || first.Hour() != 9 || first.Minute() != 30 {
		t.Fatalf("first future RTH bar = %v", first)
	}
}

func TestCalendarHonorsHolidaysEarlyCloseAndExtendedSession(t *testing.T) {
	t.Parallel()
	calendar, err := NewUSCalendar()
	if err != nil {
		t.Fatalf("NewUSCalendar() error = %v", err)
	}
	location := calendar.location

	// The bar after the 2026 Thanksgiving half day must start on Monday,
	// because a 4-hour RTH bar cannot begin after the 13:00 close.
	fridayNoon := time.Date(2026, time.November, 27, 12, 30, 0, 0, location)
	future := calendar.FutureBarTimes(fridayNoon, Timeframe1h, SessionRTH, 1)
	got := time.Unix(future[0], 0).In(location)
	if got.Weekday() != time.Monday || got.Hour() != 9 || got.Minute() != 30 {
		t.Fatalf("bar after early close = %s, want Monday 09:30 ET", got)
	}

	extended := []Candle{
		validTestCandle(time.Date(2026, time.June, 26, 3, 59, 0, 0, location).Unix()),
		validTestCandle(time.Date(2026, time.June, 26, 4, 0, 0, 0, location).Unix()),
		validTestCandle(time.Date(2026, time.June, 26, 19, 59, 0, 0, location).Unix()),
		validTestCandle(time.Date(2026, time.June, 26, 20, 0, 0, 0, location).Unix()),
	}
	normalized := calendar.Normalize(extended, Timeframe1m, SessionExtended)
	if len(normalized) != 2 {
		t.Fatalf("extended-hours candle count = %d, want 2", len(normalized))
	}

	holiday := []Candle{
		validTestCandle(time.Date(2026, time.July, 3, 10, 0, 0, 0, location).Unix()),
	}
	if got := calendar.Normalize(holiday, Timeframe1m, SessionRTH); len(got) != 0 {
		t.Fatalf("observed Independence Day produced %d bars", len(got))
	}
}

func validTestCandle(timestamp int64) Candle {
	return Candle{
		Time: timestamp, Open: 100, High: 102, Low: 99, Close: 101, Volume: 1_000,
	}
}
