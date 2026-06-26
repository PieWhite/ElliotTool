package elliott

import (
	"sort"

	"WaveSight/pkg/model"
)

// scoredStructure pairs a WaveStructure with a sortable key for ranking.
type scoredStructure struct {
	structure model.WaveStructure
}

// ScenarioBundle runs all three Elliott Wave scanners over the supplied pivots,
// converts every detected pattern into a generic WaveStructure, ranks them by
// ConfidenceScore (descending), and returns an AnalysisResponse with:
//   - Scenarios.Primary   — the highest-confidence structure.
//   - Scenarios.Alternate — the second-highest OR an inverse-directional placeholder
//     with Confidence 0.0 when only a single structure is found.
//   - Legacy flat arrays (MotiveWaves, CorrectiveWaves, IncompleteWaves) for backward compat.
func ScenarioBundle(pivots []model.Pivot) (motives []model.MotiveWave, correctives []model.CorrectiveWave, incompletes []model.IncompleteWave, pair *model.ScenarioPair) {
	// --- 1. Run existing scanners (no re-implementation) ---
	motives = MatchMotiveWaves(pivots)
	correctives = MatchCorrectiveWaves(pivots)
	incompletes = MatchIncompleteWaves(pivots)

	// --- 2. Convert all found structures into the generic WaveStructure type ---
	var all []model.WaveStructure

	for _, mw := range motives {
		all = append(all, motiveToStructure(mw))
	}
	for _, cw := range correctives {
		all = append(all, correctiveToStructure(cw))
	}
	for _, iw := range incompletes {
		all = append(all, incompleteToStructure(iw))
	}

	// --- 3. Sort by confidence descending (stable to keep scan order on ties) ---
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].ConfidenceScore > all[j].ConfidenceScore
	})

	// --- 4. Build primary / alternate scenarios ---
	switch len(all) {
	case 0:
		// No patterns found — return empty scenarios with opposite placeholders.
		pair = &model.ScenarioPair{
			Primary:   model.AnalysisScenario{Bias: "BULLISH", Confidence: 0.0, Structures: []model.WaveStructure{}},
			Alternate: model.AnalysisScenario{Bias: "BEARISH", Confidence: 0.0, Structures: []model.WaveStructure{}},
		}

	case 1:
		// Single structure — primary is the found pattern, alternate is the inverse directional placeholder.
		primaryBias := biasByType(all[0])
		pair = &model.ScenarioPair{
			Primary: model.AnalysisScenario{
				Bias:       primaryBias,
				Confidence: all[0].ConfidenceScore,
				Structures: []model.WaveStructure{all[0]},
			},
			Alternate: model.AnalysisScenario{
				Bias:       inverseBias(primaryBias),
				Confidence: 0.0,
				Structures: []model.WaveStructure{},
			},
		}

	default:
		// Two or more structures — primary is the top scorer, alternate is the runner-up.
		primaryBias := biasByType(all[0])
		alternateBias := biasByType(all[1])
		pair = &model.ScenarioPair{
			Primary: model.AnalysisScenario{
				Bias:       primaryBias,
				Confidence: all[0].ConfidenceScore,
				Structures: []model.WaveStructure{all[0]},
			},
			Alternate: model.AnalysisScenario{
				Bias:       alternateBias,
				Confidence: all[1].ConfidenceScore,
				Structures: []model.WaveStructure{all[1]},
			},
		}
	}

	return motives, correctives, incompletes, pair
}

