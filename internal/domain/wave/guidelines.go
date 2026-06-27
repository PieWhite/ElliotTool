package wave

import "math"

func nearestRatio(value float64, targets []float64) (float64, float64) {
	bestTarget := 0.0
	bestDistance := math.Inf(1)
	for _, target := range targets {
		distance := math.Abs(value - target)
		if distance < bestDistance {
			bestDistance = distance
			bestTarget = target
		}
	}
	return bestTarget, bestDistance
}

func ratioGuideline(id, source string, value float64, targets []float64, expected string) RuleEvaluation {
	target, distance := nearestRatio(value, targets)
	status := EvaluationFail
	// Ratios rank continuously; the tolerance is an evidence band, never a
	// structural validity boundary.
	if distance <= math.Max(0.03, target*0.08) {
		status = EvaluationPass
	}
	return RuleEvaluation{
		RuleID: id, Class: RuleGuideline, Status: status, Source: source,
		Summary: expected, Measured: value, Expected: expected,
	}
}

func wave2RatioEvaluation(value float64) RuleEvaluation {
	return ratioGuideline(PriorWave2Ratios, "WaveRatios pp.2,6", value, []float64{0.5, 0.618}, "near 0.50 or 0.618")
}

func wave3RatioEvaluation(value float64) RuleEvaluation {
	return ratioGuideline(PriorWave3Ratios, "WaveRatios pp.3,8", value, []float64{1.618, 2.618, 4.236, 4.25}, "near 1.618, 2.618 or 4.236/4.25")
}

func wave4RatioEvaluation(value float64) RuleEvaluation {
	return ratioGuideline(PriorWave4Ratios, "WaveRatios pp.3,10", value, []float64{0.236, 0.24, 0.382, 0.5}, "near 0.24, 0.382 or 0.50")
}

func wave5RatioEvaluation(relativeWave1, relativeZeroThree float64) RuleEvaluation {
	targetA, distanceA := nearestRatio(relativeWave1, []float64{0.618, 1, 1.618, 2.618})
	targetB, distanceB := nearestRatio(relativeZeroThree, []float64{0.382, 0.618, 1, 1.618})
	target := targetA
	distance := distanceA
	measured := relativeWave1
	if distanceB < distanceA {
		target = targetB
		distance = distanceB
		measured = relativeZeroThree
	}
	status := EvaluationFail
	if distance <= math.Max(0.03, target*0.08) {
		status = EvaluationPass
	}
	return RuleEvaluation{
		RuleID: PriorWave5Ratios, Class: RuleStatisticalPrior, Status: status,
		Source:   "EWP pp.73-74; WaveRatios pp.4,12-13",
		Summary:  "Wave 5 relation to wave 1 or the 0-to-3 distance.",
		Measured: measured, Expected: "0.382, 0.618, 1.0, 1.618 or 2.618 by applicable family",
	}
}

func alternationEvaluation(wave2, wave4 float64) RuleEvaluation {
	difference := math.Abs(wave2 - wave4)
	status := EvaluationFail
	if difference >= 0.12 {
		status = EvaluationPass
	}
	return evaluation(GuideAlternation, status, difference, "wave 2 and wave 4 differ in depth/form")
}

func equalityEvaluation(lengths []float64) RuleEvaluation {
	if len(lengths) < 5 {
		return evaluation(GuideEquality, EvaluationNotApplicable, 0, "five-wave structure required")
	}
	ratio := lengths[4] / nonZero(lengths[0])
	target, distance := nearestRatio(ratio, []float64{0.618, 1, 1.618})
	status := EvaluationFail
	if distance <= target*0.08 {
		status = EvaluationPass
	}
	return evaluation(GuideEquality, status, ratio, "wave 1 and wave 5 equality or Fibonacci relation")
}

func channelEvaluation(points []Pivot) RuleEvaluation {
	if len(points) < 5 || points[4].BarIndex == points[2].BarIndex {
		return evaluation(GuideChannel, EvaluationNotApplicable, 0, "wave 2 and wave 4 required")
	}
	m := (points[4].Price - points[2].Price) / float64(points[4].BarIndex-points[2].BarIndex)
	projected := points[3].Price + m*float64(points[len(points)-1].BarIndex-points[3].BarIndex)
	distance := math.Abs(points[len(points)-1].Price-projected) / nonZero(math.Abs(points[3].Price-points[0].Price))
	status := EvaluationFail
	if distance <= 0.15 {
		status = EvaluationPass
	}
	return evaluation(GuideChannel, status, distance, "fifth wave near parallel channel")
}

func triangleRatioEvaluation(lengths []float64, pattern PatternType) RuleEvaluation {
	target := 0.618
	if pattern == PatternTriangleExpanding {
		target = 1.618
	}
	bestDistance := math.Inf(1)
	bestRatio := 0.0
	for i := 0; i+2 < len(lengths); i++ {
		ratio := lengths[i+2] / nonZero(lengths[i])
		if distance := math.Abs(ratio - target); distance < bestDistance {
			bestDistance = distance
			bestRatio = ratio
		}
	}
	status := EvaluationFail
	if bestDistance <= target*0.12 {
		status = EvaluationPass
	}
	return RuleEvaluation{
		RuleID: "EWP-TRIANGLE-ALTERNATE-RATIO", Class: RuleGuideline, Status: status,
		Source: "EWP p.75", Summary: "Alternate triangle legs tend to relate by 0.618 or 1.618.",
		Measured: bestRatio, Expected: "alternate leg ratio near target",
	}
}

