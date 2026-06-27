package wave

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"runtime"
	"sort"
	"strings"
)

type Parser struct {
	maxNodesPerLevel int
	workers          int
}

func NewParser(maxNodesPerLevel int) *Parser {
	if maxNodesPerLevel < 100 {
		maxNodesPerLevel = 2_000
	}
	return &Parser{
		maxNodesPerLevel: maxNodesPerLevel,
		workers:          maxInt(1, minInt(4, runtime.GOMAXPROCS(0))),
	}
}

func (p *Parser) Parse(lattice PivotLattice) []WaveNode {
	if len(lattice.Branches) == 0 {
		return nil
	}
	type branchJob struct {
		index  int
		branch PivotBranch
	}
	type branchResult struct {
		index int
		nodes []WaveNode
	}
	workerCount := minInt(p.workers, len(lattice.Branches))
	jobs := make(chan branchJob)
	results := make(chan branchResult, len(lattice.Branches))
	for worker := 0; worker < workerCount; worker++ {
		go func() {
			for job := range jobs {
				results <- branchResult{index: job.index, nodes: p.parseBranch(job.branch)}
			}
		}()
	}
	go func() {
		for index, branch := range lattice.Branches {
			jobs <- branchJob{index: index, branch: branch}
		}
		close(jobs)
	}()

	byBranch := make([][]WaveNode, len(lattice.Branches))
	for range lattice.Branches {
		result := <-results
		byBranch[result.index] = result.nodes
	}

	all := make([]WaveNode, 0, 1024)
	seen := make(map[string]struct{})
	for _, nodes := range byBranch {
		for _, node := range nodes {
			if _, exists := seen[node.ID]; exists {
				continue
			}
			seen[node.ID] = struct{}{}
			all = append(all, node)
		}
	}
	sortNodes(all)
	return all
}

func (p *Parser) parseBranch(branch PivotBranch) []WaveNode {
	all := make([]WaveNode, 0, 512)
	nodesByLevel := make(map[int][]WaveNode, len(branch.Levels))
	for level, pivots := range branch.Levels {
		levelNodes := p.parseLevel(level, pivots, nodesByLevel[level-1])
		levelNodes = append(levelNodes, parseCombinations(levelNodes, level)...)
		levelNodes = deduplicateNodes(levelNodes)
		sortNodes(levelNodes)
		if len(levelNodes) > p.maxNodesPerLevel {
			levelNodes = levelNodes[:p.maxNodesPerLevel]
		}
		nodesByLevel[level] = levelNodes
		all = append(all, levelNodes...)
	}
	return all
}

func (p *Parser) parseLevel(level int, pivots []Pivot, lower []WaveNode) []WaveNode {
	if len(pivots) < 3 {
		return nil
	}
	result := make([]WaveNode, 0, len(pivots)*3)

	for i := 0; i+5 < len(pivots); i++ {
		window := append([]Pivot(nil), pivots[i:i+6]...)
		result = append(result, parseMotiveWindow(level, window, lower)...)
		if triangle, ok := parseTriangleWindow(level, window, lower); ok {
			result = append(result, triangle)
		}
	}
	for i := 0; i+3 < len(pivots); i++ {
		window := append([]Pivot(nil), pivots[i:i+4]...)
		result = append(result, parseABCWindow(level, window, lower)...)
		if developing, ok := parseDevelopingImpulseW4(level, window, lower); ok {
			result = append(result, developing)
		}
		if developing, ok := parseDevelopingTriangleD(level, window, lower); ok {
			result = append(result, developing)
		}
	}
	for i := 0; i+4 < len(pivots); i++ {
		window := append([]Pivot(nil), pivots[i:i+5]...)
		if developing, ok := parseDevelopingImpulseW5(level, window, lower); ok {
			result = append(result, developing)
		}
		if developing, ok := parseDevelopingTriangleE(level, window, lower); ok {
			result = append(result, developing)
		}
	}
	for i := 0; i+2 < len(pivots); i++ {
		window := append([]Pivot(nil), pivots[i:i+3]...)
		if developing, ok := parseDevelopingImpulseW3(level, window, lower); ok {
			result = append(result, developing)
		}
		if developing, ok := parseDevelopingZigzagC(level, window, lower); ok {
			result = append(result, developing)
		}
		if developing, ok := parseDevelopingFlatC(level, window, lower); ok {
			result = append(result, developing)
		}
	}
	for i := 0; i+1 < len(pivots); i++ {
		window := append([]Pivot(nil), pivots[i:i+2]...)
		if developing, ok := parseDevelopingImpulseW2(level, window, lower); ok {
			result = append(result, developing)
		}
	}
	return result
}

