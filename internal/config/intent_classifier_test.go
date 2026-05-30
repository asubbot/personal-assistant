package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-36.013
func TestLoad_RejectRemovedIntentClassifier_model_stage(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "intent_classifier_model_stage_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error for model_stage, got nil")
	}
	if !strings.Contains(err.Error(), "model_stage") {
		t.Fatalf("Load: error = %v, want model_stage mention", err)
	}
}

// Covers AC-36.014
func TestLoad_RejectRemovedIntentClassifier_full_lite_patterns(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "intent_classifier_full_lite_patterns_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error for full_lite_patterns, got nil")
	}
	if !strings.Contains(err.Error(), "full_lite_patterns") {
		t.Fatalf("Load: error = %v, want full_lite_patterns mention", err)
	}
}

// Covers AC-36.015, AC-36.016
func TestLoad_IntentClassifier_enabledHeuristicOnly(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "intent_classifier_enabled_heuristic_only.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IntentClassifier == nil || !cfg.IntentClassifier.Enabled {
		t.Fatal("Load: expected enabled intent_classifier")
	}
	if cfg.IntentClassifier.Heuristic == nil {
		t.Fatal("Load: expected heuristic block")
	}
	if len(cfg.IntentClassifier.Heuristic.SimplePatterns) == 0 || len(cfg.IntentClassifier.Heuristic.FullPatterns) == 0 {
		t.Fatal("Load: expected simple and full patterns")
	}
	if cfg.IntentClassifier.Heuristic.MaxSimpleLen < 1 {
		t.Fatal("Load: max_simple_len must be >= 1")
	}
}

