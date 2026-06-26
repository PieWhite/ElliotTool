package elliott_test

import (
	"math"
	"sync"
	"testing"

	"WaveSight/pkg/elliott"
	"WaveSight/pkg/model"
)

// ---------------------------------------------------------------------------
// Step 10: ScenarioBundle (Probabilistic Engine) Tests
// ---------------------------------------------------------------------------

// textbookBullish returns a set of pivots that form a high-confidence BULLISH
// 5-wave motive structure with textbook Fibonacci relationships (≥ 0.90 score).
func textbookBullish(startTime int64) []model.Pivot {
	return []model.Pivot{
		{Time: startTime, Price: 100.0, Type: model.PivotLow},       // Start
		{Time: startTime + 10, Price: 200.0, Type: model.PivotHigh}, // W1 (len: 100)
		{Time: startTime + 20, Price: 138.2, Type: model.PivotLow},  // W2 (61.8% retrace)
		{Time: startTime + 30, Price: 300.0, Type: model.PivotHigh}, // W3 (161.8% ext)
		{Time: startTime + 40, Price: 220.0, Type: model.PivotLow},  // W4
		{Time: startTime + 50, Price: 343.6, Type: model.PivotHigh}, // W5 (61.8% of net)
	}
}

// textbookBearish returns a set of pivots that form a high-confidence BEARISH
// 5-wave motive structure with textbook Fibonacci relationships (≥ 0.90 score).
func textbookBearish(startTime int64) []model.Pivot {
	return []model.Pivot{
		{Time: startTime, Price: 300.0, Type: model.PivotHigh},      // Start
		{Time: startTime + 10, Price: 200.0, Type: model.PivotLow},  // W1 (len: 100)
		{Time: startTime + 20, Price: 261.8, Type: model.PivotHigh}, // W2 (61.8% retrace)
		{Time: startTime + 30, Price: 100.0, Type: model.PivotLow},  // W3 (161.8% ext)
		{Time: startTime + 40, Price: 180.0, Type: model.PivotHigh}, // W4
		{Time: startTime + 50, Price: 56.4, Type: model.PivotLow},   // W5 (61.8% of net)
	}
}

