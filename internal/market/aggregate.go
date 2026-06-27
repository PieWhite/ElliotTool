package market

import (
	"fmt"
	"sort"
	"time"
)

type BarProvenance string

const (
	ProvenanceMinuteDerived BarProvenance = "MINUTE_DERIVED"
	ProvenanceNativeDaily   BarProvenance = "NATIVE_DAILY"
)

// DerivedCandle keeps the source coordinates of its extrema. These coordinates
// make a daily or hourly pivot refer to the same event as its minute detail.
//
//easyjson:json
type DerivedCandle struct {
	Candle
	HighTime   int64         `json:"high_time"`
	LowTime    int64         `json:"low_time"`
	SourceFrom int64         `json:"source_from"`
	SourceTo   int64         `json:"source_to"`
	Provenance BarProvenance `json:"provenance"`
	Partial    bool          `json:"partial"`
}

// CanonicalViews contains all chart projections derived from only two native
// datasets. NativeDaily remains available for provenance comparisons.
type CanonicalViews struct {
	Views       map[Timeframe][]DerivedCandle
	NativeDaily []DerivedCandle
}

func (c *Calendar) BuildCanonicalViews(
	minuteNative []Candle,
	dailyNative []Candle,
	session Session,
	asOf time.Time,
) (CanonicalViews, error) {
	if session != SessionRTH && session != SessionExtended {
		return CanonicalViews{}, fmt.Errorf("building canonical views: unsupported session %q", session)
	}
	minutes := c.Normalize(minuteNative, Timeframe1m, session)
	days := c.Normalize(dailyNative, Timeframe1D, session)

	views := make(map[Timeframe][]DerivedCandle, 7)
	views[Timeframe1m] = promoteCandles(minutes, ProvenanceMinuteDerived)
	for _, timeframe := range []Timeframe{Timeframe5m, Timeframe15m, Timeframe1h, Timeframe4h} {
		views[timeframe] = c.aggregateMinutes(minutes, timeframe, session, asOf)
	}

	nativeDays := promoteCandles(days, ProvenanceNativeDaily)
	derivedDays := c.aggregateMinutes(minutes, Timeframe1D, session, asOf)
	views[Timeframe1D] = mergeCanonicalDays(nativeDays, derivedDays)
	views[Timeframe1W] = c.aggregateDays(views[Timeframe1D], asOf)
	return CanonicalViews{Views: views, NativeDaily: nativeDays}, nil
}

func promoteCandles(candles []Candle, provenance BarProvenance) []DerivedCandle {
	result := make([]DerivedCandle, 0, len(candles))
	for index, candle := range candles {
		candle.BarIndex = index
		result = append(result, DerivedCandle{
			Candle: candle, HighTime: candle.Time, LowTime: candle.Time,
			SourceFrom: candle.Time, SourceTo: candle.Time, Provenance: provenance,
		})
	}
	return result
}

