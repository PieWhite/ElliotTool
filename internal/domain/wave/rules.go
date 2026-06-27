package wave

// RuleDefinition is the canonical executable bibliography entry.
type RuleDefinition struct {
	ID        string
	Class     RuleClass
	Source    string
	Summary   string
	AppliesTo []PatternType
}

const (
	RuleMotiveWave2Limit        = "EWP-MOTIVE-W2-100"
	RuleMotiveWave3BeyondWave1  = "EWP-MOTIVE-W3-BEYOND-W1"
	RuleMotiveWave3NotShortest  = "EWP-MOTIVE-W3-NOT-SHORTEST"
	RuleImpulseNoOverlap        = "EWP-IMPULSE-W4-NO-OVERLAP"
	RuleImpulseSubdivision      = "EWP-IMPULSE-53535"
	RuleLeadingDiagonalPosition = "EWP-LEADING-DIAGONAL-POSITION"
	RuleLeadingDiagonalShape    = "EWP-LEADING-DIAGONAL-53535"
	RuleEndingDiagonalPosition  = "EWP-ENDING-DIAGONAL-POSITION"
	RuleEndingDiagonalShape     = "EWP-ENDING-DIAGONAL-33333"
	RuleDiagonalConvergence     = "EWP-DIAGONAL-CONVERGENCE"
	RuleTruncationFiveWaves     = "EWP-TRUNCATION-FIVE-SUBWAVES"
	RuleCorrectionNeverFive     = "EWP-CORRECTION-NEVER-FIVE"
	RuleZigzagSubdivision       = "EWP-ZIGZAG-535"
	RuleFlatSubdivision         = "EWP-FLAT-335"
	RuleTriangleSubdivision     = "EWP-TRIANGLE-33333"
	RuleTrianglePosition        = "EWP-TRIANGLE-POSITION"
	RuleCombinationTriangleLast = "EWP-COMBINATION-TRIANGLE-LAST"
	RuleCombinationOneTriangle  = "EWP-COMBINATION-ONE-TRIANGLE"
	RuleCombinationOneZigzag    = "EWP-COMBINATION-ONE-ZIGZAG"
	RuleOrthodoxEndpoints       = "EWP-ORTHODOX-ENDPOINTS"
	GuideAlternation            = "EWP-GUIDE-ALTERNATION"
	GuideEquality               = "EWP-GUIDE-EQUALITY"
	GuideChannel                = "EWP-GUIDE-CHANNEL"
	GuidePreviousFourth         = "EWP-GUIDE-PREVIOUS-FOURTH"
	GuideFifthExtensionRetrace  = "EWP-GUIDE-FIFTH-EXTENSION-RETRACE"
	GuideRightLook              = "EWP-GUIDE-RIGHT-LOOK"
	GuideVolume                 = "EWP-GUIDE-VOLUME"
	GuidePersonality            = "EWP-GUIDE-PERSONALITY"
	GuideSemilog                = "EWP-GUIDE-SEMILOG"
	GuideFibonacciTime          = "EWP-GUIDE-FIBONACCI-TIME"
	PriorWave2Ratios            = "WR-PRIOR-W2"
	PriorWave3Ratios            = "WR-PRIOR-W3"
	PriorWave4Ratios            = "WR-PRIOR-W4"
	PriorWave5Ratios            = "WR-PRIOR-W5"
)

