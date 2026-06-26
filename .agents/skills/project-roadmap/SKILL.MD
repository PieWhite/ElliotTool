# MASTER SYSTEM CONTEXT & COMPLETE ROADMAP: PROJECT WAVESIGHT

## 1. Project Overview & Architecture
- **Project Name:** WaveSight (High-Performance Automated Elliott Wave & Fibonacci Analysis App).
- **Current Phase:** Phase 2 (The Frost & Prechter Advanced Theoretical Engine).
- **Core Philosophy:** Extreme performance, zero fluff, TDD, DIP, memory optimization, and strict adherence to the *Frost & Prechter Elliott Wave Principle*.

## 2. Tech Stack & Architectural Decisions
- **Backend:** Go (Golang 1.22+) using native method-based routing (`http.NewServeMux()`).
- **JSON Serialization:** `mailru/easyjson` code generation.
- **Database:** SQLite via `github.com/glebarez/go-sqlite` (Pure Go, CGO-free).
- **Frontend:** Vue 3 (Composition API, `<script setup lang="ts">`, Vite), `shallowRef` for raw arrays to prevent lag.
- **Charting:** TradingView Lightweight Charts (`lightweight-charts`).

---

## 3. Phase 1 Completion Status
- [x] **Step 1 - 3:** Polygon API Fetcher, SQLite Caching, ZigZag State Machine.
- [x] **Step 4:** Basic Motive (5) & Corrective (ABC) Rules Engine.
- [x] **Step 5 - 7:** High-Performance REST API & Vue 3 Canvas TargetBox Rendering.

---

## 4. Phase 2: Advanced Frost & Prechter Implementation (CURRENT ROADMAP)

### [x] Step 8: Advanced Motive Structures (Diagonals & Truncations)
- **Objective:** Upgrade the `MotiveWave` scanner to recognize variations of the 5-wave impulse.
- **Leading/Ending Diagonals (Wedges):** Implement logic where Rule 3 (Wave 4 cannot overlap Wave 1) is safely ignored IF the structure converges into a wedge shape (Wave 1 > Wave 3 > Wave 5).
- **Truncations:** Allow Wave 5 to terminate below the peak of Wave 3, provided the sub-waves and Fibonacci extensions of Wave 3 mathematically justify exhaustion.
- **Incomplete Motives (1-2-3):** Identify developing impulse structures (waves 1, 2, and 3 completed) and project the predictive path for waves 4 and 5.

### [x] Step 9: Advanced Corrective Structures (Triangles & Complex Threes)
- **Objective:** Expand `CorrectiveWave` beyond standard ZigZags and Flats.
- **Triangles (ABCDE):** Implemented a 5-pivot sideways scanner (`evaluateTriangle`) with strict contracting bounds (|A|>|B|>|C|>|D|>|E|). Both BULLISH (Low start) and BEARISH (High start) variants supported. `WD`/`WE` pivot fields added to model.
- **Complex Corrections (WXY):** Implemented `evaluateWXY` to detect Double Threes via an 8-pivot window. Validates W-wave (ZigZag/Flat), X-wave retracement ≤ 90%, Y-wave (ZigZag/Flat), and net directional continuation. `WX` pivot field added to model.
- **Domain Model:** `CorrectiveWave` extended with `WD *Pivot`, `WE *Pivot`, `WX *Pivot` (all `omitempty`). `type` field now documents `"TRIANGLE"` and `"WXY"`.
- **easyjson:** Regenerated `corrective_wave_easyjson.go` after model changes.
- **TDD:** 9 new table-driven tests (4 Triangle, 4 WXY, 1 edge case). All 30 tests pass. Benchmarks show ≤4 allocations per call (pre-allocated result slice only; scanning loops are zero-allocation).
- **Frontend Sync (Step 8 backfill):** `MotiveWave` TS interface gains `is_diagonal`/`is_truncated`. `IncompleteWave` interface added with `target_box`. `AnalysisResponse` gains `incomplete_waves`. `ChartWidget.vue` renders diagonals as dashed, truncated W5 as `"5T"`, incomplete waves as cyan dotted lines with teal target boxes, Triangles in teal, WXY in indigo. `BoxPrimitive` extended with optional fill/stroke color params.

