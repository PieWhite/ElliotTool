package market

import (
	"testing"
	"time"
)

func TestBuildCanonicalViewsUsesSessionAnchorsAndSharedExtrema(t *testing.T) {
	calendar, err := NewUSCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, _ := time.LoadLocation(newYorkLocation)
	start := time.Date(2026, time.June, 26, 9, 30, 0, 0, location)
	minutes := make([]Candle, 0, 90)
	for index := 0; index < 90; index++ {
		value := 100 + float64(index)/10
		minutes = append(minutes, Candle{
			Time: start.Add(time.Duration(index) * time.Minute).Unix(),
			Open: value, High: value + 1, Low: value - 1, Close: value + 0.2, Volume: 10,
		})
	}
	views, err := calendar.BuildCanonicalViews(minutes, nil, SessionRTH, start.Add(90*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	hours := views.Views[Timeframe1h]
	if len(hours) != 2 {
		t.Fatalf("got %d hourly bars, want 2", len(hours))
	}
	if got := time.Unix(hours[0].Time, 0).In(location); got.Hour() != 9 || got.Minute() != 30 {
		t.Fatalf("first hour starts at %s, want 09:30", got)
	}
	if hours[0].HighTime != minutes[59].Time {
		t.Fatalf("hour high event %d, want source minute %d", hours[0].HighTime, minutes[59].Time)
	}
	if hours[1].SourceFrom != minutes[60].Time {
		t.Fatalf("second hour source starts at %d, want %d", hours[1].SourceFrom, minutes[60].Time)
	}
}

func TestBuildCanonicalViewsPrefersMinuteDerivedRecentDayAndBuildsWeekFromDays(t *testing.T) {
	calendar, err := NewUSCalendar()
	if err != nil {
		t.Fatal(err)
	}
	location, _ := time.LoadLocation(newYorkLocation)
	day := time.Date(2026, time.June, 26, 9, 30, 0, 0, location)
	minutes := []Candle{
		{Time: day.Unix(), Open: 100, High: 104, Low: 99, Close: 103, Volume: 5},
	}
	native := []Candle{
		{Time: day.UTC().Unix(), Open: 100, High: 105, Low: 98, Close: 104, Volume: 9},
	}
	views, err := calendar.BuildCanonicalViews(minutes, native, SessionRTH, day.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	days := views.Views[Timeframe1D]
	if len(days) != 1 || days[0].Provenance != ProvenanceMinuteDerived {
		t.Fatalf("canonical day = %#v, want one minute-derived day", days)
	}
	if weeks := views.Views[Timeframe1W]; len(weeks) != 1 || weeks[0].Provenance != ProvenanceMinuteDerived {
		t.Fatalf("weekly view = %#v, want week derived from canonical day", weeks)
	}
}
