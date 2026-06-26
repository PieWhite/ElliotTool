package repository

import (
	"context"
	"database/sql"
	"testing"

	"WaveSight/pkg/model"
)

func TestSQLiteCandleRepository_All(t *testing.T) {
	// Open connection to an isolated, in-memory SQLite database
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite DB: %v", err)
	}
	defer db.Close()

	repo := NewSQLiteCandleRepository(db)
	ctx := context.Background()

	// 1. Test Migrations
	if err := repo.Migrate(ctx); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	ticker := "AAPL"
	timeframe := "10m"

	// 2. Test Save & Retrieve
	testCandles := []model.Candle{
		{Time: 1000, Open: 150.0, High: 152.0, Low: 149.0, Close: 151.5, Volume: 10000},
		{Time: 2000, Open: 151.5, High: 153.0, Low: 151.0, Close: 152.8, Volume: 12000},
		{Time: 3000, Open: 152.8, High: 155.0, Low: 152.0, Close: 154.2, Volume: 15000},
	}

	err = repo.SaveCandles(ctx, ticker, timeframe, testCandles)
	if err != nil {
		t.Fatalf("SaveCandles failed: %v", err)
	}

	retrieved, err := repo.GetCandles(ctx, ticker, timeframe, 1000, 3000)
	if err != nil {
		t.Fatalf("GetCandles failed: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("expected 3 candles, got %d", len(retrieved))
	}

	for i, expected := range testCandles {
		actual := retrieved[i]
		if actual.Time != expected.Time {
			t.Errorf("candle[%d]: expected Time %d, got %d", i, expected.Time, actual.Time)
		}
		if actual.Open != expected.Open {
			t.Errorf("candle[%d]: expected Open %f, got %f", i, expected.Open, actual.Open)
		}
		if actual.High != expected.High {
			t.Errorf("candle[%d]: expected High %f, got %f", i, expected.High, actual.High)
		}
		if actual.Low != expected.Low {
			t.Errorf("candle[%d]: expected Low %f, got %f", i, expected.Low, actual.Low)
		}
		if actual.Close != expected.Close {
			t.Errorf("candle[%d]: expected Close %f, got %f", i, expected.Close, actual.Close)
		}
		if actual.Volume != expected.Volume {
			t.Errorf("candle[%d]: expected Volume %f, got %f", i, expected.Volume, actual.Volume)
		}
	}

	// 3. Test Range Filtering
	retrievedHalf, err := repo.GetCandles(ctx, ticker, timeframe, 1500, 2500)
	if err != nil {
		t.Fatalf("GetCandles in range failed: %v", err)
	}
	if len(retrievedHalf) != 1 {
		t.Fatalf("expected 1 candle in range [1500, 2500], got %d", len(retrievedHalf))
	}
	if retrievedHalf[0].Time != 2000 {
		t.Errorf("expected retrieved candle to have Time 2000, got %d", retrievedHalf[0].Time)
	}

	// 4. Test Upsert / Overwrite (Idempotence)
	updatedCandle := []model.Candle{
		{Time: 2000, Open: 151.5, High: 160.0, Low: 151.0, Close: 159.5, Volume: 22000}, // Modified High, Close, Volume
	}
	err = repo.SaveCandles(ctx, ticker, timeframe, updatedCandle)
	if err != nil {
		t.Fatalf("SaveCandles for upsert failed: %v", err)
	}

	retrievedAfterUpsert, err := repo.GetCandles(ctx, ticker, timeframe, 1000, 3000)
	if err != nil {
		t.Fatalf("GetCandles after upsert failed: %v", err)
	}

	if len(retrievedAfterUpsert) != 3 {
		t.Fatalf("expected 3 candles after upsert, got %d", len(retrievedAfterUpsert))
	}

	actualUpdated := retrievedAfterUpsert[1]
	if actualUpdated.High != 160.0 {
		t.Errorf("expected upserted High to be 160.0, got %f", actualUpdated.High)
	}
	if actualUpdated.Close != 159.5 {
		t.Errorf("expected upserted Close to be 159.5, got %f", actualUpdated.Close)
	}
	if actualUpdated.Volume != 22000 {
		t.Errorf("expected upserted Volume to be 22000, got %f", actualUpdated.Volume)
	}

	// 5. Test empty inputs and non-existent queries
	err = repo.SaveCandles(ctx, ticker, timeframe, nil)
	if err != nil {
		t.Errorf("expected saving nil slice to not fail, got: %v", err)
	}

	retrievedEmpty, err := repo.GetCandles(ctx, "MSFT", timeframe, 1000, 3000)
	if err != nil {
		t.Fatalf("GetCandles for non-existent ticker failed: %v", err)
	}
	if len(retrievedEmpty) != 0 {
		t.Errorf("expected 0 candles for non-existent ticker, got %d", len(retrievedEmpty))
	}
}
