package wave

import "WaveSight/internal/market"

const (
	TheoryVersion = "ewp-frost-prechter+waveratios-1.0.0"
	EngineVersion = "2.0.0"
)

type Direction string

const (
	DirectionBullish Direction = "BULLISH"
	DirectionBearish Direction = "BEARISH"
)

func (d Direction) Opposite() Direction {
	if d == DirectionBullish {
		return DirectionBearish
	}
	return DirectionBullish
}

type PivotKind string

const (
	PivotHigh PivotKind = "HIGH"
	PivotLow  PivotKind = "LOW"
)

type PivotState string

const (
	PivotConfirmed   PivotState = "CONFIRMED"
	PivotProvisional PivotState = "PROVISIONAL"
	PivotAmbiguous   PivotState = "AMBIGUOUS"
)

// Pivot retains the original bar coordinate. No synthetic timestamp mutation is
// permitted anywhere in the engine.
//
//easyjson:json
type Pivot struct {
	Time       int64      `json:"time"`
	BarIndex   int        `json:"bar_index"`
	Price      float64    `json:"price"`
	Kind       PivotKind  `json:"kind"`
	State      PivotState `json:"state"`
	Prominence float64    `json:"prominence"`
}

type PatternType string

const (
	PatternImpulse             PatternType = "IMPULSE"
	PatternLeadingDiagonal     PatternType = "LEADING_DIAGONAL"
	PatternEndingDiagonal      PatternType = "ENDING_DIAGONAL"
	PatternTruncatedImpulse    PatternType = "TRUNCATED_IMPULSE"
	PatternZigzag              PatternType = "ZIGZAG"
	PatternDoubleZigzag        PatternType = "DOUBLE_ZIGZAG"
	PatternTripleZigzag        PatternType = "TRIPLE_ZIGZAG"
	PatternFlatRegular         PatternType = "FLAT_REGULAR"
	PatternFlatExpanded        PatternType = "FLAT_EXPANDED"
	PatternFlatRunning         PatternType = "FLAT_RUNNING"
	PatternTriangleContracting PatternType = "TRIANGLE_CONTRACTING"
	PatternTriangleAscending   PatternType = "TRIANGLE_ASCENDING"
	PatternTriangleDescending  PatternType = "TRIANGLE_DESCENDING"
	PatternTriangleRunning     PatternType = "TRIANGLE_RUNNING"
	PatternTriangleExpanding   PatternType = "TRIANGLE_EXPANDING"
	PatternDoubleThree         PatternType = "DOUBLE_THREE"
	PatternTripleThree         PatternType = "TRIPLE_THREE"
	PatternDevelopingImpulseW2 PatternType = "DEVELOPING_IMPULSE_EXPECTING_2"
	PatternDevelopingImpulseW3 PatternType = "DEVELOPING_IMPULSE_EXPECTING_3"
	PatternDevelopingImpulseW4 PatternType = "DEVELOPING_IMPULSE_EXPECTING_4"
	PatternDevelopingImpulseW5 PatternType = "DEVELOPING_IMPULSE_EXPECTING_5"
	PatternDevelopingZigzagC   PatternType = "DEVELOPING_ZIGZAG_EXPECTING_C"
	PatternDevelopingFlatC     PatternType = "DEVELOPING_FLAT_EXPECTING_C"
	PatternDevelopingTriangleD PatternType = "DEVELOPING_TRIANGLE_EXPECTING_D"
	PatternDevelopingTriangleE PatternType = "DEVELOPING_TRIANGLE_EXPECTING_E"
)

type WaveMode string

const (
	ModeMotive     WaveMode = "MOTIVE"
	ModeCorrective WaveMode = "CORRECTIVE"
)

type WaveFunction string

const (
	FunctionActionary   WaveFunction = "ACTIONARY"
	FunctionReactionary WaveFunction = "REACTIONARY"
)

type WaveStatus string

const (
	StatusCompleted     WaveStatus = "COMPLETED"
	StatusDeveloping    WaveStatus = "DEVELOPING"
	StatusIndeterminate WaveStatus = "INDETERMINATE"
)

type Degree string

const (
	DegreeGrandSupercycle Degree = "GRAND_SUPERCYCLE"
	DegreeSupercycle      Degree = "SUPERCYCLE"
	DegreeCycle           Degree = "CYCLE"
	DegreePrimary         Degree = "PRIMARY"
	DegreeIntermediate    Degree = "INTERMEDIATE"
	DegreeMinor           Degree = "MINOR"
	DegreeMinute          Degree = "MINUTE"
	DegreeMinuette        Degree = "MINUETTE"
	DegreeSubminuette     Degree = "SUBMINUETTE"
	DegreeObservableLeaf  Degree = "OBSERVABLE_LEAF"
)

type RuleClass string

const (
	RuleHard             RuleClass = "HARD_RULE"
	RuleGuideline        RuleClass = "GUIDELINE"
	RuleStatisticalPrior RuleClass = "STATISTICAL_PRIOR"
	RuleContext          RuleClass = "CONTEXT"
)

type EvaluationStatus string

const (
	EvaluationPass          EvaluationStatus = "PASS"
	EvaluationFail          EvaluationStatus = "FAIL"
	EvaluationNotApplicable EvaluationStatus = "NOT_APPLICABLE"
	EvaluationNotObservable EvaluationStatus = "NOT_OBSERVABLE"
)

//easyjson:json
type RuleEvaluation struct {
	RuleID   string           `json:"rule_id"`
	Class    RuleClass        `json:"class"`
	Status   EvaluationStatus `json:"status"`
	Source   string           `json:"source"`
	Summary  string           `json:"summary"`
	Measured float64          `json:"measured,omitempty"`
	Expected string           `json:"expected,omitempty"`
}

