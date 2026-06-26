package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"WaveSight/pkg/elliott"
	"WaveSight/pkg/model"
	"WaveSight/pkg/repository"
	"WaveSight/pkg/zigzag"
)

// CandleFetcher defines the interface for fetching market candles from an external API provider.
type CandleFetcher interface {
	FetchCandles(ctx context.Context, ticker string, multiplier int, timespan string, from string, to string) ([]model.Candle, error)
}

// Handler coordinates request routing, cache lookups, external API fetches, and Elliott Wave scanners.
type Handler struct {
	fetcher CandleFetcher
	repo    repository.CandleRepository
	router  *http.ServeMux
}

// NewHandler initializes a new API Handler with ServeMux routing.
func NewHandler(fetcher CandleFetcher, repo repository.CandleRepository) *Handler {
	h := &Handler{
		fetcher: fetcher,
		repo:    repo,
		router:  http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

// ServeHTTP satisfies the http.Handler interface by routing HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	h.router.ServeHTTP(w, r)
}

func (h *Handler) registerRoutes() {
	h.router.HandleFunc("GET /api/analyze/", h.handleAnalyzeMissing)
	h.router.HandleFunc("GET /api/analyze/{ticker}", h.handleAnalyze)
}

func (h *Handler) handleAnalyzeMissing(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "missing ticker parameter", http.StatusBadRequest)
}

func (h *Handler) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	ticker := r.PathValue("ticker")
	if ticker == "" {
		http.Error(w, "missing ticker parameter", http.StatusBadRequest)
		return
	}

	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1D"
	}

	deviationStr := r.URL.Query().Get("deviation")
	deviation := 0.02 // default ZigZag threshold
	if deviationStr != "" {
		var err error
		deviation, err = strconv.ParseFloat(deviationStr, 64)
		if err != nil || deviation <= 0 {
			http.Error(w, "invalid deviation parameter", http.StatusBadRequest)
			return
		}
	}

	// Convert deviation ratio (e.g. 0.02) to percentage (e.g. 2.0) if <= 1.0
	var percentDeviation float64
	if deviation <= 1.0 {
		percentDeviation = deviation * 100.0
	} else {
		percentDeviation = deviation
	}

	multiplier, timespan, fromDateStr, toDateStr, err := parseTimeframe(timeframe)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid timeframe: %v", err), http.StatusBadRequest)
		return
	}

	// Try loading from SQLite cache first
	fromTime, _ := time.Parse("2006-01-02", fromDateStr)
	toTime, _ := time.Parse("2006-01-02", toDateStr)

	var candles []model.Candle
	var childCandles []model.Candle
	var errParent error

	childTimeframe := getChildTimeframe(timeframe)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		candles, errParent = h.getOrFetchCandles(r.Context(), ticker, timeframe, multiplier, timespan, fromDateStr, toDateStr, fromTime, toTime)
	}()

	if childTimeframe != "" {
		childMultiplier, childTimespan, _, _, parseErr := parseTimeframe(childTimeframe)
		if parseErr == nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				childCandles, _ = h.getOrFetchCandles(r.Context(), ticker, childTimeframe, childMultiplier, childTimespan, fromDateStr, toDateStr, fromTime, toTime)
			}()
		}
	}

	wg.Wait()

	if errParent != nil {
		http.Error(w, errParent.Error(), http.StatusInternalServerError)
		return
	}
	if len(candles) == 0 {
		http.Error(w, fmt.Sprintf("ticker %s not found or contains no historical data", ticker), http.StatusNotFound)
		return
	}

	// Run calculations and scanning pipeline
	pivots := zigzag.CalculateZigZag(candles, percentDeviation)

	var childPivots []model.Pivot
	if len(childCandles) > 0 {
		childPivots = zigzag.CalculateZigZag(childCandles, percentDeviation)
	}

	// ScenarioBundle ranks all patterns into a primary/alternate pair while also
	// returning the legacy flat slices for backward compatibility.
	motiveWaves, correctiveWaves, incompleteWaves, scenarios := elliott.ScenarioBundle(pivots, childPivots, timeframe)

	// Compile high-performance Response struct
	resp := model.AnalysisResponse{
		Ticker:          ticker,
		Timeframe:       timeframe,
		Candles:         candles,
		Scenarios:       scenarios,
		MotiveWaves:     motiveWaves,
		CorrectiveWaves: correctiveWaves,
		IncompleteWaves: incompleteWaves,
	}

	// Serialize with generated easyjson code
	data, err := resp.MarshalJSON()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to serialize JSON response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// parseTimeframe decodes timeframe string into polygon API parameters and dynamic from/to range.
