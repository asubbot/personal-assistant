package llm

import (
	"os"
	"pa/internal/config"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewProvider_supportedTypes returns a provider for openai, openai-compatible, ollama (AC-015 unit: provider selected from config).
func TestNewProvider_supportedTypes(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key.txt")
	if err := os.WriteFile(keyPath, []byte("sk-test"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, typ := range []string{"openai", "openai-compatible", "ollama"} {
		t.Run(typ, func(t *testing.T) {
			cfg := &config.LLMProvider{
				Type:     typ,
				Endpoint: "https://api.example.com/v1",
				Model:    "gpt-4",
			}
			if typ != "ollama" {
				cfg.APIKeyPath = keyPath
			}
			prov, err := NewProvider(cfg)
			if err != nil {
				t.Fatalf("NewProvider(%q): %v", typ, err)
			}
			if prov == nil {
				t.Fatalf("NewProvider(%q): nil provider", typ)
			}
		})
	}
}

// TestNewProvider_unsupportedType returns error.
func TestNewProvider_unsupportedType(t *testing.T) {
	cfg := &config.LLMProvider{Type: "unknown", Endpoint: "http://x", Model: "m"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Fatal("NewProvider(unknown): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("NewProvider(unknown): error = %v (want 'unsupported')", err)
	}
}
