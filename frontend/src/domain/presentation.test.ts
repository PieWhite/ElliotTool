import { describe, expect, it } from 'vitest'
import { conformanceVector, scenarioDisplayName, visibleTargetZones } from './presentation'
import type { MasterScenario, TargetZone } from '../types/api'

const target = (id: string, status: TargetZone['status']): TargetZone => ({
  id,
  wave_label: 'W5',
  status,
  condition: 'Wave 4 remains valid',
  min_price: 120,
  max_price: 125,
  levels: [],
  confluence: 'MEDIUM',
  geometry: 'HORIZONTAL_BAND',
  invalidation_ids: [],
})

const scenario: MasterScenario = {
  id: 'scenario-1',
  rank: 1,
  status: 'PREFERRED',
  bias: 'BULLISH',
  current_position: 'Primary [3]',
  conformance: {
    hard_rules_passed: 8,
    hard_rules_failed: 0,
    guidelines_passed: 5,
    guidelines_failed: 2,
    not_observable: 1,
    ratio_confluences: 3,
    structural_coverage: 1,
    score: 0.92,
  },
  observation_root: {
    from: 1,
    to: 2,
    open_left_boundary: true,
    context_sequence: [],
    intervals: [],
  },
  active_path: [],
  invalidations: [],
  target_ladder: [target('active', 'ACTIVE'), target('invalid', 'INVALIDATED')],
  audit: {
    global_thesis: 'Primary [3]',
    cross_timeframe_evidence: [],
  },
  material_signature: 'signature',
}

describe('scenario presentation', () => {
  it('uses rank labels without inventing probability wording', () => {
    expect(scenarioDisplayName(scenario)).toBe('Preferred')
    expect(scenarioDisplayName({ ...scenario, status: 'ALTERNATE', rank: 3 })).toBe('Alternate 2')
  })

  it('shows a conformance vector instead of a probability', () => {
    expect(conformanceVector(scenario.conformance)).toEqual([
      { label: 'Hard rules', passed: 8, failed: 0 },
      { label: 'Guidelines', passed: 5, failed: 2 },
      { label: 'Ratio evidence', passed: 3, failed: 0 },
      { label: 'Not observable', passed: 1, failed: 0 },
    ])
  })

  it('removes invalidated target zones from the active ladder', () => {
    expect(visibleTargetZones(scenario).map((item) => item.id)).toEqual(['active'])
  })
})
