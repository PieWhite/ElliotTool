package model

// WaveStructure is a generic, type-agnostic representation of any detected Elliott Wave
// structure (motive, corrective, or incomplete). It is used inside AnalysisScenario so that
// the frontend can render any structure without knowing its specific domain type.
//
//easyjson:json
type WaveStructure struct {
	// Type identifies the pattern variant, e.g. "MOTIVE_IMPULSE", "MOTIVE_DIAGONAL",
	// "CORRECTIVE_ZIGZAG", "CORRECTIVE_FLAT", "CORRECTIVE_TRIANGLE", "CORRECTIVE_WXY",
	// "INCOMPLETE_123".
	Type            string      `json:"type"`
	Pivots          []Pivot     `json:"pivots"`
	PurpleBoxes     []TargetBox `json:"purple_boxes,omitempty"`
	ConfidenceScore float64     `json:"confidence_score"`
	Degree          string      `json:"degree,omitempty"`
}

// AnalysisScenario bundles a directional bias with one or more wave structures
// that support that interpretation. Primary and Alternate are the top two
// ranked scenarios ordered by descending confidence.
//
//easyjson:json
type AnalysisScenario struct {
	Bias       string          `json:"bias"`       // "BULLISH" or "BEARISH"
	Confidence float64         `json:"confidence"` // highest confidence among Structures
	Structures []WaveStructure `json:"structures"`
}

// ScenarioPair holds the Primary (highest-confidence) and Alternate (second-highest or
// inverse-directional fallback) Elliott Wave interpretation for a given data set.
//
//easyjson:json
type ScenarioPair struct {
	Primary   AnalysisScenario `json:"primary"`
	Alternate AnalysisScenario `json:"alternate"`
}

// AnalysisResponse represents the high-performance serialized JSON response for the Elliott analysis API.
//
//easyjson:json
type AnalysisResponse struct {
	Ticker    string   `json:"ticker"`
	Timeframe string   `json:"timeframe"`
	Candles   []Candle `json:"candles"`

	// Step 10: Probabilistic scenario pair (primary + alternate counts).
	Scenarios *ScenarioPair `json:"scenarios,omitempty"`

	// Legacy flat arrays — retained for backward compatibility with existing frontend consumers.
	// Populated alongside Scenarios so both representations are always available.
	MotiveWaves     []MotiveWave     `json:"motive_waves,omitempty"`
	CorrectiveWaves []CorrectiveWave `json:"corrective_waves,omitempty"`
	IncompleteWaves []IncompleteWave `json:"incomplete_waves,omitempty"`
}
