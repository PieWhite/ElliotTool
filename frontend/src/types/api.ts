export type Timeframe = '1m' | '5m' | '15m' | '1h' | '4h' | '1D' | '1W'
export type Session = 'RTH' | 'EXTENDED'
export type Direction = 'BULLISH' | 'BEARISH'
export type EvaluationStatus = 'PASS' | 'FAIL' | 'NOT_APPLICABLE' | 'NOT_OBSERVABLE'
export type RuleClass = 'HARD_RULE' | 'GUIDELINE' | 'STATISTICAL_PRIOR' | 'CONTEXT'
export type ScenarioStatus = 'PREFERRED' | 'ALTERNATE' | 'INDETERMINATE'
export type WaveStatus = 'COMPLETED' | 'DEVELOPING' | 'INDETERMINATE'
export type CoverageStatus = 'OBSERVED' | 'UNCERTAIN' | 'NOT_OBSERVABLE'
export type JobStatus =
  | 'QUEUED'
  | 'ACQUIRING_DAILY'
  | 'ACQUIRING_MINUTE'
  | 'AGGREGATING_VIEWS'
  | 'BUILDING_PIVOT_GRAPH'
  | 'PARSING_MASTER_TREE'
  | 'RANKING_SCENARIOS'
  | 'PERSISTING'
  | 'COMPLETED'
  | 'FAILED'

export interface AnalysisRequest {
  symbol: string
  session: Session
  as_of?: string
  focus_timeframe: Timeframe
  history_profile: 'MAX_DAILY_PLUS_2Y_MINUTE'
  max_scenarios: number
}

export interface Candle {
  time: number
  bar_index: number
  open: number
  high: number
  low: number
  close: number
  volume: number
}

export interface DerivedCandle extends Candle {
  high_time: number
  low_time: number
  source_from: number
  source_to: number
  provenance: 'MINUTE_DERIVED' | 'NATIVE_DAILY'
  partial: boolean
}

export interface Pivot {
  time: number
  bar_index: number
  price: number
  kind: 'HIGH' | 'LOW'
  state: 'CONFIRMED' | 'PROVISIONAL' | 'AMBIGUOUS'
  prominence: number
}

export interface RuleEvaluation {
  rule_id: string
  class: RuleClass
  status: EvaluationStatus
  source: string
  summary: string
  measured?: number
  expected?: string
}

export interface Measurement {
  name: string
  value: number
  unit: string
}

export interface Conformance {
  hard_rules_passed: number
  hard_rules_failed: number
  guidelines_passed: number
  guidelines_failed: number
  not_observable: number
  ratio_confluences: number
  structural_coverage: number
  score: number
}

export interface WaveNode {
  id: string
  pattern: string
  mode: 'MOTIVE' | 'CORRECTIVE'
  function: 'ACTIONARY' | 'REACTIONARY'
  direction: Direction
  degree: string
  status: WaveStatus
  label: string
  level: number
  orthodox_start: Pivot
  orthodox_end: Pivot
  pivots: Pivot[]
  children?: WaveNode[]
  measurements?: Measurement[]
  rule_evaluations: RuleEvaluation[]
  conformance: Conformance
}

export interface CanonicalWaveEvent {
  id: string
  kind: 'HIGH' | 'LOW'
  state: 'CONFIRMED' | 'PROVISIONAL' | 'OHLC_AMBIGUOUS'
  time_from: number
  time_to: number
  orthodox_time: number
  orthodox_price: number
  resolutions: Timeframe[]
  sources: Array<{
    timeframe: Timeframe
    bar_time: number
    price: number
    provenance: 'MINUTE_DERIVED' | 'NATIVE_DAILY'
  }>
  max_price_delta: number
  max_time_delta: number
}

export interface MasterWaveNode {
  id: string
  pattern: string
  mode: 'MOTIVE' | 'CORRECTIVE'
  function: 'ACTIONARY' | 'REACTIONARY'
  direction: Direction
  degree: string
  status: WaveStatus
  label: string
  start_event_id: string
  end_event_id: string
  pivot_event_ids: string[]
  child_ids: string[]
  resolutions: Timeframe[]
  orthodox_start: Pivot
  orthodox_end: Pivot
  measurements: Measurement[]
  rule_evaluations: RuleEvaluation[]
  conformance: Conformance
  source_node: WaveNode
}

export interface MasterWaveGraph {
  events: CanonicalWaveEvent[]
  nodes: MasterWaveNode[]
}