// motiveToStructure converts a MotiveWave into a generic WaveStructure for scenario bundling.
// Pivot ordering: Start, W1, W2, W3, W4, W5.
func motiveToStructure(mw model.MotiveWave) model.WaveStructure {
	typeName := "MOTIVE_IMPULSE"
	if mw.IsDiagonal {
		typeName = "MOTIVE_DIAGONAL"
	}
	if mw.IsTruncated {
		typeName = "MOTIVE_TRUNCATED"
	}

	pivots := make([]model.Pivot, 0, 6)
	if mw.Start != nil {
		pivots = append(pivots, *mw.Start)
	}
	if mw.W1 != nil {
		pivots = append(pivots, *mw.W1)
	}
	if mw.W2 != nil {
		pivots = append(pivots, *mw.W2)
	}
	if mw.W3 != nil {
		pivots = append(pivots, *mw.W3)
	}
	if mw.W4 != nil {
		pivots = append(pivots, *mw.W4)
	}
	if mw.W5 != nil {
		pivots = append(pivots, *mw.W5)
	}

	var boxes []model.TargetBox
	if mw.PurpleBox != nil {
		boxes = []model.TargetBox{*mw.PurpleBox}
	}

	return model.WaveStructure{
		Type:            typeName,
		Pivots:          pivots,
		PurpleBoxes:     boxes,
		ConfidenceScore: mw.ConfidenceScore,
	}
}

// correctiveToStructure converts a CorrectiveWave into a generic WaveStructure.
// Pivot ordering: Start, WA, WB, WC, [WX], [WD], [WE].
func correctiveToStructure(cw model.CorrectiveWave) model.WaveStructure {
	typeName := "CORRECTIVE_" + cw.Type // e.g. CORRECTIVE_ZIGZAG

	pivots := make([]model.Pivot, 0, 7)
	if cw.Start != nil {
		pivots = append(pivots, *cw.Start)
	}
	if cw.WA != nil {
		pivots = append(pivots, *cw.WA)
	}
	if cw.WB != nil {
		pivots = append(pivots, *cw.WB)
	}
	if cw.WC != nil {
		pivots = append(pivots, *cw.WC)
	}
	if cw.WX != nil {
		pivots = append(pivots, *cw.WX)
	}
	if cw.WD != nil {
		pivots = append(pivots, *cw.WD)
	}
	if cw.WE != nil {
		pivots = append(pivots, *cw.WE)
	}

	// Corrective waves don't carry a confidence score in the current model.
	// Assign a base score that reflects structural validity (0.70 for proven patterns).
	const correctiveBaseScore = 0.70

	return model.WaveStructure{
		Type:            typeName,
		Pivots:          pivots,
		PurpleBoxes:     nil,
		ConfidenceScore: correctiveBaseScore,
	}
}

// incompleteToStructure converts an IncompleteWave (1-2-3) into a generic WaveStructure.
// Pivot ordering: Start, W1, W2, W3.
func incompleteToStructure(iw model.IncompleteWave) model.WaveStructure {
	pivots := make([]model.Pivot, 0, 4)
	if iw.Start != nil {
		pivots = append(pivots, *iw.Start)
	}
	if iw.W1 != nil {
		pivots = append(pivots, *iw.W1)
	}
	if iw.W2 != nil {
		pivots = append(pivots, *iw.W2)
	}
	if iw.W3 != nil {
		pivots = append(pivots, *iw.W3)
	}

	var boxes []model.TargetBox
	if iw.TargetBox != nil {
		boxes = []model.TargetBox{*iw.TargetBox}
	}

	return model.WaveStructure{
		Type:            "INCOMPLETE_123",
		Pivots:          pivots,
		PurpleBoxes:     boxes,
		ConfidenceScore: iw.ConfidenceScore,
	}
}

// biasByType reads the direction embedded in the WaveStructure's first pivot sequence.
// For motive & incomplete structures the first pivot type determines direction:
// PivotLow start = BULLISH, PivotHigh start = BEARISH.
// For corrective structures the Type prefix is used (CORRECTIVE_ + direction is not
// embedded in the type name, so we fall back to pivot analysis).
func biasByType(ws model.WaveStructure) string {
	if len(ws.Pivots) == 0 {
		return "BULLISH"
	}
	if ws.Pivots[0].Type == model.PivotLow {
		return "BULLISH"
	}
	return "BEARISH"
}

// inverseBias returns the opposite of the supplied directional bias string.
func inverseBias(bias string) string {
	if bias == "BULLISH" {
		return "BEARISH"
	}
	return "BULLISH"
}
