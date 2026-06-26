package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

// MatchCorrectiveWaves scans a slice of pivots for valid Elliott Wave corrective structures.
// It returns all detected ZIGZAG, FLAT, TRIANGLE, and WXY patterns.
// Memory allocation inside the loops is minimized to ensure zero heap-allocation churn.
func MatchCorrectiveWaves(pivots []model.Pivot) []model.CorrectiveWave {
	n := len(pivots)
	if n < 4 {
		return nil
	}

	// Pre-allocate slice with a reasonable capacity to avoid reallocation churn.
	correctiveWaves := make([]model.CorrectiveWave, 0, n/4)

	// --- 3-wave ABC structures: ZigZag & Flat (window of 4 pivots) ---
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

	// --- Triangle ABCDE structures (window of 6 pivots: Start + A + B + C + D + E) ---
	if n >= 6 {
		for i := 0; i <= n-6; i++ {
			p0 := &pivots[i]
			p1 := &pivots[i+1]
			p2 := &pivots[i+2]
			p3 := &pivots[i+3]
			p4 := &pivots[i+4]
			p5 := &pivots[i+5]

			tri, ok := evaluateTriangle(p0, p1, p2, p3, p4, p5)
			if ok {
				correctiveWaves = append(correctiveWaves, tri)
			}
		}
	}

	// --- WXY Double Three structures (window of 8 pivots: Start + W(3) + X(1) + Y(3)) ---
	if n >= 8 {
		for i := 0; i <= n-8; i++ {
			p0 := &pivots[i]
			p1 := &pivots[i+1]
			p2 := &pivots[i+2]
			p3 := &pivots[i+3]
			p4 := &pivots[i+4]
			p5 := &pivots[i+5]
			p6 := &pivots[i+6]
			p7 := &pivots[i+7]

			wxy, ok := evaluateWXY(p0, p1, p2, p3, p4, p5, p6, p7)
			if ok {
				correctiveWaves = append(correctiveWaves, wxy)
			}
		}
	}

	return correctiveWaves
}

// evaluateCorrectiveRules classifies a 4-pivot (Start+ABC) corrective structure as ZIGZAG or FLAT.
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

// evaluateTriangle checks whether 6 consecutive pivots form a contracting Elliott Wave Triangle (ABCDE).
//
// Structure: p0 (Start) → p1 (A) → p2 (B) → p3 (C) → p4 (D) → p5 (E)
// The Start pivot is the origin before WA begins.
// Contracting rule: |AB| > |BC| > |CD| > |DE| (each successive leg is strictly shorter).
// Direction:
//   - BULLISH triangle: Start is a PivotLow, price oscillates sideways upward.
//   - BEARISH triangle: Start is a PivotHigh, price oscillates sideways downward.
func evaluateTriangle(p0, p1, p2, p3, p4, p5 *model.Pivot) (model.CorrectiveWave, bool) {
	// The five ABCDE pivots are p1..p5. p0 is the structural origin (Start).
	// Determine direction from the starting pivot type.
	var direction string
	if p0.Type == model.PivotLow {
		// BULLISH triangle: Start is Low, alternating H-L-H-L-H
		if p1.Type != model.PivotHigh ||
			p2.Type != model.PivotLow ||
			p3.Type != model.PivotHigh ||
			p4.Type != model.PivotLow ||
			p5.Type != model.PivotHigh {
			return model.CorrectiveWave{}, false
		}
		direction = "BULLISH"
	} else if p0.Type == model.PivotHigh {
		// BEARISH triangle: Start is High, alternating L-H-L-H-L
		if p1.Type != model.PivotLow ||
			p2.Type != model.PivotHigh ||
			p3.Type != model.PivotLow ||
			p4.Type != model.PivotHigh ||
			p5.Type != model.PivotLow {
			return model.CorrectiveWave{}, false
		}
		direction = "BEARISH"
	} else {
		return model.CorrectiveWave{}, false
	}

	// Calculate the absolute price lengths of each ABCDE leg.
	// Leg A: p0→p1, Leg B: p1→p2, Leg C: p2→p3, Leg D: p3→p4, Leg E: p4→p5
	legA := math.Abs(p1.Price - p0.Price)
	legB := math.Abs(p2.Price - p1.Price)
	legC := math.Abs(p3.Price - p2.Price)
	legD := math.Abs(p4.Price - p3.Price)
	legE := math.Abs(p5.Price - p4.Price)

	if legA == 0 {
		return model.CorrectiveWave{}, false
	}

	// Contracting constraint: each leg strictly shorter than the previous.
	if !(legA > legB && legB > legC && legC > legD && legD > legE) {
		return model.CorrectiveWave{}, false
	}

	return model.CorrectiveWave{
		Start:     p0,
		WA:        p1,
		WB:        p2,
		WC:        p3,
		WD:        p4,
		WE:        p5,
		Type:      "TRIANGLE",
		Direction: direction,
	}, true
}

