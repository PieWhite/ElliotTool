# WaveSight

WaveSight is an on-demand Elliott Wave analysis system for US stocks and ETFs.
It builds one recursive, rule-audited `MasterWaveTree` from maximum daily
history plus two years of minute detail. The seven chart timeframes are
projections of that same tree: zooming changes visible detail, never the
scenario, ranking or canonical wave-event IDs.

The result is one preferred scenario plus up to four materially different
alternatives. Rankings express theory conformance, not a fabricated probability
of future price.

## Run locally

Requirements: Go, Node.js and a Massive market-data key.

```text
POLYGON_API_KEY=...
DATABASE_PATH=wavesight.db
ALLOWED_ORIGINS=http://localhost:5173
```

Start the Go service with `go run .`. In `frontend`, install dependencies with
`npm ci` and start the interface with `npm run dev`.

For a single hosted image, build `Dockerfile`; the production service serves the
compiled Vue application and the API from the same origin.

The current public contract is:

- `POST /api/v3/analysis-jobs`
- `GET /api/v3/analysis-jobs/{job_id}`
- `GET /api/v3/analyses/{snapshot_id}`
- `GET /api/v3/analyses/{snapshot_id}/views/{timeframe}`
- `POST /api/v3/analyses/{snapshot_id}/refinements`
- `GET /api/v3/analyses?limit=20`

V2 is read-only:

- `GET /api/v2/analyses/{analysis_id}`
- `GET /api/v2/analyses?limit=20`

A cold scan makes two logical Massive queries—native daily and native minute—
and follows their pagination. All 1m/5m/15m/1h/4h/1D/1W views are then built
locally and cached in SQLite. RTH and extended counts share those native bars.
Historical refinements create immutable child revisions; an existing snapshot
is never rewritten.

Snapshots are deduplicated by master request, native-data fingerprint, theory
version and engine version. The application is technical scenario analysis,
not guaranteed prediction or financial advice.

## Verification

Backend:

```text
go test ./...
go test -race -short ./...
go vet ./...
go test -coverprofile coverage.out ./internal/domain/wave
go run ./cmd/coveragecheck -profile coverage.out -threshold 90
WAVESIGHT_MASTER_PERF_TEST=1 go test -run TestMasterEnginePerformanceEnvelope ./internal/domain/master
```

Frontend:

```text
npm run lint
npm run typecheck
npm test
npm run test:e2e
npm run build
```

The executable theory and multi-timeframe matrix is documented in
`.agents/skills/project-roadmap/SKILL.MD`.
