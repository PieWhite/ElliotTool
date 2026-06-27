import { expect, test } from '@playwright/test'
import type { AnalysisSnapshot, Pivot, Scenario, WaveNode } from '../src/types/api'

const analysisID = '0123456789abcdef0123456789abcdef'
const startTime = 1_735_853_400
const candles = Array.from({ length: 80 }, (_, index) => {
  const center = 100 + index * 0.35 + Math.sin(index / 5) * 4
  return {
    time: startTime + index * 86_400,
    bar_index: index,
    open: center - 0.4,
    high: center + 1.8,
    low: center - 1.7,
    close: center + 0.6,
    volume: 1_000_000 + index * 12_000,
  }
})

function pivot(index: number, price: number, kind: Pivot['kind']): Pivot {
  return {
    time: candles[index].time,
    bar_index: index,
    price,
    kind,
    state: 'CONFIRMED',
    prominence: 3,
  }
}

const conformance = {
  hard_rules_passed: 9,
  hard_rules_failed: 0,
  guidelines_passed: 6,
  guidelines_failed: 1,
  not_observable: 1,
  ratio_confluences: 3,
  structural_coverage: 1,
  score: 0.94,
}

function root(direction: Scenario['bias'], offset: number): WaveNode {
  const points = direction === 'BULLISH'
    ? [pivot(15, 101 + offset, 'LOW'), pivot(25, 116 + offset, 'HIGH'), pivot(34, 108 + offset, 'LOW'), pivot(49, 135 + offset, 'HIGH'), pivot(61, 124 + offset, 'LOW')]
    : [pivot(15, 134 + offset, 'HIGH'), pivot(25, 119 + offset, 'LOW'), pivot(34, 127 + offset, 'HIGH'), pivot(49, 105 + offset, 'LOW'), pivot(61, 115 + offset, 'HIGH')]
  return {
    id: `root-${direction}`,
    pattern: 'DEVELOPING_IMPULSE_EXPECTING_5',
    mode: 'MOTIVE',
    function: 'ACTIONARY',
    direction,
    degree: 'INTERMEDIATE',
    status: 'DEVELOPING',
    label: 'Developing impulse',
    level: 2,
    orthodox_start: points[0],
    orthodox_end: points[points.length - 1],
    pivots: points,
    children: [],
    measurements: [],
    rule_evaluations: [
      {
        rule_id: 'EWP-MOTIVE-W2-LIMIT',
        class: 'HARD_RULE',
        status: 'PASS',
        source: 'EWP p.12',
        summary: 'Wave 2 remains within the wave 1 origin.',
        measured: 0.53,
        expected: '<= 1.0',
      },
      {
        rule_id: 'EWP-GUIDE-VOLUME',
        class: 'GUIDELINE',
        status: 'PASS',
        source: 'EWP p.42',
        summary: 'Third-wave volume expanded.',
        measured: 1.42,
        expected: '>= 1.0',
      },
    ],
    conformance,
  }
}

const scenarios: Scenario[] = [
  {
    id: 'scenario-bullish',
    rank: 1,
    status: 'PREFERRED',
    bias: 'BULLISH',
    current_position: 'Intermediate (4) → expecting Minor 5',
    conformance,
    invalidations: [{ id: 'wave-4-extreme', kind: 'PRICE', price: 121, description: 'Wave 5 fails below the wave 4 extreme.' }],
    root: root('BULLISH', 0),
    target_ladder: [{
      id: 'target-w5',
      wave_label: 'W5',
      status: 'CONDITIONAL',
      condition: 'Wave 4 remains valid and the fifth wave is underway',
      min_price: 142,
      max_price: 146,
      levels: [
        { price: 143, relation: '1.000 × W1 from W4', family: 'W5_VS_W1', source: 'EWP p.73', uncertainty: 1 },
        { price: 145, relation: '0.618 × 0→3 from W4', family: 'W5_VS_ZERO_THREE', source: 'EWP p.73', uncertainty: 1 },
      ],
      confluence: 'MEDIUM',
      geometry: 'HORIZONTAL_BAND',
      time_window: {
        start_bar_offset: 10,
        end_bar_offset: 12,
        start_time: startTime + 90 * 86_400,
        end_time: startTime + 92 * 86_400,
        evidence: ['0.618 duration', 'alternate-wave equality'],
      },
      invalidation_ids: ['wave-4-extreme'],
    }],
  },
  {
    id: 'scenario-bearish',
    rank: 2,
    status: 'ALTERNATE',
    bias: 'BEARISH',
    current_position: 'Intermediate (B) → expecting Minor C',
    conformance: { ...conformance, guidelines_passed: 4, score: 0.84 },
    invalidations: [{ id: 'wave-b-high', kind: 'PRICE', price: 141, description: 'The corrective B interpretation fails above this price.' }],
    root: root('BEARISH', 0),
    target_ladder: [],
  },
]

