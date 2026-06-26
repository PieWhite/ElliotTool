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

### [ ] Step 9: Advanced Corrective Structures (Triangles & Complex Threes)
- **Objective:** Expand `CorrectiveWave` beyond standard ZigZags and Flats.
- **Triangles (ABCDE):** Implement a 5-pivot sideways scanner with contracting/expanding bounds.
- **Complex Corrections (WXY / WXYZ):** Build logic to detect "Double Threes" and "Triple Threes" linked by an "X" wave, as defined by Frost & Prechter.

### [ ] Step 10: The Probabilistic Engine (Primary vs. Alternate Counts)
- **Objective:** Transition from returning a single "truth" to returning an array of viable scenarios.
- **Scoring Weight Adjustment:** Upgrade `ConfidenceScore` to weigh strict Fibonacci ratio compliance heavily.
- **Scenario Bundling:** Output a `Primary Count` (highest score) and an `Alternate Count` (second-highest score or inverse directional bias).
- **Frontend Toggle:** Implement UI controls allowing the user to switch the canvas between the Bullish Scenario and the Bearish Scenario.

### [ ] Step 11: Comprehensive Purple Box Matrix (Target Zones)
- **Objective:** Calculate dynamic Purple Boxes for *every* developing wave based on the uploaded `WaveRatios.pdf`.
- **Wave 3 Targets:** Box at 1.618, 2.618, or 4.236 extension of Wave 1.
- **Wave 4 Targets:** Box at 38.2% retracement of Wave 3, terminating within the territory of the preceding 4th wave of one lesser degree.
- **Wave C Targets:** Box at 100% or 161.8% of Wave A.
- Ensure the backend calculates these boxes for both completed and *in-progress* structures.

### [ ] Step 12: Multi-Degree Fractal Validation (Waves within Waves)
- **Objective:** Apply the fractal nature of Elliott Wave.
- **Logic:** A macro Wave 1 on a 1D chart is only valid if the `elliott` package can break it down into a smaller 5-wave motive structure on the 1H chart.
- **Implementation:** Run the scanners concurrently over multiple timeframes and cross-reference pivot alignment.

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