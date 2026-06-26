package elliott_test

import (
	"math"
	"testing"

	"WaveSight/pkg/elliott"
	"WaveSight/pkg/model"
)

func TestMatchMotiveWaves(t *testing.T) {
	tests := []struct {
		name          string
		pivots        []model.Pivot
		expectedCount int
		verify        func(t *testing.T, results []model.MotiveWave)
	}{
		{
			name:          "Insufficient pivots (empty)",
			pivots:        []model.Pivot{},
			expectedCount: 0,
		},
		{
			name: "Insufficient pivots (less than 6)",
			pivots: []model.Pivot{
				{Time: 100, Price: 10.0, Type: model.PivotLow},
				{Time: 110, Price: 15.0, Type: model.PivotHigh},
				{Time: 120, Price: 12.0, Type: model.PivotLow},
				{Time: 130, Price: 25.0, Type: model.PivotHigh},
				{Time: 140, Price: 20.0, Type: model.PivotLow},
			},
			expectedCount: 0,
		},
		{
			name: "Textbook Bullish 5-Wave Sequence",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retracement: 61.8%)
				{Time: 130, Price: 300.0, Type: model.PivotHigh}, // W3 (len: 161.8, extension: 161.8%)
				{Time: 140, Price: 220.0, Type: model.PivotLow},  // W4 (stays above W1 high of 200)
				{Time: 150, Price: 343.6, Type: model.PivotHigh}, // W5 (len: 123.6, matches 61.8% of net 0-to-3 distance)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.MotiveWave) {
				mw := results[0]
				if mw.Direction != "BULLISH" {
					t.Errorf("expected direction BULLISH, got %q", mw.Direction)
				}
				if mw.Start.Price != 100.0 || mw.W1.Price != 200.0 || mw.W2.Price != 138.2 ||
					mw.W3.Price != 300.0 || mw.W4.Price != 220.0 || mw.W5.Price != 343.6 {
					t.Errorf("unexpected pivot values in matched MotiveWave: %+v", mw)
				}
				if mw.ConfidenceScore < 0.90 {
					t.Errorf("expected high confidence score for textbook setup, got %f", mw.ConfidenceScore)
				}
				if len(mw.PurpleBoxes) != 3 {
					t.Fatalf("expected 3 PurpleBoxes, got %d", len(mw.PurpleBoxes))
				}
				assertTargetBox(t, mw.PurpleBoxes[0], 320.0, 151, 159)
				assertTargetBox(t, mw.PurpleBoxes[1], 343.6, 151, 159)
				assertTargetBox(t, mw.PurpleBoxes[2], 381.8, 151, 159)
				if mw.W5.Price < mw.PurpleBoxes[1].MinPrice || mw.W5.Price > mw.PurpleBoxes[1].MaxPrice {
					t.Errorf("expected Wave 5 peak to land inside the 61.8%% net target box")
				}
			},
		},
		{
			name: "Invalid Wave 2 Retracement (breaks below Start)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 110.0, Type: model.PivotHigh}, // W1
				{Time: 120, Price: 98.0, Type: model.PivotLow},   // W2 (invalid: below 100)
				{Time: 130, Price: 120.0, Type: model.PivotHigh}, // W3
				{Time: 140, Price: 112.0, Type: model.PivotLow},  // W4
				{Time: 150, Price: 130.0, Type: model.PivotHigh}, // W5
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Wave 3 Shortest",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 120.0, Type: model.PivotHigh}, // W1 (len: 20)
				{Time: 120, Price: 110.0, Type: model.PivotLow},  // W2
				{Time: 130, Price: 115.0, Type: model.PivotHigh}, // W3 (len: 5, shortest)
				{Time: 140, Price: 112.0, Type: model.PivotLow},  // W4
				{Time: 150, Price: 135.0, Type: model.PivotHigh}, // W5 (len: 23)
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Wave 4 Overlap (below W1 high)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 110.0, Type: model.PivotHigh}, // W1
				{Time: 120, Price: 102.0, Type: model.PivotLow},  // W2
				{Time: 130, Price: 125.0, Type: model.PivotHigh}, // W3
				{Time: 140, Price: 108.0, Type: model.PivotLow},  // W4 (invalid: overlaps with W1 territory [100-110], 108 <= 110)
				{Time: 150, Price: 130.0, Type: model.PivotHigh}, // W5
			},
			expectedCount: 0,
		},
		{
			name: "Textbook Bearish 5-Wave Sequence",
			pivots: []model.Pivot{
				{Time: 100, Price: 300.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 200.0, Type: model.PivotLow},  // W1 (len: 100)
				{Time: 120, Price: 261.8, Type: model.PivotHigh}, // W2 (retracement: 61.8%)
				{Time: 130, Price: 100.0, Type: model.PivotLow},  // W3 (len: 161.8, extension: 161.8%)
				{Time: 140, Price: 180.0, Type: model.PivotHigh}, // W4 (stays below W1 low of 200)
				{Time: 150, Price: 56.4, Type: model.PivotLow},   // W5 (len: 123.6, matches 61.8% of net 0-to-3 distance)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.MotiveWave) {
				mw := results[0]
				if mw.Direction != "BEARISH" {
					t.Errorf("expected direction BEARISH, got %q", mw.Direction)
				}
				if mw.Start.Price != 300.0 || mw.W1.Price != 200.0 || mw.W2.Price != 261.8 ||
					mw.W3.Price != 100.0 || mw.W4.Price != 180.0 || mw.W5.Price != 56.4 {
					t.Errorf("unexpected pivot values in matched MotiveWave: %+v", mw)
				}
				if mw.ConfidenceScore < 0.90 {
					t.Errorf("expected high confidence score for textbook setup, got %f", mw.ConfidenceScore)
				}
				if len(mw.PurpleBoxes) != 3 {
					t.Fatalf("expected 3 PurpleBoxes, got %d", len(mw.PurpleBoxes))
				}
				assertTargetBox(t, mw.PurpleBoxes[0], 80.0, 151, 159)
				assertTargetBox(t, mw.PurpleBoxes[1], 56.4, 151, 159)
				assertTargetBox(t, mw.PurpleBoxes[2], 18.2, 151, 159)
				if mw.W5.Price < mw.PurpleBoxes[1].MinPrice || mw.W5.Price > mw.PurpleBoxes[1].MaxPrice {
					t.Errorf("expected Wave 5 low to land inside the 61.8%% net target box")
				}
			},
		},
		{
			name: "Invalid Bearish Wave 2 Retracement (breaks above Start)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 90.0, Type: model.PivotLow},   // W1
				{Time: 120, Price: 102.0, Type: model.PivotHigh}, // W2 (invalid: above 100)
				{Time: 130, Price: 80.0, Type: model.PivotLow},   // W3
				{Time: 140, Price: 88.0, Type: model.PivotHigh},  // W4
				{Time: 150, Price: 70.0, Type: model.PivotLow},   // W5
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Bearish Wave 3 Shortest",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 80.0, Type: model.PivotLow},   // W1 (len: 20)
				{Time: 120, Price: 90.0, Type: model.PivotHigh},  // W2
				{Time: 130, Price: 85.0, Type: model.PivotLow},   // W3 (len: 5, shortest)
				{Time: 140, Price: 88.0, Type: model.PivotHigh},  // W4
				{Time: 150, Price: 65.0, Type: model.PivotLow},   // W5 (len: 23)
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Bearish Wave 4 Overlap (above W1 low)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 90.0, Type: model.PivotLow},   // W1
				{Time: 120, Price: 98.0, Type: model.PivotHigh},  // W2
				{Time: 130, Price: 75.0, Type: model.PivotLow},   // W3
				{Time: 140, Price: 92.0, Type: model.PivotHigh},  // W4 (invalid: overlaps with W1 territory [90-100], 92 >= 90)
				{Time: 150, Price: 70.0, Type: model.PivotLow},   // W5
			},
			expectedCount: 0,
		},
		{
			name: "Perfect Fibonacci Bullish Setup (100% matches)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retracement: 61.8%)
				{Time: 130, Price: 300.0, Type: model.PivotHigh}, // W3 (len: 161.8, extension: 161.8%)
				{Time: 140, Price: 220.0, Type: model.PivotLow},  // W4 (stays above W1 high of 200.0)
				{Time: 150, Price: 343.6, Type: model.PivotHigh}, // W5 (len: 123.6, matches 61.8% of net 0-to-3 distance)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.MotiveWave) {
				mw := results[0]
				if mw.ConfidenceScore < 0.95 {
					t.Errorf("expected very high confidence score, got %f", mw.ConfidenceScore)
				}
			},
		},
		{
			name: "Valid but Sloppy Setup (fails Fib filters)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 170.0, Type: model.PivotLow},  // W2 (retraces 30%, valid but sloppy)
				{Time: 130, Price: 280.0, Type: model.PivotHigh}, // W3 (len: 110, extension: 110%)
				{Time: 140, Price: 210.0, Type: model.PivotLow},  // W4 (no overlap, valid)
				{Time: 150, Price: 360.0, Type: model.PivotHigh}, // W5 (len: 150, valid)
			},
			expectedCount: 0, // Should be filtered out because it misses Fib targets (score is 0.0)
		},
		{
			name: "Valid Bullish Converging Diagonal (Wedge)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retraces 61.8%)
				{Time: 130, Price: 216.8, Type: model.PivotHigh}, // W3 (len: 78.6, which is ~78.6% of W1)
				{Time: 140, Price: 180.0, Type: model.PivotLow},  // W4 (overlaps into Wave 1 territory since 180.0 <= 200.0)
				{Time: 150, Price: 228.6, Type: model.PivotHigh}, // W5 (len: 48.6, which is ~61.8% of W3 = 48.57)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.MotiveWave) {
				mw := results[0]
				if !mw.IsDiagonal {
					t.Errorf("expected IsDiagonal to be true")
				}
				if mw.IsTruncated {
					t.Errorf("expected IsTruncated to be false")
				}
				if mw.ConfidenceScore < 0.90 {
					t.Errorf("expected high confidence score for diagonal, got %f", mw.ConfidenceScore)
				}
			},
		},
		{
			name: "Invalid Bullish Non-Converging Overlapping Diagonal",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2
				{Time: 130, Price: 216.8, Type: model.PivotHigh}, // W3 (len: 78.6)
				{Time: 140, Price: 180.0, Type: model.PivotLow},  // W4 (overlaps)
				{Time: 150, Price: 280.0, Type: model.PivotHigh}, // W5 (len: 100, not converging since len3 <= len5)
			},
			expectedCount: 0,
		},
		{
			name: "Valid Bullish Truncated Fifth Wave",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 150.0, Type: model.PivotHigh}, // W1 (len: 50)
				{Time: 120, Price: 119.1, Type: model.PivotLow},  // W2 (retraces 61.8%)
				{Time: 130, Price: 250.0, Type: model.PivotHigh}, // W3 (len: 130.9, extension: ~2.618, Wave 3 extended)
				{Time: 140, Price: 200.0, Type: model.PivotLow},  // W4 (no overlap)
				{Time: 150, Price: 230.9, Type: model.PivotHigh}, // W5 (len: 30.9, fails to exceed W3 high 250.0, but matches 61.8% of W4)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.MotiveWave) {
				mw := results[0]
				if !mw.IsTruncated {
					t.Errorf("expected IsTruncated to be true")
				}
				if mw.IsDiagonal {
					t.Errorf("expected IsDiagonal to be false")
				}
				if mw.ConfidenceScore < 0.90 {
					t.Errorf("expected high confidence score for truncation, got %f", mw.ConfidenceScore)
				}
			},
		},
		{
			name: "Invalid Truncation (Wave 3 not extended)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retraces 61.8%)
				{Time: 130, Price: 238.2, Type: model.PivotHigh}, // W3 (len: 100, not extended since len3 <= len1)
				{Time: 140, Price: 200.0, Type: model.PivotLow},  // W4
				{Time: 150, Price: 219.1, Type: model.PivotHigh}, // W5 (fails to exceed W3 high of 238.2, invalid truncation because W3 not extended)
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := elliott.MatchMotiveWaves(tt.pivots)
			if len(results) != tt.expectedCount {
				t.Fatalf("expected %d results, got %d", tt.expectedCount, len(results))
			}
			if tt.verify != nil {
				tt.verify(t, results)
			}
		})
	}
}

