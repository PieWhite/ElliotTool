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
import { createChart, CandlestickSeries, LineSeries, HistogramSeries, createSeriesMarkers, LineStyle } from 'lightweight-charts';
import type { IChartApi, ISeriesApi, Time } from 'lightweight-charts';
import type { Candle, MotiveWave, CorrectiveWave, IncompleteWave, AnalysisScenario, WaveStructure, Pivot } from '../composables/useMarketData';
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

// Helper functions for Elliott Wave degree labeling and coloring
function formatLabel(text: string, degree?: string): string {
  if (!degree) return text;
  const d = degree.toUpperCase();
  const cleanText = text.replace(/T$/, ''); // Strip truncated T if present
  const isTruncated = text.endsWith('T');
  const suffix = isTruncated ? 'T' : '';

  if (d === 'PRIMARY') {
    const primaryMap: Record<string, string> = {
      '1': '①', '2': '②', '3': '③', '4': '④', '5': '⑤',
      'A': 'Ⓐ', 'B': 'Ⓑ', 'C': 'Ⓒ', 'D': 'Ⓓ', 'E': 'Ⓔ', 'X': 'Ⓧ', 'Y': 'Ⓨ'
    };
    return (primaryMap[cleanText] || cleanText) + suffix;
  }
  if (d === 'INTERMEDIATE') {
    return `(${cleanText})${suffix}`;
  }
  if (d === 'MINOR') {
    return cleanText + suffix;
  }
  if (d === 'MINUTE') {
    const minuteMap: Record<string, string> = {
      '1': 'i', '2': 'ii', '3': 'iii', '4': 'iv', '5': 'v',
      'A': 'a', 'B': 'b', 'C': 'c', 'D': 'd', 'E': 'e', 'X': 'x', 'Y': 'y'
    };
    return (minuteMap[cleanText] || cleanText.toLowerCase()) + suffix;
  }
  if (d === 'MINUETTE') {
    const minuetteMap: Record<string, string> = {
      '1': 'i', '2': 'ii', '3': 'iii', '4': 'iv', '5': 'v',
      'A': 'a', 'B': 'b', 'C': 'c', 'D': 'd', 'E': 'e', 'X': 'x', 'Y': 'y'
    };
    const lower = minuetteMap[cleanText] || cleanText.toLowerCase();
    return `[${lower}]${suffix}`;
  }
  if (d === 'SUBMINUETTE') {
    const subminMap: Record<string, string> = {
      '1': 'i', '2': 'ii', '3': 'iii', '4': 'iv', '5': 'v',
      'A': 'a', 'B': 'b', 'C': 'c', 'D': 'd', 'E': 'e', 'X': 'x', 'Y': 'y'
    };
    const lower = subminMap[cleanText] || cleanText.toLowerCase();
    return `(${lower})${suffix}`;
  }
  return text;
}

function getDegreeColor(degree: string | undefined, defaultColor: string): string {
  if (!degree) return defaultColor;
  const d = degree.toUpperCase();
  switch (d) {
    case 'PRIMARY': return '#a855f7';      // Purple
    case 'INTERMEDIATE': return '#f43f5e'; // Rose/Pink
    case 'MINUTE': return '#06b6d4';       // Cyan
    case 'MINUETTE': return '#10b981';     // Emerald
    case 'SUBMINUETTE': return '#818cf8';  // Indigo
    default: return defaultColor;
  }
}

type ScenarioStructureDirection = 'BULLISH' | 'BEARISH';

type ScenarioLineWidth = 1 | 2 | 3 | 4;

type ScenarioRenderConfig = {
  defaultColor: string;
  lineStyle: LineStyle;
  lineWidth: ScenarioLineWidth;
  boxFillColor: string;
  boxStrokeColor: string;
};

type MarkerCollisionState = {
  occupancy: Map<number, number>;
  priceOffsetStep: number;
};

function getStructureDirection(ws: WaveStructure, scenarioBias: ScenarioStructureDirection): ScenarioStructureDirection {
  if (!ws.pivots || ws.pivots.length < 2) return scenarioBias;

  if (ws.type.startsWith('MOTIVE_') || ws.type === 'INCOMPLETE_123') {
    return ws.pivots[0].type === 'LOW' ? 'BULLISH' : 'BEARISH';
  }

  const start = ws.pivots[0];
  const end = ws.pivots[ws.pivots.length - 1];
  if (end.price > start.price) return 'BULLISH';
  if (end.price < start.price) return 'BEARISH';
  return scenarioBias;
}