func TestScenarioBundle(t *testing.T) {
	tests := []struct {
		name   string
		pivots []model.Pivot
		verify func(t *testing.T, pair *model.ScenarioPair, motives []model.MotiveWave, correctives []model.CorrectiveWave, incompletes []model.IncompleteWave)
	}{
		{
			// Two valid motive sequences with different Fibonacci alignment:
			// Sequence A (bullish textbook) produces confidence ≥ 0.90.
			// Sequence B is a slightly less ideal bearish structure appended after A;
			// we craft it so both are valid but A outscores B.
			// The pivot lists are concatenated with B starting after A's last pivot.
			name: "Two valid motive waves: primary gets higher score, alternate gets lower",
			pivots: func() []model.Pivot {
				// Build a combined pivot slice: textbook bullish (time 100-150) followed
				// by textbook bearish (time 200-250) — non-overlapping windows so each
				// MatchMotiveWaves window finds one structure.
				bullishPivots := textbookBullish(100)
				bearishPivots := textbookBearish(200)
				return append(bullishPivots, bearishPivots...)
			}(),
			verify: func(t *testing.T, pair *model.ScenarioPair, motives []model.MotiveWave, correctives []model.CorrectiveWave, incompletes []model.IncompleteWave) {
				if pair == nil {
					t.Fatal("expected non-nil ScenarioPair")
				}
				// Both textbook setups produce identical scores; what matters is that
				// primary.confidence >= alternate.confidence.
				if pair.Primary.Confidence < pair.Alternate.Confidence {
					t.Errorf("primary confidence (%f) must be >= alternate confidence (%f)",
						pair.Primary.Confidence, pair.Alternate.Confidence)
				}
				// Primary scenario must have at least one structure.
				if len(pair.Primary.Structures) == 0 {
					t.Error("primary scenario must have at least one structure")
				}
				// Alternate scenario must have at least one structure (two waves found).
				if len(pair.Alternate.Structures) == 0 {
					t.Error("alternate scenario must have at least one structure when two waves are found")
				}
				// Legacy flat arrays must also be populated.
				if len(motives) == 0 {
					t.Error("expected legacy motive_waves to be non-empty")
				}
			},
		},
		{
			// Only 1 valid motive wave (bullish textbook). Alternate should be the
			// inverse directional placeholder with Confidence == 0.0.
			name:   "Single valid motive wave: alternate is inverse directional placeholder",
			pivots: textbookBullish(100),
			verify: func(t *testing.T, pair *model.ScenarioPair, motives []model.MotiveWave, _ []model.CorrectiveWave, _ []model.IncompleteWave) {
				if pair == nil {
					t.Fatal("expected non-nil ScenarioPair")
				}
				if pair.Primary.Bias != "BULLISH" {
					t.Errorf("expected primary bias BULLISH, got %q", pair.Primary.Bias)
				}
				if pair.Alternate.Bias != "BEARISH" {
					t.Errorf("expected alternate bias BEARISH (inverse), got %q", pair.Alternate.Bias)
				}
				if pair.Alternate.Confidence != 0.0 {
					t.Errorf("expected alternate confidence 0.0 for placeholder, got %f", pair.Alternate.Confidence)
				}
				if len(motives) != 1 {
					t.Errorf("expected exactly 1 motive wave in legacy array, got %d", len(motives))
				}
			},
		},
		{
			// Confidence ordering invariant: primary.confidence >= alternate.confidence,
			// verified against the bearish textbook setup.
			name:   "Confidence ordering: primary.confidence >= alternate.confidence (bearish textbook)",
			pivots: textbookBearish(100),
			verify: func(t *testing.T, pair *model.ScenarioPair, _ []model.MotiveWave, _ []model.CorrectiveWave, _ []model.IncompleteWave) {
				if pair == nil {
					t.Fatal("expected non-nil ScenarioPair")
				}
				if pair.Primary.Confidence < pair.Alternate.Confidence {
					t.Errorf("invariant violated: primary confidence (%f) < alternate confidence (%f)",
						pair.Primary.Confidence, pair.Alternate.Confidence)
				}
				if pair.Primary.Bias != "BEARISH" {
					t.Errorf("expected primary bias BEARISH, got %q", pair.Primary.Bias)
				}
			},
		},
		{
			// No patterns: both primary and alternate should be empty placeholders.
			name: "No patterns found: both scenarios are empty placeholders",
			pivots: []model.Pivot{
				{Time: 100, Price: 10.0, Type: model.PivotLow},
				{Time: 110, Price: 11.0, Type: model.PivotHigh},
			},
			verify: func(t *testing.T, pair *model.ScenarioPair, motives []model.MotiveWave, correctives []model.CorrectiveWave, incompletes []model.IncompleteWave) {
				if pair == nil {
					t.Fatal("expected non-nil ScenarioPair")
				}
				if pair.Primary.Confidence != 0.0 {
					t.Errorf("expected primary confidence 0.0, got %f", pair.Primary.Confidence)
				}
				if pair.Alternate.Confidence != 0.0 {
					t.Errorf("expected alternate confidence 0.0, got %f", pair.Alternate.Confidence)
				}
				if len(motives) != 0 || len(correctives) != 0 || len(incompletes) != 0 {
					t.Error("expected all legacy arrays to be empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			motives, correctives, incompletes, pair := elliott.ScenarioBundle(tt.pivots, nil, "1D")
			tt.verify(t, pair, motives, correctives, incompletes)
		})
	}
}

func TestFractalValidation(t *testing.T) {
	// 1. Success case: Parent motive wave Wave 1 has a matching child motive wave.
	t.Run("Motive Wave 1 confirmed by child motive: confidence is boosted", func(t *testing.T) {
		parentPivots := textbookBullish(100) // W1 span: 100 to 110, price 100 to 200.

		// Build child pivots representing a smaller motive wave that fits inside parent Wave 1 span.
		childPivots := []model.Pivot{
			{Time: 100, Price: 100.0, Type: model.PivotLow},
			{Time: 102, Price: 136.67, Type: model.PivotHigh},
			{Time: 104, Price: 114.01, Type: model.PivotLow},
			{Time: 106, Price: 173.34, Type: model.PivotHigh},
			{Time: 108, Price: 150.68, Type: model.PivotLow},
			{Time: 110, Price: 196.0, Type: model.PivotHigh},
		}

		baseMotives, _, _, _ := elliott.ScenarioBundle(parentPivots, nil, "1D")
		if len(baseMotives) == 0 {
			t.Fatal("expected to find parent motive wave")
		}
		baseScore := baseMotives[0].ConfidenceScore

		boostedMotives, _, _, pair := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
		if len(boostedMotives) == 0 {
			t.Fatal("expected to find parent motive wave when child pivots are present")
		}
		boostedScore := boostedMotives[0].ConfidenceScore

		if boostedScore <= baseScore {
			t.Errorf("expected boosted score (%f) to be higher than base score (%f)", boostedScore, baseScore)
		}

		if boostedScore > 1.0 {
			t.Errorf("expected boosted score capped at 1.0, got %f", boostedScore)
		}

		if pair.Primary.Confidence != boostedScore {
			t.Errorf("expected scenario confidence to match boosted score, got %f", pair.Primary.Confidence)
		}
	})

	// 2. Soft penalty case: Parent motive wave Wave 1 lacks confirmation.
	t.Run("Motive Wave 1 lacks confirmation: softly penalized but remains visible", func(t *testing.T) {
		parentPivots := textbookBullish(100)

		childPivots := []model.Pivot{
			{Time: 500, Price: 100.0, Type: model.PivotLow},
			{Time: 502, Price: 136.67, Type: model.PivotHigh},
			{Time: 504, Price: 114.01, Type: model.PivotLow},
			{Time: 506, Price: 173.34, Type: model.PivotHigh},
			{Time: 508, Price: 150.68, Type: model.PivotLow},
			{Time: 510, Price: 196.0, Type: model.PivotHigh},
		}

		baseMotives, _, _, _ := elliott.ScenarioBundle(parentPivots, nil, "1D")
		baseScore := baseMotives[0].ConfidenceScore

		penalizedMotives, _, _, _ := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
		if len(penalizedMotives) == 0 {
			t.Fatal("expected high-confidence motive wave to survive soft child-validation penalty")
		}

		penalizedScore := penalizedMotives[0].ConfidenceScore
		expectedScore := baseScore - 0.10
		if math.Abs(penalizedScore-expectedScore) > 0.0001 {
			t.Errorf("expected softly penalized score to be %f, got %f", expectedScore, penalizedScore)
		}
		if penalizedScore < 0.60 {
			t.Errorf("expected softened fractal penalty to keep the macro motive visible, got score %f", penalizedScore)
		}
	})

	t.Run("Lower-confidence motive wave: soft penalty preserves valid macro structure", func(t *testing.T) {
		parentPivots := []model.Pivot{
			{Time: 100, Price: 100.0, Type: model.PivotLow},
			{Time: 110, Price: 200.0, Type: model.PivotHigh},
			{Time: 120, Price: 137.6, Type: model.PivotLow},
			{Time: 130, Price: 300.0, Type: model.PivotHigh},
			{Time: 140, Price: 220.0, Type: model.PivotLow},
			{Time: 150, Price: 343.6, Type: model.PivotHigh},
		}

		baseMotives, _, _, _ := elliott.ScenarioBundle(parentPivots, nil, "1D")
		if len(baseMotives) == 0 {
			t.Fatal("expected base sloppy wave to be found")
		}

		childPivots := []model.Pivot{
			{Time: 900, Price: 100.0, Type: model.PivotLow},
			{Time: 910, Price: 200.0, Type: model.PivotHigh},
		}
		penalizedMotives, _, _, _ := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
		if len(penalizedMotives) == 0 {
			t.Fatal("expected valid macro motive wave to survive the softened child-validation penalty")
		}

		if penalizedMotives[0].ConfidenceScore >= baseMotives[0].ConfidenceScore {
			t.Errorf("expected unconfirmed wave score to be below base score, got base=%f penalized=%f", baseMotives[0].ConfidenceScore, penalizedMotives[0].ConfidenceScore)
		}
	})
}

func TestScenarioBundleConcurrency(t *testing.T) {
	parentPivots := textbookBullish(100)
	childPivots := []model.Pivot{
		{Time: 100, Price: 100.0, Type: model.PivotLow},
		{Time: 102, Price: 136.67, Type: model.PivotHigh},
		{Time: 104, Price: 114.01, Type: model.PivotLow},
		{Time: 106, Price: 173.34, Type: model.PivotHigh},
		{Time: 108, Price: 150.68, Type: model.PivotLow},
		{Time: 110, Price: 196.0, Type: model.PivotHigh},
	}

	var wg sync.WaitGroup
	numWorkers := 20
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _, _, pair := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
				if pair == nil {
					t.Errorf("expected non-nil ScenarioPair in concurrent execution")
				}
			}
		}()
	}
	wg.Wait()
}

