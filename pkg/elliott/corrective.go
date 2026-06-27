package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

const (
	fibToleranceCorrective = 0.03
	maxLookaheadCorrective = 16
)

// MatchCorrectiveWaves scant een slice van pivots voor valide Elliott Wave correctiestructuren.
// Het retourneert alle gedetecteerde ZIGZAG, FLAT, TRIANGLE (Contracting, Expanding, Barrier) en WXY patronen.
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

		// --- 1. 3-wave ABC structuren: ZigZag & Flat (Non-consecutive lookahead) ---
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
						if p1.Type != model.PivotLow || p2.Type != model.PivotHigh || p3.Type != model.PivotLow {
							continue
						}
						if p1.Price >= p0.Price || p2.Price <= p1.Price || p3.Price >= p2.Price {
							continue
						}
						direction = "BEARISH"
						isPatternMatch, correctiveType = evaluateCorrectiveRules(p0, p1, p2, p3, direction)

					} else if p0.Type == model.PivotLow {
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

		// --- 2. Driehoek ABCDE Structuren (6 pivots: Start + A + B + C + D + E) ---
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

		// --- 3. WXY Double Three Combinaties (10 pivots: Start + W(3) + X(3) + Y(3)) ---
		// Wave X is zelf verplicht een herkenbare abc correctie.
		if i+9 < n {
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
										for i8 := i7 + 1; i8 < endWindow; i8++ {
											p8 := &pivots[i8]
											for i9 := i8 + 1; i9 < endWindow; i9++ {
												p9 := &pivots[i9]

												wxy, ok := evaluateWXYFull(p0, p1, p2, p3, p4, p5, p6, p7, p8, p9)
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
		}
	}

	return correctiveWaves
}

// evaluateCorrectiveRules classificeert een 4-pivot ABC structuur conform Expanded/Regular wetten.
func evaluateCorrectiveRules(p0, p1, p2, p3 *model.Pivot, direction string) (bool, string) {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return false, ""
	}

	lenB := math.Abs(p2.Price - p1.Price)
	lenC := math.Abs(p3.Price - p2.Price)

	ratioB := lenB / lenA
	ratioC := lenC / lenA

	// 1. Gecorrigeerde ZigZag Logica (38.2% - 61.8% retracement voor Wave B)
	//
	if ratioB >= 0.382-fibToleranceCorrective && ratioB <= 0.618+fibToleranceCorrective {
		cleanBreak := false
		if direction == "BEARISH" && p3.Price < p1.Price {
			cleanBreak = true
		} else if direction == "BULLISH" && p3.Price > p1.Price {
			cleanBreak = true
		}

		if cleanBreak {
			if math.Abs(ratioC-0.62) <= fibToleranceCorrective ||
				math.Abs(ratioC-1.000) <= fibToleranceCorrective ||
				math.Abs(ratioC-1.62) <= fibToleranceCorrective ||
				(ratioC >= 1.000 && ratioC <= 1.62) {
				return true, "ZIGZAG"
			}
		}
	}

	// 2. Gecorrigeerde Flat Logica (Inclusief de cruciale Expanded Flats tot 138.2%)
	//
	if ratioB >= 0.900-fibToleranceCorrective && ratioB <= 1.382+fibToleranceCorrective {
		if ratioB > 1.05 {
			if ratioC >= 1.000 || math.Abs(ratioC-1.236) <= fibToleranceCorrective || math.Abs(ratioC-1.62) <= fibToleranceCorrective {
				return true, "FLAT"
			}
		} else {
			if ratioC >= 0.900-fibToleranceCorrective && ratioC <= 1.300+fibToleranceCorrective {
				return true, "FLAT"
			}
		}
	}

	return false, ""
}

