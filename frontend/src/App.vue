<template>
  <div class="app-shell">
    <header class="app-header">
      <a class="brand" href="/" aria-label="WaveSight home">
        <span class="brand-mark"><i></i><i></i><i></i></span>
        <span><strong>WaveSight</strong><small>Elliott Wave Intelligence</small></span>
      </a>
      <div class="header-meta">
        <span class="engine-state"><i></i> Engine v{{ snapshot?.engine_version ?? '2.0.0' }}</span>
        <button class="icon-button" type="button" title="Analysis history" @click="historyOpen = !historyOpen">
          History
        </button>
      </div>
    </header>

    <main>
      <form class="scan-bar" @submit.prevent="scan">
        <label class="ticker-field">
          <span>Symbol</span>
          <input v-model="symbol" maxlength="15" autocomplete="off" spellcheck="false" aria-label="Ticker symbol">
        </label>
        <label>
          <span>Timeframe</span>
          <select v-model="timeframe">
            <option v-for="item in timeframes" :key="item" :value="item">{{ item }}</option>
          </select>
        </label>
        <label>
          <span>Lookback</span>
          <select v-model.number="lookbackBars">
            <option :value="500">500 bars</option>
            <option :value="2000">2,000 bars</option>
            <option :value="5000">5,000 bars</option>
            <option :value="10000">10,000 bars</option>
            <option :value="50000">50,000 bars</option>
          </select>
        </label>
        <label>
          <span>Session</span>
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
          {{ loading ? 'Scanning structure…' : 'Scan market' }}
        </button>
      </form>

      <div v-if="error" class="status-banner error-banner" role="alert">
        <span><strong>Scan unavailable</strong>{{ error }}</span>
        <button type="button" @click="error = null">Dismiss</button>
      </div>

      <div v-if="snapshot?.data_quality.warnings?.length" class="status-banner quality-banner">
        <span>
          <strong>Data-quality note</strong>
          {{ snapshot.data_quality.warnings.join(' ') }}
        </span>
        <span>{{ snapshot.data_quality.ambiguous_pivot_count }} ambiguous pivots</span>
      </div>

      <section v-if="snapshot" class="scenario-rail" aria-label="Ranked scenarios">
        <article
          v-for="scenarioItem in scenarios"
          :key="scenarioItem.id"
          :class="['scenario-card', { active: scenarioItem.id === activeScenarioID }]"
        >
          <button type="button" class="scenario-main" @click="selectScenario(scenarioItem.id)">
                <span class="scenario-rank">{{ scenarioDisplayName(scenarioItem) }}</span>
            <span :class="['bias', scenarioItem.bias.toLowerCase()]">{{ scenarioItem.bias }}</span>
            <strong>{{ scenarioItem.root.label || scenarioItem.root.pattern.replaceAll('_', ' ') }}</strong>
            <small>{{ scenarioItem.current_position }}</small>
            <span class="conformance-line">
              <i :style="{ width: `${Math.round(scenarioItem.conformance.score * 100)}%` }"></i>
            </span>
            <span class="conformance-copy">
              {{ scenarioItem.conformance.hard_rules_passed }} hard rules ·
              {{ scenarioItem.conformance.guidelines_passed }} guidelines ·
              {{ scenarioItem.conformance.ratio_confluences }} ratio supports
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
              <strong>{{ snapshot ? `${snapshot.request.symbol} · ${snapshot.request.timeframe}` : 'Market structure' }}</strong>
              <span v-if="snapshot">{{ formatDate(snapshot.data_quality.first_time) }} — {{ formatDate(snapshot.data_quality.last_time) }}</span>
            </div>
            <div class="toolbar-actions">
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
            :scenario="activeScenario"
            :comparison="compareScenario"
            :scale="scale"
            :visible-degrees="visibleDegrees"
          />
          <div v-if="snapshot" class="chart-footer">
            <span>{{ snapshot.data_quality.candle_count.toLocaleString() }} split-adjusted bars</span>
            <span>Theory {{ snapshot.theory_version }}</span>
            <span>Snapshot {{ snapshot.id.slice(0, 8) }}</span>
          </div>
        </div>

        <aside class="analysis-panel">
          <template v-if="activeScenario">
            <section class="panel-section current-position">
              <span class="eyebrow">Current wave path</span>
              <h2>{{ activeScenario.current_position }}</h2>
              <p v-if="activeScenario.status === 'INDETERMINATE'" class="indeterminate">
                WaveSight will not manufacture a count when observable structure is insufficient.
              </p>
              <div v-else class="invalidations">
                <span v-for="item in activeScenario.invalidations" :key="item.id">
                  <i></i>{{ item.price ? formatPrice(item.price) : item.rule_id }} · {{ item.description }}
                </span>
              </div>
            </section>

            <section class="panel-section">
              <div class="section-heading">
                <span><span class="eyebrow">Conditional ladder</span><strong>Purple Box targets</strong></span>
                <small>{{ activeTargets.length }} active levels</small>
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

            <details class="panel-section audit">
              <summary>
                <span><span class="eyebrow">Transparent evidence</span><strong>Rule audit</strong></span>
                <span>{{ ruleAudit.length }}</span>
              </summary>
              <div class="audit-list">
                <article v-for="rule in ruleAudit" :key="`${rule.node}-${rule.result.rule_id}`">
                  <span :class="['rule-status', rule.result.status.toLowerCase()]">{{ rule.result.status.replace('_', ' ') }}</span>
                  <div>
                    <strong>{{ rule.result.rule_id }}</strong>
                    <small>{{ rule.node }} · {{ rule.result.class }} · {{ rule.result.source }}</small>
                    <p>{{ rule.result.summary }} <em v-if="rule.result.expected">{{ rule.result.expected }}</em></p>
                  </div>
                </article>
              </div>
            </details>

            <details class="panel-section wave-tree" open>
              <summary>
                <span><span class="eyebrow">Relative nesting</span><strong>Wave tree</strong></span>
              </summary>
              <div class="degree-filters">
                <label v-for="degree in availableDegrees" :key="degree">
                  <input
                    type="checkbox"
                    :checked="visibleDegrees.includes(degree)"
                    @change="toggleDegree(degree)"
                  >
                  {{ degree.replaceAll('_', ' ') }}
                </label>
              </div>
              <ul><WaveTreeNode :node="activeScenario.root" /></ul>
            </details>
          </template>
          <div v-else class="panel-empty">
            <span class="brand-mark large"><i></i><i></i><i></i></span>
            <h2>Structure before prediction</h2>
            <p>Start a scan to build a recursive count, explicit invalidations and document-backed targets.</p>
          </div>
        </aside>
      </section>
    </main>

    <aside :class="['history-drawer', { open: historyOpen }]">
      <header>
        <div><span class="eyebrow">Permanent snapshots</span><h2>Scan history</h2></div>
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
        <span><strong>{{ item.symbol }}</strong>{{ item.timeframe }} · {{ item.session }}</span>
        <small>{{ formatDateTime(item.generated_at) }}</small>
        <code>{{ item.id.slice(0, 12) }}</code>
      </button>
      <p v-if="!historyLoading && history.length === 0" class="empty-copy">No saved analyses yet.</p>
    </aside>
    <div v-if="historyOpen" class="drawer-backdrop" @click="historyOpen = false"></div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import ChartWidget from './components/ChartWidget.vue'
