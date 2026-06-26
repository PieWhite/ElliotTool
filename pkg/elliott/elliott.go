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

			len1 := p1.Price - p0.Price
			len3 := p3.Price - p2.Price
			len5 := p5.Price - p4.Price

			// Cardinal Rule 3: Wave 4 cannot overlap into Wave 1 territory (Wave 4 low must be above Wave 1 high)
			// EXCEPT in converging diagonals where W1 > W3 > W5.
			overlap := p4.Price <= p1.Price
			isDiagonal := false
			if overlap {
				if len1 > len3 && len3 > len5 {
					isDiagonal = true
				} else {
					continue
				}
			}

			// Truncation: Wave 5 fails to break above Wave 3, allowed if Wave 3 is extended.
			isTruncated := false
			if p5.Price <= p3.Price {
				if len3 > len1 {
					isTruncated = true
				} else {
					continue
				}
			}

			// Cardinal Rule 2: Wave 3 cannot be the shortest of waves 1, 3, and 5.
			if !isDiagonal {
				if len3 < len1 && len3 < len5 {
					continue
				}
			}

			score := calculateConfidenceScore(p0, p1, p2, p3, p4, p5, isDiagonal, isTruncated)
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
				PurpleBoxes:     calculateTargetBox(p0, p1, p2, p3, p4, p5),
				IsDiagonal:      isDiagonal,
				IsTruncated:     isTruncated,
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

			len1 := p0.Price - p1.Price
			len3 := p2.Price - p3.Price
			len5 := p4.Price - p5.Price

			// Cardinal Rule 3: Wave 4 cannot overlap into Wave 1 territory (Wave 4 high must be below Wave 1 low)
			// EXCEPT in converging diagonals where W1 > W3 > W5.
			overlap := p4.Price >= p1.Price
			isDiagonal := false
			if overlap {
				if len1 > len3 && len3 > len5 {
					isDiagonal = true
				} else {
					continue
				}
			}

			// Truncation: Wave 5 fails to break below Wave 3, allowed if Wave 3 is extended.
			isTruncated := false
			if p5.Price >= p3.Price {
				if len3 > len1 {
					isTruncated = true
				} else {
					continue
				}
			}

			// Cardinal Rule 2: Wave 3 cannot be the shortest of waves 1, 3, and 5.
			if !isDiagonal {
				if len3 < len1 && len3 < len5 {
					continue
				}
			}

			score := calculateConfidenceScore(p0, p1, p2, p3, p4, p5, isDiagonal, isTruncated)
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
				PurpleBoxes:     calculateTargetBox(p0, p1, p2, p3, p4, p5),
				IsDiagonal:      isDiagonal,
				IsTruncated:     isTruncated,
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
func calculateConfidenceScore(p0, p1, p2, p3, p4, p5 *model.Pivot, isDiagonal, isTruncated bool) float64 {
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
	if isDiagonal {
		minD3 := math.Abs(ratio3 - 0.618)
		if d := math.Abs(ratio3 - 0.786); d < minD3 {
			minD3 = d
		}
		if minD3 <= fibTolerance {
			score3 = 1.0 - (minD3 / fibTolerance)
		}
	} else {
		if ratio3 >= 2.618-fibTolerance {
			score3 = 1.0
		} else {
			d3 := math.Abs(ratio3 - 1.618)
			if d3 <= fibTolerance {
				score3 = 1.0 - (d3 / fibTolerance)
			}
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
	if isTruncated || isDiagonal {
		len4 := math.Abs(p4.Price - p3.Price)
		len3_abs := math.Abs(p3.Price - p2.Price)
		if len4 > 0 {
			if d := math.Abs((len5 / len4) - 0.382); d < minD5 {
				minD5 = d
			}
			if d := math.Abs((len5 / len4) - 0.618); d < minD5 {
				minD5 = d
			}
		}
		if len3_abs > 0 {
			if d := math.Abs((len5 / len3_abs) - 0.618); d < minD5 {
				minD5 = d
			}
			if d := math.Abs((len5 / len3_abs) - 0.382); d < minD5 {
				minD5 = d
			}
		}
		if d := math.Abs(ratio5A - 0.382); d < minD5 {
			minD5 = d
		}
	}
	score5 := 0.0
	if minD5 <= fibTolerance {
		score5 = 1.0 - (minD5 / fibTolerance)
	}

	// Average components
	return (score2 + score3 + score5) / 3.0
}

// calculateTargetBox calculates the three standard F&P Wave 5 Fibonacci target zones:
// 100% of Wave 1, 61.8% of net distance from Start to Wave 3, and 161.8% of Wave 1.
func calculateTargetBox(p0, p1, p2, p3, p4, p5 *model.Pivot) []model.TargetBox {
	len1 := math.Abs(p1.Price - p0.Price)
	net0to3 := math.Abs(p3.Price - p0.Price)
	if len1 == 0 || net0to3 == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p4.Time, p3.Time-p0.Time)

	return targetBoxesFromOrigin(p4.Price, direction, startTime, endTime, []float64{
		len1,
		net0to3 * 0.618,
		len1 * 1.618,
	})
}

// calculateWave3TargetBoxes projects Wave 3 target zones at 1.618x, 2.618x, and 4.236x of Wave 1.
func calculateWave3TargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	len1 := math.Abs(p1.Price - p0.Price)
	if len1 == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)

	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{
		len1 * 1.618,
		len1 * 2.618,
		len1 * 4.236,
	})
}

