package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"WaveSight/pkg/model"
	"WaveSight/pkg/swing"
)

type mockFetcher struct {
	fetchFn func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error)
}

func (m *mockFetcher) FetchCandles(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
	if m.fetchFn != nil {
		return m.fetchFn(ctx, ticker, multiplier, timespan, from, to)
	}
	return nil, nil
}

type mockRepository struct {
	getCandlesFn  func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error)
	saveCandlesFn func(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error
}

func (m *mockRepository) GetCandles(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
	if m.getCandlesFn != nil {
		return m.getCandlesFn(ctx, ticker, timeframe, from, to)
	}
	return nil, nil
}

func (m *mockRepository) SaveCandles(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error {
	if m.saveCandlesFn != nil {
		return m.saveCandlesFn(ctx, ticker, timeframe, candles)
	}
	return nil
}

func TestHandler_HandleAnalyze(t *testing.T) {
	testCandles := []model.Candle{
		{Time: 1780272000, Open: 100.0, High: 100.0, Low: 100.0, Close: 100.0, Volume: 1000},
		{Time: 1780275600, Open: 110.0, High: 110.0, Low: 110.0, Close: 110.0, Volume: 1000}, // up 10%
		{Time: 1780279200, Open: 99.0, High: 99.0, Low: 99.0, Close: 99.0, Volume: 1000},    // down 10%
		{Time: 1780286400, Open: 108.9, High: 108.9, Low: 108.9, Close: 108.9, Volume: 1000}, // up 10%
	}

	tests := []struct {
		name           string
		method         string
		url            string
		mockGetCandles func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error)
		mockSave       func(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error
		mockFetch      func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error)
		expectedStatus int
		verifyResponse func(t *testing.T, body []byte)
	}{
		{
			name: "Happy Path - Cached Candles",
			url:  "/api/analyze/AAPL?timeframe=1D&deviation=0.05",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				if ticker != "AAPL" {
					t.Errorf("unexpected ticker: %s", ticker)
				}
				if timeframe == "1D" {
					return testCandles, nil
				}
				return nil, nil
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, body []byte) {
				var resp model.AnalysisResponse
				if err := resp.UnmarshalJSON(body); err != nil {
					t.Fatalf("failed to unmarshal easyjson response: %v", err)
				}
				if resp.Ticker != "AAPL" {
					t.Errorf("expected ticker AAPL, got %s", resp.Ticker)
				}
				if resp.Timeframe != "1D" {
					t.Errorf("expected timeframe 1D, got %s", resp.Timeframe)
				}
				if len(resp.Candles) != 4 {
					t.Errorf("expected 4 candles, got %d", len(resp.Candles))
				}
			},
		},
		{
			name: "Happy Path - Cache Miss & Fetch Success",
			url:  "/api/analyze/MSFT?timeframe=10m&deviation=0.05",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				return nil, nil // cache miss
			},
			mockFetch: func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
				if ticker != "MSFT" || multiplier != 10 || timespan != "minute" {
					t.Errorf("unexpected fetcher args: ticker=%s, multiplier=%d, timespan=%s", ticker, multiplier, timespan)
				}
				return testCandles, nil
			},
			mockSave: func(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error {
				if ticker != "MSFT" || timeframe != "10m" || len(candles) != 4 {
					t.Errorf("unexpected save args: ticker=%s, timeframe=%s, len=%d", ticker, timeframe, len(candles))
				}
				return nil
			},
			expectedStatus: http.StatusOK,
			verifyResponse: func(t *testing.T, body []byte) {
				var resp model.AnalysisResponse
				if err := resp.UnmarshalJSON(body); err != nil {
					t.Fatalf("failed to unmarshal: %v", err)
				}
				if resp.Ticker != "MSFT" {
					t.Errorf("expected MSFT, got %s", resp.Ticker)
				}
			},
		},
		{
			name:           "Error - Missing Ticker Parameter",
			url:            "/api/analyze/",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error - Invalid Timeframe Unit Suffix",
			url:            "/api/analyze/AAPL?timeframe=1X",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error - Invalid Deviation Parameter Value",
			url:            "/api/analyze/AAPL?deviation=-0.05",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Error - Invalid Deviation Parameter Format",
			url:            "/api/analyze/AAPL?deviation=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - Database Query Failure",
			url:  "/api/analyze/AAPL?timeframe=1D",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				return nil, errors.New("db query failed")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Error - Cache Miss and Fetcher Failure",
			url:  "/api/analyze/AAPL?timeframe=1D",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				return nil, nil
			},
			mockFetch: func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
				return nil, errors.New("external api fetcher failure")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Error - Cache Miss, Fetch Success, but Cache Write Failure",
			url:  "/api/analyze/AAPL?timeframe=1D",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				return nil, nil
			},
			mockFetch: func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
				return testCandles, nil
			},
			mockSave: func(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error {
				return errors.New("failed to write to database cache")
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Error - Ticker Not Found (Empty Candles)",
			url:  "/api/analyze/INVALID?timeframe=1D",
			mockGetCandles: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
				return nil, nil // cache miss
			},
			mockFetch: func(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
				return []model.Candle{}, nil // empty candles
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "CORS Preflight (OPTIONS Request)",
			method:         http.MethodOptions,
			url:            "/api/analyze/AAPL",
			expectedStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepository{
				getCandlesFn:  tt.mockGetCandles,
				saveCandlesFn: tt.mockSave,
			}
			fetcher := &mockFetcher{
				fetchFn: tt.mockFetch,
			}
			handler := NewHandler(fetcher, repo, swing.NewVolatilitySwingDetector(14))

			method := tt.method
			if method == "" {
				method = http.MethodGet
			}

			req := httptest.NewRequest(method, tt.url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Verify CORS headers exist on all responses
			if w.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Errorf("expected Access-Control-Allow-Origin to be *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
			}
			if w.Header().Get("Access-Control-Allow-Methods") != "GET, OPTIONS" {
				t.Errorf("expected Access-Control-Allow-Methods to be GET, OPTIONS, got %s", w.Header().Get("Access-Control-Allow-Methods"))
			}
			if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, Authorization" {
				t.Errorf("expected Access-Control-Allow-Headers to be Content-Type, Authorization, got %s", w.Header().Get("Access-Control-Allow-Headers"))
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.verifyResponse != nil && w.Code == http.StatusOK {
				tt.verifyResponse(t, w.Body.Bytes())
			}
		})
	}
}

func BenchmarkAnalyzeHandler(b *testing.B) {
	candles := []model.Candle{
		{Time: 1780272000, Open: 100.0, High: 100.0, Low: 100.0, Close: 100.0, Volume: 1000},
		{Time: 1780275600, Open: 110.0, High: 110.0, Low: 110.0, Close: 110.0, Volume: 1000},
		{Time: 1780279200, Open: 99.0, High: 99.0, Low: 99.0, Close: 99.0, Volume: 1000},
		{Time: 1780286400, Open: 108.9, High: 108.9, Low: 108.9, Close: 108.9, Volume: 1000},
	}

	repo := &mockRepository{
		getCandlesFn: func(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
			return candles, nil
		},
	}
	fetcher := &mockFetcher{}
	handler := NewHandler(fetcher, repo, swing.NewVolatilitySwingDetector(14))

	req := httptest.NewRequest("GET", "/api/analyze/AAPL?timeframe=1D&deviation=0.02", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
