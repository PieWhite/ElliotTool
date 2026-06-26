package zigzag

import (
	"reflect"
	"testing"

	"WaveSight/pkg/model"
)

func TestCalculateZigZag(t *testing.T) {
	tests := []struct {
		name      string
		candles   []model.Candle
		deviation float64
		expected  []model.Pivot
	}{
		{
			name:      "Empty candles list",
			candles:   []model.Candle{},
			deviation: 5.0,
			expected:  nil,
		},
		{
			name: "Single candle list",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 105, Low: 95, Close: 100, Volume: 1000},
			},
			deviation: 5.0,
			expected:  []model.Pivot{},
		},
		{
			name: "Happy path (alternating trend)",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: 1001, Open: 110, High: 110, Low: 110, Close: 110, Volume: 1000}, // up 10%
				{Time: 1002, Open: 99, High: 99, Low: 99, Close: 99, Volume: 1000},    // down 10%
				{Time: 1003, Open: 108.9, High: 108.9, Low: 108.9, Close: 108.9, Volume: 1000}, // up 10%
			},
			deviation: 5.0,
			expected: []model.Pivot{
				{Time: 1000, Price: 100, Type: model.PivotLow},
				{Time: 1001, Price: 110, Type: model.PivotHigh},
				{Time: 1002, Price: 99, Type: model.PivotLow},
				{Time: 1003, Price: 108.9, Type: model.PivotHigh},
			},
		},
		{
			name: "Strictly monotonic uptrend",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: 1001, Open: 105, High: 105, Low: 105, Close: 105, Volume: 1000},
				{Time: 1002, Open: 110, High: 110, Low: 110, Close: 110, Volume: 1000},
				{Time: 1003, Open: 115, High: 115, Low: 115, Close: 115, Volume: 1000},
			},
			deviation: 5.0,
			expected: []model.Pivot{
				{Time: 1000, Price: 100, Type: model.PivotLow},
				{Time: 1003, Price: 115, Type: model.PivotHigh},
			},
		},
		{
			name: "Strictly monotonic downtrend",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: 1001, Open: 95, High: 95, Low: 95, Close: 95, Volume: 1000},
				{Time: 1002, Open: 90, High: 90, Low: 90, Close: 90, Volume: 1000},
				{Time: 1003, Open: 85, High: 85, Low: 85, Close: 85, Volume: 1000},
			},
			deviation: 5.0,
			expected: []model.Pivot{
				{Time: 1000, Price: 100, Type: model.PivotHigh},
				{Time: 1003, Price: 85, Type: model.PivotLow},
			},
		},
		{
			name: "High market noise/chop (ignored)",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: 1001, Open: 102, High: 102, Low: 102, Close: 102, Volume: 1000},
				{Time: 1002, Open: 99, High: 99, Low: 99, Close: 99, Volume: 1000},
				{Time: 1003, Open: 101, High: 101, Low: 101, Close: 101, Volume: 1000},
			},
			deviation: 5.0,
			expected:  []model.Pivot{},
		},
		{
			name: "Single candle range exceeding deviation",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 110, Low: 100, Close: 105, Volume: 1000},
				{Time: 1001, Open: 105, High: 105, Low: 105, Close: 105, Volume: 1000},
			},
			deviation: 5.0,
			expected: []model.Pivot{
				{Time: 1000, Price: 100, Type: model.PivotLow},
				{Time: 1000, Price: 110, Type: model.PivotHigh},
			},
		},
		{
			name: "Multiple shifts with noise in between",
			candles: []model.Candle{
				{Time: 1000, Open: 100, High: 100, Low: 100, Close: 100, Volume: 1000},
				{Time: 1001, Open: 106, High: 106, Low: 106, Close: 106, Volume: 1000}, // Up 6% (triggers Low 100, state -> SearchingHigh)
				{Time: 1002, Open: 103, High: 103, Low: 103, Close: 103, Volume: 1000}, // Down 2.8% (noise, ignored)
				{Time: 1003, Open: 108, High: 108, Low: 108, Close: 108, Volume: 1000}, // Up to 108 (state -> SearchingHigh, updates extremeMax to 108)
				{Time: 1004, Open: 101, High: 101, Low: 101, Close: 101, Volume: 1000}, // Down 6.48% (triggers High 108, state -> SearchingLow)
			},
			deviation: 5.0,
			expected: []model.Pivot{
				{Time: 1000, Price: 100, Type: model.PivotLow},
				{Time: 1003, Price: 108, Type: model.PivotHigh},
				{Time: 1004, Price: 101, Type: model.PivotLow},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := CalculateZigZag(tt.candles, tt.deviation)
			if len(actual) == 0 && len(tt.expected) == 0 {
				// both empty, pass
				return
			}
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("expected %+v, got %+v", tt.expected, actual)
			}
		})
	}
}
