package master

import (
	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
)

const EngineVersion = "3.0.0"

type HistoryProfile string

const HistoryMaxDailyTwoYearMinute HistoryProfile = "MAX_DAILY_PLUS_2Y_MINUTE"

type CoverageStatus string

const (
	CoverageObserved      CoverageStatus = "OBSERVED"
	CoverageUncertain     CoverageStatus = "UNCERTAIN"
	CoverageNotObservable CoverageStatus = "NOT_OBSERVABLE"
)

type JobStatus string

const (
	JobQueued             JobStatus = "QUEUED"
	JobAcquiringDaily     JobStatus = "ACQUIRING_DAILY"
	JobAcquiringMinute    JobStatus = "ACQUIRING_MINUTE"
	JobAggregatingViews   JobStatus = "AGGREGATING_VIEWS"
	JobBuildingPivotGraph JobStatus = "BUILDING_PIVOT_GRAPH"
	JobParsingMasterTree  JobStatus = "PARSING_MASTER_TREE"
	JobRankingScenarios   JobStatus = "RANKING_SCENARIOS"
	JobPersisting         JobStatus = "PERSISTING"
	JobCompleted          JobStatus = "COMPLETED"
	JobFailed             JobStatus = "FAILED"
)

type NativeResolution string

const (
	NativeDaily  NativeResolution = "DAILY_NATIVE"
	NativeMinute NativeResolution = "MINUTE_NATIVE"
)

type AnalysisRequest struct {
	Symbol         string           `json:"symbol"`
	Session        market.Session   `json:"session"`
	AsOf           string           `json:"as_of"`
	FocusTimeframe market.Timeframe `json:"focus_timeframe"`
	HistoryProfile HistoryProfile   `json:"history_profile"`
	MaxScenarios   int              `json:"max_scenarios"`
}

type CoverageInterval struct {
	Resolution NativeResolution `json:"resolution"`
	From       int64            `json:"from"`
	To         int64            `json:"to"`
	Complete   bool             `json:"complete"`
}

type ProviderQueryTelemetry struct {
	Resolution     NativeResolution `json:"resolution"`
	From           int64            `json:"from"`
	To             int64            `json:"to"`
	LogicalQuery   bool             `json:"logical_query"`
	PageRequests   int              `json:"page_requests"`
	Rows           int              `json:"rows"`
	CacheOnly      bool             `json:"cache_only"`
	OverlapChanged bool             `json:"overlap_changed"`
}

type DatasetManifest struct {
	Coverage         []CoverageInterval       `json:"coverage"`
	ProviderQueries  []ProviderQueryTelemetry `json:"provider_queries"`
	DailyProvenance  DailyProvenanceAudit     `json:"daily_provenance"`
	MinuteDetailFrom int64                    `json:"minute_detail_from"`
	MinuteDetailTo   int64                    `json:"minute_detail_to"`
	NativeDailyRows  int                      `json:"native_daily_rows"`
	NativeMinuteRows int                      `json:"native_minute_rows"`
}

type DailyBarDifference struct {
	Date             string  `json:"date"`
	NativeTime       int64   `json:"native_time"`
	DerivedTime      int64   `json:"derived_time"`
	MaxOHLCDeviation float64 `json:"max_ohlc_deviation"`
	VolumeDeviation  float64 `json:"volume_deviation"`
}

type DailyProvenanceAudit struct {
	Compared         int                  `json:"compared"`
	Differences      int                  `json:"differences"`
	MaxOHLCDeviation float64              `json:"max_ohlc_deviation"`
	Samples          []DailyBarDifference `json:"samples"`
}

type EventState string

const (
	EventConfirmed   EventState = "CONFIRMED"
	EventProvisional EventState = "PROVISIONAL"
	EventAmbiguous   EventState = "OHLC_AMBIGUOUS"
)

type EventSource struct {
	Timeframe  market.Timeframe     `json:"timeframe"`
	BarTime    int64                `json:"bar_time"`
	Price      float64              `json:"price"`
	Provenance market.BarProvenance `json:"provenance"`
}

type CanonicalWaveEvent struct {
	ID            string             `json:"id"`
	Kind          wave.PivotKind     `json:"kind"`
	State         EventState         `json:"state"`
	TimeFrom      int64              `json:"time_from"`
	TimeTo        int64              `json:"time_to"`
	OrthodoxTime  int64              `json:"orthodox_time"`
	OrthodoxPrice float64            `json:"orthodox_price"`
	Resolutions   []market.Timeframe `json:"resolutions"`
	Sources       []EventSource      `json:"sources"`
	MaxPriceDelta float64            `json:"max_price_delta"`
	MaxTimeDelta  int64              `json:"max_time_delta"`
}

type MasterWaveNode struct {
	ID              string                `json:"id"`
	Pattern         wave.PatternType      `json:"pattern"`
	Mode            wave.WaveMode         `json:"mode"`
	Function        wave.WaveFunction     `json:"function"`
	Direction       wave.Direction        `json:"direction"`
	Degree          wave.Degree           `json:"degree"`
	Status          wave.WaveStatus       `json:"status"`
	Label           string                `json:"label"`
	StartEventID    string                `json:"start_event_id"`
	EndEventID      string                `json:"end_event_id"`
	PivotEventIDs   []string              `json:"pivot_event_ids"`
	ChildIDs        []string              `json:"child_ids"`
	Resolutions     []market.Timeframe    `json:"resolutions"`
	OrthodoxStart   wave.Pivot            `json:"orthodox_start"`
	OrthodoxEnd     wave.Pivot            `json:"orthodox_end"`
	Measurements    []wave.Measurement    `json:"measurements"`
	RuleEvaluations []wave.RuleEvaluation `json:"rule_evaluations"`
	Conformance     wave.Conformance      `json:"conformance"`
	SourceNode      wave.WaveNode         `json:"source_node"`
}

