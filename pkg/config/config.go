package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	PolygonAPIKey string
}

// Load reads configuration from the environment.
// It attempts to load a `.env` file in the current directory if it exists,
// but does not fail if the file is missing (as variables might be provided directly in the environment).
func Load() (*Config, error) {
	// Try loading .env file if it exists, ignoring errors
	_ = loadEnvFile(".env")

	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("POLYGON_API_KEY environment variable is not set or empty")
	}

	return &Config{
		PolygonAPIKey: apiKey,
	}, nil
}

// loadEnvFile parses a .env file and sets environment variables if they are not already set.
func loadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip quotes if they surround the value
		val = strings.Trim(val, `"'`)

		// Only set the variable if it doesn't already exist in the environment
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("setting env var %s: %w", key, err)
			}
		}
	}

	return scanner.Err()
}