function getScenarioRenderConfig(ws: WaveStructure, scenarioBias: ScenarioStructureDirection): ScenarioRenderConfig {
  const direction = getStructureDirection(ws, scenarioBias);
  const directionalColor = direction === 'BULLISH' ? '#22c55e' : '#ef4444';
  const directionalBox = direction === 'BULLISH'
    ? { fill: 'rgba(34, 197, 94, 0.14)', stroke: 'rgba(34, 197, 94, 0.72)' }
    : { fill: 'rgba(239, 68, 68, 0.14)', stroke: 'rgba(239, 68, 68, 0.72)' };

  if (ws.type === 'INCOMPLETE_123') {
    return {
      defaultColor: '#22d3ee',
      lineStyle: LineStyle.Dotted,
      lineWidth: 3,
      boxFillColor: 'rgba(20, 184, 166, 0.16)',
      boxStrokeColor: 'rgba(45, 212, 191, 0.82)',
    };
  }

  if (ws.type === 'MOTIVE_DIAGONAL') {
    return {
      defaultColor: directionalColor,
      lineStyle: LineStyle.Dashed,
      lineWidth: 3,
      boxFillColor: directionalBox.fill,
      boxStrokeColor: directionalBox.stroke,
    };
  }

  if (ws.type.startsWith('MOTIVE_')) {
    return {
      defaultColor: directionalColor,
      lineStyle: LineStyle.Solid,
      lineWidth: 3,
      boxFillColor: directionalBox.fill,
      boxStrokeColor: directionalBox.stroke,
    };
  }

  if (ws.type === 'CORRECTIVE_TRIANGLE') {
    return {
      defaultColor: '#2dd4bf',
      lineStyle: LineStyle.Dashed,
      lineWidth: 2,
      boxFillColor: directionalBox.fill,
      boxStrokeColor: directionalBox.stroke,
    };
  }

  if (ws.type === 'CORRECTIVE_WXY') {
    return {
      defaultColor: '#818cf8',
      lineStyle: LineStyle.Dashed,
      lineWidth: 2,
      boxFillColor: directionalBox.fill,
      boxStrokeColor: directionalBox.stroke,
    };
  }

  return {
    defaultColor: '#f59e0b',
    lineStyle: LineStyle.Dashed,
    lineWidth: 2,
    boxFillColor: directionalBox.fill,
    boxStrokeColor: directionalBox.stroke,
  };
}

function collectAllPivots(ws: WaveStructure): Pivot[] {
  let pivots = [...(ws.pivots || [])];
  if (ws.sub_structures) {
    ws.sub_structures.forEach(sub => {
      pivots = pivots.concat(collectAllPivots(sub));
    });
  }
  return pivots;
}

function getScenarioPriceOffsetStep(scenario: AnalysisScenario): number {
  const allPivots = scenario.structures.flatMap(collectAllPivots);
  const prices = allPivots.map(p => p.price);
  if (prices.length === 0) return 0.01;

  const min = Math.min(...prices);
  const max = Math.max(...prices);
  const range = max - min;
  return Math.max(range * 0.015, Math.max(Math.abs(max), 1) * 0.001);
}

function getScenarioLabels(ws: WaveStructure): string[] {
  if (ws.type.startsWith('MOTIVE_')) {
    return ['1', '2', '3', '4', ws.type === 'MOTIVE_TRUNCATED' ? '5T' : '5'];
  }
  if (ws.type === 'INCOMPLETE_123') {
    return ['1', '2', '3'];
  }
  if (ws.type === 'CORRECTIVE_TRIANGLE') {
    return ['A', 'B', 'C', 'D', 'E'];
  }
  if (ws.type === 'CORRECTIVE_WXY') {
    return ['A', 'B', 'C', 'X', 'Y'];
  }
  if (ws.type.startsWith('CORRECTIVE_')) {
    return ['A', 'B', 'C'];
  }
  return [];
}

