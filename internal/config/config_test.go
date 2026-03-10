package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidConfig_NoError(t *testing.T) {
	path := filepath.Join("testdata", "valid_no_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(valid config): unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(valid config): got nil config")
	}
	if cfg.Version != 1 {
		t.Errorf("Load(valid config): version = %d, want 1", cfg.Version)
	}
	if cfg.Telegram.MaxMessageLength != 0 {
		t.Errorf("Load(valid config without max_message_length): MaxMessageLength = %d, want 0", cfg.Telegram.MaxMessageLength)
	}
	if cfg.Embedding == nil {
		t.Fatal("Load(valid config): embedding is required, expected non-nil")
	}
	if cfg.Embedding.Dimensions != 768 {
		t.Errorf("Load(valid config): embedding.dimensions = %d, want 768", cfg.Embedding.Dimensions)
	}
	if cfg.Embedding.Model != "nomic-embed-text" {
		t.Errorf("Load(valid config): embedding.model = %q, want nomic-embed-text", cfg.Embedding.Model)
	}
}

func TestLoad_TelegramMaxMessageLength(t *testing.T) {
	// Without field: 0
	pathNoField := filepath.Join("testdata", "valid_no_users.json")
	cfg, err := Load(pathNoField)
	if err != nil {
		t.Fatalf("Load(valid_no_users): %v", err)
	}
	if cfg.Telegram.MaxMessageLength != 0 {
		t.Errorf("MaxMessageLength without field = %d, want 0", cfg.Telegram.MaxMessageLength)
	}
	// With max_message_length: value loaded
	pathWithField := filepath.Join("testdata", "valid_max_message_length.json")
	cfg, err = Load(pathWithField)
	if err != nil {
		t.Fatalf("Load(valid_max_message_length): %v", err)
	}
	if cfg.Telegram.MaxMessageLength != 4096 {
		t.Errorf("MaxMessageLength = %d, want 4096", cfg.Telegram.MaxMessageLength)
	}
}

func TestLoad_ValidConfig_WithUsersFile_NoError(t *testing.T) {
	path := filepath.Join("testdata", "valid_with_good_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(valid config with users): unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(valid config with users): got nil config")
	}
}

// TestLoad_InvalidOrMissingFields covers AC-005: config validator with invalid/missing fields (test-strategy.md §3).
func TestLoad_InvalidOrMissingFields_ReturnsError(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		wantErr    string // substring that must appear in error, or empty to just require non-nil
	}{
		{"invalid version", "invalid_version.json", "version must be 1"},
		{"missing token_path", "missing_token_path.json", "telegram.token_path is required"},
		{"empty llm_providers", "empty_llm_providers.json", "at least one llm_providers"},
		{"missing llm type", "missing_llm_type.json", "llm_providers[0].type is required"},
		{"missing llm endpoint", "missing_llm_endpoint.json", "llm_providers[0].endpoint is required"},
		{"openai missing api_key_path", "openai_missing_api_key.json", "api_key_path is required"},
		{"missing paths.memory_dir", "missing_paths_memory_dir.json", "paths.memory_dir is required"},
		{"invalid host", "invalid_host.json", "nodes.node1.host is required"},
		{"missing auth", "invalid_auth.json", "nodes.node1.auth.private_key_path is required"},
		{"missing dedicated_user", "missing_dedicated_user.json", "nodes.n1.dedicated_user is required"},
		{"missing command_allowlist_path", "missing_command_allowlist.json", "nodes.n1.command_allowlist_path is required"},
		{"missing embedding", "missing_embedding.json", "embedding is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", tt.configFile)
			_, err := Load(path)
			if err == nil {
				t.Fatal("Load: expected error, got nil")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load: error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "nonexistent.json"))
	if err == nil {
		t.Fatal("Load(missing file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read config") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Load(missing file): error = %v (expect read/no such file)", err)
	}
}

func TestLoad_InvalidJSON_ReturnsError(t *testing.T) {
	path := filepath.Join("testdata", "not_valid_json.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(invalid JSON): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Load(invalid JSON): error = %v (expect parse/unmarshal related)", err)
	}
}

func TestLoad_UsersFileInvalidRole_ReturnsError(t *testing.T) {
	// Config points to invalid_users.json (role "superuser" not allowed)
	path := filepath.Join("testdata", "valid_with_users.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load(users file invalid role): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "role") && !strings.Contains(err.Error(), "user") {
		t.Errorf("Load(users file invalid): error = %v", err)
	}
}

func TestLoad_UsersFileNonexistent_ReturnsError(t *testing.T) {
	// Config with users_path pointing to a file that does not exist (relative to config dir)
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "config.json")
	usersPathRel := "nonexistent_users.json"
	content := `{
  "version": 1,
  "telegram": { "token_path": "/t", "users_path": "` + usersPathRel + `" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m" }],
  "paths": { "memory_dir": "/d", "log_path": "/d", "vector_index_path": "/d/pa_vectors.sqlite", "llm_log_dir": "/d", "scheduled_tasks_path": "" },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768 },
  "nodes": {}
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load(users file missing): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") && !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "read") {
		t.Errorf("Load(users file missing): error = %v", err)
	}
}