func TestMotiveWavePurpleBoxMatrix(t *testing.T) {
	pivots := []model.Pivot{
		{Time: 100, Price: 100.0, Type: model.PivotLow},
		{Time: 110, Price: 200.0, Type: model.PivotHigh},
		{Time: 120, Price: 138.2, Type: model.PivotLow},
		{Time: 130, Price: 300.0, Type: model.PivotHigh},
		{Time: 140, Price: 220.0, Type: model.PivotLow},
		{Time: 150, Price: 343.6, Type: model.PivotHigh},
	}

	results := elliott.MatchMotiveWaves(pivots)
	if len(results) != 1 {
		t.Fatalf("expected 1 motive wave, got %d", len(results))
	}

	boxes := results[0].PurpleBoxes
	if len(boxes) != 3 {
		t.Fatalf("expected exactly 3 PurpleBoxes, got %d", len(boxes))
	}

	assertTargetBox(t, boxes[0], 320.0, 151, 159) // Wave 5 = 100% of Wave 1 from Wave 4.
	assertTargetBox(t, boxes[1], 343.6, 151, 159) // Wave 5 = 61.8% of net 0->3 from Wave 4.
	assertTargetBox(t, boxes[2], 381.8, 151, 159) // Wave 5 = 161.8% of Wave 1 from Wave 4.
}

