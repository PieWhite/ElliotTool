package model

// CorrectiveWave represents a validated 3-wave Elliott Wave corrective structure (ABC).
// It stores pointers to the 4 core pivots that form the corrective wave,
// along with the type (ZIGZAG or FLAT) and direction (BULLISH or BEARISH).
//
//easyjson:json
type CorrectiveWave struct {
	Start     *Pivot `json:"start"`
	WA        *Pivot `json:"wa"`
	WB        *Pivot `json:"wb"`
	WC        *Pivot `json:"wc"`
	Type      string `json:"type"`      // "ZIGZAG" or "FLAT"
	Direction string `json:"direction"` // "BULLISH" or "BEARISH"
}
