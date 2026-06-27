package elliott

import (
	"math"

	"WaveSight/pkg/model"
)

const (
	maxFibTolerance    = 0.15 // 15% soepele ademruimte voor Fibonacci-richtlijnen (Conform Prechter)
	minConfidenceScore = 0.60
	maxLookaheadWindow = 16 // Diepte van het skippen van sub-pivots
)

// MatchMotiveWaves scant een slice van pivots voor valide 5-wave Elliott Wave motive structuren.
// Het handhaaft een absolute null-tolerance beleid op de kardinale wetten van R.N. Elliott.
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
			// BULLISH non-consecutive lookahead scan
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

								// Harde Kardinale Regel 1: Wave 2 mag nooit onder de start van Wave 1 breken
								if p2.Price < p0.Price {
									continue
								}

								// Harde Kardinale Regel 3: Wave 4 overlap controle
								overlap := p4.Price <= p1.Price
								isDiagonal := false
								if overlap {
									// Alleen toegestaan in een convergerende diagonaal (W1 > W3 > W5)
									if len1 > len3 && len3 > len5 {
										isDiagonal = true
									} else {
										continue // Illegale overlap -> direct afkeuren!
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

								// Harde Kardinale Regel 2: Wave 3 mag NOOIT de kortste zijn
								if len3 < len1 && len3 < len5 {
									continue // Illegaal -> direct afkeuren!
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
			// BEARISH non-consecutive lookahead scan
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

								// Harde Kardinale Regel 1: Wave 2 mag nooit boven de start van Wave 1 breken
								if p2.Price > p0.Price {
									continue
								}

								// Harde Kardinale Regel 3: Wave 4 overlap controle
								overlap := p4.Price >= p1.Price
								isDiagonal := false
								if overlap {
									if len1 > len3 && len3 > len5 {
										isDiagonal = true
									} else {
										continue // Illegale overlap -> direct afkeuren!
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

								// Harde Kardinale Regel 2: Wave 3 mag NOOIT de kortste zijn
								if len3 < len1 && len3 < len5 {
									continue // Illegaal -> direct afkeuren!
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

						// Harde validatie: Wave 2 mag de start van Wave 1 niet breken
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

						// Harde validatie: Wave 2 mag de start van Wave 1 niet breken
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
			}
		}
	}

	return incompleteWaves
}

// scoreWave4Continual evalueert de statistische kansverdeling van Wave 4.
func scoreWave4Continual(ratio4 float64) float64 {
	if ratio4 >= 0.30 && ratio4 <= 0.50 {
		return 1.0 // 60% van de marktgevallen (Conform pag 10)
	}
	if ratio4 >= 0.24 && ratio4 < 0.30 {
		return 0.75 + 0.25*(ratio4-0.24)/(0.30-0.24) // 15% kans zone
	}
	if ratio4 > 0.50 && ratio4 <= 0.62 {
		return 0.75 + 0.25*(0.62-ratio4)/(0.62-0.50) // 15% kans zone
	}
	if ratio4 < 0.24 && ratio4 >= 0.24-maxFibTolerance {
		return 0.75 * (ratio4 - (0.24 - maxFibTolerance)) / maxFibTolerance
	}
	if ratio4 > 0.62 && ratio4 <= 0.62+maxFibTolerance {
		return 0.75 * ((0.62 + maxFibTolerance) - ratio4) / maxFibTolerance
	}
	return 0.0
}

// calculateConfidenceScore berekent de normalisatiescore op basis van de PDF-richtlijnen.
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

	// --- 1. Wave 2 Retracement Check ---
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

	// --- 2. Wave 3 Extension Check (1.00, 1.62, 2.62, 4.25 conform pag 8) ---
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
		targets3 := []float64{1.00, 1.62, 2.62, 4.25}
		minD3 := 999.0
		for _, t := range targets3 {
			if d := math.Abs(ratio3 - t); d < minD3 {
				minD3 = d
			}
		}
		if minD3 <= maxFibTolerance {
			score3 = 1.0 - (minD3 / maxFibTolerance)
		}
	}

	// --- 3. Wave 4 Retracement Vloeibaar Continuum ---
	ratio4 := len4 / len3
	score4 := scoreWave4Continual(ratio4)

	// --- 4. Wave 5 Target Check Co-existentie (Conform pag 13) ---
	minD5 := 999.0
	targets5A := []float64{1.00, 1.62, 2.62}
	for _, t := range targets5A {
		if d := math.Abs((len5 / len1) - t); d < minD5 {
			minD5 = d
		}
	}
	targets5B := []float64{0.62, 1.00, 1.62}
	for _, t := range targets5B {
		if d := math.Abs((len5 / net0to3) - t); d < minD5 {
			minD5 = d
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
	score5 := 0.0
	if minD5 <= maxFibTolerance {
		score5 = 1.0 - (minD5 / maxFibTolerance)
	}

	// --- 5. Wet van Afwisseling Pass ---
	scoreAlt := 0.20
	if (ratio2 > 0.48 && ratio4 < 0.36) || (ratio2 < 0.36 && ratio4 > 0.48) {
		scoreAlt = 1.0
	} else if math.Abs(ratio2-ratio4) >= 0.15 {
		scoreAlt = 0.75
	}

	return (score2 + score3 + score4 + score5 + scoreAlt) / 5.0
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

	ratio3 := len3 / len1
	score3 := 0.0
	if ratio3 >= 1.00 {
		score3 = math.Min(1.0, ratio3/1.62)
	}

	return (score2 + score3) / 2.0
}

// calculateTargetBox genereert de schuine, dynamische parallelle kanalen.
// Volgt exact de kanaal-techniek van pagina 5.
func calculateTargetBox(p0, p1, p2, p3, p4, p5 *model.Pivot) []model.TargetBox {
	if p4.Time == p2.Time {
		return nil
	}

	// Bereken de helling (m) van de trendlijn tussen Wave 2 en Wave 4
	m := (p4.Price - p2.Price) / float64(p4.Time-p2.Time)

	startTime, endTime := targetTimeWindow(p4.Time, p3.Time-p0.Time)
	boxes := make([]model.TargetBox, 0, 2)

	// Lijn vanaf Top Wave 1
	p1Start := p1.Price + m*float64(startTime-p1.Time)
	p1End := p1.Price + m*float64(endTime-p1.Time)
	minP1, maxP1 := math.Min(p1Start, p1End), math.Max(p1Start, p1End)

	boxes = append(boxes, model.TargetBox{
		MinPrice:  minP1 * 0.985,
		MaxPrice:  maxP1 * 1.015,
		StartTime: startTime,
		EndTime:   endTime,
	})

	// Lijn vanaf Top Wave 3
	p3Start := p3.Price + m*float64(startTime-p3.Time)
	p3End := p3.Price + m*float64(endTime-p3.Time)
	minP3, maxP3 := math.Min(p3Start, p3End), math.Max(p3Start, p3End)

	boxes = append(boxes, model.TargetBox{
		MinPrice:  minP3 * 0.985,
		MaxPrice:  maxP3 * 1.015,
		StartTime: startTime,
		EndTime:   endTime,
	})

	return boxes
}

func calculateWave3TargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	len1 := math.Abs(p1.Price - p0.Price)
	if len1 == 0 {
		return nil
	}
	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)
	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{len1 * 1.62, len1 * 2.62, len1 * 4.25})
}

func calculateWaveCTargetBoxes(p0, p1, p2, p3 *model.Pivot) []model.TargetBox {
	lenA := math.Abs(p1.Price - p0.Price)
	if lenA == 0 {
		return nil
	}
	direction := waveDirection(p0, p1)
	startTime, endTime := targetTimeWindow(p2.Time, p3.Time-p0.Time)
	return targetBoxesFromOrigin(p2.Price, direction, startTime, endTime, []float64{lenA * 0.62, lenA, lenA * 1.62})
}

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
	minPrice, maxPrice := val1, val2
	if val2 < val1 {
		minPrice, maxPrice = val2, val1
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
		minPrice, maxPrice := val1, val2
		if val2 < val1 {
			minPrice, maxPrice = val2, val1
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
