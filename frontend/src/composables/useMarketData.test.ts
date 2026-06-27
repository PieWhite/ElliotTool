import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useMarketData } from './useMarketData'
import type { AnalysisSnapshot, TimeframeView } from '../types/api'

const emptyView = (timeframe: TimeframeView['timeframe']): TimeframeView => ({
  snapshot_id: '0123456789abcdef0123456789abcdef',
  timeframe,
  candles: [],
  visible_node_ids: [],
  ancestor_node_ids: [],
  future_logical_bars: [],
  coverage: { from: 0, to: 0, detail_from: 0, status: 'OBSERVED', message: '' },
})

const snapshot: AnalysisSnapshot = {
  id: '0123456789abcdef0123456789abcdef',
  theory_version: 'theory',
  engine_version: '3.0.0',
  generated_at: 1,
  request: {
    symbol: 'MSFT',
    focus_timeframe: '1D',
    session: 'RTH',
    as_of: '2026-06-27T00:00:00Z',
    history_profile: 'MAX_DAILY_PLUS_2Y_MINUTE',
    max_scenarios: 5,
  },
  dataset_manifest: {
    coverage: [],
    provider_queries: [],
    daily_provenance: { compared: 0, differences: 0, max_ohlc_deviation: 0, samples: [] },
    minute_detail_from: 0,
    minute_detail_to: 0,
    native_daily_rows: 0,
    native_minute_rows: 0,
  },
  master_wave_graph: { events: [], nodes: [] },
  scenarios: [],
  view_manifest: [],
  initial_view: emptyView('1D'),
}

const Harness = defineComponent({
  setup() {
    return useMarketData()
  },
  template: `
    <button id="scan" @click="scan">scan</button>
    <button id="hour" @click="selectTimeframe('1h')">hour</button>
    <span id="symbol">{{ symbol }}</span>
    <span id="snapshot">{{ snapshot?.id }}</span>
    <span id="timeframe">{{ view?.timeframe }}</span>
    <span id="error">{{ error }}</span>
  `,
})

afterEach(() => {
  vi.unstubAllGlobals()
  window.history.replaceState({}, '', '/')
})

describe('useMarketData', () => {
  it('creates a v3 master scan and caches local timeframe views', async () => {
    const hourView = emptyView('1h')
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        expect(url).toContain('/api/v3/analysis-jobs')
        expect(JSON.parse(String(init.body))).toMatchObject({
          symbol: 'AAPL',
          focus_timeframe: '1D',
          session: 'RTH',
          history_profile: 'MAX_DAILY_PLUS_2Y_MINUTE',
          max_scenarios: 5,
        })
        return new Response(JSON.stringify(snapshot), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      if (url.includes('/views/1h')) {
        return new Response(JSON.stringify(hourView), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      expect(url).toContain('/api/v3/analyses?limit=20')
      return new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    })
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(Harness)
    await wrapper.get('#scan').trigger('click')
    await flushPromises()

    expect(wrapper.get('#symbol').text()).toBe('MSFT')
    expect(wrapper.get('#snapshot').text()).toBe(snapshot.id)
    expect(window.location.pathname).toBe(`/analysis/${snapshot.id}`)

    await wrapper.get('#hour').trigger('click')
    await flushPromises()
    await wrapper.get('#hour').trigger('click')
    await flushPromises()
    expect(wrapper.get('#timeframe').text()).toBe('1h')
    expect(fetchMock.mock.calls.filter(([url]) => String(url).includes('/views/1h'))).toHaveLength(1)
  })

  it('surfaces typed problem details without retaining stale data', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({
      type: 'https://wavesight.app/problems/provider',
      title: 'Market data unavailable',
      status: 502,
      detail: 'Provider entitlement denied.',
      request_id: 'request-1',
    }), {
      status: 502,
      headers: { 'Content-Type': 'application/problem+json' },
    })))
    const wrapper = mount(Harness)
    await wrapper.get('#scan').trigger('click')
    await flushPromises()

    expect(wrapper.get('#error').text()).toContain('Provider entitlement denied')
    expect(wrapper.get('#snapshot').text()).toBe('')
  })
})
