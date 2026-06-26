<template>
  <div class="min-h-screen bg-[#070a13] text-gray-100 flex flex-col antialiased">
    <!-- Header Navigation -->
    <header class="border-b border-gray-800/80 bg-[#090d16]/70 backdrop-blur sticky top-0 z-40">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <!-- Premium Logo Icon -->
          <div class="w-9 h-9 rounded-lg bg-gradient-to-tr from-purple-600 to-indigo-500 flex items-center justify-center shadow-lg shadow-purple-500/20">
            <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
            </svg>
          </div>
          <div>
            <h1 id="main-header" class="text-xl font-bold tracking-tight bg-gradient-to-r from-white via-gray-100 to-gray-400 bg-clip-text text-transparent m-0">
              WAVESIGHT
            </h1>
            <p class="text-[10px] text-gray-400 tracking-wider uppercase font-semibold m-0">
              Elliott Wave Terminal
            </p>
          </div>
        </div>

        <div class="flex items-center gap-3">
          <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-500/10 text-purple-400 border border-purple-500/20">
            v1.0.0 Stable
          </span>
        </div>
      </div>
    </header>

    <!-- Main Workspace -->
    <main class="flex-grow max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-6 flex flex-col gap-6">
      
      <!-- Control Panel -->
      <section class="bg-[#090d16]/90 border border-gray-800/80 rounded-xl p-5 shadow-xl fade-in">
        <form @submit.prevent="fetchMarketData" class="grid grid-cols-1 md:grid-cols-4 gap-5 items-end">
          
          <!-- Ticker Search -->
          <div class="flex flex-col gap-2">
            <label for="ticker-search-input" class="text-xs font-semibold text-gray-400 uppercase tracking-wider">
              Asset Ticker
            </label>
            <div class="relative">
              <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none text-gray-500">
                <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
              </div>
              <input
                id="ticker-search-input"
                type="text"
                v-model="ticker"
                placeholder="e.g. AAPL"
                class="w-full bg-[#0d1222] border border-gray-800 focus:border-purple-500 focus:ring-1 focus:ring-purple-500 rounded-lg pl-9 pr-3 py-2 text-sm font-medium uppercase tracking-wide text-gray-100 placeholder-gray-600 transition-colors"
                :disabled="loading"
              />
            </div>
          </div>

          <!-- Timeframe Selector -->
          <div class="flex flex-col gap-2">
            <label for="timeframe-select" class="text-xs font-semibold text-gray-400 uppercase tracking-wider">
              Timeframe
            </label>
            <select
              id="timeframe-select"
              v-model="timeframe"
              class="w-full bg-[#0d1222] border border-gray-800 focus:border-purple-500 focus:ring-1 focus:ring-purple-500 rounded-lg px-3 py-2 text-sm font-medium text-gray-100 transition-colors cursor-pointer"
              :disabled="loading"
            >
              <option value="10m">10 Minutes (10m)</option>
              <option value="1h">1 Hour (1h)</option>
              <option value="1D">1 Day (1D)</option>
              <option value="1W">1 Week (1W)</option>
            </select>
          </div>

          <!-- Deviation Slider -->
          <div class="flex flex-col gap-2">
            <div class="flex justify-between items-center">
              <label for="deviation-slider" class="text-xs font-semibold text-gray-400 uppercase tracking-wider">
                ZigZag Deviation
              </label>
              <span class="text-xs font-bold text-purple-400 bg-purple-500/10 px-2 py-0.5 rounded border border-purple-500/20">
                {{ (deviation * 100).toFixed(1) }}%
              </span>
            </div>
            <div class="flex items-center gap-3">
              <input
                id="deviation-slider"
                type="range"
                v-model.number="deviation"
                min="0.01"
                max="0.05"
                step="0.005"
                class="w-full h-1.5 bg-gray-800 rounded-lg appearance-none cursor-pointer accent-purple-600"
                :disabled="loading"
              />
            </div>
          </div>

          <!-- Fetch Action -->
          <div>
            <button
              id="search-btn"
              type="submit"
              class="w-full bg-gradient-to-r from-purple-600 to-indigo-600 hover:from-purple-500 hover:to-indigo-500 active:scale-[0.99] text-white text-sm font-semibold py-2 px-4 rounded-lg shadow-lg shadow-purple-600/10 hover:shadow-purple-500/20 transition-all flex items-center justify-center gap-2 cursor-pointer"
              :disabled="loading"
            >
              <span v-if="loading" id="loading-spinner" class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></span>
              <span>{{ loading ? 'Analyzing...' : 'Scan Market' }}</span>
            </button>
          </div>
        </form>
      </section>

      <!-- Error Banner -->
      <transition name="fade">
        <div
          v-if="error"
          id="error-banner"
          class="bg-red-500/10 border border-red-500/30 rounded-xl p-4 flex items-start gap-3 text-red-400 fade-in"
        >
          <svg class="w-5 h-5 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="flex-grow">
            <h4 class="font-bold text-sm">Failed to process engine scan</h4>
            <p class="text-xs mt-1 text-red-400/90 leading-relaxed">{{ error }}</p>
          </div>
          <button @click="clearError" class="text-red-400/60 hover:text-red-400 transition-colors">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </transition>

      <!-- Chart & Details Grid -->
      <section class="grid grid-cols-1 lg:grid-cols-4 gap-6 items-start">
        
        <!-- Left 3 Columns: Chart Area -->
        <div class="lg:col-span-3 flex flex-col gap-4 relative">
          <!-- Loading skeleton overlay -->
          <div v-if="loading && candles.length === 0" class="absolute inset-0 bg-[#090d16]/40 backdrop-blur-sm z-20 flex items-center justify-center rounded-xl border border-gray-800/80">
            <div class="flex flex-col items-center gap-3">
              <div class="w-10 h-10 border-4 border-purple-500/20 border-t-purple-500 rounded-full animate-spin"></div>
              <span class="text-xs font-semibold text-gray-400">Loading candles & processing waves...</span>
            </div>
          </div>

          <!-- Chart Render Canvas -->
          <ChartWidget
            :candles="candles"
            :motiveWaves="motiveWaves"
            :correctiveWaves="correctiveWaves"
            class="fade-in"
          />

          <!-- Quick Statistics bar -->
          <div v-if="candles.length > 0" class="bg-[#090d16]/40 border border-gray-800/60 rounded-xl px-4 py-3 flex flex-wrap gap-x-8 gap-y-2 justify-between items-center text-xs text-gray-400 fade-in">
            <div class="flex items-center gap-2">
              <span class="w-1.5 h-1.5 rounded-full bg-green-500"></span>
              <span>Loaded Ticker: <strong class="text-gray-200">{{ ticker.toUpperCase() }}</strong></span>
            </div>
            <div>
              <span>Candles Analyzed: <strong class="text-gray-200">{{ candles.length }}</strong></span>
            </div>
            <div>
              <span>Period: <strong class="text-gray-200">{{ formatDate(candles[0].time) }}</strong> to <strong class="text-gray-200">{{ formatDate(candles[candles.length - 1].time) }}</strong></span>
            </div>
          </div>
        </div>

        <!-- Right 1 Column: Elliott Wave Details Panel -->
        <div class="bg-[#090d16]/90 border border-gray-800/80 rounded-xl p-5 shadow-xl flex flex-col gap-4 min-h-[500px] fade-in">
          <div>
            <h3 class="text-sm font-bold text-gray-300 uppercase tracking-wider m-0">
              Scanned Wave Patterns
            </h3>
            <p class="text-xs text-gray-500 mt-1">
              Pivots identified by the mathematical scanner.
            </p>
          </div>

          <!-- Empty State -->
          <div v-if="motiveWaves.length === 0 && correctiveWaves.length === 0" class="flex-grow flex flex-col items-center justify-center text-center p-6 border border-dashed border-gray-800 rounded-lg my-2">
            <svg class="w-8 h-8 text-gray-700 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h5 class="text-xs font-bold text-gray-400 m-0">No Waves Detected</h5>
            <p class="text-[10px] text-gray-500 mt-1 leading-normal">
              No motive or corrective waves validate standard Elliott rule constraints. Try adjusting the deviation slider.
            </p>
          </div>

          <!-- Lists of waves -->
          <div v-else class="flex flex-col gap-3 overflow-y-auto max-h-[420px] pr-1">
            
            <!-- Motive Waves -->
            <div v-for="(wave, index) in motiveWaves" :key="'motive-' + index" class="bg-[#0d1222] border border-gray-800 hover:border-purple-500/30 rounded-lg p-3 transition-colors">
              <div class="flex justify-between items-center">
                <span class="px-2 py-0.5 text-[9px] font-bold rounded" :class="wave.direction === 'BULLISH' ? 'bg-green-500/10 text-green-400 border border-green-500/20' : 'bg-red-500/10 text-red-400 border border-red-500/20'">
                  MOTIVE {{ wave.direction }}
                </span>
                <span class="text-[10px] text-gray-400 font-medium">
                  Conf: {{ (wave.confidence_score * 100).toFixed(0) }}%
                </span>
              </div>
              <div class="mt-2.5 text-xs grid grid-cols-2 gap-y-1.5 text-gray-400">
                <div>Start Price:</div>
                <div class="text-right text-gray-200 font-mono">${{ formatPrice(wave.start.price) }}</div>
                <div>Wave 5 Peak:</div>
                <div class="text-right text-gray-200 font-mono">${{ formatPrice(wave.w5.price) }}</div>
              </div>
              
              <!-- Purple Box Target Box Coordinates -->
              <div v-if="wave.purple_box" class="mt-2.5 pt-2 border-t border-gray-800/80">
                <div class="flex items-center gap-1.5 text-[10px] font-bold text-purple-400 uppercase tracking-wider">
                  <span class="w-1.5 h-1.5 rounded-full bg-purple-500 animate-pulse"></span>
                  Target Box (Purple Box)
                </div>
                <div class="mt-1 text-[11px] grid grid-cols-2 text-gray-500 font-mono leading-tight">
                  <div>Min Price:</div>
                  <div class="text-right text-gray-300">${{ formatPrice(wave.purple_box.min_price) }}</div>
                  <div>Max Price:</div>
                  <div class="text-right text-gray-300">${{ formatPrice(wave.purple_box.max_price) }}</div>
                  <div>Target window:</div>
                  <div class="text-right text-gray-400 text-[9px]">{{ formatDate(wave.purple_box.start_time) }} - {{ formatDate(wave.purple_box.end_time) }}</div>
                </div>
              </div>
            </div>

            <!-- Corrective Waves -->
            <div v-for="(wave, index) in correctiveWaves" :key="'corrective-' + index" class="bg-[#0d1222] border border-gray-800 hover:border-amber-500/30 rounded-lg p-3 transition-colors">
              <div class="flex justify-between items-center">
                <span class="px-2 py-0.5 text-[9px] font-bold rounded" :class="wave.direction === 'BULLISH' ? 'bg-green-500/10 text-green-400 border border-green-500/20' : 'bg-red-500/10 text-red-400 border border-red-500/20'">
                  CORRECTIVE {{ wave.direction }}
                </span>
                <span class="text-[9px] font-semibold bg-amber-500/10 text-amber-400 border border-amber-500/20 px-1.5 rounded uppercase">
                  {{ wave.type }}
                </span>
              </div>
              <div class="mt-2.5 text-xs grid grid-cols-2 gap-y-1.5 text-gray-400 font-mono">
                <div>Start ({{ wave.start.type }}):</div>
                <div class="text-right text-gray-200">${{ formatPrice(wave.start.price) }}</div>
                <div>A:</div>
                <div class="text-right text-gray-200">${{ formatPrice(wave.wa.price) }}</div>
                <div>B:</div>
                <div class="text-right text-gray-200">${{ formatPrice(wave.wb.price) }}</div>
                <div>C:</div>
                <div class="text-right text-gray-200">${{ formatPrice(wave.wc.price) }}</div>
              </div>
            </div>

          </div>
        </div>

      </section>
    </main>

    <!-- Footer -->
    <footer class="border-t border-gray-900 bg-[#070a13] py-6 mt-10">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center text-xs text-gray-500">
        <p>&copy; 2026 WaveSight Engine Core. All rights reserved. Mathematical pivot calculations executed with zero allocation overhead.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue';
import { useMarketData } from './composables/useMarketData';
import ChartWidget from './components/ChartWidget.vue';

const {
  ticker,
  timeframe,
  deviation,
  candles,
  motiveWaves,
  correctiveWaves,
  loading,
  error,
  fetchMarketData,
} = useMarketData();

const clearError = () => {
  error.value = null;
};

const formatDate = (timestamp: number) => {
  const date = new Date(timestamp * 1000);
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
};

const formatPrice = (price: number) => {
  return price.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
};

// Initial data load when component mounts
onMounted(() => {
  fetchMarketData();
});
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
