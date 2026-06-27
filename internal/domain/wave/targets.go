package wave

import (
	"fmt"
	"math"
	"sort"

	"WaveSight/internal/market"
)

type TargetEngine struct{}

func NewTargetEngine() *TargetEngine {
	return &TargetEngine{}
}

func (e *TargetEngine) Build(node WaveNode, candles []market.Candle, tickSize float64, futureBars []int64, invalidations []Invalidation) []TargetZone {
	if len(node.Pivots) < 2 {
		return nil
	}
	if tickSize <= 0 {
		tickSize = 0.01
	}
	atr := market.ATR(candles, 14)
	anchorATR := tickSize * 4
	if len(atr) > 0 {
		index := node.OrthodoxEnd.BarIndex
		if index >= len(atr) {
			index = len(atr) - 1
		}
		if index >= 0 && atr[index] > 0 {
			anchorATR = atr[index]
		}
	}
	uncertainty := math.Max(2*tickSize, 0.25*anchorATR)
	levels, label, condition := projectedLevels(node, uncertainty)
	if len(levels) == 0 {
		return nil
	}

	zones := clusterTargetLevels(levels, label, condition, invalidations)
	if node.Pattern == PatternDevelopingImpulseW5 {
		if channel, ok := buildChannelZone(node, uncertainty, invalidations, len(futureBars)); ok {
			zones = append(zones, channel)
		}
	}
	window := buildTimeWindow(node, futureBars)
	for i := range zones {
		if node.Status == StatusDeveloping {
			zones[i].Status = TargetActive
		}
		zones[i].TimeWindow = window
	}
	return zones
}

func buildChannelZone(node WaveNode, uncertainty float64, invalidations []Invalidation, futureCount int) (TargetZone, bool) {
	points := node.Pivots
	if len(points) < 5 || points[4].BarIndex == points[2].BarIndex || futureCount == 0 {
		return TargetZone{}, false
	}
	lineSlope := (points[4].Price - points[2].Price) /
		float64(points[4].BarIndex-points[2].BarIndex)
	endOffset := minInt(50, futureCount)
	priceAt := func(offset int) float64 {
		bar := points[4].BarIndex + offset
		return points[3].Price + lineSlope*float64(bar-points[3].BarIndex)
	}
	startPrice := priceAt(1)
	endPrice := priceAt(endOffset)
	if startPrice <= 0 || endPrice <= 0 ||
		math.IsNaN(startPrice) || math.IsNaN(endPrice) ||
		math.IsInf(startPrice, 0) || math.IsInf(endPrice, 0) {
		return TargetZone{}, false
	}
	invalidationIDs := make([]string, 0, len(invalidations))
	for _, invalidation := range invalidations {
		invalidationIDs = append(invalidationIDs, invalidation.ID)
	}
	minimum := math.Min(startPrice, endPrice) - uncertainty
	maximum := math.Max(startPrice, endPrice) + uncertainty
	return TargetZone{
		ID: "target-W5-channel", WaveLabel: "W5 channel",
		Status:    TargetConditional,
		Condition: "Wave 4 remains valid and price continues within the Elliott channel",
		MinPrice:  minimum, MaxPrice: maximum,
		Levels: []TargetLevel{{
			Price: startPrice, Relation: "parallel through W3 from the W2-W4 baseline",
			Family: "CHANNEL", Source: "EWP pp.38-40; WaveRatios p.5",
			Uncertainty: uncertainty,
		}},
		Confluence: ConfluenceLine, Geometry: GeometryChannelBand,
		Points: []GeometryPoint{
			{BarOffset: 1, Price: startPrice - uncertainty},
			{BarOffset: endOffset, Price: endPrice - uncertainty},
			{BarOffset: endOffset, Price: endPrice + uncertainty},
			{BarOffset: 1, Price: startPrice + uncertainty},
		},
		InvalidationIDs: invalidationIDs,
	}, true
}

