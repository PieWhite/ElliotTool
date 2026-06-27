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

// lowCameFirst bepaalt de interne chronologische volgorde van een kaars.
func lowCameFirst(candles []model.Candle, lowIdx, highIdx int) bool {
	if lowIdx < highIdx {
		return true
	}
	if lowIdx > highIdx {
		return false
	}
	c := candles[lowIdx]
	return c.Close >= c.Open
}

// CalculateZigZag berekent ruisvrije pivots en corrigeert actief timestamp-collisies
// zodat de trendlijnen op de frontend exact snappen op de uiterste punten van de kaarsen.
func CalculateZigZag(candles []model.Candle, deviation float64) []model.Pivot {
	n := len(candles)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return make([]model.Pivot, 0)
	}

	pivots := make([]model.Pivot, 0, n/10)

	extremeMin := candles[0].Low
	extremeMinIdx := 0
	extremeMax := candles[0].High
	extremeMaxIdx := 0

	needResetMin := false
	needResetMax := false
	state := stateNone

	for i := 1; i < n; i++ {
		c := candles[i]

		if needResetMax {
			extremeMax = c.High
			extremeMaxIdx = i
			needResetMax = false
		} else if c.High > extremeMax {
			extremeMax = c.High
			extremeMaxIdx = i
		}

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
					// VOORKOM COLLISIE: Als de high op dezelfde kaars valt, zet de timestamp 1 seconde vooruit
					tHigh := candles[extremeMaxIdx].Time
					if extremeMinIdx == extremeMaxIdx {
						tHigh += 1
					}
					pivots = append(pivots, model.Pivot{
						Time:  tHigh,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					state = stateSearchingLow
					needResetMin = true
				} else {
					pivots = append(pivots, model.Pivot{
						Time:  candles[extremeMaxIdx].Time,
						Price: extremeMax,
						Type:  model.PivotHigh,
					})
					tLow := candles[extremeMinIdx].Time
					if extremeMinIdx == extremeMaxIdx {
						tLow += 1
					}
					pivots = append(pivots, model.Pivot{
						Time:  tLow,
						Price: extremeMin,
						Type:  model.PivotLow,
					})
					state = stateSearchingHigh
					needResetMax = true
				}
			} else if downDev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMaxIdx].Time,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
				state = stateSearchingLow
				needResetMin = true
			} else if upDev >= deviation {
				pivots = append(pivots, model.Pivot{
					Time:  candles[extremeMinIdx].Time,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
				state = stateSearchingHigh
				needResetMax = true
			}

		case stateSearchingHigh:
			dev := (extremeMax - c.Low) / extremeMax * 100.0
			if dev >= deviation {
				// Controleer of de timestamp botst met de vorige geregistreerde pivot
				tHigh := candles[extremeMaxIdx].Time
				if len(pivots) > 0 && pivots[len(pivots)-1].Time == tHigh {
					tHigh += 1
				}
				pivots = append(pivots, model.Pivot{
					Time:  tHigh,
					Price: extremeMax,
					Type:  model.PivotHigh,
				})
				state = stateSearchingLow
				extremeMin = c.Low
				extremeMinIdx = i
				needResetMin = false
			}

		case stateSearchingLow:
			dev := (c.High - extremeMin) / extremeMin * 100.0
			if dev >= deviation {
				tLow := candles[extremeMinIdx].Time
				if len(pivots) > 0 && pivots[len(pivots)-1].Time == tLow {
					tLow += 1
				}
				pivots = append(pivots, model.Pivot{
					Time:  tLow,
					Price: extremeMin,
					Type:  model.PivotLow,
				})
				state = stateSearchingHigh
				extremeMax = c.High
				extremeMaxIdx = i
				needResetMax = false
			}
		}
	}

	// --- RECHTERRAND STABILISATOR: Garandeer dat de actuele koers kaars altijd sluit ---
	if len(pivots) > 0 {
		lastPivot := pivots[len(pivots)-1]
		lastCandle := candles[n-1]

		if lastPivot.Type == model.PivotLow {
			// Als de laatste stap een Low was, moet de actuele uiterste top aan de rand getekend worden
			tHigh := lastCandle.Time
			if tHigh <= lastPivot.Time {
				tHigh = lastPivot.Time + 1
			}
			pivots = append(pivots, model.Pivot{
				Time:  tHigh,
				Price: extremeMax,
				Type:  model.PivotHigh,
			})
		} else if lastPivot.Type == model.PivotHigh {
			// Als de laatste stap een High was, veranker het lijntje aan de actuele uiterste bodem
			tLow := lastCandle.Time
			if tLow <= lastPivot.Time {
				tLow = lastPivot.Time + 1
			}
			pivots = append(pivots, model.Pivot{
				Time:  tLow,
				Price: extremeMin,
				Type:  model.PivotLow,
			})
		}
	}

	return pivots
}
