package wave

import (
	"math"
	"testing"

	"WaveSight/internal/market"
)

func pivot(index int, price float64, kind PivotKind) Pivot {
	return Pivot{
		Time: int64(1_700_000_000 + index*60), BarIndex: index,
		Price: price, Kind: kind, State: PivotConfirmed, Prominence: 2,
	}
}

func bullishImpulse() []Pivot {
	return []Pivot{
		pivot(0, 100, PivotLow),
		pivot(10, 120, PivotHigh),
		pivot(18, 110, PivotLow),
		pivot(32, 150, PivotHigh),
		pivot(40, 135, PivotLow),
		pivot(52, 155, PivotHigh),
	}
}

func mirror(points []Pivot, axis float64) []Pivot {
	result := make([]Pivot, len(points))
	for i, point := range points {
		result[i] = point
		result[i].Price = 2*axis - point.Price
		if point.Kind == PivotHigh {
			result[i].Kind = PivotLow
		} else {
			result[i].Kind = PivotHigh
		}
	}
	return result
}

func TestMotiveHardRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		points      []Pivot
		want        PatternType
		wantMatches bool
	}{
		{name: "bullish impulse", points: bullishImpulse(), want: PatternImpulse, wantMatches: true},
		{name: "bearish impulse", points: mirror(bullishImpulse(), 130), want: PatternImpulse, wantMatches: true},
		{
			name: "wave two exceeds origin",
			points: []Pivot{
				pivot(0, 100, PivotLow), pivot(10, 120, PivotHigh), pivot(18, 99, PivotLow),
				pivot(32, 150, PivotHigh), pivot(40, 135, PivotLow), pivot(52, 155, PivotHigh),
			},
			want: PatternImpulse, wantMatches: false,
		},
		{
			name: "wave three does not exceed wave one",
			points: []Pivot{
				pivot(0, 100, PivotLow), pivot(10, 120, PivotHigh), pivot(18, 110, PivotLow),
				pivot(32, 119, PivotHigh), pivot(40, 114, PivotLow), pivot(52, 130, PivotHigh),
			},
			want: PatternImpulse, wantMatches: false,
		},
		{
			name: "wave three shortest",
			points: []Pivot{
				pivot(0, 100, PivotLow), pivot(10, 130, PivotHigh), pivot(18, 115, PivotLow),
				pivot(32, 136, PivotHigh), pivot(40, 131, PivotLow), pivot(52, 160, PivotHigh),
			},
			want: PatternImpulse, wantMatches: false,
		},
		{
			name: "wave four overlap invalidates impulse",
			points: []Pivot{
				pivot(0, 100, PivotLow), pivot(10, 130, PivotHigh), pivot(18, 115, PivotLow),
				pivot(32, 170, PivotHigh), pivot(40, 125, PivotLow), pivot(52, 180, PivotHigh),
			},
			want: PatternImpulse, wantMatches: false,
		},
		{
			name: "ratio mismatch is not a hard failure",
			points: []Pivot{
				pivot(0, 100, PivotLow), pivot(10, 117, PivotHigh), pivot(18, 113, PivotLow),
				pivot(32, 143, PivotHigh), pivot(40, 121, PivotLow), pivot(52, 154, PivotHigh),
			},
			want: PatternImpulse, wantMatches: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			nodes := parseMotiveWindow(0, test.points, nil)
			found := false
			for _, node := range nodes {
				if node.Pattern == test.want {
					found = true
					if node.Conformance.HardRulesFailed != 0 {
						t.Errorf("returned node has hard failure: %+v", node.RuleEvaluations)
					}
				}
			}
			if found != test.wantMatches {
				t.Fatalf("pattern %s found=%t, want %t; nodes=%+v", test.want, found, test.wantMatches, nodes)
			}
		})
	}
}