// evaluateTriangle controleert op Contracting, Expanding en Barrier Triangles gebaseerd op Frost & Prechter.
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

	isContracting := legA > legB && legB > legC && legC > legD && legD > legE
	isExpanding := legA < legB && legB < legC && legC < legD && legD < legE
	isBarrier := math.Abs(p2.Price-p4.Price)/p2.Price <= 0.015 || math.Abs(p3.Price-p5.Price)/p3.Price <= 0.015

	if !isContracting && !isExpanding && !isBarrier {
		return model.CorrectiveWave{}, false
	}

	ratioB := legB / legA
	ratioC := legC / legB
	ratioD := legD / legC
	ratioE := legE / legD

	isValidRatio := func(r float64) bool {
		return r >= 0.45 && r <= 0.90
	}

	if !isExpanding {
		if !isValidRatio(ratioB) || !isValidRatio(ratioC) || !isValidRatio(ratioD) || !isValidRatio(ratioE) {
			return model.CorrectiveWave{}, false
		}
	}

	typeName := "TRIANGLE_CONTRACTING"
	if isExpanding {
		typeName = "TRIANGLE_EXPANDING"
	}
	if isBarrier {
		typeName = "TRIANGLE_BARRIER"
	}

	return model.CorrectiveWave{
		Start:     p0,
		WA:        p1,
		WB:        p2,
		WC:        p3,
		WD:        p4,
		WE:        p5,
		Type:      typeName,
		Direction: direction,
	}, true
}

// evaluateWXYFull valideert een Double Three combinatie (10-pivots).
// Garandeert dat Wave W NOOIT een Triangle is.
func evaluateWXYFull(p0, p1, p2, p3, p4, p5, p6, p7, p8, p9 *model.Pivot) (model.CorrectiveWave, bool) {
	var direction string
	if p0.Type == model.PivotHigh {
		direction = "BEARISH"
	} else if p0.Type == model.PivotLow {
		direction = "BULLISH"
	} else {
		return model.CorrectiveWave{}, false
	}

	// 1. Valideer Wave W (p0 tot p3) -> Alleen FLAT of ZIGZAG toegestaan (Triangle uitgesloten conform Prechter pag 31)
	wMatch, wType := evaluateCorrectiveRules(p0, p1, p2, p3, direction)
	if !wMatch || wType == "TRIANGLE" {
		return model.CorrectiveWave{}, false
	}

	// 2. Valideer Wave X als een volwaardige tegen-correctie a-b-c (p3 tot p6)
	oppDirection := "BULLISH"
	if direction == "BULLISH" {
		oppDirection = "BEARISH"
	}
	xMatch, _ := evaluateCorrectiveRules(p3, p4, p5, p6, oppDirection)
	if !xMatch {
		return model.CorrectiveWave{}, false
	}

	wAmplitude := math.Abs(p3.Price - p0.Price)
	if wAmplitude == 0 {
		return model.CorrectiveWave{}, false
	}
	xLen := math.Abs(p6.Price - p3.Price)
	if xLen/wAmplitude > 0.90 {
		return model.CorrectiveWave{}, false
	}

	// 3. Valideer Wave Y (p6 tot p9) -> Mag ABC OF een terminale driehoek-compressie zijn
	yMatch := false
	isABC, _ := evaluateCorrectiveRules(p6, p7, p8, p9, direction)
	if isABC {
		yMatch = true
	} else {
		leg1 := math.Abs(p7.Price - p6.Price)
		leg2 := math.Abs(p8.Price - p7.Price)
		leg3 := math.Abs(p9.Price - p8.Price)
		if leg1 > leg2 && leg2 > leg3 && leg3 > 0 {
			yMatch = true
		}
	}

	if !yMatch {
		return model.CorrectiveWave{}, false
	}

	// 4. Richtingscontrole
	if direction == "BEARISH" {
		if p9.Price >= p3.Price {
			return model.CorrectiveWave{}, false
		}
	} else {
		if p9.Price <= p3.Price {
			return model.CorrectiveWave{}, false
		}
	}

	return model.CorrectiveWave{
		Start:       p0,
		WA:          p1,
		WB:          p2,
		WC:          p3, // Einde Wave W
		WX:          p6, // Einde Wave X
		WD:          p7,
		WE:          p9, // Eindpunt Double Three
		Type:        "WXY",
		Direction:   direction,
		PurpleBoxes: calculateWaveCTargetBoxes(p6, p7, p8, p9),
	}, true
}
