package swing

import (
	"testing"

	"WaveSight/pkg/model"
)

func TestCalculateATR(t *testing.T) {
	candles := []model.Candle{
		{Time: 1, Open: 10, High: 15, Low: 5, Close: 12},  // TR = 10
		{Time: 2, Open: 12, High: 18, Low: 10, Close: 16}, // TR = max(8, |18-12|, |10-12|) = 8
		{Time: 3, Open: 16, High: 20, Low: 14, Close: 18}, // TR = max(6, |20-16|, |14-16|) = 6
	}

	atr := CalculateATR(candles, 2)
	if len(atr) != len(candles) {
		t.Fatalf("expected ATR slice length %d, got %d", len(candles), len(atr))
	}

	// ATR at index 0: sum(TR[0]) / 1 = 10
	if atr[0] != 10.0 {
		t.Errorf("expected atr[0] to be 10.0, got %f", atr[0])
	}
	// ATR at index 1: sum(TR[0..1]) / 2 = (10 + 8) / 2 = 9.0
	if atr[1] != 9.0 {
		t.Errorf("expected atr[1] to be 9.0, got %f", atr[1])
	}
	// ATR at index 2 (window of 2): sum(TR[1..2]) / 2 = (8 + 6) / 2 = 7.0
	if atr[2] != 7.0 {
		t.Errorf("expected atr[2] to be 7.0, got %f", atr[2])
	}
}

func TestVolatilitySwingDetector_DetectSwings(t *testing.T) {
	// Let's create candles forming a clear upward swing, then downward swing
	// ATR is around 10. Multiplier is 1.5 -> threshold is 15.
	candles := []model.Candle{
		{Time: 100, Open: 100, High: 100, Low: 100, Close: 100}, // Start trend up. ATR starts.
		{Time: 101, Open: 110, High: 110, Low: 110, Close: 110},
		{Time: 102, Open: 120, High: 120, Low: 120, Close: 120},
		{Time: 103, Open: 130, High: 130, Low: 130, Close: 130},
		{Time: 104, Open: 140, High: 140, Low: 140, Close: 140}, // High peak: 140
		{Time: 105, Open: 120, High: 120, Low: 120, Close: 120}, // Reversal check: drop of 20 (from 140 to 120).
		// Reversal threshold is around 1.5 * 10 = 15. Drop is 20, which is > 15 -> confirms peak at 140 (T104) and reverses.
		{Time: 106, Open: 110, High: 110, Low: 110, Close: 110},
		{Time: 107, Open: 100, High: 100, Low: 100, Close: 100}, // Low peak: 100
		{Time: 108, Open: 125, High: 125, Low: 125, Close: 125}, // Reversal check: rise of 25 (from 100 to 125) -> confirms trough at 100 (T107).
	}

	detector := NewVolatilitySwingDetector(5)
	pivots := detector.DetectSwings(candles, 1.5)

	// We expect PivotHigh at Time 104, Price 140
	// We expect PivotLow at Time 107, Price 100
	// The right edge stabilizer should append the final extreme (T108, 125, PivotHigh)
	if len(pivots) < 3 {
		t.Fatalf("expected at least 3 pivots, got %d", len(pivots))
	}

	p1 := pivots[0]
	if p1.Time != 104 || p1.Price != 140 || p1.Type != model.PivotHigh {
		t.Errorf("expected first pivot to be High at 104 with price 140, got %+v", p1)
	}

	p2 := pivots[1]
	if p2.Time != 107 || p2.Price != 100 || p2.Type != model.PivotLow {
		t.Errorf("expected second pivot to be Low at 107 with price 100, got %+v", p2)
	}

	p3 := pivots[2]
	if p3.Time != 108 || p3.Price != 125 || p3.Type != model.PivotHigh {
		t.Errorf("expected third pivot to be High at 108 with price 125, got %+v", p3)
	}
}

func TestVolatilitySwingDetector_EmptyOrSingle(t *testing.T) {
	detector := NewVolatilitySwingDetector(5)

	if actual := detector.DetectSwings(nil, 1.5); actual != nil {
		t.Errorf("expected nil for empty candles, got %v", actual)
	}

	single := []model.Candle{{Time: 100, Open: 100, High: 100, Low: 100, Close: 100}}
	if actual := detector.DetectSwings(single, 1.5); len(actual) != 0 {
		t.Errorf("expected empty slice for single candle, got %v", actual)
	}
}

func TestVolatilitySwingDetector_NoPivotsDefault(t *testing.T) {
	// Set of candles with no movements exceeding threshold -> should return default start/end
	candles := []model.Candle{
		{Time: 100, Open: 100, High: 101, Low: 99, Close: 100},
		{Time: 101, Open: 100, High: 101, Low: 99, Close: 100},
	}
	detector := NewVolatilitySwingDetector(5)
	pivots := detector.DetectSwings(candles, 10.0) // Extremely high threshold

	if len(pivots) != 2 {
		t.Fatalf("expected 2 default pivots, got %d", len(pivots))
	}

	if pivots[0].Time != 100 || pivots[0].Type != model.PivotLow {
		t.Errorf("expected start candle default pivot to be Low, got %+v", pivots[0])
	}
	if pivots[1].Time != 101 || pivots[1].Type != model.PivotHigh {
		t.Errorf("expected end candle default pivot to be High, got %+v", pivots[1])
	}
}

func TestVolatilitySwingDetector_OverlapTimes(t *testing.T) {
	candles := []model.Candle{
		{Time: 100, Open: 100, High: 100, Low: 100, Close: 100},
		{Time: 100, Open: 120, High: 120, Low: 120, Close: 120},
		{Time: 100, Open: 90, High: 90, Low: 90, Close: 90},
		{Time: 101, Open: 110, High: 110, Low: 110, Close: 110},
	}

	detector := NewVolatilitySwingDetector(5)
	pivots := detector.DetectSwings(candles, 1.0)

	// Ensure times are strictly increasing
	for i := 1; i < len(pivots); i++ {
		if pivots[i].Time <= pivots[i-1].Time {
			t.Errorf("timestamps not strictly increasing: pivots[%d].Time=%d <= pivots[%d].Time=%d", i, pivots[i].Time, i-1, pivots[i-1].Time)
		}
	}
}
