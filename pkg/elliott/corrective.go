package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

const (
	fibToleranceCorrective = 0.03
	maxLookaheadCorrective = 16
)

// MatchCorrectiveWaves scans a slice of pivots for valid Elliott Wave corrective structures.
// It returns all detected ZIGZAG, FLAT, TRIANGLE, and WXY patterns using non-consecutive lookahead windows.
func MatchCorrectiveWaves(pivots []model.Pivot) []model.CorrectiveWave {
	n := len(pivots)
	if n < 4 {
		return nil
	}

	correctiveWaves := make([]model.CorrectiveWave, 0, n/4)

	for i := 0; i < n; i++ {
		p0 := &pivots[i]
		endWindow := i + maxLookaheadCorrective
		if endWindow > n {
			endWindow = n
		}

		// --- 1. 3-wave ABC structures: ZigZag & Flat (Lookahead engine) ---
		for i1 := i + 1; i1 < endWindow; i1++ {
			p1 := &pivots[i1]
			for i2 := i1 + 1; i2 < endWindow; i2++ {
				p2 := &pivots[i2]
				for i3 := i2 + 1; i3 < endWindow; i3++ {
					p3 := &pivots[i3]

					var direction string
					var isPatternMatch bool
					var correctiveType string

					if p0.Type == model.PivotHigh {
						// BEARISH check (correcting a bullish trend)
						if p1.Type != model.PivotLow || p2.Type != model.PivotHigh || p3.Type != model.PivotLow {
							continue
						}
						if p1.Price >= p0.Price || p2.Price <= p1.Price || p3.Price >= p2.Price {
							continue
						}
						direction = "BEARISH"
						isPatternMatch, correctiveType = evaluateCorrectiveRules(p0, p1, p2, p3, direction)

					} else if p0.Type == model.PivotLow {
						// BULLISH check (correcting a bearish trend)
						if p1.Type != model.PivotHigh || p2.Type != model.PivotLow || p3.Type != model.PivotHigh {
							continue
						}
						if p1.Price <= p0.Price || p2.Price >= p1.Price || p3.Price <= p2.Price {
							continue
						}
						direction = "BULLISH"
						isPatternMatch, correctiveType = evaluateCorrectiveRules(p0, p1, p2, p3, direction)
					}

					if isPatternMatch {
						correctiveWaves = append(correctiveWaves, model.CorrectiveWave{
							Start:       p0,
							WA:          p1,
							WB:          p2,
							WC:          p3,
							Type:        correctiveType,
							Direction:   direction,
							PurpleBoxes: calculateWaveCTargetBoxes(p0, p1, p2, p3),
						})
					}
				}
			}
		}

		// --- 2. Triangle ABCDE structures (Delineated with lookahead window) ---
		if i+5 < n {
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						for i4 := i3 + 1; i4 < endWindow; i4++ {
							p4 := &pivots[i4]
							for i5 := i4 + 1; i5 < endWindow; i5++ {
								p5 := &pivots[i5]

								tri, ok := evaluateTriangle(p0, p1, p2, p3, p4, p5)
								if ok {
									correctiveWaves = append(correctiveWaves, tri)
								}
							}
						}
					}
				}
			}
		}

		// --- 3. WXY Double Three structures (Delineated with lookahead window) ---
		if i+7 < n {
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						for i4 := i3 + 1; i4 < endWindow; i4++ {
							p4 := &pivots[i4]
							for i5 := i4 + 1; i5 < endWindow; i5++ {
								p5 := &pivots[i5]
								for i6 := i5 + 1; i6 < endWindow; i6++ {
									p6 := &pivots[i6]
									for i7 := i6 + 1; i7 < endWindow; i7++ {
										p7 := &pivots[i7]

										wxy, ok := evaluateWXY(p0, p1, p2, p3, p4, p5, p6, p7)
										if ok {
											correctiveWaves = append(correctiveWaves, wxy)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return correctiveWaves
}

// evaluateCorrectiveRules classifies a 4-pivot structure supporting Expanded/Running variations.
func evaluateCorrectiveRules(p0, p1, p2, p3 *model.Pivot, direction string) (bool, string) {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return false, ""
	}

	lenB := math.Abs(p2.Price - p1.Price)
	lenC := math.Abs(p3.Price - p2.Price)

	ratioB := lenB / lenA
	ratioC := lenC / lenA

	// 1. Corrected ZigZag Logic (Sharp Correction)
	// Wave B retraces between 38.2% and 61.8% of Wave A (Conform Frost & Prechter)
	if ratioB >= 0.382-fibToleranceCorrective && ratioB <= 0.618+fibToleranceCorrective {
		cleanBreak := false
		if direction == "BEARISH" && p3.Price < p1.Price {
			cleanBreak = true
		} else if direction == "BULLISH" && p3.Price > p1.Price {
			cleanBreak = true
		}

		if cleanBreak {
			// Wave C targets conform handboek: 0.618, 1.000 of 1.618 van Wave A
			if math.Abs(ratioC-0.618) <= fibToleranceCorrective ||
				math.Abs(ratioC-1.000) <= fibToleranceCorrective ||
				math.Abs(ratioC-1.618) <= fibToleranceCorrective ||
				(ratioC >= 1.000 && ratioC <= 1.618) {
				return true, "ZIGZAG"
			}
		}
	}

	// 2. Corrected Flat Logic (Sideways Correction - Supporting Expanded Flats)
	// Regular & Expanded Flats range from 90% up to 138.2% retracement for Wave B
	if ratioB >= 0.900-fibToleranceCorrective && ratioB <= 1.382+fibToleranceCorrective {
		if ratioB > 1.05 {
			// Expanded Flat: Wave C moves substantially past Wave A (1.236 or 1.618)
			if ratioC >= 1.000 || math.Abs(ratioC-1.236) <= fibToleranceCorrective || math.Abs(ratioC-1.618) <= fibToleranceCorrective {
				return true, "FLAT"
			}
		} else {
			// Regular Flat: Wave C terminates near the end of Wave A (90% to 130%)
			if ratioC >= 0.900-fibToleranceCorrective && ratioC <= 1.300+fibToleranceCorrective {
				return true, "FLAT"
			}
		}
	}

	return false, ""
}

// evaluateTriangle checks for a contracting Elliott Wave Triangle aligned with Fibonacci proportions.
func evaluateTriangle(p0, p1, p2, p3, p4, p5 *model.Pivot) (model.CorrectiveWave, bool) {
	var direction string
	if p0.Type == model.PivotLow {
		if p1.Type != model.PivotHigh || p2.Type != model.PivotLow || p3.Type != model.PivotHigh || p4.Type != model.PivotLow || p5.Type != model.PivotHigh {
			return model.CorrectiveWave{}, false
		}
		direction = "BULLISH"
	} else if p0.Type == model.PivotHigh {
		if p1.Type != model.PivotLow || p2.Type != model.PivotHigh || p3.Type != model.PivotLow || p4.Type != model.PivotHigh || p5.Type != model.PivotLow {
			return model.CorrectiveWave{}, false
		}
		direction = "BEARISH"
	} else {
		return model.CorrectiveWave{}, false
	}

	legA := math.Abs(p1.Price - p0.Price)
	legB := math.Abs(p2.Price - p1.Price)
	legC := math.Abs(p3.Price - p2.Price)
	legD := math.Abs(p4.Price - p3.Price)
	legE := math.Abs(p5.Price - p4.Price)

	if legA == 0 || legB == 0 || legC == 0 || legD == 0 {
		return model.CorrectiveWave{}, false
	}

	// Structural boundary constraint
	if !(legA > legB && legB > legC && legC > legD && legD > legE) {
		return model.CorrectiveWave{}, false
	}

	// Fibonacci Guideline Pass: legs should retrace roughly between 61.8% and 78.6%
	ratioB := legB / legA
	ratioC := legC / legB
	ratioD := legD / legC
	ratioE := legE / legD

	isValidRatio := func(r float64) bool {
		return r >= 0.50 && r <= 0.88 // Handreiking rondom 0.618 en 0.786
	}

	if !isValidRatio(ratioB) || !isValidRatio(ratioC) || !isValidRatio(ratioD) || !isValidRatio(ratioE) {
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

// evaluateWXY checks for Double Three combinations and allows for terminal Triangles.
func evaluateWXY(p0, p1, p2, p3, p4, p5, p6, p7 *model.Pivot) (model.CorrectiveWave, bool) {
	var direction string
	if p0.Type == model.PivotHigh {
		direction = "BEARISH"
	} else if p0.Type == model.PivotLow {
		direction = "BULLISH"
	} else {
		return model.CorrectiveWave{}, false
	}

	// 1. W-wave verification
	wMatch, _ := evaluateCorrectiveRules(p0, p1, p2, p3, direction)
	if !wMatch {
		return model.CorrectiveWave{}, false
	}

	// 2. X-wave verification (retrace threshold limit < 90%)
	wAmplitude := math.Abs(p3.Price - p0.Price)
	if wAmplitude == 0 {
		return model.CorrectiveWave{}, false
	}
	xLen := math.Abs(p4.Price - p3.Price)
	xRetrace := xLen / wAmplitude
	if xRetrace > 0.90 {
		return model.CorrectiveWave{}, false
	}

	if direction == "BEARISH" {
		if p4.Type != model.PivotHigh || p4.Price <= p3.Price {
			return model.CorrectiveWave{}, false
		}
	} else {
		if p4.Type != model.PivotLow || p4.Price >= p3.Price {
			return model.CorrectiveWave{}, false
		}
	}

	// 3. Y-wave verification (Accepts ABC Flats/ZigZags OR contracting structures)
	yMatch := false
	var purpleBoxes []model.TargetBox

	isABC, _ := evaluateCorrectiveRules(p4, p5, p6, p7, direction)
	if isABC {
		yMatch = true
		purpleBoxes = calculateWaveCTargetBoxes(p4, p5, p6, p7)
	} else {
		// Evaluates if the remaining pivots compress into a terminal pattern consolidation
		leg1 := math.Abs(p5.Price - p4.Price)
		leg2 := math.Abs(p6.Price - p5.Price)
		leg3 := math.Abs(p7.Price - p6.Price)
		if leg1 > leg2 && leg2 > leg3 && leg3 > 0 {
			yMatch = true
		}
	}

	if !yMatch {
		return model.CorrectiveWave{}, false
	}

	// 4. Net destination alignment pass
	if direction == "BEARISH" {
		if p7.Price >= p3.Price {
			return model.CorrectiveWave{}, false
		}
	} else {
		if p7.Price <= p3.Price {
			return model.CorrectiveWave{}, false
		}
	}

	return model.CorrectiveWave{
		Start:       p0,
		WA:          p1,
		WB:          p2,
		WC:          p3,
		WX:          p4,
		WD:          p5,
		WE:          p7,
		Type:        "WXY",
		Direction:   direction,
		PurpleBoxes: purpleBoxes,
	}, true
}
