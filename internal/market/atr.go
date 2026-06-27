package market

import "math"

func ATR(candles []Candle, period int) []float64 {
	if len(candles) == 0 {
		return nil
	}
	if period < 1 {
		period = 14
	}

	result := make([]float64, len(candles))
	trueRanges := make([]float64, len(candles))
	var rolling float64
	for i, candle := range candles {
		tr := candle.High - candle.Low
		if i > 0 {
			tr = math.Max(tr, math.Abs(candle.High-candles[i-1].Close))
			tr = math.Max(tr, math.Abs(candle.Low-candles[i-1].Close))
		}
		trueRanges[i] = tr
		rolling += tr
		if i >= period {
			rolling -= trueRanges[i-period]
			result[i] = rolling / float64(period)
		} else {
			result[i] = rolling / float64(i+1)
		}
	}
	return result
}