func TestDiagonalAndTruncationRules(t *testing.T) {
	t.Parallel()

	diagonal := []Pivot{
		pivot(0, 100, PivotLow),
		pivot(10, 140, PivotHigh),
		pivot(18, 115, PivotLow),
		pivot(28, 145, PivotHigh),
		pivot(36, 130, PivotLow),
		pivot(46, 150, PivotHigh),
	}
	nodes := parseMotiveWindow(0, diagonal, nil)
	assertPatternPresent(t, nodes, PatternLeadingDiagonal)
	assertPatternPresent(t, nodes, PatternEndingDiagonal)
	assertPatternAbsent(t, nodes, PatternImpulse)

	expanding := []Pivot{
		pivot(0, 100, PivotLow),
		pivot(10, 120, PivotHigh),
		pivot(18, 108, PivotLow),
		pivot(28, 140, PivotHigh),
		pivot(36, 115, PivotLow),
		pivot(46, 160, PivotHigh),
	}
	expandingNodes := parseMotiveWindow(0, expanding, nil)
	assertPatternAbsent(t, expandingNodes, PatternLeadingDiagonal)
	assertPatternAbsent(t, expandingNodes, PatternEndingDiagonal)

	truncated := []Pivot{
		pivot(0, 100, PivotLow),
		pivot(10, 120, PivotHigh),
		pivot(18, 110, PivotLow),
		pivot(32, 160, PivotHigh),
		pivot(40, 140, PivotLow),
		pivot(52, 158, PivotHigh),
	}
	truncatedNodes := parseMotiveWindow(0, truncated, nil)
	assertPatternPresent(t, truncatedNodes, PatternTruncatedImpulse)
}

func TestCorrectiveCatalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		points []Pivot
		want   PatternType
	}{
		{
			name: "zigzag",
			points: []Pivot{
				pivot(0, 200, PivotHigh), pivot(10, 150, PivotLow),
				pivot(18, 175, PivotHigh), pivot(30, 120, PivotLow),
			},
			want: PatternZigzag,
		},
		{
			name: "regular flat",
			points: []Pivot{
				pivot(0, 200, PivotHigh), pivot(10, 160, PivotLow),
				pivot(18, 196, PivotHigh), pivot(30, 155, PivotLow),
			},
			want: PatternFlatRegular,
		},
		{
			name: "expanded flat",
			points: []Pivot{
				pivot(0, 200, PivotHigh), pivot(10, 170, PivotLow),
				pivot(18, 210, PivotHigh), pivot(30, 150, PivotLow),
			},
			want: PatternFlatExpanded,
		},
		{
			name: "running flat",
			points: []Pivot{
				pivot(0, 200, PivotHigh), pivot(10, 170, PivotLow),
				pivot(18, 210, PivotHigh), pivot(30, 180, PivotLow),
			},
			want: PatternFlatRunning,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertPatternPresent(t, parseABCWindow(0, test.points, nil), test.want)
		})
	}
}

func TestExpandingTriangleIsAccepted(t *testing.T) {
	t.Parallel()

	points := []Pivot{
		pivot(0, 150, PivotHigh),
		pivot(10, 120, PivotLow),
		pivot(20, 160, PivotHigh),
		pivot(30, 105, PivotLow),
		pivot(40, 175, PivotHigh),
		pivot(50, 90, PivotLow),
	}
	node, ok := parseTriangleWindow(0, points, nil)
	if !ok {
		t.Fatal("expected expanding triangle to be accepted")
	}
	if node.Pattern != PatternTriangleExpanding {
		t.Fatalf("got %s, want %s", node.Pattern, PatternTriangleExpanding)
	}
}

func TestTriangleVariants(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		points []Pivot
		want   PatternType
	}{
		{
			name: "contracting",
			points: []Pivot{
				pivot(0, 150, PivotHigh), pivot(10, 100, PivotLow),
				pivot(20, 140, PivotHigh), pivot(30, 110, PivotLow),
				pivot(40, 132, PivotHigh), pivot(50, 118, PivotLow),
			},
			want: PatternTriangleContracting,
		},
		{
			name: "ascending",
			points: []Pivot{
				pivot(0, 150, PivotHigh), pivot(10, 100, PivotLow),
				pivot(20, 150, PivotHigh), pivot(30, 110, PivotLow),
				pivot(40, 150, PivotHigh), pivot(50, 120, PivotLow),
			},
			want: PatternTriangleAscending,
		},
		{
			name: "descending",
			points: []Pivot{
				pivot(0, 150, PivotHigh), pivot(10, 100, PivotLow),
				pivot(20, 140, PivotHigh), pivot(30, 100, PivotLow),
				pivot(40, 130, PivotHigh), pivot(50, 100, PivotLow),
			},
			want: PatternTriangleDescending,
		},
		{
			name: "running",
			points: []Pivot{
				pivot(0, 150, PivotHigh), pivot(10, 110, PivotLow),
				pivot(20, 155, PivotHigh), pivot(30, 120, PivotLow),
				pivot(40, 145, PivotHigh), pivot(50, 130, PivotLow),
			},
			want: PatternTriangleRunning,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			node, ok := parseTriangleWindow(0, test.points, nil)
			if !ok || node.Pattern != test.want {
				t.Fatalf("triangle = %s/%t, want %s/true", node.Pattern, ok, test.want)
			}
		})
	}
}

