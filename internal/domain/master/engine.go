package master

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
)

var timeframeOrder = []market.Timeframe{
	market.Timeframe1W,
	market.Timeframe1D,
	market.Timeframe4h,
	market.Timeframe1h,
	market.Timeframe15m,
	market.Timeframe5m,
	market.Timeframe1m,
}

type AnalyzeInput struct {
	Symbol           string
	Session          market.Session
	AsOf             time.Time
	FocusTimeframe   market.Timeframe
	MaxScenarios     int
	ParentSnapshotID string
	Views            market.CanonicalViews
	Manifest         DatasetManifest
	FutureBars       map[market.Timeframe][]int64
}

type Engine struct {
	wave *wave.Engine
}

func NewEngine() *Engine {
	return &Engine{wave: wave.NewEngine()}
}

type sourcedNode struct {
	node      wave.WaveNode
	timeframe market.Timeframe
	startTime int64
	endTime   int64
	masterID  string
}

type graphBuilder struct {
	events map[string]*CanonicalWaveEvent
	nodes  map[string]*MasterWaveNode
	views  map[market.Timeframe][]market.DerivedCandle
}

func (e *Engine) Analyze(input AnalyzeInput) (AnalysisSnapshot, map[market.Timeframe]TimeframeView) {
	if input.MaxScenarios < 1 || input.MaxScenarios > 5 {
		input.MaxScenarios = 5
	}
	builder := &graphBuilder{
		events: make(map[string]*CanonicalWaveEvent, 2_048),
		nodes:  make(map[string]*MasterWaveNode, 4_096),
		views:  input.Views.Views,
	}
	all := make([]sourcedNode, 0, 8_192)
	qualityByView := make(map[market.Timeframe]wave.DataQuality, len(timeframeOrder))
	for _, timeframe := range timeframeOrder {
		candles := market.PlainCandles(input.Views.Views[timeframe])
		if len(candles) < 20 {
			continue
		}
		nodes, quality := e.wave.ParseAll(candles, timeframe)
		qualityByView[timeframe] = quality
		for _, node := range nodes {
			id := builder.registerNode(node, timeframe)
			masterNode := builder.nodes[id]
			all = append(all, sourcedNode{
				node: node, timeframe: timeframe, startTime: builder.events[masterNode.StartEventID].OrthodoxTime,
				endTime: builder.events[masterNode.EndEventID].OrthodoxTime, masterID: id,
			})
		}
	}
	builder.inferDegrees()
	graph := builder.graph()
	scenarios := e.rankScenarios(input, graph, all, qualityByView)
	graph = pruneGraph(graph, scenarios)
	viewManifest, views := buildViews(input, graph, scenarios)
	initial := views[input.FocusTimeframe]
	snapshot := AnalysisSnapshot{
		ParentSnapshotID: input.ParentSnapshotID,
		TheoryVersion:    wave.TheoryVersion, EngineVersion: EngineVersion,
		GeneratedAt: time.Now().UTC().Unix(),
		Request: AnalysisRequest{
			Symbol: input.Symbol, Session: input.Session, AsOf: input.AsOf.UTC().Format(time.RFC3339),
			FocusTimeframe: input.FocusTimeframe, HistoryProfile: HistoryMaxDailyTwoYearMinute,
			MaxScenarios: input.MaxScenarios,
		},
		DatasetManifest: input.Manifest, Graph: graph, Scenarios: scenarios,
		ViewManifest: viewManifest, InitialView: initial,
	}
	return snapshot, views
}

func (b *graphBuilder) registerNode(node wave.WaveNode, timeframe market.Timeframe) string {
	pivotIDs := make([]string, 0, len(node.Pivots))
	for _, pivot := range node.Pivots {
		pivotIDs = append(pivotIDs, b.registerEvent(pivot, timeframe))
	}
	childIDs := make([]string, 0, len(node.Children))
	for _, child := range node.Children {
		childIDs = append(childIDs, b.registerNode(child, timeframe))
	}
	raw := strings.Join([]string{
		string(node.Pattern), string(node.Direction), string(node.Status),
		strings.Join(pivotIDs, ","), strings.Join(childIDs, ","),
	}, "|")
	id := "wave-" + shortHash(raw)
	if current, exists := b.nodes[id]; exists {
		current.Resolutions = appendUniqueTimeframe(current.Resolutions, timeframe)
		if richerNode(node, current.SourceNode) {
			current.SourceNode = node
			current.RuleEvaluations = append([]wave.RuleEvaluation(nil), node.RuleEvaluations...)
			current.Measurements = append([]wave.Measurement(nil), node.Measurements...)
			current.Conformance = node.Conformance
			current.ChildIDs = childIDs
		}
		return id
	}
	startID, endID := "", ""
	if len(pivotIDs) > 0 {
		startID, endID = pivotIDs[0], pivotIDs[len(pivotIDs)-1]
	}
	b.nodes[id] = &MasterWaveNode{
		ID: id, Pattern: node.Pattern, Mode: node.Mode, Function: node.Function,
		Direction: node.Direction, Degree: node.Degree, Status: node.Status, Label: node.Label,
		StartEventID: startID, EndEventID: endID, PivotEventIDs: pivotIDs, ChildIDs: childIDs,
		Resolutions: []market.Timeframe{timeframe}, OrthodoxStart: node.OrthodoxStart,
		OrthodoxEnd: node.OrthodoxEnd, Measurements: append([]wave.Measurement(nil), node.Measurements...),
		RuleEvaluations: append([]wave.RuleEvaluation(nil), node.RuleEvaluations...),
		Conformance:     node.Conformance, SourceNode: node,
	}
	return id
}

