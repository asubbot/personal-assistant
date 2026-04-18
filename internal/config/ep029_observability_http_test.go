package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-29.001: observability_http block loads when all fields are explicit.
func TestLoad_ValidObservabilityHTTP(t *testing.T) {
	path := filepath.Join("testdata", "valid_observability_http.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ObservabilityHTTP == nil {
		t.Fatal("expected observability_http")
	}
	if cfg.ObservabilityHTTP.ListenAddress != "127.0.0.1:19090" {
		t.Errorf("listen_address = %q", cfg.ObservabilityHTTP.ListenAddress)
	}
	if cfg.ObservabilityHTTP.ProbeLLM {
		t.Error("probe_llm should be false")
	}
}

// Covers AC-29.001: invalid observability paths rejected.
// Covers AC-29.006: omitting observability_http does not create an implicit listener config.
func TestLoad_ValidNoUsers_ObservabilityHTTPNil(t *testing.T) {
	path := filepath.Join("testdata", "valid_no_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ObservabilityHTTP != nil {
		t.Fatalf("expected nil observability_http, got %+v", cfg.ObservabilityHTTP)
	}
}

// Covers AC-29.001: invalid observability_http paths rejected at load.
func TestLoad_InvalidObservabilityHTTPPaths(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid_observability_http_same_paths.json"))
	if err == nil {
		t.Fatal("expected error for identical health and readiness paths")
	}
	if !strings.Contains(err.Error(), "must differ") {
		t.Fatalf("error = %v", err)
	}
}

// Covers AC-29.001: whitespace-only listen_address rejected.
func TestLoad_InvalidObservabilityHTTPEmptyListen(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid_observability_http_empty_listen.json"))
	if err == nil {
		t.Fatal("expected error for empty listen_address")
	}
	if !strings.Contains(err.Error(), "listen_address") {
		t.Fatalf("error = %v", err)
	}
}

// Covers AC-29.001: health_path must start with /.
func TestLoad_InvalidObservabilityHTTPRelativeHealthPath(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid_observability_http_relative_health_path.json"))
	if err == nil {
		t.Fatal("expected error for health_path not starting with /")
	}
	if !strings.Contains(err.Error(), "health_path") {
		t.Fatalf("error = %v", err)
	}
}
