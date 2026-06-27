import { describe, expect, it } from 'vitest'
import { conformanceVector, scenarioDisplayName, visibleTargetZones } from './presentation'
import type { Scenario, TargetZone } from '../types/api'

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

const scenario: Scenario = {
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
  invalidations: [],
  root: {
    id: 'root',
    pattern: 'IMPULSE',
    mode: 'MOTIVE',
    function: 'ACTIONARY',
    direction: 'BULLISH',
    degree: 'PRIMARY',
    status: 'COMPLETED',
    label: 'Impulse',
    level: 1,
    orthodox_start: { time: 1, bar_index: 0, price: 100, kind: 'LOW', state: 'CONFIRMED', prominence: 1 },
    orthodox_end: { time: 2, bar_index: 1, price: 120, kind: 'HIGH', state: 'CONFIRMED', prominence: 1 },
    pivots: [],
    rule_evaluations: [],
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
  },
  target_ladder: [target('active', 'ACTIVE'), target('invalid', 'INVALIDATED')],
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
