---
name: project-roadmap
description: WaveSight theory conformance matrix and release gates.
---

# WaveSight conformance contract

This file is not an independent Elliott Wave source. The executable registry in
`internal/domain/wave/rules.go` is derived from:

1. A.J. Frost and Robert Prechter, *Elliott Wave Principle*.
2. *WaveRatios* only for supplementary ratio priority.

Existing code and historical tests never override those documents.

## Executable matrix

| Concern | Executable owner | Required evidence |
|---|---|---|
| Motive rules and 5-3-5-3-5 grammar | `rules.go`, `parser.go` | positive/negative mirrored fixtures |
| Leading diagonal, 1/A only | `childAllowed`, placement audit | 5-3-5-3-5, overlap, convergence |
| Ending diagonal, 5/C only | `childAllowed`, placement audit | 3-3-3-3-3, overlap, convergence |
| Truncation | motive parser | internal five plus strong-third guideline |
| Zigzags and combinations | corrective parser | W-X-Y(-X-Z), every zigzag 5-3-5 |
| Flats | corrective parser | regular, expanded and running 3-3-5 |
| Triangles | corrective parser | contracting, barrier, running and expanding 3-3-3-3-3 |
| Combination restrictions | combination parser | triangle terminal; at most one triangle/zigzag |
| Developing structures | partial grammar parser | confirmed children, next mode, invalidation, targets |
| Relative degree | pivot-lattice level | never fixed to timeframe |
| Orthodox endpoints | `WaveNode` audit | stored endpoints drive every measurement |
| Guidelines and priors | `guidelines.go` | ranking evidence only; never structural rejection |
| Price and time targets | `targets.go` | exact levels retained; boxes need independent families |
| Scenario selection | `engine.go` | current region, deterministic top five, explicit indeterminate |
| Immutable snapshots | repository and v2 API | request hash, candle fingerprint and versioned ID |
| Chart rendering | `WaveOverlayPrimitive.ts` | one canvas overlay, logical future bars, no timestamp mutation |

## Release gates

- Every registered rule has PASS and FAIL audit-path coverage.
- Handwritten `internal/domain/wave` statement coverage is at least 90%;
  generated easyjson is checked separately by reproducible generation.
- `go test`, race detection, vet, formatting and performance envelopes pass.
- ESLint, strict TypeScript, Vitest, Playwright desktop/mobile and production
  build pass.
- No legacy v1 arrays or adapters are exposed.
- No scenario with a hard-rule failure can leave the engine.
- Unobservable breadth, news or sentiment stays `NOT_OBSERVABLE`.
- A target without independent confluence renders as a line, never as a box.
