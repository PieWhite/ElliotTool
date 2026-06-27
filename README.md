# WaveSight

WaveSight is an on-demand Elliott Wave analysis system for US stocks and ETFs.
It builds recursive, rule-audited counts from split-adjusted historical bars and
returns one preferred scenario plus up to four genuine alternatives. Rankings
express theory conformance, not a fabricated probability of future price.

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

The public contract is:

- `POST /api/v2/analyses`
- `GET /api/v2/analyses/{analysis_id}`
- `GET /api/v2/analyses?limit=20`

Snapshots are immutable and deduplicated by request, candle fingerprint,
theory version and engine version. The application is technical scenario
analysis, not guaranteed prediction or financial advice.

## Verification

Backend:

```text
go test ./...
go test -race -short ./...
go vet ./...
go test -coverprofile coverage.out ./internal/domain/wave
go run ./cmd/coveragecheck -profile coverage.out -threshold 90
```

Frontend:

```text
npm run lint
npm run typecheck
npm test
npm run test:e2e
npm run build
```

The executable theory matrix is documented in
`.agents/skills/project-roadmap/SKILL.MD`.