func parseMotiveWindow(level int, points []Pivot, lower []WaveNode) []WaveNode {
	if !alternates(points) {
		return nil
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok || !continuesMotiveShape(points, direction) {
		return nil
	}

	lengths := segmentLengths(points)
	evaluations := motiveCoreEvaluations(points, direction, lengths)
	if hasHardFailure(evaluations) {
		return nil
	}

	overlap := wave4OverlapsWave1(points, direction)
	truncated := wave5Truncated(points, direction)
	result := make([]WaveNode, 0, 3)

	if !overlap {
		pattern := PatternImpulse
		if truncated {
			pattern = PatternTruncatedImpulse
			evaluations = append(evaluations, structuralEvaluation(
				RuleTruncationFiveWaves, level, points, lower,
				[]WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective, ModeMotive},
			))
			evaluations = append(evaluations, truncationStrengthEvaluation(lengths))
		} else {
			evaluations = append(evaluations, structuralEvaluation(
				RuleImpulseSubdivision, level, points, lower,
				[]WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective, ModeMotive},
			))
		}
		evaluations = append(evaluations, evaluation(RuleImpulseNoOverlap, EvaluationPass, points[4].Price, "wave 4 outside wave 1 territory"))
		if !hasHardFailure(evaluations) {
			result = append(result, buildNode(pattern, ModeMotive, direction, StatusCompleted, level, points, lower, evaluations))
		}
	}

	if overlap && convergingDiagonal(lengths) {
		leadingEvaluations := append(cloneEvaluations(evaluations),
			evaluation(RuleDiagonalConvergence, EvaluationPass, lengths[4]/lengths[0], "converging actionary and reactionary waves"),
			evaluation(RuleLeadingDiagonalPosition, EvaluationNotObservable, 0, "must be placed as wave 1 or A"),
			structuralEvaluation(RuleLeadingDiagonalShape, level, points, lower,
				[]WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective, ModeMotive}),
		)
		if !hasHardFailure(leadingEvaluations) {
			result = append(result, buildNode(PatternLeadingDiagonal, ModeMotive, direction, StatusCompleted, level, points, lower, leadingEvaluations))
		}

		endingEvaluations := append(cloneEvaluations(evaluations),
			evaluation(RuleDiagonalConvergence, EvaluationPass, lengths[4]/lengths[0], "converging actionary and reactionary waves"),
			evaluation(RuleEndingDiagonalPosition, EvaluationNotObservable, 0, "must be placed as wave 5 or C"),
			structuralEvaluation(RuleEndingDiagonalShape, level, points, lower,
				[]WaveMode{ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective}),
		)
		if !hasHardFailure(endingEvaluations) {
			result = append(result, buildNode(PatternEndingDiagonal, ModeMotive, direction, StatusCompleted, level, points, lower, endingEvaluations))
		}
	}
	return result
}

func motiveCoreEvaluations(points []Pivot, direction Direction, lengths []float64) []RuleEvaluation {
	wave2Valid := false
	wave3Beyond := false
	if direction == DirectionBullish {
		wave2Valid = points[2].Price >= points[0].Price
		wave3Beyond = points[3].Price > points[1].Price
	} else {
		wave2Valid = points[2].Price <= points[0].Price
		wave3Beyond = points[3].Price < points[1].Price
	}
	wave3NotShortest := lengths[2] >= math.Min(lengths[0], lengths[4])

	evaluations := []RuleEvaluation{
		evaluation(RuleMotiveWave2Limit, boolStatus(wave2Valid), lengths[1]/nonZero(lengths[0]), "<= 1.0 retracement"),
		evaluation(RuleMotiveWave3BeyondWave1, boolStatus(wave3Beyond), points[3].Price, "beyond wave 1 endpoint"),
		evaluation(RuleMotiveWave3NotShortest, boolStatus(wave3NotShortest), lengths[2], "wave 3 >= min(wave 1, wave 5)"),
		wave2RatioEvaluation(lengths[1] / nonZero(lengths[0])),
		wave3RatioEvaluation(lengths[2] / nonZero(lengths[0])),
		wave4RatioEvaluation(lengths[3] / nonZero(lengths[2])),
		wave5RatioEvaluation(lengths[4]/nonZero(lengths[0]), lengths[4]/nonZero(math.Abs(points[3].Price-points[0].Price))),
		alternationEvaluation(lengths[1]/nonZero(lengths[0]), lengths[3]/nonZero(lengths[2])),
		equalityEvaluation(lengths),
		channelEvaluation(points),
		volumeNotObservable(),
		personalityNotObservable(),
	}
	return evaluations
}