func projectedLevels(node WaveNode, uncertainty float64) ([]TargetLevel, string, string) {
	points := node.Pivots
	direction := node.Direction
	levels := make([]TargetLevel, 0, 12)
	add := func(price float64, relation, family, source string) {
		if price <= 0 || math.IsNaN(price) || math.IsInf(price, 0) {
			return
		}
		levels = append(levels, TargetLevel{
			Price: price, Relation: relation, Family: family, Source: source,
			Uncertainty: uncertainty,
		})
	}

	switch node.Pattern {
	case PatternDevelopingImpulseW2:
		length1 := math.Abs(points[1].Price - points[0].Price)
		for _, ratio := range []float64{0.14, 0.25, 0.382, 0.5, 0.618} {
			source := "EWP p.72; WaveRatios pp.2,6"
			relation := fmt.Sprintf("%.3f retracement of W1", ratio)
			if ratio == 0.14 || ratio == 0.25 {
				source = "WaveRatios low-priority relation"
				relation += " (low priority)"
			}
			add(project(points[1].Price, direction.Opposite(), length1*ratio),
				relation, "W2_RETRACEMENT", source)
		}
		return levels, "W2", "Wave 1 remains the operative start"
	case PatternDevelopingImpulseW3:
		length1 := math.Abs(points[1].Price - points[0].Price)
		for _, ratio := range []float64{1.618, 2.618, 4.236, 4.25, 6.85} {
			source := "EWP pp.72-73; WaveRatios pp.3,8"
			relation := fmt.Sprintf("%.3f × W1 from W2", ratio)
			if ratio == 6.85 {
				source = "WaveRatios low-priority relation"
				relation += " (rare)"
			}
			add(project(points[2].Price, direction, length1*ratio),
				relation, "W3_EXTENSION", source)
		}
		return levels, "W3", "Wave 2 remains above/below the wave 1 origin"
	case PatternDevelopingImpulseW4:
		length3 := math.Abs(points[3].Price - points[2].Price)
		for _, ratio := range []float64{0.14, 0.236, 0.24, 0.25, 0.382, 0.5} {
			source := "EWP p.72; WaveRatios pp.3,10"
			relation := fmt.Sprintf("%.3f retracement of W3", ratio)
			if ratio == 0.14 || ratio == 0.25 {
				source = "WaveRatios low-priority relation"
				relation += " (low priority)"
			}
			add(project(points[3].Price, direction.Opposite(), length3*ratio),
				relation, "W4_RETRACEMENT", source)
		}
		if childPrice, ok := previousFourthPrice(node); ok {
			add(childPrice, "previous fourth-wave area", "PREVIOUS_FOURTH", "EWP pp.35-36")
		}
		return levels, "W4", "Wave 3 is complete and wave 4 stays within motive limits"
	case PatternDevelopingImpulseW5:
		length1 := math.Abs(points[1].Price - points[0].Price)
		zeroThree := math.Abs(points[3].Price - points[0].Price)
		for _, ratio := range []float64{0.618, 1} {
			add(project(points[4].Price, direction, length1*ratio),
				fmt.Sprintf("%.3f × W1 from W4", ratio), "W5_VS_W1", "EWP pp.72-73; WaveRatios p.4")
		}
		for _, ratio := range []float64{0.618, 1, 1.618} {
			add(project(points[4].Price, direction, zeroThree*ratio),
				fmt.Sprintf("%.3f × 0→3 from W4", ratio), "W5_VS_ZERO_THREE", "EWP p.73; WaveRatios pp.12-13")
		}
		if channelPrice, ok := projectedChannelPrice(points); ok {
			add(channelPrice, "parallel channel from W3", "CHANNEL", "EWP pp.38-40; WaveRatios p.5")
		}
		return levels, "W5", "Wave 4 remains valid and the fifth wave is underway"
	case PatternDevelopingZigzagC:
		lengthA := math.Abs(points[1].Price - points[0].Price)
		for _, ratio := range []float64{0.618, 1, 1.618, 2.618} {
			add(project(points[2].Price, direction, lengthA*ratio),
				fmt.Sprintf("%.3f × A from B", ratio), "C_VS_A", "EWP p.74")
		}
		return levels, "C", "Wave B remains corrective"
	case PatternDevelopingFlatC:
		lengthA := math.Abs(points[1].Price - points[0].Price)
		lengthB := math.Abs(points[2].Price - points[1].Price)
		add(project(points[2].Price, direction, lengthA),
			"1.000 × A from B", "FLAT_C_VS_A", "EWP pp.24-27,74")
		if lengthB >= lengthA {
			for _, ratio := range []float64{1.618, 2.618} {
				add(project(points[2].Price, direction, lengthA*ratio),
					fmt.Sprintf("%.3f × A from expanded B", ratio),
					"EXPANDED_FLAT_C", "EWP p.74")
			}
		}
		return levels, "C", "Flat B remains valid and C subdivides as a motive five"
	case PatternDevelopingTriangleD, PatternDevelopingTriangleE:
		reference := math.Abs(points[len(points)-2].Price - points[len(points)-3].Price)
		for _, ratio := range []float64{0.618, 1.618} {
			add(project(points[len(points)-1].Price, directionForNextLeg(points), reference*ratio),
				fmt.Sprintf("%.3f × alternate triangle leg", ratio), "TRIANGLE_ALTERNATE", "EWP p.75")
		}
		label := "D"
		if node.Pattern == PatternDevelopingTriangleE {
			label = "E"
		}
		return levels, label, "Triangle boundaries and corrective subdivisions remain valid"
	case PatternImpulse, PatternTruncatedImpulse, PatternLeadingDiagonal, PatternEndingDiagonal:
		total := math.Abs(points[len(points)-1].Price - points[0].Price)
		for _, ratio := range []float64{0.382, 0.5, 0.618} {
			add(project(points[len(points)-1].Price, direction.Opposite(), total*ratio),
				fmt.Sprintf("%.3f retracement of completed motive", ratio), "POST_MOTIVE_RETRACE", "EWP pp.35-37,72")
		}
		if len(points) >= 5 {
			add(points[4].Price, "preceding fourth-wave area", "PREVIOUS_FOURTH", "EWP pp.35-36")
		}
		if len(points) >= 6 && len(node.Children) >= 5 {
			lengths := segmentLengths(points)
			fifth := node.Children[4]
			if lengths[4] >= 1.618*math.Max(lengths[0], lengths[2]) && len(fifth.Pivots) >= 3 {
				add(
					fifth.Pivots[2].Price,
					"wave 2 area of the fifth-wave extension",
					"FIFTH_EXTENSION_W2",
					"EWP p.37",
				)
			}
		}
		if node.Pattern == PatternTruncatedImpulse || node.Pattern == PatternEndingDiagonal {
			add(points[0].Price, "full reversal toward pattern origin", "EXHAUSTION_REVERSAL", "EWP p.19")
		}
		return levels, "A/2", "The completed motive is followed by a corrective phase"
	case PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding:
		width := maxPrice(points) - minPrice(points)
		add(project(points[len(points)-1].Price, node.Direction.Opposite(), width),
			"widest triangle width", "TRIANGLE_THRUST", "EWP p.29")
		return levels, "Thrust", "Wave E is complete and the triangle remains valid"
	case PatternZigzag, PatternFlatRegular, PatternFlatExpanded, PatternFlatRunning,
		PatternDoubleZigzag, PatternTripleZigzag, PatternDoubleThree, PatternTripleThree:
		total := math.Abs(points[len(points)-1].Price - points[0].Price)
		for _, ratio := range []float64{0.618, 1} {
			add(project(points[len(points)-1].Price, node.Direction.Opposite(), total*ratio),
				fmt.Sprintf("%.3f reversal of correction", ratio), "POST_CORRECTION", "EWP wave progression")
		}
		return levels, "1/A", "The correction is complete and a new actionary wave begins"
	default:
		return nil, "", ""
	}
}