func (b *graphBuilder) registerEvent(pivot wave.Pivot, timeframe market.Timeframe) string {
	eventTime := pivot.Time
	provenance := market.ProvenanceNativeDaily
	barTime := pivot.Time
	view := b.views[timeframe]
	if pivot.BarIndex >= 0 && pivot.BarIndex < len(view) {
		bar := view[pivot.BarIndex]
		barTime = bar.Time
		provenance = bar.Provenance
		if pivot.Kind == wave.PivotHigh {
			eventTime = bar.HighTime
		} else {
			eventTime = bar.LowTime
		}
	}
	key := string(pivot.Kind) + "|" + strconv.FormatInt(eventTime, 10) + "|" + strconv.FormatFloat(pivot.Price, 'g', 12, 64)
	id := "event-" + shortHash(key)
	source := EventSource{Timeframe: timeframe, BarTime: barTime, Price: pivot.Price, Provenance: provenance}
	state := EventConfirmed
	if pivot.State == wave.PivotProvisional {
		state = EventProvisional
	} else if pivot.State == wave.PivotAmbiguous {
		state = EventAmbiguous
	}
	if current, exists := b.events[id]; exists {
		current.Resolutions = appendUniqueTimeframe(current.Resolutions, timeframe)
		if !hasEventSource(current.Sources, source) {
			current.Sources = append(current.Sources, source)
		}
		delta := math.Abs(pivot.Price - current.OrthodoxPrice)
		if delta > current.MaxPriceDelta {
			current.MaxPriceDelta = delta
		}
		timeDelta := absInt64(eventTime - current.OrthodoxTime)
		if timeDelta > current.MaxTimeDelta {
			current.MaxTimeDelta = timeDelta
		}
		if state == EventAmbiguous || (state == EventProvisional && current.State == EventConfirmed) {
			current.State = state
		}
		return id
	}
	b.events[id] = &CanonicalWaveEvent{
		ID: id, Kind: pivot.Kind, State: state, TimeFrom: eventTime, TimeTo: eventTime,
		OrthodoxTime: eventTime, OrthodoxPrice: pivot.Price,
		Resolutions: []market.Timeframe{timeframe}, Sources: []EventSource{source},
	}
	return id
}

func (b *graphBuilder) inferDegrees() {
	if len(b.nodes) == 0 {
		return
	}
	minSpan, maxSpan := int64(math.MaxInt64), int64(1)
	spans := make(map[string]int64, len(b.nodes))
	for id, node := range b.nodes {
		span := maxInt64(1, b.events[node.EndEventID].OrthodoxTime-b.events[node.StartEventID].OrthodoxTime)
		spans[id] = span
		if span < minSpan {
			minSpan = span
		}
		if span > maxSpan {
			maxSpan = span
		}
	}
	degrees := []wave.Degree{
		wave.DegreeObservableLeaf, wave.DegreeSubminuette, wave.DegreeMinuette,
		wave.DegreeMinute, wave.DegreeMinor, wave.DegreeIntermediate,
		wave.DegreePrimary, wave.DegreeCycle, wave.DegreeSupercycle,
		wave.DegreeGrandSupercycle,
	}
	denominator := math.Log(float64(maxSpan) / float64(minSpan))
	for id, node := range b.nodes {
		index := 0
		if denominator > 0 {
			index = int(math.Round(math.Log(float64(spans[id])/float64(minSpan)) / denominator * float64(len(degrees)-1)))
		}
		if index < 0 {
			index = 0
		}
		if index >= len(degrees) {
			index = len(degrees) - 1
		}
		node.Degree = degrees[index]
	}
}

