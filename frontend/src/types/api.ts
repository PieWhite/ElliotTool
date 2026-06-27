export type Timeframe = '1m' | '5m' | '15m' | '1h' | '4h' | '1D' | '1W'
export type Session = 'RTH' | 'EXTENDED'
export type Direction = 'BULLISH' | 'BEARISH'
export type EvaluationStatus = 'PASS' | 'FAIL' | 'NOT_APPLICABLE' | 'NOT_OBSERVABLE'
export type RuleClass = 'HARD_RULE' | 'GUIDELINE' | 'STATISTICAL_PRIOR' | 'CONTEXT'
export type ScenarioStatus = 'PREFERRED' | 'ALTERNATE' | 'INDETERMINATE'
export type WaveStatus = 'COMPLETED' | 'DEVELOPING' | 'INDETERMINATE'
export type ConfluenceGrade = 'SINGLE_LEVEL' | 'MEDIUM' | 'HIGH'
export type TargetStatus = 'ACTIVE' | 'CONDITIONAL' | 'INVALIDATED'

export interface AnalysisRequest {
  symbol: string
  timeframe: Timeframe
  session: Session
  as_of?: string
  lookback_bars?: number
  max_scenarios?: number
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
  expected: string
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

export interface TimeWindow {
  start_bar_offset: number
  end_bar_offset: number
  start_time?: number
  end_time?: number
  evidence: string[]
}

export interface GeometryPoint {
  bar_offset: number
  price: number
}

export interface TargetZone {
  id: string
  wave_label: string
  status: TargetStatus
  condition: string
  min_price: number
  max_price: number
  levels: TargetLevel[]
  confluence: ConfluenceGrade
  geometry: 'HORIZONTAL_BAND' | 'CHANNEL_POLYGON'
  points?: GeometryPoint[]
  time_window?: TimeWindow
  invalidation_ids: string[]
}

export interface Scenario {
  id: string
  rank: number
  status: ScenarioStatus
  bias: Direction
  current_position: string
  conformance: Conformance
  invalidations: Invalidation[]
  root: WaveNode
  target_ladder: TargetZone[]
}

export interface DataQuality {
  candle_count: number
  first_time: number
  last_time: number
  missing_intervals: number
  ambiguous_pivot_count: number
  warnings?: string[]
}

export interface AnalysisSnapshot {
  id: string
  theory_version: string
  engine_version: string
  generated_at: number
  request: AnalysisRequest
  data_quality: DataQuality
  candles: Candle[]
  scenarios: Scenario[]
  future_bars: number[]
}

export interface SnapshotMetadata {
  id: string
  symbol: string
  timeframe: string
  session: string
  as_of: number
  generated_at: number
  theory_version: string
  engine_version: string
  request_hash: string
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
