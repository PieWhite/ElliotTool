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
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"WaveSight/internal/domain/master"
	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
	"WaveSight/pkg/polygon"
	"WaveSight/pkg/repository"
)

const (
	maxAnalysisDuration = 20 * time.Minute
	nativeHistoryStart  = "2003-09-10"
)

type DetailedCandleFetcher interface {
	FetchCandlesDetailed(
		ctx context.Context,
		ticker string,
		multiplier int,
		timespan, from, to string,
	) (polygon.FetchResult, error)
}

type v3Task struct {
	Job            master.AnalysisJob
	RequestKey     string
	ParentID       string
	RefinementFrom *time.Time
	RefinementTo   *time.Time
}

func (h *Handler) startV3Workers(count int) {
	if h.v3Store == nil {
		return
	}
	for worker := 0; worker < count; worker++ {
		go func() {
			for task := range h.v3Queue {
				ctx, cancel := context.WithTimeout(context.Background(), maxAnalysisDuration)
				h.runV3Task(ctx, task)
				cancel()
			}
		}()
	}
}

func (h *Handler) handleV2CreateGone(writer http.ResponseWriter, _ *http.Request) {
	h.writeProblem(
		writer, writer.Header().Get("X-Request-ID"), http.StatusGone,
		"v2-read-only", "V2 analysis creation is retired",
		"Existing v2 snapshots remain readable. Create coherent master analyses through /api/v3/analysis-jobs.",
	)
}

func (h *Handler) handleCreateV3Job(writer http.ResponseWriter, request *http.Request) {
	requestID := writer.Header().Get("X-Request-ID")
	if h.v3Store == nil {
		h.writeProblem(writer, requestID, http.StatusServiceUnavailable, "v3-store-unavailable", "V3 unavailable", "The configured repository does not support master snapshots.")
		return
	}
	var input master.AnalysisRequest
	if !h.decodeJSON(writer, request, requestID, &input) {
		return
	}
	normalized, asOf, err := h.validateV3Request(input)
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-analysis-request", "Invalid analysis request", err.Error())
		return
	}
	key := v3RequestKey(normalized, "")
	if _, payload, err := h.v3Store.FindSnapshotV3(request.Context(), key); err == nil {
		h.writeJSON(writer, http.StatusOK, payload)
		return
	} else if !errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-read-error", "Snapshot lookup failed", err.Error())
		return
	}
	if _, payload, err := h.v3Store.FindJob(request.Context(), key); err == nil {
		h.writeJSON(writer, http.StatusAccepted, payload)
		return
	} else if !errors.Is(err, repository.ErrJobNotFound) {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "job-read-error", "Job lookup failed", err.Error())
		return
	}
	now := h.now().UTC().Unix()
	job := master.AnalysisJob{
		ID:     "job-" + shortHash(fmt.Sprintf("%s|%d", key, h.now().UnixNano())),
		Status: master.JobQueued, Progress: 0, Message: "Scan queued",
		Request: normalized, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.persistJob(request.Context(), key, job); err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "job-write-error", "Job could not be queued", err.Error())
		return
	}
	task := v3Task{Job: job, RequestKey: key}
	select {
	case h.v3Queue <- task:
		payload, err := job.MarshalJSON()
		if err != nil {
			h.writeProblem(writer, requestID, http.StatusInternalServerError, "serialization-error", "Job serialization failed", err.Error())
			return
		}
		h.writeJSON(writer, http.StatusAccepted, payload)
	default:
		job.Status = master.JobFailed
		job.Error = "Analysis queue is full."
		job.Message = job.Error
		job.UpdatedAt = h.now().UTC().Unix()
		_ = h.persistJob(context.Background(), key, job)
		h.writeProblem(writer, requestID, http.StatusServiceUnavailable, "analysis-queue-full", "Analysis capacity reached", "Retry when another master scan has completed.")
	}
	_ = asOf
}

