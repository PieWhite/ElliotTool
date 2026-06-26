# Description
Provides strict guidelines for writing Golang backend code, focusing on high-performance concurrency, Test-Driven Development (TDD), and idiomatic "Effective Go" standards for high-throughput applications.

# Instructions

## 1. High-Performance Data Handling
- **JSON Serialization:** DO NOT use the standard `encoding/json` package for large payloads or hot paths due to its reflection overhead. Exclusively use `mailru/easyjson` (or equivalent code-generated parsers) for structs like `Candle` and API responses to maximize throughput and minimize allocations.
- **Memory Efficiency:** Preallocate slices and maps when the size is known: `make([]Candle, 0, 50000)`. Use value receivers for small structs, and pointers for large payloads to avoid heap allocations and memory copying.
- **Strings:** Use `strings.Builder` for string concatenation in loops. Use `[]byte` over `string` where possible when parsing raw network data to avoid allocation overhead.

## 2. Idiomatic "Effective Go" & Architecture
- **Visibility Naming:** Use `PascalCase` for exported (public) structs, fields, functions, and interfaces. Use `camelCase` ONLY for unexported (private) variables. Avoid package name stuttering (e.g., use `polygon.Client`, not `polygon.PolygonClient`).
- **Interfaces & DIP:** Accept interfaces, return structs. Keep interfaces small (1-3 methods, e.g., `Fetcher`, `Parser`) to decouple services and easily mock external API calls or database repositories.
- **Error Handling:** Never discard errors. Return them as the last value and check immediately. Wrap errors with context (`fmt.Errorf("fetching polygon data: %w", err)`). Use `errors.Is()` and `errors.As()` for inspection. Avoid `panic` in production.
- **Context:** Always pass `context.Context` as the FIRST parameter in functions that perform I/O, API calls, or may be cancelled.

## 3. Concurrency
- **Goroutines:** Leverage Goroutines safely for parallel processing of data streams. Use `sync.Mutex` or `sync.RWMutex` for simple shared state and channels for coordination. 
- **Leak Prevention:** Prevent Goroutine leaks by ensuring explicit exit paths via context cancellation or `done` channels.

## 4. Linting & Formatting
- **Standardization:** Code MUST conform to `gofmt` / `goimports`. 
- **GolangCI-Lint:** Ensure the code passes `golangci-lint` with strict settings (including `errcheck`, `govet`, `staticcheck`, and `gosimple`). Fix all warnings proactively.

## 5. Active TDD & Testing
- **Active TDD:** Test-Driven Development is mandatory. Write the Unit Tests *first* or alongside the implementation.
- **Test Libraries:** Avoid heavy third-party assertion libraries unless specifically authorized. Use standard library `if expected != actual` checks and `t.Errorf()` to keep the test binary fast.
- **Table-Driven Tests:** Use table-driven tests (`[]struct`) for logic with multiple input/output combinations (especially mathematical algorithms).
- **Integration Tests:** For SQLite operations, test against an in-memory DB (`:memory:`), verifying schema, writes, and reads realistically.