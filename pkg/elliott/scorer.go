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

// ScenarioBundle scant de pivots, past fractalvalidatie toe, rangschikt ze via DP
// en bouwt de ultieme chronologische ketting zonder wiskundige weglatingen.
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

	pair = &model.ScenarioPair{Primary: primaryScenario, Alternate: alternateScenario}
	return motives, correctives, incompletes, pair
}

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

func correctiveToStructure(cw model.CorrectiveWave) model.WaveStructure {
	typeName := "CORRECTIVE_" + cw.Type

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

	const correctiveBaseScore = 0.68
	return model.WaveStructure{
		Type:            typeName,
		Pivots:          pivots,
		PurpleBoxes:     append([]model.TargetBox(nil), cw.PurpleBoxes...),
		ConfidenceScore: correctiveBaseScore,
		Degree:          cw.Degree,
	}
}

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

	isMotive := func(t string) bool { return strings.HasPrefix(t, "MOTIVE_") }
	isCorrective := func(t string) bool { return strings.HasPrefix(t, "CORRECTIVE_") }

	for i := n - 1; i >= 0; i-- {
		skipScore := 0.0
		if i+1 < n {
			skipScore = dp[i+1]
		}

		bestTakeScore := candidates[i].ConfidenceScore
		bestNextIdx := -1
		iEnd := candidates[i].Pivots[len(candidates[i].Pivots)-1].Time

		for j := i + 1; j < n; j++ {
			if candidates[j].Pivots[0].Time >= iEnd {
				// --- STRUKTURELE OPEENVOLGINGSVALIDATIE (Frost & Prechter) ---
				typeI := candidates[i].Type
				typeJ := candidates[j].Type

				isValidSequence := false
				if isMotive(typeI) && (isCorrective(typeJ) || typeJ == "INCOMPLETE_123") {
					isValidSequence = true
				} else if isCorrective(typeI) && (isMotive(typeJ) || typeJ == "INCOMPLETE_123") {
					isValidSequence = true
				} else if typeI == "INCOMPLETE_123" && isCorrective(typeJ) {
					isValidSequence = true
				}

				if !isValidSequence {
					continue
				}

				currentPathScore := candidates[i].ConfidenceScore + dp[j]
				if currentPathScore > bestTakeScore {
					bestTakeScore = currentPathScore
					bestNextIdx = j
				}
			}
		}

		if bestTakeScore >= skipScore {
			dp[i] = bestTakeScore
			nextOpt[i] = bestNextIdx
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

	totalScore := 0.0
	for _, ws := range chain {
		if strings.HasPrefix(ws.Type, "MOTIVE_") {
			totalScore += ws.ConfidenceScore * 1.25
		} else {
			totalScore += ws.ConfidenceScore
		}
	}

	avg := totalScore / float64(len(chain))
	if avg > 1.0 {
		return 1.0
	}
	return avg
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

	timeTolerance := int64(float64(macroDuration) * 0.25)
	macroPriceStart := macroStartPivot.Price
	macroPriceEnd := macroW1Pivot.Price
	macroPriceSpan := math.Abs(macroPriceEnd - macroPriceStart)

	for i := range childMotives {
		child := &childMotives[i]
		if child.Start == nil || child.W5 == nil {
			continue
		}
		if child.Direction != macroDirection {
			continue
		}
		if math.Abs(float64(child.Start.Time-macroStart)) > float64(timeTolerance) {
			continue
		}
		if math.Abs(float64(child.W5.Time-macroEnd)) > float64(timeTolerance) {
			continue
		}
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
	return "MINOR"
}

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
