<template>
  <div class="app-shell">
    <header class="app-header">
      <a class="brand" href="/" aria-label="WaveSight home">
        <span class="brand-mark"><i></i><i></i><i></i></span>
        <span><strong>WaveSight</strong><small>Coherent Elliott Intelligence</small></span>
      </a>
      <div class="header-meta">
        <span class="engine-state"><i></i> Master engine v{{ snapshot?.engine_version ?? '3.0.0' }}</span>
        <button class="icon-button" type="button" @click="historyOpen = !historyOpen">History</button>
      </div>
    </header>

    <main>
      <form class="scan-bar master-scan-bar" @submit.prevent="scan">
        <label class="ticker-field">
          <span>Symbol</span>
          <input v-model="symbol" maxlength="15" autocomplete="off" spellcheck="false" aria-label="Ticker symbol">
        </label>
        <label>
          <span>Chart view</span>
          <select :value="timeframe" aria-label="Chart timeframe" @change="changeTimeframe">
            <option v-for="item in timeframes" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          <span>Session count</span>
          <select v-model="session">
            <option value="RTH">Regular hours</option>
            <option value="EXTENDED">Extended hours</option>
          </select>
        </label>
        <label class="asof-field">
          <span>As of</span>
          <input v-model="asOf" type="datetime-local" aria-label="Analysis date and time">
        </label>
        <button class="scan-button" type="submit" :disabled="loading">
          <span v-if="loading" class="spinner"></span>
          {{ loading ? 'Building master count…' : 'Scan full history' }}
        </button>
      </form>

      <div v-if="loading" class="scan-progress" aria-live="polite">
        <div><span>{{ progressStatus?.replaceAll('_', ' ') }}</span><strong>{{ progress }}%</strong></div>
        <div class="progress-track"><i :style="{ width: `${progress}%` }"></i></div>
        <p>{{ progressMessage }}</p>
      </div>

      <div v-if="error" class="status-banner error-banner" role="alert">
        <span><strong>Analysis unavailable</strong>{{ error }}</span>
        <button type="button" @click="error = null">Dismiss</button>
      </div>

      <div v-if="snapshot" class="status-banner quality-banner dataset-banner">
        <span>
          <strong>One coherent count</strong>
          {{ snapshot.dataset_manifest.native_daily_rows.toLocaleString() }} daily bars ·
          {{ snapshot.dataset_manifest.native_minute_rows.toLocaleString() }} minute bars ·
          timeframe changes are local views
        </span>
        <span>{{ providerSummary }} · {{ provenanceSummary }}</span>
      </div>

      <section v-if="snapshot" class="scenario-rail" aria-label="Ranked master scenarios">
        <article
          v-for="scenarioItem in scenarios"
          :key="scenarioItem.id"
          :class="['scenario-card', { active: scenarioItem.id === activeScenarioID }]"
        >
          <button type="button" class="scenario-main" @click="selectScenario(scenarioItem.id)">
            <span class="scenario-rank">{{ scenarioDisplayName(scenarioItem) }}</span>
            <span :class="['bias', scenarioItem.bias.toLowerCase()]">{{ scenarioItem.bias }}</span>
            <strong>{{ scenarioItem.audit.global_thesis }}</strong>
            <small>{{ scenarioItem.current_position }}</small>
            <span class="conformance-line">
              <i :style="{ width: `${Math.round(scenarioItem.conformance.structural_coverage * 100)}%` }"></i>
            </span>
            <span class="conformance-copy">
              {{ scenarioItem.conformance.hard_rules_passed }} hard rules ·
              {{ scenarioItem.conformance.guidelines_passed }} guidelines ·
              {{ scenarioItem.active_path.length }} active degrees
            </span>
          </button>
          <button
            v-if="scenarioItem.id !== activeScenarioID"
            type="button"
            :class="['compare-button', { selected: scenarioItem.id === compareScenarioID }]"
            @click="selectComparison(scenarioItem.id)"
          >
            {{ scenarioItem.id === compareScenarioID ? 'Comparing' : 'Compare' }}
          </button>
        </article>
      </section>

      <section class="workspace">
        <div class="chart-column">
          <div class="chart-toolbar">
            <div>
              <strong>{{ snapshot ? `${snapshot.request.symbol} · ${timeframe}` : 'Master market structure' }}</strong>
              <span v-if="view">{{ formatDate(view.coverage.from) }} — {{ formatDate(view.coverage.to) }}</span>
            </div>
            <div class="toolbar-actions">
              <span v-if="viewLoading" class="local-view-state">Loading local view…</span>
              <button type="button" :class="{ active: scale === 'ARITHMETIC' }" @click="scale = 'ARITHMETIC'">Arithmetic</button>
              <button type="button" :class="{ active: scale === 'LOG' }" @click="scale = 'LOG'">Log</button>
              <button type="button" :disabled="!snapshot" @click="chart?.exportPNG()">PNG</button>
              <button type="button" :disabled="!snapshot" @click="exportJSON">JSON</button>
              <button type="button" :disabled="!snapshot" @click="copyShareLink">{{ copied ? 'Copied' : 'Share' }}</button>
            </div>
          </div>
          <ChartWidget
            ref="chart"
            :snapshot="snapshot"
            :view="view"
            :scenario="activeScenario"
            :comparison="compareScenario"
            :scale="scale"
            :selected-node="selectedNodeID"
            @select-node="selectNode"
          />
          <div v-if="snapshot && view" class="chart-footer">
            <span>{{ view.candles.length.toLocaleString() }} rendered {{ timeframe }} bars</span>
            <span>{{ view.visible_node_ids.length }} visible structures · parents retained</span>
            <span>Snapshot {{ snapshot.id.slice(0, 8) }}</span>
            <span v-if="snapshot.parent_snapshot_id">Revision of {{ snapshot.parent_snapshot_id.slice(0, 8) }}</span>
          </div>
        </div>

        <aside class="analysis-panel">
          <template v-if="activeScenario && snapshot">
            <section class="panel-section current-position">
              <span class="eyebrow">Global thesis</span>
              <h2>{{ activeScenario.audit.global_thesis }}</h2>
              <p class="thesis-copy">{{ activeScenario.current_position }}</p>
              <div class="invalidations">
                <span v-for="item in activeScenario.invalidations" :key="item.id">
                  <i></i>{{ item.price ? formatPrice(item.price) : item.rule_id }} · {{ item.description }}
                </span>
              </div>
            </section>

            <section v-if="refinementRange" class="panel-section refinement-card">
              <span class="eyebrow">Detail not loaded</span>
              <strong>Open the selected historical parent wave</strong>
              <p>Daily structure is known here; minute subdivisions are deliberately not invented.</p>
              <button type="button" :disabled="loading" @click="runRefinement">Load this wave’s detail</button>
            </section>

            <section class="panel-section">
              <div class="section-heading">
                <span><span class="eyebrow">Conditional ladder</span><strong>Purple Box targets</strong></span>
                <small>{{ activeTargets.length }} active</small>
              </div>
              <div v-if="activeTargets.length" class="target-list">
                <article v-for="target in activeTargets" :key="target.id" class="target-card">
                  <header>
                    <span class="target-wave">{{ target.wave_label }}</span>
                    <span :class="['confluence', target.confluence.toLowerCase()]">{{ target.confluence.replace('_', ' ') }}</span>
                  </header>
                  <strong>
                    {{ formatPrice(target.min_price) }}
                    <template v-if="target.max_price !== target.min_price"> — {{ formatPrice(target.max_price) }}</template>
                  </strong>
                  <p>{{ target.condition }}</p>
                  <div class="target-levels">
                    <span v-for="level in target.levels" :key="`${target.id}-${level.family}-${level.price}`">
                      {{ formatPrice(level.price) }} <small>{{ level.relation }}</small>
                    </span>
                  </div>
                  <footer>
                    <span>{{ target.status }}</span>
                    <span v-if="target.time_window">
                      {{ formatDate(target.time_window.start_time ?? 0) }} — {{ formatDate(target.time_window.end_time ?? 0) }}
                    </span>
                    <span v-else>Open-ended · no time confluence</span>
                  </footer>
                </article>
              </div>
              <p v-else class="empty-copy">No active target zone is justified for this structural state.</p>
            </section>

            <details class="panel-section audit cross-audit" open>
              <summary>
                <span><span class="eyebrow">Cross-timeframe evidence</span><strong>One story, seven views</strong></span>
                <span>{{ activeScenario.audit.cross_timeframe_evidence.length }}</span>
              </summary>
              <div class="timeframe-matrix">
                <article v-for="row in activeScenario.audit.cross_timeframe_evidence" :key="row.timeframe">
                  <strong>{{ row.timeframe }}</strong>
                  <span :class="row.coverage.toLowerCase()">{{ row.coverage.replace('_', ' ') }}</span>
                  <small>{{ row.endpoint_aligned ? 'Endpoints aligned' : 'Parent projection' }} · {{ row.visible_children }} children</small>
                </article>
              </div>
            </details>

            <details v-if="activeScenario.audit.why_preferred" class="panel-section audit" open>
              <summary>
                <span><span class="eyebrow">Why preferred</span><strong>First material divergence</strong></span>
              </summary>
              <div class="comparison-audit">
                <p>{{ activeScenario.audit.why_preferred.first_divergence }}</p>
                <span v-for="evidence in activeScenario.audit.why_preferred.preferred_evidence" :key="evidence">{{ evidence }}</span>
                <small>
                  {{ activeScenario.audit.why_preferred.different_bias ? 'Different bias' : 'Same bias' }} ·
                  {{ activeScenario.audit.why_preferred.different_targets ? 'Different target ladder' : 'Same targets' }}
                </small>
              </div>
            </details>

            <details class="panel-section audit">
              <summary>
                <span><span class="eyebrow">Detailed rule audit</span><strong>{{ selectedNode?.label ?? 'Selected wave' }}</strong></span>
                <span>{{ selectedRules.length }}</span>
              </summary>
              <div class="audit-filters">
                <button
                  v-for="filter in auditFilters"
                  :key="filter"
                  type="button"
                  :class="{ active: auditFilter === filter }"
                  @click="auditFilter = filter"
                >{{ filter.replace('_', ' ') }}</button>
              </div>
              <div class="audit-list">
                <article v-for="rule in selectedRules" :key="rule.rule_id">
                  <span :class="['rule-status', rule.status.toLowerCase()]">{{ rule.status.replace('_', ' ') }}</span>
                  <div>
                    <strong>{{ rule.rule_id }}</strong>
                    <small>{{ selectedNode?.degree.replaceAll('_', ' ') }} · {{ rule.class }} · {{ rule.source }}</small>
                    <p>{{ rule.summary }} <em v-if="rule.expected">{{ rule.expected }}</em></p>
                  </div>
                </article>
              </div>
            </details>

            <details class="panel-section wave-tree" open>
              <summary>
                <span><span class="eyebrow">Master wave tree</span><strong>Chart-wide context</strong></span>
              </summary>
              <ul>
                <WaveTreeNode
                  v-for="node in treeRoots"
                  :key="node.id"
                  :node="node"
                  :nodes="snapshot.master_wave_graph.nodes"
                  :selected-node="selectedNodeID"
                  @select="selectNode"
                />
              </ul>
            </details>
          </template>
          <div v-else class="panel-empty">
            <span class="brand-mark large"><i></i><i></i><i></i></span>
            <h2>One market, one wave hierarchy</h2>
            <p>Start a scan to connect the weekly structure to its minute subdivisions without creating separate timeframe stories.</p>
          </div>
        </aside>
      </section>
    </main>

    <aside :class="['history-drawer', { open: historyOpen }]">
      <header>
        <div><span class="eyebrow">Immutable snapshots</span><h2>Scan history</h2></div>
        <button type="button" @click="historyOpen = false">Close</button>
      </header>
      <p v-if="historyLoading" class="empty-copy">Loading history…</p>
      <button
        v-for="item in history"
        :key="item.id"
        type="button"
        class="history-item"
        @click="loadSnapshot(item.id); historyOpen = false"
      >
        <span><strong>{{ item.symbol }}</strong>{{ item.focus_timeframe }} · {{ item.session }}</span>
        <small>{{ formatDateTime(item.generated_at) }}</small>
        <code>{{ item.id.slice(0, 12) }}<template v-if="item.parent_snapshot_id"> · revision</template></code>
      </button>
      <p v-if="!historyLoading && history.length === 0" class="empty-copy">No saved analyses yet.</p>
    </aside>
    <div v-if="historyOpen" class="drawer-backdrop" @click="historyOpen = false"></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import ChartWidget from './components/ChartWidget.vue'
