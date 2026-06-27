package api

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"WaveSight/internal/domain/master"
	"WaveSight/internal/market"
	"WaveSight/pkg/polygon"
	"WaveSight/pkg/repository"
)

type fakeFetcher struct {
	daily  []market.Candle
	minute []market.Candle
	err    error
	calls  int
}

func (f *fakeFetcher) FetchCandles(
	_ context.Context, _ string, _ int, timespan string, _, _ string,
) ([]market.Candle, error) {
	f.calls++
	if timespan == "minute" {
		return append([]market.Candle(nil), f.minute...), f.err
	}
	return append([]market.Candle(nil), f.daily...), f.err
}

func (f *fakeFetcher) FetchCandlesDetailed(
	ctx context.Context, ticker string, multiplier int, timespan, from, to string,
) (polygon.FetchResult, error) {
	candles, err := f.FetchCandles(ctx, ticker, multiplier, timespan, from, to)
	return polygon.FetchResult{Candles: candles, PageRequests: 1}, err
}

func apiTestHandler(t *testing.T, candles []market.Candle) (*Handler, *fakeFetcher) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
	store := repository.NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	calendar, err := market.NewUSCalendar()
	if err != nil {
		t.Fatalf("NewUSCalendar() error = %v", err)
	}
	fetcher := &fakeFetcher{daily: candles, minute: validMinuteCandles()}
	handler := NewHandler(fetcher, store, calendar, HandlerConfig{
		AllowedOrigins: []string{"https://wavesight.test"}, MaxConcurrentScans: 2,
	})
	handler.now = func() time.Time {
		return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	}
	return handler, fetcher
}

func validDailyCandles() []market.Candle {
	start := time.Date(2025, 1, 2, 21, 0, 0, 0, time.UTC)
	candles := make([]market.Candle, 400)
	for index := range candles {
		price := 100 + float64(index%17)
		candles[index] = market.Candle{
			Time: start.AddDate(0, 0, index).Unix(),
			Open: price, High: price + 2, Low: price - 2, Close: price + 1, Volume: 1_000,
		}
	}
	return candles
}

func validMinuteCandles() []market.Candle {
	location, _ := time.LoadLocation("America/New_York")
	start := time.Date(2026, 6, 24, 9, 30, 0, 0, location)
	candles := make([]market.Candle, 240)
	for index := range candles {
		price := 100 + float64(index%29)/3
		candles[index] = market.Candle{
			Time: start.Add(time.Duration(index) * time.Minute).Unix(),
			Open: price, High: price + 1.2, Low: price - 1.1,
			Close: price + 0.4, Volume: 100,
		}
	}
	return candles
}

func TestV3AnalysisLifecycleDeduplicationAndLocalViews(t *testing.T) {
	t.Parallel()
	handler, fetcher := apiTestHandler(t, validDailyCandles())
	body := []byte(`{
		"symbol":"aapl","focus_timeframe":"1D","session":"RTH",
		"as_of":"2026-06-27T12:00:00Z",
		"history_profile":"MAX_DAILY_PLUS_2Y_MINUTE","max_scenarios":5
	}`)

	create := httptest.NewRequest(http.MethodPost, "/api/v3/analysis-jobs", bytes.NewReader(body))
	create.Header.Set("Origin", "https://wavesight.test")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d, body = %s", created.Code, created.Body.String())
	}
	if created.Header().Get("Access-Control-Allow-Origin") != "https://wavesight.test" {
		t.Fatalf("allowed CORS origin was not returned")
	}
	var job master.AnalysisJob
	if err := job.UnmarshalJSON(created.Body.Bytes()); err != nil {
		t.Fatalf("AnalysisJob.UnmarshalJSON() error = %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for job.Status != master.JobCompleted && job.Status != master.JobFailed && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		payload, err := handler.v3Store.GetJob(context.Background(), job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := job.UnmarshalJSON(payload); err != nil {
			t.Fatal(err)
		}
	}
	if job.Status != master.JobCompleted {
		t.Fatalf("job did not complete: %+v", job)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v3/analyses/"+job.SnapshotID, nil)
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, get)
	if got.Code != http.StatusOK {
		t.Fatalf("GET status/body = %d/%s", got.Code, got.Body.String())
	}
	var snapshot master.AnalysisSnapshot
	if err := snapshot.UnmarshalJSON(got.Body.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.ID) != 32 || snapshot.Request.Symbol != "AAPL" ||
		len(snapshot.ViewManifest) != 7 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}

	for _, timeframe := range []string{"1W", "1h", "1m"} {
		viewResponse := httptest.NewRecorder()
		handler.ServeHTTP(
			viewResponse,
			httptest.NewRequest(http.MethodGet, "/api/v3/analyses/"+snapshot.ID+"/views/"+timeframe, nil),
		)
		if viewResponse.Code != http.StatusOK {
			t.Fatalf("%s view = %d/%s", timeframe, viewResponse.Code, viewResponse.Body.String())
		}
	}
	duplicate := httptest.NewRequest(http.MethodPost, "/api/v3/analysis-jobs", bytes.NewReader(body))
	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, duplicate)
	if duplicateResponse.Code != http.StatusOK {
		t.Fatalf("deduplicated POST status = %d, body = %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	if fetcher.calls != 2 {
		t.Fatalf("provider calls = %d, want exactly two logical datasets", fetcher.calls)
	}

	history := httptest.NewRequest(http.MethodGet, "/api/v3/analyses?limit=20", nil)
	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, history)
	var listed SnapshotHistoryV3
	if err := listed.UnmarshalJSON(historyResponse.Body.Bytes()); err != nil {
		t.Fatalf("SnapshotHistoryV3.UnmarshalJSON() error = %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != snapshot.ID {
		t.Fatalf("history = %+v", listed.Items)
	}
}