func (h *Handler) handleGetV3Job(writer http.ResponseWriter, request *http.Request) {
	if h.v3Store == nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusServiceUnavailable, "v3-store-unavailable", "V3 unavailable", "The configured repository does not support master jobs.")
		return
	}
	payload, err := h.v3Store.GetJob(request.Context(), request.PathValue("id"))
	if errors.Is(err, repository.ErrJobNotFound) {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusNotFound, "job-not-found", "Job not found", "No analysis job exists for this ID.")
		return
	}
	if err != nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusInternalServerError, "job-read-error", "Job unavailable", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) handleGetV3Analysis(writer http.ResponseWriter, request *http.Request) {
	payload, err := h.v3Store.GetSnapshotV3(request.Context(), request.PathValue("id"))
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusNotFound, "snapshot-not-found", "Snapshot not found", "No immutable v3 snapshot exists for this ID.")
		return
	}
	if err != nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusInternalServerError, "snapshot-read-error", "Snapshot unavailable", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) handleGetV3View(writer http.ResponseWriter, request *http.Request) {
	timeframe, ok := market.ParseTimeframe(request.PathValue("timeframe"))
	if !ok {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusBadRequest, "invalid-timeframe", "Invalid timeframe", "Timeframe must be 1m, 5m, 15m, 1h, 4h, 1D or 1W.")
		return
	}
	payload, err := h.v3Store.GetViewV3(request.Context(), request.PathValue("id"), timeframe)
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusNotFound, "view-not-found", "View not found", "The requested local projection is unavailable.")
		return
	}
	if err != nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusInternalServerError, "view-read-error", "View unavailable", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) handleV3History(writer http.ResponseWriter, request *http.Request) {
	limit := 20
	if raw := request.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusBadRequest, "invalid-limit", "Invalid history limit", "Limit must be between 1 and 100.")
			return
		}
		limit = parsed
	}
	items, err := h.v3Store.ListSnapshotsV3(request.Context(), limit)
	if err != nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusInternalServerError, "history-read-error", "Analysis history unavailable", err.Error())
		return
	}
	payload, err := (&SnapshotHistoryV3{Items: items}).MarshalJSON()
	if err != nil {
		h.writeProblem(writer, writer.Header().Get("X-Request-ID"), http.StatusInternalServerError, "serialization-error", "History serialization failed", err.Error())
		return
	}
	h.writeJSON(writer, http.StatusOK, payload)
}

func (h *Handler) handleCreateRefinement(writer http.ResponseWriter, request *http.Request) {
	requestID := writer.Header().Get("X-Request-ID")
	parentPayload, err := h.v3Store.GetSnapshotV3(request.Context(), request.PathValue("id"))
	if errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, requestID, http.StatusNotFound, "snapshot-not-found", "Snapshot not found", "The parent snapshot does not exist.")
		return
	}
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-read-error", "Snapshot unavailable", err.Error())
		return
	}
	var parent master.AnalysisSnapshot
	if err := parent.UnmarshalJSON(parentPayload); err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-decode-error", "Snapshot is invalid", err.Error())
		return
	}
	var refinement master.RefinementRequest
	if !h.decodeJSON(writer, request, requestID, &refinement) {
		return
	}
	from, err := time.Parse(time.RFC3339, refinement.From)
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-refinement", "Invalid refinement", "from must be RFC3339.")
		return
	}
	to, err := time.Parse(time.RFC3339, refinement.To)
	if err != nil || !to.After(from) || to.After(time.Unix(parent.DatasetManifest.MinuteDetailFrom, 0)) {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-refinement", "Invalid refinement", "to must be after from and cover previously unloaded minute history.")
		return
	}
	key := v3RequestKey(parent.Request, parent.ID+"|"+from.UTC().Format(time.RFC3339)+"|"+to.UTC().Format(time.RFC3339))
	now := h.now().UTC().Unix()
	if snapshotID, _, err := h.v3Store.FindSnapshotV3(request.Context(), key); err == nil {
		completed := master.AnalysisJob{
			ID: "job-" + shortHash(key), Status: master.JobCompleted, Progress: 100,
			Message: "Historical refinement already exists", SnapshotID: snapshotID,
			Request: parent.Request, CreatedAt: now, UpdatedAt: now,
		}
		payload, marshalErr := completed.MarshalJSON()
		if marshalErr != nil {
			h.writeProblem(writer, requestID, http.StatusInternalServerError, "serialization-error", "Refinement serialization failed", marshalErr.Error())
			return
		}
		h.writeJSON(writer, http.StatusOK, payload)
		return
	} else if !errors.Is(err, repository.ErrSnapshotNotFound) {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "snapshot-read-error", "Refinement lookup failed", err.Error())
		return
	}
	job := master.AnalysisJob{
		ID:     "job-" + shortHash(fmt.Sprintf("%s|%d", key, h.now().UnixNano())),
		Status: master.JobQueued, Message: "Historical refinement queued",
		Request: parent.Request, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.persistJob(request.Context(), key, job); err != nil {
		h.writeProblem(writer, requestID, http.StatusInternalServerError, "job-write-error", "Refinement could not be queued", err.Error())
		return
	}
	task := v3Task{
		Job: job, RequestKey: key, ParentID: parent.ID,
		RefinementFrom: &from, RefinementTo: &to,
	}
	select {
	case h.v3Queue <- task:
		payload, _ := job.MarshalJSON()
		h.writeJSON(writer, http.StatusAccepted, payload)
	default:
		h.writeProblem(writer, requestID, http.StatusServiceUnavailable, "analysis-queue-full", "Analysis capacity reached", "Retry later.")
	}
}

