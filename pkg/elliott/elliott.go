package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

// MatchMotiveWaves scans a slice of pivots for valid 5-wave Elliott Wave motive structures.
// It uses a sliding-window approach over 6 consecutive pivots.
// The implementation strictly enforces the three cardinal Elliott Wave rules.
// Memory allocation inside the loop is minimized to ensure zero heap-allocation churn.
func MatchMotiveWaves(pivots []model.Pivot) []model.MotiveWave {
	n := len(pivots)
	if n < 6 {
		return nil
	}

	// Pre-allocate slice with a reasonable capacity to avoid reallocation churn.
	// 6 pivots per motive wave structure, so maximum possible is n/6.
	motiveWaves := make([]model.MotiveWave, 0, n/6)

	for i := 0; i <= n-6; i++ {
		p0 := &pivots[i]
		p1 := &pivots[i+1]
		p2 := &pivots[i+2]
		p3 := &pivots[i+3]
		p4 := &pivots[i+4]
		p5 := &pivots[i+5]

		// Check alternating pivot types and detect direction.
		// A bullish wave starts with a LOW pivot, a bearish wave starts with a HIGH pivot.
		if p0.Type == model.PivotLow {
			// BULLISH check
			if p1.Type != model.PivotHigh ||
				p2.Type != model.PivotLow ||
				p3.Type != model.PivotHigh ||
				p4.Type != model.PivotLow ||
				p5.Type != model.PivotHigh {
				continue
			}

			// Verify basic structural direction movements (up-down-up-down-up)
			if p1.Price <= p0.Price || // W1 goes up
				p2.Price >= p1.Price || // W2 goes down
				p3.Price <= p2.Price || // W3 goes up
				p4.Price >= p3.Price || // W4 goes down
				p5.Price <= p4.Price { // W5 goes up
				continue
			}

			// Cardinal Rule 1: Wave 2 cannot retrace below the start of Wave 1.
			if p2.Price < p0.Price {
				continue
			}

			// Cardinal Rule 3: Wave 4 cannot overlap into Wave 1 territory (Wave 4 low must be above Wave 1 high).
			if p4.Price <= p1.Price {
				continue
			}

			// Cardinal Rule 2: Wave 3 cannot be the shortest of waves 1, 3, and 5.
			len1 := p1.Price - p0.Price
			len3 := p3.Price - p2.Price
			len5 := p5.Price - p4.Price
			if len3 < len1 && len3 < len5 {
				continue
			}

			score := calculateConfidenceScore(p0, p1, p2, p3, p4, p5)
			if score < minConfidenceScore {
				continue
			}

			motiveWaves = append(motiveWaves, model.MotiveWave{
				Start:           p0,
				W1:              p1,
				W2:              p2,
				W3:              p3,
				W4:              p4,
				W5:              p5,
				Direction:       "BULLISH",
				ConfidenceScore: score,
				PurpleBox:       calculateTargetBox(p0, p1, p2, p3, p4, p5),
			})

		} else if p0.Type == model.PivotHigh {
			// BEARISH check
			if p1.Type != model.PivotLow ||
				p2.Type != model.PivotHigh ||
				p3.Type != model.PivotLow ||
				p4.Type != model.PivotHigh ||
				p5.Type != model.PivotLow {
				continue
			}

			// Verify basic structural direction movements (down-up-down-up-down)
			if p1.Price >= p0.Price || // W1 goes down
				p2.Price <= p1.Price || // W2 goes up
				p3.Price >= p2.Price || // W3 goes down
				p4.Price <= p3.Price || // W4 goes up
				p5.Price >= p4.Price { // W5 goes down
				continue
			}

			// Cardinal Rule 1: Wave 2 cannot retrace above the start of Wave 1.
			if p2.Price > p0.Price {
				continue
			}

			// Cardinal Rule 3: Wave 4 cannot overlap into Wave 1 territory (Wave 4 high must be below Wave 1 low).
			if p4.Price >= p1.Price {
				continue
			}

			// Cardinal Rule 2: Wave 3 cannot be the shortest of waves 1, 3, and 5.
			len1 := math.Abs(p1.Price - p0.Price)
			len3 := math.Abs(p3.Price - p2.Price)
			len5 := math.Abs(p5.Price - p4.Price)
			if len3 < len1 && len3 < len5 {
				continue
			}

			score := calculateConfidenceScore(p0, p1, p2, p3, p4, p5)
			if score < minConfidenceScore {
				continue
			}

			motiveWaves = append(motiveWaves, model.MotiveWave{
				Start:           p0,
				W1:              p1,
				W2:              p2,
				W3:              p3,
				W4:              p4,
				W5:              p5,
				Direction:       "BEARISH",
				ConfidenceScore: score,
				PurpleBox:       calculateTargetBox(p0, p1, p2, p3, p4, p5),
			})
		}
	}

	return motiveWaves
}

