package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Supporting AC-19.019: valid config with explicit jobs_db_path loads.
func TestLoad_ValidConfig_JobsDBPath(t *testing.T) {
	path := filepath.Join("testdata", "valid_no_users.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(config with jobs_db_path): unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(config with jobs_db_path): got nil config")
	}
	if cfg.Paths.JobsDBPath == "" {
		t.Errorf("Load(config with jobs_db_path): JobsDBPath is empty")
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
// Covers AC-04.020: embedding.batch_size required and between 1 and 1000 (invalid_embedding_batch_size, missing_embedding_batch_size cases below).
// Covers AC-30.006: default_response_format must be text (json_object rejected at load).
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
		{"missing embedding", "missing_embedding.json", "top-level key \"embedding\""},
		{"invalid pa_timezone", "invalid_pa_timezone.json", "invalid pa_timezone"},
		{"llm_log_retention_days < 1", "llm_log_retention_zero.json", "llm_log_retention_days must be >= 1"},
		{"nodes without ssh_known_hosts_path", "nodes_missing_ssh_known_hosts_path.json", "paths.ssh_known_hosts_path is required when nodes are configured"},
		{"missing tool_catalog_path", "missing_tool_catalog_path.json", "paths.tool_catalog_path is required"},
		{"embedding batch_size out of range", "invalid_embedding_batch_size.json", "embedding.batch_size is required and must be between 1 and 1000"},
		{"missing embedding batch_size", "missing_embedding_batch_size.json", "embedding.batch_size is required and must be between 1 and 1000"},
		{"missing supports_tools", "missing_supports_tools.json", "supports_tools is required"}, // Covers AC-04.028 (REQ-04.034)
		{"tools.llm_escalation rejected", "tools_llm_escalation_rejected.json", "llm_escalation"},
		{"missing tools section", "missing_tools.json", "top-level key \"tools\""},
		{"missing read_memory", "missing_read_memory.json", "top-level key \"read_memory\""},
		{"missing write_memory", "missing_write_memory.json", "top-level key \"write_memory\""},
		{"tools.selection enabled max zero", "tools_selection_enabled_max_zero.json", "max_tools_for_llm_request must be >= 1"},
		{"missing log_redaction section", "missing_log_redaction.json", "top-level key \"log_redaction\""},
		{"missing pa_timezone", "missing_pa_timezone.json", "top-level key \"pa_timezone\""},
		{"tools.selection zero top_k", "tools_selection_zero.json", "tool_search_top_k must be >= 1"},
		{"conversation_context zero max runes", "conversation_context_zero.json", "max_dynamic_system_runes must be >= 1"},
		{"memory_vector negative notes_top_k", "conversation_memory_vector_notes_negative.json", "memory_vector top_k fields must be >= 0"},
		{"memory_vector notes_top_k over max", "conversation_memory_vector_notes_over_max.json", "conversation_context.memory_vector.notes_top_k must be <="},
		// EP-008: reject invalid LLM defaults and removed keys at load.
		{"llm default_max_tokens zero", "llm_default_max_tokens_zero.json", "default_max_tokens must be >= 1"},
		{"llm default_temperature negative", "llm_default_temperature_negative.json", "default_temperature must be in [0, 2]"},
		{"llm default_temperature above 2", "llm_default_temperature_above_two.json", "default_temperature must be in [0, 2]"},
		{"llm default_response_format invalid", "llm_default_response_format_invalid.json", "default_response_format must be \"text\""},
		{"llm default_response_format empty", "llm_default_response_format_empty.json", "default_response_format is required"},
		{"llm default_response_format json_object rejected", "llm_json_object_without_supports_json_mode.json", "default_response_format must be \"text\""},
		{"llm supports_json_mode rejected", "llm_supports_json_mode_rejected.json", "supports_json_mode is not supported"},
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

// Covers AC-01.005, AC-37.018: config load rejects unknown top-level keys with a clear error.
func TestLoad_UnknownTopLevelKey_ReturnsError(t *testing.T) {
	path := filepath.Join("testdata", "unknown_root_key.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown top-level key") || !strings.Contains(err.Error(), "foo") {
		t.Fatalf("Load: error = %v, want unknown top-level key mentioning foo", err)
	}
}

// Covers AC-19.019: legacy paths.scheduled_tasks_path is rejected even when other fields are valid.
func TestLoad_LegacyScheduledTasksPath_ReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "config.json")
	content := `{
  "version": 1,
  "telegram": { "token_path": "/t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": { "memory_dir": "/d", "log_path": "/d", "vector_index_path": "/d/pa_vectors.sqlite", "llm_log_dir": "/d", "llm_log_retention_days": 7, "scheduled_tasks_path": "legacy.json", "jobs_db_path": "jobs.sqlite", "tool_catalog_path": "tools.yaml" },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {},
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load(config with legacy scheduled_tasks_path): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scheduled_tasks_path") {
		t.Fatalf("Load(config with legacy scheduled_tasks_path): error = %v, want scheduled_tasks_path mention", err)
	}
}

// EP-008 REQ-08.001: boundary default_temperature (0 and 2) loads successfully — within [0,2] validation.
// Covers AC-08.001: provider default_temperature accepts boundary values.
func TestLoad_LLMProviderDefaults_boundaryTemperature_loads(t *testing.T) {
	for _, name := range []string{"llm_default_temperature_zero.json", "llm_default_temperature_two.json"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join("testdata", name)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load(%s): %v", name, err)
			}
			if cfg == nil || len(cfg.LLMProviders) != 1 {
				t.Fatalf("Load(%s): unexpected config", name)
			}
		})
	}
}

// Covers AC-01.014: conversation_context.memory_vector may set all lanes to zero (disables automatic semantic retrieval while keeping config valid).
func TestLoad_ConversationContext_memoryVectorAllZeros_loads(t *testing.T) {
	path := filepath.Join("testdata", "conversation_memory_vector_all_zero.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil || cfg.ConversationContext == nil {
		t.Fatal("Load: nil config")
	}
	mv := cfg.ConversationContext.MemoryVector
	if mv.NotesTopK != 0 || mv.SummariesTopK != 0 || mv.TurnsTopK != 0 {
		t.Fatalf("MemoryVector = %+v, want all zeros", mv)
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

// Covers AC-04.001, AC-04.002: when tool_catalog_path is set, invalid path or invalid catalog causes startup failure.
func TestLoad_ToolCatalogPath_InvalidPath_ReturnsError(t *testing.T) {
	// Config with tool_catalog_path pointing to nonexistent file; path resolved relative to config dir.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfgJSON := `{
	  "version": 1,
	  "telegram": { "token_path": "/run/secrets/token", "users_path": "" },
	  "llm_providers": [ { "type": "ollama", "endpoint": "http://localhost:11434", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" } ],
	  "paths": {
	    "memory_dir": "/data/memory",
	    "log_path": "/data/pa.log",
	    "vector_index_path": "/data/pa.sqlite",
	    "llm_log_dir": "/data/llm",
	    "llm_log_retention_days": 7,
	    "jobs_db_path": "jobs.sqlite",
	    "tool_catalog_path": "nonexistent_catalog.yaml"
	  },
	  "embedding": { "type": "ollama", "endpoint": "http://localhost:11434", "model": "nomic", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
	  "nodes": {},
	  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
	  "log_redaction": { "additional_patterns": [] },
	  "pa_timezone": "UTC",
	  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
	  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
	  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
	}`
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load(config with nonexistent tool_catalog_path): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tool catalog") && !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("Load: error = %v (expect tool catalog or nonexistent)", err)
	}
}

// Covers AC-04.002: when tool_catalog_path is set and file is valid, catalog is parsed at load.
func TestLoad_ToolCatalogPath_ValidCatalog_LoadsCatalog(t *testing.T) {
	path := filepath.Join("testdata", "valid_with_tool_catalog.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load(valid config with tool catalog): %v", err)
	}
	if cfg.ToolCatalog == nil {
		t.Fatal("Load(valid config with tool catalog): ToolCatalog is nil")
	}
	if len(cfg.ToolCatalog.Tools) != 1 {
		t.Errorf("Load(valid config with tool catalog): len(Tools) = %d, want 1", len(cfg.ToolCatalog.Tools))
	}
	tool, ok := cfg.ToolCatalog.Tools["test_tool"]
	if !ok {
		t.Fatal("Load(valid config with tool catalog): tool test_tool not found")
	}
	if tool.Template != "echo {{arg}}" || tool.NodeID != "nas" {
		t.Errorf("Load(valid config with tool catalog): tool = template %q node_id %q", tool.Template, tool.NodeID)
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

// Covers AC-30.004, AC-30.013: tools.text_based_enabled is rejected at load.
func TestLoad_ToolsTextBasedEnabled_rejected(t *testing.T) {
	path := filepath.Join("testdata", "tools_text_based_enabled_rejected.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "text_based_enabled") {
		t.Fatalf("Load: error = %v, want text_based_enabled rejection", err)
	}
}

// Covers AC-30.005: supports_json_mode is rejected at load.
func TestLoad_llmSupportsJSONMode_rejected(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "llm_supports_json_mode_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "supports_json_mode") {
		t.Fatalf("Load: error = %v, want supports_json_mode rejection", err)
	}
}

func configTestRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from test working directory")
		}
		dir = parent
	}
}

// Covers AC-30.003: legacy text-tool package path removed.
func TestEP030_tooltextDirectoryAbsent(t *testing.T) {
	p := filepath.Join(configTestRepoRoot(t), "internal", "tooltext")
	if _, err := os.Stat(p); err == nil {
		t.Fatalf("%s exists; internal/tooltext must not be present", p)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", p, err)
	}
}

// Covers AC-30.012: operator docs describe native-only tools and removed configuration keys.
func TestDocs_configuration_md_EP030(t *testing.T) {
	p := filepath.Join(configTestRepoRoot(t), "docs", "configuration.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	for _, needle := range []string{
		"supports_tools",
		"default_response_format",
		"text_based_enabled",
		"supports_json_mode",
		"Native tool calling",
	} {
		if !strings.Contains(s, needle) {
			t.Errorf("docs/configuration.md should mention %q for operator guidance", needle)
		}
	}
}

// Covers AC-34.007 (REQ-34.007): tools.llm_escalation is rejected at load.
func TestLoad_ToolsLLMEscalation_rejected(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_llm_escalation_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "llm_escalation") {
		t.Fatalf("Load: error = %v, want llm_escalation rejection", err)
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
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": { "memory_dir": "/d", "log_path": "/d", "vector_index_path": "/d/pa_vectors.sqlite", "llm_log_dir": "/d", "llm_log_retention_days": 7, "jobs_db_path": "jobs.sqlite", "tool_catalog_path": "tools.yaml" },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {},
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Catalog must exist so load fails on users file, not catalog.
	if err := os.WriteFile(filepath.Join(cfgDir, "tools.yaml"), []byte("tools:\n  - id: _x\n    index_text: x\n    template: echo x\n    node_id: _n\n    arguments: []\n"), 0o600); err != nil {
		t.Fatalf("setup tools.yaml: %v", err)
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
// Covers AC-01.005: traceability for TestLoad_NodesWithNonexistentSSHKnownHostsFile_ReturnsError.
func TestLoad_NodesWithNonexistentSSHKnownHostsFile_ReturnsError(t *testing.T) {
	cfgDir := t.TempDir()
	configPath := filepath.Join(cfgDir, "config.json")
	knownHostsRel := "nonexistent_known_hosts"
	content := `{
  "version": 1,
  "telegram": { "token_path": "/t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": {
    "memory_dir": "` + cfgDir + `",
    "log_path": "` + cfgDir + `/pa.log",
    "vector_index_path": "` + cfgDir + `/pa_vectors.sqlite",
    "llm_log_dir": "` + cfgDir + `",
    "llm_log_retention_days": 7,
    "jobs_db_path": "jobs.sqlite",
    "tool_catalog_path": "tools.yaml",
    "ssh_known_hosts_path": "` + knownHostsRel + `"
  },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {
    "n1": {
      "host": "host.example.com",
      "dedicated_user": "pa",
      "auth": { "private_key_path": "/key" },
      "command_allowlist_path": "/allowlist.txt"
    }
  },
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "tools.yaml"), []byte("tools:\n  - id: _x\n    index_text: x\n    template: echo x\n    node_id: _n\n    arguments: []\n"), 0o600); err != nil {
		t.Fatalf("setup tools.yaml: %v", err)
	}
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load(nodes with nonexistent ssh_known_hosts_path): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ssh_known_hosts_path") && !strings.Contains(err.Error(), "no such file") {
		t.Errorf("Load: error = %v (expect ssh_known_hosts_path or no such file)", err)
	}
}

// Covers AC-14.014: operator docs mention conversation_session (see docs/configuration.md).
func TestDocs_configuration_mentionsConversationSession(t *testing.T) {
	repoRoot := findRepoRootFromConfigPackage(t)
	p := filepath.Join(repoRoot, "docs", "configuration.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	if !strings.Contains(s, "conversation_session") || !strings.Contains(s, "max_session_exchanges") {
		t.Errorf("%s must document conversation_session and max_session_exchanges", p)
	}
}

func findRepoRootFromConfigPackage(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// Covers AC-14.001: conversation_session with enabled and positive max loads.
func TestLoad_ConversationSession_enabled_OK(t *testing.T) {
	path := filepath.Join("testdata", "conversation_session_ok.json")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ConversationSession == nil || !cfg.ConversationSession.Enabled || cfg.ConversationSession.MaxSessionExchanges != 5 {
		t.Fatalf("ConversationSession = %+v", cfg.ConversationSession)
	}
}

// Covers AC-14.002: enabled session with max < 1 fails load.
func TestLoad_ConversationSession_enabled_badMax(t *testing.T) {
	path := filepath.Join("testdata", "conversation_session_bad_max.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for max_session_exchanges < 1 when enabled")
	}
	if !strings.Contains(err.Error(), "conversation_session.max_session_exchanges") {
		t.Errorf("Load: error = %v", err)
	}
}

// Covers AC-09.017: invalid create_tool_secret_patterns fails config load.
func TestLoad_CreateToolSecretPatterns_InvalidRegex(t *testing.T) {
	path := filepath.Join("testdata", "create_tool_bad_regex.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for invalid regexp in create_tool_secret_patterns")
	}
	if !strings.Contains(err.Error(), "create_tool_secret_patterns") {
		t.Errorf("Load: error = %v", err)
	}
}

