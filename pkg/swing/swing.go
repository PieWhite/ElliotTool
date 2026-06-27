package swing

import (
	"math"

	"WaveSight/pkg/model"
)

// SwingDetector defines the interface for identifying candidate pivots from candle data.
type SwingDetector interface {
	DetectSwings(candles []model.Candle, multiplier float64) []model.Pivot
}

// VolatilitySwingDetector implements SwingDetector using Average True Range (ATR)
// to dynamically filter out minor wiggles based on current market volatility.
type VolatilitySwingDetector struct {
	Period int
}

// NewVolatilitySwingDetector creates a new VolatilitySwingDetector.
func NewVolatilitySwingDetector(period int) *VolatilitySwingDetector {
	return &VolatilitySwingDetector{
		Period: period,
	}
}

// DetectSwings computes pivots using a dynamic, ATR-based trailing threshold.
func (d *VolatilitySwingDetector) DetectSwings(candles []model.Candle, multiplier float64) []model.Pivot {
	n := len(candles)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []model.Pivot{}
	}

	// Default multiplier fallback if invalid
	if multiplier <= 0 {
		multiplier = 1.5
	}

	atr := CalculateATR(candles, d.Period)

	pivots := make([]model.Pivot, 0)
	
	type TrendState int
	const (
		stateNeutral TrendState = iota
		stateUp
		stateDown
	)
	state := stateNeutral

	// Track the current extremum since the last confirmed pivot change
	var extremePrice float64
	var extremeTime int64

	// Helper to add pivot and handle duplicate timestamp collisions
	addPivot := func(t int64, price float64, pType model.PivotType) {
		// Avoid duplicates or out-of-order timestamps
		if len(pivots) > 0 && pivots[len(pivots)-1].Time >= t {
			t = pivots[len(pivots)-1].Time + 1
		}
		pivots = append(pivots, model.Pivot{
			Time:  t,
			Price: price,
			Type:  pType,
		})
	}

	for i := 0; i < n; i++ {
		c := candles[i]
		currentATR := atr[i]
		if currentATR <= 0 {
			// fallback if ATR is zero (e.g. flat price)
			currentATR = c.Close * 0.01 // 1% fallback
			if currentATR <= 0 {
				currentATR = 0.01
			}
		}
		threshold := currentATR * multiplier

		switch state {
		case stateNeutral:
			// Initialize state based on the first candle
			extremePrice = c.High
			extremeTime = c.Time
			state = stateUp

		case stateUp:
			// In an uptrend, we look for a new high or a reversal down
			if c.High > extremePrice {
				extremePrice = c.High
				extremeTime = c.Time
			}
			// Reversal condition: price drops below the extreme high minus threshold
			if c.Low < extremePrice-threshold {
				// Confirm the extreme high as a PivotHigh
				addPivot(extremeTime, extremePrice, model.PivotHigh)
				// Switch to downtrend, set initial extreme low to current candle's low
				state = stateDown
				extremePrice = c.Low
				extremeTime = c.Time
			}

		case stateDown:
			// In a downtrend, we look for a new low or a reversal up
			if c.Low < extremePrice {
				extremePrice = c.Low
				extremeTime = c.Time
			}
			// Reversal condition: price rises above the extreme low plus threshold
			if c.High > extremePrice+threshold {
				// Confirm the extreme low as a PivotLow
				addPivot(extremeTime, extremePrice, model.PivotLow)
				// Switch to uptrend, set initial extreme high to current candle's high
				state = stateUp
				extremePrice = c.High
				extremeTime = c.Time
			}
		}
	}

	// --- RIGHT EDGE STABILIZER: Append the final extreme point as a pivot ---
	if len(pivots) > 0 {
		lastPivot := pivots[len(pivots)-1]
		if lastPivot.Type == model.PivotLow && extremeTime > lastPivot.Time {
			addPivot(extremeTime, extremePrice, model.PivotHigh)
		} else if lastPivot.Type == model.PivotHigh && extremeTime > lastPivot.Time {
			addPivot(extremeTime, extremePrice, model.PivotLow)
		}
	} else {
		// If no pivots were found at all, add the start and end of the candles
		addPivot(candles[0].Time, candles[0].Low, model.PivotLow)
		addPivot(candles[n-1].Time, candles[n-1].High, model.PivotHigh)
	}

	return pivots
}

// CalculateATR computes the rolling Average True Range over a given window period.
func CalculateATR(candles []model.Candle, period int) []float64 {
	n := len(candles)
	if n == 0 {
		return nil
	}
	tr := make([]float64, n)
	atr := make([]float64, n)

	// Period must be at least 1, default to 14 if <= 0
	if period <= 0 {
		period = 14
	}

	for i := 0; i < n; i++ {
		if i == 0 {
			tr[i] = candles[i].High - candles[i].Low
		} else {
			hL := candles[i].High - candles[i].Low
			hC := math.Abs(candles[i].High - candles[i-1].Close)
			lC := math.Abs(candles[i].Low - candles[i-1].Close)
			tr[i] = math.Max(hL, math.Max(hC, lC))
		}
	}

	// Calculate SMA of True Range for ATR
	var sum float64
	for i := 0; i < n; i++ {
		sum += tr[i]
		if i < period {
			atr[i] = sum / float64(i+1)
		} else {
			sum -= tr[i-period]
			atr[i] = sum / float64(period)
		}
	}
	return atr
}
