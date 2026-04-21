package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testConfigWithVectorSearchTools(toolsBlock string) string {
	return `{
  "version": 1,
  "telegram": { "token_path": "t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s" }],
  "paths": {
    "memory_dir": "memory",
    "log_path": "pa.log",
    "vector_index_path": "vec.sqlite",
    "llm_log_dir": "llm_logs",
    "llm_log_retention_days": 7,
    "jobs_db_path": "jobs.sqlite",
    "tool_catalog_path": "tools.yaml"
  },
  "embedding": { "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m", "dimensions": 4, "batch_size": 100, "http_timeout": "60s" },
  "nodes": {},
  "tools": ` + toolsBlock + `,
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "tool_pre_selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50 },
  "conversation_context": { "max_dynamic_system_runes": 4000, "memory_vector": { "notes_top_k": 0, "summaries_top_k": 0, "turns_top_k": 0 } },
  "read_memory": { "max_span_days": 31, "max_output_bytes": 262144 },
  "write_memory": { "max_append_bytes": 65536, "max_file_bytes": 5242880 },
  "vector_store_reliability": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": false },
  "jobs_store_reliability": { "journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": true },
  "web_tools": null,
  "runtime_skills": null,
  "conversation_session": null,
  "intent_classifier": null,
  "observability_http": null
}`
}

func writeConfigAndCatalog(t *testing.T, dir, cfgJSON string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	catalog := "tools:\n  - id: _placeholder\n    index_text: placeholder\n    template: echo x\n    node_id: _none\n    arguments: []\n"
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte(catalog), 0o600); err != nil {
		t.Fatalf("write tools.yaml: %v", err)
	}
	return cfgPath
}

// Covers AC-32.004 and AC-32.006: unified vector_search_tools block loads and is accessible for memory settings.
func TestLoad_VectorSearchToolsConfig_Valid(t *testing.T) {
	tools := `{
    "vector_search_tools": {
      "search_vector_memory": {"enabled": true, "default_top_k": 4, "max_top_k": 9, "max_output_bytes": 5000, "snippet_runes": 180},
      "search_vector_tool": {"enabled": true, "default_top_k": 3, "max_top_k": 7, "max_output_bytes": 4096, "snippet_runes": 120},
      "search_vector_skill": {"enabled": false, "default_top_k": 2, "max_top_k": 5, "max_output_bytes": 3072, "snippet_runes": 100}
    }
  }`
	cfgPath := writeConfigAndCatalog(t, t.TempDir(), testConfigWithVectorSearchTools(tools))
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := cfg.VectorSearchToolSettings("search_vector_memory")
	if got.DefaultTopK != 4 || got.MaxTopK != 9 || got.MaxOutputBytes != 5000 || got.SnippetRunes != 180 {
		t.Fatalf("memory settings = %+v", got)
	}
	if cfg.VectorSearchToolSettings("search_vector_skill").Enabled {
		t.Fatal("search_vector_skill enabled must be false")
	}
}

// Covers AC-32.005: invalid unified vector_search_tools bounds fail fast on config load.
func TestLoad_VectorSearchToolsConfig_InvalidBounds(t *testing.T) {
	tools := `{
    "vector_search_tools": {
      "search_vector_memory": {"enabled": true, "default_top_k": 8, "max_top_k": 4, "max_output_bytes": 4096, "snippet_runes": 120},
      "search_vector_tool": {"enabled": true, "default_top_k": 3, "max_top_k": 7, "max_output_bytes": 4096, "snippet_runes": 120},
      "search_vector_skill": {"enabled": true, "default_top_k": 3, "max_top_k": 7, "max_output_bytes": 4096, "snippet_runes": 120}
    }
  }`
	cfgPath := writeConfigAndCatalog(t, t.TempDir(), testConfigWithVectorSearchTools(tools))
	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tools.vector_search_tools.search_vector_memory.default_top_k") {
		t.Fatalf("unexpected error: %v", err)
	}
}