### [x] Step 10: The Probabilistic Engine (Primary vs. Alternate Counts)
- **Objective:** Transition from returning a single "truth" to returning an array of viable scenarios.
- **Model (`pkg/model/response.go`):** Added `WaveStructure` (type, pivots, purple_boxes, confidence_score), `AnalysisScenario` (bias, confidence, structures), and `ScenarioPair` (primary, alternate). `AnalysisResponse` gains `Scenarios *ScenarioPair`. Legacy flat arrays kept with `omitempty`. All new structs annotated `//easyjson:json`; `response_easyjson.go` regenerated.
- **Scorer (`pkg/elliott/scorer.go`):** `ScenarioBundle` function delegates to `MatchMotiveWaves`, `MatchCorrectiveWaves`, and `MatchIncompleteWaves` without re-implementing scanning logic. Converts results to `WaveStructure`, sorts by confidence descending, and emits primary (highest-confidence) + alternate (second-highest, or inverse-bias placeholder with confidence 0.0 when only 1 structure found).
- **API (`pkg/api/api.go`):** Wired `ScenarioBundle` into `handleAnalyze`; response carries both `scenarios` object and legacy flat arrays simultaneously.
- **TDD (`pkg/elliott/scorer_test.go`):** 4 table-driven tests: two-wave ordering, single-wave inverse fallback, confidence-ordering invariant (primary ≥ alternate), empty-input edge case. All 34 tests pass.
- **Frontend Interfaces (`useMarketData.ts`):** Added `WaveStructure`, `AnalysisScenario` TypeScript interfaces. `AnalysisResponse` gains `scenarios`. Composable exposes `scenarios` shallowRef, `activeScenario` computed, `activeScenarioBias` ref, and `setScenario` helper.
- **UI Toggle (`App.vue`):** Added a 5-column grid with a **Scenario View** toggle column containing ▲ BULLISH / ▼ BEARISH buttons. Active BULLISH = green border/bg, active BEARISH = red border/bg. Confidence badge shows % for each scenario dynamically.
- **Chart Rendering (`ChartWidget.vue`):** Accepts optional `activeScenario?: AnalysisScenario` prop. When provided, `renderScenarioStructures` renders each `WaveStructure`'s pivot sequence as a colour-coded line (motive = green/red solid, corrective = amber/teal/indigo dashed, incomplete = cyan dotted) and attaches `purple_boxes`. Falls back to legacy flat array rendering when no scenario is active. Watch list includes `activeScenario` so toggling immediately redraws.

### [x] Step 11: Comprehensive Purple Box Matrix (Target Zones)
- **Objective:** Calculate dynamic Purple Boxes for *every* developing wave based on the uploaded `WaveRatios.pdf`.
- **Wave 3 Targets:** Added `calculateWave3TargetBoxes` for 1.618, 2.618, and 4.236 extensions of Wave 1.
- **Wave 4 Targets:** Box at 38.2% retracement of Wave 3, terminating within the territory of the preceding 4th wave of one lesser degree.
- **Wave 5 Targets:** Completed motive waves now emit `purple_boxes` with three F&P target zones: 100% of Wave 1, 61.8% of net Start-to-Wave-3, and 161.8% of Wave 1.
- **Wave C Targets:** Corrective ABC/WXY structures now emit `purple_boxes` at 100% and 161.8% of Wave A.
- **Domain Model:** `MotiveWave.PurpleBox` replaced with `MotiveWave.PurpleBoxes []TargetBox`; `CorrectiveWave` gains `PurpleBoxes []TargetBox`. `motive_wave_easyjson.go` and `corrective_wave_easyjson.go` regenerated.
- **Scorer:** Scenario `WaveStructure.PurpleBoxes` now receives all motive/corrective target boxes.
- **Frontend:** TypeScript contracts use `purple_boxes`; `ChartWidget.vue` renders every motive, corrective, and scenario target box as a separate `BoxPrimitive`.
- **TDD:** Added matrix coverage for textbook bullish motive Wave 5 targets and textbook bearish ZigZag Wave C targets. `go test ./pkg/elliott/... -v` passes.

### [x] Step 12: Multi-Degree Fractal Validation (Waves within Waves)
- **Objective:** Apply the fractal nature of Elliott Wave.
- **Logic:** A macro Wave 1 on a 1D chart is only valid if the `elliott` package can break it down into a smaller 5-wave motive structure on the 1H chart.
- **Implementation:**
  - Designed a concurrent validation layer that scans parent and child timeframes in parallel using goroutines.
  - Implemented relative time tolerance (25%) and price tolerance (20%) to match parent Wave 1/incomplete W1 with child 5-wave motive substructures.
  - Configured confidence scoring adjustments: confirmed waves receive a `+0.15` boost (capped at `1.0`), and unconfirmed waves receive a `-0.30` penalty, resulting in rejection if score falls below `0.60`.
  - Concurrently fetches parent and child candles in the API router for `"1D" -> "1H"`, `"1H" -> "15m"`, and `"15m" -> "1m"` pairs.
  - Preserved 100% API and frontend compatibility while providing fully validated multi-degree scenarios.

