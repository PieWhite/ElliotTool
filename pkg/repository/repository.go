package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/glebarez/go-sqlite"

	"WaveSight/internal/market"
)

var ErrSnapshotNotFound = errors.New("analysis snapshot not found")

type Store interface {
	Migrate(ctx context.Context) error
	SaveCandles(ctx context.Context, ticker string, timeframe market.Timeframe, candles []market.Candle) error
	GetCandles(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) ([]market.Candle, error)
	HasCoverage(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) (bool, error)
	SaveCoverage(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) error
	SaveSnapshot(ctx context.Context, metadata SnapshotMetadata, payload []byte) error
	GetSnapshot(ctx context.Context, id string) ([]byte, error)
	ListSnapshots(ctx context.Context, limit int) ([]SnapshotMetadata, error)
}

//easyjson:json
type SnapshotMetadata struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	Timeframe       string `json:"timeframe"`
	Session         string `json:"session"`
	AsOf            int64  `json:"as_of"`
	GeneratedAt     int64  `json:"generated_at"`
	TheoryVersion   string `json:"theory_version"`
	EngineVersion   string `json:"engine_version"`
	RequestHash     string `json:"request_hash"`
	DataFingerprint string `json:"data_fingerprint"`
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) Migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA foreign_keys=ON`,
		`PRAGMA busy_timeout=5000`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL DEFAULT (unixepoch())
		)`,
		`CREATE TABLE IF NOT EXISTS candles (
			ticker TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			open REAL NOT NULL,
			high REAL NOT NULL,
			low REAL NOT NULL,
			close REAL NOT NULL,
			volume REAL NOT NULL,
			PRIMARY KEY (ticker, timeframe, timestamp)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_candles_range
			ON candles(ticker, timeframe, timestamp)`,
		`CREATE TABLE IF NOT EXISTS candle_coverage (
			ticker TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			from_timestamp INTEGER NOT NULL,
			to_timestamp INTEGER NOT NULL,
			PRIMARY KEY (ticker, timeframe, from_timestamp, to_timestamp)
		)`,
		`CREATE TABLE IF NOT EXISTS analysis_snapshots (
			id TEXT PRIMARY KEY,
			request_hash TEXT NOT NULL,
			data_fingerprint TEXT NOT NULL,
			symbol TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			session TEXT NOT NULL,
			as_of INTEGER NOT NULL,
			generated_at INTEGER NOT NULL,
			theory_version TEXT NOT NULL,
			engine_version TEXT NOT NULL,
			payload BLOB NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_snapshot_history
			ON analysis_snapshots(generated_at DESC, symbol, timeframe)`,
		`CREATE TABLE IF NOT EXISTS analysis_index (
			symbol TEXT NOT NULL,
			timeframe TEXT NOT NULL,
			session TEXT NOT NULL,
			as_of INTEGER NOT NULL,
			snapshot_id TEXT NOT NULL,
			PRIMARY KEY (symbol, timeframe, session, as_of, snapshot_id),
			FOREIGN KEY (snapshot_id) REFERENCES analysis_snapshots(id) ON DELETE RESTRICT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_analysis_lookup
			ON analysis_index(symbol, timeframe, session, as_of DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrating WaveSight database: %w", err)
		}
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "request_hash", definition: `TEXT NOT NULL DEFAULT ''`},
		{name: "data_fingerprint", definition: `TEXT NOT NULL DEFAULT ''`},
	} {
		if err := s.ensureColumn(ctx, "analysis_snapshots", column.name, column.definition); err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version) VALUES (2)`); err != nil {
		return fmt.Errorf("recording schema migration: %w", err)
	}
	return nil
}

func (s *SQLiteStore) ensureColumn(ctx context.Context, table, column, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return fmt.Errorf("inspecting %s schema: %w", table, err)
	}
	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("scanning %s schema: %w", table, err)
		}
		if name == column {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterating %s schema: %w", table, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("closing %s schema inspection: %w", table, err)
	}
	if found {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+column+` `+definition); err != nil {
		return fmt.Errorf("adding %s.%s: %w", table, column, err)
	}
	return nil
}