type jsonUnmarshaler interface {
	UnmarshalJSON([]byte) error
}

func (h *Handler) decodeJSON(
	writer http.ResponseWriter,
	request *http.Request,
	requestID string,
	target jsonUnmarshaler,
) bool {
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, maxRequestBody))
	if err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return false
	}
	if err := request.Body.Close(); err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-body", "Invalid request body", err.Error())
		return false
	}
	if err := target.UnmarshalJSON(body); err != nil {
		h.writeProblem(writer, requestID, http.StatusBadRequest, "invalid-json", "Invalid JSON", err.Error())
		return false
	}
	return true
}

func (h *Handler) validateV3Request(
	input master.AnalysisRequest,
) (master.AnalysisRequest, time.Time, error) {
	input.Symbol = strings.ToUpper(strings.TrimSpace(input.Symbol))
	if !symbolPattern.MatchString(input.Symbol) {
		return input, time.Time{}, fmt.Errorf("symbol must be a US stock or ETF ticker")
	}
	if input.Session == "" {
		input.Session = market.SessionRTH
	}
	if input.Session != market.SessionRTH && input.Session != market.SessionExtended {
		return input, time.Time{}, fmt.Errorf("session must be RTH or EXTENDED")
	}
	if input.FocusTimeframe == "" {
		input.FocusTimeframe = market.Timeframe1D
	}
	if _, ok := market.ParseTimeframe(string(input.FocusTimeframe)); !ok {
		return input, time.Time{}, fmt.Errorf("focus_timeframe must be 1m, 5m, 15m, 1h, 4h, 1D or 1W")
	}
	if input.HistoryProfile == "" {
		input.HistoryProfile = master.HistoryMaxDailyTwoYearMinute
	}
	if input.HistoryProfile != master.HistoryMaxDailyTwoYearMinute {
		return input, time.Time{}, fmt.Errorf("unsupported history_profile")
	}
	if input.MaxScenarios == 0 {
		input.MaxScenarios = 5
	}
	if input.MaxScenarios < 1 || input.MaxScenarios > 5 {
		return input, time.Time{}, fmt.Errorf("max_scenarios must be between 1 and 5")
	}
	asOf := h.now().UTC()
	if strings.TrimSpace(input.AsOf) != "" {
		parsed, err := time.Parse(time.RFC3339, input.AsOf)
		if err != nil {
			return input, time.Time{}, fmt.Errorf("as_of must be RFC3339: %w", err)
		}
		asOf = parsed.UTC()
	}
	input.AsOf = asOf.Format(time.RFC3339)
	return input, asOf, nil
}