func TestDevelopingFlatExpectingC(t *testing.T) {
	t.Parallel()
	points := []Pivot{
		pivot(0, 200, PivotHigh),
		pivot(10, 170, PivotLow),
		pivot(20, 210, PivotHigh),
	}
	node, ok := parseDevelopingFlatC(0, points, nil)
	if !ok || node.Pattern != PatternDevelopingFlatC {
		t.Fatalf("developing flat = %+v/%t", node, ok)
	}
	levels, label, _ := projectedLevels(node, 1)
	if label != "C" || len(levels) != 3 {
		t.Fatalf("flat targets label/count = %q/%d, want C/3", label, len(levels))
	}
}

func TestStructuralGrammarRejectsTriangleInImpulseWaveTwo(t *testing.T) {
	t.Parallel()
	points := bullishImpulse()
	children := make([]WaveNode, 0, 5)
	patterns := []PatternType{
		PatternImpulse, PatternTriangleContracting, PatternImpulse, PatternFlatRegular, PatternImpulse,
	}
	modes := []WaveMode{ModeMotive, ModeCorrective, ModeMotive, ModeCorrective, ModeMotive}
	for index := range patterns {
		direction, _ := directionBetween(points[index], points[index+1])
		children = append(children, WaveNode{
			ID:      string(patterns[index]) + string(rune(index)),
			Pattern: patterns[index], Mode: modes[index], Direction: direction,
			Status: StatusCompleted, OrthodoxStart: points[index], OrthodoxEnd: points[index+1],
			Pivots:      []Pivot{points[index], points[index+1]},
			Conformance: Conformance{Score: 1, StructuralCoverage: 1},
		})
	}
	result := structuralEvaluation(RuleImpulseSubdivision, 1, points, children, modes)
	if result.Status != EvaluationFail {
		t.Fatalf("triangle in wave 2 status = %s, want FAIL", result.Status)
	}
	children[1].Pattern = PatternFlatRegular
	result = structuralEvaluation(RuleImpulseSubdivision, 1, points, children, modes)
	if result.Status != EvaluationPass {
		t.Fatalf("three-wave correction in wave 2 status = %s, want PASS", result.Status)
	}
}

func TestParserIsDeterministicAcrossWorkerScheduling(t *testing.T) {
	t.Parallel()
	lattice := PivotLattice{Branches: []PivotBranch{
		{Levels: [][]Pivot{bullishImpulse()}},
		{Levels: [][]Pivot{mirror(bullishImpulse(), 130)}},
	}}
	parser := NewParser(500)
	var baseline []string
	for iteration := 0; iteration < 25; iteration++ {
		nodes := parser.Parse(lattice)
		ids := make([]string, len(nodes))
		for index := range nodes {
			ids[index] = nodes[index].ID
		}
		if iteration == 0 {
			baseline = ids
			continue
		}
		if len(ids) != len(baseline) {
			t.Fatalf("iteration %d node count = %d, want %d", iteration, len(ids), len(baseline))
		}
		for index := range ids {
			if ids[index] != baseline[index] {
				t.Fatalf("iteration %d id[%d] = %s, want %s", iteration, index, ids[index], baseline[index])
			}
		}
	}
}

func TestHistoricalNodesAreNotPromotedToCurrentScenarios(t *testing.T) {
	t.Parallel()
	node := buildNode(
		PatternImpulse, ModeMotive, DirectionBullish, StatusCompleted,
		0, bullishImpulse(), nil, motiveCoreEvaluations(
			bullishImpulse(), DirectionBullish, segmentLengths(bullishImpulse()),
		),
	)
	if got := currentCandidates([]WaveNode{node}, 10_000); len(got) != 0 {
		t.Fatalf("currentCandidates() returned %d stale nodes", len(got))
	}
}

