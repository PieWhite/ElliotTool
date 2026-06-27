import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useMarketData } from './useMarketData'
import type { AnalysisSnapshot } from '../types/api'

const snapshot: AnalysisSnapshot = {
  id: '0123456789abcdef0123456789abcdef',
  theory_version: 'theory',
  engine_version: 'engine',
  generated_at: 1,
  request: {
    symbol: 'MSFT',
    timeframe: '1h',
    session: 'RTH',
    lookback_bars: 500,
    max_scenarios: 5,
  },
  data_quality: {
    candle_count: 0,
    first_time: 0,
    last_time: 0,
    missing_intervals: 0,
    ambiguous_pivot_count: 0,
  },
  candles: [],
  scenarios: [],
  future_bars: [],
}

const Harness = defineComponent({
  setup() {
    return useMarketData()
  },
  template: `
    <button id="scan" @click="scan">scan</button>
    <span id="symbol">{{ symbol }}</span>
    <span id="snapshot">{{ snapshot?.id }}</span>
    <span id="error">{{ error }}</span>
  `,
})

afterEach(() => {
  vi.unstubAllGlobals()
  window.history.replaceState({}, '', '/')
})

describe('useMarketData', () => {
  it('posts the v2 contract, installs the immutable snapshot and refreshes history', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'POST') {
        expect(JSON.parse(String(init.body))).toMatchObject({
          symbol: 'AAPL',
          timeframe: '1D',
          session: 'RTH',
          max_scenarios: 5,
        })
        return new Response(JSON.stringify(snapshot), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      expect(url).toContain('/api/v2/analyses?limit=20')
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
    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(window.location.pathname).toBe(`/analysis/${snapshot.id}`)
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
