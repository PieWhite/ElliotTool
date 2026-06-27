package market

import (
	"fmt"
	"sort"
	"time"
)

const newYorkLocation = "America/New_York"

// Calendar normalizes US equity bars and projects future tradable bar times.
type Calendar struct {
	location *time.Location
}

func NewUSCalendar() (*Calendar, error) {
	location, err := time.LoadLocation(newYorkLocation)
	if err != nil {
		return nil, fmt.Errorf("loading New York timezone: %w", err)
	}
	return &Calendar{location: location}, nil
}

func (c *Calendar) Normalize(candles []Candle, timeframe Timeframe, session Session) []Candle {
	if len(candles) == 0 {
		return nil
	}

	normalized := make([]Candle, 0, len(candles))
	for _, candle := range candles {
		if !validCandle(candle) {
			continue
		}
		local := time.Unix(candle.Time, 0).In(c.location)
		if timeframe != Timeframe1W && !isTradingDay(local) {
			continue
		}
		if timeframe != Timeframe1D && timeframe != Timeframe1W {
			minutes := local.Hour()*60 + local.Minute()
			openMinutes, closeMinutes := sessionMinutes(local, session)
			if minutes < openMinutes || minutes >= closeMinutes {
				continue
			}
		}
		normalized = append(normalized, candle)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Time < normalized[j].Time
	})

	deduplicated := normalized[:0]
	for _, candle := range normalized {
		if len(deduplicated) > 0 && deduplicated[len(deduplicated)-1].Time == candle.Time {
			deduplicated[len(deduplicated)-1] = candle
			continue
		}
		candle.BarIndex = len(deduplicated)
		deduplicated = append(deduplicated, candle)
	}
	return deduplicated
}

func validCandle(c Candle) bool {
	if c.Time <= 0 || c.Open <= 0 || c.High <= 0 || c.Low <= 0 || c.Close <= 0 {
		return false
	}
	return c.High >= c.Low && c.High >= c.Open && c.High >= c.Close &&
		c.Low <= c.Open && c.Low <= c.Close
}

// FutureBarTimes returns tradable timestamps without pretending that weekends
// are regular bars. Provider calendars remain authoritative for historical data.
func (c *Calendar) FutureBarTimes(last time.Time, timeframe Timeframe, session Session, count int) []int64 {
	if count <= 0 {
		return nil
	}
	result := make([]int64, 0, count)
	cursor := last.In(c.location)

	for len(result) < count {
		cursor = c.nextBar(cursor, timeframe, session)
		result = append(result, cursor.Unix())
	}
	return result
}

func (c *Calendar) nextBar(current time.Time, timeframe Timeframe, session Session) time.Time {
	if timeframe == Timeframe1D || timeframe == Timeframe1W {
		stepDays := 1
		if timeframe == Timeframe1W {
			stepDays = 7
		}
		next := current.AddDate(0, 0, stepDays)
		for !isTradingDay(next) {
			next = next.AddDate(0, 0, 1)
		}
		return next
	}

	next := current.Add(timeframe.Duration())
	for {
		if !isTradingDay(next) {
			next = c.nextTradingOpen(next, session)
			continue
		}
		openMinutes, closeMinutes := sessionMinutes(next, session)
		minutes := next.Hour()*60 + next.Minute()
		if minutes < openMinutes {
			next = atMinute(next, openMinutes, c.location)
		}
		if minutes >= closeMinutes {
			next = c.nextTradingOpen(next.AddDate(0, 0, 1), session)
			continue
		}
		return next
	}
}

func (c *Calendar) nextTradingOpen(candidate time.Time, session Session) time.Time {
	next := candidate.In(c.location)
	for !isTradingDay(next) {
		next = next.AddDate(0, 0, 1)
	}
	openMinutes, _ := sessionMinutes(next, session)
	return atMinute(next, openMinutes, c.location)
}

func (c *Calendar) TradingDaysBefore(value time.Time, count int) time.Time {
	if count <= 0 {
		return value
	}
	current := value.In(c.location)
	for count > 0 {
		current = current.AddDate(0, 0, -1)
		if isTradingDay(current) {
			count--
		}
	}
	return current
}

