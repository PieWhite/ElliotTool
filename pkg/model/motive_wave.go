package model

// MotiveWave represents a validated 5-wave Elliott Wave structure.
// It stores pointers to the 6 core pivots that form the motive wave,
// along with the direction (BULLISH or BEARISH).
//
//easyjson:json
type MotiveWave struct {
	Start           *Pivot      `json:"start"`
	W1              *Pivot      `json:"w1"`
	W2              *Pivot      `json:"w2"`
	W3              *Pivot      `json:"w3"`
	W4              *Pivot      `json:"w4"`
	W5              *Pivot      `json:"w5"`
	Direction       string      `json:"direction"` // "BULLISH" or "BEARISH"
	ConfidenceScore float64     `json:"confidence_score"`
	PurpleBoxes     []TargetBox `json:"purple_boxes,omitempty"`
	IsDiagonal      bool        `json:"is_diagonal"`
	IsTruncated     bool        `json:"is_truncated"`
}
