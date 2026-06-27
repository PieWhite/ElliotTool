package master

import (
	"math"
	"testing"
	"time"

	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
)

func TestCrossScaleGraphMergesTheSameOrthodoxEvents(t *testing.T) {
	start := time.Date(2026, 6, 26, 13, 30, 0, 0, time.UTC).Unix()
	end := start + 59*60
	builder := graphBuilder{
		events: make(map[string]*CanonicalWaveEvent),
		nodes:  make(map[string]*MasterWaveNode),
		views: map[market.Timeframe][]market.DerivedCandle{
			market.Timeframe1m: {
				{Candle: market.Candle{Time: start}, HighTime: start, LowTime: start},
				{Candle: market.Candle{Time: end}, HighTime: end, LowTime: end},
			},
			market.Timeframe1h: {
				{
					Candle: market.Candle{Time: start}, HighTime: end, LowTime: start,
					SourceFrom: start, SourceTo: end,
				},
				{
					Candle: market.Candle{Time: end}, HighTime: end, LowTime: end,
					SourceFrom: end, SourceTo: end,
				},
			},
		},
	}
	minuteNode := testNode(start, end, 0, 1)
	hourNode := testNode(start, end, 0, 1)
	minuteID := builder.registerNode(minuteNode, market.Timeframe1m)
	hourID := builder.registerNode(hourNode, market.Timeframe1h)
	if minuteID != hourID {
		t.Fatalf("same structure produced %q and %q", minuteID, hourID)
	}
	graph := builder.graph()
	if len(graph.Events) != 2 || len(graph.Nodes) != 1 {
		t.Fatalf("graph events/nodes = %d/%d", len(graph.Events), len(graph.Nodes))
	}
	if len(graph.Nodes[0].Resolutions) != 2 {
		t.Fatalf("node resolutions = %v", graph.Nodes[0].Resolutions)
	}
}

func TestEngineBuildsOneChartWideSnapshotAndSevenStableViews(t *testing.T) {
	t.Parallel()
	const count = 2_000
	start := time.Date(2026, 1, 2, 14, 30, 0, 0, time.UTC)
	minutes := make([]market.DerivedCandle, count)
	for index := range minutes {
		price := 100 + float64(index)*0.01 + math.Sin(float64(index)/13)*7
		timestamp := start.Add(time.Duration(index) * time.Minute).Unix()
		minutes[index] = market.DerivedCandle{
			Candle: market.Candle{
				Time: timestamp, BarIndex: index, Open: price - 0.2,
				High: price + 1, Low: price - 1, Close: price + 0.2,
				Volume: 1_000 + float64(index),
			},
			HighTime: timestamp, LowTime: timestamp,
			SourceFrom: timestamp, SourceTo: timestamp,
			Provenance: market.ProvenanceMinuteDerived,
		}
	}
	views := map[market.Timeframe][]market.DerivedCandle{
		market.Timeframe1m:  minutes,
		market.Timeframe5m:  sampleDerived(minutes, 5),
		market.Timeframe15m: sampleDerived(minutes, 15),
		market.Timeframe1h:  sampleDerived(minutes, 60),
		market.Timeframe4h:  sampleDerived(minutes, 240),
		market.Timeframe1D:  sampleDerived(minutes, 390),
		market.Timeframe1W:  sampleDerived(minutes, 1_950),
	}
	future := make(map[market.Timeframe][]int64, len(views))
	for timeframe, candles := range views {
		for offset := 1; offset <= 10; offset++ {
			future[timeframe] = append(
				future[timeframe], candles[len(candles)-1].Time+int64(offset)*60,
			)
		}
	}
	snapshot, projections := NewEngine().Analyze(AnalyzeInput{
		Symbol: "TEST", Session: market.SessionRTH,
		AsOf:           time.Unix(minutes[len(minutes)-1].Time, 0),
		FocusTimeframe: market.Timeframe1D, MaxScenarios: 5,
		Views: market.CanonicalViews{Views: views},
		Manifest: DatasetManifest{
			MinuteDetailFrom: minutes[0].Time,
			MinuteDetailTo:   minutes[len(minutes)-1].Time,
			NativeMinuteRows: len(minutes),
		},
		FutureBars: future,
	})
	if len(snapshot.ViewManifest) != 7 || len(projections) != 7 {
		t.Fatalf("view manifest/projections = %d/%d", len(snapshot.ViewManifest), len(projections))
	}
	if snapshot.InitialView.Timeframe != market.Timeframe1D {
		t.Fatalf("initial view = %s", snapshot.InitialView.Timeframe)
	}
	if len(snapshot.Scenarios) == 0 || len(snapshot.Scenarios) > 5 {
		t.Fatalf("scenario count = %d", len(snapshot.Scenarios))
	}
	for _, scenario := range snapshot.Scenarios {
		if scenario.Conformance.HardRulesFailed != 0 {
			t.Fatalf("hard-invalid scenario escaped: %+v", scenario)
		}
	}
}

