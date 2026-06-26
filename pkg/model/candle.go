package model

// Candle represents a single OHLCV candlestick, containing price and volume data.
// It uses Unix timestamps in seconds for compatibility with TradingView Lightweight Charts.
//
//easyjson:json
type Candle struct {
	Time   int64   `json:"time"`   // Unix timestamp in seconds
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
}
