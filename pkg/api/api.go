package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"WaveSight/internal/domain/master"
	"WaveSight/internal/market"
	"WaveSight/pkg/repository"
)

const maxRequestBody = 64 << 10

var symbolPattern = regexp.MustCompile(`^[A-Z][A-Z0-9.\-]{0,14}$`)

type CandleFetcher interface {
	FetchCandles(ctx context.Context, ticker string, multiplier int, timespan, from, to string) ([]market.Candle, error)
}

type HandlerConfig struct {
	AllowedOrigins     []string
	MaxConcurrentScans int
	StaticDir          string
}

type Handler struct {
	fetcher        CandleFetcher
	store          repository.Store
	calendar       *market.Calendar
	router         *http.ServeMux
	origins        map[string]struct{}
	limiter        *rateLimiter
	now            func() time.Time
	v3Store        repository.V3Store
	masterAnalyzer *master.Engine
	v3Queue        chan v3Task
	nativeLocks    sync.Map
}

func NewHandler(fetcher CandleFetcher, store repository.Store, calendar *market.Calendar, config HandlerConfig) *Handler {
	if config.MaxConcurrentScans < 1 {
		config.MaxConcurrentScans = 4
	}
	origins := make(map[string]struct{}, len(config.AllowedOrigins))
	for _, origin := range config.AllowedOrigins {
		origins[origin] = struct{}{}
	}
	handler := &Handler{
		fetcher: fetcher, store: store, calendar: calendar,
		router: http.NewServeMux(), origins: origins,
		limiter: newRateLimiter(180, 30), now: time.Now,
		masterAnalyzer: master.NewEngine(),
		v3Queue:        make(chan v3Task, config.MaxConcurrentScans*4),
	}
	if v3Store, ok := store.(repository.V3Store); ok {
		handler.v3Store = v3Store
	}
	handler.registerRoutes()
	handler.startV3Workers(config.MaxConcurrentScans)
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
	h.router.HandleFunc("POST /api/v2/analyses", h.handleV2CreateGone)
	h.router.HandleFunc("GET /api/v2/analyses", h.handleHistory)
	h.router.HandleFunc("GET /api/v2/analyses/{id}", h.handleGetAnalysis)
	h.router.HandleFunc("POST /api/v3/analysis-jobs", h.handleCreateV3Job)
	h.router.HandleFunc("GET /api/v3/analysis-jobs/{id}", h.handleGetV3Job)
	h.router.HandleFunc("GET /api/v3/analyses", h.handleV3History)
	h.router.HandleFunc("GET /api/v3/analyses/{id}", h.handleGetV3Analysis)
	h.router.HandleFunc("GET /api/v3/analyses/{id}/views/{timeframe}", h.handleGetV3View)
	h.router.HandleFunc("POST /api/v3/analyses/{id}/refinements", h.handleCreateRefinement)
}

func (h *Handler) handleHealth(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
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
