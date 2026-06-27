package wave

import (
	"fmt"
	"math"
	"sort"

	"WaveSight/internal/market"
)

type Engine struct {
	parser  *Parser
	targets *TargetEngine
}

func NewEngine() *Engine {
	return &Engine{
		parser:  NewParser(2_000),
		targets: NewTargetEngine(),
	}
}

func (e *Engine) Analyze(input AnalyzeInput) AnalysisResult {
	if input.MaxScenarios < 1 || input.MaxScenarios > 5 {
		input.MaxScenarios = 5
	}
	quality := assessDataQuality(input.Candles, input.Timeframe)
	lattice := BuildPivotLattice(input.Candles)
	for _, branch := range lattice.Branches {
		for _, level := range branch.Levels {
			for _, pivot := range level {
				if pivot.State == PivotAmbiguous {
					quality.AmbiguousPivotCount++
				}
			}
			break
		}
	}

	nodes := e.parser.Parse(lattice)
	for index := range nodes {
		nodes[index] = applyMarketEvidence(nodes[index], input.Candles)
	}
	candidates := currentCandidates(nodes, len(input.Candles))
	if len(candidates) == 0 {
		quality.Warnings = append(quality.Warnings, "No fully rule-valid current Elliott structure was observable.")
		return AnalysisResult{
			DataQuality: quality,
			Scenarios: []Scenario{{
				ID: "indeterminate", Rank: 1, Status: ScenarioIndeterminate,
				CurrentPosition: "No current count satisfies the observable structure requirements.",
				Root: WaveNode{
					ID: "indeterminate-root", Status: StatusIndeterminate,
					Label: "Indeterminate", Degree: DegreeObservableLeaf,
				},
			}},
			FutureBars: append([]int64(nil), input.FutureBars...),
		}
	}

	if len(candidates) > input.MaxScenarios {
		candidates = candidates[:input.MaxScenarios]
	}
	scenarios := make([]Scenario, 0, len(candidates))
	for index, root := range candidates {
		invalidations := buildInvalidations(root)
		status := ScenarioAlternate
		if index == 0 {
			status = ScenarioPreferred
		}
		scenario := Scenario{
			ID:   fmt.Sprintf("scenario-%d-%s", index+1, root.ID),
			Rank: index + 1, Status: status, Bias: scenarioBias(root),
			CurrentPosition: currentPosition(root), Conformance: root.Conformance,
			Invalidations: invalidations, Root: root,
		}
		scenario.TargetLadder = e.targets.Build(root, input.Candles, input.TickSize, input.FutureBars, invalidations)
		scenarios = append(scenarios, scenario)
	}

	return AnalysisResult{
		DataQuality: quality,
		Scenarios:   scenarios,
		FutureBars:  append([]int64(nil), input.FutureBars...),
	}
}

func currentCandidates(nodes []WaveNode, candleCount int) []WaveNode {
	if len(nodes) == 0 {
		return nil
	}
	lastIndex := candleCount - 1
	relevanceWindow := maxInt(8, candleCount/20)
	result := make([]WaveNode, 0, 64)
	for _, node := range nodes {
		if node.Conformance.HardRulesFailed > 0 || node.Conformance.Score <= 0 {
			continue
		}
		if node.OrthodoxEnd.BarIndex < lastIndex-relevanceWindow {
			continue
		}
		result = append(result, node)
	}
	sort.SliceStable(result, func(i, j int) bool {
		left := result[i]
		right := result[j]
		leftCurrent := lastIndex - left.OrthodoxEnd.BarIndex
		rightCurrent := lastIndex - right.OrthodoxEnd.BarIndex
		if left.Status != right.Status {
			return left.Status == StatusDeveloping
		}
		if leftCurrent != rightCurrent {
			return leftCurrent < rightCurrent
		}
		if left.Level != right.Level {
			return left.Level > right.Level
		}
		if left.Conformance.StructuralCoverage != right.Conformance.StructuralCoverage {
			return left.Conformance.StructuralCoverage > right.Conformance.StructuralCoverage
		}
		if left.Conformance.GuidelinesPassed != right.Conformance.GuidelinesPassed {
			return left.Conformance.GuidelinesPassed > right.Conformance.GuidelinesPassed
		}
		if left.Conformance.RatioConfluences != right.Conformance.RatioConfluences {
			return left.Conformance.RatioConfluences > right.Conformance.RatioConfluences
		}
		if left.Conformance.Score != right.Conformance.Score {
			return left.Conformance.Score > right.Conformance.Score
		}
		return left.ID < right.ID
	})

	seen := make(map[string]struct{})
	deduplicated := result[:0]
	for _, node := range result {
		key := fmt.Sprintf("%s:%d:%d:%s", node.Pattern, node.OrthodoxStart.BarIndex, node.OrthodoxEnd.BarIndex, node.Direction)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		deduplicated = append(deduplicated, node)
	}
	return deduplicated
}

