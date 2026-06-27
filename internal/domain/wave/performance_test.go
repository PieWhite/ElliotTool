package wave

import (
	"os"
	"runtime"
	"testing"
	"time"

	"WaveSight/internal/market"
)

func TestEnginePerformanceEnvelopes(t *testing.T) {
	if os.Getenv("WAVESIGHT_PERF_TEST") != "1" {
		t.Skip("set WAVESIGHT_PERF_TEST=1 to run performance acceptance gates")
	}
	tests := []struct {
		name        string
		candleCount int
		maxDuration time.Duration
	}{
		{name: "10k", candleCount: 10_000, maxDuration: 2 * time.Second},
		{name: "50k", candleCount: 50_000, maxDuration: 5 * time.Second},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			candles := testCandles(test.candleCount)
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			started := time.Now()
			result := NewEngine().Analyze(AnalyzeInput{
				Candles: candles, Timeframe: market.Timeframe1m,
				Session: market.SessionRTH, MaxScenarios: 5, TickSize: 0.01,
			})
			elapsed := time.Since(started)
			runtime.ReadMemStats(&after)
			residentHeap := uint64(0)
			if after.HeapSys > before.HeapSys {
				residentHeap = after.HeapSys - before.HeapSys
			}
			if elapsed > test.maxDuration {
				t.Fatalf("%d-candle analysis took %s, limit %s", test.candleCount, elapsed, test.maxDuration)
			}
			if residentHeap > 512<<20 {
				t.Fatalf("%d-candle analysis retained %.1f MiB of heap, limit 512 MiB", test.candleCount, float64(residentHeap)/(1<<20))
			}
			if len(result.Scenarios) == 0 {
				t.Fatal("performance analysis returned no explicit scenario state")
			}
		})
	}
}

func BenchmarkEngine10k(b *testing.B) {
	candles := testCandles(10_000)
	engine := NewEngine()
	b.ReportAllocs()
	b.ResetTimer()
	for iteration := 0; iteration < b.N; iteration++ {
		_ = engine.Analyze(AnalyzeInput{
			Candles: candles, Timeframe: market.Timeframe1m,
			Session: market.SessionRTH, MaxScenarios: 5, TickSize: 0.01,
		})
	}
}
