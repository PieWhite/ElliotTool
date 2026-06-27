package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"WaveSight/internal/market"
)

func (s *SQLiteStore) SaveNativeCandles(
	ctx context.Context,
	ticker, resolution string,
	candles []market.Candle,
) (bool, error) {
	if len(candles) == 0 {
		return false, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("beginning native candle transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	changed := false
	read, err := tx.PrepareContext(ctx, `
		SELECT open, high, low, close, volume
		FROM native_bars WHERE ticker = ? AND resolution = ? AND timestamp = ?
	`)
	if err != nil {
		return false, fmt.Errorf("preparing native candle comparison: %w", err)
	}
	defer func() { _ = read.Close() }()
	write, err := tx.PrepareContext(ctx, `
		INSERT INTO native_bars
			(ticker, resolution, timestamp, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ticker, resolution, timestamp) DO UPDATE SET
			open=excluded.open, high=excluded.high, low=excluded.low,
			close=excluded.close, volume=excluded.volume
	`)
	if err != nil {
		return false, fmt.Errorf("preparing native candle upsert: %w", err)
	}
	defer func() { _ = write.Close() }()
	for _, candle := range candles {
		var open, high, low, closeValue, volume float64
		err := read.QueryRowContext(ctx, ticker, resolution, candle.Time).Scan(
			&open, &high, &low, &closeValue, &volume,
		)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return false, fmt.Errorf("comparing native candle %d: %w", candle.Time, err)
		default:
			if open != candle.Open || high != candle.High || low != candle.Low ||
				closeValue != candle.Close || volume != candle.Volume {
				changed = true
			}
		}
		if _, err := write.ExecContext(
			ctx, ticker, resolution, candle.Time, candle.Open, candle.High,
			candle.Low, candle.Close, candle.Volume,
		); err != nil {
			return false, fmt.Errorf("upserting native candle %d: %w", candle.Time, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("committing native candle transaction: %w", err)
	}
	return changed, nil
}

func (s *SQLiteStore) GetNativeCandles(
	ctx context.Context,
	ticker, resolution string,
	from, to int64,
) ([]market.Candle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT timestamp, open, high, low, close, volume
		FROM native_bars
		WHERE ticker = ? AND resolution = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`, ticker, resolution, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying native candles: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]market.Candle, 0, 16_384)
	for rows.Next() {
		var candle market.Candle
		if err := rows.Scan(
			&candle.Time, &candle.Open, &candle.High, &candle.Low,
			&candle.Close, &candle.Volume,
		); err != nil {
			return nil, fmt.Errorf("scanning native candle: %w", err)
		}
		candle.BarIndex = len(result)
		result = append(result, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating native candles: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) NativeCoverage(
	ctx context.Context,
	ticker, resolution string,
) ([]CoverageRange, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT from_timestamp, to_timestamp
		FROM native_coverage
		WHERE ticker = ? AND resolution = ?
		ORDER BY from_timestamp ASC
	`, ticker, resolution)
	if err != nil {
		return nil, fmt.Errorf("querying native coverage: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]CoverageRange, 0, 4)
	for rows.Next() {
		var item CoverageRange
		if err := rows.Scan(&item.From, &item.To); err != nil {
			return nil, fmt.Errorf("scanning native coverage: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating native coverage: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) SaveNativeCoverage(
	ctx context.Context,
	ticker, resolution string,
	from, to int64,
) error {
	if from > to {
		from, to = to, from
	}
	existing, err := s.NativeCoverage(ctx, ticker, resolution)
	if err != nil {
		return err
	}
	existing = append(existing, CoverageRange{From: from, To: to})
	sort.Slice(existing, func(i, j int) bool { return existing[i].From < existing[j].From })
	merged := make([]CoverageRange, 0, len(existing))
	for _, item := range existing {
		if len(merged) == 0 || item.From > merged[len(merged)-1].To+1 {
			merged = append(merged, item)
			continue
		}
		if item.To > merged[len(merged)-1].To {
			merged[len(merged)-1].To = item.To
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning native coverage transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(
		ctx, `DELETE FROM native_coverage WHERE ticker = ? AND resolution = ?`,
		ticker, resolution,
	); err != nil {
		return fmt.Errorf("replacing native coverage: %w", err)
	}
	for _, item := range merged {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO native_coverage
				(ticker, resolution, from_timestamp, to_timestamp)
			VALUES (?, ?, ?, ?)
		`, ticker, resolution, item.From, item.To); err != nil {
			return fmt.Errorf("writing merged native coverage: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing native coverage transaction: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveSnapshotV3(
	ctx context.Context,
	metadata SnapshotMetadataV3,
	payload []byte,
	views map[market.Timeframe][]byte,
	events, nodes map[string][]byte,
	relations []NodeRelation,
	scenarios []RankedPayload,
) error {
	if len(payload) == 0 {
		return fmt.Errorf("saving v3 snapshot %s: empty payload", metadata.ID)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning v3 snapshot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var parent any
	if metadata.ParentSnapshotID != "" {
		parent = metadata.ParentSnapshotID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO analysis_snapshots_v3
			(id, parent_snapshot_id, request_key, data_fingerprint, symbol, session,
			 as_of, focus_timeframe, generated_at, theory_version, engine_version, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, metadata.ID, parent, metadata.RequestKey, metadata.DataFingerprint,
		metadata.Symbol, metadata.Session, metadata.AsOf, metadata.FocusTimeframe,
		metadata.GeneratedAt, metadata.TheoryVersion, metadata.EngineVersion, payload); err != nil {
		return fmt.Errorf("saving v3 snapshot: %w", err)
	}
	for timeframe, view := range views {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO analysis_views_v3(snapshot_id, timeframe, payload)
			VALUES (?, ?, ?)
		`, metadata.ID, string(timeframe), view); err != nil {
			return fmt.Errorf("saving v3 %s view: %w", timeframe, err)
		}
	}
	if err := insertPayloadMap(ctx, tx, "canonical_wave_events", "event_id", metadata.ID, events); err != nil {
		return err
	}
	if err := insertPayloadMap(ctx, tx, "wave_nodes", "node_id", metadata.ID, nodes); err != nil {
		return err
	}
	for _, relation := range relations {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO wave_node_relations
				(snapshot_id, parent_node_id, child_node_id, position)
			VALUES (?, ?, ?, ?)
		`, metadata.ID, relation.ParentID, relation.ChildID, relation.Position); err != nil {
			return fmt.Errorf("saving wave relation %s/%s: %w", relation.ParentID, relation.ChildID, err)
		}
	}
	for _, scenario := range scenarios {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO master_scenario_assignments
				(snapshot_id, scenario_id, rank, payload)
			VALUES (?, ?, ?, ?)
		`, metadata.ID, scenario.ID, scenario.Rank, scenario.Payload); err != nil {
			return fmt.Errorf("saving scenario assignment %s: %w", scenario.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing v3 snapshot: %w", err)
	}
	return nil
}

func insertPayloadMap(
	ctx context.Context,
	tx *sql.Tx,
	table, idColumn, snapshotID string,
	values map[string][]byte,
) error {
	query := fmt.Sprintf(
		"INSERT OR IGNORE INTO %s(snapshot_id, %s, payload) VALUES (?, ?, ?)",
		table, idColumn,
	)
	for id, payload := range values {
		if _, err := tx.ExecContext(ctx, query, snapshotID, id, payload); err != nil {
			return fmt.Errorf("saving %s %s: %w", table, id, err)
		}
	}
	return nil
}

func (s *SQLiteStore) GetSnapshotV3(ctx context.Context, id string) ([]byte, error) {
	return readPayload(ctx, s.db, `SELECT payload FROM analysis_snapshots_v3 WHERE id = ?`, id, ErrSnapshotNotFound)
}

func (s *SQLiteStore) GetViewV3(
	ctx context.Context,
	id string,
	timeframe market.Timeframe,
) ([]byte, error) {
	return readPayload(
		ctx, s.db,
		`SELECT payload FROM analysis_views_v3 WHERE snapshot_id = ? AND timeframe = ?`,
		[]any{id, string(timeframe)}, ErrSnapshotNotFound,
	)
}

func (s *SQLiteStore) FindSnapshotV3(
	ctx context.Context,
	requestKey string,
) (string, []byte, error) {
	var id string
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT id, payload FROM analysis_snapshots_v3
		WHERE request_key = ?
		ORDER BY generated_at DESC LIMIT 1
	`, requestKey).Scan(&id, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrSnapshotNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("finding v3 snapshot: %w", err)
	}
	return id, payload, nil
}

func (s *SQLiteStore) ListSnapshotsV3(
	ctx context.Context,
	limit int,
) ([]SnapshotMetadataV3, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, COALESCE(parent_snapshot_id, ''), request_key, symbol, session,
			as_of, focus_timeframe, generated_at, theory_version, engine_version,
			data_fingerprint
		FROM analysis_snapshots_v3
		ORDER BY generated_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing v3 snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]SnapshotMetadataV3, 0, limit)
	for rows.Next() {
		var item SnapshotMetadataV3
		if err := rows.Scan(
			&item.ID, &item.ParentSnapshotID, &item.RequestKey, &item.Symbol,
			&item.Session, &item.AsOf, &item.FocusTimeframe, &item.GeneratedAt,
			&item.TheoryVersion, &item.EngineVersion, &item.DataFingerprint,
		); err != nil {
			return nil, fmt.Errorf("scanning v3 snapshot metadata: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating v3 snapshots: %w", err)
	}
	return result, nil
}

func (s *SQLiteStore) SaveJob(
	ctx context.Context,
	id, requestKey, status string,
	payload []byte,
	updatedAt int64,
) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO analysis_jobs_v3(id, request_key, status, payload, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status=excluded.status, payload=excluded.payload, updated_at=excluded.updated_at
	`, id, requestKey, status, payload, updatedAt); err != nil {
		return fmt.Errorf("saving v3 analysis job: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetJob(ctx context.Context, id string) ([]byte, error) {
	return readPayload(ctx, s.db, `SELECT payload FROM analysis_jobs_v3 WHERE id = ?`, id, ErrJobNotFound)
}

func (s *SQLiteStore) FindJob(
	ctx context.Context,
	requestKey string,
) (string, []byte, error) {
	var id string
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT id, payload FROM analysis_jobs_v3
		WHERE request_key = ? AND status NOT IN ('COMPLETED', 'FAILED')
		ORDER BY updated_at DESC LIMIT 1
	`, requestKey).Scan(&id, &payload)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, ErrJobNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("finding v3 analysis job: %w", err)
	}
	return id, payload, nil
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func readPayload(
	ctx context.Context,
	queryer rowQuerier,
	query string,
	argument any,
	notFound error,
) ([]byte, error) {
	args := []any{argument}
	if values, ok := argument.([]any); ok {
		args = values
	}
	var payload []byte
	err := queryer.QueryRowContext(ctx, query, args...).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading persisted payload: %w", err)
	}
	return payload, nil
}