func (b *graphBuilder) graph() MasterWaveGraph {
	events := make([]CanonicalWaveEvent, 0, len(b.events))
	for _, event := range b.events {
		sort.Slice(event.Resolutions, func(i, j int) bool {
			return timeframeRank(event.Resolutions[i]) < timeframeRank(event.Resolutions[j])
		})
		sort.Slice(event.Sources, func(i, j int) bool {
			return timeframeRank(event.Sources[i].Timeframe) < timeframeRank(event.Sources[j].Timeframe)
		})
		events = append(events, *event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].OrthodoxTime != events[j].OrthodoxTime {
			return events[i].OrthodoxTime < events[j].OrthodoxTime
		}
		return events[i].ID < events[j].ID
	})
	nodes := make([]MasterWaveNode, 0, len(b.nodes))
	for _, node := range b.nodes {
		sort.Slice(node.Resolutions, func(i, j int) bool {
			return timeframeRank(node.Resolutions[i]) < timeframeRank(node.Resolutions[j])
		})
		nodes = append(nodes, *node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return MasterWaveGraph{Events: events, Nodes: nodes}
}

func (e *Engine) rankScenarios(
	input AnalyzeInput,
	graph MasterWaveGraph,
	all []sourcedNode,
	quality map[market.Timeframe]wave.DataQuality,
) []MasterScenario {
	if len(all) == 0 || len(graph.Events) == 0 {
		return []MasterScenario{indeterminateScenario(graph)}
	}
	eventByID, nodeByID := graphIndexes(graph)
	latestByTimeframe := make(map[market.Timeframe]int64)
	for _, candidate := range all {
		if candidate.endTime > latestByTimeframe[candidate.timeframe] {
			latestByTimeframe[candidate.timeframe] = candidate.endTime
		}
	}
	roots := make([]sourcedNode, 0, 128)
	seen := make(map[string]struct{})
	for _, candidate := range all {
		if candidate.endTime != latestByTimeframe[candidate.timeframe] {
			continue
		}
		if _, exists := seen[candidate.masterID]; exists {
			continue
		}
		seen[candidate.masterID] = struct{}{}
		roots = append(roots, candidate)
	}
	observationFrom, observationTo := graph.Events[0].OrthodoxTime, graph.Events[len(graph.Events)-1].OrthodoxTime
	completed := completedCandidates(graph, eventByID)
	scenarios := make([]MasterScenario, 0, len(roots))
	for _, root := range roots {
		node := nodeByID[root.masterID]
		sourceNode := root.node
		context := selectContext(completed, node, eventByID, observationFrom, observationTo)
		activePath := activePath(node.ID, nodeByID)
		invalidations := wave.InvalidationsFor(sourceNode)
		candles := market.PlainCandles(input.Views.Views[root.timeframe])
		targets := e.wave.BuildTargets(
			sourceNode, candles, 0.01, input.FutureBars[root.timeframe], invalidations,
		)
		observation := buildObservationRoot(
			observationFrom, observationTo, context, eventByID,
			nodeByID, input.Manifest.MinuteDetailFrom,
		)
		scenario := MasterScenario{
			ID:   "scenario-" + shortHash(node.ID+"|"+strings.Join(context, ",")),
			Bias: wave.BiasFor(sourceNode), CurrentPosition: wave.PositionFor(sourceNode),
			Conformance:     scenarioConformance(context, node, nodeByID),
			ObservationRoot: observation, ActivePath: activePath,
			Invalidations: invalidations, TargetLadder: targets,
		}
		scenario.Audit = buildAudit(
			scenario, graph, quality, nodeByID,
			observationFrom, input.Manifest.MinuteDetailFrom,
		)
		scenario.MaterialSignature = materialSignature(
			scenario, nodeByID, input.FocusTimeframe,
		)
		scenarios = append(scenarios, scenario)
	}
	sort.SliceStable(scenarios, func(i, j int) bool {
		return scenarioLess(scenarios[i], scenarios[j], nodeByID)
	})
	scenarios = deduplicateScenarios(scenarios, input.MaxScenarios)
	if len(scenarios) == 0 {
		return []MasterScenario{indeterminateScenario(graph)}
	}
	for index := range scenarios {
		scenarios[index].Rank = index + 1
		scenarios[index].Status = wave.ScenarioAlternate
		if index == 0 {
			scenarios[index].Status = wave.ScenarioPreferred
		}
	}
	if len(scenarios) > 1 {
		comparison := compareScenarios(scenarios[0], scenarios[1], nodeByID)
		scenarios[0].Audit.WhyPreferred = &comparison
	}
	return scenarios
}

func completedCandidates(graph MasterWaveGraph, events map[string]CanonicalWaveEvent) []MasterWaveNode {
	result := make([]MasterWaveNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node.Status != wave.StatusCompleted || node.Conformance.HardRulesFailed > 0 {
			continue
		}
		if events[node.EndEventID].OrthodoxTime <= events[node.StartEventID].OrthodoxTime {
			continue
		}
		result = append(result, node)
	}
	return result
}

// selectContext is a deterministic weighted interval scheduler. It explains
// the entire observed history with non-overlapping completed structures while
// leaving genuine gaps explicit instead of inventing waves.
func selectContext(
	candidates []MasterWaveNode,
	active MasterWaveNode,
	events map[string]CanonicalWaveEvent,
	from, to int64,
) []string {
	filtered := make([]MasterWaveNode, 0, len(candidates))
	activeStart := events[active.StartEventID].OrthodoxTime
	for _, node := range candidates {
		end := events[node.EndEventID].OrthodoxTime
		if end <= activeStart && end >= from {
			filtered = append(filtered, node)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		leftEnd := events[filtered[i].EndEventID].OrthodoxTime
		rightEnd := events[filtered[j].EndEventID].OrthodoxTime
		if leftEnd != rightEnd {
			return leftEnd < rightEnd
		}
		return filtered[i].ID < filtered[j].ID
	})
	if len(filtered) > 600 {
		filtered = filtered[len(filtered)-600:]
	}
	previous := make([]int, len(filtered))
	for index := range filtered {
		previous[index] = -1
		start := events[filtered[index].StartEventID].OrthodoxTime
		for scan := index - 1; scan >= 0; scan-- {
			if events[filtered[scan].EndEventID].OrthodoxTime <= start {
				previous[index] = scan
				break
			}
		}
	}
	best := make([]float64, len(filtered)+1)
	take := make([]bool, len(filtered))
	for index, node := range filtered {
		start := events[node.StartEventID].OrthodoxTime
		end := events[node.EndEventID].OrthodoxTime
		spanWeight := float64(end-start) / float64(maxInt64(1, to-from))
		weight := spanWeight*100 + node.Conformance.StructuralCoverage*5 +
			float64(node.Conformance.GuidelinesPassed+node.Conformance.RatioConfluences)
		include := weight
		if previous[index] >= 0 {
			include += best[previous[index]+1]
		}
		if include > best[index] {
			best[index+1] = include
			take[index] = true
		} else {
			best[index+1] = best[index]
		}
	}
	selected := make([]string, 0, 32)
	for index := len(filtered) - 1; index >= 0; {
		if take[index] {
			selected = append(selected, filtered[index].ID)
			index = previous[index]
		} else {
			index--
		}
	}
	for left, right := 0, len(selected)-1; left < right; left, right = left+1, right-1 {
		selected[left], selected[right] = selected[right], selected[left]
	}
	selected = append(selected, active.ID)
	return selected
}

func buildObservationRoot(
	from, to int64,
	context []string,
	events map[string]CanonicalWaveEvent,
	nodes map[string]MasterWaveNode,
	minuteDetailFrom int64,
) ObservationRoot {
	return ObservationRoot{
		From: from, To: to, OpenLeftBoundary: true,
		ContextSequence: append([]string(nil), context...),
		Intervals:       intervalsForContext(from, to, context, events, nodes, minuteDetailFrom),
	}
}

func intervalsForContext(
	from, to int64,
	context []string,
	events map[string]CanonicalWaveEvent,
	nodes map[string]MasterWaveNode,
	minuteDetailFrom int64,
) []NarrativeInterval {
	if len(context) == 0 {
		return []NarrativeInterval{gapInterval(from, to, minuteDetailFrom)}
	}
	result := make([]NarrativeInterval, 0, len(context)*2+1)
	cursor := from
	for _, id := range context {
		node, exists := nodes[id]
		if !exists {
			continue
		}
		start, startOK := events[node.StartEventID]
		end, endOK := events[node.EndEventID]
		if !startOK || !endOK || end.OrthodoxTime <= start.OrthodoxTime {
			continue
		}
		nodeFrom := maxInt64(from, start.OrthodoxTime)
		nodeTo := end.OrthodoxTime
		if nodeFrom > cursor {
			result = append(result, gapInterval(cursor, nodeFrom, minuteDetailFrom))
		}
		if nodeTo > cursor {
			result = append(result, NarrativeInterval{
				From: nodeFrom, To: nodeTo, Status: CoverageObserved, NodeID: id,
				Explanation: "Counted by a hard-rule-valid Elliott structure in the master assignment.",
			})
			cursor = nodeTo
		}
	}
	if cursor < to {
		result = append(result, gapInterval(cursor, to, minuteDetailFrom))
	}
	return result
}

func gapInterval(from, to, minuteDetailFrom int64) NarrativeInterval {
	status := CoverageUncertain
	explanation := "Prices are observed, but no rule-valid completed structure explains this interval."
	if to <= minuteDetailFrom {
		status = CoverageNotObservable
		explanation = "Daily structure is observed; lower-degree minute subdivisions are not loaded."
	}
	return NarrativeInterval{From: from, To: to, Status: status, Explanation: explanation}
}

func activePath(rootID string, nodes map[string]MasterWaveNode) []string {
	path := []string{rootID}
	currentID := rootID
	seen := map[string]struct{}{rootID: {}}
	for {
		current := nodes[currentID]
		parentID := ""
		parentRank := math.MaxInt
		for id, candidate := range nodes {
			if _, exists := seen[id]; exists || !containsString(candidate.ChildIDs, currentID) {
				continue
			}
			rank := degreeRank(candidate.Degree)
			if rank > degreeRank(current.Degree) && rank < parentRank {
				parentID, parentRank = id, rank
			}
		}
		if parentID == "" {
			break
		}
		path = append([]string{parentID}, path...)
		seen[parentID] = struct{}{}
		currentID = parentID
	}
	return path
}

func scenarioConformance(context []string, active MasterWaveNode, nodes map[string]MasterWaveNode) wave.Conformance {
	result := active.Conformance
	if len(context) == 0 {
		return result
	}
	totalCoverage := active.Conformance.StructuralCoverage
	for _, id := range context {
		if id == active.ID {
			continue
		}
		node, exists := nodes[id]
		if !exists {
			continue
		}
		result.HardRulesPassed += node.Conformance.HardRulesPassed
		result.GuidelinesPassed += node.Conformance.GuidelinesPassed
		result.GuidelinesFailed += node.Conformance.GuidelinesFailed
		result.NotObservable += node.Conformance.NotObservable
		result.RatioConfluences += node.Conformance.RatioConfluences
		totalCoverage += node.Conformance.StructuralCoverage
	}
	result.StructuralCoverage = totalCoverage / float64(len(context))
	result.Score = math.Min(1, result.StructuralCoverage*0.65+
		float64(result.GuidelinesPassed+result.RatioConfluences)/float64(maxInt(1, result.HardRulesPassed+result.GuidelinesPassed+result.RatioConfluences))*0.35)
	return result
}

func buildAudit(
	scenario MasterScenario,
	graph MasterWaveGraph,
	quality map[market.Timeframe]wave.DataQuality,
	nodes map[string]MasterWaveNode,
	observationFrom, minuteDetailFrom int64,
) ScenarioAudit {
	labels := make([]string, 0, len(scenario.ActivePath))
	for _, id := range scenario.ActivePath {
		if node, exists := nodes[id]; exists {
			labels = append(labels, degreeNotation(node))
		}
	}
	rows := make([]TimeframeEvidence, 0, len(timeframeOrder))
	for _, timeframe := range timeframeOrder {
		row := TimeframeEvidence{
			Timeframe: timeframe, Position: strings.Join(labels, " → "),
			Coverage: CoverageObserved, Status: "CONSISTENT_MASTER_ASSIGNMENT",
		}
		if _, exists := quality[timeframe]; !exists {
			row.Coverage = CoverageNotObservable
			row.Status = "DETAIL_NOT_LOADED"
		} else if timeframe != market.Timeframe1W && timeframe != market.Timeframe1D &&
			observationFrom < minuteDetailFrom {
			row.Coverage = CoverageNotObservable
			row.Status = "PARTIAL_DETAIL_COVERAGE"
		}
		for _, id := range scenario.ActivePath {
			node, exists := nodes[id]
			if !exists {
				continue
			}
			if containsTimeframe(node.Resolutions, timeframe) {
				row.EndpointAligned = true
				row.VisibleChildren += len(node.ChildIDs)
				if row.ParentNodeID == "" {
					row.ParentNodeID = node.ID
				}
			}
		}
		rows = append(rows, row)
	}
	return ScenarioAudit{GlobalThesis: strings.Join(labels, " → "), CrossTimeframeEvidence: rows}
}

func degreeNotation(node MasterWaveNode) string {
	label := node.Label
	switch node.Pattern {
	case wave.PatternDevelopingImpulseW2:
		label = "wave 1 complete; expecting wave 2"
	case wave.PatternDevelopingImpulseW3:
		label = "wave 2 complete; expecting wave 3"
	case wave.PatternDevelopingImpulseW4:
		label = "wave 3 complete; expecting wave 4"
	case wave.PatternDevelopingImpulseW5:
		label = "wave 4 complete; expecting wave 5"
	case wave.PatternDevelopingZigzagC:
		label = "zigzag A-B complete; expecting C"
	case wave.PatternDevelopingFlatC:
		label = "flat A-B complete; expecting C"
	case wave.PatternDevelopingTriangleD:
		label = "triangle A-B-C complete; expecting D"
	case wave.PatternDevelopingTriangleE:
		label = "triangle A-B-C-D complete; expecting E"
	}
	if label == "" {
		label = strings.ReplaceAll(string(node.Pattern), "_", " ")
	}
	return fmt.Sprintf("%s %s", humanDegree(node.Degree), label)
}

func humanDegree(degree wave.Degree) string {
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(string(degree), "_", " ")))
	for index := range words {
		words[index] = strings.ToUpper(words[index][:1]) + words[index][1:]
	}
	return strings.Join(words, " ")
}