var ruleRegistry = []RuleDefinition{
	{RuleMotiveWave2Limit, RuleHard, "EWP p.13", "Wave 2 retraces no more than 100% of wave 1.", []PatternType{PatternImpulse, PatternLeadingDiagonal, PatternEndingDiagonal, PatternTruncatedImpulse}},
	{RuleMotiveWave3BeyondWave1, RuleHard, "EWP p.13", "Wave 3 travels beyond the end of wave 1.", []PatternType{PatternImpulse, PatternLeadingDiagonal, PatternEndingDiagonal, PatternTruncatedImpulse}},
	{RuleMotiveWave3NotShortest, RuleHard, "EWP p.13", "Wave 3 is never the shortest actionary wave.", []PatternType{PatternImpulse, PatternLeadingDiagonal, PatternEndingDiagonal, PatternTruncatedImpulse}},
	{RuleImpulseNoOverlap, RuleHard, "EWP p.13", "Wave 4 does not enter wave 1 price territory in a cash-market impulse.", []PatternType{PatternImpulse, PatternTruncatedImpulse}},
	{RuleImpulseSubdivision, RuleHard, "EWP pp.9,13", "Impulse subdivision is 5-3-5-3-5.", []PatternType{PatternImpulse, PatternTruncatedImpulse}},
	{RuleLeadingDiagonalPosition, RuleHard, "EWP p.19", "Leading diagonal occurs only in wave 1 or A.", []PatternType{PatternLeadingDiagonal}},
	{RuleLeadingDiagonalShape, RuleHard, "EWP p.19", "Leading diagonal subdivision is 5-3-5-3-5.", []PatternType{PatternLeadingDiagonal}},
	{RuleEndingDiagonalPosition, RuleHard, "EWP p.17", "Ending diagonal occurs only in wave 5 or C.", []PatternType{PatternEndingDiagonal}},
	{RuleEndingDiagonalShape, RuleHard, "EWP p.17", "Ending diagonal subdivision is 3-3-3-3-3.", []PatternType{PatternEndingDiagonal}},
	{RuleDiagonalConvergence, RuleHard, "EWP pp.17-19", "Accepted diagonals form a converging wedge.", []PatternType{PatternLeadingDiagonal, PatternEndingDiagonal}},
	{RuleTruncationFiveWaves, RuleHard, "EWP p.15", "A truncated fifth must contain five subwaves.", []PatternType{PatternTruncatedImpulse}},
	{RuleCorrectionNeverFive, RuleHard, "EWP p.20", "A completed correction is never a five.", []PatternType{PatternZigzag, PatternFlatRegular, PatternFlatExpanded, PatternFlatRunning, PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding}},
	{RuleZigzagSubdivision, RuleHard, "EWP pp.21-24", "Zigzag subdivision is 5-3-5.", []PatternType{PatternZigzag, PatternDoubleZigzag, PatternTripleZigzag}},
	{RuleFlatSubdivision, RuleHard, "EWP pp.24-27", "Flat subdivision is 3-3-5.", []PatternType{PatternFlatRegular, PatternFlatExpanded, PatternFlatRunning}},
	{RuleTriangleSubdivision, RuleHard, "EWP pp.27-29", "Triangle subdivision is 3-3-3-3-3.", []PatternType{PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding}},
	{RuleTrianglePosition, RuleHard, "EWP p.29", "Triangles occur before the final actionary wave or terminally in a combination.", []PatternType{PatternTriangleContracting, PatternTriangleAscending, PatternTriangleDescending, PatternTriangleRunning, PatternTriangleExpanding}},
	{RuleCombinationTriangleLast, RuleHard, "EWP p.30", "A triangle in a combination is terminal.", []PatternType{PatternDoubleThree, PatternTripleThree}},
	{RuleCombinationOneTriangle, RuleHard, "EWP p.30", "A combination contains no more than one triangle.", []PatternType{PatternDoubleThree, PatternTripleThree}},
	{RuleCombinationOneZigzag, RuleHard, "EWP p.30", "A sideways combination contains no more than one zigzag.", []PatternType{PatternDoubleThree, PatternTripleThree}},
	{RuleOrthodoxEndpoints, RuleHard, "EWP p.31", "Measurements use orthodox pattern endpoints.", nil},
	{GuideAlternation, RuleGuideline, "EWP pp.32-34", "Waves 2 and 4 tend to alternate in form, depth and complexity.", nil},
	{GuideEquality, RuleGuideline, "EWP pp.37-38", "The two non-extended motive waves tend toward equality or a 0.618 relation.", nil},
	{GuideChannel, RuleGuideline, "EWP pp.38-40; WaveRatios p.5", "Parallel channels estimate fourth and fifth wave boundaries.", nil},
	{GuidePreviousFourth, RuleGuideline, "EWP pp.35-36", "Corrections tend to terminate in the previous fourth-wave area.", nil},
	{GuideFifthExtensionRetrace, RuleGuideline, "EWP p.37", "A correction after a fifth-wave extension tends to reach wave 2 of the extension.", nil},
	{GuideRightLook, RuleGuideline, "EWP pp.42-43", "Overall form must have the right look.", nil},
	{GuideVolume, RuleGuideline, "EWP p.42", "Volume behavior helps distinguish third, fifth and corrective waves.", nil},
	{GuidePersonality, RuleGuideline, "EWP pp.43-46", "Wave personality supports position identification.", nil},
	{GuideSemilog, RuleGuideline, "EWP pp.41-42", "Large-degree acceleration should also be evaluated on semilog scale.", nil},
	{GuideFibonacciTime, RuleGuideline, "EWP pp.83-87", "Fibonacci time supports but does not independently determine a turn.", nil},
	{PriorWave2Ratios, RuleStatisticalPrior, "WaveRatios pp.2,6", "Wave 2 most often retraces around 50%-62% of wave 1.", nil},
	{PriorWave3Ratios, RuleStatisticalPrior, "WaveRatios pp.3,8", "Wave 3 commonly relates to wave 1 by 1.62, 2.62 or 4.25.", nil},
	{PriorWave4Ratios, RuleStatisticalPrior, "WaveRatios pp.3,10", "Wave 4 commonly retraces 24%, 38% or 50% of wave 3.", nil},
	{PriorWave5Ratios, RuleStatisticalPrior, "WaveRatios pp.4,12-13", "Wave 5 relates to wave 1 and the 0-to-3 distance.", nil},
}

func RuleRegistry() []RuleDefinition {
	result := make([]RuleDefinition, len(ruleRegistry))
	copy(result, ruleRegistry)
	return result
}

func RuleByID(id string) (RuleDefinition, bool) {
	for _, definition := range ruleRegistry {
		if definition.ID == id {
			return definition, true
		}
	}
	return RuleDefinition{}, false
}

func evaluation(id string, status EvaluationStatus, measured float64, expected string) RuleEvaluation {
	definition, ok := RuleByID(id)
	if !ok {
		return RuleEvaluation{RuleID: id, Status: status, Measured: measured, Expected: expected}
	}
	return RuleEvaluation{
		RuleID:   id,
		Class:    definition.Class,
		Status:   status,
		Source:   definition.Source,
		Summary:  definition.Summary,
		Measured: measured,
		Expected: expected,
	}
}