func parseTimeframe(tf string) (multiplier int, timespan string, from string, to string, err error) {
	if tf == "" {
		return 0, "", "", "", fmt.Errorf("empty timeframe")
	}

	var digits strings.Builder
	var suffix strings.Builder
	for _, r := range tf {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		} else {
			suffix.WriteRune(r)
		}
	}

	if digits.Len() == 0 {
		return 0, "", "", "", fmt.Errorf("missing multiplier digits in timeframe: %s", tf)
	}

	multiplier, err = strconv.Atoi(digits.String())
	if err != nil {
		return 0, "", "", "", fmt.Errorf("invalid timeframe multiplier: %w", err)
	}

	unit := strings.ToLower(suffix.String())
	if unit == "" {
		unit = "d" // default to day if no unit suffix is present
	}

	var daysBack int
	switch unit {
	case "m", "min", "minute", "minutes":
		timespan = "minute"
		daysBack = 30 // fetch 30 days back for minutes
	case "h", "hour", "hours":
		timespan = "hour"
		daysBack = 365 // fetch 1 year back for hours
	case "d", "day", "days":
		timespan = "day"
		daysBack = 365 * 2 // fetch 2 years back for days
	case "w", "week", "weeks":
		timespan = "week"
		daysBack = 365 * 5 // fetch 5 years back for weeks
	default:
		return 0, "", "", "", fmt.Errorf("unsupported timeframe unit: %s", unit)
	}

	now := time.Now()
	from = now.AddDate(0, 0, -daysBack).Format("2006-01-02")
	to = now.Format("2006-01-02")
	return multiplier, timespan, from, to, nil
}

func getChildTimeframe(parent string) string {
	p := strings.ToUpper(parent)
	if p == "1D" || p == "D" || strings.HasSuffix(p, "DAY") || strings.HasSuffix(p, "DAYS") {
		return "1H"
	}
	if p == "1H" || p == "H" || strings.HasSuffix(p, "HOUR") || strings.HasSuffix(p, "HOURS") {
		return "15m"
	}
	if p == "15M" || p == "15MIN" || p == "15MINUTES" || p == "15MINUTE" {
		return "1m"
	}
	return ""
}

// getOrFetchCandles tries loading candles from SQLite cache first, then falls back to fetching from external client.
func (h *Handler) getOrFetchCandles(ctx context.Context, ticker, timeframe string, multiplier int, timespan string, fromDateStr, toDateStr string, fromTime, toTime time.Time) ([]model.Candle, error) {
	candles, err := h.repo.GetCandles(ctx, ticker, timeframe, fromTime.Unix(), toTime.Unix())
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}

	if len(candles) == 0 {
		// Cache miss: fetch from external client
		candles, err = h.fetcher.FetchCandles(ctx, ticker, multiplier, timespan, fromDateStr, toDateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch candles: %w", err)
		}

		if len(candles) > 0 {
			// Write fetched candles to cache
			if err := h.repo.SaveCandles(ctx, ticker, timeframe, candles); err != nil {
				return nil, fmt.Errorf("failed to cache candles: %w", err)
			}
		}
	}
	return candles, nil
}

