import type { Conformance, Scenario, TargetZone } from '../types/api'

export interface ConformanceDatum {
  label: string
  passed: number
  failed: number
}

export function scenarioDisplayName(scenario: Scenario): string {
  return scenario.status === 'PREFERRED'
    ? 'Preferred'
    : scenario.status === 'INDETERMINATE'
      ? 'Indeterminate'
      : `Alternate ${Math.max(1, scenario.rank - 1)}`
}

export function conformanceVector(value: Conformance): ConformanceDatum[] {
  return [
    { label: 'Hard rules', passed: value.hard_rules_passed, failed: value.hard_rules_failed },
    { label: 'Guidelines', passed: value.guidelines_passed, failed: value.guidelines_failed },
    { label: 'Ratio evidence', passed: value.ratio_confluences, failed: 0 },
    { label: 'Not observable', passed: value.not_observable, failed: 0 },
  ]
}

export function visibleTargetZones(scenario: Scenario | null): TargetZone[] {
  if (!scenario) return []
  return scenario.target_ladder.filter((target) => target.status !== 'INVALIDATED')
}