// Covers AC-01.005: two nodes must not share the same private_key_path after resolve.
func TestLoad_Nodes_duplicatePrivateKeyPath(t *testing.T) {
	cfgDir := t.TempDir()
	secretsDir := filepath.Join(cfgDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(secretsDir, "shared")
	if err := os.WriteFile(keyFile, []byte("not-a-real-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "known_hosts"), []byte("localhost ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "allow.txt"), []byte("uptime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "tools.yaml"), []byte("tools:\n  - id: _x\n    index_text: x\n    template: echo x\n    node_id: _n\n    arguments: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "config.json")
	content := `{
  "version": 1,
  "telegram": { "token_path": "` + filepath.Join(cfgDir, "token") + `", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": {
    "memory_dir": "` + cfgDir + `",
    "log_path": "` + cfgDir + `/pa.log",
    "vector_index_path": "` + cfgDir + `/v.sqlite",
    "llm_log_dir": "` + cfgDir + `",
    "llm_log_retention_days": 7,
    "jobs_db_path": "jobs.sqlite",
    "tool_catalog_path": "tools.yaml",
    "ssh_known_hosts_path": "known_hosts"
  },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {
    "n1": { "host": "h1", "dedicated_user": "u1", "auth": { "private_key_path": "shared" }, "command_allowlist_path": "allow.txt" },
    "n2": { "host": "h2", "dedicated_user": "u2", "auth": { "private_key_path": "shared" }, "command_allowlist_path": "allow.txt" }
  },
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "token"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PA_SECRETS_DIR", secretsDir)
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load: expected error when two nodes share the same private_key_path")
	}
	if !strings.Contains(err.Error(), "same SSH private_key_path") || !strings.Contains(err.Error(), "n1") || !strings.Contains(err.Error(), "n2") {
		t.Fatalf("Load: error = %v", err)
	}
}

// Covers AC-01.005: two nodes with different key files load successfully.
func TestLoad_Nodes_distinctPrivateKeyPaths_OK(t *testing.T) {
	cfgDir := t.TempDir()
	secretsDir := filepath.Join(cfgDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "key1"), []byte("k1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "key2"), []byte("k2"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "known_hosts"), []byte("localhost ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "allow.txt"), []byte("uptime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "tools.yaml"), []byte("tools:\n  - id: _x\n    index_text: x\n    template: echo x\n    node_id: _n\n    arguments: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "config.json")
	content := `{
  "version": 1,
  "telegram": { "token_path": "` + filepath.Join(cfgDir, "token") + `", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": {
    "memory_dir": "` + cfgDir + `",
    "log_path": "` + cfgDir + `/pa.log",
    "vector_index_path": "` + cfgDir + `/v.sqlite",
    "llm_log_dir": "` + cfgDir + `",
    "llm_log_retention_days": 7,
    "jobs_db_path": "jobs.sqlite",
    "tool_catalog_path": "tools.yaml",
    "ssh_known_hosts_path": "known_hosts"
  },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {
    "n1": { "host": "h1", "dedicated_user": "u1", "auth": { "private_key_path": "key1" }, "command_allowlist_path": "allow.txt" },
    "n2": { "host": "h2", "dedicated_user": "u2", "auth": { "private_key_path": "key2" }, "command_allowlist_path": "allow.txt" }
  },
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "token"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PA_SECRETS_DIR", secretsDir)
	_, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
}

// Covers AC-01.005: two nodes pointing at different paths that resolve to the same file (symlink) must fail load.
func TestLoad_Nodes_symlinkPrivateKeySameFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not assumed on Windows CI")
	}
	cfgDir := t.TempDir()
	secretsDir := filepath.Join(cfgDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	realKey := filepath.Join(secretsDir, "real")
	aliasKey := filepath.Join(secretsDir, "alias")
	if err := os.WriteFile(realKey, []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realKey, aliasKey); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "known_hosts"), []byte("localhost ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "allow.txt"), []byte("uptime\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "tools.yaml"), []byte("tools:\n  - id: _x\n    index_text: x\n    template: echo x\n    node_id: _n\n    arguments: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(cfgDir, "config.json")
	content := `{
  "version": 1,
  "telegram": { "token_path": "` + filepath.Join(cfgDir, "token") + `", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://x", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": {
    "memory_dir": "` + cfgDir + `",
    "log_path": "` + cfgDir + `/pa.log",
    "vector_index_path": "` + cfgDir + `/v.sqlite",
    "llm_log_dir": "` + cfgDir + `",
    "llm_log_retention_days": 7,
    "jobs_db_path": "jobs.sqlite",
    "tool_catalog_path": "tools.yaml",
    "ssh_known_hosts_path": "known_hosts"
  },
  "embedding": { "type": "ollama", "endpoint": "http://x", "model": "m", "dimensions": 768, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {
    "n1": { "host": "h1", "dedicated_user": "u1", "auth": { "private_key_path": "real" }, "command_allowlist_path": "allow.txt" },
    "n2": { "host": "h2", "dedicated_user": "u2", "auth": { "private_key_path": "alias" }, "command_allowlist_path": "allow.txt" }
  },
  "tools": { "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 } },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 }, "sqlite_store_defaults": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL" }, "vector_store_reliability": { "foreign_keys": false }, "jobs_store_reliability": { "foreign_keys": true }, "web_tools": null, "runtime_skills": null, "conversation_session": null, "intent_classifier": null, "observability_http": null
}`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "token"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PA_SECRETS_DIR", secretsDir)
	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load: expected error when two nodes use symlinked paths to the same key file")
	}
	if !strings.Contains(err.Error(), "resolve to the same SSH private key file") {
		t.Fatalf("Load: error = %v", err)
	}
}