func parseABCWindow(level int, points []Pivot, lower []WaveNode) []WaveNode {
	if !alternates(points) {
		return nil
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok || !continuesCorrectionShape(points, direction) {
		return nil
	}

	lenA := math.Abs(points[1].Price - points[0].Price)
	lenB := math.Abs(points[2].Price - points[1].Price)
	lenC := math.Abs(points[3].Price - points[2].Price)
	if lenA == 0 || lenC == 0 {
		return nil
	}
	ratioB := lenB / lenA
	ratioC := lenC / lenA
	beyondStart := priceBeyond(points[2].Price, points[0].Price, direction.Opposite())
	cBeyondA := priceBeyond(points[3].Price, points[1].Price, direction)
	result := make([]WaveNode, 0, 2)

	if !beyondStart && cBeyondA {
		evals := []RuleEvaluation{
			evaluation(RuleCorrectionNeverFive, EvaluationPass, 0, "three-wave completed correction"),
			structuralEvaluation(RuleZigzagSubdivision, level, points, lower, []WaveMode{ModeMotive, ModeCorrective, ModeMotive}),
			ratioGuideline("EWP-ZIGZAG-C-RATIO", "EWP p.74", ratioC, []float64{0.618, 1, 1.618}, "C relative to A"),
			ratioGuideline("EWP-ZIGZAG-B-DEPTH", "EWP pp.21-22,72", ratioB, []float64{0.5, 0.618}, "B retracement of A"),
		}
		if !hasHardFailure(evals) {
			result = append(result, buildNode(PatternZigzag, ModeCorrective, direction, StatusCompleted, level, points, lower, evals))
		}
	}

	if beyondStart {
		pattern := PatternFlatExpanded
		if !cBeyondA {
			pattern = PatternFlatRunning
		}
		evals := flatEvaluations(level, points, lower, ratioB, ratioC, pattern)
		if !hasHardFailure(evals) {
			result = append(result, buildNode(pattern, ModeCorrective, direction, StatusCompleted, level, points, lower, evals))
		}
	} else if ratioB >= 0.80 {
		evals := flatEvaluations(level, points, lower, ratioB, ratioC, PatternFlatRegular)
		if !hasHardFailure(evals) {
			result = append(result, buildNode(PatternFlatRegular, ModeCorrective, direction, StatusCompleted, level, points, lower, evals))
		}
	}
	return result
}

func flatEvaluations(level int, points []Pivot, lower []WaveNode, ratioB, ratioC float64, pattern PatternType) []RuleEvaluation {
	evals := []RuleEvaluation{
		evaluation(RuleCorrectionNeverFive, EvaluationPass, 0, "three-wave completed correction"),
		structuralEvaluation(RuleFlatSubdivision, level, points, lower, []WaveMode{ModeCorrective, ModeCorrective, ModeMotive}),
	}
	switch pattern {
	case PatternFlatExpanded:
		evals = append(evals,
			ratioGuideline("EWP-EXPANDED-FLAT-B", "EWP pp.25,74", ratioB, []float64{1.236, 1.382}, "B relative to A"),
			ratioGuideline("EWP-EXPANDED-FLAT-C", "EWP p.74", ratioC, []float64{1.618, 2.618}, "C relative to A"))
	case PatternFlatRunning:
		evals = append(evals,
			evaluation("EWP-RUNNING-FLAT-RARE", EvaluationNotObservable, ratioC, "requires strong adjacent motive context"))
	default:
		evals = append(evals,
			ratioGuideline("EWP-REGULAR-FLAT-EQUALITY", "EWP pp.24-25,74", ratioC, []float64{1}, "A, B and C approximately equal"))
	}
	return evals
}

func parseTriangleWindow(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok {
		return WaveNode{}, false
	}

	highs := make([]Pivot, 0, 3)
	lows := make([]Pivot, 0, 3)
	for _, point := range points {
		if point.Kind == PivotHigh {
			highs = append(highs, point)
		} else {
			lows = append(lows, point)
		}
	}
	if len(highs) < 2 || len(lows) < 2 {
		return WaveNode{}, false
	}
	highSlope := slope(highs[0], highs[len(highs)-1])
	lowSlope := slope(lows[0], lows[len(lows)-1])
	totalRange := maxPrice(points) - minPrice(points)
	barrierTolerance := totalRange * 0.05

	var pattern PatternType
	running := priceBeyond(points[2].Price, points[0].Price, direction.Opposite())
	switch {
	case running && highSlope <= 0 && lowSlope >= 0:
		pattern = PatternTriangleRunning
	case highSlope < 0 && lowSlope > 0:
		pattern = PatternTriangleContracting
	case highSlope > 0 && lowSlope < 0:
		pattern = PatternTriangleExpanding
	case math.Abs(highs[len(highs)-1].Price-highs[0].Price) <= barrierTolerance && lowSlope > 0:
		pattern = PatternTriangleAscending
	case math.Abs(lows[len(lows)-1].Price-lows[0].Price) <= barrierTolerance && highSlope < 0:
		pattern = PatternTriangleDescending
	default:
		return WaveNode{}, false
	}

	lengths := segmentLengths(points)
	evals := []RuleEvaluation{
		evaluation(RuleCorrectionNeverFive, EvaluationPass, 0, "five overlapping corrective legs"),
		structuralEvaluation(RuleTriangleSubdivision, level, points, lower,
			[]WaveMode{ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective}),
		evaluation(RuleTrianglePosition, EvaluationNotObservable, 0, "must be placed as 4, B, X or terminal combination"),
		triangleRatioEvaluation(lengths, pattern),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(pattern, ModeCorrective, direction, StatusCompleted, level, points, lower, evals), true
}

func parseDevelopingImpulseW4(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok || !continuesCorrectionShape(points, direction) {
		return WaveNode{}, false
	}
	if !priceBeyond(points[3].Price, points[1].Price, direction) ||
		priceBeyond(points[2].Price, points[0].Price, direction.Opposite()) {
		return WaveNode{}, false
	}
	evals := []RuleEvaluation{
		evaluation(RuleMotiveWave2Limit, EvaluationPass, math.Abs(points[1].Price-points[2].Price)/nonZero(math.Abs(points[1].Price-points[0].Price)), "<= 1.0"),
		evaluation(RuleMotiveWave3BeyondWave1, EvaluationPass, points[3].Price, "beyond wave 1"),
		structuralEvaluation("EWP-DEVELOPING-123", level, points, lower, []WaveMode{ModeMotive, ModeCorrective, ModeMotive}),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(PatternDevelopingImpulseW4, ModeMotive, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseDevelopingImpulseW2(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if len(points) != 2 || !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok {
		return WaveNode{}, false
	}
	evals := []RuleEvaluation{
		structuralEvaluation("EWP-DEVELOPING-W1", level, points, lower, []WaveMode{ModeMotive}),
		evaluation(PriorWave2Ratios, EvaluationNotApplicable, 0, "wave 2 has not started"),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(PatternDevelopingImpulseW2, ModeMotive, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseDevelopingImpulseW3(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if len(points) != 3 || !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok {
		return WaveNode{}, false
	}
	valid := true
	if direction == DirectionBullish {
		valid = points[2].Price >= points[0].Price
	} else {
		valid = points[2].Price <= points[0].Price
	}
	if !valid {
		return WaveNode{}, false
	}
	ratio := math.Abs(points[1].Price-points[2].Price) / nonZero(math.Abs(points[1].Price-points[0].Price))
	evals := []RuleEvaluation{
		evaluation(RuleMotiveWave2Limit, EvaluationPass, ratio, "<= 1.0"),
		wave2RatioEvaluation(ratio),
		structuralEvaluation("EWP-DEVELOPING-W12", level, points, lower, []WaveMode{ModeMotive, ModeCorrective}),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(PatternDevelopingImpulseW3, ModeMotive, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseDevelopingImpulseW5(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok || !continuesMotivePartial(points, direction) {
		return WaveNode{}, false
	}
	overlap := wave4OverlapsWave1(points, direction)
	evals := []RuleEvaluation{
		evaluation(RuleMotiveWave2Limit, EvaluationPass, 0, "<= 1.0"),
		evaluation(RuleMotiveWave3BeyondWave1, EvaluationPass, points[3].Price, "beyond wave 1"),
	}
	if overlap {
		evals = append(evals, evaluation(RuleImpulseNoOverlap, EvaluationNotApplicable, points[4].Price, "possible diagonal"))
	} else {
		evals = append(evals, evaluation(RuleImpulseNoOverlap, EvaluationPass, points[4].Price, "outside wave 1 territory"))
	}
	evals = append(evals, structuralEvaluation("EWP-DEVELOPING-1234", level, points, lower,
		[]WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective}))
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(PatternDevelopingImpulseW5, ModeMotive, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseDevelopingZigzagC(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok || priceBeyond(points[2].Price, points[0].Price, direction.Opposite()) {
		return WaveNode{}, false
	}
	evals := []RuleEvaluation{
		structuralEvaluation("EWP-DEVELOPING-ZIGZAG-AB", level, points, lower, []WaveMode{ModeMotive, ModeCorrective}),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(PatternDevelopingZigzagC, ModeCorrective, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseDevelopingFlatC(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	if len(points) != 3 || !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok {
		return WaveNode{}, false
	}
	lengthA := math.Abs(points[1].Price - points[0].Price)
	lengthB := math.Abs(points[2].Price - points[1].Price)
	ratioB := lengthB / nonZero(lengthA)
	if ratioB < 0.80 {
		return WaveNode{}, false
	}
	evals := []RuleEvaluation{
		structuralEvaluation(
			"EWP-DEVELOPING-FLAT-AB", level, points, lower,
			[]WaveMode{ModeCorrective, ModeCorrective},
		),
		ratioGuideline(
			"EWP-FLAT-B-DEPTH", "EWP pp.24-27,74", ratioB,
			[]float64{1, 1.236, 1.382}, "B retracement or extension of A",
		),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(
		PatternDevelopingFlatC, ModeCorrective, direction, StatusDeveloping,
		level, points, lower, evals,
	), true
}

func parseDevelopingTriangleD(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	return parseDevelopingTriangle(level, points, lower, PatternDevelopingTriangleD)
}

func parseDevelopingTriangleE(level int, points []Pivot, lower []WaveNode) (WaveNode, bool) {
	return parseDevelopingTriangle(level, points, lower, PatternDevelopingTriangleE)
}

func parseDevelopingTriangle(level int, points []Pivot, lower []WaveNode, pattern PatternType) (WaveNode, bool) {
	if !alternates(points) {
		return WaveNode{}, false
	}
	direction, ok := directionBetween(points[0], points[1])
	if !ok {
		return WaveNode{}, false
	}
	modes := make([]WaveMode, len(points)-1)
	for i := range modes {
		modes[i] = ModeCorrective
	}
	evals := []RuleEvaluation{
		structuralEvaluation(RuleTriangleSubdivision, level, points, lower, modes),
		evaluation(RuleTrianglePosition, EvaluationNotObservable, 0, "must be placed before a final actionary wave"),
	}
	if hasHardFailure(evals) {
		return WaveNode{}, false
	}
	return buildNode(pattern, ModeCorrective, direction, StatusDeveloping, level, points, lower, evals), true
}

func parseCombinations(nodes []WaveNode, level int) []WaveNode {
	simple := make([]WaveNode, 0, len(nodes))
	for _, node := range nodes {
		if node.Status == StatusCompleted && node.Mode == ModeCorrective &&
			(node.Pattern == PatternZigzag || isFlat(node.Pattern) || isTriangle(node.Pattern)) {
			simple = append(simple, node)
		}
	}
	if len(simple) < 3 {
		return nil
	}

	byStart := make(map[int][]WaveNode)
	for _, node := range simple {
		byStart[node.OrthodoxStart.BarIndex] = append(byStart[node.OrthodoxStart.BarIndex], node)
	}

	result := make([]WaveNode, 0)
	for _, w := range simple {
		for _, x := range byStart[w.OrthodoxEnd.BarIndex] {
			if x.Direction == w.Direction || isTriangle(x.Pattern) {
				continue
			}
			for _, y := range byStart[x.OrthodoxEnd.BarIndex] {
				if y.Direction != w.Direction || isTriangle(w.Pattern) {
					continue
				}
				children := []WaveNode{w, x, y}
				pattern := PatternDoubleThree
				if w.Pattern == PatternZigzag && y.Pattern == PatternZigzag {
					pattern = PatternDoubleZigzag
				}
				evals := combinationEvaluations(pattern, children)
				if hasHardFailure(evals) {
					continue
				}
				result = append(result, buildCombinationNode(pattern, level, children, evals))

				for _, secondX := range byStart[y.OrthodoxEnd.BarIndex] {
					if secondX.Direction == w.Direction || isTriangle(secondX.Pattern) {
						continue
					}
					for _, z := range byStart[secondX.OrthodoxEnd.BarIndex] {
						if z.Direction != w.Direction || isTriangle(y.Pattern) {
							continue
						}
						tripleChildren := []WaveNode{w, x, y, secondX, z}
						triplePattern := PatternTripleThree
						if w.Pattern == PatternZigzag &&
							y.Pattern == PatternZigzag &&
							z.Pattern == PatternZigzag {
							triplePattern = PatternTripleZigzag
						}
						tripleEvaluations := combinationEvaluations(triplePattern, tripleChildren)
						if hasHardFailure(tripleEvaluations) {
							continue
						}
						result = append(result, buildCombinationNode(
							triplePattern, level, tripleChildren, tripleEvaluations,
						))
					}
				}
			}
		}
	}
	return result
}

func combinationEvaluations(pattern PatternType, children []WaveNode) []RuleEvaluation {
	triangles := 0
	zigzags := 0
	for index, child := range children {
		if isTriangle(child.Pattern) {
			triangles++
		}
		if index%2 == 0 && child.Pattern == PatternZigzag {
			zigzags++
		}
	}
	lastIsTriangle := triangles == 0 || isTriangle(children[len(children)-1].Pattern)
	evals := []RuleEvaluation{
		evaluation(RuleCombinationTriangleLast, boolStatus(lastIsTriangle), float64(triangles), "triangle absent or terminal"),
		evaluation(RuleCombinationOneTriangle, boolStatus(triangles <= 1), float64(triangles), "<= 1 triangle"),
	}
	if pattern == PatternDoubleThree || pattern == PatternTripleThree {
		evals = append(evals, evaluation(RuleCombinationOneZigzag, boolStatus(zigzags <= 1), float64(zigzags), "<= 1 zigzag"))
	} else {
		required := (len(children) + 1) / 2
		evals = append(evals, evaluation(
			RuleZigzagSubdivision,
			boolStatus(zigzags == required),
			float64(zigzags),
			"every actionary component is a zigzag",
		))
	}
	return evals
}

func buildCombinationNode(pattern PatternType, level int, children []WaveNode, evals []RuleEvaluation) WaveNode {
	pivots := make([]Pivot, 0, len(children)+1)
	pivots = append(pivots, children[0].OrthodoxStart)
	for _, child := range children {
		pivots = append(pivots, child.OrthodoxEnd)
	}
	evals = completeRuleAudit(pattern, ModeCorrective, pivots, evals)
	node := WaveNode{
		Pattern: pattern, Mode: ModeCorrective, Function: FunctionActionary,
		Direction: children[0].Direction, Degree: degreeForLevel(level), Status: StatusCompleted,
		Label: patternLabel(pattern), Level: level, OrthodoxStart: pivots[0],
		OrthodoxEnd: pivots[len(pivots)-1], Pivots: pivots, Children: children,
		RuleEvaluations: evals,
	}
	node.ID = nodeID(node)
	node.Measurements = nodeMeasurements(pivots)
	node.Conformance = calculateConformance(node.RuleEvaluations, len(children), len(children))
	return node
}

func buildNode(pattern PatternType, mode WaveMode, direction Direction, status WaveStatus, level int, points []Pivot, lower []WaveNode, evals []RuleEvaluation) WaveNode {
	expectedModes := expectedModes(pattern, len(points)-1)
	children := matchChildren(points, expectedModes, lower, grammarRuleForPattern(pattern))
	evals = completeRuleAudit(pattern, mode, points, evals)
	node := WaveNode{
		Pattern: pattern, Mode: mode, Function: FunctionActionary,
		Direction: direction, Degree: degreeForLevel(level), Status: status,
		Label: patternLabel(pattern), Level: level, OrthodoxStart: points[0],
		OrthodoxEnd: points[len(points)-1], Pivots: append([]Pivot(nil), points...),
		Children: children, RuleEvaluations: cloneEvaluations(evals),
	}
	node.ID = nodeID(node)
	node.Measurements = nodeMeasurements(points)
	node.Conformance = calculateConformance(node.RuleEvaluations, len(children), len(expectedModes))
	return node
}

func grammarRuleForPattern(pattern PatternType) string {
	switch pattern {
	case PatternImpulse:
		return RuleImpulseSubdivision
	case PatternTruncatedImpulse:
		return RuleTruncationFiveWaves
	case PatternLeadingDiagonal:
		return RuleLeadingDiagonalShape
	case PatternEndingDiagonal:
		return RuleEndingDiagonalShape
	case PatternZigzag:
		return RuleZigzagSubdivision
	case PatternFlatRegular, PatternFlatExpanded, PatternFlatRunning:
		return RuleFlatSubdivision
	case PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding,
		PatternDevelopingTriangleD, PatternDevelopingTriangleE:
		return RuleTriangleSubdivision
	case PatternDevelopingImpulseW2:
		return "EWP-DEVELOPING-W1"
	case PatternDevelopingImpulseW3:
		return "EWP-DEVELOPING-W12"
	case PatternDevelopingImpulseW4:
		return "EWP-DEVELOPING-123"
	case PatternDevelopingImpulseW5:
		return "EWP-DEVELOPING-1234"
	case PatternDevelopingZigzagC:
		return "EWP-DEVELOPING-ZIGZAG-AB"
	case PatternDevelopingFlatC:
		return "EWP-DEVELOPING-FLAT-AB"
	default:
		return ""
	}
}

func structuralEvaluation(ruleID string, level int, points []Pivot, lower []WaveNode, modes []WaveMode) RuleEvaluation {
	if level == 0 {
		result := evaluation(ruleID, EvaluationNotObservable, 0, "subdivision below available resolution")
		if result.Class == "" {
			result.Class = RuleHard
			result.Source = "EWP observable-resolution boundary"
			result.Summary = "Required internal subdivision must be present when lower-resolution data is observable."
		}
		return result
	}
	children := matchChildren(points, modes, lower, ruleID)
	if len(children) == len(modes) {
		result := evaluation(ruleID, EvaluationPass, float64(len(children)), fmt.Sprintf("%d required child structures", len(modes)))
		if result.Class == "" {
			result.Class = RuleHard
			result.Source = "EWP pattern grammar"
			result.Summary = "Required internal subdivision is fully observable."
		}
		return result
	}
	result := evaluation(ruleID, EvaluationFail, float64(len(children)), fmt.Sprintf("%d required child structures", len(modes)))
	if result.Class == "" {
		result.Class = RuleHard
		result.Source = "EWP pattern grammar"
		result.Summary = "Required internal subdivision is incomplete."
	}
	return result
}

func matchChildren(points []Pivot, modes []WaveMode, lower []WaveNode, grammarRuleID string) []WaveNode {
	if len(modes) == 0 || len(points) != len(modes)+1 {
		return nil
	}
	result := make([]WaveNode, 0, len(modes))
	for i, mode := range modes {
		direction, ok := directionBetween(points[i], points[i+1])
		if !ok {
			return nil
		}
		var best *WaveNode
		for idx := range lower {
			candidate := &lower[idx]
			if candidate.Status != StatusCompleted || candidate.Mode != mode || candidate.Direction != direction ||
				candidate.OrthodoxStart.BarIndex != points[i].BarIndex ||
				candidate.OrthodoxEnd.BarIndex != points[i+1].BarIndex {
				continue
			}
			if !childAllowed(grammarRuleID, i, len(modes), candidate.Pattern) {
				continue
			}
			if best == nil || candidate.Conformance.Score > best.Conformance.Score {
				best = candidate
			}
		}
		if best == nil {
			continue
		}
		placed := *best
		placed.RuleEvaluations = cloneEvaluations(placed.RuleEvaluations)
		switch placed.Pattern {
		case PatternLeadingDiagonal:
			replaceEvaluation(&placed.RuleEvaluations, evaluation(
				RuleLeadingDiagonalPosition, EvaluationPass, float64(i), "wave 1 or A position",
			))
		case PatternEndingDiagonal:
			replaceEvaluation(&placed.RuleEvaluations, evaluation(
				RuleEndingDiagonalPosition, EvaluationPass, float64(i), "wave 5 or C position",
			))
		default:
			if isTriangle(placed.Pattern) {
				replaceEvaluation(&placed.RuleEvaluations, evaluation(
					RuleTrianglePosition, EvaluationPass, float64(i), "wave 4, B, X or terminal combination",
				))
			}
		}
		placed.Conformance = calculateConformance(
			placed.RuleEvaluations, len(placed.Children), len(expectedModes(placed.Pattern, len(placed.Pivots)-1)),
		)
		result = append(result, placed)
	}
	return result
}

func childAllowed(ruleID string, index, componentCount int, pattern PatternType) bool {
	if pattern == PatternLeadingDiagonal && index != 0 {
		return false
	}
	if pattern == PatternEndingDiagonal && index != componentCount-1 {
		return false
	}
	if isTriangle(pattern) {
		switch ruleID {
		case RuleImpulseSubdivision, RuleTruncationFiveWaves, "EWP-DEVELOPING-1234":
			return index == 3
		case RuleZigzagSubdivision, "EWP-DEVELOPING-ZIGZAG-AB":
			return index == 1
		default:
			return false
		}
	}
	switch ruleID {
	case RuleLeadingDiagonalShape:
		if index%2 == 1 {
			return isSimpleThree(pattern)
		}
	case RuleEndingDiagonalShape, RuleTriangleSubdivision:
		return isSimpleThree(pattern)
	case RuleFlatSubdivision:
		if index < 2 {
			return isSimpleThree(pattern)
		}
	}
	return true
}

func isSimpleThree(pattern PatternType) bool {
	return pattern == PatternZigzag || isFlat(pattern)
}

func calculateConformance(evaluations []RuleEvaluation, childCount, expectedChildren int) Conformance {
	var result Conformance
	for _, evaluation := range evaluations {
		switch evaluation.Status {
		case EvaluationNotObservable:
			result.NotObservable++
		case EvaluationPass:
			if evaluation.Class == RuleHard {
				result.HardRulesPassed++
			} else {
				result.GuidelinesPassed++
				if strings.Contains(evaluation.RuleID, "RATIO") ||
					evaluation.Class == RuleStatisticalPrior {
					result.RatioConfluences++
				}
			}
		case EvaluationFail:
			if evaluation.Class == RuleHard {
				result.HardRulesFailed++
			} else {
				result.GuidelinesFailed++
			}
		}
	}
	if expectedChildren == 0 {
		result.StructuralCoverage = 1
	} else {
		result.StructuralCoverage = float64(childCount) / float64(expectedChildren)
	}
	guidelineTotal := result.GuidelinesPassed + result.GuidelinesFailed
	guidelineRate := 0.5
	if guidelineTotal > 0 {
		guidelineRate = float64(result.GuidelinesPassed) / float64(guidelineTotal)
	}
	result.Score = clamp01(0.65*result.StructuralCoverage + 0.25*guidelineRate + 0.10*math.Min(1, float64(result.RatioConfluences)/3))
	if result.HardRulesFailed > 0 {
		result.Score = 0
	}
	return result
}

func sortNodes(nodes []WaveNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Level != nodes[j].Level {
			return nodes[i].Level > nodes[j].Level
		}
		if nodes[i].OrthodoxEnd.BarIndex != nodes[j].OrthodoxEnd.BarIndex {
			return nodes[i].OrthodoxEnd.BarIndex > nodes[j].OrthodoxEnd.BarIndex
		}
		if nodes[i].Conformance.StructuralCoverage != nodes[j].Conformance.StructuralCoverage {
			return nodes[i].Conformance.StructuralCoverage > nodes[j].Conformance.StructuralCoverage
		}
		if nodes[i].Conformance.GuidelinesPassed != nodes[j].Conformance.GuidelinesPassed {
			return nodes[i].Conformance.GuidelinesPassed > nodes[j].Conformance.GuidelinesPassed
		}
		if nodes[i].Conformance.RatioConfluences != nodes[j].Conformance.RatioConfluences {
			return nodes[i].Conformance.RatioConfluences > nodes[j].Conformance.RatioConfluences
		}
		if nodes[i].Conformance.Score != nodes[j].Conformance.Score {
			return nodes[i].Conformance.Score > nodes[j].Conformance.Score
		}
		return nodes[i].ID < nodes[j].ID
	})
}

func deduplicateNodes(nodes []WaveNode) []WaveNode {
	seen := make(map[string]int, len(nodes))
	result := make([]WaveNode, 0, len(nodes))
	for _, node := range nodes {
		signature := nodeSignature(node)
		if index, ok := seen[signature]; ok {
			if node.Conformance.Score > result[index].Conformance.Score {
				result[index] = node
			}
			continue
		}
		seen[signature] = len(result)
		result = append(result, node)
	}
	return result
}

func nodeID(node WaveNode) string {
	sum := sha256.Sum256([]byte(nodeSignature(node)))
	return hex.EncodeToString(sum[:8])
}

func nodeSignature(node WaveNode) string {
	var builder strings.Builder
	builder.WriteString(string(node.Pattern))
	builder.WriteByte('|')
	builder.WriteString(string(node.Status))
	builder.WriteByte('|')
	builder.WriteString(fmt.Sprintf("%d", node.Level))
	for _, pivot := range node.Pivots {
		builder.WriteString(fmt.Sprintf("|%d:%s:%.10f", pivot.BarIndex, pivot.Kind, pivot.Price))
	}
	for _, child := range node.Children {
		builder.WriteByte('|')
		builder.WriteString(child.ID)
	}
	return builder.String()
}

func nodeMeasurements(points []Pivot) []Measurement {
	result := make([]Measurement, 0, (len(points)-1)*2)
	for i := 0; i+1 < len(points); i++ {
		result = append(result,
			Measurement{Name: fmt.Sprintf("wave_%d_price", i+1), Value: math.Abs(points[i+1].Price - points[i].Price), Unit: "price"},
			Measurement{Name: fmt.Sprintf("wave_%d_bars", i+1), Value: float64(points[i+1].BarIndex - points[i].BarIndex), Unit: "bars"},
		)
	}
	return result
}

func expectedModes(pattern PatternType, componentCount int) []WaveMode {
	var modes []WaveMode
	switch pattern {
	case PatternImpulse, PatternLeadingDiagonal, PatternTruncatedImpulse:
		modes = []WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective, ModeMotive}
	case PatternEndingDiagonal, PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding:
		modes = []WaveMode{ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective, ModeCorrective}
	case PatternZigzag:
		modes = []WaveMode{ModeMotive, ModeCorrective, ModeMotive}
	case PatternFlatRegular, PatternFlatExpanded, PatternFlatRunning:
		modes = []WaveMode{ModeCorrective, ModeCorrective, ModeMotive}
	case PatternDevelopingImpulseW4:
		modes = []WaveMode{ModeMotive, ModeCorrective, ModeMotive}
	case PatternDevelopingImpulseW2:
		modes = []WaveMode{ModeMotive}
	case PatternDevelopingImpulseW3:
		modes = []WaveMode{ModeMotive, ModeCorrective}
	case PatternDevelopingImpulseW5:
		modes = []WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective}
	case PatternDevelopingZigzagC:
		modes = []WaveMode{ModeMotive, ModeCorrective}
	case PatternDevelopingFlatC:
		modes = []WaveMode{ModeCorrective, ModeCorrective}
	default:
		if pattern == PatternDevelopingTriangleD || pattern == PatternDevelopingTriangleE {
			modes = make([]WaveMode, componentCount)
			for i := range modes {
				modes[i] = ModeCorrective
			}
		}
	}
	if len(modes) > componentCount {
		modes = modes[:componentCount]
	}
	return modes
}

func degreeForLevel(level int) Degree {
	degrees := [...]Degree{
		DegreeObservableLeaf,
		DegreeSubminuette,
		DegreeMinuette,
		DegreeMinute,
		DegreeMinor,
		DegreeIntermediate,
		DegreePrimary,
		DegreeCycle,
		DegreeSupercycle,
		DegreeGrandSupercycle,
	}
	if level < 0 {
		return DegreeObservableLeaf
	}
	if level >= len(degrees) {
		return DegreeGrandSupercycle
	}
	return degrees[level]
}