func BenchmarkMatchMotiveWaves(b *testing.B) {
	// Construct a realistic list of pivots (e.g., alternating series of highs/lows)
	pivots := make([]model.Pivot, 1000)
	for i := 0; i < 1000; i++ {
		t := model.PivotLow
		price := 100.0 + float64(i%2)*10.0
		if i%2 == 1 {
			t = model.PivotHigh
		}
		pivots[i] = model.Pivot{
			Time:  int64(i * 100),
			Price: price,
			Type:  t,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = elliott.MatchMotiveWaves(pivots)
	}
}

func TestMatchCorrectiveWaves(t *testing.T) {
	tests := []struct {
		name          string
		pivots        []model.Pivot
		expectedCount int
		verify        func(t *testing.T, results []model.CorrectiveWave)
	}{
		{
			name:          "Insufficient pivots (empty)",
			pivots:        []model.Pivot{},
			expectedCount: 0,
		},
		{
			name: "Insufficient pivots (less than 4)",
			pivots: []model.Pivot{
				{Time: 100, Price: 10.0, Type: model.PivotHigh},
				{Time: 110, Price: 5.0, Type: model.PivotLow},
				{Time: 120, Price: 8.0, Type: model.PivotHigh},
			},
			expectedCount: 0,
		},
		{
			name: "Textbook Bearish ZigZag correction (correcting bullish impulse)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 50.0, Type: model.PivotLow},   // WA (length: 50)
				{Time: 120, Price: 70.0, Type: model.PivotHigh},  // WB (retracement: 40% [20/50])
				{Time: 130, Price: 20.0, Type: model.PivotLow},   // WC (length: 50 [100% of WA], cleanly breaks past 50.0)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.CorrectiveWave) {
				cw := results[0]
				if cw.Direction != "BEARISH" {
					t.Errorf("expected direction BEARISH, got %q", cw.Direction)
				}
				if cw.Type != "ZIGZAG" {
					t.Errorf("expected type ZIGZAG, got %q", cw.Type)
				}
				if cw.Start.Price != 100.0 || cw.WA.Price != 50.0 || cw.WB.Price != 70.0 || cw.WC.Price != 20.0 {
					t.Errorf("unexpected pivot values: %+v", cw)
				}
			},
		},
		{
			name: "Textbook Bullish Flat correction (correcting bearish impulse)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 150.0, Type: model.PivotHigh}, // WA (length: 50)
				{Time: 120, Price: 100.0, Type: model.PivotLow},  // WB (retracement: 100% [50/50])
				{Time: 130, Price: 150.0, Type: model.PivotHigh}, // WC (length: 50 [100% of WA])
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.CorrectiveWave) {
				cw := results[0]
				if cw.Direction != "BULLISH" {
					t.Errorf("expected direction BULLISH, got %q", cw.Direction)
				}
				if cw.Type != "FLAT" {
					t.Errorf("expected type FLAT, got %q", cw.Type)
				}
				if cw.Start.Price != 100.0 || cw.WA.Price != 150.0 || cw.WB.Price != 100.0 || cw.WC.Price != 150.0 {
					t.Errorf("unexpected pivot values: %+v", cw)
				}
			},
		},
		{
			name: "Invalid corrective structure (Wave B retracing 150% of Wave A)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 50.0, Type: model.PivotLow},   // WA (length: 50)
				{Time: 120, Price: 125.0, Type: model.PivotHigh}, // WB (retraces 150%)
				{Time: 130, Price: 75.0, Type: model.PivotLow},   // WC
			},
			expectedCount: 0,
		},
		{
			name: "Invalid ZigZag Wave C fails to cleanly break past WA",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 50.0, Type: model.PivotLow},   // WA (length: 50)
				{Time: 120, Price: 70.0, Type: model.PivotHigh},  // WB (retracement: 40% [20/50])
				{Time: 130, Price: 51.0, Type: model.PivotLow},   // WC (terminates above WA of 50.0, no clean break)
			},
			expectedCount: 0,
		},
		{
			name: "Invalid ZigZag Wave C length too short (80% of WA)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 50.0, Type: model.PivotLow},   // WA (length: 50)
				{Time: 120, Price: 70.0, Type: model.PivotHigh},  // WB (retracement: 40% [20/50])
				{Time: 130, Price: 30.0, Type: model.PivotLow},   // WC (length: 40 [80% of WA], invalid)
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Flat Wave B retraces too little (80%)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 150.0, Type: model.PivotHigh}, // WA (length: 50)
				{Time: 120, Price: 110.0, Type: model.PivotLow},  // WB (retracement: 80% [40/50])
				{Time: 130, Price: 150.0, Type: model.PivotHigh}, // WC (length: 40)
			},
			expectedCount: 0,
		},
		{
			name: "Invalid Flat Wave C too long (140% of WA)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 150.0, Type: model.PivotHigh}, // WA (length: 50)
				{Time: 120, Price: 100.0, Type: model.PivotLow},  // WB (retracement: 100% [50/50])
				{Time: 130, Price: 170.0, Type: model.PivotHigh}, // WC (length: 70 [140% of WA])
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := elliott.MatchCorrectiveWaves(tt.pivots)
			if len(results) != tt.expectedCount {
				t.Fatalf("expected %d results, got %d", tt.expectedCount, len(results))
			}
			if tt.verify != nil {
				tt.verify(t, results)
			}
		})
	}
}

