package model

// PivotType represents the type of pivot: HIGH or LOW.
type PivotType string

const (
	PivotHigh PivotType = "HIGH"
	PivotLow  PivotType = "LOW"
)

// Pivot represents a local peak (HIGH) or trough (LOW) identified by the ZigZag algorithm.
// It uses Unix timestamps in seconds matching the source candle's timestamp.
//
//easyjson:json
type Pivot struct {
	Time  int64     `json:"time"`
	Price float64   `json:"price"`
	Type  PivotType `json:"type"`
}
