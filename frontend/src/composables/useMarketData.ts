import { computed, onBeforeUnmount, ref, shallowRef } from 'vue'
import type {
  AnalysisRequest,
  AnalysisSnapshot,
  Scenario,
  Session,
  SnapshotHistory,
  SnapshotMetadata,
  Timeframe,
} from '../types/api'
import { isProblemResponse } from '../types/api'

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ?? ''

async function responseError(response: Response): Promise<Error> {
  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('json')) {
    const body: unknown = await response.json()
    if (isProblemResponse(body)) {
      return new Error(`${body.title}: ${body.detail} (request ${body.request_id})`)
    }
  }
  const body = await response.text()
  return new Error(body || `WaveSight request failed with status ${response.status}`)
}

function snapshotIDFromLocation(): string | null {
  const match = window.location.pathname.match(/\/analysis\/([a-f0-9]{32})$/i)
  return match?.[1] ?? new URLSearchParams(window.location.search).get('analysis')
}

export function useMarketData() {
  const symbol = ref('AAPL')
  const timeframe = ref<Timeframe>('1D')
  const session = ref<Session>('RTH')
  const lookbackBars = ref(2000)
  const asOf = ref('')

  const snapshot = shallowRef<AnalysisSnapshot | null>(null)
  const history = shallowRef<SnapshotMetadata[]>([])
  const activeScenarioID = ref('')
  const compareScenarioID = ref('')
  const loading = ref(false)
  const historyLoading = ref(false)
  const error = ref<string | null>(null)
  let activeRequest: AbortController | null = null

  const scenarios = computed(() => snapshot.value?.scenarios ?? [])
  const activeScenario = computed<Scenario | null>(() => {
    const items = scenarios.value
    return items.find((scenario) => scenario.id === activeScenarioID.value) ?? items[0] ?? null
  })
  const compareScenario = computed<Scenario | null>(() => {
    if (!compareScenarioID.value) return null
    return scenarios.value.find((scenario) => scenario.id === compareScenarioID.value) ?? null
  })

  function installSnapshot(value: AnalysisSnapshot, updateURL = true) {
    snapshot.value = value
    symbol.value = value.request.symbol
    timeframe.value = value.request.timeframe
    session.value = value.request.session
    lookbackBars.value = value.request.lookback_bars ?? value.candles.length
    asOf.value = value.request.as_of?.slice(0, 16) ?? ''
    activeScenarioID.value = value.scenarios[0]?.id ?? ''
    compareScenarioID.value = ''
    if (updateURL) {
      window.history.replaceState({}, '', `/analysis/${value.id}`)
    }
  }

  async function scan() {
    activeRequest?.abort()
    activeRequest = new AbortController()
    loading.value = true
    error.value = null
    const request: AnalysisRequest = {
      symbol: symbol.value.trim().toUpperCase(),
      timeframe: timeframe.value,
      session: session.value,
      lookback_bars: lookbackBars.value,
      max_scenarios: 5,
    }
    if (asOf.value) {
      request.as_of = new Date(asOf.value).toISOString()
    }
    try {
      const response = await fetch(`${API_BASE_URL}/api/v2/analyses`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
        signal: activeRequest.signal,
      })
      if (!response.ok) throw await responseError(response)
      installSnapshot(await response.json() as AnalysisSnapshot)
      await loadHistory()
    } catch (reason: unknown) {
      if (reason instanceof DOMException && reason.name === 'AbortError') return
      error.value = reason instanceof Error ? reason.message : 'The analysis could not be completed.'
    } finally {
      loading.value = false
    }
  }

  async function loadSnapshot(id: string, updateURL = true) {
    loading.value = true
    error.value = null
    try {
      const response = await fetch(`${API_BASE_URL}/api/v2/analyses/${encodeURIComponent(id)}`)
      if (!response.ok) throw await responseError(response)
      installSnapshot(await response.json() as AnalysisSnapshot, updateURL)
    } catch (reason: unknown) {
      error.value = reason instanceof Error ? reason.message : 'The saved analysis could not be loaded.'
    } finally {
      loading.value = false
    }
  }

  async function loadHistory() {
    historyLoading.value = true
    try {
      const response = await fetch(`${API_BASE_URL}/api/v2/analyses?limit=20`)
      if (!response.ok) throw await responseError(response)
      const value = await response.json() as SnapshotHistory
      history.value = value.items
    } catch (reason: unknown) {
      if (!snapshot.value) {
        error.value = reason instanceof Error ? reason.message : 'Analysis history is unavailable.'
      }
    } finally {
      historyLoading.value = false
    }
  }

  function selectScenario(id: string) {
    activeScenarioID.value = id
    if (compareScenarioID.value === id) compareScenarioID.value = ''
  }

  function selectComparison(id: string) {
    compareScenarioID.value = compareScenarioID.value === id ? '' : id
  }

  function exportJSON() {
    if (!snapshot.value) return
    const blob = new Blob([JSON.stringify(snapshot.value, null, 2)], { type: 'application/json' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `wavesight-${snapshot.value.request.symbol}-${snapshot.value.id}.json`
    link.click()
    URL.revokeObjectURL(link.href)
  }

  async function initialize() {
    const id = snapshotIDFromLocation()
    await Promise.all([id ? loadSnapshot(id, false) : Promise.resolve(), loadHistory()])
  }

  onBeforeUnmount(() => activeRequest?.abort())

  return {
    symbol,
    timeframe,
    session,
    lookbackBars,
    asOf,
    snapshot,
    history,
    scenarios,
    activeScenario,
    compareScenario,
    activeScenarioID,
    compareScenarioID,
    loading,
    historyLoading,
    error,
    scan,
    loadSnapshot,
    loadHistory,
    selectScenario,
    selectComparison,
    exportJSON,
    initialize,
  }
}
