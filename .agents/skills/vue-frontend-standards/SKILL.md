# Description
Provides strict guidelines for writing the Vue 3 frontend, focusing on the Composition API, strict TypeScript, Clean Architecture, and Web Performance Budgets.

# Instructions

## 1. Vue 3 & TypeScript Standards
- **Strict Composition API:** Exclusively use Vue 3 `<script setup lang="ts">`. Never use the Options API.
- **Strict TypeScript:** All frontend code must be written in strict TypeScript. Define precise interfaces and types for all API payloads (e.g., `Candle`, `WaveTarget`). Avoid `any`.
- **Performance Reactivity:** Use `ref` for primitives. **CRITICAL:** Use `shallowRef` instead of `ref` or `reactive` when storing large datasets (like the arrays of thousands of OHLCV candles). Deep reactivity on massive arrays will crash Vue's performance.

## 2. Architecture & Clean Components
- **Separation of Concerns:** Keep Vue components purely focused on UI and rendering. Extract API data fetching, state management, and charting data transformations into separate TypeScript composables (e.g., `useMarketData.ts`).
- **Component Lifecycle:** Always clean up event listeners and chart instances in `onUnmounted` to prevent memory leaks.
- **Error Boundaries:** Handle backend errors gracefully. If the Go API returns an error (e.g., `429 Too Many Requests` from Polygon), show a clean error state in the UI, do not crash the app.

## 3. Web Performance Budgets (KISS)
- **Lazy Loading:** Use dynamic imports for heavy components or libraries (like the TradingView Lightweight Charts library). Load them on user interaction or route change rather than blocking the initial page load.
- **Tree-Shaking:** Use tree-shaking-friendly imports (e.g., `import { createChart } from 'lightweight-charts'`, not `import * as lw`).
- **DOM Efficiency:** Avoid DOM bloat. Never render thousands of raw HTML elements to represent data; strictly use the Canvas-based charting library for visualizing market data.

## 4. UI/UX Strategy
- **Client-Side Rendering (CSR):** Since this is a highly interactive charting dashboard, prioritize a snappy Client-Side experience.
- **Progressive Enhancement:** Ensure basic structural loading states (skeletons or spinners) are visible immediately while the Go backend crunches the Elliott Wave data.