package elliott

import (
	"math"
	"sort"
	"strings"
	"sync"

	"WaveSight/pkg/model"
)

const (
	fractalValidationBoost   = 0.15
	fractalValidationPenalty = 0.10
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
func ScenarioBundle(pivots []model.Pivot, childPivots []model.Pivot, parentTimeframe string) (motives []model.MotiveWave, correctives []model.CorrectiveWave, incompletes []model.IncompleteWave, pair *model.ScenarioPair) {
	var parentMotives []model.MotiveWave
	var parentCorrectives []model.CorrectiveWave
	var parentIncompletes []model.IncompleteWave
	var childMotives []model.MotiveWave

	parentDegree := GetDegreeForTimeframe(parentTimeframe)
	childDegree := GetChildDegree(parentDegree)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		parentMotives = MatchMotiveWaves(pivots)
		for idx := range parentMotives {
			parentMotives[idx].Degree = parentDegree
		}
	}()

	go func() {
		defer wg.Done()
		parentCorrectives = MatchCorrectiveWaves(pivots)
		for idx := range parentCorrectives {
			parentCorrectives[idx].Degree = parentDegree
		}
	}()

	go func() {
		defer wg.Done()
		parentIncompletes = MatchIncompleteWaves(pivots)
		for idx := range parentIncompletes {
			parentIncompletes[idx].Degree = parentDegree
		}
	}()

	if len(childPivots) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			childMotives = MatchMotiveWaves(childPivots)
			for idx := range childMotives {
				childMotives[idx].Degree = childDegree
			}
		}()
	}

	wg.Wait()

	// Perform fractal validation if child pivots are available
	subWavesMap := make(map[int64][]model.WaveStructure)
	var subWavesMu sync.Mutex

	if len(childPivots) > 0 {
		validatedMotives := make([]model.MotiveWave, 0, len(parentMotives))
		for _, mw := range parentMotives {
			if child, ok := findValidSubwave(mw.Start, mw.W1, mw.Direction, childMotives); ok {
				mw.ConfidenceScore = math.Min(1.0, mw.ConfidenceScore+fractalValidationBoost)
				subWavesMu.Lock()
				subWavesMap[mw.Start.Time] = append(subWavesMap[mw.Start.Time], motiveToStructure(child))
				subWavesMu.Unlock()
			} else {
				mw.ConfidenceScore -= fractalValidationPenalty
			}
			if mw.ConfidenceScore >= minConfidenceScore {
				validatedMotives = append(validatedMotives, mw)
			}
		}
		parentMotives = validatedMotives

		validatedIncompletes := make([]model.IncompleteWave, 0, len(parentIncompletes))
		for _, iw := range parentIncompletes {
			if child, ok := findValidSubwave(iw.Start, iw.W1, iw.Direction, childMotives); ok {
				iw.ConfidenceScore = math.Min(1.0, iw.ConfidenceScore+fractalValidationBoost)
				subWavesMu.Lock()
				subWavesMap[iw.Start.Time] = append(subWavesMap[iw.Start.Time], motiveToStructure(child))
				subWavesMu.Unlock()
			} else {
				iw.ConfidenceScore -= fractalValidationPenalty
			}
			if iw.ConfidenceScore >= minConfidenceScore {
				validatedIncompletes = append(validatedIncompletes, iw)
			}
		}
		parentIncompletes = validatedIncompletes
	}

	motives = parentMotives
	correctives = parentCorrectives
	incompletes = parentIncompletes

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

	// --- 3. Split candidates into Bullish and Bearish pools ---
	bullishCandidates := make([]model.WaveStructure, 0, len(all))
	bearishCandidates := make([]model.WaveStructure, 0, len(all))

	for _, ws := range all {
		if len(ws.Pivots) < 2 {
			continue
		}
		if isBullishScenarioComponent(ws) {
			bullishCandidates = append(bullishCandidates, ws)
		} else if isBearishScenarioComponent(ws) {
			bearishCandidates = append(bearishCandidates, ws)
		}
	}

	// --- 4. Chain non-overlapping waves for each pool ---
	bullishChain := findBestChain(bullishCandidates)
	bearishChain := findBestChain(bearishCandidates)

	bullishConf := getChainConfidence(bullishChain)
	bearishConf := getChainConfidence(bearishChain)

	var primaryScenario model.AnalysisScenario
	var alternateScenario model.AnalysisScenario

	if bullishConf >= bearishConf {
		primaryScenario = model.AnalysisScenario{
			Bias:       "BULLISH",
			Confidence: bullishConf,
			Structures: assembleScenarioStructures(bullishChain, subWavesMap),
		}
		alternateScenario = model.AnalysisScenario{
			Bias:       "BEARISH",
			Confidence: bearishConf,
			Structures: assembleScenarioStructures(bearishChain, subWavesMap),
		}
	} else {
		primaryScenario = model.AnalysisScenario{
			Bias:       "BEARISH",
			Confidence: bearishConf,
			Structures: assembleScenarioStructures(bearishChain, subWavesMap),
		}
		alternateScenario = model.AnalysisScenario{
			Bias:       "BULLISH",
			Confidence: bullishConf,
			Structures: assembleScenarioStructures(bullishChain, subWavesMap),
		}
	}

	pair = &model.ScenarioPair{
		Primary:   primaryScenario,
		Alternate: alternateScenario,
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

	return model.WaveStructure{
		Type:            typeName,
		Pivots:          pivots,
		PurpleBoxes:     append([]model.TargetBox(nil), mw.PurpleBoxes...),
		ConfidenceScore: mw.ConfidenceScore,
		Degree:          mw.Degree,
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
		PurpleBoxes:     append([]model.TargetBox(nil), cw.PurpleBoxes...),
		ConfidenceScore: correctiveBaseScore,
		Degree:          cw.Degree,
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
		Degree:          iw.Degree,
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

// findValidSubwave checks if there is a child motive wave that aligns with the time and price span of a macro Wave 1.
func findValidSubwave(macroStartPivot, macroW1Pivot *model.Pivot, macroDirection string, childMotives []model.MotiveWave) (model.MotiveWave, bool) {
	if macroStartPivot == nil || macroW1Pivot == nil {
		return model.MotiveWave{}, false
	}

	macroStart := macroStartPivot.Time
	macroEnd := macroW1Pivot.Time
	macroDuration := macroEnd - macroStart
	if macroDuration <= 0 {
		return model.MotiveWave{}, false
	}

	// 25% duration tolerance
	timeTolerance := int64(float64(macroDuration) * 0.25)

	macroPriceStart := macroStartPivot.Price
	macroPriceEnd := macroW1Pivot.Price
	macroPriceSpan := math.Abs(macroPriceEnd - macroPriceStart)

	for i := range childMotives {
		child := &childMotives[i]
		if child.Start == nil || child.W5 == nil {
			continue
		}

		// 1. Direction must match
		if child.Direction != macroDirection {
			continue
		}

		// 2. Start time must be within tolerance
		if math.Abs(float64(child.Start.Time-macroStart)) > float64(timeTolerance) {
			continue
		}

		// 3. End time must be within tolerance
		if math.Abs(float64(child.W5.Time-macroEnd)) > float64(timeTolerance) {
			continue
		}

		// 4. Price must align within tolerance (e.g. child start/end price within 20% of macro wave price span)
		if macroPriceSpan > 0 {
			priceStartDiff := math.Abs(child.Start.Price - macroPriceStart)
			priceEndDiff := math.Abs(child.W5.Price - macroPriceEnd)
			if priceStartDiff <= macroPriceSpan*0.20 && priceEndDiff <= macroPriceSpan*0.20 {
				return *child, true
			}
		}
	}
	return model.MotiveWave{}, false
}

func isBullishScenarioComponent(ws model.WaveStructure) bool {
	if strings.HasPrefix(ws.Type, "MOTIVE_") {
		return ws.Pivots[0].Type == model.PivotLow
	}
	if ws.Type == "INCOMPLETE_123" {
		return ws.Pivots[0].Type == model.PivotLow
	}
	if strings.HasPrefix(ws.Type, "CORRECTIVE_") {
		return ws.Pivots[0].Type == model.PivotHigh
	}
	return false
}

func isBearishScenarioComponent(ws model.WaveStructure) bool {
	if strings.HasPrefix(ws.Type, "MOTIVE_") {
		return ws.Pivots[0].Type == model.PivotHigh
	}
	if ws.Type == "INCOMPLETE_123" {
		return ws.Pivots[0].Type == model.PivotHigh
	}
	if strings.HasPrefix(ws.Type, "CORRECTIVE_") {
		return ws.Pivots[0].Type == model.PivotLow
	}
	return false
}

func findBestChain(candidates []model.WaveStructure) []model.WaveStructure {
	n := len(candidates)
	if n == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		t1 := candidates[i].Pivots[0].Time
		t2 := candidates[j].Pivots[0].Time
		if t1 == t2 {
			return candidates[i].Pivots[len(candidates[i].Pivots)-1].Time < candidates[j].Pivots[len(candidates[j].Pivots)-1].Time
		}
		return t1 < t2
	})

	dp := make([]float64, n)
	nextOpt := make([]int, n)
	for i := 0; i < n; i++ {
		nextOpt[i] = -1
	}

	for i := n - 1; i >= 0; i-- {
		skipScore := 0.0
		if i+1 < n {
			skipScore = dp[i+1]
		}

		takeScore := candidates[i].ConfidenceScore
		nextIdx := -1
		iEnd := candidates[i].Pivots[len(candidates[i].Pivots)-1].Time
		for j := i + 1; j < n; j++ {
			if candidates[j].Pivots[0].Time >= iEnd {
				nextIdx = j
				break
			}
		}

		if nextIdx != -1 {
			takeScore += dp[nextIdx]
		}

		if takeScore >= skipScore {
			dp[i] = takeScore
			nextOpt[i] = nextIdx
		} else {
			dp[i] = skipScore
			nextOpt[i] = -2
		}
	}

	var bestChain []model.WaveStructure
	curr := 0
	for curr < n && curr != -1 {
		if nextOpt[curr] == -2 {
			curr++
		} else {
			bestChain = append(bestChain, candidates[curr])
			curr = nextOpt[curr]
		}
	}

	return bestChain
}