func TestV3AnalysisValidationAndProblems(t *testing.T) {
	t.Parallel()
	handler, _ := apiTestHandler(t, validDailyCandles())
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "invalid JSON", body: `{`, status: http.StatusBadRequest},
		{name: "invalid symbol", body: `{"symbol":"$","focus_timeframe":"1D"}`, status: http.StatusBadRequest},
		{name: "invalid timeframe", body: `{"symbol":"AAPL","focus_timeframe":"10m"}`, status: http.StatusBadRequest},
		{name: "too many scenarios", body: `{"symbol":"AAPL","focus_timeframe":"1D","max_scenarios":6}`, status: http.StatusBadRequest},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v3/analysis-jobs", bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "application/problem+json" {
				t.Fatalf("problem Content-Type = %q", response.Header().Get("Content-Type"))
			}
			var problem Problem
			if err := problem.UnmarshalJSON(response.Body.Bytes()); err != nil {
				t.Fatalf("Problem.UnmarshalJSON() error = %v", err)
			}
			if problem.RequestID == "" || problem.Status != test.status {
				t.Fatalf("problem = %+v", problem)
			}
		})
	}
}

func TestV2AnalysisCreationIsReadOnly(t *testing.T) {
	t.Parallel()
	handler, _ := apiTestHandler(t, validDailyCandles()[:10])
	request := httptest.NewRequest(
		http.MethodPost, "/api/v2/analyses",
		bytes.NewBufferString(`{"symbol":"AAPL","timeframe":"1D","lookback_bars":200}`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusGone {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestDailyProvenanceAuditExposesNativeDerivedDifferences(t *testing.T) {
	t.Parallel()
	native := []market.DerivedCandle{{
		Candle:     market.Candle{Time: 1_782_432_000, Open: 100, High: 105, Low: 99, Close: 104, Volume: 1_000},
		Provenance: market.ProvenanceNativeDaily,
	}}
	derived := []market.DerivedCandle{{
		Candle:     market.Candle{Time: 1_782_466_200, Open: 100, High: 104, Low: 99, Close: 103, Volume: 900},
		Provenance: market.ProvenanceMinuteDerived,
	}}
	audit := compareDailyProvenance(native, derived)
	if audit.Compared != 1 || audit.Differences != 1 ||
		audit.MaxOHLCDeviation != 1 || len(audit.Samples) != 1 {
		t.Fatalf("daily provenance audit = %+v", audit)
	}
}

func TestNativeCoverageSelectsOnlyMissingTailOrRefinementGap(t *testing.T) {
	t.Parallel()
	coverage := []repository.CoverageRange{
		{From: 100, To: 200},
		{From: 300, To: 400},
	}
	from, needed := missingTail(
		coverage, time.Unix(100, 0).UTC(), time.Unix(500, 0).UTC(),
	)
	if !needed || from.Unix() != 400 {
		t.Fatalf("missing tail = %s/%t", from, needed)
	}
	gap, needed := firstMissingStart(
		coverage, time.Unix(150, 0).UTC(), time.Unix(350, 0).UTC(),
	)
	if !needed || gap.Unix() != 201 {
		t.Fatalf("refinement gap = %s/%t", gap, needed)
	}
	_, needed = missingTail(
		[]repository.CoverageRange{{From: 50, To: 600}},
		time.Unix(100, 0).UTC(), time.Unix(500, 0).UTC(),
	)
	if needed {
		t.Fatal("fully covered native range requested another provider query")
	}
}

func TestConcurrentSessionScansShareOneNativeFetch(t *testing.T) {
	t.Parallel()
	handler, fetcher := apiTestHandler(t, validDailyCandles())
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var wait sync.WaitGroup
	errors := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, err := handler.acquireNative(
				context.Background(), "AAPL", master.NativeDaily, from, to, false,
			)
			errors <- err
		}()
	}
	wait.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if fetcher.calls != 1 {
		t.Fatalf("concurrent session provider calls = %d, want one shared native fetch", fetcher.calls)
	}
}

func TestSPAHandlerServesAssetsAndShareRouteFallback(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("<main>WaveSight</main>"), 0o600); err != nil {
		t.Fatalf("writing index fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "app.js"), []byte("console.log('wavesight')"), 0o600); err != nil {
		t.Fatalf("writing asset fixture: %v", err)
	}
	handler := newSPAHandler(directory)
	for _, requestPath := range []string{"/", "/analysis/0123456789abcdef0123456789abcdef"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if response.Code != http.StatusOK || response.Body.String() != "<main>WaveSight</main>" {
			t.Fatalf("GET %s = %d/%q", requestPath, response.Code, response.Body.String())
		}
	}
	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if asset.Code != http.StatusOK || asset.Body.String() != "console.log('wavesight')" {
		t.Fatalf("asset = %d/%q", asset.Code, asset.Body.String())
	}
	method := httptest.NewRecorder()
	handler.ServeHTTP(method, httptest.NewRequest(http.MethodPost, "/", nil))
	if method.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST static status = %d", method.Code)
	}
}
