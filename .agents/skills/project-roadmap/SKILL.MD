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
| Master scenario selection | `internal/domain/master/engine.go` | full observation narrative, material alternatives only, explicit gaps |
| Cross-scale event identity | `CanonicalWaveEvent` | one orthodox event ID reused by every relevant view |
| Canonical data | `market/aggregate.go` | native daily + native minute; session-anchored local projections |
| Relative degree | master graph | inferred from nesting/span across the dataset, never fixed to timeframe |
| View projection | `TimeframeView` | ancestor retention, three-bar child threshold, no parser/provider call |
| Immutable snapshots | repository and v3 API | native fingerprint, master request, revision parent and versioned ID |
| Historical refinement | v3 revision jobs | fetch only missing minute interval; never mutate the parent |
| Chart rendering | `WaveOverlayPrimitive.ts` | one canvas overlay, canonical events, logical future bars, no timestamp mutation |

## Release gates

- Every registered rule has PASS and FAIL audit-path coverage.
- Handwritten `internal/domain/wave` statement coverage is at least 90%;
  generated easyjson is checked separately by reproducible generation.
- `go test`, race detection, vet, formatting and performance envelopes pass.
- ESLint, strict TypeScript, Vitest, Playwright desktop/mobile and production
  build pass.
- No legacy v1 arrays or adapters are exposed.
- No scenario with a hard-rule failure can leave the engine.
- A cold master scan contains exactly two logical provider datasets.
- Timeframe changes preserve scenario IDs, ranking, active path and event IDs.
- Weekly nodes remain ancestors of their visible hourly and minute children.
- Missing lower-degree coverage is `NOT_OBSERVABLE`; it is never synthesized.
- Leaf-only differences cannot create a separate scenario card.
- Refinements produce an immutable child snapshot and reproducible share URL.
- Unobservable breadth, news or sentiment stays `NOT_OBSERVABLE`.
- A target without independent confluence renders as a line, never as a box.