func getChainConfidence(chain []model.WaveStructure) float64 {
	if len(chain) == 0 {
		return 0.0
	}
	maxConf := 0.0
	for _, ws := range chain {
		if ws.ConfidenceScore > maxConf {
			maxConf = ws.ConfidenceScore
		}
	}
	return maxConf
}

func assembleScenarioStructures(chain []model.WaveStructure, subWavesMap map[int64][]model.WaveStructure) []model.WaveStructure {
	var assembled []model.WaveStructure
	for _, parent := range chain {
		assembled = append(assembled, parent)
		if sub, ok := subWavesMap[parent.Pivots[0].Time]; ok {
			assembled = append(assembled, sub...)
		}
	}
	return assembled
}

// GetDegreeForTimeframe maps a timeframe string to a standard Elliott Wave degree.
func GetDegreeForTimeframe(tf string) string {
	p := strings.ToUpper(tf)
	if p == "1D" || p == "D" || strings.HasSuffix(p, "DAY") || strings.HasSuffix(p, "DAYS") {
		return "MINOR"
	}
	if p == "1H" || p == "H" || strings.HasSuffix(p, "HOUR") || strings.HasSuffix(p, "HOURS") {
		return "MINUTE"
	}
	if p == "15M" || p == "15MIN" || p == "15MINUTES" || p == "15MINUTE" {
		return "MINUETTE"
	}
	return "MINOR" // default fallback
}

// GetChildDegree maps a parent degree to the next lower degree in the hierarchy.
func GetChildDegree(parentDegree string) string {
	switch parentDegree {
	case "GRAND_SUPERCYCLE":
		return "SUPERCYCLE"
	case "SUPERCYCLE":
		return "CYCLE"
	case "CYCLE":
		return "PRIMARY"
	case "PRIMARY":
		return "INTERMEDIATE"
	case "INTERMEDIATE":
		return "MINOR"
	case "MINOR":
		return "MINUTE"
	case "MINUTE":
		return "MINUETTE"
	case "MINUETTE":
		return "SUBMINUETTE"
	default:
		return "MINUTE"
	}
}
