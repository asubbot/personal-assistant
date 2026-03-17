package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Supporting AC-01.005 (US-03): valid config with empty or omitted scheduled_tasks_path loads.
func TestLoad_ValidConfig_EmptyScheduledTasksPath(t *testing.T) {
	path := filepath.Join("testdata", "valid_no_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(config with empty scheduled_tasks_path): unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(config with empty scheduled_tasks_path): got nil config")
	}
	if cfg.Paths.ScheduledTasksPath != "" {
		t.Errorf("Load(config with empty scheduled_tasks_path): ScheduledTasksPath = %q, want empty", cfg.Paths.ScheduledTasksPath)
	}
}

// TestLoad_ValidConfig_NoError — Supporting test for AC-01.005 (US-03): valid config loads without error.
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

// TestLoad_TelegramMaxMessageLength — Supporting test for AC-01.002 (US-01): max_message_length from config (0 when omitted, value when set).
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

// Supporting AC-01.005 (US-03): valid config with users_path loads.
func TestLoad_ValidConfig_WithUsersFile_NoError(t *testing.T) {
	// users_path "testdata/good_users.json" is resolved with PA_SECRETS_DIR; use "." so path is testdata/good_users.json.
	prev := os.Getenv("PA_SECRETS_DIR")
	_ = os.Setenv("PA_SECRETS_DIR", ".")
	t.Cleanup(func() { _ = os.Setenv("PA_SECRETS_DIR", prev) })
	path := filepath.Join("testdata", "valid_with_good_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(valid config with users): unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(valid config with users): got nil config")
	}
}

// TestLoad_InvalidOrMissingFields covers AC-01.005: config validator with invalid/missing fields (test-strategy.md §3).
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
		{"invalid pa_timezone", "invalid_pa_timezone.json", "invalid pa_timezone"},
		{"llm_log_retention_days < 1", "llm_log_retention_zero.json", "llm_log_retention_days must be >= 1"},
		{"nodes without ssh_known_hosts_path", "nodes_missing_ssh_known_hosts_path.json", "paths.ssh_known_hosts_path is required when nodes are configured"},
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

// Covers AC-01.005 (US-03): config file missing returns clear error (refuse to start or report error).
func TestLoad_MissingFile_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "nonexistent.json"))
	if err == nil {
		t.Fatal("Load(missing file): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read config") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Load(missing file): error = %v (expect read/no such file)", err)
	}
}

// Covers AC-01.005 (US-03): config invalid JSON returns clear error.
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

// Covers AC-01.005 (US-03): referenced users file with invalid role returns clear error.
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

// TestLoad_LogRedactionReservedID_ReturnsError — REQ-01.029: reserved additional pattern id refuses start.
// Covers AC-01.041 (US-16): log_redaction reserved pattern identifier or invalid regex → refuse start, clear error.
func TestLoad_LogRedactionReservedID_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "log_redaction_reserved_id.json"))
	if err == nil {
		t.Fatal("Load(log_redaction reserved id): expected error")
	}
	if !strings.Contains(err.Error(), "log_redaction") || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("Load: error = %v (expect log_redaction reserved message)", err)
	}
}

// TestLoad_LogRedactionInvalidRegex_ReturnsError — REQ-01.029: invalid regex in additional pattern refuses start.
// Covers AC-01.041 (US-16): log_redaction invalid regex → refuse start, clear error.
func TestLoad_LogRedactionInvalidRegex_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "log_redaction_invalid_regex.json"))
	if err == nil {
		t.Fatal("Load(log_redaction invalid regex): expected error")
	}
	if !strings.Contains(err.Error(), "log_redaction") || !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("Load: error = %v (expect log_redaction invalid regex message)", err)
	}
}

// Supporting AC-01.033 (US-19): invalid IANA timezone in pa_timezone refuses start.
func TestLoad_InvalidPATimezone_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid_pa_timezone.json"))
	if err == nil {
		t.Fatal("Load(invalid pa_timezone): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "pa_timezone") {
		t.Errorf("Load(invalid pa_timezone): error = %v (expect pa_timezone in message)", err)
	}
}

// Supporting AC-01.033 (US-19): valid pa_timezone (e.g. Europe/Moscow, UTC) loads successfully.
func TestLoad_ValidPATimezone_loads(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "valid_pa_timezone.json"))
	if err != nil {
		t.Fatalf("Load(valid_pa_timezone): %v", err)
	}
	if cfg.PATimezone != "Europe/Moscow" {
		t.Errorf("PATimezone = %q, want Europe/Moscow", cfg.PATimezone)
	}
	// Empty/omitted pa_timezone loads (valid_no_users.json has no pa_timezone)
	cfg2, err := Load(filepath.Join("testdata", "valid_no_users.json"))
	if err != nil {
		t.Fatalf("Load(valid_no_users): %v", err)
	}
	if cfg2.PATimezone != "" {
		t.Errorf("PATimezone when omitted = %q, want empty", cfg2.PATimezone)
	}
}

// Covers AC-01.005 (US-03): users_path points to nonexistent file returns clear error.
func TestLoad_UsersFileNonexistent_ReturnsError(t *testing.T) {
	// Config with users_path pointing to a file that does not exist (path is CWD-relative)
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "config.json")
	usersPathRel := "nonexistent_users.json"
	content := `{
  "version": 1,
  "telegram": { "token_path": "/t", "users_path": "` + usersPathRel + `" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m" }],
  "paths": { "memory_dir": "/d", "log_path": "/d", "vector_index_path": "/d/pa_vectors.sqlite", "llm_log_dir": "/d", "llm_log_retention_days": 7, "scheduled_tasks_path": "" },
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

// Covers SSH known_hosts: when nodes are configured, ssh_known_hosts_path must point to an existing file.
func TestLoad_NodesWithNonexistentSSHKnownHostsFile_ReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "config.json")
	knownHostsRel := "nonexistent_known_hosts"
	content := `{
  "version": 1,
  "telegram": { "token_path": "/t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m" }],
  "paths": {
    "memory_dir": "` + cfgDir + `",
    "log_path": "` + cfgDir + `/pa.log",
    "vector_index_path": "` + cfgDir + `/pa_vectors.sqlite",
    "llm_log_dir": "` + cfgDir + `",
    "llm_log_retention_days": 7,
    "scheduled_tasks_path": "",
    "ssh_known_hosts_path": "` + knownHostsRel + `"
  },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768 },
  "nodes": {
    "n1": {
      "host": "host.example.com",
      "dedicated_user": "pa",
      "auth": { "private_key_path": "/key" },
      "command_allowlist_path": "/allowlist.txt"
    }
  }
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load(nodes with nonexistent ssh_known_hosts_path): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ssh_known_hosts_path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Load: error = %v (expect ssh_known_hosts_path or no such file)", err)
	}
}
