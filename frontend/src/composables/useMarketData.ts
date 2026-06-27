import { computed, onBeforeUnmount, ref, shallowRef } from 'vue'
import type {
  AnalysisJob,
  AnalysisRequest,
  AnalysisSnapshot,
  MasterScenario,
  Session,
  SnapshotHistory,
  SnapshotMetadata,
  Timeframe,
  TimeframeView,
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

function wait(milliseconds: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(resolve, milliseconds)
    signal.addEventListener('abort', () => {
      window.clearTimeout(timer)
      reject(new DOMException('Aborted', 'AbortError'))
    }, { once: true })
  })
}

export function useMarketData() {
  const symbol = ref('AAPL')
  const timeframe = ref<Timeframe>('1D')
  const session = ref<Session>('RTH')
  const asOf = ref('')
  const snapshot = shallowRef<AnalysisSnapshot | null>(null)
  const view = shallowRef<TimeframeView | null>(null)
  const history = shallowRef<SnapshotMetadata[]>([])
  const activeScenarioID = ref('')
  const compareScenarioID = ref('')
  const selectedNodeID = ref('')
  const loading = ref(false)
  const viewLoading = ref(false)
  const historyLoading = ref(false)
  const progress = ref(0)
  const progressStatus = ref<AnalysisJob['status'] | null>(null)
  const progressMessage = ref('')
  const error = ref<string | null>(null)
  const viewCache = new Map<Timeframe, TimeframeView>()
  let activeRequest: AbortController | null = null

  const scenarios = computed(() => snapshot.value?.scenarios ?? [])
  const activeScenario = computed<MasterScenario | null>(() => {
    const items = scenarios.value
    return items.find((scenario) => scenario.id === activeScenarioID.value) ?? items[0] ?? null
  })
  const compareScenario = computed<MasterScenario | null>(() => {
    if (!compareScenarioID.value) return null
    return scenarios.value.find((scenario) => scenario.id === compareScenarioID.value) ?? null
  })

  function installSnapshot(value: AnalysisSnapshot, updateURL = true): void {
    snapshot.value = value
    symbol.value = value.request.symbol
    session.value = value.request.session
    timeframe.value = value.initial_view.timeframe
    asOf.value = value.request.as_of?.slice(0, 16) ?? ''
    viewCache.clear()
    viewCache.set(value.initial_view.timeframe, value.initial_view)
    view.value = value.initial_view
    activeScenarioID.value = value.scenarios[0]?.id ?? ''
    compareScenarioID.value = ''
    selectedNodeID.value = value.scenarios[0]?.active_path.at(-1) ?? ''
    if (updateURL) {
      window.history.replaceState({}, '', `/analysis/${value.id}`)
    }
  }

  async function pollJob(initial: AnalysisJob, signal: AbortSignal): Promise<AnalysisJob> {
    let job = initial
    while (job.status !== 'COMPLETED' && job.status !== 'FAILED') {
      progress.value = job.progress
      progressStatus.value = job.status
      progressMessage.value = job.message
      await wait(350, signal)
      const response = await fetch(
        `${API_BASE_URL}/api/v3/analysis-jobs/${encodeURIComponent(job.id)}`,
        { signal },
      )
      if (!response.ok) throw await responseError(response)
      job = await response.json() as AnalysisJob
    }
    progress.value = job.progress
    progressStatus.value = job.status
    progressMessage.value = job.message
    if (job.status === 'FAILED') throw new Error(job.error || 'The master analysis failed.')
    return job
  }

  async function scan(): Promise<void> {
    activeRequest?.abort()
    activeRequest = new AbortController()
    loading.value = true
    error.value = null
    progress.value = 0
    progressStatus.value = 'QUEUED'
    progressMessage.value = 'Starting coherent master scan'
    const request: AnalysisRequest = {
      symbol: symbol.value.trim().toUpperCase(),
      focus_timeframe: timeframe.value,
      session: session.value,
      history_profile: 'MAX_DAILY_PLUS_2Y_MINUTE',
      max_scenarios: 5,
    }
    if (asOf.value) request.as_of = new Date(asOf.value).toISOString()
    try {
      const response = await fetch(`${API_BASE_URL}/api/v3/analysis-jobs`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
        signal: activeRequest.signal,
      })
      if (!response.ok) throw await responseError(response)
      if (response.status === 200) {
        installSnapshot(await response.json() as AnalysisSnapshot)
        if (timeframe.value !== request.focus_timeframe) {
          await selectTimeframe(request.focus_timeframe)
        }
      } else {
        const job = await pollJob(await response.json() as AnalysisJob, activeRequest.signal)
        if (!job.snapshot_id) throw new Error('The completed job did not return a snapshot.')
        await loadSnapshot(job.snapshot_id, true, activeRequest.signal)
      }
      await loadHistory()
    } catch (reason: unknown) {
      if (reason instanceof DOMException && reason.name === 'AbortError') return
      error.value = reason instanceof Error ? reason.message : 'The analysis could not be completed.'
    } finally {
      loading.value = false
    }
  }

  async function loadSnapshot(
    id: string,
    updateURL = true,
    signal?: AbortSignal,
  ): Promise<void> {
    if (!signal) loading.value = true
    error.value = null
    try {
      const response = await fetch(
        `${API_BASE_URL}/api/v3/analyses/${encodeURIComponent(id)}`,
        { signal },
      )
      if (!response.ok) throw await responseError(response)
      installSnapshot(await response.json() as AnalysisSnapshot, updateURL)
    } catch (reason: unknown) {
      if (reason instanceof DOMException && reason.name === 'AbortError') return
      error.value = reason instanceof Error ? reason.message : 'The saved analysis could not be loaded.'
    } finally {
      if (!signal) loading.value = false
    }
  }

  async function selectTimeframe(next: Timeframe): Promise<void> {
    timeframe.value = next
    const current = snapshot.value
    if (!current) return
    const cached = viewCache.get(next)
    if (cached) {
      view.value = cached
      return
    }
    viewLoading.value = true
    error.value = null
    try {
      const response = await fetch(
        `${API_BASE_URL}/api/v3/analyses/${encodeURIComponent(current.id)}/views/${next}`,
      )
      if (!response.ok) throw await responseError(response)
      const loaded = await response.json() as TimeframeView
      if (snapshot.value?.id !== current.id) return
      viewCache.set(next, loaded)
      if (timeframe.value === next) view.value = loaded
    } catch (reason: unknown) {
      error.value = reason instanceof Error ? reason.message : 'The local chart view is unavailable.'
    } finally {
      viewLoading.value = false
    }
  }

  async function refineHistory(from: number, to: number, parentNodeID?: string): Promise<void> {
    if (!snapshot.value) return
    activeRequest?.abort()
    activeRequest = new AbortController()
    loading.value = true
    error.value = null
    try {
      const response = await fetch(
        `${API_BASE_URL}/api/v3/analyses/${encodeURIComponent(snapshot.value.id)}/refinements`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            from: new Date(from * 1000).toISOString(),
            to: new Date(to * 1000).toISOString(),
            parent_node_id: parentNodeID,
          }),
          signal: activeRequest.signal,
        },
      )
      if (!response.ok) throw await responseError(response)
      const job = await pollJob(await response.json() as AnalysisJob, activeRequest.signal)
      if (!job.snapshot_id) throw new Error('The refinement did not return a revision.')
      await loadSnapshot(job.snapshot_id, true, activeRequest.signal)
      await loadHistory()
    } catch (reason: unknown) {
      if (reason instanceof DOMException && reason.name === 'AbortError') return
      error.value = reason instanceof Error ? reason.message : 'Historical detail could not be refined.'
    } finally {
      loading.value = false
    }
  }

  async function loadHistory(): Promise<void> {
    historyLoading.value = true
    try {
      const response = await fetch(`${API_BASE_URL}/api/v3/analyses?limit=20`)
      if (!response.ok) throw await responseError(response)
      history.value = (await response.json() as SnapshotHistory).items
    } catch (reason: unknown) {
      if (!snapshot.value) {
        error.value = reason instanceof Error ? reason.message : 'Analysis history is unavailable.'
      }
    } finally {
      historyLoading.value = false
    }
  }

  function selectScenario(id: string): void {
    activeScenarioID.value = id
    compareScenarioID.value = compareScenarioID.value === id ? '' : compareScenarioID.value
    const scenario = snapshot.value?.scenarios.find((item) => item.id === id)
    selectedNodeID.value = scenario?.active_path.at(-1) ?? ''
  }

  function selectComparison(id: string): void {
    compareScenarioID.value = compareScenarioID.value === id ? '' : id
  }

  function selectNode(id: string): void {
    selectedNodeID.value = id
  }

  function exportJSON(): void {
    if (!snapshot.value) return
    const blob = new Blob([JSON.stringify(snapshot.value, null, 2)], { type: 'application/json' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `wavesight-${snapshot.value.request.symbol}-${snapshot.value.id}.json`
    link.click()
    URL.revokeObjectURL(link.href)
  }

  async function initialize(): Promise<void> {
    const id = snapshotIDFromLocation()
    await Promise.all([id ? loadSnapshot(id, false) : Promise.resolve(), loadHistory()])
  }

  onBeforeUnmount(() => activeRequest?.abort())

  return {
    symbol,
    timeframe,
    session,
    asOf,
    snapshot,
    view,
    history,
    scenarios,
    activeScenario,
    compareScenario,
    activeScenarioID,
    compareScenarioID,
    selectedNodeID,
    loading,
    viewLoading,
    historyLoading,
    progress,
    progressStatus,
    progressMessage,
    error,
    scan,
    loadSnapshot,
    loadHistory,
    selectTimeframe,
    refineHistory,
    selectScenario,
    selectComparison,
    selectNode,
    exportJSON,
    initialize,
  }
}