func TestNarrativeIntervalsKeepUnexplainedAndUnobservableHistoryExplicit(t *testing.T) {
	events := map[string]CanonicalWaveEvent{
		"start": {ID: "start", OrthodoxTime: 100},
		"end":   {ID: "end", OrthodoxTime: 200},
	}
	nodes := map[string]MasterWaveNode{
		"counted": {ID: "counted", StartEventID: "start", EndEventID: "end"},
	}
	intervals := intervalsForContext(0, 300, []string{"counted"}, events, nodes, 75)
	if len(intervals) != 3 {
		t.Fatalf("intervals = %+v", intervals)
	}
	if intervals[0].Status != CoverageUncertain ||
		intervals[1].Status != CoverageObserved ||
		intervals[2].Status != CoverageUncertain {
		t.Fatalf("unexpected coverage statuses: %+v", intervals)
	}
	old := intervalsForContext(0, 50, nil, events, nodes, 75)
	if len(old) != 1 || old[0].Status != CoverageNotObservable {
		t.Fatalf("old detail gap = %+v", old)
	}
}

func TestScenarioDeduplicationCollapsesLeafOnlyVariants(t *testing.T) {
	values := []MasterScenario{
		{ID: "preferred", MaterialSignature: "same"},
		{ID: "leaf-variant", MaterialSignature: "same"},
		{ID: "material-alternative", MaterialSignature: "different"},
	}
	got := deduplicateScenarios(values, 5)
	if len(got) != 2 || got[0].ID != "preferred" || got[1].ID != "material-alternative" {
		t.Fatalf("deduplicated scenarios = %+v", got)
	}
}

func TestTimeframeProjectionPreservesScenarioAndKeepsMacroAncestor(t *testing.T) {
	t.Parallel()
	start := int64(1_700_000_000)
	candles := make([]market.DerivedCandle, 10)
	for index := range candles {
		timestamp := start + int64(index*60)
		candles[index] = market.DerivedCandle{
			Candle:   market.Candle{Time: timestamp, BarIndex: index, Open: 100, High: 102, Low: 99, Close: 101},
			HighTime: timestamp, LowTime: timestamp, SourceFrom: timestamp, SourceTo: timestamp,
		}
	}
	graph := MasterWaveGraph{
		Events: []CanonicalWaveEvent{
			{ID: "start", OrthodoxTime: candles[0].Time, OrthodoxPrice: 100},
			{ID: "end", OrthodoxTime: candles[9].Time, OrthodoxPrice: 110},
		},
		Nodes: []MasterWaveNode{{
			ID: "macro", Degree: wave.DegreePrimary, StartEventID: "start",
			EndEventID: "end", PivotEventIDs: []string{"start", "end"},
		}},
	}
	scenarios := []MasterScenario{{
		ID:              "scenario-stable",
		ActivePath:      []string{"macro"},
		ObservationRoot: ObservationRoot{ContextSequence: []string{"macro"}},
	}}
	input := AnalyzeInput{
		FocusTimeframe: market.Timeframe1W,
		Views: market.CanonicalViews{Views: map[market.Timeframe][]market.DerivedCandle{
			market.Timeframe1W: candles,
			market.Timeframe1h: candles,
		}},
		FutureBars: map[market.Timeframe][]int64{},
	}
	_, views := buildViews(input, graph, scenarios)
	if scenarios[0].ID != "scenario-stable" {
		t.Fatalf("view projection changed scenario identity: %+v", scenarios[0])
	}
	for _, timeframe := range []market.Timeframe{market.Timeframe1W, market.Timeframe1h} {
		projected := views[timeframe]
		if len(projected.AncestorNodeIDs) != 1 || projected.AncestorNodeIDs[0] != "macro" {
			t.Fatalf("%s ancestors = %v", timeframe, projected.AncestorNodeIDs)
		}
	}
}

func testNode(start, end int64, startIndex, endIndex int) wave.WaveNode {
	startPivot := wave.Pivot{
		Time: start, BarIndex: startIndex, Price: 100,
		Kind: wave.PivotLow, State: wave.PivotConfirmed,
	}
	endPivot := wave.Pivot{
		Time: end, BarIndex: endIndex, Price: 110,
		Kind: wave.PivotHigh, State: wave.PivotConfirmed,
	}
	return wave.WaveNode{
		Pattern: wave.PatternDevelopingImpulseW2, Mode: wave.ModeMotive,
		Direction: wave.DirectionBullish, Status: wave.StatusDeveloping,
		OrthodoxStart: startPivot, OrthodoxEnd: endPivot,
		Pivots:      []wave.Pivot{startPivot, endPivot},
		Conformance: wave.Conformance{HardRulesPassed: 1, StructuralCoverage: 1, Score: 1},
	}
}
