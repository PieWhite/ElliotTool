package polygon

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return m.do(request)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestFetchCandlesFollowsPagination(t *testing.T) {
	t.Parallel()
	requests := 0
	client := NewClient("secret", &mockHTTPClient{do: func(request *http.Request) (*http.Response, error) {
		requests++
		if request.URL.Query().Get("apiKey") != "secret" {
			t.Fatalf("pagination request did not preserve the API key")
		}
		if requests == 1 {
			if request.URL.Query().Get("adjusted") != "true" ||
				request.URL.Query().Get("sort") != "asc" ||
				request.URL.Query().Get("limit") != "50000" {
				t.Fatalf("unexpected aggregate query: %s", request.URL.RawQuery)
			}
			return response(http.StatusOK, `{
				"status":"OK",
				"results":[{"o":100,"h":102,"l":99,"c":101,"v":1000,"t":1780272000000}],
				"next_url":"https://api.massive.test/page/2"
			}`), nil
		}
		if request.URL.Host != "api.massive.test" {
			t.Fatalf("unexpected next_url host: %s", request.URL.Host)
		}
		return response(http.StatusOK, `{
			"status":"OK",
			"results":[{"o":101,"h":103,"l":100,"c":102,"v":1200,"t":1780275600000}]
		}`), nil
	}})
	client.SetBaseURL("https://api.massive.test")

	candles, err := client.FetchCandles(
		context.Background(), "AAPL", 5, "minute", "2026-06-01", "2026-06-02",
	)
	if err != nil {
		t.Fatalf("FetchCandles() error = %v", err)
	}
	if requests != 2 || len(candles) != 2 {
		t.Fatalf("got %d requests and %d candles, want 2 and 2", requests, len(candles))
	}
	if candles[0].Time != 1780272000 || candles[1].Close != 102 {
		t.Fatalf("unexpected normalized candles: %+v", candles)
	}
}

func TestFetchCandlesErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		client    HTTPClient
		wantText  string
		rateLimit bool
		cancelled bool
	}{
		{
			name: "transport",
			client: &mockHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("connection reset")
			}},
			wantText: "connection reset",
		},
		{
			name: "unauthorized",
			client: &mockHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return response(http.StatusUnauthorized, `{"status":"ERROR","error":"not entitled"}`), nil
			}},
			wantText: "not entitled",
		},
		{
			name: "provider status",
			client: &mockHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return response(http.StatusOK, `{"status":"ERROR","error":"bad aggregate request"}`), nil
			}},
			wantText: "bad aggregate request",
		},
		{
			name: "invalid JSON",
			client: &mockHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return response(http.StatusOK, `{invalid`), nil
			}},
			wantText: "decoding Massive response",
		},
		{
			name: "rate limit",
			client: &mockHTTPClient{do: func(*http.Request) (*http.Response, error) {
				return response(http.StatusTooManyRequests, `{"status":"ERROR","error":"slow down"}`), nil
			}},
			wantText:  "slow down",
			rateLimit: true,
		},
		{
			name: "context cancellation",
			client: &mockHTTPClient{do: func(request *http.Request) (*http.Response, error) {
				<-request.Context().Done()
				return nil, request.Context().Err()
			}},
			wantText:  "context canceled",
			cancelled: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			if test.cancelled {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			client := NewClient("key", test.client)
			_, err := client.FetchCandles(ctx, "AAPL", 1, "day", "2026-01-01", "2026-02-01")
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("FetchCandles() error = %v, want text %q", err, test.wantText)
			}
			var typed *RateLimitError
			if test.rateLimit != errors.As(err, &typed) {
				t.Fatalf("RateLimitError = %t, want %t", errors.As(err, &typed), test.rateLimit)
			}
		})
	}
}

func TestFetchCandlesRejectsPaginationLoop(t *testing.T) {
	t.Parallel()
	client := NewClient("key", &mockHTTPClient{do: func(request *http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"status":"OK","next_url":"`+request.URL.String()+`"}`), nil
	}})
	_, err := client.FetchCandles(context.Background(), "AAPL", 1, "day", "2026-01-01", "2026-02-01")
	if err == nil || !strings.Contains(err.Error(), "pagination loop") {
		t.Fatalf("FetchCandles() error = %v, want pagination loop", err)
	}
}