func materialSignature(
	scenario MasterScenario,
	nodes map[string]MasterWaveNode,
	focus market.Timeframe,
) string {
	parts := []string{string(scenario.Bias)}
	for _, id := range scenario.ObservationRoot.ContextSequence {
		if node, exists := nodes[id]; exists && degreeVisibleOnView(node.Degree, focus) {
			parts = append(parts, "context", string(node.Pattern), string(node.Degree))
		}
	}
	for _, id := range scenario.ActivePath {
		if node, exists := nodes[id]; exists && degreeVisibleOnView(node.Degree, focus) {
			parts = append(parts, string(node.Pattern), string(node.Degree))
		}
	}
	for _, invalidation := range scenario.Invalidations {
		parts = append(parts, invalidation.ID, strconv.FormatFloat(invalidation.Price, 'f', 4, 64))
	}
	for _, target := range scenario.TargetLadder {
		parts = append(parts, target.WaveLabel,
			strconv.FormatFloat(target.MinPrice, 'f', 4, 64),
			strconv.FormatFloat(target.MaxPrice, 'f', 4, 64))
	}
	return shortHash(strings.Join(parts, "|"))
}

func scenarioLess(left, right MasterScenario, nodes map[string]MasterWaveNode) bool {
	if left.Conformance.HardRulesFailed != right.Conformance.HardRulesFailed {
		return left.Conformance.HardRulesFailed < right.Conformance.HardRulesFailed
	}
	if left.Conformance.StructuralCoverage != right.Conformance.StructuralCoverage {
		return left.Conformance.StructuralCoverage > right.Conformance.StructuralCoverage
	}
	leftCross, rightCross := crossScaleCount(left, nodes), crossScaleCount(right, nodes)
	if leftCross != rightCross {
		return leftCross > rightCross
	}
	if left.Conformance.GuidelinesPassed != right.Conformance.GuidelinesPassed {
		return left.Conformance.GuidelinesPassed > right.Conformance.GuidelinesPassed
	}
	if left.Conformance.RatioConfluences != right.Conformance.RatioConfluences {
		return left.Conformance.RatioConfluences > right.Conformance.RatioConfluences
	}
	if len(left.ActivePath) != len(right.ActivePath) {
		return len(left.ActivePath) < len(right.ActivePath)
	}
	return left.ID < right.ID
}

