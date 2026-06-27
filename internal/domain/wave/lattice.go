package wave

import (
	"math"
	"sort"

	"WaveSight/internal/market"
)

var prominenceThresholds = [...]float64{0, 0.5, 1, 2, 4, 8}

// PivotLattice contains alternate OHLC order branches and volatility-ranked
// levels. Level zero always retains the complete normalized extrema chain.
type PivotLattice struct {
	Branches []PivotBranch
}

type PivotBranch struct {
	Levels [][]Pivot
}

func BuildPivotLattice(candles []market.Candle) PivotLattice {
	if len(candles) < 3 {
		return PivotLattice{}
	}

	atr := market.ATR(candles, 14)
	lowFirst := localExtrema(candles, atr, true)
	highFirst := localExtrema(candles, atr, false)

	branches := make([]PivotBranch, 0, 2)
	branches = append(branches, PivotBranch{Levels: buildLevels(lowFirst)})
	if !samePivotSequence(lowFirst, highFirst) {
		branches = append(branches, PivotBranch{Levels: buildLevels(highFirst)})
	}
	return PivotLattice{Branches: branches}
}

func localExtrema(candles []market.Candle, atr []float64, lowFirst bool) []Pivot {
	candidates := make([]Pivot, 0, len(candles)/2)
	for i := 1; i < len(candles)-1; i++ {
		candle := candles[i]
		isHigh := candle.High >= candles[i-1].High && candle.High >= candles[i+1].High
		isLow := candle.Low <= candles[i-1].Low && candle.Low <= candles[i+1].Low
		state := PivotConfirmed
		if isHigh && isLow {
			state = PivotAmbiguous
		}

		addHigh := func() {
			candidates = append(candidates, Pivot{
				Time: candle.Time, BarIndex: candle.BarIndex, Price: candle.High,
				Kind: PivotHigh, State: state,
			})
		}
		addLow := func() {
			candidates = append(candidates, Pivot{
				Time: candle.Time, BarIndex: candle.BarIndex, Price: candle.Low,
				Kind: PivotLow, State: state,
			})
		}

		if lowFirst {
			if isLow {
				addLow()
			}
			if isHigh {
				addHigh()
			}
		} else {
			if isHigh {
				addHigh()
			}
			if isLow {
				addLow()
			}
		}
	}

	candidates = normalizeAlternation(candidates)
	if len(candidates) == 0 {
		return nil
	}

	// Preserve an observable boundary at the beginning and a provisional
	// boundary at the right edge.
	first := candles[0]
	if candidates[0].Kind == PivotHigh {
		candidates = append([]Pivot{{
			Time: first.Time, BarIndex: first.BarIndex, Price: first.Low,
			Kind: PivotLow, State: PivotConfirmed,
		}}, candidates...)
	} else {
		candidates = append([]Pivot{{
			Time: first.Time, BarIndex: first.BarIndex, Price: first.High,
			Kind: PivotHigh, State: PivotConfirmed,
		}}, candidates...)
	}

	last := candles[len(candles)-1]
	if candidates[len(candidates)-1].Kind == PivotLow {
		candidates = append(candidates, Pivot{
			Time: last.Time, BarIndex: last.BarIndex, Price: last.High,
			Kind: PivotHigh, State: PivotProvisional,
		})
	} else {
		candidates = append(candidates, Pivot{
			Time: last.Time, BarIndex: last.BarIndex, Price: last.Low,
			Kind: PivotLow, State: PivotProvisional,
		})
	}
	candidates = normalizeAlternation(candidates)

	for i := range candidates {
		left := math.Inf(1)
		right := math.Inf(1)
		if i > 0 {
			left = math.Abs(candidates[i].Price - candidates[i-1].Price)
		}
		if i+1 < len(candidates) {
			right = math.Abs(candidates[i+1].Price - candidates[i].Price)
		}
		distance := math.Min(left, right)
		if math.IsInf(distance, 1) {
			distance = math.Max(left, right)
		}
		barATR := atr[minInt(candidates[i].BarIndex, len(atr)-1)]
		if barATR > 0 {
			candidates[i].Prominence = distance / barATR
		}
	}
	return candidates
}

func normalizeAlternation(input []Pivot) []Pivot {
	if len(input) == 0 {
		return nil
	}
	sorted := append([]Pivot(nil), input...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].BarIndex == sorted[j].BarIndex {
			return i < j
		}
		return sorted[i].BarIndex < sorted[j].BarIndex
	})

	result := make([]Pivot, 0, len(sorted))
	for _, pivot := range sorted {
		if len(result) == 0 || result[len(result)-1].Kind != pivot.Kind {
			result = append(result, pivot)
			continue
		}
		last := &result[len(result)-1]
		if (pivot.Kind == PivotHigh && pivot.Price >= last.Price) ||
			(pivot.Kind == PivotLow && pivot.Price <= last.Price) {
			*last = pivot
		}
	}
	return result
}

func buildLevels(base []Pivot) [][]Pivot {
	if len(base) == 0 {
		return nil
	}
	levels := make([][]Pivot, 0, len(prominenceThresholds))
	for _, threshold := range prominenceThresholds {
		level := make([]Pivot, 0, len(base))
		for i, pivot := range base {
			if threshold == 0 || pivot.Prominence >= threshold || i == 0 || i == len(base)-1 {
				level = append(level, pivot)
			}
		}
		level = normalizeAlternation(level)
		if len(level) < 2 {
			continue
		}
		if len(levels) == 0 || !samePivotSequence(levels[len(levels)-1], level) {
			levels = append(levels, level)
		}
	}
	return levels
}

func samePivotSequence(a, b []Pivot) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].BarIndex != b[i].BarIndex || a[i].Kind != b[i].Kind || a[i].Price != b[i].Price {
			return false
		}
	}
	return true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
