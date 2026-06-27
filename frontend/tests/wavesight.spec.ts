import { expect, test } from '@playwright/test'
import type {
  AnalysisSnapshot,
  CanonicalWaveEvent,
  DerivedCandle,
  MasterScenario,
  MasterWaveNode,
  Pivot,
  Timeframe,
  TimeframeView,
} from '../src/types/api'

const analysisID = '0123456789abcdef0123456789abcdef'
const revisionID = 'fedcba9876543210fedcba9876543210'
const startTime = 1_735_853_400
const timeframes: Timeframe[] = ['1m', '5m', '15m', '1h', '4h', '1D', '1W']
const candles: DerivedCandle[] = Array.from({ length: 80 }, (_, index) => {
  const center = 100 + index * 0.35 + Math.sin(index / 5) * 4
  const time = startTime + index * 86_400
  return {
    time,
    bar_index: index,
    open: center - 0.4,
    high: center + 1.8,
    low: center - 1.7,
    close: center + 0.6,
    volume: 1_000_000 + index * 12_000,
    high_time: time,
    low_time: time,
    source_from: time,
    source_to: time,
    provenance: 'MINUTE_DERIVED',
    partial: false,
  }
})

const conformance = {
  hard_rules_passed: 9,
  hard_rules_failed: 0,
  guidelines_passed: 6,
  guidelines_failed: 1,
  not_observable: 1,
  ratio_confluences: 3,
  structural_coverage: 0.94,
  score: 0.94,
}

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

const points = [
  pivot(15, 101, 'LOW'),
  pivot(25, 116, 'HIGH'),
  pivot(34, 108, 'LOW'),
  pivot(49, 135, 'HIGH'),
  pivot(61, 124, 'LOW'),
]

const events: CanonicalWaveEvent[] = points.map((point, index) => ({
  id: `event-${index}`,
  kind: point.kind,
  state: 'CONFIRMED',
  time_from: point.time,
  time_to: point.time,
  orthodox_time: point.time,
  orthodox_price: point.price,
  resolutions: timeframes,
  sources: [{
    timeframe: '1m',
    bar_time: point.time,
    price: point.price,
    provenance: 'MINUTE_DERIVED',
  }],
  max_price_delta: 0,
  max_time_delta: 0,
}))
events.unshift(
  {
    id: 'event-old-start',
    kind: 'LOW',
    state: 'CONFIRMED',
    time_from: 1_577_836_800,
    time_to: 1_577_836_800,
    orthodox_time: 1_577_836_800,
    orthodox_price: 60,
    resolutions: ['1W', '1D'],
    sources: [],
    max_price_delta: 0,
    max_time_delta: 0,
  },
  {
    id: 'event-old-end',
    kind: 'HIGH',
    state: 'CONFIRMED',
    time_from: 1_609_459_200,
    time_to: 1_609_459_200,
    orthodox_time: 1_609_459_200,
    orthodox_price: 90,
    resolutions: ['1W', '1D'],
    sources: [],
    max_price_delta: 0,
    max_time_delta: 0,
  },
)

function sourceNode(id: string, nodePoints: Pivot[], direction: 'BULLISH' | 'BEARISH') {
  return {
    id,
    pattern: 'DEVELOPING_IMPULSE_EXPECTING_5',
    mode: 'MOTIVE' as const,
    function: 'ACTIONARY' as const,
    direction,
    degree: 'INTERMEDIATE',
    status: 'DEVELOPING' as const,
    label: 'Developing impulse',
    level: 2,
    orthodox_start: nodePoints[0],
    orthodox_end: nodePoints[nodePoints.length - 1],
    pivots: nodePoints,
    children: [],
    measurements: [],
    rule_evaluations: [{
      rule_id: 'EWP-MOTIVE-W2-LIMIT',
      class: 'HARD_RULE' as const,
      status: 'PASS' as const,
      source: 'EWP p.12',
      summary: 'Wave 2 remains within the wave 1 origin.',
      measured: 0.53,
      expected: '<= 1.0',
    }],
    conformance,
  }
}

const activeNode: MasterWaveNode = {
  id: 'wave-active',
  pattern: 'DEVELOPING_IMPULSE_EXPECTING_5',
  mode: 'MOTIVE',
  function: 'ACTIONARY',
  direction: 'BULLISH',
  degree: 'INTERMEDIATE',
  status: 'DEVELOPING',
  label: 'Developing impulse',
  start_event_id: 'event-0',
  end_event_id: 'event-4',
  pivot_event_ids: ['event-0', 'event-1', 'event-2', 'event-3', 'event-4'],
  child_ids: [],
  resolutions: timeframes,
  orthodox_start: points[0],
  orthodox_end: points[4],
  measurements: [],
  rule_evaluations: sourceNode('source-active', points, 'BULLISH').rule_evaluations,
  conformance,
  source_node: sourceNode('source-active', points, 'BULLISH'),
}