func TestCorrectiveWavePurpleBoxMatrix(t *testing.T) {
	pivots := []model.Pivot{
		{Time: 100, Price: 300.0, Type: model.PivotHigh},
		{Time: 110, Price: 200.0, Type: model.PivotLow},
		{Time: 120, Price: 240.0, Type: model.PivotHigh},
		{Time: 130, Price: 140.0, Type: model.PivotLow},
	}

	results := elliott.MatchCorrectiveWaves(pivots)
	if len(results) != 1 {
		t.Fatalf("expected 1 corrective wave, got %d", len(results))
	}

	cw := results[0]
	if cw.Type != "ZIGZAG" || cw.Direction != "BEARISH" {
		t.Fatalf("expected bearish ZIGZAG, got %s %s", cw.Direction, cw.Type)
	}

	boxes := cw.PurpleBoxes
	if len(boxes) != 2 {
		t.Fatalf("expected exactly 2 PurpleBoxes, got %d", len(boxes))
	}

	assertTargetBox(t, boxes[0], 140.0, 131, 139) // Wave C = 100% of Wave A from Wave B.
	assertTargetBox(t, boxes[1], 78.2, 131, 139)  // Wave C = 161.8% of Wave A from Wave B.
}

func BenchmarkMatchCorrectiveWaves(b *testing.B) {
	// Construct a realistic list of pivots that does not trigger matches to test scanning loop overhead
	pivots := make([]model.Pivot, 1000)
	for i := 0; i < 1000; i++ {
		t := model.PivotLow
		price := float64(i * 10)
		if i%2 == 1 {
			t = model.PivotHigh
		}
		pivots[i] = model.Pivot{
			Time:  int64(i * 100),
			Price: price,
			Type:  t,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = elliott.MatchCorrectiveWaves(pivots)
	}
}

func assertTargetBox(t *testing.T, box model.TargetBox, targetPrice float64, startTime, endTime int64) {
	t.Helper()

	expectedMin := targetPrice * 0.985
	expectedMax := targetPrice * 1.015
	if expectedMax < expectedMin {
		expectedMin, expectedMax = expectedMax, expectedMin
	}

	if math.Abs(box.MinPrice-expectedMin) > 0.0001 {
		t.Errorf("expected MinPrice %f, got %f", expectedMin, box.MinPrice)
	}
	if math.Abs(box.MaxPrice-expectedMax) > 0.0001 {
		t.Errorf("expected MaxPrice %f, got %f", expectedMax, box.MaxPrice)
	}
	if box.StartTime != startTime {
		t.Errorf("expected StartTime %d, got %d", startTime, box.StartTime)
	}
	if box.EndTime != endTime {
		t.Errorf("expected EndTime %d, got %d", endTime, box.EndTime)
	}
}

func TestMatchIncompleteWaves(t *testing.T) {
	tests := []struct {
		name          string
		pivots        []model.Pivot
		expectedCount int
		verify        func(t *testing.T, results []model.IncompleteWave)
	}{
		{
			name:          "Insufficient pivots (empty)",
			pivots:        []model.Pivot{},
			expectedCount: 0,
		},
		{
			name: "Insufficient pivots (less than 4)",
			pivots: []model.Pivot{
				{Time: 100, Price: 10.0, Type: model.PivotLow},
				{Time: 110, Price: 15.0, Type: model.PivotHigh},
				{Time: 120, Price: 12.0, Type: model.PivotLow},
			},
			expectedCount: 0,
		},
		{
			name: "Textbook Bullish 1-2-3 Structure (Incomplete)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retracement: 61.8%)
				{Time: 130, Price: 300.0, Type: model.PivotHigh}, // W3 (len: 161.8, extension: 161.8%)
			},
			expectedCount: 1,
			verify: func(t *testing.T, results []model.IncompleteWave) {
				iw := results[0]
				if iw.Direction != "BULLISH" {
					t.Errorf("expected direction BULLISH, got %q", iw.Direction)
				}
				if iw.ConfidenceScore < 0.90 {
					t.Errorf("expected high confidence score, got %f", iw.ConfidenceScore)
				}
				if iw.TargetBox == nil {
					t.Fatalf("expected TargetBox to be populated")
				}
				// 38.2% retracement of Wave 3 height (300 - 138.2 = 161.8) -> 300 - 0.382 * 161.8 = 238.1924
				expectedPrice := 300.0 - 0.382*161.8
				expectedMin := expectedPrice * 0.985
				expectedMax := expectedPrice * 1.015
				if math.Abs(iw.TargetBox.MinPrice-expectedMin) > 0.0001 {
					t.Errorf("expected MinPrice %f, got %f", expectedMin, iw.TargetBox.MinPrice)
				}
				if math.Abs(iw.TargetBox.MaxPrice-expectedMax) > 0.0001 {
					t.Errorf("expected MaxPrice %f, got %f", expectedMax, iw.TargetBox.MaxPrice)
				}
				// Time delta = 30. Start time projection: 130 + 30 * 0.382 = 141.46 (~141)
				// End time projection: 130 + 30 * 0.618 = 148.54 (~149)
				if iw.TargetBox.StartTime != 141 {
					t.Errorf("expected StartTime 141, got %d", iw.TargetBox.StartTime)
				}
				if iw.TargetBox.EndTime != 149 {
					t.Errorf("expected EndTime 149, got %d", iw.TargetBox.EndTime)
				}
			},
		},
		{
			name: "Completed 5-Wave starts are excluded from Incomplete matching",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
				{Time: 120, Price: 138.2, Type: model.PivotLow},  // W2 (retracement: 61.8%)
				{Time: 130, Price: 300.0, Type: model.PivotHigh}, // W3 (len: 161.8, extension: 161.8%)
				{Time: 140, Price: 220.0, Type: model.PivotLow},  // W4
				{Time: 150, Price: 343.6, Type: model.PivotHigh}, // W5
			},
			expectedCount: 0, // Since it matches a completed motive wave starting at Time 100, the 1-2-3 starting at 100 is excluded!
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := elliott.MatchIncompleteWaves(tt.pivots)
			if len(results) != tt.expectedCount {
				t.Fatalf("expected %d results, got %d", tt.expectedCount, len(results))
			}
			if tt.verify != nil {
				tt.verify(t, results)
			}
		})
	}
}

