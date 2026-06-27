package market

import "time"

// Session selects which US equity trading session is analyzed.
type Session string

const (
	SessionRTH      Session = "RTH"
	SessionExtended Session = "EXTENDED"
)

// Candle is a split-adjusted OHLCV aggregate. Time is Unix seconds at the
// beginning of the provider bar; BarIndex is assigned after normalization.
//
//easyjson:json
type Candle struct {
	Time     int64   `json:"time"`
	BarIndex int     `json:"bar_index"`
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
}

// Timeframe is the canonical set supported by WaveSight.
type Timeframe string

const (
	Timeframe1m  Timeframe = "1m"
	Timeframe5m  Timeframe = "5m"
	Timeframe15m Timeframe = "15m"
	Timeframe1h  Timeframe = "1h"
	Timeframe4h  Timeframe = "4h"
	Timeframe1D  Timeframe = "1D"
	Timeframe1W  Timeframe = "1W"
)

func ParseTimeframe(value string) (Timeframe, bool) {
	switch value {
	case "1m":
		return Timeframe1m, true
	case "5m":
		return Timeframe5m, true
	case "15m":
		return Timeframe15m, true
	case "1h":
		return Timeframe1h, true
	case "4h":
		return Timeframe4h, true
	case "1D", "1d":
		return Timeframe1D, true
	case "1W", "1w":
		return Timeframe1W, true
	default:
		return "", false
	}
}

func (t Timeframe) ProviderRange() (int, string) {
	switch t {
	case Timeframe1m:
		return 1, "minute"
	case Timeframe5m:
		return 5, "minute"
	case Timeframe15m:
		return 15, "minute"
	case Timeframe1h:
		return 1, "hour"
	case Timeframe4h:
		return 4, "hour"
	case Timeframe1W:
		return 1, "week"
	default:
		return 1, "day"
	}
}

func (t Timeframe) Duration() time.Duration {
	switch t {
	case Timeframe1m:
		return time.Minute
	case Timeframe5m:
		return 5 * time.Minute
	case Timeframe15m:
		return 15 * time.Minute
	case Timeframe1h:
		return time.Hour
	case Timeframe4h:
		return 4 * time.Hour
	case Timeframe1W:
		return 7 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func (t Timeframe) DefaultLookbackBars() int {
	switch t {
	case Timeframe1m:
		return 8_000
	case Timeframe5m:
		return 10_000
	case Timeframe15m:
		return 10_000
	case Timeframe1h:
		return 8_000
	case Timeframe4h:
		return 6_000
	case Timeframe1W:
		return 2_000
	default:
		return 5_000
	}
}