func truncationStrengthEvaluation(lengths []float64) RuleEvaluation {
	if len(lengths) < 5 {
		return evaluation("EWP-TRUNCATION-STRONG-THIRD", EvaluationNotApplicable, 0, "five waves required")
	}
	ratio := lengths[2] / nonZero(lengths[0])
	status := EvaluationFail
	if ratio >= 1.618 {
		status = EvaluationPass
	}
	return RuleEvaluation{
		RuleID: "EWP-TRUNCATION-STRONG-THIRD", Class: RuleGuideline, Status: status,
		Source: "EWP pp.15-16", Summary: "Truncation often follows an extensively strong third wave.",
		Measured: ratio, Expected: "strong third wave; no hard 2.618 requirement",
	}
}

func volumeNotObservable() RuleEvaluation {
	return evaluation(GuideVolume, EvaluationNotObservable, 0, "evaluated by scenario context when volume series is available")
}

func personalityNotObservable() RuleEvaluation {
	return evaluation(GuidePersonality, EvaluationNotObservable, 0, "breadth/news context not observable from one instrument")
}

func rightLookEvaluation(points []Pivot) RuleEvaluation {
	if len(points) < 3 {
		return evaluation(GuideRightLook, EvaluationNotApplicable, 0, "at least two legs required")
	}
	lengths := segmentLengths(points)
	minimum := math.Inf(1)
	maximum := 0.0
	for _, length := range lengths {
		minimum = math.Min(minimum, length)
		maximum = math.Max(maximum, length)
	}
	balance := maximum / nonZero(minimum)
	status := EvaluationFail
	if balance <= 12 {
		status = EvaluationPass
	}
	return evaluation(GuideRightLook, status, balance, "largest leg / smallest leg remains visually coherent")
}

func semilogEvaluation(points []Pivot) RuleEvaluation {
	if len(points) < 2 {
		return evaluation(GuideSemilog, EvaluationNotApplicable, 0, "multiple price points required")
	}
	minimum := minPrice(points)
	maximum := maxPrice(points)
	if minimum <= 0 {
		return evaluation(GuideSemilog, EvaluationNotObservable, 0, "semilog scale requires positive prices")
	}
	rangeMultiple := maximum / minimum
	if rangeMultiple < 2 {
		return evaluation(GuideSemilog, EvaluationNotApplicable, rangeMultiple, "arithmetic and semilog differ materially only over large ranges")
	}
	return evaluation(GuideSemilog, EvaluationPass, math.Log(rangeMultiple), "arithmetic and semilog geometry both evaluated")
}

func fibonacciTimeEvaluation(points []Pivot) RuleEvaluation {
	if len(points) < 4 {
		return evaluation(GuideFibonacciTime, EvaluationNotApplicable, 0, "at least three completed durations required")
	}
	durations := make([]float64, 0, len(points)-1)
	for index := 0; index+1 < len(points); index++ {
		duration := points[index+1].BarIndex - points[index].BarIndex
		if duration > 0 {
			durations = append(durations, float64(duration))
		}
	}
	if len(durations) < 3 {
		return evaluation(GuideFibonacciTime, EvaluationNotObservable, 0, "insufficient positive bar durations")
	}
	bestDistance := math.Inf(1)
	bestRatio := 0.0
	for index := 1; index < len(durations); index++ {
		ratio := durations[index] / nonZero(durations[index-1])
		_, distance := nearestRatio(ratio, []float64{0.382, 0.5, 0.618, 1, 1.618, 2.618})
		if distance < bestDistance {
			bestDistance = distance
			bestRatio = ratio
		}
	}
	status := EvaluationFail
	if bestDistance <= 0.08 {
		status = EvaluationPass
	}
	return evaluation(GuideFibonacciTime, status, bestRatio, "supporting duration ratio near a Fibonacci relation")
}

func completeRuleAudit(pattern PatternType, mode WaveMode, points []Pivot, evaluations []RuleEvaluation) []RuleEvaluation {
	result := cloneEvaluations(evaluations)
	appendMissing := func(item RuleEvaluation) {
		for _, existing := range result {
			if existing.RuleID == item.RuleID {
				return
			}
		}
		result = append(result, item)
	}
	appendMissing(evaluation(
		RuleOrthodoxEndpoints, EvaluationPass, float64(len(points)),
		"all measurements use stored orthodox pivots",
	))
	appendMissing(rightLookEvaluation(points))
	appendMissing(semilogEvaluation(points))
	appendMissing(fibonacciTimeEvaluation(points))
	appendMissing(evaluation(
		GuidePreviousFourth, EvaluationNotApplicable, 0,
		"activated by the target engine when a lower-degree fourth is observable",
	))
	appendMissing(evaluation(
		GuideFifthExtensionRetrace, EvaluationNotApplicable, 0,
		"activated after an observable fifth-wave extension completes",
	))
	if mode == ModeMotive {
		appendMissing(volumeNotObservable())
		appendMissing(personalityNotObservable())
	}
	if pattern == PatternTruncatedImpulse {
		appendMissing(truncationStrengthEvaluation(segmentLengths(points)))
	}
	return result
}
