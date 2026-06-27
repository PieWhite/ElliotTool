package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"WaveSight/internal/market"
)

func testStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("db.Close() error = %v", err)
		}
	})
	store := NewSQLiteStore(db)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	return store
}

func TestSQLiteStoreCandlesAndCoverage(t *testing.T) {
	t.Parallel()
	store := testStore(t)
	ctx := context.Background()
	candles := []market.Candle{
		{Time: 1_000, Open: 100, High: 103, Low: 99, Close: 102, Volume: 10_000},
		{Time: 2_000, Open: 102, High: 106, Low: 101, Close: 105, Volume: 12_000},
		{Time: 3_000, Open: 105, High: 107, Low: 102, Close: 103, Volume: 11_000},
	}
	if err := store.SaveCandles(ctx, "AAPL", market.Timeframe1D, candles); err != nil {
		t.Fatalf("SaveCandles() error = %v", err)
	}

	got, err := store.GetCandles(ctx, "AAPL", market.Timeframe1D, 1_500, 3_000)
	if err != nil {
		t.Fatalf("GetCandles() error = %v", err)
	}
	if len(got) != 2 || got[0].Time != 2_000 || got[1].Close != 103 {
		t.Fatalf("GetCandles() = %+v", got)
	}

	candles[1].High = 110
	if err := store.SaveCandles(ctx, "AAPL", market.Timeframe1D, candles[1:2]); err != nil {
		t.Fatalf("SaveCandles(upsert) error = %v", err)
	}
	got, err = store.GetCandles(ctx, "AAPL", market.Timeframe1D, 2_000, 2_000)
	if err != nil || len(got) != 1 || got[0].High != 110 {
		t.Fatalf("upsert result = %+v, error = %v", got, err)
	}

	covered, err := store.HasCoverage(ctx, "AAPL", market.Timeframe1D, 1_000, 3_000)
	if err != nil || covered {
		t.Fatalf("HasCoverage() before save = %t, %v", covered, err)
	}
	if err := store.SaveCoverage(ctx, "AAPL", market.Timeframe1D, 500, 3_500); err != nil {
		t.Fatalf("SaveCoverage() error = %v", err)
	}
	covered, err = store.HasCoverage(ctx, "AAPL", market.Timeframe1D, 1_000, 3_000)
	if err != nil || !covered {
		t.Fatalf("HasCoverage() after save = %t, %v", covered, err)
	}
}

func TestSQLiteStoreSnapshotsAreImmutableAndListed(t *testing.T) {
	t.Parallel()
	store := testStore(t)
	ctx := context.Background()
	metadata := SnapshotMetadata{
		ID: "0123456789abcdef0123456789abcdef", Symbol: "AAPL", Timeframe: "1D",
		Session: "RTH", AsOf: 1_000, GeneratedAt: 2_000,
		TheoryVersion: "theory-1", EngineVersion: "engine-1",
	}
	first := []byte(`{"id":"first"}`)
	if err := store.SaveSnapshot(ctx, metadata, first); err != nil {
		t.Fatalf("SaveSnapshot() error = %v", err)
	}
	if err := store.SaveSnapshot(ctx, metadata, []byte(`{"id":"replacement"}`)); err != nil {
		t.Fatalf("SaveSnapshot(duplicate) error = %v", err)
	}
	got, err := store.GetSnapshot(ctx, metadata.ID)
	if err != nil || string(got) != string(first) {
		t.Fatalf("GetSnapshot() = %s, %v; immutable payload changed", got, err)
	}
	items, err := store.ListSnapshots(ctx, 20)
	if err != nil || len(items) != 1 || items[0].ID != metadata.ID {
		t.Fatalf("ListSnapshots() = %+v, %v", items, err)
	}
	_, err = store.GetSnapshot(ctx, "missing")
	if !errors.Is(err, ErrSnapshotNotFound) {
		t.Fatalf("GetSnapshot(missing) error = %v", err)
	}
}
