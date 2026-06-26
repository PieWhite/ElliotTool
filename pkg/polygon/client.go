package polygon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"WaveSight/pkg/model"
)

// HTTPClient defines the interface for making HTTP requests.
// It allows mocking the network layer in unit tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the Polygon.io API client.
type Client struct {
	apiKey     string
	httpClient HTTPClient
	baseURL    string
}

// NewClient creates a new Polygon API client.
func NewClient(apiKey string, httpClient HTTPClient) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    "https://api.massive.com",
	}
}

// SetBaseURL allows overriding the base URL (useful for testing).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// polygonResponse represents the aggregate endpoint response from Polygon.io.
type polygonResponse struct {
	Status       string          `json:"status"`
	Results      []polygonResult `json:"results"`
	ResultsCount int             `json:"resultsCount"`
	Error        string          `json:"error"`
}

// polygonResult represents a single bar/candle in Polygon.io response.
type polygonResult struct {
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume float64 `json:"v"`
	Time   int64   `json:"t"` // Milliseconds since epoch
}

// FetchCandles fetches historical OHLCV data from the Polygon.io aggregates endpoint.
func (c *Client) FetchCandles(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error) {
	// Construct the URL path
	u := fmt.Sprintf("%s/v2/aggs/ticker/%s/range/%d/%s/%s/%s?adjusted=true&sort=asc&limit=50000&apiKey=%s",
		c.baseURL,
		url.PathEscape(ticker),
		multiplier,
		url.PathEscape(timespan),
		url.PathEscape(from),
		url.PathEscape(to),
		url.QueryEscape(c.apiKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		var apiErr polygonResponse
		if json.Unmarshal(bodyBytes, &apiErr) == nil && apiErr.Error != "" {
			return nil, fmt.Errorf("polygon API error (status code %d): %s", resp.StatusCode, apiErr.Error)
		}
		return nil, fmt.Errorf("polygon API returned non-200 status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response polygonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decoding json response: %w", err)
	}

	if response.Status != "OK" && response.Status != "DELAYED" {
		return nil, fmt.Errorf("polygon API status error: %s (error: %s)", response.Status, response.Error)
	}

	candles := make([]model.Candle, 0, len(response.Results))
	for _, res := range response.Results {
		candles = append(candles, model.Candle{
			Time:   res.Time / 1000, // convert millisecond timestamp to seconds
			Open:   res.Open,
			High:   res.High,
			Low:    res.Low,
			Close:  res.Close,
			Volume: res.Volume,
		})
	}

	return candles, nil
}
