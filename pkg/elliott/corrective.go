package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

// MatchCorrectiveWaves scans a slice of pivots for valid 3-wave Elliott Wave corrective structures (ABC).
// It uses a sliding-window approach over 4 consecutive pivots.
// Memory allocation inside the loop is minimized to ensure zero heap-allocation churn.
func MatchCorrectiveWaves(pivots []model.Pivot) []model.CorrectiveWave {
	n := len(pivots)
	if n < 4 {
		return nil
	}

	// Pre-allocate slice with a reasonable capacity to avoid reallocation churn.
	correctiveWaves := make([]model.CorrectiveWave, 0, n/4)

	for i := 0; i <= n-4; i++ {
		p0 := &pivots[i]
		p1 := &pivots[i+1]
		p2 := &pivots[i+2]
		p3 := &pivots[i+3]

		var direction string
		var isPatternMatch bool
		var correctiveType string

		if p0.Type == model.PivotHigh {
			// BEARISH check (correcting a bullish trend, net direction down)
			if p1.Type != model.PivotLow ||
				p2.Type != model.PivotHigh ||
				p3.Type != model.PivotLow {
				continue
			}

			// Verify basic structural direction movements (down-up-down)
			if p1.Price >= p0.Price ||
				p2.Price <= p1.Price ||
				p3.Price >= p2.Price {
				continue
			}

			direction = "BEARISH"
			isPatternMatch, correctiveType = evaluateCorrectiveRules(p0, p1, p2, p3, direction)

		} else if p0.Type == model.PivotLow {
			// BULLISH check (correcting a bearish trend, net direction up)
			if p1.Type != model.PivotHigh ||
				p2.Type != model.PivotLow ||
				p3.Type != model.PivotHigh {
				continue
			}

			// Verify basic structural direction movements (up-down-up)
			if p1.Price <= p0.Price ||
				p2.Price >= p1.Price ||
				p3.Price <= p2.Price {
				continue
			}

			direction = "BULLISH"
			isPatternMatch, correctiveType = evaluateCorrectiveRules(p0, p1, p2, p3, direction)
		}

		if isPatternMatch {
			correctiveWaves = append(correctiveWaves, model.CorrectiveWave{
				Start:     p0,
				WA:        p1,
				WB:        p2,
				WC:        p3,
				Type:      correctiveType,
				Direction: direction,
			})
		}
	}

	return correctiveWaves
}

func evaluateCorrectiveRules(p0, p1, p2, p3 *model.Pivot, direction string) (bool, string) {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return false, ""
	}

	lenB := math.Abs(p2.Price - p1.Price)
	lenC := math.Abs(p3.Price - p2.Price)

	ratioB := lenB / lenA
	ratioC := lenC / lenA

	// 1. ZigZag Logic (Sharp Correction)
	// Wave B must retrace between 38.2% and 50.0% of Wave A (+/- 2% tolerance)
	if ratioB >= 0.382-0.02 && ratioB <= 0.500+0.02 {
		// Wave C must cleanly break past the extreme price level of Wave A
		cleanBreak := false
		if direction == "BEARISH" && p3.Price < p1.Price {
			cleanBreak = true
		} else if direction == "BULLISH" && p3.Price > p1.Price {
			cleanBreak = true
		}

		if cleanBreak {
			// Wave C typically reaching 100% to 161.8% of Wave A's length (+/- 2% tolerance)
			if ratioC >= 1.000-0.02 && ratioC <= 1.618+0.02 {
				return true, "ZIGZAG"
			}
		}
	}

	// 2. Flat Logic (Sideways Correction)
	// Wave B must retrace nearly the entire length of Wave A (between 90.0% and 105.0% retracement)
	if ratioB >= 0.900 && ratioB <= 1.050 {
		// Wave C must terminate very close to or just slightly past the end of Wave A (between 90.0% and 130.0% of Wave A's length)
		if ratioC >= 0.900 && ratioC <= 1.300 {
			return true, "FLAT"
		}
	}

	return false, ""
}