func (h *Handler) runV3Task(ctx context.Context, task v3Task) {
	fail := func(err error) {
		task.Job.Status = master.JobFailed
		task.Job.Progress = 100
		task.Job.Message = "Analysis failed"
		task.Job.Error = err.Error()
		task.Job.UpdatedAt = h.now().UTC().Unix()
		_ = h.persistJob(context.Background(), task.RequestKey, task.Job)
	}
	update := func(status master.JobStatus, progress int, message string) error {
		task.Job.Status, task.Job.Progress, task.Job.Message = status, progress, message
		task.Job.UpdatedAt = h.now().UTC().Unix()
		return h.persistJob(ctx, task.RequestKey, task.Job)
	}
	asOf, err := time.Parse(time.RFC3339, task.Job.Request.AsOf)
	if err != nil {
		fail(fmt.Errorf("parsing job as_of: %w", err))
		return
	}
	if err := update(master.JobAcquiringDaily, 8, "Loading complete native daily history"); err != nil {
		fail(err)
		return
	}
	dailyFrom, _ := time.Parse("2006-01-02", nativeHistoryStart)
	daily, dailyTelemetry, err := h.acquireNative(
		ctx, task.Job.Request.Symbol, master.NativeDaily, dailyFrom, asOf, false,
	)
	if err != nil {
		fail(err)
		return
	}
	if err := update(master.JobAcquiringMinute, 25, "Loading canonical minute detail"); err != nil {
		fail(err)
		return
	}
	minuteFrom := asOf.AddDate(-2, 0, 0)
	if task.RefinementFrom != nil && task.RefinementFrom.Before(minuteFrom) {
		minuteFrom = *task.RefinementFrom
	}
	minuteTo := asOf
	if task.RefinementTo != nil {
		minuteTo = *task.RefinementTo
	}
	_, minuteTelemetry, err := h.acquireNative(
		ctx, task.Job.Request.Symbol, master.NativeMinute, minuteFrom, minuteTo, task.RefinementFrom != nil,
	)
	if err != nil {
		fail(err)
		return
	}
	minuteCoverage, err := h.v3Store.NativeCoverage(
		ctx, task.Job.Request.Symbol, string(master.NativeMinute),
	)
	if err != nil {
		fail(err)
		return
	}
	detailFrom := minuteFrom.Unix()
	for _, interval := range minuteCoverage {
		if interval.To >= dailyFrom.Unix() && interval.From <= asOf.Unix() && interval.From < detailFrom {
			detailFrom = interval.From
		}
	}
	allMinutes, err := h.v3Store.GetNativeCandles(
		ctx, task.Job.Request.Symbol, string(master.NativeMinute), detailFrom, asOf.Unix(),
	)
	if err != nil {
		fail(err)
		return
	}
	if err := update(master.JobAggregatingViews, 42, "Building seven local chart projections"); err != nil {
		fail(err)
		return
	}
	views, err := h.calendar.BuildCanonicalViews(allMinutes, daily, task.Job.Request.Session, asOf)
	if err != nil {
		fail(err)
		return
	}
	future := make(map[market.Timeframe][]int64, 7)
	for _, timeframe := range []market.Timeframe{
		market.Timeframe1m, market.Timeframe5m, market.Timeframe15m,
		market.Timeframe1h, market.Timeframe4h, market.Timeframe1D, market.Timeframe1W,
	} {
		candles := views.Views[timeframe]
		if len(candles) > 0 {
			future[timeframe] = h.calendar.FutureBarTimes(
				time.Unix(candles[len(candles)-1].Time, 0), timeframe,
				task.Job.Request.Session, 200,
			)
		}
	}
	coverageManifest := []master.CoverageInterval{
		{Resolution: master.NativeDaily, From: dailyFrom.Unix(), To: asOf.Unix(), Complete: true},
	}
	for _, interval := range minuteCoverage {
		if interval.To < dailyFrom.Unix() || interval.From > asOf.Unix() {
			continue
		}
		coverageManifest = append(coverageManifest, master.CoverageInterval{
			Resolution: master.NativeMinute, From: interval.From, To: interval.To, Complete: true,
		})
	}
	manifest := master.DatasetManifest{
		Coverage:        coverageManifest,
		ProviderQueries: []master.ProviderQueryTelemetry{dailyTelemetry, minuteTelemetry},
		DailyProvenance: compareDailyProvenance(
			views.NativeDaily, views.Views[market.Timeframe1D],
		),
		MinuteDetailFrom: detailFrom, MinuteDetailTo: asOf.Unix(),
		NativeDailyRows: len(daily), NativeMinuteRows: len(allMinutes),
	}
	if err := update(master.JobBuildingPivotGraph, 55, "Aligning pivots across resolutions"); err != nil {
		fail(err)
		return
	}
	if err := update(master.JobParsingMasterTree, 67, "Parsing the complete Elliott narrative"); err != nil {
		fail(err)
		return
	}
	snapshot, timeframeViews := h.masterAnalyzer.Analyze(master.AnalyzeInput{
		Symbol: task.Job.Request.Symbol, Session: task.Job.Request.Session,
		AsOf: asOf, FocusTimeframe: task.Job.Request.FocusTimeframe,
		MaxScenarios: task.Job.Request.MaxScenarios, ParentSnapshotID: task.ParentID,
		Views: views, Manifest: manifest, FutureBars: future,
	})
	if err := update(master.JobRankingScenarios, 82, "Ranking materially different master assignments"); err != nil {
		fail(err)
		return
	}
	fingerprint := nativeFingerprint(daily, allMinutes)
	snapshot.ID = v3SnapshotID(task.RequestKey, fingerprint, task.ParentID)
	snapshot.GeneratedAt = h.now().UTC().Unix()
	for timeframe, view := range timeframeViews {
		view.SnapshotID = snapshot.ID
		timeframeViews[timeframe] = view
		if timeframe == snapshot.Request.FocusTimeframe {
			snapshot.InitialView = view
		}
	}
	if err := update(master.JobPersisting, 92, "Persisting immutable master snapshot and views"); err != nil {
		fail(err)
		return
	}
	if err := h.persistV3Snapshot(ctx, task.RequestKey, fingerprint, snapshot, timeframeViews); err != nil {
		fail(err)
		return
	}
	task.Job.Status = master.JobCompleted
	task.Job.Progress = 100
	task.Job.Message = "Master analysis completed"
	task.Job.SnapshotID = snapshot.ID
	task.Job.UpdatedAt = h.now().UTC().Unix()
	if err := h.persistJob(context.Background(), task.RequestKey, task.Job); err != nil {
		return
	}
}