// Covers AC-36.015
func TestLoad_IntentClassifier_invalidRegexRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "version": 1,
  "telegram": {"token_path": "/run/secrets/token", "users_path": ""},
  "llm_providers": [{"type": "ollama", "endpoint": "http://localhost:11434", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s"}],
  "paths": {"memory_dir": "/data/memory", "log_path": "/data/pa.log", "vector_index_path": "/data/pa_vectors.sqlite", "llm_log_dir": "/data/llm_logs", "llm_log_retention_days": 7, "jobs_db_path": "jobs.sqlite", "tool_catalog_path": "valid_tools.yaml"},
  "embedding": {"type": "ollama", "endpoint": "http://localhost:11434", "model": "nomic-embed-text", "dimensions": 768, "batch_size": 100, "http_timeout": "60s"},
  "nodes": {},
  "tools": {},
  "log_redaction": {"additional_patterns": []},
  "pa_timezone": "UTC",
  "tool_pre_selection": {"tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50},
  "conversation_context": {"max_dynamic_system_runes": 4000, "memory_vector": {"notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10}},
  "read_memory": {"max_span_days": 31, "max_output_bytes": 262144},
  "write_memory": {"max_append_bytes": 65536, "max_file_bytes": 5242880},
  "vector_store_reliability": {"journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": false},
  "jobs_store_reliability": {"journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": true},
  "intent_classifier": {"enabled": true, "heuristic": {"simple_patterns": ["[invalid"], "max_simple_len": 40}},
  "observability_http": null, "web_tools": null, "runtime_skills": null, "conversation_session": null
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load: expected invalid regex error")
	}
}

// Covers AC-36.015
func TestLoad_IntentClassifier_maxSimpleLenBelowOneRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{
  "version": 1,
  "telegram": {"token_path": "/run/secrets/token", "users_path": ""},
  "llm_providers": [{"type": "ollama", "endpoint": "http://localhost:11434", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "default_response_format": "text", "http_timeout": "60s"}],
  "paths": {"memory_dir": "/data/memory", "log_path": "/data/pa.log", "vector_index_path": "/data/pa_vectors.sqlite", "llm_log_dir": "/data/llm_logs", "llm_log_retention_days": 7, "jobs_db_path": "jobs.sqlite", "tool_catalog_path": "valid_tools.yaml"},
  "embedding": {"type": "ollama", "endpoint": "http://localhost:11434", "model": "nomic-embed-text", "dimensions": 768, "batch_size": 100, "http_timeout": "60s"},
  "nodes": {},
  "tools": {},
  "log_redaction": {"additional_patterns": []},
  "pa_timezone": "UTC",
  "tool_pre_selection": {"tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50},
  "conversation_context": {"max_dynamic_system_runes": 4000, "memory_vector": {"notes_top_k": 10, "summaries_top_k": 10, "turns_top_k": 10}},
  "read_memory": {"max_span_days": 31, "max_output_bytes": 262144},
  "write_memory": {"max_append_bytes": 65536, "max_file_bytes": 5242880},
  "vector_store_reliability": {"journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": false},
  "jobs_store_reliability": {"journal_mode": "WAL", "busy_timeout": "5s", "synchronous": "NORMAL", "foreign_keys": true},
  "intent_classifier": {"enabled": true, "heuristic": {"simple_patterns": ["^hi$"], "max_simple_len": 0}},
  "observability_http": null, "web_tools": null, "runtime_skills": null, "conversation_session": null
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "max_simple_len") {
		t.Fatalf("Load: want max_simple_len error, got %v", err)
	}
}

// Covers AC-36.016
func TestLoad_validWithToolCatalog_intentClassifierNull(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "valid_with_tool_catalog.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IntentClassifier != nil {
		t.Fatal("intent_classifier must be null")
	}
}

// Covers AC-36.022
func TestConfigExample_intentClassifierNull_noRemovedKeys(t *testing.T) {
	root := findRepoRootFromConfigPackage(t)
	data, err := os.ReadFile(filepath.Join(root, "config.examples", "config.example.json"))
	if err != nil {
		t.Fatalf("read config.example.json: %v", err)
	}
	if err := rejectRemovedUnsupportedConfigKeys(data); err != nil {
		t.Fatalf("config.example.json: %v", err)
	}
	var rootObj map[string]json.RawMessage
	if err := json.Unmarshal(data, &rootObj); err != nil {
		t.Fatalf("parse: %v", err)
	}
	rawIC, ok := rootObj["intent_classifier"]
	if !ok {
		t.Fatal("config.example.json: intent_classifier key must be present")
	}
	if string(rawIC) != "null" {
		t.Fatalf("config.example.json: intent_classifier = %s, want null", rawIC)
	}
}

// Covers AC-36.022
func TestLoad_IntentClassifierEnabledHeuristicOnly_testdata(t *testing.T) {
	TestLoad_IntentClassifier_enabledHeuristicOnly(t)
}

// Covers AC-17.015
func TestValidateIntentClassifier_PatternsConfigurable(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			Heuristic: &HeuristicConfig{
				SimplePatterns: []string{`^hello$`, `^bye$`},
				FullPatterns:   []string{`(search|find)`},
				MaxSimpleLen:   50,
			},
		},
	}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Covers AC-17.015
func TestValidateIntentClassifier_InvalidRegexFails(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: true,
			Heuristic: &HeuristicConfig{
				SimplePatterns: []string{`[invalid`},
				MaxSimpleLen:   50,
			},
		},
	}
	if err := validateIntentClassifier(cfg); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

// Covers AC-17.014
func TestValidateIntentClassifier_Disabled(t *testing.T) {
	cfg := &Config{
		IntentClassifier: &IntentClassifierConfig{
			Enabled: false,
		},
	}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error for disabled classifier: %v", err)
	}
}

// Covers AC-17.014
func TestValidateIntentClassifier_Nil(t *testing.T) {
	cfg := &Config{}
	if err := validateIntentClassifier(cfg); err != nil {
		t.Fatalf("unexpected error for nil classifier: %v", err)
	}
}

// Covers AC-17.018
func TestDocs_configuration_mentionsIntentClassifier(t *testing.T) {
	repoRoot := findRepoRootFromConfigPackage(t)
	p := filepath.Join(repoRoot, "docs", "configuration.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	s := string(b)
	if !strings.Contains(s, "intent_classifier") {
		t.Errorf("%s must document intent_classifier", p)
	}
}