// calculateWaveCTargetBoxes projects Wave C target zones at 100% and 161.8% of Wave A.
func calculateWaveCTargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)

	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{
		lenA,
		lenA * 1.618,
	})
}

func waveDirection(p0, p1 *model.Pivot) string {
	if p1.Price >= p0.Price {
		return "BULLISH"
	}
	return "BEARISH"
}

func targetTimeWindow(originTime int64, deltaT int64) (int64, int64) {
	delta := float64(deltaT)
	if delta <= 0 {
		delta = 600
	}

	startTime := float64(originTime) + (delta * 0.382)
	endTime := float64(originTime) + (delta * 0.618)
	return int64(math.Round(startTime)), int64(math.Round(endTime))
}

func targetBoxesFromOrigin(originPrice float64, direction string, startTime, endTime int64, distances []float64) []model.TargetBox {
	boxes := make([]model.TargetBox, 0, len(distances))
	for _, distance := range distances {
		targetPrice := originPrice + distance
		if direction == "BEARISH" {
			targetPrice = originPrice - distance
		}

		val1 := targetPrice * 0.985
		val2 := targetPrice * 1.015
		minPrice := val1
		maxPrice := val2
		if val2 < val1 {
			minPrice = val2
			maxPrice = val1
		}

		boxes = append(boxes, model.TargetBox{
			MinPrice:  minPrice,
			MaxPrice:  maxPrice,
			StartTime: startTime,
			EndTime:   endTime,
		})
	}

	return boxes
}