func (s *SQLiteStore) SaveCandles(ctx context.Context, ticker string, timeframe market.Timeframe, candles []market.Candle) error {
	if len(candles) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning candle transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO candles (ticker, timeframe, timestamp, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ticker, timeframe, timestamp) DO UPDATE SET
			open=excluded.open, high=excluded.high, low=excluded.low,
			close=excluded.close, volume=excluded.volume
	`)
	if err != nil {
		return fmt.Errorf("preparing candle upsert: %w", err)
	}
	defer func() {
		_ = stmt.Close()
	}()

	for _, candle := range candles {
		if _, err := stmt.ExecContext(
			ctx, ticker, string(timeframe), candle.Time, candle.Open, candle.High,
			candle.Low, candle.Close, candle.Volume,
		); err != nil {
			return fmt.Errorf("upserting candle %d: %w", candle.Time, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing candle transaction: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetCandles(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) ([]market.Candle, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT timestamp, open, high, low, close, volume
		FROM candles
		WHERE ticker = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`, ticker, string(timeframe), from, to)
	if err != nil {
		return nil, fmt.Errorf("querying candles: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	candles := make([]market.Candle, 0, 8_192)
	for rows.Next() {
		var candle market.Candle
		if err := rows.Scan(
			&candle.Time, &candle.Open, &candle.High, &candle.Low, &candle.Close, &candle.Volume,
		); err != nil {
			return nil, fmt.Errorf("scanning candle: %w", err)
		}
		candles = append(candles, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating candles: %w", err)
	}
	return candles, nil
}

func (s *SQLiteStore) HasCoverage(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM candle_coverage
		WHERE ticker = ? AND timeframe = ?
		  AND from_timestamp <= ? AND to_timestamp >= ?
	`, ticker, string(timeframe), from, to).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking candle coverage: %w", err)
	}
	return count > 0, nil
}

func (s *SQLiteStore) SaveCoverage(ctx context.Context, ticker string, timeframe market.Timeframe, from, to int64) error {
	if _, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO candle_coverage
			(ticker, timeframe, from_timestamp, to_timestamp)
		VALUES (?, ?, ?, ?)
	`, ticker, string(timeframe), from, to); err != nil {
		return fmt.Errorf("saving candle coverage: %w", err)
	}
	return nil
}

func (s *SQLiteStore) SaveSnapshot(ctx context.Context, metadata SnapshotMetadata, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("saving snapshot %s: empty payload", metadata.ID)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning snapshot transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO analysis_snapshots
			(id, request_hash, data_fingerprint, symbol, timeframe, session, as_of,
			 generated_at, theory_version, engine_version, payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, metadata.ID, metadata.RequestHash, metadata.DataFingerprint,
		metadata.Symbol, metadata.Timeframe, metadata.Session, metadata.AsOf,
		metadata.GeneratedAt, metadata.TheoryVersion, metadata.EngineVersion, payload); err != nil {
		return fmt.Errorf("saving analysis snapshot: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO analysis_index
			(symbol, timeframe, session, as_of, snapshot_id)
		VALUES (?, ?, ?, ?, ?)
	`, metadata.Symbol, metadata.Timeframe, metadata.Session, metadata.AsOf, metadata.ID); err != nil {
		return fmt.Errorf("indexing analysis snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing analysis snapshot: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetSnapshot(ctx context.Context, id string) ([]byte, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT payload FROM analysis_snapshots WHERE id = ?
	`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSnapshotNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("reading analysis snapshot: %w", err)
	}
	return payload, nil
}

func (s *SQLiteStore) ListSnapshots(ctx context.Context, limit int) ([]SnapshotMetadata, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, symbol, timeframe, session, as_of, generated_at, theory_version, engine_version
		FROM analysis_snapshots
		ORDER BY generated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("listing analysis snapshots: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make([]SnapshotMetadata, 0, limit)
	for rows.Next() {
		var metadata SnapshotMetadata
		if err := rows.Scan(
			&metadata.ID, &metadata.Symbol, &metadata.Timeframe, &metadata.Session,
			&metadata.AsOf, &metadata.GeneratedAt, &metadata.TheoryVersion, &metadata.EngineVersion,
		); err != nil {
			return nil, fmt.Errorf("scanning snapshot metadata: %w", err)
		}
		result = append(result, metadata)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating snapshot metadata: %w", err)
	}
	return result, nil
}