const oldPivots: Pivot[] = [
  { time: 1_577_836_800, bar_index: 0, price: 60, kind: 'LOW', state: 'CONFIRMED', prominence: 5 },
  { time: 1_609_459_200, bar_index: 1, price: 90, kind: 'HIGH', state: 'CONFIRMED', prominence: 5 },
]
const oldNode: MasterWaveNode = {
  ...activeNode,
  id: 'wave-old',
  pattern: 'DEVELOPING_IMPULSE_EXPECTING_2',
  degree: 'PRIMARY',
  label: 'Historical primary wave',
  start_event_id: 'event-old-start',
  end_event_id: 'event-old-end',
  pivot_event_ids: ['event-old-start', 'event-old-end'],
  resolutions: ['1W', '1D'],
  orthodox_start: oldPivots[0],
  orthodox_end: oldPivots[1],
  source_node: sourceNode('source-old', oldPivots, 'BULLISH'),
}

const scenario = (id: string, rank: number, bias: 'BULLISH' | 'BEARISH'): MasterScenario => ({
  id,
  rank,
  status: rank === 1 ? 'PREFERRED' : 'ALTERNATE',
  bias,
  current_position: bias === 'BULLISH'
    ? 'Intermediate (4) → expecting Minor 5'
    : 'Intermediate (B) → expecting Minor C',
  conformance: rank === 1 ? conformance : { ...conformance, guidelines_passed: 4, score: 0.84 },
  observation_root: {
    from: events[0].orthodox_time,
    to: points[4].time,
    open_left_boundary: true,
    context_sequence: ['wave-old', 'wave-active'],
    intervals: [
      {
        from: events[0].orthodox_time,
        to: events[1].orthodox_time,
        status: 'OBSERVED',
        node_id: 'wave-old',
        explanation: 'Counted historical structure.',
      },
    ],
  },
  active_path: ['wave-active'],
  invalidations: [{
    id: 'wave-4-extreme',
    kind: 'PRICE',
    price: bias === 'BULLISH' ? 121 : 141,
    description: 'The active interpretation fails beyond this price.',
  }],
  target_ladder: rank === 1 ? [{
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
    invalidation_ids: ['wave-4-extreme'],
  }] : [],
  audit: {
    global_thesis: bias === 'BULLISH'
      ? 'Cycle III → Primary [4] → Intermediate (C) → Minor 5'
      : 'Cycle III → Primary [4] → Intermediate (B)',
    cross_timeframe_evidence: timeframes.map((timeframe) => ({
      timeframe,
      position: 'Cycle III → Primary [4]',
      parent_node_id: 'wave-active',
      visible_children: timeframe === '1W' ? 0 : 3,
      endpoint_aligned: true,
      coverage: 'OBSERVED',
      status: 'CONSISTENT_MASTER_ASSIGNMENT',
    })),
    why_preferred: rank === 1 ? {
      alternative_id: 'scenario-bearish',
      first_divergence: 'Intermediate (C) versus Intermediate (B)',
      preferred_evidence: ['94% average structural coverage', '6 cross-structure guidelines passed'],
      different_targets: true,
      different_bias: true,
    } : undefined,
  },
  material_signature: id,
})

function makeView(timeframe: Timeframe, snapshotID = analysisID): TimeframeView {
  return {
    snapshot_id: snapshotID,
    timeframe,
    candles,
    visible_node_ids: timeframe === '1W' ? ['wave-old'] : ['wave-active'],
    ancestor_node_ids: ['wave-active'],
    future_logical_bars: Array.from({ length: 100 }, (_, index) => candles.at(-1)!.time + (index + 1) * 86_400),
    coverage: {
      from: candles[0].time,
      to: candles.at(-1)!.time,
      detail_from: startTime,
      status: 'OBSERVED',
      message: '',
    },
  }
}

function makeSnapshot(id = analysisID, parentSnapshotID?: string): AnalysisSnapshot {
  return {
    id,
    parent_snapshot_id: parentSnapshotID,
    theory_version: 'ewp-frost-prechter+waveratios-1.0.0',
    engine_version: '3.0.0',
    generated_at: 1_782_559_200,
    request: {
      symbol: 'AAPL',
      focus_timeframe: '1D',
      session: 'RTH',
      as_of: '2026-06-27T00:00:00Z',
      history_profile: 'MAX_DAILY_PLUS_2Y_MINUTE',
      max_scenarios: 5,
    },
    dataset_manifest: {
      coverage: [],
      provider_queries: [
        { resolution: 'DAILY_NATIVE', from: 0, to: 1, logical_query: true, page_requests: 1, rows: 80, cache_only: false, overlap_changed: false },
        { resolution: 'MINUTE_NATIVE', from: 0, to: 1, logical_query: true, page_requests: 4, rows: 200_000, cache_only: false, overlap_changed: false },
      ],
      daily_provenance: { compared: 80, differences: 0, max_ohlc_deviation: 0, samples: [] },
      minute_detail_from: startTime,
      minute_detail_to: candles.at(-1)!.time,
      native_daily_rows: 5_000,
      native_minute_rows: 200_000,
    },
    master_wave_graph: { events, nodes: [oldNode, activeNode] },
    scenarios: [scenario('scenario-bullish', 1, 'BULLISH'), scenario('scenario-bearish', 2, 'BEARISH')],
    view_manifest: timeframes.map((timeframe) => ({
      timeframe,
      candle_count: candles.length,
      from: candles[0].time,
      to: candles.at(-1)!.time,
    })),
    initial_view: makeView('1D', id),
  }
}