import WaveTreeNode from './components/WaveTreeNode.vue'
import { useMarketData } from './composables/useMarketData'
import { scenarioDisplayName, visibleTargetZones } from './domain/presentation'
import type { RuleEvaluation, Timeframe, WaveNode } from './types/api'

interface ChartExport {
  exportPNG: () => void
}

interface AuditedRule {
  node: string
  result: RuleEvaluation
}

const timeframes: Timeframe[] = ['1m', '5m', '15m', '1h', '4h', '1D', '1W']
const chart = ref<ChartExport | null>(null)
const scale = ref<'ARITHMETIC' | 'LOG'>('ARITHMETIC')
const historyOpen = ref(false)
const copied = ref(false)
const visibleDegrees = ref<string[]>([])

const {
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
  selectScenario,
  selectComparison,
  exportJSON,
  initialize,
} = useMarketData()

function walkNodes(root: WaveNode | undefined, visit: (node: WaveNode) => void): void {
  if (!root) return
  visit(root)
  for (const child of root.children ?? []) walkNodes(child, visit)
}

const availableDegrees = computed(() => {
  const values = new Set<string>()
  walkNodes(activeScenario.value?.root, (node) => values.add(node.degree))
  return [...values]
})

watch(availableDegrees, (degrees) => {
  const next = new Set(visibleDegrees.value)
  for (const degree of degrees) next.add(degree)
  visibleDegrees.value = [...next]
}, { immediate: true })

const ruleAudit = computed<AuditedRule[]>(() => {
  const result: AuditedRule[] = []
  walkNodes(activeScenario.value?.root, (node) => {
    for (const evaluation of node.rule_evaluations) {
      result.push({ node: node.label || node.pattern, result: evaluation })
    }
  })
  return result.sort((left, right) => {
    const order = { FAIL: 0, PASS: 1, NOT_OBSERVABLE: 2, NOT_APPLICABLE: 3 }
    return order[left.result.status] - order[right.result.status]
  })
})
const activeTargets = computed(() => visibleTargetZones(activeScenario.value))

function toggleDegree(degree: string): void {
  visibleDegrees.value = visibleDegrees.value.includes(degree)
    ? visibleDegrees.value.filter((item) => item !== degree)
    : [...visibleDegrees.value, degree]
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
