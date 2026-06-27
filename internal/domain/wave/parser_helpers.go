package wave

import "math"

func alternates(points []Pivot) bool {
	for i := 1; i < len(points); i++ {
		if points[i].BarIndex < points[i-1].BarIndex || points[i].Kind == points[i-1].Kind {
			return false
		}
	}
	return true
}

func directionBetween(start, end Pivot) (Direction, bool) {
	if end.Price > start.Price {
		return DirectionBullish, true
	}
	if end.Price < start.Price {
		return DirectionBearish, true
	}
	return "", false
}

func continuesMotiveShape(points []Pivot, direction Direction) bool {
	if len(points) != 6 {
		return false
	}
	for i := 0; i+1 < len(points); i++ {
		expected := direction
		if i%2 == 1 {
			expected = direction.Opposite()
		}
		actual, ok := directionBetween(points[i], points[i+1])
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func continuesMotivePartial(points []Pivot, direction Direction) bool {
	for i := 0; i+1 < len(points); i++ {
		expected := direction
		if i%2 == 1 {
			expected = direction.Opposite()
		}
		actual, ok := directionBetween(points[i], points[i+1])
		if !ok || actual != expected {
			return false
		}
	}
	if direction == DirectionBullish {
		return points[2].Price >= points[0].Price && points[3].Price > points[1].Price
	}
	return points[2].Price <= points[0].Price && points[3].Price < points[1].Price
}

func continuesCorrectionShape(points []Pivot, direction Direction) bool {
	for i := 0; i+1 < len(points); i++ {
		expected := direction
		if i%2 == 1 {
			expected = direction.Opposite()
		}
		actual, ok := directionBetween(points[i], points[i+1])
		if !ok || actual != expected {
			return false
		}
	}
	return true
}

func segmentLengths(points []Pivot) []float64 {
	lengths := make([]float64, len(points)-1)
	for i := range lengths {
		lengths[i] = math.Abs(points[i+1].Price - points[i].Price)
	}
	return lengths
}

func wave4OverlapsWave1(points []Pivot, direction Direction) bool {
	if len(points) < 5 {
		return false
	}
	if direction == DirectionBullish {
		return points[4].Price <= points[1].Price
	}
	return points[4].Price >= points[1].Price
}

func wave5Truncated(points []Pivot, direction Direction) bool {
	if len(points) < 6 {
		return false
	}
	if direction == DirectionBullish {
		return points[5].Price <= points[3].Price
	}
	return points[5].Price >= points[3].Price
}

func convergingDiagonal(lengths []float64) bool {
	return len(lengths) == 5 &&
		lengths[0] > lengths[2] && lengths[2] > lengths[4] &&
		lengths[1] > lengths[3]
}

func priceBeyond(value, reference float64, direction Direction) bool {
	if direction == DirectionBullish {
		return value > reference
	}
	return value < reference
}

func slope(start, end Pivot) float64 {
	delta := end.BarIndex - start.BarIndex
	if delta == 0 {
		return 0
	}
	return (end.Price - start.Price) / float64(delta)
}

func minPrice(points []Pivot) float64 {
	result := math.Inf(1)
	for _, point := range points {
		result = math.Min(result, point.Price)
	}
	return result
}

func maxPrice(points []Pivot) float64 {
	result := math.Inf(-1)
	for _, point := range points {
		result = math.Max(result, point.Price)
	}
	return result
}

func boolStatus(ok bool) EvaluationStatus {
	if ok {
		return EvaluationPass
	}
	return EvaluationFail
}

func hasHardFailure(evaluations []RuleEvaluation) bool {
	for _, item := range evaluations {
		if item.Class == RuleHard && item.Status == EvaluationFail {
			return true
		}
	}
	return false
}

func cloneEvaluations(input []RuleEvaluation) []RuleEvaluation {
	result := make([]RuleEvaluation, len(input))
	copy(result, input)
	return result
}

func nonZero(value float64) float64 {
	if math.Abs(value) < 1e-12 {
		return 1e-12
	}
	return value
}

func clamp01(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}

func isFlat(pattern PatternType) bool {
	return pattern == PatternFlatRegular || pattern == PatternFlatExpanded || pattern == PatternFlatRunning
}

func isTriangle(pattern PatternType) bool {
	return pattern == PatternTriangleContracting || pattern == PatternTriangleAscending ||
		pattern == PatternTriangleDescending || pattern == PatternTriangleRunning ||
		pattern == PatternTriangleExpanding
}

func patternLabel(pattern PatternType) string {
	switch pattern {
	case PatternImpulse:
		return "Impulse"
	case PatternLeadingDiagonal:
		return "Leading diagonal"
	case PatternEndingDiagonal:
		return "Ending diagonal"
	case PatternTruncatedImpulse:
		return "Impulse with truncated fifth"
	case PatternZigzag:
		return "Zigzag"
	case PatternDoubleZigzag:
		return "Double zigzag"
	case PatternTripleZigzag:
		return "Triple zigzag"
	case PatternFlatRegular:
		return "Regular flat"
	case PatternFlatExpanded:
		return "Expanded flat"
	case PatternFlatRunning:
		return "Running flat"
	case PatternTriangleContracting:
		return "Contracting triangle"
	case PatternTriangleAscending:
		return "Ascending triangle"
	case PatternTriangleDescending:
		return "Descending triangle"
	case PatternTriangleRunning:
		return "Running triangle"
	case PatternTriangleExpanding:
		return "Expanding triangle"
	case PatternDoubleThree:
		return "Double three"
	case PatternTripleThree:
		return "Triple three"
	case PatternDevelopingImpulseW2:
		return "Developing impulse: expecting wave 2"
	case PatternDevelopingImpulseW3:
		return "Developing impulse: expecting wave 3"
	case PatternDevelopingImpulseW4:
		return "Developing impulse: expecting wave 4"
	case PatternDevelopingImpulseW5:
		return "Developing impulse: expecting wave 5"
	case PatternDevelopingZigzagC:
		return "Developing zigzag: expecting wave C"
	case PatternDevelopingFlatC:
		return "Developing flat: expecting wave C"
	case PatternDevelopingTriangleD:
		return "Developing triangle: expecting wave D"
	case PatternDevelopingTriangleE:
		return "Developing triangle: expecting wave E"
	default:
		return string(pattern)
	}
}
