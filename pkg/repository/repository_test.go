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

func TestSQLiteStoreV3NativeCoverageAndImmutableRevisions(t *testing.T) {
	t.Parallel()
	store := testStore(t)
	ctx := context.Background()
	candles := []market.Candle{
		{Time: 1_000, Open: 10, High: 12, Low: 9, Close: 11, Volume: 100},
		{Time: 2_000, Open: 11, High: 13, Low: 10, Close: 12, Volume: 120},
	}
	changed, err := store.SaveNativeCandles(ctx, "AAPL", "MINUTE_NATIVE", candles)
	if err != nil || changed {
		t.Fatalf("first native save changed/error = %t/%v", changed, err)
	}
	candles[1].High = 14
	changed, err = store.SaveNativeCandles(ctx, "AAPL", "MINUTE_NATIVE", candles[1:])
	if err != nil || !changed {
		t.Fatalf("provider correction changed/error = %t/%v", changed, err)
	}
	if err := store.SaveNativeCoverage(ctx, "AAPL", "MINUTE_NATIVE", 500, 1_500); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveNativeCoverage(ctx, "AAPL", "MINUTE_NATIVE", 1_501, 3_000); err != nil {
		t.Fatal(err)
	}
	coverage, err := store.NativeCoverage(ctx, "AAPL", "MINUTE_NATIVE")
	if err != nil || len(coverage) != 1 || coverage[0].From != 500 || coverage[0].To != 3_000 {
		t.Fatalf("merged coverage = %+v, %v", coverage, err)
	}

	parent := SnapshotMetadataV3{
		ID: "parent", RequestKey: "request-parent", DataFingerprint: "data-1",
		Symbol: "AAPL", Session: "RTH", AsOf: 3_000, FocusTimeframe: "1D",
		GeneratedAt: 4_000, TheoryVersion: "theory", EngineVersion: "3",
	}
	if err := store.SaveSnapshotV3(
		ctx, parent, []byte(`{"id":"parent"}`),
		map[market.Timeframe][]byte{market.Timeframe1D: []byte(`{"timeframe":"1D"}`)},
		map[string][]byte{"event-1": []byte(`{"id":"event-1"}`)},
		map[string][]byte{"wave-1": []byte(`{"id":"wave-1"}`)},
		nil,
		[]RankedPayload{{ID: "scenario-1", Rank: 1, Payload: []byte(`{"id":"scenario-1"}`)}},
	); err != nil {
		t.Fatal(err)
	}
	child := parent
	child.ID = "child"
	child.ParentSnapshotID = parent.ID
	child.RequestKey = "request-child"
	child.DataFingerprint = "data-2"
	child.GeneratedAt++
	if err := store.SaveSnapshotV3(
		ctx, child, []byte(`{"id":"child"}`),
		map[market.Timeframe][]byte{market.Timeframe1D: []byte(`{"timeframe":"1D","revision":true}`)},
		nil, nil, nil, nil,
	); err != nil {
		t.Fatal(err)
	}
	parentPayload, _ := store.GetSnapshotV3(ctx, parent.ID)
	childPayload, _ := store.GetSnapshotV3(ctx, child.ID)
	if string(parentPayload) != `{"id":"parent"}` || string(childPayload) != `{"id":"child"}` {
		t.Fatalf("revision mutated parent: %s / %s", parentPayload, childPayload)
	}
	view, err := store.GetViewV3(ctx, child.ID, market.Timeframe1D)
	if err != nil || string(view) != `{"timeframe":"1D","revision":true}` {
		t.Fatalf("child view = %s, %v", view, err)
	}
	items, err := store.ListSnapshotsV3(ctx, 20)
	if err != nil || len(items) != 2 || items[0].ParentSnapshotID != parent.ID {
		t.Fatalf("v3 history = %+v, %v", items, err)
	}
}