const snapshot = makeSnapshot()
const revision = makeSnapshot(revisionID, analysisID)

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v3/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    if (url.pathname.endsWith('/analysis-jobs') && request.method() === 'POST') {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'scan-job',
          status: 'QUEUED',
          progress: 0,
          message: 'Scan queued',
          request: snapshot.request,
          created_at: 1,
          updated_at: 1,
        }),
      })
      return
    }
    if (url.pathname.endsWith('/refinements') && request.method() === 'POST') {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'refine-job',
          status: 'QUEUED',
          progress: 0,
          message: 'Refinement queued',
          request: snapshot.request,
          created_at: 1,
          updated_at: 1,
        }),
      })
      return
    }
    if (url.pathname.includes('/analysis-jobs/')) {
      const refine = url.pathname.endsWith('/refine-job')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: refine ? 'refine-job' : 'scan-job',
          status: 'COMPLETED',
          progress: 100,
          message: 'Completed',
          snapshot_id: refine ? revisionID : analysisID,
          request: snapshot.request,
          created_at: 1,
          updated_at: 2,
        }),
      })
      return
    }
    const viewMatch = url.pathname.match(/\/analyses\/([a-f0-9]{32})\/views\/(.+)$/)
    if (viewMatch) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(makeView(viewMatch[2] as Timeframe, viewMatch[1])),
      })
      return
    }
    if (url.pathname.endsWith(`/${analysisID}`)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(snapshot) })
      return
    }
    if (url.pathname.endsWith(`/${revisionID}`)) {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(revision) })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items: [{
          id: analysisID,
          symbol: 'AAPL',
          focus_timeframe: '1D',
          session: 'RTH',
          as_of: 1_782_518_400,
          generated_at: snapshot.generated_at,
          theory_version: snapshot.theory_version,
          engine_version: snapshot.engine_version,
          request_key: 'request-key',
          data_fingerprint: 'data-fingerprint',
        }],
      }),
    })
  })
})

test('builds one master scenario and changes all views without rescanning', async ({ page }) => {
  let scanPosts = 0
  page.on('request', (request) => {
    if (request.method() === 'POST' && request.url().endsWith('/api/v3/analysis-jobs')) scanPosts++
  })
  await page.goto('/')
  await page.getByRole('button', { name: 'Scan full history' }).click()

  await expect(page.getByText('Preferred', { exact: true })).toBeVisible()
  await expect(page.getByText('Purple Box targets')).toBeVisible()
  await expect(page.getByText('$142 — $146')).toBeVisible()
  await expect(page).toHaveURL(new RegExp(`/analysis/${analysisID}$`))

  for (const timeframe of timeframes) {
    await page.getByLabel('Chart timeframe').selectOption(timeframe)
    await expect(page.locator('.chart-toolbar strong')).toContainText(timeframe)
  }
  expect(scanPosts).toBe(1)
  await expect(page.getByText('Cycle III → Primary [4] → Intermediate (C) → Minor 5').first()).toBeVisible()

  const alternate = page.locator('.scenario-card').filter({ hasText: 'Alternate 1' })
  await alternate.getByRole('button').first().click()
  await expect(page.getByRole('heading', { name: 'Cycle III → Primary [4] → Intermediate (B)' })).toBeVisible()
})

test('loads a share URL and creates an immutable detail revision', async ({ page }) => {
  await page.goto(`/analysis/${analysisID}`)
  await expect(page.getByRole('heading', { name: 'Cycle III → Primary [4] → Intermediate (C) → Minor 5' })).toBeVisible()
  await page.getByRole('button', { name: /Historical primary wave/ }).click()
  await expect(page.getByText('Detail not loaded')).toBeVisible()
  await page.getByRole('button', { name: 'Load this wave’s detail' }).click()
  await expect(page).toHaveURL(new RegExp(`/analysis/${revisionID}$`))
  await expect(page.getByText(`Revision of ${analysisID.slice(0, 8)}`)).toBeVisible()

  await page.getByRole('button', { name: 'History', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Scan history' })).toBeVisible()
})