func TestLowPriorityWaveRatiosRemainPatternScoped(t *testing.T) {
	t.Parallel()
	waveTwo := buildNode(
		PatternDevelopingImpulseW2, ModeMotive, DirectionBullish, StatusDeveloping,
		0, bullishImpulse()[:2], nil, nil,
	)
	levels, _, _ := projectedLevels(waveTwo, 1)
	found014, found025 := false, false
	for _, level := range levels {
		if level.Relation == "0.140 retracement of W1 (low priority)" {
			found014 = true
		}
		if level.Relation == "0.250 retracement of W1 (low priority)" {
			found025 = true
		}
		if level.Family != "W2_RETRACEMENT" {
			t.Fatalf("low-priority relation escaped W2 family: %+v", level)
		}
	}
	if !found014 || !found025 {
		t.Fatalf("low priority W2 ratios found .14=%t .25=%t", found014, found025)
	}
}

func TestAnalysisResultEasyJSONRoundTrip(t *testing.T) {
	t.Parallel()
	node := buildNode(
		PatternDevelopingImpulseW5, ModeMotive, DirectionBullish, StatusDeveloping,
		0, bullishImpulse()[:5], nil, nil,
	)
	input := AnalysisResult{
		DataQuality: DataQuality{CandleCount: 10, FirstTime: 1, LastTime: 10},
		Scenarios: []Scenario{{
			ID: "scenario", Rank: 1, Status: ScenarioPreferred, Bias: DirectionBullish,
			Root: node, TargetLadder: []TargetZone{{
				ID: "target", WaveLabel: "W5", Status: TargetConditional,
				MinPrice: 150, MaxPrice: 155, Confluence: ConfluenceMedium,
				Levels:     []TargetLevel{{Price: 152, Relation: "1.0 × W1", Family: "W5", Source: "EWP", Uncertainty: 1}},
				TimeWindow: &TimeWindow{StartBarOffset: 3, EndBarOffset: 5, Evidence: []string{"duration equality"}},
			}},
		}},
		FutureBars: []int64{11, 12},
	}
	payload, err := input.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	var output AnalysisResult
	if err := output.UnmarshalJSON(payload); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if len(output.Scenarios) != 1 || output.Scenarios[0].Root.Pattern != PatternDevelopingImpulseW5 ||
		output.Scenarios[0].TargetLadder[0].Confluence != ConfluenceMedium {
		t.Fatalf("round trip output = %+v", output)
	}
}
func TestDoubleAndTripleCombinations(t *testing.T) {
	t.Parallel()
	makeCorrection := func(pattern PatternType, start, end int, startPrice, endPrice float64) WaveNode {
		direction := DirectionBearish
		startKind, endKind := PivotHigh, PivotLow
		if endPrice > startPrice {
			direction = DirectionBullish
			startKind, endKind = PivotLow, PivotHigh
		}
		points := []Pivot{pivot(start, startPrice, startKind), pivot(end, endPrice, endKind)}
		return WaveNode{
			ID:              string(pattern) + string(rune(start)),
			Pattern:         pattern,
			Mode:            ModeCorrective,
			Status:          StatusCompleted,
			Direction:       direction,
			OrthodoxStart:   points[0],
			OrthodoxEnd:     points[1],
			Pivots:          points,
			Conformance:     Conformance{Score: 1},
			RuleEvaluations: []RuleEvaluation{},
		}
	}
	nodes := []WaveNode{
		makeCorrection(PatternZigzag, 0, 10, 200, 160),
		makeCorrection(PatternFlatRegular, 10, 20, 160, 180),
		makeCorrection(PatternZigzag, 20, 30, 180, 140),
		makeCorrection(PatternFlatExpanded, 30, 40, 140, 170),
		makeCorrection(PatternZigzag, 40, 50, 170, 120),
	}
	combinations := parseCombinations(nodes, 1)
	assertPatternPresent(t, combinations, PatternDoubleZigzag)
	assertPatternPresent(t, combinations, PatternTripleZigzag)

	nodes[0].Pattern = PatternFlatRegular
	nodes[2].Pattern = PatternFlatExpanded
	nodes[4].Pattern = PatternTriangleContracting
	combinations = parseCombinations(nodes, 1)
	assertPatternPresent(t, combinations, PatternTripleThree)
}