func crossScaleCount(scenario MasterScenario, nodes map[string]MasterWaveNode) int {
	count := 0
	for _, id := range scenario.ActivePath {
		count += len(nodes[id].Resolutions)
	}
	return count
}

func deduplicateScenarios(values []MasterScenario, limit int) []MasterScenario {
	seen := make(map[string]struct{}, len(values))
	result := make([]MasterScenario, 0, minInt(limit, len(values)))
	for _, value := range values {
		if _, exists := seen[value.MaterialSignature]; exists {
			continue
		}
		seen[value.MaterialSignature] = struct{}{}
		result = append(result, value)
		if len(result) == limit {
			break
		}
	}
	return result
}

func compareScenarios(preferred, alternate MasterScenario, nodes map[string]MasterWaveNode) AlternativeComparison {
	first := "The assignments diverge in their active wave interpretation."
	for index := 0; index < minInt(len(preferred.ActivePath), len(alternate.ActivePath)); index++ {
		if preferred.ActivePath[index] != alternate.ActivePath[index] {
			left := nodes[preferred.ActivePath[index]]
			right := nodes[alternate.ActivePath[index]]
			first = fmt.Sprintf("%s %s versus %s %s", left.Degree, left.Label, right.Degree, right.Label)
			break
		}
	}
	evidence := []string{
		fmt.Sprintf("%.0f%% average structural coverage", preferred.Conformance.StructuralCoverage*100),
		fmt.Sprintf("%d cross-structure guidelines passed", preferred.Conformance.GuidelinesPassed),
		fmt.Sprintf("%d Fibonacci relations support the assignment", preferred.Conformance.RatioConfluences),
	}
	return AlternativeComparison{
		AlternativeID: alternate.ID, FirstDivergence: first, PreferredEvidence: evidence,
		DifferentTargets: targetSignature(preferred.TargetLadder) != targetSignature(alternate.TargetLadder),
		DifferentBias:    preferred.Bias != alternate.Bias,
	}
}