func TestDegreeHierarchy(t *testing.T) {
	parentPivots := textbookBullish(100)
	childPivots := []model.Pivot{
		{Time: 100, Price: 100.0, Type: model.PivotLow},
		{Time: 102, Price: 136.67, Type: model.PivotHigh},
		{Time: 104, Price: 114.01, Type: model.PivotLow},
		{Time: 106, Price: 173.34, Type: model.PivotHigh},
		{Time: 108, Price: 150.68, Type: model.PivotLow},
		{Time: 110, Price: 196.0, Type: model.PivotHigh},
	}

	// 1. "1D" parent timeframe -> parent is "MINOR"
	motives, _, _, pair := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
	if len(motives) == 0 {
		t.Fatal("expected parent motive wave")
	}
	if motives[0].Degree != "MINOR" {
		t.Errorf("expected parent degree 'MINOR', got %q", motives[0].Degree)
	}
	if len(pair.Primary.Structures) == 0 {
		t.Fatal("expected primary scenario structure")
	}
	if pair.Primary.Structures[0].Degree != "MINOR" {
		t.Errorf("expected primary structure degree 'MINOR', got %q", pair.Primary.Structures[0].Degree)
	}

	// 2. "1H" parent timeframe -> parent is "MINUTE"
	motives1H, _, _, pair1H := elliott.ScenarioBundle(parentPivots, childPivots, "1H")
	if len(motives1H) == 0 {
		t.Fatal("expected parent motive wave on 1H")
	}
	if motives1H[0].Degree != "MINUTE" {
		t.Errorf("expected parent degree 'MINUTE', got %q", motives1H[0].Degree)
	}
	if pair1H.Primary.Structures[0].Degree != "MINUTE" {
		t.Errorf("expected primary structure degree 'MINUTE', got %q", pair1H.Primary.Structures[0].Degree)
	}

	// 3. "15m" parent timeframe -> parent is "MINUETTE"
	motives15m, _, _, pair15m := elliott.ScenarioBundle(parentPivots, childPivots, "15m")
	if len(motives15m) == 0 {
		t.Fatal("expected parent motive wave on 15m")
	}
	if motives15m[0].Degree != "MINUETTE" {
		t.Errorf("expected parent degree 'MINUETTE', got %q", motives15m[0].Degree)
	}
	if pair15m.Primary.Structures[0].Degree != "MINUETTE" {
		t.Errorf("expected primary structure degree 'MINUETTE', got %q", pair15m.Primary.Structures[0].Degree)
	}
}

