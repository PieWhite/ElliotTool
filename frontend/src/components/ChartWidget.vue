<template>
  <section class="chart-shell" aria-label="Elliott Wave price chart">
    <div v-if="!snapshot" class="chart-empty">
      <div class="empty-orbit" aria-hidden="true"></div>
      <strong>Ready for a theory-conformant scan</strong>
      <span>The active count, invalidations and conditional Purple Boxes appear here.</span>
    </div>
    <div ref="container" class="chart-canvas"></div>
    <div v-if="snapshot" class="chart-key">
      <span><i class="key-wave"></i> Active wave tree</span>
      <span><i class="key-box"></i> Confluence zone</span>
      <span><i class="key-invalid"></i> Invalidation</span>
      <span v-if="comparison"><i class="key-compare"></i> Comparison</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  CandlestickSeries,
  ColorType,
  createChart,
  HistogramSeries,
  PriceScaleMode,
} from 'lightweight-charts'
import type {
  CandlestickData,
  HistogramData,
  IChartApi,
  ISeriesApi,
  Time,
  UTCTimestamp,
  WhitespaceData,
} from 'lightweight-charts'
import { WaveOverlayPrimitive } from './WaveOverlayPrimitive'
import type { AnalysisSnapshot, Scenario } from '../types/api'

const props = defineProps<{
  snapshot: AnalysisSnapshot | null
  scenario: Scenario | null
  comparison: Scenario | null
  scale: 'ARITHMETIC' | 'LOG'
  visibleDegrees: string[]
}>()

const container = ref<HTMLDivElement | null>(null)
let chart: IChartApi | null = null
let candleSeries: ISeriesApi<'Candlestick', Time> | null = null
let volumeSeries: ISeriesApi<'Histogram', Time> | null = null
let overlay: WaveOverlayPrimitive | null = null
let resizeObserver: ResizeObserver | null = null

function renderData(): void {
  if (!chart || !candleSeries || !volumeSeries) return
  const snapshot = props.snapshot
  if (!snapshot) {
    candleSeries.setData([])
    volumeSeries.setData([])
    overlay?.update({ scenario: null, comparison: null, futureBars: [], visibleDegrees: props.visibleDegrees })
    return
  }

  const candleData: Array<CandlestickData<UTCTimestamp> | WhitespaceData<UTCTimestamp>> =
    snapshot.candles.map((candle) => ({
      time: candle.time as UTCTimestamp,
      open: candle.open,
      high: candle.high,
      low: candle.low,
      close: candle.close,
    }))
  candleData.push(...snapshot.future_bars.map((time) => ({ time: time as UTCTimestamp })))
  candleSeries.setData(candleData)

  const volumeData: HistogramData<UTCTimestamp>[] = snapshot.candles.map((candle) => ({
    time: candle.time as UTCTimestamp,
    value: candle.volume,
    color: candle.close >= candle.open ? 'rgba(74, 222, 177, 0.24)' : 'rgba(255, 102, 125, 0.22)',
  }))
  volumeSeries.setData(volumeData)
  overlay?.update({
    scenario: props.scenario,
    comparison: props.comparison,
    futureBars: snapshot.future_bars,
    visibleDegrees: props.visibleDegrees,
  })
  const from = Math.max(0, snapshot.candles.length - 180)
  const to = snapshot.candles.length + Math.min(42, snapshot.future_bars.length)
  chart.timeScale().setVisibleLogicalRange({ from, to })
}

function applyScale(): void {
  chart?.priceScale('right').applyOptions({
    mode: props.scale === 'LOG' ? PriceScaleMode.Logarithmic : PriceScaleMode.Normal,
  })
}

function exportPNG(): void {
  if (!chart || !props.snapshot) return
  const canvas = chart.takeScreenshot(true, false)
  const link = document.createElement('a')
  link.href = canvas.toDataURL('image/png')
  link.download = `wavesight-${props.snapshot.request.symbol}-${props.snapshot.id}.png`
  link.click()
}

defineExpose({ exportPNG })

onMounted(() => {
  if (!container.value) return
  chart = createChart(container.value, {
    layout: {
      background: { type: ColorType.Solid, color: '#080a12' },
      textColor: '#8990a6',
      fontFamily: 'Inter, system-ui, sans-serif',
    },
    grid: {
      vertLines: { color: 'rgba(91, 83, 120, 0.09)' },
      horzLines: { color: 'rgba(91, 83, 120, 0.12)' },
    },
    crosshair: {
      vertLine: { color: 'rgba(184, 116, 255, 0.38)', labelBackgroundColor: '#6d35a8' },
      horzLine: { color: 'rgba(184, 116, 255, 0.38)', labelBackgroundColor: '#6d35a8' },
    },
    rightPriceScale: {
      borderColor: 'rgba(126, 115, 153, 0.25)',
      scaleMargins: { top: 0.08, bottom: 0.24 },
    },
    timeScale: {
      borderColor: 'rgba(126, 115, 153, 0.25)',
      timeVisible: true,
      secondsVisible: false,
      rightOffset: 30,
      fixLeftEdge: true,
    },
    handleScroll: true,
    handleScale: true,
  })
  candleSeries = chart.addSeries(CandlestickSeries, {
    upColor: '#3dd7a5',
    downColor: '#f05c78',
    borderVisible: false,
    wickUpColor: '#3dd7a5',
    wickDownColor: '#f05c78',
  })
  volumeSeries = chart.addSeries(HistogramSeries, {
    priceFormat: { type: 'volume' },
    priceScaleId: 'volume',
    lastValueVisible: false,
    priceLineVisible: false,
  })
  volumeSeries.priceScale().applyOptions({
    scaleMargins: { top: 0.82, bottom: 0 },
  })
  overlay = new WaveOverlayPrimitive({
    scenario: props.scenario,
    comparison: props.comparison,
    futureBars: props.snapshot?.future_bars ?? [],
    visibleDegrees: props.visibleDegrees,
  })
  candleSeries.attachPrimitive(overlay)
  resizeObserver = new ResizeObserver(() => {
    if (chart && container.value) {
      chart.resize(container.value.clientWidth, container.value.clientHeight)
    }
  })
  resizeObserver.observe(container.value)
  applyScale()
  renderData()
})

watch(
  () => [props.snapshot, props.scenario, props.comparison, props.visibleDegrees] as const,
  renderData,
  { deep: false },
)
watch(() => props.scale, applyScale)

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
  if (candleSeries && overlay) candleSeries.detachPrimitive(overlay)
  chart?.remove()
  chart = null
  candleSeries = null
  volumeSeries = null
  overlay = null
})
</script>
