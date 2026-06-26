package model

// TargetBox represents the coordinates of the Purple Box price target projection zone.
//
//easyjson:json
type TargetBox struct {
	MinPrice  float64 `json:"min_price"`
	MaxPrice  float64 `json:"max_price"`
	StartTime int64   `json:"start_time"`
	EndTime   int64   `json:"end_time"`
}
