package model

// CorrectiveWave represents a validated Elliott Wave corrective structure.
// Standard ABC structures (ZIGZAG, FLAT) use Start, WA, WB, WC.
// Triangles (TRIANGLE) additionally populate WD and WE.
// Double Threes (WXY) additionally populate WX as the connecting X-wave pivot.
//
//easyjson:json
type CorrectiveWave struct {
	Start     *Pivot `json:"start"`
	WA        *Pivot `json:"wa"`
	WB        *Pivot `json:"wb"`
	WC        *Pivot `json:"wc"`
	WD        *Pivot `json:"wd,omitempty"` // Triangle leg D (pivot 4 of ABCDE)
	WE        *Pivot `json:"we,omitempty"` // Triangle leg E (pivot 5 of ABCDE)
	WX        *Pivot `json:"wx,omitempty"` // WXY Double Three: X-wave connector pivot
	Type      string `json:"type"`         // "ZIGZAG", "FLAT", "TRIANGLE", or "WXY"
	Direction string `json:"direction"`    // "BULLISH" or "BEARISH"
}