### [x] Step 13: Multi-Degree Labeling & Degree Hierarchy Rendering
- **Objective:** Distinguish wave degrees visually and structurally per Frost & Prechter.
- **Logic:** Waves within waves are labeled with distinct degree notations (e.g. Primary `①`, Intermediate `(1)`, Minor `1`, Minute `i`, Minuette `[i]`, Subminuette `(i)`).
- **Implementation:**
  - **Backend Models:** Added `degree` metadata field to `MotiveWave`, `CorrectiveWave`, `IncompleteWave`, and `WaveStructure` models.
  - **JSON Serialization:** Re-generated `easyjson` files (`motive_wave_easyjson.go`, `corrective_wave_easyjson.go`, `incomplete_wave_easyjson.go`, `response_easyjson.go`) to output the `degree` field.
  - **Scoring & Validation:** Updated `ScenarioBundle` in `scorer.go` to accept a `parentTimeframe` parameter. Maps the timeframe to appropriate Elliott Wave degrees (e.g. 1D -> MINOR, 1H -> MINUTE, 15m -> MINUETTE). Propagates the mapped `Degree` to all waves and `WaveStructure` responses.
  - **Frontend Refactoring:** Updated TypeScript definitions in `useMarketData.ts` to carry the optional `degree` parameter. Refactored `ChartWidget.vue` to format wave label markers using Frost & Prechter notations (e.g. primary circled, intermediate parenthesized, minor plain, minute lowercase, minuette bracketed lowercase, subminuette parenthesized lowercase) and color-code both the lines and markers accordingly.
  - **Verification:** Added TDD tests validating degree hierarchy mapping in `scorer_test.go` and verified clean frontend production builds.

### [x] Step 14: Chronological Chaining & Hierarchical Wave Nesting (Complete Chart Analysis)
- **Objective:** Assemble local isolated wave structures into a continuous, multi-degree nested chart analysis.
- **Implementation:**
  - **Hierarchical Nesting:** Refactored `hasValidSubwave` to `findValidSubwave`, which returns the matching child structure that validated the parent wave. Retained these matched child structures and associated them with their parents in a sub-wave lookup map inside `ScenarioBundle`.
  - **Chronological Chaining (Interval Scheduling):** Designed and implemented a dynamic programming (DP) interval scheduling algorithm (`findBestChain`) running in $O(N^2)$ time with zero recursion overhead. It groups waves into Bullish and Bearish candidate pools and finds the optimal chain of mutually non-overlapping wave structures that maximizes the sum of confidence scores.
  - **Scenario Assembly:** Refactored scenario compilation to output the entire Bullish chain (with its nested child sub-waves) as the Bullish scenario and the Bearish chain (with its nested child sub-waves) as the Bearish scenario. The higher confidence chain becomes `Primary`, and the other becomes `Alternate`.
  - **Frontend Verification:** Checked and confirmed that `ChartWidget.vue`'s scenario rendering handles rendering of multiple lines, markers, and target boxes concurrently with degree-specific styling, producing a complete, nested, and chained chart analysis.
  - **TDD:** Added `TestChronologicalChainingAndNesting` in `scorer_test.go` verifying correct DP assembly of a motive-corrective-motive chain with a nested sub-wave. All tests pass successfully.

### [ ] Step 15: Wave Detection Refinement & Scenario Visual Completeness (Complete Chart Analysis)
- **Objective:** Refine backend scoring rules and frontend charting elements so motive (1-5), incomplete (1-2-3), and corrective (ABC/Triangle/WXY) waves along with their Purple Box target zones render comprehensively and look like a professional multi-degree chart analysis.
- **Backend Scoring & Validation Adjustments:**
  - Soften the fractal validation penalty: instead of hard-rejecting macro waves when child validation is missing or slightly offset, apply a milder penalty or allow a validation tier system to preserve high-scoring macro motive structures.
  - Add configuration toggles or parameters for deviation adjustment so that users can control scan sensitivity directly.
- **Frontend & Canvas Rendering Completeness:**
  - Update `renderScenarioStructures` in `ChartWidget.vue` to map distinct styling parameters per structure type (e.g., solid lines for motive impulses, dashed lines for diagonals/wedges, dotted lines for incomplete structures).
  - Add color parameterization for `BoxPrimitive` targets inside scenario view: green-tinted boxes for bullish Wave 3/5 projections, red-tinted for bearish, and teal for incomplete Wave 4 target boxes.
  - Implement marker collision prevention (e.g., vertical offsets or stacking) to keep labels clear and legible when parent and child waves share the same pivot timestamps.

---

## 5. Target JSON Contract Strategy (Phase 2 Upgrade)
The API layer will be upgraded to serve probabilistic scenarios:
```json
{
  "ticker": "AAPL",
  "timeframe": "1D",
  "candles": [ ... ],
  "scenarios": {
    "primary": {
      "bias": "BULLISH",
      "confidence": 0.94,
      "structures": [
        { "type": "MOTIVE_IMPULSE", "pivots": [...], "purple_boxes": [...] }
      ]
    },
    "alternate": {
      "bias": "BEARISH",
      "confidence": 0.76,
      "structures": [
        { "type": "CORRECTIVE_ZIGZAG", "pivots": [...], "purple_boxes": [...] }
      ]
    }
  }
}