func buildInvalidations(node WaveNode) []Invalidation {
	if len(node.Pivots) < 2 {
		return nil
	}
	points := node.Pivots
	result := make([]Invalidation, 0, 3)
	addPrice := func(id string, price float64, description string) {
		if price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0) {
			result = append(result, Invalidation{
				ID: id, Kind: InvalidationPrice, Price: price, Description: description,
			})
		}
	}
	addRule := func(id, ruleID, description string) {
		result = append(result, Invalidation{
			ID: id, Kind: InvalidationRule, RuleID: ruleID, Description: description,
		})
	}

	switch node.Pattern {
	case PatternDevelopingImpulseW2, PatternDevelopingImpulseW3:
		addPrice("wave-1-origin", points[0].Price, "Wave 2 may not move beyond the origin of wave 1.")
	case PatternDevelopingImpulseW4:
		addPrice("wave-1-territory", points[1].Price, "An impulse wave 4 may not enter wave 1 price territory.")
		addPrice("wave-3-origin", points[2].Price, "Wave 4 may not retrace more than 100% of wave 3.")
	case PatternDevelopingImpulseW5:
		addPrice("wave-4-extreme", points[4].Price, "The projected fifth wave is invalid if price reverses beyond wave 4.")
	case PatternDevelopingZigzagC:
		addPrice("wave-a-origin", points[0].Price, "The current B-wave interpretation fails beyond the correction origin.")
	case PatternDevelopingFlatC:
		addRule(
			"flat-c-five-required", RuleFlatSubdivision,
			"The flat interpretation requires wave C to complete as a motive five.",
		)
	case PatternDevelopingTriangleD, PatternDevelopingTriangleE:
		addPrice("triangle-boundary-high", maxPrice(points), "A decisive break invalidates the active triangle boundary.")
		addPrice("triangle-boundary-low", minPrice(points), "A decisive break invalidates the active triangle boundary.")
	default:
		addPrice("pattern-origin", points[0].Price, "A full move through the orthodox origin invalidates continuation from this pattern.")
	}
	return result
}

func scenarioBias(node WaveNode) Direction {
	if node.Status == StatusCompleted {
		return node.Direction.Opposite()
	}
	switch node.Pattern {
	case PatternDevelopingImpulseW2, PatternDevelopingImpulseW4:
		return node.Direction.Opposite()
	case PatternDevelopingTriangleD, PatternDevelopingTriangleE:
		return directionForNextLeg(node.Pivots)
	default:
		return node.Direction
	}
}

func currentPosition(node WaveNode) string {
	switch node.Pattern {
	case PatternDevelopingImpulseW2:
		return fmt.Sprintf("%s: wave 1 complete; expecting wave 2", node.Degree)
	case PatternDevelopingImpulseW3:
		return fmt.Sprintf("%s: wave 2 complete; expecting wave 3", node.Degree)
	case PatternDevelopingImpulseW4:
		return fmt.Sprintf("%s: wave 3 complete; expecting wave 4", node.Degree)
	case PatternDevelopingImpulseW5:
		return fmt.Sprintf("%s: wave 4 complete; expecting wave 5", node.Degree)
	case PatternDevelopingZigzagC:
		return fmt.Sprintf("%s: zigzag A-B complete; expecting C", node.Degree)
	case PatternDevelopingFlatC:
		return fmt.Sprintf("%s: flat A-B complete; expecting C", node.Degree)
	case PatternDevelopingTriangleD:
		return fmt.Sprintf("%s: triangle A-B-C complete; expecting D", node.Degree)
	case PatternDevelopingTriangleE:
		return fmt.Sprintf("%s: triangle A-B-C-D complete; expecting E", node.Degree)
	default:
		return fmt.Sprintf("%s %s complete; expecting a %s phase", node.Degree, node.Label, node.Mode)
	}
}

func assessDataQuality(candles []market.Candle, timeframe market.Timeframe) DataQuality {
	quality := DataQuality{CandleCount: len(candles)}
	if len(candles) == 0 {
		quality.Warnings = []string{"No normalized candles were available."}
		return quality
	}
	quality.FirstTime = candles[0].Time
	quality.LastTime = candles[len(candles)-1].Time
	expected := timeframe.Duration().Seconds()
	for i := 1; i < len(candles); i++ {
		gap := float64(candles[i].Time - candles[i-1].Time)
		if expected > 0 && gap > expected*10 {
			quality.MissingIntervals++
		}
	}
	if quality.MissingIntervals > 0 {
		quality.Warnings = append(quality.Warnings, "Large data gaps were observed; rule evidence may be less complete.")
	}
	if len(candles) < 200 {
		quality.Warnings = append(quality.Warnings, "Fewer than 200 bars limits observable wave degree and subdivision depth.")
	}
	return quality
}