import WaveTreeNode from './components/WaveTreeNode.vue'
import { useMarketData } from './composables/useMarketData'
import { scenarioDisplayName, visibleTargetZones } from './domain/presentation'
import type { EvaluationStatus, MasterWaveNode, RuleEvaluation, Timeframe } from './types/api'

interface ChartExport {
  exportPNG: () => void
}

const timeframes: Timeframe[] = ['1m', '5m', '15m', '1h', '4h', '1D', '1W']
const auditFilters = ['ALL', 'HARD_RULE', 'GUIDELINE', 'NOT_OBSERVABLE'] as const
type AuditFilter = typeof auditFilters[number]
const chart = ref<ChartExport | null>(null)
const scale = ref<'ARITHMETIC' | 'LOG'>('ARITHMETIC')
const historyOpen = ref(false)
const copied = ref(false)
const auditFilter = ref<AuditFilter>('ALL')

const {
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
  selectTimeframe,
  refineHistory,
  selectScenario,
  selectComparison,
  selectNode,
  exportJSON,
  initialize,
} = useMarketData()

const nodeByID = computed(() => new Map(
  snapshot.value?.master_wave_graph.nodes.map((node) => [node.id, node]) ?? [],
))
const selectedNode = computed(() => nodeByID.value.get(selectedNodeID.value))
const treeRoots = computed<MasterWaveNode[]>(() => {
  const ids = activeScenario.value?.observation_root.context_sequence ?? []
  const unique = [...new Set(ids)]
  return unique.map((id) => nodeByID.value.get(id)).filter((node): node is MasterWaveNode => node !== undefined)
})
const selectedRules = computed<RuleEvaluation[]>(() => {
  const rules = selectedNode.value?.rule_evaluations ?? []
  return rules.filter((rule) => {
    if (auditFilter.value === 'ALL') return true
    if (auditFilter.value === 'NOT_OBSERVABLE') return rule.status === 'NOT_OBSERVABLE'
    return rule.class === auditFilter.value
  }).sort((left, right) => statusOrder(left.status) - statusOrder(right.status))
})
const activeTargets = computed(() => visibleTargetZones(activeScenario.value))
const providerSummary = computed(() => {
  const queries = snapshot.value?.dataset_manifest.provider_queries ?? []
  const pages = queries.reduce((sum, query) => sum + query.page_requests, 0)
  return pages === 0 ? 'Fully served from local cache' : `${queries.filter((query) => !query.cache_only).length} datasets · ${pages} provider pages`
})
const provenanceSummary = computed(() => {
  const audit = snapshot.value?.dataset_manifest.daily_provenance
  if (!audit || audit.compared === 0) return 'daily provenance not comparable'
  return audit.differences === 0
    ? `${audit.compared} recent days aligned`
    : `${audit.differences}/${audit.compared} native/derived deviations · max ${formatPrice(audit.max_ohlc_deviation)}`
})
const refinementRange = computed(() => {
  const current = selectedNode.value
  const detailFrom = snapshot.value?.dataset_manifest.minute_detail_from ?? 0
  const events = snapshot.value?.master_wave_graph.events ?? []
  if (!current || !detailFrom) return null
  const start = events.find((event) => event.id === current.start_event_id)?.orthodox_time
  const end = events.find((event) => event.id === current.end_event_id)?.orthodox_time
  if (!start || !end || end >= detailFrom) return null
  return { from: start, to: Math.min(end, detailFrom - 1), nodeID: current.id }
})

function statusOrder(status: EvaluationStatus): number {
  return { FAIL: 0, PASS: 1, NOT_OBSERVABLE: 2, NOT_APPLICABLE: 3 }[status]
}

function changeTimeframe(event: Event): void {
  const target = event.target
  if (target instanceof HTMLSelectElement) void selectTimeframe(target.value as Timeframe)
}

function runRefinement(): void {
  const range = refinementRange.value
  if (range) void refineHistory(range.from, range.to, range.nodeID)
}

function formatPrice(value: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: value < 10 ? 2 : 0,
    maximumFractionDigits: value < 10 ? 4 : 2,
  }).format(value)
}

function formatDate(value: number): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' }).format(new Date(value * 1000))
}

function formatDateTime(value: number): string {
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value * 1000))
}

async function copyShareLink(): Promise<void> {
  if (!snapshot.value) return
  await navigator.clipboard.writeText(window.location.href)
  copied.value = true
  window.setTimeout(() => { copied.value = false }, 1600)
}

onMounted(initialize)
</script>