// MatchIncompleteWaves scans a slice of pivots for verified developing 1-2-3 Elliott Wave structures.
// It excludes any structure that is already completed (i.e. whose Start pivot matches a completed MotiveWave).
func MatchIncompleteWaves(pivots []model.Pivot) []model.IncompleteWave {
	n := len(pivots)
	if n < 4 {
		return nil
	}

	// First, match all completed motive waves to find their start times.
	completedWaves := MatchMotiveWaves(pivots)
	// We count incomplete waves up to maximum n/4 capacity.
	incompleteWaves := make([]model.IncompleteWave, 0, n/4)

	for i := 0; i <= n-4; i++ {
		p0 := &pivots[i]

		// Check if there is already a completed motive wave starting at this pivot's time.
		// Since completedWaves is typically small, a linear scan preserves zero heap allocation
		// compared to allocating a map.
		hasCompleted := false
		for j := range completedWaves {
			if completedWaves[j].Start != nil && completedWaves[j].Start.Time == p0.Time {
				hasCompleted = true
				break
			}
		}
		if hasCompleted {
			continue
		}

		p1 := &pivots[i+1]
		p2 := &pivots[i+2]
		p3 := &pivots[i+3]

		if p0.Type == model.PivotLow {
			// BULLISH check
			if p1.Type != model.PivotHigh ||
				p2.Type != model.PivotLow ||
				p3.Type != model.PivotHigh {
				continue
			}

			// Verify direction
			if p1.Price <= p0.Price ||
				p2.Price >= p1.Price ||
				p3.Price <= p2.Price {
				continue
			}

			// Cardinal Rule 1: Wave 2 cannot retrace below the start of Wave 1.
			if p2.Price < p0.Price {
				continue
			}

			score := calculateIncompleteConfidenceScore(p0, p1, p2, p3)
			if score < minConfidenceScore {
				continue
			}

			incompleteWaves = append(incompleteWaves, model.IncompleteWave{
				Start:           p0,
				W1:              p1,
				W2:              p2,
				W3:              p3,
				Direction:       "BULLISH",
				ConfidenceScore: score,
				TargetBox:       calculateWave4TargetBox(p0, p1, p2, p3, "BULLISH"),
			})

		} else if p0.Type == model.PivotHigh {
			// BEARISH check
			if p1.Type != model.PivotLow ||
				p2.Type != model.PivotHigh ||
				p3.Type != model.PivotLow {
				continue
			}

			// Verify direction
			if p1.Price >= p0.Price ||
				p2.Price <= p1.Price ||
				p3.Price >= p2.Price {
				continue
			}

			// Cardinal Rule 1: Wave 2 cannot retrace above the start of Wave 1.
			if p2.Price > p0.Price {
				continue
			}

			score := calculateIncompleteConfidenceScore(p0, p1, p2, p3)
			if score < minConfidenceScore {
				continue
			}

			incompleteWaves = append(incompleteWaves, model.IncompleteWave{
				Start:           p0,
				W1:              p1,
				W2:              p2,
				W3:              p3,
				Direction:       "BEARISH",
				ConfidenceScore: score,
				TargetBox:       calculateWave4TargetBox(p0, p1, p2, p3, "BEARISH"),
			})
		}
	}

	return incompleteWaves
}

// calculateIncompleteConfidenceScore calculates the Fibonacci alignment score for a 1-2-3 structure.
func calculateIncompleteConfidenceScore(p0, p1, p2, p3 *model.Pivot) float64 {
	len1 := math.Abs(p1.Price - p0.Price)
	len2 := math.Abs(p1.Price - p2.Price)
	len3 := math.Abs(p3.Price - p2.Price)

	if len1 == 0 {
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

	// Average components
	return (score2 + score3) / 2.0
}

// calculateWave4TargetBox calculates predictive coordinates for the upcoming Wave 4 based on 38.2% retracement of Wave 3.
func calculateWave4TargetBox(p0, p1, p2, p3 *model.Pivot, direction string) *model.TargetBox {
	len3 := math.Abs(p3.Price - p2.Price)
	var targetPrice float64
	if direction == "BULLISH" {
		targetPrice = p3.Price - 0.382*len3
	} else {
		targetPrice = p3.Price + 0.382*len3
	}

	val1 := targetPrice * 0.985
	val2 := targetPrice * 1.015
	minPrice := val1
	maxPrice := val2
	if val2 < val1 {
		minPrice = val2
		maxPrice = val1
	}

	deltaT := float64(p3.Time - p0.Time)
	if deltaT <= 0 {
		deltaT = 600
	}
	startTime := float64(p3.Time) + (deltaT * 0.382)
	endTime := float64(p3.Time) + (deltaT * 0.618)

	return &model.TargetBox{
		MinPrice:  minPrice,
		MaxPrice:  maxPrice,
		StartTime: int64(math.Round(startTime)),
		EndTime:   int64(math.Round(endTime)),
	}
}
