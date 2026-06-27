package wave

import "testing"

func TestRuleRegistryIsUniqueAndComplete(t *testing.T) {
	t.Parallel()

	required := []string{
		RuleMotiveWave2Limit,
		RuleMotiveWave3BeyondWave1,
		RuleMotiveWave3NotShortest,
		RuleImpulseNoOverlap,
		RuleImpulseSubdivision,
		RuleLeadingDiagonalPosition,
		RuleLeadingDiagonalShape,
		RuleEndingDiagonalPosition,
		RuleEndingDiagonalShape,
		RuleDiagonalConvergence,
		RuleTruncationFiveWaves,
		RuleCorrectionNeverFive,
		RuleZigzagSubdivision,
		RuleFlatSubdivision,
		RuleTriangleSubdivision,
		RuleTrianglePosition,
		RuleCombinationTriangleLast,
		RuleCombinationOneTriangle,
		RuleCombinationOneZigzag,
		RuleOrthodoxEndpoints,
		GuideAlternation,
		GuideEquality,
		GuideChannel,
		GuidePreviousFourth,
		GuideFifthExtensionRetrace,
		GuideRightLook,
		GuideVolume,
		GuidePersonality,
		GuideSemilog,
		GuideFibonacciTime,
		PriorWave2Ratios,
		PriorWave3Ratios,
		PriorWave4Ratios,
		PriorWave5Ratios,
	}

	seen := make(map[string]struct{})
	for _, definition := range RuleRegistry() {
		if definition.ID == "" || definition.Source == "" || definition.Summary == "" || definition.Class == "" {
			t.Fatalf("incomplete rule definition: %+v", definition)
		}
		if _, exists := seen[definition.ID]; exists {
			t.Fatalf("duplicate rule ID %q", definition.ID)
		}
		seen[definition.ID] = struct{}{}
	}
	for _, id := range required {
		if _, exists := seen[id]; !exists {
			t.Errorf("required rule %q is absent from registry", id)
		}
	}
}

func TestEveryRegisteredRuleCarriesPositiveAndNegativeAuditMetadata(t *testing.T) {
	t.Parallel()
	for _, definition := range RuleRegistry() {
		definition := definition
		t.Run(definition.ID, func(t *testing.T) {
			t.Parallel()
			for _, status := range []EvaluationStatus{EvaluationPass, EvaluationFail} {
				result := evaluation(definition.ID, status, 1, "fixture expectation")
				if result.RuleID != definition.ID || result.Class != definition.Class ||
					result.Source != definition.Source || result.Summary != definition.Summary ||
					result.Status != status {
					t.Fatalf("audit evaluation = %+v, definition = %+v", result, definition)
				}
			}
		})
	}
}
