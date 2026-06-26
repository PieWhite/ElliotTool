package polygon

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"WaveSight/pkg/model"
)

// MockHTTPClient implements HTTPClient for unit testing.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("DoFunc not implemented")
}

func TestFetchCandles_Success(t *testing.T) {
	apiKey := "test_api_key"
	ticker := "AAPL"
	multiplier := 5
	timespan := "minute"
	from := "2026-06-01"
	to := "2026-06-02"

	mockResponseJSON := `{
		"status": "OK",
		"ticker": "AAPL",
		"queryCount": 2,
		"resultsCount": 2,
		"adjusted": true,
		"results": [
			{
				"o": 150.5,
				"h": 151.2,
				"l": 149.8,
				"c": 150.9,
				"v": 50000.0,
				"t": 1780272000000
			},
			{
				"o": 150.9,
				"h": 152.0,
				"l": 150.5,
				"c": 151.8,
				"v": 60000.0,
				"t": 1780275600000
			}
		]
	}`

	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			// Verify URL structure
			expectedPath := "/v2/aggs/ticker/AAPL/range/5/minute/2026-06-01/2026-06-02"
			if !strings.Contains(req.URL.Path, expectedPath) {
				t.Errorf("expected path to contain %q, got %q", expectedPath, req.URL.Path)
			}

			// Verify query parameters
			q := req.URL.Query()
			if q.Get("apiKey") != apiKey {
				t.Errorf("expected apiKey %q, got %q", apiKey, q.Get("apiKey"))
			}
			if q.Get("adjusted") != "true" {
				t.Errorf("expected adjusted to be true, got %q", q.Get("adjusted"))
			}
			if q.Get("sort") != "asc" {
				t.Errorf("expected sort to be asc, got %q", q.Get("sort"))
			}
			if q.Get("limit") != "50000" {
				t.Errorf("expected limit to be 50000, got %q", q.Get("limit"))
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseJSON)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := NewClient(apiKey, mockClient)
	candles, err := client.FetchCandles(context.Background(), ticker, multiplier, timespan, from, to)
	if err != nil {
		t.Fatalf("FetchCandles failed unexpectedly: %v", err)
	}

	if len(candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(candles))
	}

	// Verify timestamps are converted to seconds
	expectedCandles := []model.Candle{
		{Time: 1780272000, Open: 150.5, High: 151.2, Low: 149.8, Close: 150.9, Volume: 50000.0},
		{Time: 1780275600, Open: 150.9, High: 152.0, Low: 150.5, Close: 151.8, Volume: 60000.0},
	}

	for i, expected := range expectedCandles {
		actual := candles[i]
		if actual.Time != expected.Time {
			t.Errorf("candle[%d]: expected Time %d, got %d", i, expected.Time, actual.Time)
		}
		if actual.Open != expected.Open {
			t.Errorf("candle[%d]: expected Open %f, got %f", i, expected.Open, actual.Open)
		}
		if actual.High != expected.High {
			t.Errorf("candle[%d]: expected High %f, got %f", i, expected.High, actual.High)
		}
		if actual.Low != expected.Low {
			t.Errorf("candle[%d]: expected Low %f, got %f", i, expected.Low, actual.Low)
		}
		if actual.Close != expected.Close {
			t.Errorf("candle[%d]: expected Close %f, got %f", i, expected.Close, actual.Close)
		}
		if actual.Volume != expected.Volume {
			t.Errorf("candle[%d]: expected Volume %f, got %f", i, expected.Volume, actual.Volume)
		}
	}
}

func TestFetchCandles_HttpError(t *testing.T) {
	mockErr := errors.New("network timeout")
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return nil, mockErr
		},
	}

	client := NewClient("key", mockClient)
	_, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-06-01", "2026-06-02")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, mockErr) && !strings.Contains(err.Error(), mockErr.Error()) {
		t.Errorf("expected error containing %q, got %v", mockErr.Error(), err)
	}
}

func TestFetchCandles_Non200Status(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ERROR","error":"Invalid API Key"}`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := NewClient("invalid_key", mockClient)
	_, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-06-01", "2026-06-02")

	if err == nil {
		t.Fatal("expected error for HTTP 401, got nil")
	}
	if !strings.Contains(err.Error(), "status code 401") && !strings.Contains(err.Error(), "Invalid API Key") {
		t.Errorf("expected error details, got %v", err)
	}
}

func TestFetchCandles_APIErrorStatus(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ERROR","error":"Some internal Polygon error"}`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := NewClient("key", mockClient)
	_, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-06-01", "2026-06-02")

	if err == nil {
		t.Fatal("expected error for JSON status ERROR, got nil")
	}
	if !strings.Contains(err.Error(), "Some internal Polygon error") {
		t.Errorf("expected error containing Polygon error message, got %v", err)
	}
}

func TestFetchCandles_InvalidJSON(t *testing.T) {
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{invalid json`)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := NewClient("key", mockClient)
	_, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-06-01", "2026-06-02")

	if err == nil {
		t.Fatal("expected JSON parsing error, got nil")
	}
}

func TestFetchCandles_EmptyResults(t *testing.T) {
	mockResponseJSON := `{"status":"OK","ticker":"AAPL","queryCount":0,"resultsCount":0,"adjusted":true}`
	mockClient := &MockHTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResponseJSON)),
				Header:     make(http.Header),
			}, nil
		},
	}

	client := NewClient("key", mockClient)
	candles, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-06-01", "2026-06-02")

	if err != nil {
		t.Fatalf("expected no error for empty results, got: %v", err)
	}
	if len(candles) != 0 {
		t.Errorf("expected 0 candles, got %d", len(candles))
	}
}
