<template>
  <div class="relative w-full h-full bg-[#0d111d] rounded-xl border border-gray-800/80 overflow-hidden shadow-2xl">
    <!-- Chart Mounting Target -->
    <div ref="chartContainer" class="w-full h-[550px]"></div>

    <!-- Floating Stats Overlay -->
    <div class="absolute top-4 left-4 z-10 flex gap-2 pointer-events-none">
      <span class="px-3 py-1 bg-gray-900/80 backdrop-blur text-xs font-semibold text-purple-400 rounded-md border border-purple-500/20 shadow-lg uppercase tracking-wider">
        Canvas Renderer Active
      </span>
      <!-- Step 10: Scenario badge -->
      <span
        v-if="activeScenario"
        class="px-3 py-1 backdrop-blur text-xs font-semibold rounded-md border shadow-lg uppercase tracking-wider"
        :class="activeScenario.bias === 'BULLISH'
          ? 'bg-green-500/10 text-green-400 border-green-500/30'
          : 'bg-red-500/10 text-red-400 border-red-500/30'"
      >
        {{ activeScenario.bias === 'BULLISH' ? '▲' : '▼' }}
        {{ activeScenario.bias }} Scenario
        <span class="ml-1 opacity-70">{{ (activeScenario.confidence * 100).toFixed(0) }}%</span>
      </span>
      <!-- Legacy badges when no scenario is active -->
      <template v-else>
        <span v-if="motiveWaves.length > 0" class="px-3 py-1 bg-green-500/10 backdrop-blur text-xs font-semibold text-green-400 rounded-md border border-green-500/20 shadow-lg">
          Motive Waves: {{ motiveWaves.length }}
        </span>
        <span v-if="correctiveWaves.length > 0" class="px-3 py-1 bg-amber-500/10 backdrop-blur text-xs font-semibold text-amber-400 rounded-md border border-amber-500/20 shadow-lg">
          Corrective Waves: {{ correctiveWaves.length }}
        </span>
        <span v-if="incompleteWaves.length > 0" class="px-3 py-1 bg-cyan-500/10 backdrop-blur text-xs font-semibold text-cyan-400 rounded-md border border-cyan-500/20 shadow-lg">
          Developing: {{ incompleteWaves.length }}
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { createChart, CandlestickSeries, LineSeries, HistogramSeries, createSeriesMarkers } from 'lightweight-charts';
import type { IChartApi, ISeriesApi, Time } from 'lightweight-charts';
import type { Candle, MotiveWave, CorrectiveWave, IncompleteWave, AnalysisScenario } from '../composables/useMarketData';
import { BoxPrimitive } from './BoxPrimitive';

const props = defineProps<{
  candles: Candle[];
  motiveWaves: MotiveWave[];
  correctiveWaves: CorrectiveWave[];
  incompleteWaves: IncompleteWave[];
  // Step 10: When provided, renders the scenario structures instead of the flat arrays.
  activeScenario?: AnalysisScenario;
}>();

const chartContainer = ref<HTMLDivElement | null>(null);
let chart: IChartApi | null = null;
let candlestickSeries: ISeriesApi<'Candlestick'> | null = null;
let volumeSeries: ISeriesApi<'Histogram'> | null = null;
let waveSeriesList: ISeriesApi<'Line'>[] = [];