//easyjson:json
type Measurement struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// WaveNode is the recursive, auditable Elliott structure.
//
//easyjson:json
type WaveNode struct {
	ID              string           `json:"id"`
	Pattern         PatternType      `json:"pattern"`
	Mode            WaveMode         `json:"mode"`
	Function        WaveFunction     `json:"function"`
	Direction       Direction        `json:"direction"`
	Degree          Degree           `json:"degree"`
	Status          WaveStatus       `json:"status"`
	Label           string           `json:"label"`
	Level           int              `json:"level"`
	OrthodoxStart   Pivot            `json:"orthodox_start"`
	OrthodoxEnd     Pivot            `json:"orthodox_end"`
	Pivots          []Pivot          `json:"pivots"`
	Children        []WaveNode       `json:"children,omitempty"`
	Measurements    []Measurement    `json:"measurements,omitempty"`
	RuleEvaluations []RuleEvaluation `json:"rule_evaluations"`
	Conformance     Conformance      `json:"conformance"`
}

//easyjson:json
type Conformance struct {
	HardRulesPassed    int     `json:"hard_rules_passed"`
	HardRulesFailed    int     `json:"hard_rules_failed"`
	GuidelinesPassed   int     `json:"guidelines_passed"`
	GuidelinesFailed   int     `json:"guidelines_failed"`
	NotObservable      int     `json:"not_observable"`
	RatioConfluences   int     `json:"ratio_confluences"`
	StructuralCoverage float64 `json:"structural_coverage"`
	Score              float64 `json:"score"`
}

type InvalidationKind string

const (
	InvalidationPrice InvalidationKind = "PRICE"
	InvalidationRule  InvalidationKind = "RULE"
)

//easyjson:json
type Invalidation struct {
	ID          string           `json:"id"`
	Kind        InvalidationKind `json:"kind"`
	Price       float64          `json:"price,omitempty"`
	RuleID      string           `json:"rule_id,omitempty"`
	Description string           `json:"description"`
}

type TargetStatus string

const (
	TargetActive      TargetStatus = "ACTIVE"
	TargetConditional TargetStatus = "CONDITIONAL"
	TargetInvalidated TargetStatus = "INVALIDATED"
)

type ConfluenceGrade string

const (
	ConfluenceLine   ConfluenceGrade = "SINGLE_LEVEL"
	ConfluenceMedium ConfluenceGrade = "MEDIUM"
	ConfluenceHigh   ConfluenceGrade = "HIGH"
)

type TargetGeometry string

const (
	GeometryHorizontalBand TargetGeometry = "HORIZONTAL_BAND"
	GeometryChannelBand    TargetGeometry = "CHANNEL_POLYGON"
)

//easyjson:json
type TargetLevel struct {
	Price       float64 `json:"price"`
	Relation    string  `json:"relation"`
	Family      string  `json:"family"`
	Source      string  `json:"source"`
	Uncertainty float64 `json:"uncertainty"`
}

//easyjson:json
type TimeWindow struct {
	StartBarOffset int      `json:"start_bar_offset"`
	EndBarOffset   int      `json:"end_bar_offset"`
	StartTime      int64    `json:"start_time,omitempty"`
	EndTime        int64    `json:"end_time,omitempty"`
	Evidence       []string `json:"evidence"`
}

//easyjson:json
type GeometryPoint struct {
	BarOffset int     `json:"bar_offset"`
	Price     float64 `json:"price"`
}

//easyjson:json
type TargetZone struct {
	ID              string          `json:"id"`
	WaveLabel       string          `json:"wave_label"`
	Status          TargetStatus    `json:"status"`
	Condition       string          `json:"condition"`
	MinPrice        float64         `json:"min_price"`
	MaxPrice        float64         `json:"max_price"`
	Levels          []TargetLevel   `json:"levels"`
	Confluence      ConfluenceGrade `json:"confluence"`
	Geometry        TargetGeometry  `json:"geometry"`
	Points          []GeometryPoint `json:"points,omitempty"`
	TimeWindow      *TimeWindow     `json:"time_window,omitempty"`
	InvalidationIDs []string        `json:"invalidation_ids"`
}

type ScenarioStatus string

const (
	ScenarioPreferred     ScenarioStatus = "PREFERRED"
	ScenarioAlternate     ScenarioStatus = "ALTERNATE"
	ScenarioIndeterminate ScenarioStatus = "INDETERMINATE"
)

//easyjson:json
type Scenario struct {
	ID              string         `json:"id"`
	Rank            int            `json:"rank"`
	Status          ScenarioStatus `json:"status"`
	Bias            Direction      `json:"bias"`
	CurrentPosition string         `json:"current_position"`
	Conformance     Conformance    `json:"conformance"`
	Invalidations   []Invalidation `json:"invalidations"`
	Root            WaveNode       `json:"root"`
	TargetLadder    []TargetZone   `json:"target_ladder"`
}

//easyjson:json
type DataQuality struct {
	CandleCount         int      `json:"candle_count"`
	FirstTime           int64    `json:"first_time"`
	LastTime            int64    `json:"last_time"`
	MissingIntervals    int      `json:"missing_intervals"`
	AmbiguousPivotCount int      `json:"ambiguous_pivot_count"`
	Warnings            []string `json:"warnings,omitempty"`
}

//easyjson:json
type AnalysisResult struct {
	DataQuality DataQuality `json:"data_quality"`
	Scenarios   []Scenario  `json:"scenarios"`
	FutureBars  []int64     `json:"future_bars"`
}

// AnalyzeInput is intentionally independent from transport DTOs.
type AnalyzeInput struct {
	Candles      []market.Candle
	Timeframe    market.Timeframe
	Session      market.Session
	MaxScenarios int
	FutureBars   []int64
	TickSize     float64
}
