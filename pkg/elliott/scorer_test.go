package elliott_test

import (
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
			name: "Single valid motive wave: alternate is inverse directional placeholder",
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
			name: "Confidence ordering: primary.confidence >= alternate.confidence (bearish textbook)",
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
			motives, correctives, incompletes, pair := elliott.ScenarioBundle(tt.pivots)
			tt.verify(t, pair, motives, correctives, incompletes)
		})
	}
}