// Redraw chart layers when input data updates
const renderChartData = () => {
  if (!chart) return;

  // 1. Clear previous drawings and extra series to prevent memory leaks or overlay visual pollution
  waveSeriesList.forEach(s => {
    try {
      chart?.removeSeries(s);
    } catch (e) {
      console.warn('Failed to remove wave line series:', e);
    }
  });
  waveSeriesList = [];

  if (candlestickSeries) {
    try {
      chart.removeSeries(candlestickSeries);
      candlestickSeries = null;
    } catch (e) {
      console.warn('Failed to remove candlestick series:', e);
    }
  }

  if (volumeSeries) {
    try {
      chart.removeSeries(volumeSeries);
      volumeSeries = null;
    } catch (e) {
      console.warn('Failed to remove volume series:', e);
    }
  }

  // 2. Return early if there is no data to plot
  if (props.candles.length === 0) return;

  // 3. Re-create main candlestick series using unified v5 addSeries API
  candlestickSeries = chart.addSeries(CandlestickSeries, {
    upColor: '#10b981',
    downColor: '#ef4444',
    borderVisible: false,
    wickUpColor: '#10b981',
    wickDownColor: '#ef4444',
  });

  const chartCandles = props.candles.map(c => ({
    time: c.time as Time,
    open: c.open,
    high: c.high,
    low: c.low,
    close: c.close,
  }));

  if (candlestickSeries) {
    candlestickSeries.setData(chartCandles);
  }

  // 4. Re-create volume series histogram at the bottom
  volumeSeries = chart.addSeries(HistogramSeries, {
    priceFormat: {
      type: 'volume',
    },
    priceScaleId: '', // Overlay scale ID
  });

  if (volumeSeries) {
    volumeSeries.priceScale().applyOptions({
      scaleMargins: {
        top: 0.82, // Reserve top 82% space for candles
        bottom: 0,
      },
    });

    const volumeData = props.candles.map(c => ({
      time: c.time as Time,
      value: c.volume,
      color: c.close >= c.open ? 'rgba(16, 185, 129, 0.22)' : 'rgba(239, 68, 68, 0.22)',
    }));

    volumeSeries.setData(volumeData);
  }

  // 5. Render: use active scenario structures when available, else fall back to flat arrays.
  if (props.activeScenario && props.activeScenario.structures.length > 0) {
    renderScenarioStructures(props.activeScenario);
  } else {
    renderFlatWaves();
  }

  // 8. Auto-fit candles to view
  chart.timeScale().fitContent();
};

