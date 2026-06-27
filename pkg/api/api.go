package api

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
	"WaveSight/pkg/polygon"
	"WaveSight/pkg/repository"
)

const maxRequestBody = 64 << 10

var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.\-]{0,14}$`)

type CandleFetcher interface {
	FetchCandles(ctx context.Context, ticker string, multiplier int, timespan, from, to string) ([]market.Candle, error)
}

type Analyzer interface {
	Analyze(input wave.AnalyzeInput) wave.AnalysisResult
}

type HandlerConfig struct {
	AllowedOrigins     []string
	MaxConcurrentScans int
	StaticDir          string
}

type Handler struct {
	fetcher  CandleFetcher
	store    repository.Store
	analyzer Analyzer
	calendar *market.Calendar
	router   *http.ServeMux
	origins  map[string]struct{}
	slots    chan struct{}
	queue    chan struct{}
	limiter  *rateLimiter
	now      func() time.Time
}

func NewHandler(fetcher CandleFetcher, store repository.Store, analyzer Analyzer, calendar *market.Calendar, config HandlerConfig) *Handler {
	if config.MaxConcurrentScans < 1 {
		config.MaxConcurrentScans = 4
	}
	origins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		origins[origin] = struct{}{}
	}
	handler := &Handler{
		fetcher: fetcher, store: store, analyzer: analyzer, calendar: calendar,
		router: http.NewServeMux(), origins: origins,
		slots:   make(chan struct{}, config.MaxConcurrentScans),
		queue:   make(chan struct{}, config.MaxConcurrentScans*4),
		limiter: newRateLimiter(30, 10), now: time.Now,
	}
	handler.registerRoutes()
	if config.StaticDir != "" {
		handler.router.Handle("/", newSPAHandler(config.StaticDir))
	}
	return handler
}

func (h *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	requestID := request.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("%x", h.now().UnixNano())
	}
	writer.Header().Set("X-Request-ID", requestID)
	if origin := request.Header.Get("Origin"); origin != "" {
		if _, allowed := h.origins[origin]; allowed {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Vary", "Origin")
		}
	}
	writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
	if request.Method == http.MethodOptions {
		writer.WriteHeader(http.StatusNoContent)
		return
	}

	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	if !h.limiter.allow(host, h.now()) {
		h.writeProblem(writer, requestID, http.StatusTooManyRequests, "rate-limit", "Too many requests", "Please retry later.")
		return
	}
	h.router.ServeHTTP(writer, request)
}

func (h *Handler) registerRoutes() {
	h.router.HandleFunc("GET /healthz", h.handleHealth)
	h.router.HandleFunc("POST /api/v2/analyses", h.handleCreateAnalysis)
	h.router.HandleFunc("GET /api/v2/analyses", h.handleHistory)
	h.router.HandleFunc("GET /api/v2/analyses/{id}", h.handleGetAnalysis)
}

func (h *Handler) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}

func (h *Handler) handleCreateAnalysis(writer http.ResponseWriter, request *http.Request) {
	requestID := writer.Header().Get("X-Request-ID")
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, maxRequestBody))
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return
	}
	if err := request.Body.Close(); err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return
	}
	var input AnalysisRequest
	if err := input.UnmarshalJSON(body); err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-json", "Invalid JSON", err.Error())
		return
	}
	normalized, timeframe, session, asOf, err := h.validateRequest(input)
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-analysis-request", "Invalid analysis request", err.Error())
		return
	}

	select {
	case h.queue <- struct{}{}:
		defer func() { <-h.queue }()
	default:
		h.writeProblem(writer, requestID, http.StatusServiceUnavailable, "analysis-queue-full", "Analysis capacity reached", "Retry when another scan has completed.")
		return
	}
	select {
	case h.slots <- struct{}{}:
		defer func() { <-h.slots }()
	case <-request.Context().Done():
		h.writeProblem(writer, requestID, http.StatusRequestTimeout, "cancelled", "Request cancelled", request.Context().Err().Error())
		return
	}

	from, to := analysisRange(timeframe, asOf, normalized.LookbackBars)
	candles, err := h.getOrFetchCandles(request.Context(), normalized.Symbol, timeframe, from, to)
	if err != nil {
		var rateLimit *polygon.RateLimitError
		if errors.As(err, &rateLimit) {
			h.writeProblem(writer, requestID, http.StatusTooManyRequests, "provider-rate-limit", "Market data rate limit", rateLimit.Error())
			return
		}
		h.writeProblem(writer, requestID, http.StatusBadGateway, "market-data-error", "Market data unavailable", err.Error())
		return
	}
	candles = h.calendar.Normalize(candles, timeframe, session)
	candles = market.TrimToLookback(candles, normalized.LookbackBars)
	if len(candles) < 20 {
		h.writeProblem(writer, requestID, http.StatusUnprocessableEntity, "insufficient-data", "Insufficient market data", "At least 20 normalized bars are required.")
		return
	}

	snapshotID, requestHash, dataFingerprint := snapshotHash(normalized, candles)
	if cached, err := h.store.GetSnapshot(request.Context(), snapshotID); err == nil {
		h.writeJSON(writer, http.StatusOK, cached)
		return
	} else if !errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-read-error", "Snapshot read failed", err.Error())
		return
	}

	futureBars := h.calendar.FutureBarTimes(
		time.Unix(candles[len(candles)-1].Time, 0), timeframe, session, 200,
	)
	result := h.analyzer.Analyze(wave.AnalyzeInput{
		Candles: candles, Timeframe: timeframe, Session: session,
		MaxScenarios: normalized.MaxScenarios, FutureBars: futureBars, TickSize: 0.01,
	})

	generatedAt := h.now().UTC().Unix()
	snapshot := AnalysisSnapshot{
		ID: snapshotID, TheoryVersion: wave.TheoryVersion, EngineVersion: wave.EngineVersion,
		GeneratedAt: generatedAt, Request: normalized, DataQuality: result.DataQuality,
		Candles: candles, Scenarios: result.Scenarios, FutureBars: result.FutureBars,
	}
	payload, err := snapshot.MarshalJSON()
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "serialization-error", "Snapshot serialization failed", err.Error())
		return
	}
	metadata := repository.SnapshotMetadata{
		ID: snapshotID, Symbol: normalized.Symbol, Timeframe: normalized.Timeframe,
		Session: normalized.Session, AsOf: asOf.Unix(), GeneratedAt: generatedAt,
		TheoryVersion: wave.TheoryVersion, EngineVersion: wave.EngineVersion,
		RequestHash: requestHash, DataFingerprint: dataFingerprint,
	}
	if err := h.store.SaveSnapshot(request.Context(), metadata, payload); err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-write-error", "Snapshot persistence failed", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusCreated, payload)
}

func (h *Handler) handleGetAnalysis(writer http.ResponseWriter, request *http.Request) {
	requestID := writer.Header().Get("X-Request-ID")
	id := request.PathValue("id")
	if len(id) != 32 {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-snapshot-id", "Invalid snapshot ID", "Snapshot IDs contain 32 hexadecimal characters.")
		return
	}
	payload, err := h.store.GetSnapshot(request.Context(), id)
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, requestID, http.StatusNotFound, "snapshot-not-found", "Snapshot not found", "No immutable snapshot exists for this ID.")
		return
	}
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-read-error", "Snapshot read failed", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) handleHistory(writer http.ResponseWriter, request *http.Request) {
	requestID := writer.Header().Get("X-Request-ID")
	limit := 20
	if value := request.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-limit", "Invalid history limit", "Limit must be between 1 and 100.")
			return
		}
		limit = parsed
	}
	items, err := h.store.ListSnapshots(request.Context(), limit)
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "history-read-error", "Analysis history unavailable", err.Error())
		return
	}
	payload, err := (&SnapshotHistory{Items: items}).MarshalJSON()
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "serialization-error", "History serialization failed", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) validateRequest(input AnalysisRequest) (AnalysisRequest, market.Timeframe, market.Session, time.Time, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	if !symbolPattern.MatchString(input.Symbol) {
		return input, "", "", time.Time{}, fmt.Errorf("symbol must be a US stock or ETF ticker")
	}
	timeframe, ok := market.ParseTimeframe(input.Timeframe)
	if !ok {
		return input, "", "", time.Time{}, fmt.Errorf("timeframe must be 1m, 5m, 15m, 1h, 4h, 1D or 1W")
	}
	input.Timeframe = string(timeframe)
	session := market.Session(strings.ToUpper(strings.TrimSpace(input.Session)))
	if session == "" {
		session = market.SessionRTH
	}
	if session != market.SessionRTH && session != market.SessionExtended {
		return input, "", "", time.Time{}, fmt.Errorf("session must be RTH or EXTENDED")
	}
	input.Session = string(session)

	asOf := h.now().UTC()
	if strings.TrimSpace(input.AsOf) != "" {
		parsed, err := time.Parse(time.RFC3339, input.AsOf)
		if err != nil {
			return input, "", "", time.Time{}, fmt.Errorf("as_of must be RFC3339: %w", err)
		}
		asOf = parsed.UTC()
	}
	input.AsOf = asOf.Format(time.RFC3339)
	if input.LookbackBars <= 0 {
		input.LookbackBars = timeframe.DefaultLookbackBars()
	}
	if input.LookbackBars < 200 || input.LookbackBars > 50_000 {
		return input, "", "", time.Time{}, fmt.Errorf("lookback_bars must be between 200 and 50000")
	}
	if input.MaxScenarios <= 0 {
		input.MaxScenarios = 5
	}
	if input.MaxScenarios > 5 {
		return input, "", "", time.Time{}, fmt.Errorf("max_scenarios cannot exceed 5")
	}
	return input, timeframe, session, asOf, nil
}

func (h *Handler) getOrFetchCandles(ctx context.Context, symbol string, timeframe market.Timeframe, from, to time.Time) ([]market.Candle, error) {
	covered, err := h.store.HasCoverage(ctx, symbol, timeframe, from.Unix(), to.Unix())
	if err != nil {
		return nil, err
	}
	if !covered {
		multiplier, timespan := timeframe.ProviderRange()
		fetched, err := h.fetcher.FetchCandles(
			ctx, symbol, multiplier, timespan,
			from.Format("2006-01-02"), to.Format("2006-01-02"),
		)
		if err != nil {
			return nil, err
		}
		if err := h.store.SaveCandles(ctx, symbol, timeframe, fetched); err != nil {
			return nil, err
		}
		if err := h.store.SaveCoverage(ctx, symbol, timeframe, from.Unix(), to.Unix()); err != nil {
			return nil, err
		}
	}
	return h.store.GetCandles(ctx, symbol, timeframe, from.Unix(), to.Unix())
}

func analysisRange(timeframe market.Timeframe, asOf time.Time, lookbackBars int) (time.Time, time.Time) {
	days := 730
	switch timeframe {
	case market.Timeframe1m:
		days = maxInt(45, lookbackBars/390*2)
	case market.Timeframe5m:
		days = maxInt(180, lookbackBars/78*2)
	case market.Timeframe15m:
		days = maxInt(540, lookbackBars/26*2)
	case market.Timeframe1h:
		days = maxInt(1_825, lookbackBars/7*2)
	case market.Timeframe4h:
		days = maxInt(3_650, lookbackBars/2*2)
	case market.Timeframe1D:
		days = maxInt(7_300, lookbackBars/252*365)
	case market.Timeframe1W:
		days = maxInt(14_600, lookbackBars/52*365)
	}
	return asOf.AddDate(0, 0, -days), asOf
}

func snapshotHash(request AnalysisRequest, candles []market.Candle) (string, string, string) {
	requestHasher := sha256.New()
	_, _ = io.WriteString(requestHasher, request.Symbol)
	_, _ = io.WriteString(requestHasher, "|"+request.Timeframe+"|"+request.Session+"|"+request.AsOf)
	_, _ = io.WriteString(requestHasher, fmt.Sprintf("|%d|%d", request.LookbackBars, request.MaxScenarios))
	requestSum := requestHasher.Sum(nil)
	requestHash := hex.EncodeToString(requestSum[:16])

	candleHasher := sha256.New()
	var buffer [8]byte
	for _, candle := range candles {
		binary.LittleEndian.PutUint64(buffer[:], uint64(candle.Time))
		_, _ = candleHasher.Write(buffer[:])
		for _, value := range []float64{candle.Open, candle.High, candle.Low, candle.Close, candle.Volume} {
			binary.LittleEndian.PutUint64(buffer[:], math.Float64bits(value))
			_, _ = candleHasher.Write(buffer[:])
		}
	}
	candleSum := candleHasher.Sum(nil)
	dataFingerprint := hex.EncodeToString(candleSum[:16])

	hasher := sha256.New()
	_, _ = io.WriteString(hasher, requestHash+"|"+dataFingerprint)
	_, _ = io.WriteString(hasher, "|"+wave.TheoryVersion+"|"+wave.EngineVersion)
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum[:16]), requestHash, dataFingerprint
}

func (h *Handler) writeProblem(writer http.ResponseWriter, requestID string, status int, problemType, title, detail string) {
	problem := Problem{
		Type: "https://wavesight.app/problems/" + problemType, Title: title,
		Status: status, Detail: detail, RequestID: requestID,
	}
	payload, err := problem.MarshalJSON()
	if err != nil {
		http.Error(writer, title, status)
		return
	}
	h.writePayload(writer, status, "application/problem+json", payload)
}

func (h *Handler) writeJSON(writer http.ResponseWriter, status int, payload []byte) {
	h.writePayload(writer, status, "application/json", payload)
}

func (h *Handler) writePayload(writer http.ResponseWriter, status int, contentType string, payload []byte) {
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(status)
	_, _ = writer.Write(payload)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