func BenchmarkMatchIncompleteWaves(b *testing.B) {
	pivots := make([]model.Pivot, 1000)
	for i := 0; i < 1000; i++ {
		t := model.PivotLow
		price := 100.0 + float64(i%2)*10.0
		if i%2 == 1 {
			t = model.PivotHigh
		}
		pivots[i] = model.Pivot{
			Time:  int64(i * 100),
			Price: price,
			Type:  t,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = elliott.MatchIncompleteWaves(pivots)
	}
}

// ---------------------------------------------------------------------------
// Step 9: Triangle & WXY Double Three Tests
// ---------------------------------------------------------------------------

func TestMatchTriangles(t *testing.T) {
	tests := []struct {
		name          string
		pivots        []model.Pivot
		wantTriangles int
		verify        func(t *testing.T, results []model.CorrectiveWave)
	}{
		{
			name:          "Insufficient pivots (fewer than 6)",
			pivots:        []model.Pivot{{Time: 100, Price: 100.0, Type: model.PivotLow}},
			wantTriangles: 0,
		},
		{
			// A valid bearish contracting triangle:
			// Start (High) → A (Low) → B (High) → C (Low) → D (High) → E (Low)
			// Each leg: 50 > 40 > 30 > 20 > 10 ✓
			name: "Valid Bearish Contracting Triangle (ABCDE)",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 150.0, Type: model.PivotLow},  // A (leg = 50)
				{Time: 120, Price: 190.0, Type: model.PivotHigh}, // B (leg = 40)
				{Time: 130, Price: 160.0, Type: model.PivotLow},  // C (leg = 30)
				{Time: 140, Price: 180.0, Type: model.PivotHigh}, // D (leg = 20)
				{Time: 150, Price: 170.0, Type: model.PivotLow},  // E (leg = 10)
			},
			wantTriangles: 1,
			verify: func(t *testing.T, results []model.CorrectiveWave) {
				tri := results[0]
				if tri.Type != "TRIANGLE" {
					t.Errorf("expected type TRIANGLE, got %q", tri.Type)
				}
				if tri.Direction != "BEARISH" {
					t.Errorf("expected direction BEARISH, got %q", tri.Direction)
				}
				if tri.Start.Price != 200.0 {
					t.Errorf("expected Start.Price 200.0, got %f", tri.Start.Price)
				}
				if tri.WD == nil || tri.WD.Price != 180.0 {
					t.Errorf("expected WD (D-pivot) Price 180.0, got %v", tri.WD)
				}
				if tri.WE == nil || tri.WE.Price != 170.0 {
					t.Errorf("expected WE (E-pivot) Price 170.0, got %v", tri.WE)
				}
			},
		},
		{
			// A valid bullish contracting triangle:
			// Start (Low) → A (High) → B (Low) → C (High) → D (Low) → E (High)
			// Each leg: 60 > 40 > 25 > 15 > 8 ✓
			name: "Valid Bullish Contracting Triangle (ABCDE)",
			pivots: []model.Pivot{
				{Time: 100, Price: 100.0, Type: model.PivotLow},  // Start
				{Time: 110, Price: 160.0, Type: model.PivotHigh}, // A (leg = 60)
				{Time: 120, Price: 120.0, Type: model.PivotLow},  // B (leg = 40)
				{Time: 130, Price: 145.0, Type: model.PivotHigh}, // C (leg = 25)
				{Time: 140, Price: 130.0, Type: model.PivotLow},  // D (leg = 15)
				{Time: 150, Price: 138.0, Type: model.PivotHigh}, // E (leg = 8)
			},
			wantTriangles: 1,
			verify: func(t *testing.T, results []model.CorrectiveWave) {
				tri := results[0]
				if tri.Type != "TRIANGLE" {
					t.Errorf("expected type TRIANGLE, got %q", tri.Type)
				}
				if tri.Direction != "BULLISH" {
					t.Errorf("expected direction BULLISH, got %q", tri.Direction)
				}
			},
		},
		{
			// An expanding (invalid) triangle where leg B > leg A.
			// Start (High) → A → B(long) → C → D → E
			name: "Invalid Expanding Triangle (leg B > leg A — rejected)",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 170.0, Type: model.PivotLow},  // A (leg = 30)
				{Time: 120, Price: 210.0, Type: model.PivotHigh}, // B (leg = 40 — LONGER than A)
				{Time: 130, Price: 175.0, Type: model.PivotLow},  // C (leg = 35)
				{Time: 140, Price: 195.0, Type: model.PivotHigh}, // D (leg = 20)
				{Time: 150, Price: 180.0, Type: model.PivotLow},  // E (leg = 15)
			},
			wantTriangles: 0,
		},
		{
			// Triangle where leg C = leg D (not strictly contracting) — rejected.
			name: "Invalid Triangle (leg C equals leg D — not strictly contracting)",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // Start
				{Time: 110, Price: 150.0, Type: model.PivotLow},  // A (leg = 50)
				{Time: 120, Price: 190.0, Type: model.PivotHigh}, // B (leg = 40)
				{Time: 130, Price: 160.0, Type: model.PivotLow},  // C (leg = 30)
				{Time: 140, Price: 190.0, Type: model.PivotHigh}, // D (leg = 30 — equal to C)
				{Time: 150, Price: 175.0, Type: model.PivotLow},  // E (leg = 15)
			},
			wantTriangles: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := elliott.MatchCorrectiveWaves(tt.pivots)
			// Filter only TRIANGLE results for assertion.
			var triangles []model.CorrectiveWave
			for _, r := range results {
				if r.Type == "TRIANGLE" {
					triangles = append(triangles, r)
				}
			}
			if len(triangles) != tt.wantTriangles {
				t.Fatalf("expected %d TRIANGLE(s), got %d (all results: %d)", tt.wantTriangles, len(triangles), len(results))
			}
			if tt.verify != nil {
				tt.verify(t, triangles)
			}
		})
	}
}

