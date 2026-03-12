package embedding

import (
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-033 (US-19): NewEmbedder(nil config) returns error (startup validation).
func TestNewEmbedder_nilConfig_returnsError(t *testing.T) {
	_, err := NewEmbedder(nil)
	if err == nil {
		t.Fatal("NewEmbedder(nil): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "config is nil") {
		t.Errorf("NewEmbedder(nil): error = %v", err)
	}
}

// Covers AC-033 (US-19): NewEmbedder(unsupported type) returns error (startup validation).
func TestNewEmbedder_unsupportedType_returnsError(t *testing.T) {
	cfg := &config.EmbeddingProvider{
		Type:       "custom",
		Endpoint:   "http://localhost",
		Model:      "m",
		Dimensions: 768,
	}
	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Fatal("NewEmbedder(unsupported type): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported type") && !strings.Contains(err.Error(), "supported:") {
		t.Errorf("NewEmbedder: error = %v", err)
	}
}

// Supporting AC-013, AC-014 (US-07): NewEmbedder(supported types) returns non-nil embedder.
func TestNewEmbedder_supportedTypes_returnsEmbedder(t *testing.T) {
	types := []string{"openai", "openai-compatible", "ollama", "OpenAI", "OLLAMA"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			cfg := &config.EmbeddingProvider{
				Type:       typ,
				Endpoint:   "http://localhost:11434",
				Model:      "nomic-embed-text",
				Dimensions: 768,
			}
			emb, err := NewEmbedder(cfg)
			if err != nil {
				t.Fatalf("NewEmbedder(%q): %v", typ, err)
			}
			if emb == nil {
				t.Fatalf("NewEmbedder(%q): nil embedder", typ)
			}
		})
	}
}

// Covers AC-033 (US-19): NewEmbedder(openai, missing API key file) returns error (startup validation).
func TestNewEmbedder_openaiWithAPIKeyPath_missingFile_returnsError(t *testing.T) {
	cfg := &config.EmbeddingProvider{
		Type:       "openai",
		Endpoint:   "https://api.openai.com/v1",
		APIKeyPath: filepath.Join(t.TempDir(), "nonexistent"),
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
	}
	_, err := NewEmbedder(cfg)
	if err == nil {
		t.Fatal("NewEmbedder(openai, missing key file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "api_key_path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("NewEmbedder: error = %v", err)
	}
}