function attachCollisionAwareMarkers(
  markerPoints: { time: number; value: number }[],
  labels: string[],
  color: string,
  degree: string | undefined,
  collisionState: MarkerCollisionState,
) {
  if (!chart || markerPoints.length === 0 || labels.length === 0) return;

  const labelPoints = markerPoints.slice(0, labels.length).map((point, idx) => {
    const currentSlot = collisionState.occupancy.get(point.time) ?? 0;
    collisionState.occupancy.set(point.time, currentSlot + 1);

    const direction = currentSlot % 2 === 0 ? 1 : -1;
    const stackLevel = Math.floor((currentSlot + 1) / 2);
    const verticalOffset = currentSlot === 0
      ? 0
      : direction * stackLevel * collisionState.priceOffsetStep;

    return {
      time: point.time,
      value: point.value + verticalOffset,
      position: currentSlot === 0
        ? 'inBar' as const
        : direction > 0 ? 'aboveBar' as const : 'belowBar' as const,
      text: formatLabel(labels[idx], degree),
    };
  });

  const labelSeries = chart.addSeries(LineSeries, {
    color: 'rgba(0, 0, 0, 0)',
    lineWidth: 1,
    lineStyle: LineStyle.Solid,
    lineVisible: false,
    crosshairMarkerVisible: false,
    lastValueVisible: false,
    priceLineVisible: false,
  });

  labelSeries.setData(labelPoints.map(p => ({ time: p.time as Time, value: p.value })));
  waveSeriesList.push(labelSeries);

  const markers = labelPoints.map(point => ({
    time: point.time as Time,
    position: point.position,
    shape: 'circle' as const,
    color,
    text: point.text,
    size: 1.4,
  }));

  markers.sort((a, b) => (a.time as number) - (b.time as number));
  createSeriesMarkers(labelSeries, markers);
}

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
  const scenarioBias = scenario.bias;
  const markerCollisionState: MarkerCollisionState = {
    occupancy: new Map<number, number>(),
    priceOffsetStep: getScenarioPriceOffsetStep(scenario),
  };

  const drawStructure = (ws: WaveStructure) => {
    if (!ws.pivots || ws.pivots.length < 2) return;

    const renderConfig = getScenarioRenderConfig(ws, scenarioBias);
    const color = getDegreeColor(ws.degree, renderConfig.defaultColor);

    const lineSeries = chart!.addSeries(LineSeries, {
      color,
      lineWidth: renderConfig.lineWidth,
      lineStyle: renderConfig.lineStyle,
      lineVisible: true,
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    const points = ws.pivots.map(p => ({ time: p.time as number, value: p.price }));

    // Ensure strictly ascending times for Lightweight Charts while preserving cross-structure
    // timestamp collisions for the label-offset pass below.
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    attachCollisionAwareMarkers(
      points.slice(1),
      getScenarioLabels(ws),
      color,
      ws.degree,
      markerCollisionState,
    );

    if (ws.purple_boxes && candlestickSeries) {
      ws.purple_boxes.forEach(box => {
        const primitive = new BoxPrimitive(
          box.start_time,
          box.end_time,
          box.min_price,
          box.max_price,
          renderConfig.boxFillColor,
          renderConfig.boxStrokeColor,
        );
        candlestickSeries!.attachPrimitive(primitive);
      });
    }

    // Recursively draw sub-structures
    if (ws.sub_structures && ws.sub_structures.length > 0) {
      ws.sub_structures.forEach(drawStructure);
    }
  };

  scenario.structures.forEach(drawStructure);

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
    const defaultColor = isBullish ? '#22c55e' : '#ef4444'; // Green or Red
    const waveColor = getDegreeColor(wave.degree, defaultColor);

    // Diagonals render as solid in this test.
    const lineStyle = LineStyle.Solid;

    // Create line series connecting pivots
    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle,
      lineVisible: true,
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
        color: waveColor,
        text: formatLabel('1', wave.degree),
        size: 1.4,
        price: wave.w1.price,
      },
      {
        time: points[2].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel('2', wave.degree),
        size: 1.4,
        price: wave.w2.price,
      },
      {
        time: points[3].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel('3', wave.degree),
        size: 1.4,
        price: wave.w3.price,
      },
      {
        time: points[4].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel('4', wave.degree),
        size: 1.4,
        price: wave.w4.price,
      },
      {
        time: points[5].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel(wave5Label, wave.degree),
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
    let defaultColor = '#f59e0b'; // Amber (ZigZag/Flat)
    if (wave.type === 'TRIANGLE') defaultColor = '#2dd4bf';  // Teal
    if (wave.type === 'WXY') defaultColor = '#818cf8';       // Indigo

    const waveColor = getDegreeColor(wave.degree, defaultColor);

    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle: LineStyle.Solid, // Solid line for testing
      lineVisible: true,
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

    // Slice pivots 1..n for labeling (skip Start at index 0)
    const markerPoints = points.slice(1);
    const markers = markerPoints.slice(0, labels.length).map((p, idx) => ({
      time: p.time as Time,
      position: 'inBar' as const,
      shape: 'circle' as const,
      color: waveColor,
      text: formatLabel(labels[idx], wave.degree),
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

  // 7. Draw Incomplete (developing) 1-2-3 waves
  props.incompleteWaves.forEach((wave) => {
    if (!wave.start || !wave.w1 || !wave.w2 || !wave.w3) return;

    const defaultColor = '#22d3ee'; // Cyan
    const waveColor = getDegreeColor(wave.degree, defaultColor);

    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle: LineStyle.Solid, // Solid line for testing
      lineVisible: true,
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
        color: waveColor,
        text: formatLabel('1', wave.degree),
        size: 1.4,
        price: wave.w1.price,
      },
      {
        time: points[2].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel('2', wave.degree),
        size: 1.4,
        price: wave.w2.price,
      },
      {
        time: points[3].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: waveColor,
        text: formatLabel('3', wave.degree),
        size: 1.4,
        price: wave.w3.price,
      },
    ];

    markers.sort((a, b) => (a.time as number) - (b.time as number));
    createSeriesMarkers(lineSeries, markers);

    // Render the predictive Wave 4 target_box
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