// renderScenarioStructures — draws all WaveStructure entries from a scenario.
// Renders a colour-coded line for each structure's pivot sequence and attaches
// any purple_boxes to the candlestick series.
const renderScenarioStructures = (scenario: AnalysisScenario) => {
  const isBullishScenario = scenario.bias === 'BULLISH';

  scenario.structures.forEach((ws) => {
    if (!ws.pivots || ws.pivots.length < 2) return;

    // Pick colour by structure type
    let color = isBullishScenario ? '#22c55e' : '#ef4444'; // default motive green/red
    let lineStyle = 0; // solid
    if (ws.type.startsWith('CORRECTIVE_')) {
      if (ws.type === 'CORRECTIVE_TRIANGLE') color = '#2dd4bf'; // teal
      else if (ws.type === 'CORRECTIVE_WXY') color = '#818cf8'; // indigo
      else color = '#f59e0b'; // amber
      lineStyle = 2; // dashed
    } else if (ws.type === 'INCOMPLETE_123') {
      color = '#22d3ee'; // cyan
      lineStyle = 3; // dotted
    } else if (ws.type === 'MOTIVE_DIAGONAL') {
      lineStyle = 2;
    }

    const lineSeries = chart!.addSeries(LineSeries, {
      color,
      lineWidth: 2,
      lineStyle,
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    const points = ws.pivots.map(p => ({ time: p.time as number, value: p.price }));

    // Ensure strictly ascending times
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    // Attach purple boxes if present
    if (ws.purple_boxes && candlestickSeries) {
      ws.purple_boxes.forEach(box => {
        const primitive = new BoxPrimitive(
          box.start_time,
          box.end_time,
          box.min_price,
          box.max_price,
        );
        candlestickSeries!.attachPrimitive(primitive);
      });
    }
  });

  // Auto-fit candles to view
  chart?.timeScale().fitContent();
};

// renderFlatWaves — legacy rendering of the flat motive/corrective/incomplete arrays.
const renderFlatWaves = () => {
  // 5. Draw Motive Waves (1-5) and TargetBoxes (Purple Box)
  //    Step 8: Diagonal waves are drawn dashed; truncated Wave 5 label is "5T".
  props.motiveWaves.forEach((wave) => {
    if (!wave.start || !wave.w1 || !wave.w2 || !wave.w3 || !wave.w4 || !wave.w5) return;

    const isBullish = wave.direction === 'BULLISH';
    const waveColor = isBullish ? '#22c55e' : '#ef4444'; // Green or Red

    // Diagonals render as dashed (lineStyle 2), standard impulses render solid (lineStyle 0).
    const lineStyle = wave.is_diagonal ? 2 : 0;

    // Create line series connecting pivots
    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle,
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    const points = [
      { time: wave.start.time as number, value: wave.start.price },
      { time: wave.w1.time as number, value: wave.w1.price },
      { time: wave.w2.time as number, value: wave.w2.price },
      { time: wave.w3.time as number, value: wave.w3.price },
      { time: wave.w4.time as number, value: wave.w4.price },
      { time: wave.w5.time as number, value: wave.w5.price },
    ];

    // Ensure strictly ascending times
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    // Wave 5 label: show "5T" if truncated, otherwise "5".
    const wave5Label = wave.is_truncated ? '5T' : '5';

    // Place labels for Motive Pivots (1 to 5/5T)
    const markers = [
      {
        time: points[1].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#3b82f6', // Bright Blue for pivots
        text: '1',
        size: 1.4,
        price: wave.w1.price,
      },
      {
        time: points[2].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#3b82f6',
        text: '2',
        size: 1.4,
        price: wave.w2.price,
      },
      {
        time: points[3].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#3b82f6',
        text: '3',
        size: 1.4,
        price: wave.w3.price,
      },
      {
        time: points[4].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#3b82f6',
        text: '4',
        size: 1.4,
        price: wave.w4.price,
      },
      {
        time: points[5].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: wave.is_truncated ? '#f59e0b' : '#3b82f6', // Amber for truncated Wave 5
        text: wave5Label,
        size: 1.4,
        price: wave.w5.price,
      },
    ];

    // Sort markers chronologically to comply with Lightweight Charts rules
    markers.sort((a, b) => (a.time as number) - (b.time as number));
    
    // Attach series markers using Lightweight Charts v5 createSeriesMarkers helper
    createSeriesMarkers(lineSeries, markers);

    // Attach all Purple Box TargetBox primitives to the candlestick series.
    if (wave.purple_boxes && candlestickSeries) {
      wave.purple_boxes.forEach(box => {
        const primitive = new BoxPrimitive(
          box.start_time,
          box.end_time,
          box.min_price,
          box.max_price
        );
        candlestickSeries!.attachPrimitive(primitive);
      });
    }
  });

  // 6. Draw Corrective Waves (A-B-C / A-B-C-D-E / WXY)
  props.correctiveWaves.forEach((wave) => {
    if (!wave.start || !wave.wa || !wave.wb || !wave.wc) return;

    // Use Amber for standard ABC, Teal for Triangles, Indigo for WXY Double Threes.
    let waveColor = '#f59e0b'; // Amber (ZigZag/Flat)
    if (wave.type === 'TRIANGLE') waveColor = '#2dd4bf';  // Teal
    if (wave.type === 'WXY') waveColor = '#818cf8';       // Indigo

    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle: 2, // Dashed line for all corrective waves
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    // Build point sequence depending on type.
    const points: { time: number; value: number }[] = [
      { time: wave.start.time as number, value: wave.start.price },
      { time: wave.wa.time as number, value: wave.wa.price },
      { time: wave.wb.time as number, value: wave.wb.price },
      { time: wave.wc.time as number, value: wave.wc.price },
    ];

    // Triangle & WXY: add extra pivots
    if ((wave.type === 'TRIANGLE' || wave.type === 'WXY') && wave.wx) {
      points.push({ time: wave.wx.time as number, value: wave.wx.price });
    }
    if (wave.wd) {
      points.push({ time: wave.wd.time as number, value: wave.wd.price });
    }
    if (wave.we) {
      points.push({ time: wave.we.time as number, value: wave.we.price });
    }

    // Sort points chronologically (they should already be, but guard against edge cases)
    points.sort((a, b) => a.time - b.time);

    // Ensure strictly ascending times
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    // Place labels for ABC pivots (and D/E/X for complex types)
    const labels = ['A', 'B', 'C'];
    if (wave.type === 'TRIANGLE') {
      labels.push('D', 'E');
    } else if (wave.type === 'WXY') {
      labels.push('X', 'Y');
    }

    const labelColor = wave.type === 'TRIANGLE'
      ? '#2dd4bf'
      : wave.type === 'WXY'
        ? '#818cf8'
        : '#ec4899'; // Pinkish-magenta for standard ABC

    // Slice pivots 1..n for labeling (skip Start at index 0)
    const markerPoints = points.slice(1);
    const markers = markerPoints.slice(0, labels.length).map((p, idx) => ({
      time: p.time as Time,
      position: 'inBar' as const,
      shape: 'circle' as const,
      color: labelColor,
      text: labels[idx],
      size: 1.4,
      price: p.value,
    }));

    markers.sort((a, b) => (a.time as number) - (b.time as number));
    
    // Attach series markers using Lightweight Charts v5 createSeriesMarkers helper
    createSeriesMarkers(lineSeries, markers);

    // Attach all corrective Purple Box TargetBox primitives to the candlestick series.
    if (wave.purple_boxes && candlestickSeries) {
      wave.purple_boxes.forEach(box => {
        const primitive = new BoxPrimitive(
          box.start_time,
          box.end_time,
          box.min_price,
          box.max_price
        );
        candlestickSeries!.attachPrimitive(primitive);
      });
    }
  });

  // 7. Draw Incomplete (developing) 1-2-3 waves — Step 8
  //    Rendered as a distinct dashed cyan line; Wave 4 target_box rendered in teal/cyan.
  props.incompleteWaves.forEach((wave) => {
    if (!wave.start || !wave.w1 || !wave.w2 || !wave.w3) return;

    const lineSeries = chart!.addSeries(LineSeries, {
      color: '#22d3ee', // Cyan
      lineWidth: 2,
      lineStyle: 3, // Dotted dashed — distinct from solid motive and dashed corrective
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    const points = [
      { time: wave.start.time as number, value: wave.start.price },
      { time: wave.w1.time as number, value: wave.w1.price },
      { time: wave.w2.time as number, value: wave.w2.price },
      { time: wave.w3.time as number, value: wave.w3.price },
    ];

    // Ensure strictly ascending times
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    // Label the 1-2-3 pivots
    const markers = [
      {
        time: points[1].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#22d3ee',
        text: '①',
        size: 1.4,
        price: wave.w1.price,
      },
      {
        time: points[2].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#22d3ee',
        text: '②',
        size: 1.4,
        price: wave.w2.price,
      },
      {
        time: points[3].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#22d3ee',
        text: '③',
        size: 1.4,
        price: wave.w3.price,
      },
    ];

    markers.sort((a, b) => (a.time as number) - (b.time as number));
    createSeriesMarkers(lineSeries, markers);

    // Render the predictive Wave 4 target_box as a teal/cyan BoxPrimitive
    if (wave.target_box && candlestickSeries) {
      const box = new BoxPrimitive(
        wave.target_box.start_time,
        wave.target_box.end_time,
        wave.target_box.min_price,
        wave.target_box.max_price,
        'rgba(20, 184, 166, 0.15)',  // teal fill
        'rgba(45, 212, 191, 0.80)',  // teal stroke
      );
      candlestickSeries.attachPrimitive(box);
    }
  });

  // Auto-fit candles to view
  chart?.timeScale().fitContent();
};

const handleResize = () => {
  if (chart && chartContainer.value) {
    chart.resize(chartContainer.value.clientWidth, chartContainer.value.clientHeight);
  }
};

onMounted(() => {
  if (!chartContainer.value) return;

  // Initialize Canvas Charting widget
  chart = createChart(chartContainer.value, {
    layout: {
      background: { color: '#090d16' },
      textColor: '#9ca3af',
    },
    grid: {
      vertLines: { color: 'rgba(31, 41, 55, 0.4)' },
      horzLines: { color: 'rgba(31, 41, 55, 0.4)' },
    },
    crosshair: {
      mode: 0, // Crosshair moves freely
      vertLine: {
        color: 'rgba(147, 51, 234, 0.4)',
        width: 1,
        style: 3,
      },
      horzLine: {
        color: 'rgba(147, 51, 234, 0.4)',
        width: 1,
        style: 3,
      },
    },
    timeScale: {
      borderColor: '#1f2937',
      timeVisible: true,
      secondsVisible: false,
    },
  });

  // Load initial candles if any
  renderChartData();

  window.addEventListener('resize', handleResize);
});

// Watch for prop modifications and redraw canvas (including activeScenario toggle)
watch(() => [props.candles, props.motiveWaves, props.correctiveWaves, props.incompleteWaves, props.activeScenario], () => {
  renderChartData();
}, { deep: false });

onUnmounted(() => {
  window.removeEventListener('resize', handleResize);
  if (chart) {
    chart.remove(); // Unified cleanup method in Lightweight Charts v5
    chart = null;
  }
});
</script>