type MasterWaveGraph struct {
	Events []CanonicalWaveEvent `json:"events"`
	Nodes  []MasterWaveNode     `json:"nodes"`
}

type NarrativeInterval struct {
	From        int64          `json:"from"`
	To          int64          `json:"to"`
	Status      CoverageStatus `json:"status"`
	NodeID      string         `json:"node_id,omitempty"`
	Explanation string         `json:"explanation"`
}

type ObservationRoot struct {
	From             int64               `json:"from"`
	To               int64               `json:"to"`
	OpenLeftBoundary bool                `json:"open_left_boundary"`
	ContextSequence  []string            `json:"context_sequence"`
	Intervals        []NarrativeInterval `json:"intervals"`
}

type TimeframeEvidence struct {
	Timeframe       market.Timeframe `json:"timeframe"`
	Position        string           `json:"position"`
	ParentNodeID    string           `json:"parent_node_id,omitempty"`
	VisibleChildren int              `json:"visible_children"`
	EndpointAligned bool             `json:"endpoint_aligned"`
	Coverage        CoverageStatus   `json:"coverage"`
	Status          string           `json:"status"`
}

type AlternativeComparison struct {
	AlternativeID     string   `json:"alternative_id"`
	FirstDivergence   string   `json:"first_divergence"`
	PreferredEvidence []string `json:"preferred_evidence"`
	DifferentTargets  bool     `json:"different_targets"`
	DifferentBias     bool     `json:"different_bias"`
}

type ScenarioAudit struct {
	GlobalThesis           string                 `json:"global_thesis"`
	CrossTimeframeEvidence []TimeframeEvidence    `json:"cross_timeframe_evidence"`
	WhyPreferred           *AlternativeComparison `json:"why_preferred,omitempty"`
}

type MasterScenario struct {
	ID                string              `json:"id"`
	Rank              int                 `json:"rank"`
	Status            wave.ScenarioStatus `json:"status"`
	Bias              wave.Direction      `json:"bias"`
	CurrentPosition   string              `json:"current_position"`
	Conformance       wave.Conformance    `json:"conformance"`
	ObservationRoot   ObservationRoot     `json:"observation_root"`
	ActivePath        []string            `json:"active_path"`
	Invalidations     []wave.Invalidation `json:"invalidations"`
	TargetLadder      []wave.TargetZone   `json:"target_ladder"`
	Audit             ScenarioAudit       `json:"audit"`
	MaterialSignature string              `json:"material_signature"`
}

type ViewCoverage struct {
	From       int64          `json:"from"`
	To         int64          `json:"to"`
	DetailFrom int64          `json:"detail_from"`
	Status     CoverageStatus `json:"status"`
	Message    string         `json:"message"`
}

type ViewManifestItem struct {
	Timeframe   market.Timeframe `json:"timeframe"`
	CandleCount int              `json:"candle_count"`
	From        int64            `json:"from"`
	To          int64            `json:"to"`
}

type TimeframeView struct {
	SnapshotID        string                 `json:"snapshot_id"`
	Timeframe         market.Timeframe       `json:"timeframe"`
	Candles           []market.DerivedCandle `json:"candles"`
	VisibleNodeIDs    []string               `json:"visible_node_ids"`
	AncestorNodeIDs   []string               `json:"ancestor_node_ids"`
	FutureLogicalBars []int64                `json:"future_logical_bars"`
	Coverage          ViewCoverage           `json:"coverage"`
}

type AnalysisSnapshot struct {
	ID               string             `json:"id"`
	ParentSnapshotID string             `json:"parent_snapshot_id,omitempty"`
	TheoryVersion    string             `json:"theory_version"`
	EngineVersion    string             `json:"engine_version"`
	GeneratedAt      int64              `json:"generated_at"`
	Request          AnalysisRequest    `json:"request"`
	DatasetManifest  DatasetManifest    `json:"dataset_manifest"`
	Graph            MasterWaveGraph    `json:"master_wave_graph"`
	Scenarios        []MasterScenario   `json:"scenarios"`
	ViewManifest     []ViewManifestItem `json:"view_manifest"`
	InitialView      TimeframeView      `json:"initial_view"`
}

type AnalysisJob struct {
	ID         string          `json:"id"`
	Status     JobStatus       `json:"status"`
	Progress   int             `json:"progress"`
	Message    string          `json:"message"`
	SnapshotID string          `json:"snapshot_id,omitempty"`
	Error      string          `json:"error,omitempty"`
	Request    AnalysisRequest `json:"request"`
	CreatedAt  int64           `json:"created_at"`
	UpdatedAt  int64           `json:"updated_at"`
}

type RefinementRequest struct {
	From         string `json:"from"`
	To           string `json:"to"`
	ParentNodeID string `json:"parent_node_id,omitempty"`
}

type SnapshotMetadata struct {
	ID               string           `json:"id"`
	ParentSnapshotID string           `json:"parent_snapshot_id,omitempty"`
	Symbol           string           `json:"symbol"`
	Session          market.Session   `json:"session"`
	AsOf             int64            `json:"as_of"`
	GeneratedAt      int64            `json:"generated_at"`
	FocusTimeframe   market.Timeframe `json:"focus_timeframe"`
	TheoryVersion    string           `json:"theory_version"`
	EngineVersion    string           `json:"engine_version"`
}
