<template>
  <div class="relative w-full h-full bg-[#0d111d] rounded-xl border border-gray-800/80 overflow-hidden shadow-2xl">
    <!-- Chart Mounting Target -->
    <div ref="chartContainer" class="w-full h-[550px]"></div>

    <!-- Floating Stats Overlay -->
    <div class="absolute top-4 left-4 z-10 flex gap-2 pointer-events-none">
      <span class="px-3 py-1 bg-gray-900/80 backdrop-blur text-xs font-semibold text-purple-400 rounded-md border border-purple-500/20 shadow-lg uppercase tracking-wider">
        Canvas Renderer Active
      </span>
      <span v-if="motiveWaves.length > 0" class="px-3 py-1 bg-green-500/10 backdrop-blur text-xs font-semibold text-green-400 rounded-md border border-green-500/20 shadow-lg">
        Motive Waves: {{ motiveWaves.length }}
      </span>
      <span v-if="correctiveWaves.length > 0" class="px-3 py-1 bg-amber-500/10 backdrop-blur text-xs font-semibold text-amber-400 rounded-md border border-amber-500/20 shadow-lg">
        Corrective Waves: {{ correctiveWaves.length }}
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, watch } from 'vue';
import { createChart, CandlestickSeries, LineSeries, HistogramSeries, createSeriesMarkers } from 'lightweight-charts';
import type { IChartApi, ISeriesApi, Time } from 'lightweight-charts';
import type { Candle, MotiveWave, CorrectiveWave } from '../composables/useMarketData';
import { BoxPrimitive } from './BoxPrimitive';

const props = defineProps<{
  candles: Candle[];
  motiveWaves: MotiveWave[];
  correctiveWaves: CorrectiveWave[];
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

  // 5. Draw Motive Waves (1-5) and TargetBoxes (Purple Box)
  props.motiveWaves.forEach((wave) => {
    if (!wave.start || !wave.w1 || !wave.w2 || !wave.w3 || !wave.w4 || !wave.w5) return;

    const isBullish = wave.direction === 'BULLISH';
    const waveColor = isBullish ? '#22c55e' : '#ef4444'; // Green or Red

    // Create line series connecting pivots
    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle: 0, // Solid line
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

    // Place labels for Motive Pivots (1 to 5)
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
        color: '#3b82f6',
        text: '5',
        size: 1.4,
        price: wave.w5.price,
      },
    ];

    // Sort markers chronologically to comply with Lightweight Charts rules
    markers.sort((a, b) => (a.time as number) - (b.time as number));
    
    // Attach series markers using Lightweight Charts v5 createSeriesMarkers helper
    createSeriesMarkers(lineSeries, markers);

    // Attach Purple Box TargetBox Primitive to Candlestick series if exists
    if (wave.purple_box && candlestickSeries) {
      const box = new BoxPrimitive(
        wave.purple_box.start_time,
        wave.purple_box.end_time,
        wave.purple_box.min_price,
        wave.purple_box.max_price
      );
      candlestickSeries.attachPrimitive(box);
    }
  });

  // 6. Draw Corrective Waves (A-B-C)
  props.correctiveWaves.forEach((wave) => {
    if (!wave.start || !wave.wa || !wave.wb || !wave.wc) return;

    // Use Amber or Indigo accent to differentiate corrective structures
    const waveColor = '#f59e0b'; // Amber

    const lineSeries = chart!.addSeries(LineSeries, {
      color: waveColor,
      lineWidth: 2,
      lineStyle: 2, // Dashed line for corrective waves
      crosshairMarkerVisible: false,
      lastValueVisible: false,
      priceLineVisible: false,
    });

    const points = [
      { time: wave.start.time as number, value: wave.start.price },
      { time: wave.wa.time as number, value: wave.wa.price },
      { time: wave.wb.time as number, value: wave.wb.price },
      { time: wave.wc.time as number, value: wave.wc.price },
    ];

    // Ensure strictly ascending times
    for (let i = 1; i < points.length; i++) {
      if (points[i].time <= points[i - 1].time) {
        points[i].time = points[i - 1].time + 1;
      }
    }

    lineSeries.setData(points.map(p => ({ time: p.time as Time, value: p.value })));
    waveSeriesList.push(lineSeries);

    // Place labels for Corrective Pivots (A, B, C)
    const markers = [
      {
        time: points[1].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#ec4899', // Pinkish-magenta for corrective labels
        text: 'A',
        size: 1.4,
        price: wave.wa.price,
      },
      {
        time: points[2].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#ec4899',
        text: 'B',
        size: 1.4,
        price: wave.wb.price,
      },
      {
        time: points[3].time as Time,
        position: 'inBar' as const,
        shape: 'circle' as const,
        color: '#ec4899',
        text: 'C',
        size: 1.4,
        price: wave.wc.price,
      },
    ];

    markers.sort((a, b) => (a.time as number) - (b.time as number));
    
    // Attach series markers using Lightweight Charts v5 createSeriesMarkers helper
    createSeriesMarkers(lineSeries, markers);
  });

  // 7. Auto-fit candles to view
  chart.timeScale().fitContent();
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

// Watch for prop modifications and redraw canvas
watch(() => [props.candles, props.motiveWaves, props.correctiveWaves], () => {
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