func compareDailyProvenance(
	native, canonical []market.DerivedCandle,
) master.DailyProvenanceAudit {
	nativeByDate := make(map[string]market.DerivedCandle, len(native))
	for _, candle := range native {
		nativeByDate[time.Unix(candle.Time, 0).UTC().Format("2006-01-02")] = candle
	}
	result := master.DailyProvenanceAudit{Samples: make([]master.DailyBarDifference, 0, 20)}
	for _, derived := range canonical {
		if derived.Provenance != market.ProvenanceMinuteDerived {
			continue
		}
		date := time.Unix(derived.Time, 0).UTC().Format("2006-01-02")
		nativeCandle, exists := nativeByDate[date]
		if !exists {
			continue
		}
		result.Compared++
		maxDelta := 0.0
		for _, delta := range []float64{
			math.Abs(nativeCandle.Open - derived.Open),
			math.Abs(nativeCandle.High - derived.High),
			math.Abs(nativeCandle.Low - derived.Low),
			math.Abs(nativeCandle.Close - derived.Close),
		} {
			if delta > maxDelta {
				maxDelta = delta
			}
		}
		volumeDelta := math.Abs(nativeCandle.Volume - derived.Volume)
		if maxDelta <= 1e-9 && volumeDelta <= 1e-6 {
			continue
		}
		result.Differences++
		if maxDelta > result.MaxOHLCDeviation {
			result.MaxOHLCDeviation = maxDelta
		}
		if len(result.Samples) < 20 {
			result.Samples = append(result.Samples, master.DailyBarDifference{
				Date: date, NativeTime: nativeCandle.Time, DerivedTime: derived.Time,
				MaxOHLCDeviation: maxDelta, VolumeDeviation: volumeDelta,
			})
		}
	}
	return result
}

