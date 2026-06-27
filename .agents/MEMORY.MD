# Description
Manages the long-term memory, technical decisions, and learned lessons for this specific project. Triggers when a complex bug is resolved, a new architectural decision is made, or when the user asks to save a lesson.

# Instructions
- **The Memory Bank:** The file `.agents/MEMORY.md` is your long-term memory storage. 
- **Read Before Acting:** When solving complex bugs related to Go or Vue in this project, briefly scan `.agents/MEMORY.md` first to ensure you are not repeating past mistakes or violating project-specific workarounds.
- **Proactive Logging:** If we spend significant time debugging a specific issue (e.g., a quirk in Polygon.io's API, a specific reactivity issue in Vue, or a Go concurrency gotcha), you MUST proactively suggest adding a brief technical summary to `.agents/MEMORY.md`.
- **Formatting:** When writing to `.agents/MEMORY.md`, use clear headings (e.g., `### Vue Chart Reactivity (Date)`), explain the *Problem*, and document the *Solution/Decision*.

### Wave Calculation, Scoring, and Chart Visibility Map (2026-06-27)

**Problem / Context:**
The user asked which files are responsible for Elliott Wave calculations, detection algorithms, confidence scoring, and deciding whether a detected wave is worthy of being shown on the chart.

**Solution / Decision:**
Use the following file map when debugging why a wave did or did not appear on the chart.

#### Core wave detection and calculation

- `pkg/elliott/elliott.go`
  - Primary Elliott Wave rules engine.
  - Detects raw wave structures from pivots.
  - Important functions:
    - `MatchMotiveWaves`: finds completed bullish/bearish 5-wave motive structures; applies Elliott structural rules; handles diagonals and truncations.
    - `MatchCorrectiveWaves`: finds corrective structures such as ZigZags, Flats, Triangles, and WXY/double threes.
    - `MatchIncompleteWaves`: finds developing `1-2-3` structures and creates predictive Wave 4 target boxes.
    - `calculateConfidenceScore`: scores completed motive waves from Fibonacci relationships.
    - `calculateIncompleteConfidenceScore`: scores incomplete `1-2-3` waves.
    - `minConfidenceScore`: minimum confidence threshold for accepting/showing waves; currently `0.60`.
    - Target box functions: `calculateTargetBox`, `calculateWave3TargetBoxes`, `calculateWave4TargetBox`, and `calculateWaveCTargetBoxes`.

#### Scenario scoring, ranking, nesting, and filtering

- `pkg/elliott/scorer.go`
  - Scenario assembly and judgment layer.
  - Decides which detected structures become part of primary/alternate scenarios.
  - Important functions/constants:
    - `ScenarioBundle`: main scenario-generation entry point; calls motive/corrective/incomplete scanners; applies fractal validation; filters by confidence; builds scenarios.
    - `findValidSubwave`: checks whether a macro wave has a matching lower-timeframe child motive wave for fractal validation.
    - `fractalValidationBoost`: confidence boost when child validation is found; currently `+0.15`.
    - `fractalValidationPenalty`: confidence penalty when child validation is missing; currently softened to `-0.10`.
    - `motiveToStructure`, `correctiveToStructure`, `incompleteToStructure`: convert detected waves into generic chart scenario structures.
    - `findBestChain`: chooses the best chronological chain of non-overlapping structures. A detected wave may not appear if it loses to a better scenario chain.
    - `getChainConfidence`: calculates scenario confidence from selected structures.
    - `assembleScenarioStructures`: returns the selected hierarchical scenario chain directly (nesting sub-structures recursively).
    - `isBullishScenarioComponent` and `isBearishScenarioComponent`: classify structures into bullish/bearish scenario pools.

#### Pivot generation before Elliott detection

- `pkg/swing/swing.go`
  - Defines the `SwingDetector` interface and the `VolatilitySwingDetector` implementation.
  - Elliott detection operates on pivots, not raw candles.
  - Uses volatility-adaptive rolling Average True Range ($k \times \text{ATR}$) to decide which price swings become significant pivots.
  - If pivots are too sparse or noisy, wave detection and visibility will change.

#### API orchestration

- `pkg/api/api.go`
  - Receives frontend analysis requests and orchestrates backend analysis.
  - Fetches candles, generates parent/child pivots, calls `elliott.ScenarioBundle`, and returns the response.
  - Important for checking timeframe, deviation, and child-timeframe validation behavior.

#### Backend data models

- `pkg/model/motive_wave.go`
  - Completed motive wave model: `Start`, `W1`-`W5`, `Direction`, `ConfidenceScore`, `PurpleBoxes`, `IsDiagonal`, `IsTruncated`, `Degree`.

- `pkg/model/corrective_wave.go`
  - Corrective model: `Start`, `WA`, `WB`, `WC`, optional `WD`, `WE`, `WX`, `Type`, `Direction`, `PurpleBoxes`, `Degree`.

- `pkg/model/incomplete_wave.go`
  - Developing `1-2-3` model: `Start`, `W1`, `W2`, `W3`, `Direction`, `ConfidenceScore`, `TargetBox`, `Degree`.

- `pkg/model/response.go`
  - Final API response models: `WaveStructure` (now containing recursive `SubStructures` list), `AnalysisScenario`, `ScenarioPair`, and `AnalysisResponse`.

#### Frontend rendering, not calculation

- `frontend/src/components/ChartWidget.vue`
  - Does not calculate or judge waves.
  - Renders already-calculated backend structures.
  - Controls line color/style, labels, marker placement, target box color, and scenario display.
  - Important functions:
    - `renderScenarioStructures`: recursively traverses the wave structure tree and renders every sub-structure.
    - `renderFlatWaves`
    - `getScenarioRenderConfig`
    - `attachCollisionAwareMarkers`

- `frontend/src/composables/useMarketData.ts`
  - Defines frontend TypeScript response interfaces and calls the backend API.
  - Receives candles, motive waves, corrective waves, incomplete waves, and scenarios.

#### Debugging order for missing or unexpected waves

When asking “why did this wave appear or not appear on the chart?”, inspect in this order:

1. `pkg/swing/swing.go`
   - Was the pivot generated by the ATR swing detector?

2. `pkg/elliott/elliott.go`
   - Did the pivots satisfy Elliott/Fibonacci rules?
   - Did the wave score above `minConfidenceScore`?

3. `pkg/elliott/scorer.go`
   - Did the wave survive fractal validation?
   - Was it selected into the best bullish/bearish scenario chain?

4. `pkg/api/api.go`
   - Was the correct timeframe/deviation used?
   - Were child pivots provided?

5. `frontend/src/components/ChartWidget.vue`
   - Was the returned structure rendered clearly?

**Most important backend files:**

```text
pkg/swing/swing.go
pkg/elliott/elliott.go
pkg/elliott/scorer.go
pkg/api/api.go
```

**Most important frontend display file:**

```text
frontend/src/components/ChartWidget.vue
```