package wave

import (
	"math"
	"testing"

	"WaveSight/internal/market"
)

func testCandles(count int) []market.Candle {
	result := make([]market.Candle, count)
	for index := range result {
		center := 100 + 0.08*float64(index) + math.Sin(float64(index)/4)*8
		result[index] = market.Candle{
			Time: int64(1_700_000_000 + index*3_600), BarIndex: index,
			Open: center - 0.4, High: center + 1.5, Low: center - 1.5,
			Close: center + 0.4, Volume: 1_000 + float64(index*20),
		}
	}
	return result
}

func TestPivotLatticePreservesAmbiguityProminenceAndProvisionalEdge(t *testing.T) {
	t.Parallel()
	candles := testCandles(80)
	// Force one outside bar to be both a local high and low.
	candles[35].High = candles[34].High + 20
	candles[35].Low = candles[34].Low - 20
	lattice := BuildPivotLattice(candles)
	if len(lattice.Branches) != 2 {
		t.Fatalf("ambiguous lattice branches = %d, want 2", len(lattice.Branches))
	}
	for _, branch := range lattice.Branches {
		if len(branch.Levels) == 0 || len(branch.Levels[0]) < 2 {
			t.Fatalf("empty lattice branch: %+v", branch)
		}
		last := branch.Levels[0][len(branch.Levels[0])-1]
		if last.State != PivotProvisional {
			t.Fatalf("last pivot state = %s, want PROVISIONAL", last.State)
		}
		for _, point := range branch.Levels[0] {
			if point.Time != candles[point.BarIndex].Time {
				t.Fatalf("pivot timestamp was shifted: %+v", point)
			}
		}
	}
	if got := BuildPivotLattice(candles[:2]); len(got.Branches) != 0 {
		t.Fatalf("short lattice branches = %d", len(got.Branches))
	}
}

func TestNormalizeAlternationKeepsMoreExtremeSameKindPivot(t *testing.T) {
	t.Parallel()
	input := []Pivot{
		pivot(10, 120, PivotHigh),
		pivot(5, 110, PivotHigh),
		pivot(15, 125, PivotHigh),
		pivot(20, 100, PivotLow),
	}
	got := normalizeAlternation(input)
	if len(got) != 2 || got[0].BarIndex != 15 || got[0].Price != 125 {
		t.Fatalf("normalizeAlternation() = %+v", got)
	}
	if samePivotSequence(got, input) {
		t.Fatal("different pivot sequences reported equal")
	}
	if !samePivotSequence(got, append([]Pivot(nil), got...)) {
		t.Fatal("identical pivot sequences reported different")
	}
}

