import { ref, shallowRef } from 'vue';

export interface Candle {
  time: number; // Unix timestamp in seconds
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface Pivot {
  time: number;
  price: number;
  type: 'HIGH' | 'LOW';
}

export interface TargetBox {
  min_price: number;
  max_price: number;
  start_time: number;
  end_time: number;
}

export interface MotiveWave {
  start: Pivot;
  w1: Pivot;
  w2: Pivot;
  w3: Pivot;
  w4: Pivot;
  w5: Pivot;
  direction: 'BULLISH' | 'BEARISH';
  confidence_score: number;
  purple_box?: TargetBox;
}

export interface CorrectiveWave {
  start: Pivot;
  wa: Pivot;
  wb: Pivot;
  wc: Pivot;
  type: 'ZIGZAG' | 'FLAT';
  direction: 'BULLISH' | 'BEARISH';
}

export interface AnalysisResponse {
  ticker: string;
  timeframe: string;
  candles: Candle[];
  motive_waves: MotiveWave[];
  corrective_waves: CorrectiveWave[];
}

export function useMarketData() {
  // Configurable search inputs and states
  const ticker = ref<string>('AAPL');
  const timeframe = ref<string>('1D');
  const deviation = ref<number>(0.02);

  // High-performance arrays using shallowRef to avoid Vue recursive reactive proxy overhead
  const candles = shallowRef<Candle[]>([]);
  const motiveWaves = shallowRef<MotiveWave[]>([]);
  const correctiveWaves = shallowRef<CorrectiveWave[]>([]);

  // Loading and error states
  const loading = ref<boolean>(false);
  const error = ref<string | null>(null);

  const fetchMarketData = async () => {
    loading.value = true;
    error.value = null;

    try {
      const apiBaseUrl = import.meta.env.VITE_API_BASE_URL;
      if (!apiBaseUrl) {
        throw new Error('VITE_API_BASE_URL environment variable is not defined.');
      }

      // Format ticker to uppercase for consistency
      const formattedTicker = ticker.value.trim().toUpperCase();
      if (!formattedTicker) {
        throw new Error('Ticker parameter cannot be empty.');
      }

      const params = new URLSearchParams({
        timeframe: timeframe.value,
        deviation: deviation.value.toString(),
      });

      const response = await fetch(`${apiBaseUrl}/api/analyze/${formattedTicker}?${params.toString()}`);

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || `API responded with status code ${response.status}`);
      }

      const data: AnalysisResponse = await response.json();

      // Ensure candles are sorted chronologically by time
      const sortedCandles = (data.candles || []).slice().sort((a, b) => a.time - b.time);

      // Mutate shallowRef values directly
      candles.value = sortedCandles;
      motiveWaves.value = data.motive_waves || [];
      correctiveWaves.value = data.corrective_waves || [];
    } catch (err: any) {
      console.error('Error fetching market analysis:', err);
      error.value = err.message || 'An unexpected error occurred while fetching analysis data.';
      // Reset values in case of failure to prevent displaying stale chart visuals
      candles.value = [];
      motiveWaves.value = [];
      correctiveWaves.value = [];
    } finally {
      loading.value = false;
    }
  };

  return {
    ticker,
    timeframe,
    deviation,
    candles,
    motiveWaves,
    correctiveWaves,
    loading,
    error,
    fetchMarketData,
  };
}