func clusterTargetLevels(levels []TargetLevel, label, condition string, invalidations []Invalidation) []TargetZone {
	sort.Slice(levels, func(i, j int) bool { return levels[i].Price < levels[j].Price })
	type cluster struct {
		levels []TargetLevel
		min    float64
		max    float64
	}
	clusters := make([]cluster, 0, len(levels))
	for _, level := range levels {
		minimum := level.Price - level.Uncertainty
		maximum := level.Price + level.Uncertainty
		if len(clusters) == 0 || minimum > clusters[len(clusters)-1].max {
			clusters = append(clusters, cluster{levels: []TargetLevel{level}, min: minimum, max: maximum})
			continue
		}
		current := &clusters[len(clusters)-1]
		current.levels = append(current.levels, level)
		current.min = math.Max(current.min, minimum)
		current.max = math.Min(current.max, maximum)
		if current.min > current.max {
			current.min = minimum
			current.max = maximum
		}
	}

	invalidationIDs := make([]string, 0, len(invalidations))
	for _, invalidation := range invalidations {
		invalidationIDs = append(invalidationIDs, invalidation.ID)
	}

	result := make([]TargetZone, 0, len(clusters))
	for index, group := range clusters {
		families := make(map[string]struct{})
		for _, level := range group.levels {
			families[level.Family] = struct{}{}
		}
		grade := ConfluenceLine
		if len(families) >= 3 {
			grade = ConfluenceHigh
		} else if len(families) >= 2 {
			grade = ConfluenceMedium
		}
		minimum, maximum := group.min, group.max
		if grade == ConfluenceLine {
			minimum = group.levels[0].Price
			maximum = group.levels[0].Price
		}
		result = append(result, TargetZone{
			ID: fmt.Sprintf("target-%s-%d", label, index+1), WaveLabel: label,
			Status: TargetConditional, Condition: condition, MinPrice: minimum,
			MaxPrice: maximum, Levels: group.levels, Confluence: grade,
			Geometry: GeometryHorizontalBand, InvalidationIDs: invalidationIDs,
		})
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Confluence != result[j].Confluence {
			return confluenceOrder(result[i].Confluence) > confluenceOrder(result[j].Confluence)
		}
		return result[i].MinPrice < result[j].MinPrice
	})
	return result
}