export interface Invalidation {
  id: string
  kind: 'PRICE' | 'RULE'
  price?: number
  rule_id?: string
  description: string
}

export interface TargetLevel {
  price: number
  relation: string
  family: string
  source: string
  uncertainty: number
}

export interface TargetZone {
  id: string
  wave_label: string
  status: 'ACTIVE' | 'CONDITIONAL' | 'INVALIDATED'
  condition: string
  min_price: number
  max_price: number
  levels: TargetLevel[]
  confluence: 'SINGLE_LEVEL' | 'MEDIUM' | 'HIGH'
  geometry: 'HORIZONTAL_BAND' | 'CHANNEL_POLYGON'
  points?: Array<{ bar_offset: number; price: number }>
  time_window?: {
    start_bar_offset: number
    end_bar_offset: number
    start_time?: number
    end_time?: number
    evidence: string[]
  }
  invalidation_ids: string[]
}

export interface TimeframeEvidence {
  timeframe: Timeframe
  position: string
  parent_node_id?: string
  visible_children: number
  endpoint_aligned: boolean
  coverage: CoverageStatus
  status: string
}

export interface MasterScenario {
  id: string
  rank: number
  status: ScenarioStatus
  bias: Direction
  current_position: string
  conformance: Conformance
  observation_root: {
    from: number
    to: number
    open_left_boundary: boolean
    context_sequence: string[]
    intervals: Array<{
      from: number
      to: number
      status: CoverageStatus
      node_id?: string
      explanation: string
    }>
  }
  active_path: string[]
  invalidations: Invalidation[]
  target_ladder: TargetZone[]
  audit: {
    global_thesis: string
    cross_timeframe_evidence: TimeframeEvidence[]
    why_preferred?: {
      alternative_id: string
      first_divergence: string
      preferred_evidence: string[]
      different_targets: boolean
      different_bias: boolean
    }
  }
  material_signature: string
}

export interface DatasetManifest {
  coverage: Array<{
    resolution: 'DAILY_NATIVE' | 'MINUTE_NATIVE'
    from: number
    to: number
    complete: boolean
  }>
  provider_queries: Array<{
    resolution: 'DAILY_NATIVE' | 'MINUTE_NATIVE'
    from: number
    to: number
    logical_query: boolean
    page_requests: number
    rows: number
    cache_only: boolean
    overlap_changed: boolean
  }>
  daily_provenance: {
    compared: number
    differences: number
    max_ohlc_deviation: number
    samples: Array<{
      date: string
      native_time: number
      derived_time: number
      max_ohlc_deviation: number
      volume_deviation: number
    }>
  }
  minute_detail_from: number
  minute_detail_to: number
  native_daily_rows: number
  native_minute_rows: number
}

export interface TimeframeView {
  snapshot_id: string
  timeframe: Timeframe
  candles: DerivedCandle[]
  visible_node_ids: string[]
  ancestor_node_ids: string[]
  future_logical_bars: number[]
  coverage: {
    from: number
    to: number
    detail_from: number
    status: CoverageStatus
    message: string
  }
}

export interface AnalysisSnapshot {
  id: string
  parent_snapshot_id?: string
  theory_version: string
  engine_version: string
  generated_at: number
  request: AnalysisRequest
  dataset_manifest: DatasetManifest
  master_wave_graph: MasterWaveGraph
  scenarios: MasterScenario[]
  view_manifest: Array<{
    timeframe: Timeframe
    candle_count: number
    from: number
    to: number
  }>
  initial_view: TimeframeView
}

export interface AnalysisJob {
  id: string
  status: JobStatus
  progress: number
  message: string
  snapshot_id?: string
  error?: string
  request: AnalysisRequest
  created_at: number
  updated_at: number
}

export interface SnapshotMetadata {
  id: string
  parent_snapshot_id?: string
  symbol: string
  session: Session
  as_of: number
  generated_at: number
  focus_timeframe: Timeframe
  theory_version: string
  engine_version: string
  request_key: string
  data_fingerprint: string
}

export interface SnapshotHistory {
  items: SnapshotMetadata[]
}

export interface ProblemResponse {
  type: string
  title: string
  status: number
  detail: string
  request_id: string
}

export function isProblemResponse(value: unknown): value is ProblemResponse {
  if (typeof value !== 'object' || value === null) return false
  const candidate = value as Record<string, unknown>
  return typeof candidate.title === 'string' &&
    typeof candidate.detail === 'string' &&
    typeof candidate.status === 'number'
}