func TestTargetEngineCoversEveryProjectionFamily(t *testing.T) {
	t.Parallel()
	candles := testCandles(100)
	future := make([]int64, 30)
	for index := range future {
		future[index] = candles[len(candles)-1].Time + int64(index+1)*3_600
	}
	equalDurationImpulse := []Pivot{
		pivot(0, 100, PivotLow), pivot(5, 120, PivotHigh),
		pivot(10, 110, PivotLow), pivot(15, 145, PivotHigh),
		pivot(20, 130, PivotLow), pivot(25, 155, PivotHigh),
	}
	baseInvalidations := []Invalidation{{ID: "origin", Kind: InvalidationPrice, Price: 99}}
	tests := []struct {
		name    string
		pattern PatternType
		points  []Pivot
	}{
		{name: "wave 2", pattern: PatternDevelopingImpulseW2, points: equalDurationImpulse[:2]},
		{name: "wave 3", pattern: PatternDevelopingImpulseW3, points: equalDurationImpulse[:3]},
		{name: "wave 4", pattern: PatternDevelopingImpulseW4, points: equalDurationImpulse[:4]},
		{name: "wave 5", pattern: PatternDevelopingImpulseW5, points: equalDurationImpulse[:5]},
		{name: "zigzag C", pattern: PatternDevelopingZigzagC, points: equalDurationImpulse[:3]},
		{name: "flat C", pattern: PatternDevelopingFlatC, points: []Pivot{
			pivot(0, 150, PivotHigh), pivot(5, 120, PivotLow), pivot(10, 155, PivotHigh),
		}},
		{name: "triangle D", pattern: PatternDevelopingTriangleD, points: equalDurationImpulse[:4]},
		{name: "triangle E", pattern: PatternDevelopingTriangleE, points: equalDurationImpulse[:5]},
		{name: "impulse complete", pattern: PatternImpulse, points: equalDurationImpulse},
		{name: "truncation complete", pattern: PatternTruncatedImpulse, points: equalDurationImpulse},
		{name: "leading complete", pattern: PatternLeadingDiagonal, points: equalDurationImpulse},
		{name: "ending complete", pattern: PatternEndingDiagonal, points: equalDurationImpulse},
		{name: "triangle thrust", pattern: PatternTriangleContracting, points: equalDurationImpulse},
		{name: "running triangle thrust", pattern: PatternTriangleRunning, points: equalDurationImpulse},
		{name: "zigzag complete", pattern: PatternZigzag, points: equalDurationImpulse[:4]},
		{name: "flat complete", pattern: PatternFlatExpanded, points: equalDurationImpulse[:4]},
		{name: "double three complete", pattern: PatternDoubleThree, points: equalDurationImpulse},
		{name: "triple zigzag complete", pattern: PatternTripleZigzag, points: equalDurationImpulse},
	}
	engine := NewTargetEngine()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			node := WaveNode{
				ID: "node", Pattern: test.pattern, Mode: ModeMotive,
				Direction: DirectionBullish, Status: StatusDeveloping,
				Pivots: test.points, OrthodoxStart: test.points[0],
				OrthodoxEnd: test.points[len(test.points)-1],
			}
			if test.pattern == PatternDevelopingImpulseW4 {
				node.Children = []WaveNode{{}, {}, {
					Pivots: []Pivot{
						pivot(10, 110, PivotLow), pivot(11, 120, PivotHigh),
						pivot(12, 115, PivotLow), pivot(13, 140, PivotHigh),
						pivot(14, 130, PivotLow),
					},
				}}
			}
			zones := engine.Build(node, candles, 0.01, future, baseInvalidations)
			if len(zones) == 0 {
				t.Fatalf("Build(%s) returned no zones", test.pattern)
			}
			for _, zone := range zones {
				if zone.MinPrice <= 0 || zone.MaxPrice <= 0 ||
					math.IsNaN(zone.MinPrice) || math.IsInf(zone.MaxPrice, 0) {
					t.Fatalf("invalid zone: %+v", zone)
				}
				if len(zone.InvalidationIDs) != 1 {
					t.Fatalf("zone invalidation IDs = %+v", zone.InvalidationIDs)
				}
			}
		})
	}
	if zones := engine.Build(WaveNode{}, nil, 0, nil, nil); zones != nil {
		t.Fatalf("empty node zones = %+v", zones)
	}
}

func TestTargetTimeWindowRequiresIndependentConfluence(t *testing.T) {
	t.Parallel()
	future := make([]int64, 20)
	for index := range future {
		future[index] = int64(1_800_000_000 + index*3_600)
	}
	confluent := WaveNode{Pivots: []Pivot{
		pivot(0, 100, PivotLow), pivot(5, 120, PivotHigh),
		pivot(10, 110, PivotLow), pivot(15, 140, PivotHigh),
		pivot(20, 130, PivotLow),
	}}
	window := buildTimeWindow(confluent, future)
	if window == nil || window.StartTime == 0 || len(window.Evidence) != 2 {
		t.Fatalf("confluent time window = %+v", window)
	}
	notConfluent := WaveNode{Pivots: []Pivot{
		pivot(0, 100, PivotLow), pivot(20, 120, PivotHigh),
		pivot(22, 110, PivotLow), pivot(52, 140, PivotHigh),
		pivot(72, 130, PivotLow),
	}}
	if got := buildTimeWindow(notConfluent, future); got != nil {
		t.Fatalf("non-confluent time window = %+v", got)
	}
	if got := buildTimeWindow(WaveNode{}, future); got != nil {
		t.Fatalf("short node time window = %+v", got)
	}
}

