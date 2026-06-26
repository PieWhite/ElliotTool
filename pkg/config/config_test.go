package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_FromEnvVar(t *testing.T) {
	// Clean up environment after test
	origKey := os.Getenv("POLYGON_API_KEY")
	defer func() {
		_ = os.Setenv("POLYGON_API_KEY", origKey)
	}()

	testKey := "env_var_test_key"
	_ = os.Setenv("POLYGON_API_KEY", testKey)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if cfg.PolygonAPIKey != testKey {
		t.Errorf("expected API key %q, got %q", testKey, cfg.PolygonAPIKey)
	}
}

func TestLoad_MissingEnvVar(t *testing.T) {
	origKey := os.Getenv("POLYGON_API_KEY")
	defer func() {
		_ = os.Setenv("POLYGON_API_KEY", origKey)
	}()

	_ = os.Unsetenv("POLYGON_API_KEY")

	// Temporarily rename .env if it exists in the test runner directory so it doesn't load it
	if _, err := os.Stat(".env"); err == nil {
		_ = os.Rename(".env", ".env.tmp")
		defer func() {
			_ = os.Rename(".env.tmp", ".env")
		}()
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error when POLYGON_API_KEY is not set, got nil")
	}
}

func TestLoad_FromEnvFile(t *testing.T) {
	origKey := os.Getenv("POLYGON_API_KEY")
	defer func() {
		_ = os.Setenv("POLYGON_API_KEY", origKey)
	}()

	_ = os.Unsetenv("POLYGON_API_KEY")

	// Setup a temporary directory and temporary .env file
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	// Change working dir to temp dir
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}

	envContent := `
# This is a comment
POLYGON_API_KEY="env_file_test_key"
ANOTHER_VAR=123
`
	if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write temp .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error loading from .env file, got: %v", err)
	}

	if cfg.PolygonAPIKey != "env_file_test_key" {
		t.Errorf("expected API key %q, got %q", "env_file_test_key", cfg.PolygonAPIKey)
	}
}

func TestLoadEnvFile_Malformed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_malformed")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envPath := filepath.Join(tmpDir, ".env")
	envContent := `
INVALID_LINE_NO_EQUALS
POLYGON_API_KEY=valid_key
`
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	_ = os.Unsetenv("POLYGON_API_KEY")
	err = loadEnvFile(envPath)
	if err != nil {
		t.Fatalf("expected no error for malformed lines, got: %v", err)
	}

	if os.Getenv("POLYGON_API_KEY") != "valid_key" {
		t.Errorf("expected valid_key to be set, got %q", os.Getenv("POLYGON_API_KEY"))
	}
}