func (c *Calendar) aggregateMinutes(
	minutes []Candle,
	timeframe Timeframe,
	session Session,
	asOf time.Time,
) []DerivedCandle {
	if len(minutes) == 0 {
		return nil
	}
	type bucket struct {
		key      int64
		expected int
	}
	buckets := make(map[int64][]Candle)
	meta := make(map[int64]bucket)
	for _, candle := range minutes {
		local := time.Unix(candle.Time, 0).In(c.location)
		openMinute, closeMinute := sessionMinutes(local, session)
		minuteOfDay := local.Hour()*60 + local.Minute()
		if minuteOfDay < openMinute || minuteOfDay >= closeMinute {
			continue
		}
		var key time.Time
		expected := 1
		switch timeframe {
		case Timeframe5m, Timeframe15m, Timeframe1h, Timeframe4h:
			width := int(timeframe.Duration() / time.Minute)
			offset := minuteOfDay - openMinute
			startMinute := openMinute + (offset/width)*width
			key = atMinute(local, startMinute, c.location)
			expected = width
		case Timeframe1D:
			key = atMinute(local, openMinute, c.location)
			expected = closeMinute - openMinute
		default:
			key = local.Truncate(time.Minute)
		}
		buckets[key.Unix()] = append(buckets[key.Unix()], candle)
		meta[key.Unix()] = bucket{key: key.Unix(), expected: expected}
	}

	keys := make([]int64, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	result := make([]DerivedCandle, 0, len(keys))
	for _, key := range keys {
		bars := buckets[key]
		value := foldCandles(key, bars, ProvenanceMinuteDerived)
		value.Partial = len(bars) < meta[key].expected || value.SourceTo >= asOf.Unix()
		value.BarIndex = len(result)
		result = append(result, value)
	}
	return result
}

func (c *Calendar) aggregateDays(days []DerivedCandle, asOf time.Time) []DerivedCandle {
	if len(days) == 0 {
		return nil
	}
	groups := make(map[int64][]DerivedCandle)
	for _, day := range days {
		local := time.Unix(day.Time, 0).In(c.location)
		mondayOffset := (int(local.Weekday()) + 6) % 7
		monday := dateOnly(local).AddDate(0, 0, -mondayOffset)
		groups[monday.Unix()] = append(groups[monday.Unix()], day)
	}
	keys := make([]int64, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	result := make([]DerivedCandle, 0, len(keys))
	for _, key := range keys {
		items := groups[key]
		raw := make([]Candle, 0, len(items))
		for _, item := range items {
			raw = append(raw, item.Candle)
		}
		value := foldCandles(key, raw, items[0].Provenance)
		value.HighTime = items[0].HighTime
		value.LowTime = items[0].LowTime
		value.SourceFrom = items[0].SourceFrom
		value.SourceTo = items[len(items)-1].SourceTo
		for _, item := range items {
			if item.High == value.High && item.HighTime < value.HighTime {
				value.HighTime = item.HighTime
			}
			if item.Low == value.Low && item.LowTime < value.LowTime {
				value.LowTime = item.LowTime
			}
			if item.Provenance == ProvenanceMinuteDerived {
				value.Provenance = ProvenanceMinuteDerived
			}
		}
		value.Partial = len(items) < 4 || value.SourceTo >= asOf.Unix()
		value.BarIndex = len(result)
		result = append(result, value)
	}
	return result
}

func foldCandles(key int64, bars []Candle, provenance BarProvenance) DerivedCandle {
	first := bars[0]
	last := bars[len(bars)-1]
	value := DerivedCandle{
		Candle: Candle{
			Time: key, Open: first.Open, High: first.High, Low: first.Low,
			Close: last.Close,
		},
		HighTime: first.Time, LowTime: first.Time,
		SourceFrom: first.Time, SourceTo: last.Time, Provenance: provenance,
	}
	for _, candle := range bars {
		if candle.High > value.High {
			value.High = candle.High
			value.HighTime = candle.Time
		}
		if candle.Low < value.Low {
			value.Low = candle.Low
			value.LowTime = candle.Time
		}
		value.Volume += candle.Volume
	}
	return value
}

func mergeCanonicalDays(native, derived []DerivedCandle) []DerivedCandle {
	byDate := make(map[string]DerivedCandle, len(native)+len(derived))
	for _, candle := range native {
		byDate[time.Unix(candle.Time, 0).UTC().Format("2006-01-02")] = candle
	}
	for _, candle := range derived {
		// Minute-derived bars are authoritative wherever minute coverage exists.
		byDate[time.Unix(candle.Time, 0).UTC().Format("2006-01-02")] = candle
	}
	result := make([]DerivedCandle, 0, len(byDate))
	for _, candle := range byDate {
		result = append(result, candle)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Time < result[j].Time })
	for index := range result {
		result[index].BarIndex = index
	}
	return result
}

func PlainCandles(values []DerivedCandle) []Candle {
	result := make([]Candle, len(values))
	for index := range values {
		result[index] = values[index].Candle
		result[index].BarIndex = index
	}
	return result
}