// evaluateWXY checks whether 8 consecutive pivots form an Elliott Wave Double Three (WXY).
//
// Structure: p0 (Start) → p1..p3 (first corrective W: 3 legs) → p4 (X connector) → p5..p7 (second corrective Y: 3 legs)
// Window: 8 pivots = 1 (Start) + 3 (W-legs: A,B,C) + 1 (X-bounce) + 3 (Y-legs: A,B,C)
//
// Rules:
//  1. The W component (p0..p3) must pass evaluateCorrectiveRules as ZIGZAG or FLAT.
//  2. The X-wave (p3→p4) must retrace less than 90% of the W leg amplitude to avoid invalid reversals.
//  3. The Y component (p4..p7) must pass evaluateCorrectiveRules as ZIGZAG or FLAT.
//  4. Overall net direction: the structure must move net in the direction of W (bearish or bullish).
func evaluateWXY(p0, p1, p2, p3, p4, p5, p6, p7 *model.Pivot) (model.CorrectiveWave, bool) {
	// Determine direction from starting pivot type.
	var direction string
	if p0.Type == model.PivotHigh {
		direction = "BEARISH"
	} else if p0.Type == model.PivotLow {
		direction = "BULLISH"
	} else {
		return model.CorrectiveWave{}, false
	}

	// 1. W-wave: validate p0..p3 as a corrective structure.
	//    Basic direction check for W.
	wMatch, _ := evaluateCorrectiveRules(p0, p1, p2, p3, direction)
	if !wMatch {
		return model.CorrectiveWave{}, false
	}

	// 2. X-wave: p3 → p4 (single pivot bounce in the opposite direction of W).
	//    X must not retrace more than 90% of the W-wave amplitude (p0→p3).
	wAmplitude := math.Abs(p3.Price - p0.Price)
	if wAmplitude == 0 {
		return model.CorrectiveWave{}, false
	}
	xLen := math.Abs(p4.Price - p3.Price)
	xRetrace := xLen / wAmplitude
	if xRetrace > 0.90 {
		return model.CorrectiveWave{}, false
	}

	// X pivot must bounce opposite to W direction.
	if direction == "BEARISH" {
		// W moved down (p0 High → p3 Low). X must bounce up from p3.
		if p4.Type != model.PivotHigh || p4.Price <= p3.Price {
			return model.CorrectiveWave{}, false
		}
	} else {
		// W moved up (p0 Low → p3 High). X must bounce down from p3.
		if p4.Type != model.PivotLow || p4.Price >= p3.Price {
			return model.CorrectiveWave{}, false
		}
	}

	// 3. Y-wave: validate p4..p7 as a corrective structure in the same direction as W.
	yMatch, _ := evaluateCorrectiveRules(p4, p5, p6, p7, direction)
	if !yMatch {
		return model.CorrectiveWave{}, false
	}

	// 4. Net direction check: Y must end further than W's end in the direction of the move.
	if direction == "BEARISH" {
		// Both W and Y are bearish; p7 must be lower than p3 (Y's end is lower than W's end).
		if p7.Price >= p3.Price {
			return model.CorrectiveWave{}, false
		}
	} else {
		// Both W and Y are bullish; p7 must be higher than p3.
		if p7.Price <= p3.Price {
			return model.CorrectiveWave{}, false
		}
	}

	return model.CorrectiveWave{
		Start:     p0,
		WA:        p1, // W-wave's A pivot
		WB:        p2, // W-wave's B pivot
		WC:        p3, // W-wave's C pivot (end of W-wave)
		WX:        p4, // X-wave connector pivot
		WD:        p5, // Y-wave's A pivot
		WE:        p7, // Y-wave's C pivot (terminal endpoint of the Double Three)
		Type:      "WXY",
		Direction: direction,
	}, true
}
