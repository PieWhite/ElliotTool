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
	"testing"
	"time"

	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
	"WaveSight/pkg/repository"
)

type fakeFetcher struct {
	candles []market.Candle
	err     error
	calls   int
}

func (f *fakeFetcher) FetchCandles(
	_ context.Context, _ string, _ int, _ string, _, _ string,
) ([]market.Candle, error) {
	f.calls++
	return append([]market.Candle(nil), f.candles...), f.err
}

type fakeAnalyzer struct {
	calls int
}

func (a *fakeAnalyzer) Analyze(input wave.AnalyzeInput) wave.AnalysisResult {
	a.calls++
	return wave.AnalysisResult{
		DataQuality: wave.DataQuality{
			CandleCount: len(input.Candles),
			FirstTime:   input.Candles[0].Time,
			LastTime:    input.Candles[len(input.Candles)-1].Time,
		},
		Scenarios: []wave.Scenario{{
			ID: "scenario-1", Rank: 1, Status: wave.ScenarioPreferred,
			Bias: wave.DirectionBullish, CurrentPosition: "Intermediate (4)",
			Root: wave.WaveNode{
				ID: "root", Label: "Developing impulse", Status: wave.StatusDeveloping,
			},
		}},
		FutureBars: append([]int64(nil), input.FutureBars...),
	}
}

func apiTestHandler(t *testing.T, candles []market.Candle) (*Handler, *fakeFetcher, *fakeAnalyzer) {
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
	fetcher := &fakeFetcher{candles: candles}
	analyzer := &fakeAnalyzer{}
	handler := NewHandler(fetcher, store, analyzer, calendar, HandlerConfig{
		AllowedOrigins: []string{"https://wavesight.test"}, MaxConcurrentScans: 2,
	})
	handler.now = func() time.Time {
		return time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	}
	return handler, fetcher, analyzer
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

func TestV2AnalysisLifecycleAndDeduplication(t *testing.T) {
	t.Parallel()
	handler, fetcher, analyzer := apiTestHandler(t, validDailyCandles())
	body := []byte(`{
		"symbol":"aapl","timeframe":"1D","session":"RTH",
		"as_of":"2026-06-27T00:00:00Z","lookback_bars":200,"max_scenarios":5
	}`)

	create := httptest.NewRequest(http.MethodPost, "/api/v2/analyses", bytes.NewReader(body))
	create.Header.Set("Origin", "https://wavesight.test")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body = %s", created.Code, created.Body.String())
	}
	if created.Header().Get("Access-Control-Allow-Origin") != "https://wavesight.test" {
		t.Fatalf("allowed CORS origin was not returned")
	}
	var snapshot AnalysisSnapshot
	if err := snapshot.UnmarshalJSON(created.Body.Bytes()); err != nil {
		t.Fatalf("AnalysisSnapshot.UnmarshalJSON() error = %v", err)
	}
	if len(snapshot.ID) != 32 || snapshot.Request.Symbol != "AAPL" ||
		len(snapshot.Candles) != 200 || len(snapshot.Scenarios) != 1 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v2/analyses/"+snapshot.ID, nil)
	got := httptest.NewRecorder()
	handler.ServeHTTP(got, get)
	if got.Code != http.StatusOK || got.Body.String() != created.Body.String() {
		t.Fatalf("GET status/body = %d/%s", got.Code, got.Body.String())
	}

	duplicate := httptest.NewRequest(http.MethodPost, "/api/v2/analyses", bytes.NewReader(body))
	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, duplicate)
	if duplicateResponse.Code != http.StatusOK {
		t.Fatalf("deduplicated POST status = %d, body = %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	if fetcher.calls != 1 || analyzer.calls != 1 {
		t.Fatalf("fetcher/analyzer calls = %d/%d, want 1/1", fetcher.calls, analyzer.calls)
	}

	history := httptest.NewRequest(http.MethodGet, "/api/v2/analyses?limit=20", nil)
	historyResponse := httptest.NewRecorder()
	handler.ServeHTTP(historyResponse, history)
	var listed SnapshotHistory
	if err := listed.UnmarshalJSON(historyResponse.Body.Bytes()); err != nil {
		t.Fatalf("SnapshotHistory.UnmarshalJSON() error = %v", err)
	}
	if len(listed.Items) != 1 || listed.Items[0].ID != snapshot.ID {
		t.Fatalf("history = %+v", listed.Items)
	}
}

func TestV2AnalysisValidationAndProblems(t *testing.T) {
	t.Parallel()
	handler, _, _ := apiTestHandler(t, validDailyCandles())
	tests := []struct {
		name   string
		body   string
		status int
	}{
		{name: "invalid JSON", body: `{`, status: http.StatusBadRequest},
		{name: "invalid symbol", body: `{"symbol":"$","timeframe":"1D"}`, status: http.StatusBadRequest},
		{name: "invalid timeframe", body: `{"symbol":"AAPL","timeframe":"10m"}`, status: http.StatusBadRequest},
		{name: "too many scenarios", body: `{"symbol":"AAPL","timeframe":"1D","max_scenarios":6}`, status: http.StatusBadRequest},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v2/analyses", bytes.NewBufferString(test.body))
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

func TestV2AnalysisRejectsInsufficientData(t *testing.T) {
	t.Parallel()
	handler, _, analyzer := apiTestHandler(t, validDailyCandles()[:10])
	request := httptest.NewRequest(
		http.MethodPost, "/api/v2/analyses",
		bytes.NewBufferString(`{"symbol":"AAPL","timeframe":"1D","lookback_bars":200}`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || analyzer.calls != 0 {
		t.Fatalf("status/analyzer calls = %d/%d, body = %s", response.Code, analyzer.calls, response.Body.String())
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