func buildTimeWindow(node WaveNode, futureBars []int64) *TimeWindow {
	if len(node.Pivots) < 4 || len(futureBars) == 0 {
		return nil
	}
	durations := make([]int, len(node.Pivots)-1)
	for i := range durations {
		durations[i] = node.Pivots[i+1].BarIndex - node.Pivots[i].BarIndex
	}
	last := durations[len(durations)-1]
	alternate := durations[0]
	if len(durations) >= 4 {
		alternate = durations[len(durations)-3]
	}
	firstEstimate := int(math.Round(float64(last) * 0.618))
	secondEstimate := alternate
	tolerance := maxInt(2, int(math.Round(float64(maxInt(firstEstimate, secondEstimate))*0.10)))
	if absInt(firstEstimate-secondEstimate) > tolerance {
		return nil
	}
	start := maxInt(1, minInt(firstEstimate, secondEstimate)-1)
	end := maxInt(start, maxInt(firstEstimate, secondEstimate)+1)
	if start > len(futureBars) {
		return nil
	}
	if end > len(futureBars) {
		end = len(futureBars)
	}
	return &TimeWindow{
		StartBarOffset: start, EndBarOffset: end,
		StartTime: futureBars[start-1], EndTime: futureBars[end-1],
		Evidence: []string{"0.618 × preceding wave duration", "equality with alternate wave duration"},
	}
}

func project(origin float64, direction Direction, distance float64) float64 {
	if direction == DirectionBullish {
		return origin + distance
	}
	return origin - distance
}

func directionForNextLeg(points []Pivot) Direction {
	direction, ok := directionBetween(points[len(points)-2], points[len(points)-1])
	if !ok {
		return DirectionBullish
	}
	return direction.Opposite()
}

func previousFourthPrice(node WaveNode) (float64, bool) {
	if len(node.Children) < 3 {
		return 0, false
	}
	third := node.Children[2]
	if len(third.Pivots) < 5 {
		return 0, false
	}
	return third.Pivots[len(third.Pivots)-2].Price, true
}

func projectedChannelPrice(points []Pivot) (float64, bool) {
	if len(points) < 5 || points[4].BarIndex == points[2].BarIndex {
		return 0, false
	}
	m := (points[4].Price - points[2].Price) / float64(points[4].BarIndex-points[2].BarIndex)
	horizon := maxInt(1, points[3].BarIndex-points[2].BarIndex)
	targetBar := points[4].BarIndex + horizon
	return points[3].Price + m*float64(targetBar-points[3].BarIndex), true
}

func confluenceOrder(grade ConfluenceGrade) int {
	switch grade {
	case ConfluenceHigh:
		return 3
	case ConfluenceMedium:
		return 2
	default:
		return 1
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
