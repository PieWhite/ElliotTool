package zigzag

import (
	"WaveSight/pkg/model"
)

type trendState int

const (
	stateNone trendState = iota
	stateSearchingHigh
	stateSearchingLow
)

// lowCameFirst returns true if the low at lowIdx occurred before the high at highIdx.
// If they are different indices, it simply compares the indices.
// If they are the same index, it uses the candle's Open/Close relation:
// a green or flat candle implies price went from Low to High, so Low came first.
// a red candle implies price went from High to Low, so High came first.
func lowCameFirst(candles []model.Candle, lowIdx, highIdx int) bool {
	if lowIdx < highIdx {
		return true
	}
	if lowIdx > highIdx {
		return false
	}
	// Same index
	c := candles[lowIdx]
	return c.Close >= c.Open
}

// CalculateZigZag computes the ZigZag pivots (peaks and troughs) for a given slice of candles
// based on a percentage deviation parameter.
//
// The deviation parameter should be given as a percentage (e.g. 5.0 for 5%).
// Memory efficiency is achieved by pre-allocating the returned slice and keeping the state stack-allocated.
func CalculateZigZag(candles []model.Candle, deviation float64) []model.Pivot {
	n := len(candles)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return make([]model.Pivot, 0)
	}

	// Preallocate pivots assuming roughly 10% of candles are pivots (min capacity 2)
	cap := n / 10
	if cap < 2 {
		cap = 2
	}
	pivots := make([]model.Pivot, 0, cap)

	extremeMin := candles[0].Low
	extremeMinIdx := 0
	extremeMax := candles[0].High
	extremeMaxIdx := 0

	needResetMin := false
	needResetMax := false
	state := stateNone

	for i := 1; i < n; i++ {
		c := candles[i]

		// Update High extreme
		if needResetMax {
			extremeMax = c.High
			extremeMaxIdx = i
			needResetMax = false
		} else if c.High > extremeMax {
			extremeMax = c.High
			extremeMaxIdx = i
		}

		// Update Low extreme
		if needResetMin {
			extremeMin = c.Low
			extremeMinIdx = i
			needResetMin = false
		} else if c.Low < extremeMin {
			extremeMin = c.Low
			extremeMinIdx = i
		}

		switch state {
		case stateNone:
			downDev := (extremeMax - c.Low) / extremeMax * 100.0
			upDev := (c.High - extremeMin) / extremeMin * 100.0

			if downDev >= deviation && upDev >= deviation {
				if lowCameFirst(candles, extremeMinIdx, extremeMaxIdx) {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMinIdx].Time,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					state = stateSearchingLow

					// Reset extremeMin search
					startIdx := extremeMaxIdx + 1
					if candles[extremeMaxIdx].Close < candles[extremeMaxIdx].Open {
						startIdx = extremeMaxIdx
					}
					if startIdx <= i {
						extremeMin = candles[startIdx].Low
						extremeMinIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].Low < extremeMin {
								extremeMin = candles[k].Low
								extremeMinIdx = k
							}
						}
					} else {
						needResetMin = true
					}
				} else {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMinIdx].Time,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					state = stateSearchingHigh

					// Reset extremeMax search
					startIdx := extremeMinIdx + 1
					if candles[extremeMinIdx].Close >= candles[extremeMinIdx].Open {
						startIdx = extremeMinIdx
					}
					if startIdx <= i {
						extremeMax = candles[startIdx].High
						extremeMaxIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].High > extremeMax {
								extremeMax = candles[k].High
								extremeMaxIdx = k
							}
						}
					} else {
						needResetMax = true
					}
				}
			} else if downDev >= deviation {
				if !lowCameFirst(candles, extremeMinIdx, extremeMaxIdx) {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					state = stateSearchingLow

					// Reset extremeMin search
					startIdx := extremeMaxIdx + 1
					if candles[extremeMaxIdx].Close < candles[extremeMaxIdx].Open {
						startIdx = extremeMaxIdx
					}
					if startIdx <= i {
						extremeMin = candles[startIdx].Low
						extremeMinIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].Low < extremeMin {
								extremeMin = candles[k].Low
								extremeMinIdx = k
							}
						}
					} else {
						needResetMin = true
					}
				} else {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMinIdx].Time,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					state = stateSearchingLow

					// Reset extremeMin search
					startIdx := extremeMaxIdx + 1
					if candles[extremeMaxIdx].Close < candles[extremeMaxIdx].Open {
						startIdx = extremeMaxIdx
					}
					if startIdx <= i {
						extremeMin = candles[startIdx].Low
						extremeMinIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].Low < extremeMin {
								extremeMin = candles[k].Low
								extremeMinIdx = k
							}
						}
					} else {
						needResetMin = true
					}
				}
			} else if upDev >= deviation {
				if lowCameFirst(candles, extremeMinIdx, extremeMaxIdx) {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMinIdx].Time,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					state = stateSearchingHigh

					// Reset extremeMax search
					startIdx := extremeMinIdx + 1
					if candles[extremeMinIdx].Close >= candles[extremeMinIdx].Open {
						startIdx = extremeMinIdx
					}
					if startIdx <= i {
						extremeMax = candles[startIdx].High
						extremeMaxIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].High > extremeMax {
								extremeMax = candles[k].High
								extremeMaxIdx = k
							}
						}
					} else {
						needResetMax = true
					}
				} else {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMinIdx].Time,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					state = stateSearchingHigh

					// Reset extremeMax search
					startIdx := extremeMinIdx + 1
					if candles[extremeMinIdx].Close >= candles[extremeMinIdx].Open {
						startIdx = extremeMinIdx
					}
					if startIdx <= i {
						extremeMax = candles[startIdx].High
						extremeMaxIdx = startIdx
						for k := startIdx + 1; k <= i; k++ {
							if candles[k].High > extremeMax {
								extremeMax = candles[k].High
								extremeMaxIdx = k
							}
						}
					} else {
						needResetMax = true
					}
				}
			}

		case stateSearchingHigh:
			dev := (extremeMax - c.Low) / extremeMax * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMaxIdx].Time,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
				state = stateSearchingLow

				// Reset extremeMin search
				startIdx := extremeMaxIdx + 1
				if candles[extremeMaxIdx].Close < candles[extremeMaxIdx].Open {
					startIdx = extremeMaxIdx
				}
				if startIdx <= i {
					extremeMin = candles[startIdx].Low
					extremeMinIdx = startIdx
					for k := startIdx + 1; k <= i; k++ {
						if candles[k].Low < extremeMin {
							extremeMin = candles[k].Low
							extremeMinIdx = k
						}
					}
				} else {
					needResetMin = true
				}
			}

		case stateSearchingLow:
			dev := (c.High - extremeMin) / extremeMin * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMinIdx].Time,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
				state = stateSearchingHigh

				// Reset extremeMax search
				startIdx := extremeMinIdx + 1
				if candles[extremeMinIdx].Close >= candles[extremeMinIdx].Open {
					startIdx = extremeMinIdx
				}
				if startIdx <= i {
					extremeMax = candles[startIdx].High
					extremeMaxIdx = startIdx
					for k := startIdx + 1; k <= i; k++ {
						if candles[k].High > extremeMax {
							extremeMax = candles[k].High
							extremeMaxIdx = k
						}
					}
				} else {
					needResetMax = true
				}
			}
		}
	}

	// Finalize at the end of input
	switch state {
	case stateNone:
		if lowCameFirst(candles, extremeMinIdx, extremeMaxIdx) {
			dev := (extremeMax - extremeMin) / extremeMin * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMinIdx].Time,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMaxIdx].Time,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
			}
		} else {
			dev := (extremeMax - extremeMin) / extremeMax * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMaxIdx].Time,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMinIdx].Time,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
			}
		}
	case stateSearchingHigh:
		if len(pivots) > 0 {
			lastLow := pivots[len(pivots)-1]
			dev := (extremeMax - lastLow.Price) / lastLow.Price * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMaxIdx].Time,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
			}
		}
	case stateSearchingLow:
		if len(pivots) > 0 {
			lastHigh := pivots[len(pivots)-1]
			dev := (lastHigh.Price - extremeMin) / lastHigh.Price * 100.0
			if dev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMinIdx].Time,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
			}
		}
	}

	return pivots
}