func (h *Handler) acquireNative(
	ctx context.Context,
	symbol string,
	resolution master.NativeResolution,
	from, to time.Time,
	refinement bool,
) ([]market.Candle, master.ProviderQueryTelemetry, error) {
	lockKey := symbol + "|" + string(resolution)
	lockValue, _ := h.nativeLocks.LoadOrStore(lockKey, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	telemetry := master.ProviderQueryTelemetry{
		Resolution: resolution, From: from.Unix(), To: to.Unix(),
		LogicalQuery: true, CacheOnly: true,
	}
	coverage, err := h.v3Store.NativeCoverage(ctx, symbol, string(resolution))
	if err != nil {
		return nil, telemetry, err
	}
	fetchFrom, needsFetch := missingTail(coverage, from, to)
	if refinement {
		fetchFrom, needsFetch = firstMissingStart(coverage, from, to)
	}
	if needsFetch {
		if !refinement && len(coverage) > 0 {
			fetchFrom = h.calendar.TradingDaysBefore(fetchFrom, 5)
			if fetchFrom.Before(from) {
				fetchFrom = from
			}
		}
		multiplier, timespan := 1, "day"
		if resolution == master.NativeMinute {
			timespan = "minute"
		}
		result, err := h.fetchDetailed(
			ctx, symbol, multiplier, timespan,
			fetchFrom.Format("2006-01-02"), to.Format("2006-01-02"),
		)
		if err != nil {
			return nil, telemetry, err
		}
		changed, err := h.v3Store.SaveNativeCandles(ctx, symbol, string(resolution), result.Candles)
		if err != nil {
			return nil, telemetry, err
		}
		if err := h.v3Store.SaveNativeCoverage(ctx, symbol, string(resolution), fetchFrom.Unix(), to.Unix()); err != nil {
			return nil, telemetry, err
		}
		telemetry.From = fetchFrom.Unix()
		telemetry.CacheOnly = false
		telemetry.PageRequests = result.PageRequests
		telemetry.Rows = len(result.Candles)
		telemetry.OverlapChanged = changed
	}
	candles, err := h.v3Store.GetNativeCandles(ctx, symbol, string(resolution), from.Unix(), to.Unix())
	return candles, telemetry, err
}

func (h *Handler) fetchDetailed(
	ctx context.Context,
	symbol string,
	multiplier int,
	timespan, from, to string,
) (polygon.FetchResult, error) {
	if detailed, ok := h.fetcher.(DetailedCandleFetcher); ok {
		return detailed.FetchCandlesDetailed(ctx, symbol, multiplier, timespan, from, to)
	}
	candles, err := h.fetcher.FetchCandles(ctx, symbol, multiplier, timespan, from, to)
	if err != nil {
		return polygon.FetchResult{}, err
	}
	return polygon.FetchResult{Candles: candles, PageRequests: 1}, nil
}

func missingTail(
	coverage []repository.CoverageRange,
	from, to time.Time,
) (time.Time, bool) {
	for _, item := range coverage {
		if item.From <= from.Unix() && item.To >= to.Unix() {
			return time.Time{}, false
		}
	}
	latest := int64(0)
	for _, item := range coverage {
		if item.To > latest && item.To >= from.Unix() {
			latest = item.To
		}
	}
	if latest > 0 && latest < to.Unix() {
		return time.Unix(latest, 0).UTC(), true
	}
	return from, true
}

func firstMissingStart(
	coverage []repository.CoverageRange,
	from, to time.Time,
) (time.Time, bool) {
	cursor := from.Unix()
	sorted := append([]repository.CoverageRange(nil), coverage...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].From < sorted[j].From })
	for _, item := range sorted {
		if item.To < cursor || item.From > to.Unix() {
			continue
		}
		if item.From > cursor {
			return time.Unix(cursor, 0).UTC(), true
		}
		if item.To >= cursor {
			cursor = item.To + 1
		}
	}
	if cursor <= to.Unix() {
		return time.Unix(cursor, 0).UTC(), true
	}
	return time.Time{}, false
}

func (h *Handler) persistJob(
	ctx context.Context,
	requestKey string,
	job master.AnalysisJob,
) error {
	payload, err := job.MarshalJSON()
	if err != nil {
		return fmt.Errorf("serializing job: %w", err)
	}
	return h.v3Store.SaveJob(ctx, job.ID, requestKey, string(job.Status), payload, job.UpdatedAt)
}

