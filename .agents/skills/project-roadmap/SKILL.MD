# Description
The master architecture, tech stack, and production-grade step-by-step execution plan for the WaveSight project. Trigger this when asked about the project status, next steps, or overall context.

# Project: WaveSight (Elliott Wave & Fibonacci Charting App)
**Document Type:** Master Context & AI Agent Directives
**Current Phase:** Phase 1 (MVP - Production-Grade Mathematical Engine)

## 1. Project Overview
The goal of this project is to build a high-performance, automated Elliott Wave analysis web application. The app fetches OHLCV data for global markets, caches it efficiently, filters market noise using a custom state-machine ZigZag algorithm, and runs a strict, multi-layered quantitative rule engine to validate Elliott Wave patterns and project Fibonacci target zones (the "Purple Box") for frontend visualization.

## 2. Tech Stack & Technologies
- **Backend:** Go (Golang 1.22+) - *Chosen for high performance, concurrency, and low latency.*
- **JSON Serialization:** `mailru/easyjson` - *Strictly used over standard reflection-based encoding/json on hot paths to eliminate allocation overhead during mass candle parsing.*
- **Frontend:** Vue 3 (Composition API, `<script setup lang="ts">`, Vite, strict TypeScript) - *For type-safe, optimized client-side interactivity.*
- **Charting Library:** TradingView Lightweight Charts (`lightweight-charts`) - *For high-performance canvas rendering.*
- **Database / Caching:** SQLite via `github.com/glebarez/go-sqlite` - *A pure-Go, CGO-free driver selected to ensure effortless compilation, cross-platform portability, and clean in-memory integration testing.*
- **Market Data Provider:** Polygon.io (REST API, Aggregates/OHLCV endpoint) - *Utilizing maximum batch boundaries (limit=50000).*

## 3. Architecture & Data Flow
1. **User Request:** User requests a ticker and timeframe via the Vue 3 dashboard.
2. **Backend Cache Check:** Go queries SQLite. If valid, cached candles are read instantly via optimized index scans.
3. **Data Fetching:** On cache-miss, Go fetches a 50,000 candle batch from the aggregates API, executing bulk-upserts inside a single database transaction (`ON CONFLICT DO UPDATE`).
4. **Analysis Pipeline:**
   - **Noise Filtering:** The `zigzag` package runs a lineair ($O(N)$) state-machine to detect structural pivots, resolving intraday price order using `lowCameFirst` logic.
   - **Pattern Matching (Sliding Window):** The `elliott` package evaluates pivot sub-sequences against cardinal rules, Fibonacci targets, and geometric channels.
5. **API Response:** Go transmits a high-throughput, code-generated JSON payload to the client.
6. **Visualization:** Vue processes the payload and mounts rendering timelines to the canvas layer.

## 4. Backend Rules & Guidelines (Go)
- **Zero Reflection on Hot Paths:** Structs like `Candle` and `Pivot` must be annotated with `//easyjson:json`. Do not use default reflection-based JSON marshalling for analytics endpoints.
- **Memory Optimization:** Always pre-allocate slice capacities when executing mass transformations (e.g., `make([]Pivot, 0, len(candles)/10)`). Use value receivers for small mathematical structs and pointers for large payloads.
- **Dependency Inversion (DIP):** Decouple your infrastructure. External HTTP requests and database layers must be locked behind mockable Go interfaces (`HTTPClient`, `CandleRepository`).
- **No External Test Assertions:** Maintain a lean build. Avoid third-party testing frameworks like `testify`. Strictly utilize standard library `if expected != actual` checks and `t.Errorf()`. Use table-driven testing loops.
- **Strict Environment Separation:** Critical API keys must be loaded via a lightweight loader using `os.Getenv` from a `.env` file that is strictly blacklisted in `.gitignore`.

## 5. Frontend Rules & Guidelines (Vue 3)
- **Performance Reactivity:** Never use deep reactivity (`ref` or `reactive`) for mass market datasets. Massive arrays of OHLCV candles and processed pivots must be wrapped strictly inside a **`shallowRef`** to bypass Vue's proxy overhead and prevent browser lag.
- **Composition Isolation:** Vue components must remain pure UI templates. Business logic, API communication, and chart coordinate transformers must be completely abstracted into composable files (e.g., `useMarketData.ts`).
- **Lifecycle Cleanup:** Chart wrappers must explicitly tear down canvas instances and unsubscribe from window resize events inside `onUnmounted` to protect against memory leaks.

## 6. Step-by-Step Execution Plan
**AI Agent: Do not jump ahead. Wait for the human developer to specify which step we are currently working on.**

- [x] **Step 1: Setup & Polygon API** (CGO-free interface implementation, 50k batching limit, Go module initialized).
- [x] **Step 2: SQLite Caching & Security** (Pure-Go driver integration, transactional batch bulk-inserts with UPSERT, .env protection).
- [x] **Step 3: The ZigZag Algorithm** (Single-pass state machine, intraday candle price action order resolver, TDD validated edge-cases).
- [x] **Step 4: Elliott Wave Core Rule Engine (5-Wave Motive Structures)**
- [x] **Step 4b: Fibonacci Ratio Validator (Scoring & Filtering Engine)**
- [x] **Step 4c: Channeling & Price Target Projection**
- [x] **Step 5: Corrective Waves Engine (3-Wave ABC Structures)**
- [x] **Step 6: High-Performance JSON API Exposure**
- [x] **Step 7: Vue 3 Frontend Dashboard & Canvas Charting**


## 7. JSON Contract Strategy (Target Format)
```json
{
  "ticker": "AAPL",
  "timeframe": "10m",
  "candles": [
    {"time": 1780272000, "open": 180.1, "high": 181.5, "low": 179.9, "close": 181.0, "volume": 5000}
  ],
  "motive_waves": [
    {
      "start": {"time": 1780272000, "price": 179.9, "type": "LOW"},
      "w1": {"time": 1780275600, "price": 181.5, "type": "HIGH"},
      "w2": {"time": 1780279200, "price": 180.2, "type": "LOW"},
      "w3": {"time": 1780286400, "price": 184.8, "type": "HIGH"},
      "w4": {"time": 1780290000, "price": 182.1, "type": "LOW"},
      "w5": {"time": 1780300800, "price": 186.5, "type": "HIGH"},
      "direction": "BULLISH",
      "confidence_score": 0.92,
      "purple_box": {
        "start_time": 1780279200,
        "end_time": 1780286400,
        "min_price": 182.8,
        "max_price": 184.5
      }
    }
  ],
  "corrective_waves": [
    {
      "start": {"time": 1780300800, "price": 186.5, "type": "HIGH"},
      "wa": {"time": 1780304400, "price": 183.0, "type": "LOW"},
      "wb": {"time": 1780308000, "price": 185.0, "type": "HIGH"},
      "wc": {"time": 1780311600, "price": 181.0, "type": "LOW"},
      "type": "ZIGZAG",
      "direction": "BEARISH"
    }
  ]
}
```