const snapshot: AnalysisSnapshot = {
  id: analysisID,
  theory_version: 'ewp-frost-prechter+waveratios-1.0.0',
  engine_version: '2.0.0',
  generated_at: 1_782_559_200,
  request: {
    symbol: 'AAPL',
    timeframe: '1D',
    session: 'RTH',
    as_of: '2026-06-27T00:00:00Z',
    lookback_bars: 2000,
    max_scenarios: 5,
  },
  data_quality: {
    candle_count: candles.length,
    first_time: candles[0].time,
    last_time: candles[candles.length - 1].time,
    missing_intervals: 0,
    ambiguous_pivot_count: 0,
  },
  candles,
  scenarios,
  future_bars: Array.from({ length: 100 }, (_, index) => candles[candles.length - 1].time + (index + 1) * 86_400),
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v2/analyses**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (request.method() === 'POST') {
      await route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(snapshot) })
      return
    }
    if (url.pathname.endsWith(`/${analysisID}`)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(snapshot) })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [{
          id: analysisID,
          symbol: 'AAPL',
          timeframe: '1D',
          session: 'RTH',
          as_of: 1_782_518_400,
          generated_at: snapshot.generated_at,
          theory_version: snapshot.theory_version,
          engine_version: snapshot.engine_version,
          request_hash: 'request-hash',
          data_fingerprint: 'data-fingerprint',
        }],
      }),
    })
  })
})

test('scans, ranks, compares and exports a snapshot', async ({ page }) => {
  await page.goto('/')
  await page.getByRole('button', { name: 'Scan market' }).click()

  await expect(page.getByText('Preferred', { exact: true })).toBeVisible()
  await expect(page.getByText('Purple Box targets')).toBeVisible()
  await expect(page.getByText('$142 — $146')).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`/analysis/${analysisID}$`))

  const alternate = page.locator('.scenario-card').filter({ hasText: 'Alternate 1' })
  await alternate.getByRole('button').first().click()
  await expect(page.getByRole('heading', { name: 'Intermediate (B) → expecting Minor C' })).toBeVisible()

  const preferred = page.locator('.scenario-card').filter({ hasText: 'Preferred' })
  await preferred.getByRole('button', { name: 'Compare' }).click()
  await expect(preferred.getByRole('button', { name: 'Comparing' })).toBeVisible()

  await page.getByRole('button', { name: 'Log', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Log', exact: true })).toHaveClass(/active/)

  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: 'JSON', exact: true }).click()
  expect((await download).suggestedFilename()).toContain(`wavesight-AAPL-${analysisID}`)
})

test('loads a permanent share URL and remains usable on mobile', async ({ page }) => {
  await page.goto(`/analysis/${analysisID}`)
  await expect(page.getByRole('heading', { name: 'Intermediate (4) → expecting Minor 5' })).toBeVisible()
  await expect(page.locator('.chart-shell')).toBeVisible()
  await page.getByRole('button', { name: 'History' }).click()
  await expect(page.getByRole('heading', { name: 'Scan history' })).toBeVisible()
})