func TestScaleAndMirrorInvariance(t *testing.T) {
	t.Parallel()

	original := bullishImpulse()
	scaled := make([]Pivot, len(original))
	for i, point := range original {
		scaled[i] = point
		scaled[i].Price *= 10
		scaled[i].BarIndex *= 3
		scaled[i].Time *= 3
	}
	originalNodes := parseMotiveWindow(0, original, nil)
	scaledNodes := parseMotiveWindow(0, scaled, nil)
	if len(originalNodes) != len(scaledNodes) {
		t.Fatalf("scale changed node count: %d != %d", len(originalNodes), len(scaledNodes))
	}
	mirrored := parseMotiveWindow(0, mirror(original, 130), nil)
	assertPatternPresent(t, mirrored, PatternImpulse)
}

func TestTargetConfluenceAndNoSyntheticTime(t *testing.T) {
	t.Parallel()

	node := buildNode(
		PatternDevelopingImpulseW5,
		ModeMotive,
		DirectionBullish,
		StatusDeveloping,
		0,
		bullishImpulse()[:5],
		nil,
		[]RuleEvaluation{evaluation(RuleMotiveWave2Limit, EvaluationPass, 0.5, "<= 1")},
	)
	levels, label, condition := projectedLevels(node, 1)
	if label != "W5" || condition == "" || len(levels) < 6 {
		t.Fatalf("unexpected W5 projections: label=%q levels=%d", label, len(levels))
	}
	zones := clusterTargetLevels(levels, label, condition, nil)
	if len(zones) == 0 {
		t.Fatal("expected target zones")
	}
	for _, zone := range zones {
		if zone.TimeWindow != nil {
			t.Error("time window must not be fabricated by price clustering")
		}
		if math.IsNaN(zone.MinPrice) || math.IsInf(zone.MaxPrice, 0) {
			t.Fatalf("invalid target zone: %+v", zone)
		}
	}
}

func TestObservableVolumeAndMomentumEvidence(t *testing.T) {
	t.Parallel()
	points := bullishImpulse()
	node := buildNode(
		PatternImpulse,
		ModeMotive,
		DirectionBullish,
		StatusCompleted,
		0,
		points,
		nil,
		[]RuleEvaluation{volumeNotObservable(), personalityNotObservable()},
	)
	candles := make([]market.Candle, 60)
	for index := range candles {
		volume := 100.0
		if index > points[2].BarIndex && index <= points[3].BarIndex {
			volume = 250
		}
		candles[index] = market.Candle{Volume: volume}
	}
	node = applyMarketEvidence(node, candles)
	for _, ruleID := range []string{GuideVolume, GuidePersonality} {
		found := false
		for _, result := range node.RuleEvaluations {
			if result.RuleID == ruleID {
				found = true
				if result.Status != EvaluationPass {
					t.Fatalf("%s status = %s, want PASS", ruleID, result.Status)
				}
			}
		}
		if !found {
			t.Fatalf("missing evidence for %s", ruleID)
		}
	}
}

func FuzzParserDoesNotReturnHardInvalidNodes(f *testing.F) {
	seed := bullishImpulse()
	f.Add(seed[0].Price, seed[1].Price, seed[2].Price, seed[3].Price, seed[4].Price, seed[5].Price)
	f.Fuzz(func(t *testing.T, p0, p1, p2, p3, p4, p5 float64) {
		values := []float64{p0, p1, p2, p3, p4, p5}
		points := make([]Pivot, 6)
		for i, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 1e12 {
				t.Skip()
			}
			kind := PivotLow
			if i%2 == 1 {
				kind = PivotHigh
			}
			points[i] = pivot(i*10, value, kind)
		}
		for _, node := range parseMotiveWindow(0, points, nil) {
			if node.Conformance.HardRulesFailed != 0 {
				t.Fatalf("hard-invalid node escaped parser: %+v", node)
			}
		}
	})
}

func assertPatternPresent(t *testing.T, nodes []WaveNode, pattern PatternType) {
	t.Helper()
	for _, node := range nodes {
		if node.Pattern == pattern {
			return
		}
	}
	t.Fatalf("pattern %s not found in %+v", pattern, nodes)
}

func assertPatternAbsent(t *testing.T, nodes []WaveNode, pattern PatternType) {
	t.Helper()
	for _, node := range nodes {
		if node.Pattern == pattern {
			t.Fatalf("unexpected pattern %s in %+v", pattern, nodes)
		}
	}
}
