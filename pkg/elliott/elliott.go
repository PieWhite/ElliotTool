package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

const (
	cardinalTolerance  = 0.01 // Zeer strikt voor kardinale regels (zoals overlap)
	maxFibTolerance    = 0.15 // 15% soepele ademruimte voor Fibonacci-richtlijnen (Conform Prechter)
	minConfidenceScore = 0.60
	maxLookaheadWindow = 16
)

// MatchMotiveWaves scant pivots voor valide 5-wave Elliott Wave motive structuren.
func MatchMotiveWaves(pivots []model.Pivot) []model.MotiveWave {
	n := len(pivots)
	if n < 6 {
		return nil
	}

	motiveWaves := make([]model.MotiveWave, 0, n/2)

	for i := 0; i < n; i++ {
		p0 := &pivots[i]
		endWindow := i + maxLookaheadWindow
		if endWindow > n {
			endWindow = n
		}

		if p0.Type == model.PivotLow {
			// BULLISH lookahead scan
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				if p1.Type != model.PivotHigh || p1.Price <= p0.Price {
					continue
				}

				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					if p2.Type != model.PivotLow || p2.Price >= p1.Price || p2.Price < p0.Price {
						continue
					}

					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						if p3.Type != model.PivotHigh || p3.Price <= p2.Price {
							continue
						}

						for i4 := i3 + 1; i4 < endWindow; i4++ {
							p4 := &pivots[i4]
							if p4.Type != model.PivotLow || p4.Price >= p3.Price {
								continue
							}

							for i5 := i4 + 1; i5 < endWindow; i5++ {
								p5 := &pivots[i5]
								if p5.Type != model.PivotHigh || p5.Price <= p4.Price {
									continue
								}

								len1 := p1.Price - p0.Price
								len3 := p3.Price - p2.Price
								len5 := p5.Price - p4.Price

								// Kardinale Regel 3: Wave 4 overlap controle
								overlap := p4.Price <= p1.Price
								isDiagonal := false
								if overlap {
									if len1 > len3 && len3 > len5 {
										isDiagonal = true
									} else {
										// Strikte uitsluiting bij ongeldige overlap
										if p4.Price <= p1.Price*(1.0-cardinalTolerance) {
											continue
										}
									}
								}

								// Truncation check
								isTruncated := false
								if p5.Price <= p3.Price {
									if len3 > len1 {
										isTruncated = true
									} else {
										continue
									}
								}

								// Kardinale Regel 2: Wave 3 mag nooit de kortste zijn
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
							}
						}
					}
				}
			}

		} else if p0.Type == model.PivotHigh {
			// BEARISH lookahead scan
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				if p1.Type != model.PivotLow || p1.Price >= p0.Price {
					continue
				}

				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					if p2.Type != model.PivotHigh || p2.Price <= p1.Price || p2.Price > p0.Price {
						continue
					}

					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						if p3.Type != model.PivotLow || p3.Price >= p2.Price {
							continue
						}

						for i4 := i3 + 1; i4 < endWindow; i4++ {
							p4 := &pivots[i4]
							if p4.Type != model.PivotHigh || p4.Price <= p3.Price {
								continue
							}

							for i5 := i4 + 1; i5 < endWindow; i5++ {
								p5 := &pivots[i5]
								if p5.Type != model.PivotLow || p5.Price >= p4.Price {
									continue
								}

								len1 := p0.Price - p1.Price
								len3 := p2.Price - p3.Price
								len5 := p4.Price - p5.Price

								// Kardinale Regel 3: Wave 4 overlap controle
								overlap := p4.Price >= p1.Price
								isDiagonal := false
								if overlap {
									if len1 > len3 && len3 > len5 {
										isDiagonal = true
									} else {
										if p4.Price >= p1.Price*(1.0+cardinalTolerance) {
											continue
										}
									}
								}

								// Truncation check
								isTruncated := false
								if p5.Price >= p3.Price {
									if len3 > len1 {
										isTruncated = true
									} else {
										continue
									}
								}

								// Kardinale Regel 2: Wave 3 mag nooit de kortste zijn
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
					}
				}
			}
		}
	}

	return motiveWaves
}

