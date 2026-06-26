import { ref, shallowRef, computed } from 'vue';

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
  purple_boxes?: TargetBox[];
  is_diagonal: boolean;   // Step 8: true if wave is a converging diagonal (wedge)
  is_truncated: boolean;  // Step 8: true if Wave 5 fails to exceed Wave 3
}

export interface CorrectiveWave {
  start: Pivot;
  wa: Pivot;
  wb: Pivot;
  wc: Pivot;
  wd?: Pivot;  // Triangle D-pivot or WXY Y-wave A-pivot
  we?: Pivot;  // Triangle E-pivot or WXY Y-wave C terminal pivot
  wx?: Pivot;  // WXY X-wave connector pivot
  type: 'ZIGZAG' | 'FLAT' | 'TRIANGLE' | 'WXY';
  direction: 'BULLISH' | 'BEARISH';
  purple_boxes?: TargetBox[];
}

// Step 8: Incomplete (developing) 1-2-3 structure with a predictive Wave 4 target box.
export interface IncompleteWave {
  start: Pivot;
  w1: Pivot;
  w2: Pivot;
  w3: Pivot;
  direction: 'BULLISH' | 'BEARISH';
  confidence_score: number;
  target_box?: TargetBox;
}

// Step 10: Generic wave structure used inside a scenario (type-agnostic for frontend rendering).
export interface WaveStructure {
  type: string;                // e.g. "MOTIVE_IMPULSE", "CORRECTIVE_ZIGZAG", "INCOMPLETE_123"
  pivots: Pivot[];
  purple_boxes?: TargetBox[];
  confidence_score: number;
}

// Step 10: A directional scenario (Primary or Alternate) containing all supporting structures.
export interface AnalysisScenario {
  bias: 'BULLISH' | 'BEARISH';
  confidence: number;
  structures: WaveStructure[];
}

export interface AnalysisResponse {
  ticker: string;
  timeframe: string;
  candles: Candle[];
  // Step 10: Probabilistic scenario pair
  scenarios?: {
    primary: AnalysisScenario;
    alternate: AnalysisScenario;
  };
  // Legacy flat arrays (backward compat)
  motive_waves: MotiveWave[];
  corrective_waves: CorrectiveWave[];
  incomplete_waves: IncompleteWave[]; // Step 8: developing 1-2-3 structures
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
  const incompleteWaves = shallowRef<IncompleteWave[]>([]);

  // Step 10: Scenario pair and active scenario toggle
  const scenarios = shallowRef<AnalysisResponse['scenarios']>(undefined);
  // 'primary' | 'alternate' — controls which scenario the chart renders
  const activeScenarioBias = ref<'primary' | 'alternate'>('primary');

  // Derived active scenario (reactive to bias toggle + scenarios update)
  const activeScenario = computed<AnalysisScenario | undefined>(() => {
    if (!scenarios.value) return undefined;
    return activeScenarioBias.value === 'primary'
      ? scenarios.value.primary
      : scenarios.value.alternate;
  });

  const setScenario = (bias: 'primary' | 'alternate') => {
    activeScenarioBias.value = bias;
  };

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
      incompleteWaves.value = data.incomplete_waves || [];
      scenarios.value = data.scenarios;

      // Default the toggle to primary on every fresh fetch
      activeScenarioBias.value = 'primary';
    } catch (err: any) {
      console.error('Error fetching market analysis:', err);
      error.value = err.message || 'An unexpected error occurred while fetching analysis data.';
      // Reset values in case of failure to prevent displaying stale chart visuals
      candles.value = [];
      motiveWaves.value = [];
      correctiveWaves.value = [];
      incompleteWaves.value = [];
      scenarios.value = undefined;
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
    incompleteWaves,
    scenarios,
    activeScenario,
    activeScenarioBias,
    setScenario,
    loading,
    error,
    fetchMarketData,
  };
}