const (
	fibTolerance       = 0.02
	minConfidenceScore = 0.60
)

// calculateConfidenceScore calculates the Fibonacci alignment score for a 5-wave motive structure.
// The score is normalized between 0.0 and 1.0 based on the average of three checks:
// 1. Wave 2 Retracement Check: Ideal retracement of Wave 1 is 50.0%, 61.8%, or 78.6% (tolerance +/- 0.02).
// 2. Wave 3 Extension Check: Ideal extension relative to Wave 1 is 161.8% or 261.8% (or higher).
// 3. Wave 5 Target Check: Ideal length matches 100% of Wave 1, or 61.8% of the net price distance from Start to Wave 3.
func calculateConfidenceScore(p0, p1, p2, p3, p4, p5 *model.Pivot) float64 {
	// Calculate Wave absolute price lengths
	len1 := math.Abs(p1.Price - p0.Price)
	len2 := math.Abs(p1.Price - p2.Price)
	len3 := math.Abs(p3.Price - p2.Price)
	len5 := math.Abs(p5.Price - p4.Price)
	net0to3 := math.Abs(p3.Price - p0.Price)

	if len1 == 0 || net0to3 == 0 {
		return 0.0
	}

	// --- Wave 2 Retracement Check ---
	ratio2 := len2 / len1
	minD2 := math.Abs(ratio2 - 0.50)
	if d := math.Abs(ratio2 - 0.618); d < minD2 {
		minD2 = d
	}
	if d := math.Abs(ratio2 - 0.786); d < minD2 {
		minD2 = d
	}
	score2 := 0.0
	if minD2 <= fibTolerance {
		score2 = 1.0 - (minD2 / fibTolerance)
	}

	// --- Wave 3 Extension Check ---
	ratio3 := len3 / len1
	score3 := 0.0
	if ratio3 >= 2.618-fibTolerance {
		score3 = 1.0
	} else {
		d3 := math.Abs(ratio3 - 1.618)
		if d3 <= fibTolerance {
			score3 = 1.0 - (d3 / fibTolerance)
		}
	}

	// --- Wave 5 Target Check ---
	ratio5A := len5 / len1
	ratio5B := len5 / net0to3
	d5A := math.Abs(ratio5A - 1.0)
	d5B := math.Abs(ratio5B - 0.618)
	minD5 := d5A
	if d5B < minD5 {
		minD5 = d5B
	}
	score5 := 0.0
	if minD5 <= fibTolerance {
		score5 = 1.0 - (minD5 / fibTolerance)
	}

	// Average components
	return (score2 + score3 + score5) / 3.0
}

// calculateTargetBox calculates the trend channeling coordinates and Fibonacci time extensions.
func calculateTargetBox(p0, p1, p2, p3, p4, p5 *model.Pivot) *model.TargetBox {
	// Baseline (Wave 2 to Wave 4)
	x2 := float64(p2.Time)
	y2 := p2.Price
	x4 := float64(p4.Time)
	y4 := p4.Price

	if x4 == x2 {
		return nil
	}

	m := (y4 - y2) / (x4 - x2)

	// Parallel Boundary through Wave 3 (p3)
	x3 := float64(p3.Time)
	y3 := p3.Price
	b3 := y3 - m*x3

	// Midpoint of Wave 5 (starts at p4, ends at p5)
	x5 := float64(p5.Time)
	xMid := (x4 + x5) / 2.0

	// Projected price on the parallel boundary at Wave 5 midpoint
	yProj := m*xMid + b3

	// Strict +/- 1.5% buffer
	val1 := yProj * 0.985
	val2 := yProj * 1.015
	minPrice := val1
	maxPrice := val2
	if val2 < val1 {
		minPrice = val2
		maxPrice = val1
	}

	// Fibonacci Time Extensions (X-Axis)
	deltaT := float64(p3.Time - p0.Time)
	startTime := float64(p4.Time) + (deltaT * 0.382)
	endTime := float64(p4.Time) + (deltaT * 0.618)

	return &model.TargetBox{
		MinPrice:  minPrice,
		MaxPrice:  maxPrice,
		StartTime: int64(math.Round(startTime)),
		EndTime:   int64(math.Round(endTime)),
	}
}

