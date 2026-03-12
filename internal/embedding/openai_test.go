package embedding

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// Supporting AC-013 (US-07): embedding provider construction — ollama-style config without API key succeeds.
func TestNewOpenAICompatible_validConfig_noAPIKey(t *testing.T) {
	cfg := &config.EmbeddingProvider{
		Type:       "ollama",
		Endpoint:   "http://localhost:11434",
		Model:      "nomic-embed-text",
		Dimensions: 768,
	}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	if p == nil {
		t.Fatal("NewOpenAICompatible: nil")
	}
}

// Covers AC-033 (US-19): embedding provider — empty model returns error (startup validation).
func TestNewOpenAICompatible_emptyModel_returnsError(t *testing.T) {
	cfg := &config.EmbeddingProvider{
		Type:       "ollama",
		Endpoint:   "http://localhost:11434",
		Model:      "",
		Dimensions: 768,
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil {
		t.Fatal("NewOpenAICompatible(empty model): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "model is required") {
		t.Errorf("NewOpenAICompatible: error = %v", err)
	}
}

// Covers AC-033 (US-19): embedding provider — missing API key file returns error (startup validation).
func TestNewOpenAICompatible_missingAPIKeyFile_returnsError(t *testing.T) {
	cfg := &config.EmbeddingProvider{
		Type:       "openai",
		Endpoint:   "https://api.openai.com/v1",
		APIKeyPath: filepath.Join(t.TempDir(), "nonexistent"),
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
	}
	_, err := NewOpenAICompatible(cfg)
	if err == nil {
		t.Fatal("NewOpenAICompatible(missing key file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read api_key_path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("NewOpenAICompatible: error = %v", err)
	}
}

// Supporting AC-013 (US-07): embedding provider — valid API key file yields non-nil provider.
func TestNewOpenAICompatible_validAPIKeyFile_succeeds(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.txt")
	if err := os.WriteFile(keyPath, []byte("sk-test"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg := &config.EmbeddingProvider{
		Type:       "openai",
		Endpoint:   "https://api.openai.com/v1",
		APIKeyPath: keyPath,
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
	}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}
	if p == nil {
		t.Fatal("NewOpenAICompatible: nil")
	}
}

// Supporting AC-013 (US-07): Embed success path for vector indexing.
func TestOpenAICompatible_Embed_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Errorf("path = %s, want /embeddings", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1, -0.2, 0.3],"index":0}]}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: server.URL, Model: "m", Dimensions: 3}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatalf("NewOpenAICompatible: %v", err)
	}

	vec, err := p.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("Embed: len = %d, want 3", len(vec))
	}
	if vec[0] != 0.1 || vec[1] != -0.2 || vec[2] != 0.3 {
		t.Errorf("Embed: vec = %v", vec)
	}
}

// Covers AC-037 (US-07): Embed error path — empty response data returns error; system does not crash.
func TestOpenAICompatible_Embed_emptyData_returnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: server.URL, Model: "m", Dimensions: 768}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed(empty data): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "empty data") {
		t.Errorf("Embed: error = %v", err)
	}
}

// Covers AC-037 (US-07): Embed error path — API error (e.g. 401) returns error; system does not crash.
func TestOpenAICompatible_Embed_apiError_returnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: server.URL, Model: "m", Dimensions: 768}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed(401): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("Embed: error = %v", err)
	}
}

// Covers AC-037 (US-07): Embed error path — invalid JSON returns error; system does not crash.
func TestOpenAICompatible_Embed_invalidJSON_returnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: server.URL, Model: "m", Dimensions: 768}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed(invalid JSON): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Embed: error = %v", err)
	}
}

// Covers AC-037 (US-07): Embed error path — canceled context returns error; system does not crash.
func TestOpenAICompatible_Embed_contextCanceled_returnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1]}]}`))
	}))
	defer server.Close()

	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: server.URL, Model: "m", Dimensions: 768}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = p.Embed(ctx, "hello")
	if err == nil {
		t.Fatal("Embed(canceled ctx): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("Embed: error = %v", err)
	}
}

// Covers AC-037 (US-07): Embed error path — unreachable server returns error; system does not crash.
func TestOpenAICompatible_Embed_serverUnreachable_returnsError(t *testing.T) {
	cfg := &config.EmbeddingProvider{Type: "ollama", Endpoint: "http://127.0.0.1:19999", Model: "m", Dimensions: 768}
	p, err := NewOpenAICompatible(cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("Embed(unreachable): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "request") && !strings.Contains(err.Error(), "connection") && !strings.Contains(err.Error(), "refused") {
		t.Logf("Embed: error = %v", err)
	}
}