func TestEngineReturnsCurrentOrExplicitlyIndeterminateScenarios(t *testing.T) {
	t.Parallel()
	engine := NewEngine()
	empty := engine.Analyze(AnalyzeInput{Timeframe: market.Timeframe1h})
	if len(empty.Scenarios) != 1 || empty.Scenarios[0].Status != ScenarioIndeterminate {
		t.Fatalf("empty analysis scenarios = %+v", empty.Scenarios)
	}

	candles := testCandles(400)
	future := make([]int64, 50)
	for index := range future {
		future[index] = candles[len(candles)-1].Time + int64(index+1)*3_600
	}
	result := engine.Analyze(AnalyzeInput{
		Candles: candles, Timeframe: market.Timeframe1h, Session: market.SessionRTH,
		MaxScenarios: 5, FutureBars: future, TickSize: 0.01,
	})
	if len(result.Scenarios) == 0 || len(result.Scenarios) > 5 {
		t.Fatalf("scenario count = %d", len(result.Scenarios))
	}
	for _, scenario := range result.Scenarios {
		if scenario.Root.Conformance.HardRulesFailed != 0 {
			t.Fatalf("hard-invalid scenario escaped engine: %+v", scenario)
		}
	}
	if len(result.FutureBars) != len(future) {
		t.Fatalf("future bars = %d, want %d", len(result.FutureBars), len(future))
	}
}

func TestScenarioInvalidationsBiasAndCurrentPosition(t *testing.T) {
	t.Parallel()
	points := bullishImpulse()
	patterns := []PatternType{
		PatternDevelopingImpulseW2,
		PatternDevelopingImpulseW3,
		PatternDevelopingImpulseW4,
		PatternDevelopingImpulseW5,
		PatternDevelopingZigzagC,
		PatternDevelopingFlatC,
		PatternDevelopingTriangleD,
		PatternDevelopingTriangleE,
		PatternImpulse,
	}
	for _, pattern := range patterns {
		count := 6
		switch pattern {
		case PatternDevelopingImpulseW2:
			count = 2
		case PatternDevelopingImpulseW3, PatternDevelopingZigzagC, PatternDevelopingFlatC:
			count = 3
		case PatternDevelopingImpulseW4, PatternDevelopingTriangleD:
			count = 4
		case PatternDevelopingImpulseW5, PatternDevelopingTriangleE:
			count = 5
		}
		node := WaveNode{
			Pattern: pattern, Direction: DirectionBullish, Degree: DegreeIntermediate,
			Status: StatusDeveloping, Pivots: points[:count],
		}
		if len(buildInvalidations(node)) == 0 {
			t.Errorf("%s produced no invalidation", pattern)
		}
		if currentPosition(node) == "" || scenarioBias(node) == "" {
			t.Errorf("%s missing current presentation", pattern)
		}
	}
}

func TestDataQualityDetectsGapsAndLimitedHistory(t *testing.T) {
	t.Parallel()
	candles := testCandles(30)
	candles[20].Time += int64((24 * 60 * 60))
	quality := assessDataQuality(candles, market.Timeframe1h)
	if quality.MissingIntervals == 0 || len(quality.Warnings) < 2 {
		t.Fatalf("quality = %+v", quality)
	}
	if empty := assessDataQuality(nil, market.Timeframe1D); len(empty.Warnings) != 1 {
		t.Fatalf("empty quality = %+v", empty)
	}
}