func targetSignature(targets []wave.TargetZone) string {
	parts := make([]string, 0, len(targets)*3)
	for _, target := range targets {
		parts = append(parts, target.WaveLabel,
			strconv.FormatFloat(target.MinPrice, 'f', 4, 64),
			strconv.FormatFloat(target.MaxPrice, 'f', 4, 64))
	}
	return strings.Join(parts, "|")
}

func indeterminateScenario(graph MasterWaveGraph) MasterScenario {
	from, to := int64(0), int64(0)
	if len(graph.Events) > 0 {
		from, to = graph.Events[0].OrthodoxTime, graph.Events[len(graph.Events)-1].OrthodoxTime
	}
	return MasterScenario{
		ID: "indeterminate", Rank: 1, Status: wave.ScenarioIndeterminate,
		CurrentPosition: "No coherent, rule-valid master assignment is observable.",
		ObservationRoot: ObservationRoot{
			From: from, To: to, OpenLeftBoundary: true,
			Intervals: []NarrativeInterval{{
				From: from, To: to, Status: CoverageNotObservable,
				Explanation: "WaveSight did not manufacture subdivisions unsupported by the available data.",
			}},
		},
		Audit:             ScenarioAudit{GlobalThesis: "Indeterminate — more confirmed structure is required."},
		MaterialSignature: "indeterminate",
	}
}

