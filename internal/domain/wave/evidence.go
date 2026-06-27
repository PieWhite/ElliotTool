package wave

import (
	"math"

	"WaveSight/internal/market"
)

// applyMarketEvidence turns only evidence available in the supplied OHLCV
// series into ranking input. Breadth, sentiment and news remain explicitly
// unobservable because a single-instrument candle series cannot establish them.
func applyMarketEvidence(node WaveNode, candles []market.Candle) WaveNode {
	for index := range node.Children {
		node.Children[index] = applyMarketEvidence(node.Children[index], candles)
	}
	if node.Mode != ModeMotive || len(node.Pivots) < 4 {
		return node
	}

	waveOneVolume, waveOneOK := averageVolume(candles, node.Pivots[0].BarIndex, node.Pivots[1].BarIndex)
	waveThreeVolume, waveThreeOK := averageVolume(candles, node.Pivots[2].BarIndex, node.Pivots[3].BarIndex)
	if waveOneOK && waveThreeOK {
		ratio := waveThreeVolume / nonZero(waveOneVolume)
		status := EvaluationFail
		if ratio >= 1 {
			status = EvaluationPass
		}
		replaceEvaluation(&node.RuleEvaluations, RuleEvaluation{
			RuleID: GuideVolume, Class: RuleGuideline, Status: status,
			Source:   "EWP p.42",
			Summary:  "Third-wave participation should normally expand relative to wave 1.",
			Measured: ratio, Expected: "average wave 3 volume / wave 1 volume >= 1",
		})
	}

	waveOneMomentum, waveOneOK := segmentMomentum(node.Pivots[0], node.Pivots[1])
	waveThreeMomentum, waveThreeOK := segmentMomentum(node.Pivots[2], node.Pivots[3])
	if waveOneOK && waveThreeOK {
		ratio := waveThreeMomentum / nonZero(waveOneMomentum)
		status := EvaluationFail
		if ratio >= 1 {
			status = EvaluationPass
		}
		replaceEvaluation(&node.RuleEvaluations, RuleEvaluation{
			RuleID: GuidePersonality, Class: RuleGuideline, Status: status,
			Source:   "EWP pp.43-46",
			Summary:  "Observable third-wave price momentum supports impulsive personality; breadth and news remain unobserved.",
			Measured: ratio, Expected: "absolute wave 3 price-per-bar / wave 1 price-per-bar >= 1",
		})
	}

	expectedChildren := len(expectedModes(node.Pattern, len(node.Pivots)-1))
	if node.Pattern == PatternDoubleZigzag || node.Pattern == PatternDoubleThree ||
		node.Pattern == PatternTripleZigzag || node.Pattern == PatternTripleThree {
		expectedChildren = len(node.Children)
	}
	node.Conformance = calculateConformance(
		node.RuleEvaluations, len(node.Children), expectedChildren,
	)
	return node
}

func replaceEvaluation(evaluations *[]RuleEvaluation, replacement RuleEvaluation) {
	for index := range *evaluations {
		if (*evaluations)[index].RuleID == replacement.RuleID {
			(*evaluations)[index] = replacement
			return
		}
	}
	*evaluations = append(*evaluations, replacement)
}

func averageVolume(candles []market.Candle, start, end int) (float64, bool) {
	if start < 0 || end <= start || start >= len(candles) {
		return 0, false
	}
	end = minInt(end, len(candles)-1)
	total := 0.0
	count := 0
	for index := start + 1; index <= end; index++ {
		if candles[index].Volume <= 0 || math.IsNaN(candles[index].Volume) ||
			math.IsInf(candles[index].Volume, 0) {
			continue
		}
		total += candles[index].Volume
		count++
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

func segmentMomentum(start, end Pivot) (float64, bool) {
	bars := end.BarIndex - start.BarIndex
	if bars <= 0 {
		return 0, false
	}
	value := math.Abs(end.Price-start.Price) / float64(bars)
	return value, value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