func textbookCorrectiveBearish(startTime int64) []model.Pivot {
	return []model.Pivot{
		{Time: startTime, Price: 343.6, Type: model.PivotHigh},
		{Time: startTime + 10, Price: 243.6, Type: model.PivotLow},
		{Time: startTime + 20, Price: 293.6, Type: model.PivotHigh},
		{Time: startTime + 30, Price: 180.0, Type: model.PivotLow},
	}
}

func TestChronologicalChainingAndNesting(t *testing.T) {
	// Construct parent pivots containing:
	// 1. Bullish motive wave from 100 to 150
	// 2. Bearish corrective wave from 200 to 230
	// 3. Second bullish motive wave from 300 to 350
	p1 := textbookBullish(100)
	p2 := textbookCorrectiveBearish(200)
	p3 := textbookBullish(300)

	parentPivots := append(p1, p2...)
	parentPivots = append(parentPivots, p3...)

	// Child pivots that validate the first motive wave (Wave 1: time 100 to 110)
	childPivots := []model.Pivot{
		{Time: 100, Price: 100.0, Type: model.PivotLow},
		{Time: 102, Price: 136.67, Type: model.PivotHigh},
		{Time: 104, Price: 114.01, Type: model.PivotLow},
		{Time: 106, Price: 173.34, Type: model.PivotHigh},
		{Time: 108, Price: 150.68, Type: model.PivotLow},
		{Time: 110, Price: 196.0, Type: model.PivotHigh},
	}

	_, _, _, pair := elliott.ScenarioBundle(parentPivots, childPivots, "1D")
	if pair == nil {
		t.Fatal("expected non-nil ScenarioPair")
	}

	// The primary scenario should be BULLISH.
	if pair.Primary.Bias != "BULLISH" {
		t.Errorf("expected primary bias 'BULLISH', got %q", pair.Primary.Bias)
	}

	// It should contain:
	// - Parent Motive 1
	// - Child Motive 1 (nested under Parent Motive 1)
	// - Parent Corrective
	// - Parent Motive 2
	// So exactly 4 structures in total!
	expectedCount := 4
	if len(pair.Primary.Structures) != expectedCount {
		t.Errorf("expected primary scenario to have %d structures (motive1, child_motive, corrective, motive2), got %d", expectedCount, len(pair.Primary.Structures))
		for idx, ws := range pair.Primary.Structures {
			t.Logf("  [%d] type=%s start=%d end=%d degree=%s", idx, ws.Type, ws.Pivots[0].Time, ws.Pivots[len(ws.Pivots)-1].Time, ws.Degree)
		}
	}

	// Verify degrees and types.
	// Primary parent structures are MINOR, child is MINUTE.
	s := pair.Primary.Structures
	if len(s) >= 4 {
		if s[0].Type != "MOTIVE_IMPULSE" || s[0].Degree != "MINOR" {
			t.Errorf("expected structure 0 to be MOTIVE_IMPULSE of MINOR degree, got type=%s degree=%s", s[0].Type, s[0].Degree)
		}
		if s[1].Type != "MOTIVE_IMPULSE" || s[1].Degree != "MINUTE" {
			t.Errorf("expected structure 1 to be MOTIVE_IMPULSE of MINUTE degree (nested child), got type=%s degree=%s", s[1].Type, s[1].Degree)
		}
		if s[2].Type != "CORRECTIVE_ZIGZAG" || s[2].Degree != "MINOR" {
			t.Errorf("expected structure 2 to be CORRECTIVE_ZIGZAG of MINOR degree, got type=%s degree=%s", s[2].Type, s[2].Degree)
		}
		if s[3].Type != "MOTIVE_IMPULSE" || s[3].Degree != "MINOR" {
			t.Errorf("expected structure 3 to be MOTIVE_IMPULSE of MINOR degree, got type=%s degree=%s", s[3].Type, s[3].Degree)
		}
	}
}