func atMinute(day time.Time, minute int, location *time.Location) time.Time {
	return time.Date(
		day.Year(), day.Month(), day.Day(), minute/60, minute%60, 0, 0, location,
	)
}

func sessionMinutes(day time.Time, session Session) (int, int) {
	if session == SessionExtended {
		return 4 * 60, 20 * 60
	}
	closeMinutes := 16 * 60
	if isEarlyClose(day) {
		closeMinutes = 13 * 60
	}
	return 9*60 + 30, closeMinutes
}

func isTradingDay(day time.Time) bool {
	if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
		return false
	}
	year := day.Year()
	date := dateOnly(day)
	holidays := []time.Time{
		observedFixedHoliday(year, time.January, 1, day.Location()),
		nthWeekday(year, time.January, time.Monday, 3, day.Location()),
		nthWeekday(year, time.February, time.Monday, 3, day.Location()),
		easterSunday(year, day.Location()).AddDate(0, 0, -2),
		lastWeekday(year, time.May, time.Monday, day.Location()),
		observedFixedHoliday(year, time.July, 4, day.Location()),
		nthWeekday(year, time.September, time.Monday, 1, day.Location()),
		nthWeekday(year, time.November, time.Thursday, 4, day.Location()),
		observedFixedHoliday(year, time.December, 25, day.Location()),
	}
	if year >= 2022 {
		holidays = append(holidays, observedFixedHoliday(year, time.June, 19, day.Location()))
	}
	for _, holiday := range holidays {
		if date.Equal(dateOnly(holiday)) {
			return false
		}
	}
	return true
}

func isEarlyClose(day time.Time) bool {
	if !isTradingDay(day) {
		return false
	}
	location := day.Location()
	year := day.Year()
	date := dateOnly(day)
	thanksgiving := nthWeekday(year, time.November, time.Thursday, 4, location)
	dayAfterThanksgiving := thanksgiving.AddDate(0, 0, 1)
	if date.Equal(dateOnly(dayAfterThanksgiving)) {
		return true
	}
	julyThird := time.Date(year, time.July, 3, 0, 0, 0, 0, location)
	if julyThird.Weekday() >= time.Monday && julyThird.Weekday() <= time.Friday &&
		isTradingDay(julyThird) && date.Equal(julyThird) {
		return true
	}
	christmasEve := time.Date(year, time.December, 24, 0, 0, 0, 0, location)
	return christmasEve.Weekday() >= time.Monday &&
		christmasEve.Weekday() <= time.Friday &&
		isTradingDay(christmasEve) &&
		date.Equal(christmasEve)
}

func observedFixedHoliday(year int, month time.Month, day int, location *time.Location) time.Time {
	holiday := time.Date(year, month, day, 0, 0, 0, 0, location)
	switch holiday.Weekday() {
	case time.Saturday:
		return holiday.AddDate(0, 0, -1)
	case time.Sunday:
		return holiday.AddDate(0, 0, 1)
	default:
		return holiday
	}
}

func nthWeekday(year int, month time.Month, weekday time.Weekday, occurrence int, location *time.Location) time.Time {
	date := time.Date(year, month, 1, 0, 0, 0, 0, location)
	offset := (int(weekday) - int(date.Weekday()) + 7) % 7
	return date.AddDate(0, 0, offset+(occurrence-1)*7)
}

func lastWeekday(year int, month time.Month, weekday time.Weekday, location *time.Location) time.Time {
	date := time.Date(year, month+1, 0, 0, 0, 0, 0, location)
	offset := (int(date.Weekday()) - int(weekday) + 7) % 7
	return date.AddDate(0, 0, -offset)
}

func easterSunday(year int, location *time.Location) time.Time {
	// Gregorian computus (Meeus/Jones/Butcher), valid for modern exchange calendars.
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := time.Month((h + l - 7*m + 114) / 31)
	day := (h+l-7*m+114)%31 + 1
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

func TrimToLookback(candles []Candle, bars int) []Candle {
	if bars <= 0 || len(candles) <= bars {
		return candles
	}
	trimmed := append([]Candle(nil), candles[len(candles)-bars:]...)
	for i := range trimmed {
		trimmed[i].BarIndex = i
	}
	return trimmed
}
