package api

import (
	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
	"WaveSight/pkg/repository"
)

//easyjson:json
type AnalysisRequest struct {
	Symbol       string `json:"symbol"`
	Timeframe    string `json:"timeframe"`
	Session      string `json:"session"`
	AsOf         string `json:"as_of,omitempty"`
	LookbackBars int    `json:"lookback_bars,omitempty"`
	MaxScenarios int    `json:"max_scenarios,omitempty"`
}

//easyjson:json
type AnalysisSnapshot struct {
	ID            string           `json:"id"`
	TheoryVersion string           `json:"theory_version"`
	EngineVersion string           `json:"engine_version"`
	GeneratedAt   int64            `json:"generated_at"`
	Request       AnalysisRequest  `json:"request"`
	DataQuality   wave.DataQuality `json:"data_quality"`
	Candles       []market.Candle  `json:"candles"`
	Scenarios     []wave.Scenario  `json:"scenarios"`
	FutureBars    []int64          `json:"future_bars"`
}

//easyjson:json
type SnapshotHistory struct {
	Items []repository.SnapshotMetadata `json:"items"`
}

//easyjson:json
type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail"`
	RequestID string `json:"request_id"`
}
