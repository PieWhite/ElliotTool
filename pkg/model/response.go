package model

// AnalysisResponse represents the high-performance serialized JSON response for the Elliott analysis API.
//
//easyjson:json
type AnalysisResponse struct {
	Ticker          string           `json:"ticker"`
	Timeframe       string           `json:"timeframe"`
	Candles         []Candle         `json:"candles"`
	MotiveWaves     []MotiveWave     `json:"motive_waves"`
	CorrectiveWaves []CorrectiveWave `json:"corrective_waves"`
	IncompleteWaves []IncompleteWave `json:"incomplete_waves"`
}
