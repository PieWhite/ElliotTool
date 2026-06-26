package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"

	"WaveSight/pkg/model"
)

// CandleRepository defines the interface for persisting and querying market candle data.
type CandleRepository interface {
	SaveCandles(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error
	GetCandles(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error)
}

// SQLiteCandleRepository implements CandleRepository using a SQLite database.
type SQLiteCandleRepository struct {
	db *sql.DB
}

// NewSQLiteCandleRepository creates a new SQLiteCandleRepository.
func NewSQLiteCandleRepository(db *sql.DB) *SQLiteCandleRepository {
	return &SQLiteCandleRepository{
		db: db,
	}
}

// Migrate creates the required database tables and indexes if they do not exist.
func (r *SQLiteCandleRepository) Migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS candles (
		ticker TEXT NOT NULL,
		timeframe TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		open REAL NOT NULL,
		high REAL NOT NULL,
		low REAL NOT NULL,
		close REAL NOT NULL,
		volume REAL NOT NULL,
		PRIMARY KEY (ticker, timeframe, timestamp)
	);
	`
	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("migrating sqlite database: %w", err)
	}
	return nil
}

// SaveCandles writes multiple candles to the SQLite database.
// It uses a transaction and a prepared statement to ensure high-performance bulk insertion.
// It uses "ON CONFLICT DO UPDATE" to overwrite existing candles with updated values.
func (r *SQLiteCandleRepository) SaveCandles(ctx context.Context, ticker string, timeframe string, candles []model.Candle) error {
	if len(candles) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO candles (ticker, timeframe, timestamp, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(ticker, timeframe, timestamp) DO UPDATE SET
			open=excluded.open,
			high=excluded.high,
			low=excluded.low,
			close=excluded.close,
			volume=excluded.volume
	`)
	if err != nil {
		return fmt.Errorf("preparing insert statement: %w", err)
	}
	defer stmt.Close()

	for _, c := range candles {
		_, err := stmt.ExecContext(ctx, ticker, timeframe, c.Time, c.Open, c.High, c.Low, c.Close, c.Volume)
		if err != nil {
			return fmt.Errorf("inserting candle at timestamp %d: %w", c.Time, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetCandles retrieves historical candles within a timestamp range (inclusive), sorted chronologically.
func (r *SQLiteCandleRepository) GetCandles(ctx context.Context, ticker string, timeframe string, from int64, to int64) ([]model.Candle, error) {
	query := `
		SELECT timestamp, open, high, low, close, volume
		FROM candles
		WHERE ticker = ? AND timeframe = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`
	rows, err := r.db.QueryContext(ctx, query, ticker, timeframe, from, to)
	if err != nil {
		return nil, fmt.Errorf("querying candles: %w", err)
	}
	defer rows.Close()

	// Start with a reasonable capacity to minimize slice growth overhead
	candles := make([]model.Candle, 0, 1024)

	for rows.Next() {
		var c model.Candle
		err := rows.Scan(&c.Time, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume)
		if err != nil {
			return nil, fmt.Errorf("scanning candle row: %w", err)
		}
		candles = append(candles, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating candle rows: %w", err)
	}

	return candles, nil
}
