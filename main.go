package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"WaveSight/pkg/api"
	"WaveSight/pkg/config"
	"WaveSight/pkg/polygon"
	"WaveSight/pkg/repository"
	"WaveSight/pkg/swing"
)

func main() {
	log.Println("Starting WaveSight backend...")

	// 1. Load config (.env or environment)
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Open SQLite Database
	db, err := sql.Open("sqlite", "candles.db")
	if err != nil {
		log.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// 3. Create repository and run migrations
	repo := repository.NewSQLiteCandleRepository(db)
	ctx := context.Background()
	if err := repo.Migrate(ctx); err != nil {
		log.Fatalf("Failed to migrate SQLite database: %v", err)
	}

	// 4. Create Polygon client
	polygonClient := polygon.NewClient(cfg.PolygonAPIKey, &http.Client{})

	// 5. Initialize API Handler with volatility-adaptive swing detector (14-period ATR)
	detector := swing.NewVolatilitySwingDetector(14)
	handler := api.NewHandler(polygonClient, repo, detector)

	// 6. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}