func (h *Handler) persistV3Snapshot(
	ctx context.Context,
	requestKey, fingerprint string,
	snapshot master.AnalysisSnapshot,
	views map[market.Timeframe]master.TimeframeView,
) error {
	payload, err := snapshot.MarshalJSON()
	if err != nil {
		return fmt.Errorf("serializing master snapshot: %w", err)
	}
	viewPayloads := make(map[market.Timeframe][]byte, len(views))
	for timeframe, view := range views {
		encoded, err := view.MarshalJSON()
		if err != nil {
			return fmt.Errorf("serializing %s view: %w", timeframe, err)
		}
		viewPayloads[timeframe] = encoded
	}
	eventPayloads := make(map[string][]byte, len(snapshot.Graph.Events))
	for index := range snapshot.Graph.Events {
		event := snapshot.Graph.Events[index]
		encoded, err := event.MarshalJSON()
		if err != nil {
			return fmt.Errorf("serializing event %s: %w", event.ID, err)
		}
		eventPayloads[event.ID] = encoded
	}
	nodePayloads := make(map[string][]byte, len(snapshot.Graph.Nodes))
	relations := make([]repository.NodeRelation, 0, len(snapshot.Graph.Nodes)*3)
	for index := range snapshot.Graph.Nodes {
		node := snapshot.Graph.Nodes[index]
		encoded, err := node.MarshalJSON()
		if err != nil {
			return fmt.Errorf("serializing node %s: %w", node.ID, err)
		}
		nodePayloads[node.ID] = encoded
		for position, childID := range node.ChildIDs {
			relations = append(relations, repository.NodeRelation{
				ParentID: node.ID, ChildID: childID, Position: position,
			})
		}
	}
	scenarios := make([]repository.RankedPayload, 0, len(snapshot.Scenarios))
	for index := range snapshot.Scenarios {
		scenario := snapshot.Scenarios[index]
		encoded, err := scenario.MarshalJSON()
		if err != nil {
			return fmt.Errorf("serializing scenario %s: %w", scenario.ID, err)
		}
		scenarios = append(scenarios, repository.RankedPayload{
			ID: scenario.ID, Rank: scenario.Rank, Payload: encoded,
		})
	}
	asOf, _ := time.Parse(time.RFC3339, snapshot.Request.AsOf)
	return h.v3Store.SaveSnapshotV3(ctx, repository.SnapshotMetadataV3{
		ID: snapshot.ID, ParentSnapshotID: snapshot.ParentSnapshotID,
		RequestKey: requestKey, Symbol: snapshot.Request.Symbol,
		Session: string(snapshot.Request.Session), AsOf: asOf.Unix(),
		FocusTimeframe: string(snapshot.Request.FocusTimeframe),
		GeneratedAt:    snapshot.GeneratedAt, TheoryVersion: snapshot.TheoryVersion,
		EngineVersion: snapshot.EngineVersion, DataFingerprint: fingerprint,
	}, payload, viewPayloads, eventPayloads, nodePayloads, relations, scenarios)
}

func v3RequestKey(request master.AnalysisRequest, refinement string) string {
	value := strings.Join([]string{
		request.Symbol, string(request.Session), request.AsOf,
		string(request.HistoryProfile), strconv.Itoa(request.MaxScenarios), refinement,
	}, "|")
	return shortHash(value)
}

func nativeFingerprint(daily, minute []market.Candle) string {
	hasher := sha256.New()
	var buffer [8]byte
	for _, candles := range [][]market.Candle{daily, minute} {
		for _, candle := range candles {
			binary.LittleEndian.PutUint64(buffer[:], uint64(candle.Time))
			_, _ = hasher.Write(buffer[:])
			for _, value := range []float64{candle.Open, candle.High, candle.Low, candle.Close, candle.Volume} {
				binary.LittleEndian.PutUint64(buffer[:], math.Float64bits(value))
				_, _ = hasher.Write(buffer[:])
			}
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)[:16])
}

func v3SnapshotID(requestKey, fingerprint, parent string) string {
	return shortHash(strings.Join([]string{
		requestKey, fingerprint, wave.TheoryVersion, master.EngineVersion, parent,
	}, "|")) + shortHash(fingerprint+"|"+requestKey)
}

func shortHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}
