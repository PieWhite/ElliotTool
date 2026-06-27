package polygon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"WaveSight/internal/market"
)

const defaultBaseURL = "https://api.massive.com"

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	apiKey     string
	httpClient HTTPClient
	baseURL    string
}

func NewClient(apiKey string, httpClient HTTPClient) *Client {
	return &Client{apiKey: apiKey, httpClient: httpClient, baseURL: defaultBaseURL}
}

func (c *Client) SetBaseURL(value string) {
	c.baseURL = strings.TrimRight(value, "/")
}

//easyjson:json
type pageResponse struct {
	Status       string       `json:"status"`
	Results      []pageResult `json:"results"`
	ResultsCount int          `json:"resultsCount"`
	Error        string       `json:"error"`
	NextURL      string       `json:"next_url"`
}

//easyjson:json
type pageResult struct {
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume float64 `json:"v"`
	Time   int64   `json:"t"`
}

type RateLimitError struct {
	Message string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

type FetchResult struct {
	Candles      []market.Candle
	PageRequests int
}

func (c *Client) FetchCandles(ctx context.Context, ticker string, multiplier int, timespan, from, to string) ([]market.Candle, error) {
	result, err := c.FetchCandlesDetailed(ctx, ticker, multiplier, timespan, from, to)
	return result.Candles, err
}

func (c *Client) FetchCandlesDetailed(ctx context.Context, ticker string, multiplier int, timespan, from, to string) (FetchResult, error) {
	if c.httpClient == nil {
		return FetchResult{}, fmt.Errorf("fetching Massive candles: nil HTTP client")
	}
	if c.apiKey == "" {
		return FetchResult{}, fmt.Errorf("fetching Massive candles: empty API key")
	}

	nextURL := fmt.Sprintf(
		"%s/v2/aggs/ticker/%s/range/%d/%s/%s/%s?adjusted=true&sort=asc&limit=50000",
		c.baseURL,
		url.PathEscape(ticker),
		multiplier,
		url.PathEscape(timespan),
		url.PathEscape(from),
		url.PathEscape(to),
	)
	candles := make([]market.Candle, 0, 8_192)
	seenPages := make(map[string]struct{}, 8)
	pageRequests := 0

	for nextURL != "" {
		if _, exists := seenPages[nextURL]; exists {
			return FetchResult{}, fmt.Errorf("fetching Massive candles: pagination loop detected")
		}
		seenPages[nextURL] = struct{}{}

		requestURL, err := c.withAPIKey(nextURL)
		if err != nil {
			return FetchResult{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return FetchResult{}, fmt.Errorf("creating Massive request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return FetchResult{}, fmt.Errorf("executing Massive request: %w", err)
		}
		pageRequests++
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		closeErr := resp.Body.Close()
		if readErr != nil {
			return FetchResult{}, fmt.Errorf("reading Massive response: %w", readErr)
		}
		if closeErr != nil {
			return FetchResult{}, fmt.Errorf("closing Massive response: %w", closeErr)
		}

		var page pageResponse
		if err := page.UnmarshalJSON(body); err != nil {
			return FetchResult{}, fmt.Errorf("decoding Massive response: %w", err)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			return FetchResult{}, &RateLimitError{Message: nonEmpty(page.Error, "Massive rate limit reached")}
		}
		if resp.StatusCode != http.StatusOK {
			return FetchResult{}, fmt.Errorf("Massive status %d: %s", resp.StatusCode, nonEmpty(page.Error, string(body)))
		}
		if page.Status != "OK" && page.Status != "DELAYED" {
			return FetchResult{}, fmt.Errorf("Massive API status %q: %s", page.Status, page.Error)
		}

		for _, result := range page.Results {
			candles = append(candles, market.Candle{
				Time: result.Time / 1_000, Open: result.Open, High: result.High,
				Low: result.Low, Close: result.Close, Volume: result.Volume,
			})
		}
		nextURL = page.NextURL
	}
	return FetchResult{Candles: candles, PageRequests: pageRequests}, nil
}

func (c *Client) withAPIKey(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing Massive pagination URL: %w", err)
	}
	query := parsed.Query()
	query.Set("apiKey", c.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