func TestMatchWXYDoubleThree(t *testing.T) {
	tests := []struct {
		name    string
		pivots  []model.Pivot
		wantWXY int
		verify  func(t *testing.T, results []model.CorrectiveWave)
	}{
		{
			name:    "Insufficient pivots (fewer than 8)",
			pivots:  []model.Pivot{{Time: 100, Price: 100.0, Type: model.PivotLow}},
			wantWXY: 0,
		},
		{
			// Valid Bearish WXY Double Three:
			// W = ZigZag: High(200) → Low(150) → High(170) → Low(100)
			//   WA=50, WB=20 (40% of A = ZigZag OK), WC=70 (140% of WA = ZigZag OK)
			// X = bounce from p3(Low=100) → p4(High=140): retraces 40/100 = 40% < 90% ✓
			// Y = ZigZag: High(140) → Low(80) → High(110) → Low(50)
			//   YA=60, YB=30 (50% of YA = ZigZag), YC=60 (100% of YA = ZigZag)
			// Y-end (50) < W-end (100) ✓
			name: "Valid Bearish WXY Double Three",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // p0: Start (W-start)
				{Time: 110, Price: 150.0, Type: model.PivotLow},  // p1: W-A (leg 50)
				{Time: 120, Price: 170.0, Type: model.PivotHigh}, // p2: W-B (retrace 40%=20/50, zigzag)
				{Time: 130, Price: 100.0, Type: model.PivotLow},  // p3: W-C (leg 70, 140% of WA)
				{Time: 140, Price: 140.0, Type: model.PivotHigh}, // p4: X (bounce 40/100=40% < 90%)
				{Time: 150, Price: 80.0, Type: model.PivotLow},   // p5: Y-A (leg 60)
				{Time: 160, Price: 110.0, Type: model.PivotHigh}, // p6: Y-B (retrace 50% = 30/60)
				{Time: 170, Price: 50.0, Type: model.PivotLow},   // p7: Y-C (leg 60, 100% of YA)
			},
			wantWXY: 1,
			verify: func(t *testing.T, results []model.CorrectiveWave) {
				wxy := results[0]
				if wxy.Type != "WXY" {
					t.Errorf("expected type WXY, got %q", wxy.Type)
				}
				if wxy.Direction != "BEARISH" {
					t.Errorf("expected direction BEARISH, got %q", wxy.Direction)
				}
				if wxy.WX == nil || wxy.WX.Price != 140.0 {
					t.Errorf("expected WX Price 140.0, got %v", wxy.WX)
				}
				if wxy.WE == nil || wxy.WE.Price != 50.0 {
					t.Errorf("expected WE (Y-C terminal) Price 50.0, got %v", wxy.WE)
				}
			},
		},
		{
			// Invalid: X wave retraces 95% of W (> 90%), must be rejected.
			// W: High(200) → Low(100) → High(150) → Low(80): WA=100, WB=50 (50% zig), WC=70 (70%<98%)
			// Actually need WC ≥ 100%-2% of WA to pass ZigZag (ratioC=0.70 < 0.98 — fails ZigZag)
			// And ratioB=0.50 but ratioC=0.70 < 0.98, also Flat fails (ratioB=0.50 < 0.90)
			// So W doesn't even qualify — which would make wantWXY=0 for a different reason.
			// Let's construct W as valid ZigZag then give X that retraces 95%:
			// W-ZigZag: High(200) → Low(150) → High(170) → Low(100)
			//   WA=50, WB=20 (40%), WC=70 (140%) → valid ZigZag
			// X = bounce 95% of W-amplitude (100): 100*0.95=95 → X at 100+95=195
			name: "Invalid WXY: X-wave retraces more than 90% of W (rejected)",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // p0: W-start
				{Time: 110, Price: 150.0, Type: model.PivotLow},  // p1: W-A (50)
				{Time: 120, Price: 170.0, Type: model.PivotHigh}, // p2: W-B (20=40% zigzag)
				{Time: 130, Price: 100.0, Type: model.PivotLow},  // p3: W-C (70=140%)
				{Time: 140, Price: 195.0, Type: model.PivotHigh}, // p4: X retraces 95% (195-100=95, 95/100=95%)
				{Time: 150, Price: 140.0, Type: model.PivotLow},  // p5: Y-A
				{Time: 160, Price: 167.0, Type: model.PivotHigh}, // p6: Y-B (40%)
				{Time: 170, Price: 80.0, Type: model.PivotLow},   // p7: Y-C
			},
			wantWXY: 0,
		},
		{
			// Invalid: Y-wave end is NOT lower than W-wave end (net direction fails).
			name: "Invalid WXY: Y-end not lower than W-end (net direction fails)",
			pivots: []model.Pivot{
				{Time: 100, Price: 200.0, Type: model.PivotHigh}, // p0: W-start
				{Time: 110, Price: 150.0, Type: model.PivotLow},  // p1: W-A (50)
				{Time: 120, Price: 170.0, Type: model.PivotHigh}, // p2: W-B (20=40%)
				{Time: 130, Price: 100.0, Type: model.PivotLow},  // p3: W-C (70=140%)
				{Time: 140, Price: 140.0, Type: model.PivotHigh}, // p4: X (40/100=40%)
				{Time: 150, Price: 80.0, Type: model.PivotLow},   // p5: Y-A (60)
				{Time: 160, Price: 110.0, Type: model.PivotHigh}, // p6: Y-B (30=50%)
				{Time: 170, Price: 105.0, Type: model.PivotLow},  // p7: Y-C (105 > W-end 100 — net direction FAILS)
			},
			wantWXY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := elliott.MatchCorrectiveWaves(tt.pivots)
			// Filter only WXY results.
			var wxyResults []model.CorrectiveWave
			for _, r := range results {
				if r.Type == "WXY" {
					wxyResults = append(wxyResults, r)
				}
			}
			if len(wxyResults) != tt.wantWXY {
				t.Fatalf("expected %d WXY result(s), got %d (all results: %d)", tt.wantWXY, len(wxyResults), len(results))
			}
			if tt.verify != nil {
				tt.verify(t, wxyResults)
			}
		})
	}
}

func BenchmarkMatchTrianglesAndWXY(b *testing.B) {
	// Construct a realistic pivot list that exercises both new scanners.
	// Alternating H/L pattern with gently declining amplitude to hit triangle shape.
	pivots := make([]model.Pivot, 1000)
	for i := 0; i < 1000; i++ {
		pivType := model.PivotLow
		price := 100.0 + float64(i%2)*10.0
		if i%2 == 1 {
			pivType = model.PivotHigh
		}
		pivots[i] = model.Pivot{
			Time:  int64(i * 100),
			Price: price,
			Type:  pivType,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = elliott.MatchCorrectiveWaves(pivots)
	}
}
