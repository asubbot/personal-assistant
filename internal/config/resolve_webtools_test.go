package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Covers AC-11.003 / REQ-11.006 — brave key path resolved against PA_SECRETS_DIR.
func TestResolvePaths_WebToolsBraveKeyRelative(t *testing.T) {
	tmp := t.TempDir()
	prev := os.Getenv("PA_SECRETS_DIR")
	_ = os.Setenv("PA_SECRETS_DIR", tmp)
	t.Cleanup(func() { _ = os.Setenv("PA_SECRETS_DIR", prev) })

	cfg := &Config{
		WebTools: &WebToolsConfig{
			Enabled: true,
			Search: WebSearchConfig{
				BraveAPIKeyPath: "brave.key",
			},
		},
	}
	ResolvePaths(cfg, filepath.Join("/etc", "pa", "config.json"))
	got := cfg.WebTools.Search.BraveAPIKeyPath
	want := filepath.Join(tmp, "brave.key")
	if got != want {
		t.Fatalf("BraveAPIKeyPath = %q, want %q", got, want)
	}
}
