package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"WaveSight/internal/domain/wave"
	"WaveSight/internal/market"
	"WaveSight/pkg/api"
	"WaveSight/pkg/config"
	"WaveSight/pkg/polygon"
	"WaveSight/pkg/repository"
)

func main() {
	if err := run(); err != nil {
		log.Printf("WaveSight stopped: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}
	db, err := sql.Open("sqlite", cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("opening SQLite: %w", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("closing SQLite: %v", closeErr)
		}
	}()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := repository.NewSQLiteStore(db)
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	calendar, err := market.NewUSCalendar()
	if err != nil {
		return err
	}
	httpClient := &http.Client{Timeout: cfg.ProviderTimeout}
	provider := polygon.NewClient(cfg.PolygonAPIKey, httpClient)
	provider.SetBaseURL(cfg.ProviderBaseURL)
	handler := api.NewHandler(
		provider,
		store,
		wave.NewEngine(),
		calendar,
		api.HandlerConfig{
			AllowedOrigins: cfg.AllowedOrigins, MaxConcurrentScans: cfg.MaxConcurrentScans,
			StaticDir: cfg.FrontendDir,
		},
	)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      45 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("WaveSight %s listening on http://localhost:%s", wave.EngineVersion, cfg.Port)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutting down HTTP server: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serving HTTP: %w", err)
	}
}