func graphIndexes(graph MasterWaveGraph) (map[string]CanonicalWaveEvent, map[string]MasterWaveNode) {
	events := make(map[string]CanonicalWaveEvent, len(graph.Events))
	for _, event := range graph.Events {
		events[event.ID] = event
	}
	nodes := make(map[string]MasterWaveNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	return events, nodes
}

func pruneGraph(graph MasterWaveGraph, scenarios []MasterScenario) MasterWaveGraph {
	keepNodes := make(map[string]struct{})
	nodeByID := make(map[string]MasterWaveNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodeByID[node.ID] = node
	}
	var include func(string)
	include = func(id string) {
		if _, exists := keepNodes[id]; exists {
			return
		}
		node, exists := nodeByID[id]
		if !exists {
			return
		}
		keepNodes[id] = struct{}{}
		for _, child := range node.ChildIDs {
			include(child)
		}
	}
	for _, scenario := range scenarios {
		for _, id := range scenario.ObservationRoot.ContextSequence {
			include(id)
		}
		for _, id := range scenario.ActivePath {
			include(id)
		}
	}
	events := make(map[string]struct{})
	nodes := make([]MasterWaveNode, 0, len(keepNodes))
	for _, node := range graph.Nodes {
		if _, exists := keepNodes[node.ID]; !exists {
			continue
		}
		nodes = append(nodes, node)
		for _, id := range node.PivotEventIDs {
			events[id] = struct{}{}
		}
	}
	filteredEvents := make([]CanonicalWaveEvent, 0, len(events))
	for _, event := range graph.Events {
		if _, exists := events[event.ID]; exists {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return MasterWaveGraph{Events: filteredEvents, Nodes: nodes}
}

func buildViews(
	input AnalyzeInput,
	graph MasterWaveGraph,
	scenarios []MasterScenario,
) ([]ViewManifestItem, map[market.Timeframe]TimeframeView) {
	events, nodes := graphIndexes(graph)
	activeAncestors := make(map[string]struct{})
	for _, scenario := range scenarios {
		for _, id := range scenario.ActivePath {
			activeAncestors[id] = struct{}{}
		}
	}
	manifest := make([]ViewManifestItem, 0, len(timeframeOrder))
	views := make(map[market.Timeframe]TimeframeView, len(timeframeOrder))
	for _, timeframe := range timeframeOrder {
		candles := input.Views.Views[timeframe]
		item := ViewManifestItem{Timeframe: timeframe, CandleCount: len(candles)}
		if len(candles) > 0 {
			item.From = candles[0].Time
			item.To = candles[len(candles)-1].Time
		}
		manifest = append(manifest, item)
		visible := make([]string, 0, len(nodes))
		ancestors := make([]string, 0, len(activeAncestors))
		for id, node := range nodes {
			startIndex := barIndexForEvent(candles, events[node.StartEventID])
			endIndex := barIndexForEvent(candles, events[node.EndEventID])
			if startIndex < 0 || endIndex-startIndex < 3 {
				continue
			}
			if _, active := activeAncestors[id]; active {
				ancestors = append(ancestors, id)
				continue
			}
			if degreeVisibleOnView(node.Degree, timeframe) {
				visible = append(visible, id)
			}
		}
		sort.Strings(visible)
		sort.Strings(ancestors)
		coverage := ViewCoverage{
			From: item.From, To: item.To, DetailFrom: input.Manifest.MinuteDetailFrom,
			Status: CoverageObserved,
		}
		if timeframe != market.Timeframe1D && timeframe != market.Timeframe1W &&
			item.From > 0 && input.Manifest.MinuteDetailFrom > item.From {
			coverage.Status = CoverageNotObservable
			coverage.Message = "Older intraday subdivisions are not loaded. Refine a parent wave to add detail."
		}
		views[timeframe] = TimeframeView{
			Timeframe: timeframe, Candles: append([]market.DerivedCandle(nil), candles...),
			VisibleNodeIDs: visible, AncestorNodeIDs: ancestors,
			FutureLogicalBars: append([]int64(nil), input.FutureBars[timeframe]...),
			Coverage:          coverage,
		}
	}
	return manifest, views
}

func barIndexForEvent(candles []market.DerivedCandle, event CanonicalWaveEvent) int {
	if event.ID == "" {
		return -1
	}
	index := sort.Search(len(candles), func(index int) bool {
		return candles[index].SourceTo >= event.OrthodoxTime
	})
	if index < len(candles) && candles[index].SourceFrom <= event.OrthodoxTime && candles[index].SourceTo >= event.OrthodoxTime {
		return index
	}
	if index < len(candles) && candles[index].Time == event.OrthodoxTime {
		return index
	}
	return -1
}

func degreeVisibleOnView(degree wave.Degree, timeframe market.Timeframe) bool {
	rank := degreeRank(degree)
	switch timeframe {
	case market.Timeframe1W:
		return rank >= degreeRank(wave.DegreePrimary)
	case market.Timeframe1D:
		return rank >= degreeRank(wave.DegreeIntermediate)
	case market.Timeframe4h, market.Timeframe1h:
		return rank >= degreeRank(wave.DegreeMinute)
	default:
		return true
	}
}

func degreeRank(degree wave.Degree) int {
	order := []wave.Degree{
		wave.DegreeObservableLeaf, wave.DegreeSubminuette, wave.DegreeMinuette,
		wave.DegreeMinute, wave.DegreeMinor, wave.DegreeIntermediate,
		wave.DegreePrimary, wave.DegreeCycle, wave.DegreeSupercycle,
		wave.DegreeGrandSupercycle,
	}
	for index, item := range order {
		if item == degree {
			return index
		}
	}
	return 0
}

func richerNode(candidate, current wave.WaveNode) bool {
	if len(candidate.Children) != len(current.Children) {
		return len(candidate.Children) > len(current.Children)
	}
	if candidate.Conformance.StructuralCoverage != current.Conformance.StructuralCoverage {
		return candidate.Conformance.StructuralCoverage > current.Conformance.StructuralCoverage
	}
	return candidate.Conformance.Score > current.Conformance.Score
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func appendUniqueTimeframe(values []market.Timeframe, value market.Timeframe) []market.Timeframe {
	if containsTimeframe(values, value) {
		return values
	}
	return append(values, value)
}

func containsTimeframe(values []market.Timeframe, value market.Timeframe) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, current := range values {
		if current == value {
			return true
		}
	}
	return false
}

func hasEventSource(values []EventSource, value EventSource) bool {
	for _, current := range values {
		if current.Timeframe == value.Timeframe && current.BarTime == value.BarTime &&
			current.Price == value.Price && current.Provenance == value.Provenance {
			return true
		}
	}
	return false
}

func timeframeRank(value market.Timeframe) int {
	for index, timeframe := range timeframeOrder {
		if timeframe == value {
			return index
		}
	}
	return len(timeframeOrder)
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
