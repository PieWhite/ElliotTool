package master

import (
	"math"
	"os"
	"testing"
	"time"

	"WaveSight/internal/market"
)

func TestMasterEnginePerformanceEnvelope(t *testing.T) {
	if os.Getenv("WAVESIGHT_MASTER_PERF_TEST") != "1" {
		t.Skip("set WAVESIGHT_MASTER_PERF_TEST=1 to run the 200k-bar acceptance envelope")
	}
	const minuteCount = 200_000
	start := time.Date(2024, time.January, 2, 14, 30, 0, 0, time.UTC)
	minutes := make([]market.DerivedCandle, minuteCount)
	for index := range minutes {
		price := 100 + float64(index)*0.0008 + math.Sin(float64(index)/97)*4
		timestamp := start.Add(time.Duration(index) * time.Minute).Unix()
		minutes[index] = market.DerivedCandle{
			Candle: market.Candle{
				Time: timestamp, BarIndex: index, Open: price - 0.1,
				High: price + 0.4, Low: price - 0.4, Close: price + 0.1, Volume: 100,
			},
			HighTime: timestamp, LowTime: timestamp, SourceFrom: timestamp, SourceTo: timestamp,
			Provenance: market.ProvenanceMinuteDerived,
		}
	}
	views := map[market.Timeframe][]market.DerivedCandle{
		market.Timeframe1m:  minutes,
		market.Timeframe5m:  sampleDerived(minutes, 5),
		market.Timeframe15m: sampleDerived(minutes, 15),
		market.Timeframe1h:  sampleDerived(minutes, 60),
		market.Timeframe4h:  sampleDerived(minutes, 240),
		market.Timeframe1D:  sampleDerived(minutes, 390),
		market.Timeframe1W:  sampleDerived(minutes, 1_950),
	}
	started := time.Now()
	snapshot, projected := NewEngine().Analyze(AnalyzeInput{
		Symbol: "PERF", Session: market.SessionRTH, AsOf: time.Unix(minutes[len(minutes)-1].Time, 0),
		FocusTimeframe: market.Timeframe1D, MaxScenarios: 5,
		Views: market.CanonicalViews{Views: views},
		Manifest: DatasetManifest{
			MinuteDetailFrom: minutes[0].Time,
			MinuteDetailTo:   minutes[len(minutes)-1].Time,
			NativeMinuteRows: len(minutes),
		},
	})
	elapsed := time.Since(started)
	if elapsed > 10*time.Second {
		t.Fatalf("master analysis took %s, acceptance limit is 10s", elapsed)
	}
	if len(snapshot.Scenarios) == 0 || len(projected) != 7 {
		t.Fatalf("master result has %d scenarios and %d views", len(snapshot.Scenarios), len(projected))
	}
}

func sampleDerived(source []market.DerivedCandle, step int) []market.DerivedCandle {
	result := make([]market.DerivedCandle, 0, len(source)/step+1)
	for index := 0; index < len(source); index += step {
		end := index + step
		if end > len(source) {
			end = len(source)
		}
		item := source[index]
		item.BarIndex = len(result)
		item.SourceFrom = source[index].Time
		item.SourceTo = source[end-1].Time
		item.Close = source[end-1].Close
		for scan := index + 1; scan < end; scan++ {
			if source[scan].High > item.High {
				item.High = source[scan].High
				item.HighTime = source[scan].HighTime
			}
			if source[scan].Low < item.Low {
				item.Low = source[scan].Low
				item.LowTime = source[scan].LowTime
			}
			item.Volume += source[scan].Volume
		}
		result = append(result, item)
	}
	return result
}
