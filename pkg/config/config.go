package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PolygonAPIKey      string
	ProviderBaseURL    string
	Port               string
	DatabasePath       string
	FrontendDir        string
	AllowedOrigins     []string
	MaxConcurrentScans int
	ProviderTimeout    time.Duration
}

func Load() (*Config, error) {
	if err := loadEnvFile(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading .env: %w", err)
	}
	apiKey := strings.TrimSpace(os.Getenv("POLYGON_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("POLYGON_API_KEY environment variable is not set or empty")
	}

	maxConcurrent := 4
	if value := strings.TrimSpace(os.Getenv("MAX_CONCURRENT_SCANS")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 32 {
			return nil, fmt.Errorf("MAX_CONCURRENT_SCANS must be between 1 and 32")
		}
		maxConcurrent = parsed
	}
	timeout := 30 * time.Second
	if value := strings.TrimSpace(os.Getenv("PROVIDER_TIMEOUT")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed < time.Second {
			return nil, fmt.Errorf("invalid PROVIDER_TIMEOUT")
		}
		timeout = parsed
	}

	return &Config{
		PolygonAPIKey:      apiKey,
		ProviderBaseURL:    envOrDefault("PROVIDER_BASE_URL", "https://api.massive.com"),
		Port:               envOrDefault("PORT", "8080"),
		DatabasePath:       envOrDefault("DATABASE_PATH", "wavesight.db"),
		FrontendDir:        envOrDefault("FRONTEND_DIR", "frontend/dist"),
		AllowedOrigins:     splitCSV(envOrDefault("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		MaxConcurrentScans: maxConcurrent,
		ProviderTimeout:    timeout,
	}, nil
}

func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("setting %s: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanning %s: %w", filename, err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
