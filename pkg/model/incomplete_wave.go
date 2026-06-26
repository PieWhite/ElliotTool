package model

// IncompleteWave represents a verified 1-2-3 Elliott Wave structure.
//
//easyjson:json
type IncompleteWave struct {
	Start           *Pivot     `json:"start"`
	W1              *Pivot     `json:"w1"`
	W2              *Pivot     `json:"w2"`
	W3              *Pivot     `json:"w3"`
	Direction       string     `json:"direction"` // "BULLISH" or "BEARISH"
	ConfidenceScore float64    `json:"confidence_score"`
	TargetBox       *TargetBox `json:"target_box,omitempty"`
	Degree          string     `json:"degree,omitempty"`
}