// MatchIncompleteWaves scant pivots voor opbouwende 1-2-3 Elliott Wave structuren.
func MatchIncompleteWaves(pivots []model.Pivot) []model.IncompleteWave {
	n := len(pivots)
	if n < 4 {
		return nil
	}

	incompleteWaves := make([]model.IncompleteWave, 0, n/2)

	for i := 0; i < n; i++ {
		p0 := &pivots[i]
		endWindow := i + maxLookaheadWindow
		if endWindow > n {
			endWindow = n
		}

		if p0.Type == model.PivotLow {
			// BULLISH lookahead pass
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				if p1.Type != model.PivotHigh || p1.Price <= p0.Price {
					continue
				}

				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					if p2.Type != model.PivotLow || p2.Price >= p1.Price || p2.Price < p0.Price {
						continue
					}

					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						if p3.Type != model.PivotHigh || p3.Price <= p2.Price {
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
					}
				}
			}

		} else if p0.Type == model.PivotHigh {
			// BEARISH lookahead pass
			for i1 := i + 1; i1 < endWindow; i1++ {
				p1 := &pivots[i1]
				if p1.Type != model.PivotLow || p1.Price >= p0.Price {
					continue
				}

				for i2 := i1 + 1; i2 < endWindow; i2++ {
					p2 := &pivots[i2]
					if p2.Type != model.PivotHigh || p2.Price <= p1.Price || p2.Price > p0.Price {
						continue
					}

					for i3 := i2 + 1; i3 < endWindow; i3++ {
						p3 := &pivots[i3]
						if p3.Type != model.PivotLow || p3.Price >= p2.Price {
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
			}
		}
	}

	return incompleteWaves
}

// calculateConfidenceScore berekent de Fibonacci alignment score via soepele lineaire afbouw.
func calculateConfidenceScore(p0, p1, p2, p3, p4, p5 *model.Pivot, isDiagonal, isTruncated bool) float64 {
	len1 := math.Abs(p1.Price - p0.Price)
	len2 := math.Abs(p1.Price - p2.Price)
	len3 := math.Abs(p3.Price - p2.Price)
	len4 := math.Abs(p3.Price - p4.Price)
	len5 := math.Abs(p5.Price - p4.Price)
	net0to3 := math.Abs(p3.Price - p0.Price)

	if len1 == 0 || net0to3 == 0 || len3 == 0 {
		return 0.0
	}

	// --- 1. Wave 2 Retracement (50%, 61.8%, 78.6%) ---
	ratio2 := len2 / len1
	minD2 := math.Abs(ratio2 - 0.50)
	if d := math.Abs(ratio2 - 0.618); d < minD2 {
		minD2 = d
	}
	if d := math.Abs(ratio2 - 0.786); d < minD2 {
		minD2 = d
	}
	score2 := 0.0
	if minD2 <= maxFibTolerance {
		score2 = 1.0 - (minD2 / maxFibTolerance)
	}

	// --- 2. Wave 3 Extension ---
	ratio3 := len3 / len1
	score3 := 0.0
	if isDiagonal {
		minD3 := math.Abs(ratio3 - 0.618)
		if d := math.Abs(ratio3 - 0.786); d < minD3 {
			minD3 = d
		}
		if minD3 <= maxFibTolerance {
			score3 = 1.0 - (minD3 / maxFibTolerance)
		}
	} else {
		if ratio3 >= 1.618-cardinalTolerance {
			score3 = 1.0
		} else {
			d3 := math.Abs(ratio3 - 1.00) // W3 kan gelijk zijn aan W1 bij W5 extensie
			if d3 <= maxFibTolerance {
				score3 = 1.0 - (d3 / maxFibTolerance)
			}
		}
	}

	// --- 3. Wave 4 Retracement (24%, 38.2%, 50%) ---
	ratio4 := len4 / len3
	minD4 := math.Abs(ratio4 - 0.24)
	if d := math.Abs(ratio4 - 0.382); d < minD4 {
		minD4 = d
	}
	if d := math.Abs(ratio4 - 0.50); d < minD4 {
		minD4 = d
	}
	score4 := 0.0
	if minD4 <= maxFibTolerance {
		score4 = 1.0 - (minD4 / maxFibTolerance)
	}

	// --- 4. Dynamische Wave 5 Score ---
	score5 := 0.0
	minD5 := 999.0

	if ratio3 >= 1.618 {
		// Wave 3 verlengd: verhouding tot W1 (1.00, 1.618, 2.618)
		r5A := len5 / len1
		targets := []float64{1.00, 1.618, 2.618}
		for _, t := range targets {
			if d := math.Abs(r5A - t); d < minD5 {
				minD5 = d
			}
		}
	} else {
		// Wave 3 kort: verhouding tot netto 0->3 (0.618, 1.00, 1.618)
		r5B := len5 / net0to3
		targets := []float64{0.618, 1.00, 1.618}
		for _, t := range targets {
			if d := math.Abs(r5B - t); d < minD5 {
				minD5 = d
			}
		}
	}

	if isTruncated || isDiagonal {
		if d := math.Abs((len5 / len4) - 0.382); d < minD5 {
			minD5 = d
		}
		if d := math.Abs((len5 / len4) - 0.618); d < minD5 {
			minD5 = d
		}
	}

	if minD5 <= maxFibTolerance {
		score5 = 1.0 - (minD5 / maxFibTolerance)
	}

	return (score2 + score3 + score4 + score5) / 4.0
}

// calculateIncompleteConfidenceScore berekent de score voor een opbouwende 1-2-3 structuur.
func calculateIncompleteConfidenceScore(p0, p1, p2, p3 *model.Pivot) float64 {
	len1 := math.Abs(p1.Price - p0.Price)
	len2 := math.Abs(p1.Price - p2.Price)
	len3 := math.Abs(p3.Price - p2.Price)

	if len1 == 0 {
		return 0.0
	}

	ratio2 := len2 / len1
	minD2 := math.Abs(ratio2 - 0.50)
	if d := math.Abs(ratio2 - 0.618); d < minD2 {
		minD2 = d
	}
	if d := math.Abs(ratio2 - 0.786); d < minD2 {
		minD2 = d
	}
	score2 := 0.0
	if minD2 <= maxFibTolerance {
		score2 = 1.0 - (minD2 / maxFibTolerance)
	}

	// Versoepeld voor ontluikende trends: groter dan 100% is valide uitbraak
	ratio3 := len3 / len1
	score3 := 0.0
	if ratio3 >= 1.00 {
		score3 = math.Min(1.0, ratio3/1.618)
	}

	return (score2 + score3) / 2.0
}

// calculateTargetBox genereert paarse doelvakken op basis van de Wave 3 extensiestatus.
func calculateTargetBox(p0, p1, p2, p3, p4, p5 *model.Pivot) []model.TargetBox {
	len1 := math.Abs(p1.Price - p0.Price)
	len3 := math.Abs(p3.Price - p2.Price)
	net0to3 := math.Abs(p3.Price - p0.Price)
	if len1 == 0 || net0to3 == 0 || len3 == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p4.Time, p3.Time-p0.Time)

	ratio3 := len3 / len1
	var distances []float64

	if ratio3 >= 1.618 {
		distances = []float64{len1, len1 * 1.618, len1 * 2.618}
	} else {
		distances = []float64{net0to3 * 0.618, net0to3, net0to3 * 1.618}
	}

	return targetBoxesFromOrigin(p4.Price, direction, startTime, endTime, distances)
}

// calculateWave3TargetBoxes gebruikt de eSignal-verhoudingen (1.62, 2.62, 4.25).
func calculateWave3TargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	len1 := math.Abs(p1.Price - p0.Price)
	if len1 == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)

	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{
		len1 * 1.62,
		len1 * 2.62,
		len1 * 4.25,
	})
}

// calculateWaveCTargetBoxes voegt de missende 0.618 ratio toe.
func calculateWaveCTargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return nil
	}

	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)

	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{
		lenA * 0.618,
		lenA,
		lenA * 1.618,
	})
}

// calculateWave4TargetBox voorspelt het 38.2% doelniveau.
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
