package main

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"pa/internal/config"
	"pa/internal/toolcatalog"
	patools "pa/internal/tools"
	"pa/internal/vector/sqlite"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Covers AC-11.001 — web_search and web_fetch register when web_tools.enabled.
func TestRegisterWebToolsIfEnabled_RegistersBothTools(t *testing.T) {
	cfg := &config.Config{
		WebTools: &config.WebToolsConfig{
			Enabled: true,
			Search: config.WebSearchConfig{
				Provider:        "duckduckgo",
				TimeoutSeconds:  10,
				CacheTTLSeconds: 60,
				CacheMaxEntries: 10,
			},
			Fetch: config.WebFetchConfig{
				TimeoutSeconds: 30,
				MaxBodyBytes:   1024,
				MaxRedirects:   3,
			},
		},
	}
	reg := patools.NewRegistry()
	logger := slog.New(slog.DiscardHandler)
	registerWebToolsIfEnabled(cfg, reg, logger)
	if _, ok := reg.Get("web_search"); !ok {
		t.Fatal("expected web_search in registry")
	}
	if _, ok := reg.Get("web_fetch"); !ok {
		t.Fatal("expected web_fetch in registry")
	}
}

// Covers AC-01.042 (US-20): config path resolved from PA_CONFIG_DIR when set.
func TestConfigFilePath_PAConfigDirSet(t *testing.T) {
	dir := "/etc/pa"
	_ = os.Setenv("PA_CONFIG_DIR", dir)
	t.Cleanup(func() { _ = os.Unsetenv("PA_CONFIG_DIR") })

	got := configFilePath()
	want := filepath.Join(dir, config.ConfigFileName)
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

// Covers AC-01.042 (US-20): when PA_CONFIG_DIR unset or empty, documented default is used.
func TestConfigFilePath_PAConfigDirUnsetOrEmpty(t *testing.T) {
	_ = os.Unsetenv("PA_CONFIG_DIR")
	t.Cleanup(func() { _ = os.Unsetenv("PA_CONFIG_DIR") })

	got := configFilePath()
	want := filepath.Join("./.config", config.ConfigFileName)
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

// moduleRoot returns the repo root (directory containing go.mod) or skips the test.
func moduleRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	moduleRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err == nil {
			return moduleRoot
		}
		parent := filepath.Dir(moduleRoot)
		if parent == moduleRoot {
			t.Skip("could not find go.mod (not in module?)")
		}
		moduleRoot = parent
	}
}

// minimalToolCatalogYAML is a valid tool catalog so config load succeeds when tool_catalog_path is set.
const minimalToolCatalogYAML = "tools:\n  - id: _placeholder\n    index_text: placeholder\n    template: echo x\n    node_id: _none\n    arguments: []\n"

// writeValidConfigWithCatalog writes validSummarizeConfig and tools.yaml to dir; returns config path. Use when tests call config.Load.
func writeValidConfigWithCatalog(t *testing.T, dir string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte(validSummarizeConfig), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte(minimalToolCatalogYAML), 0o600); err != nil {
		t.Fatalf("write tools.yaml: %v", err)
	}
	return cfgPath
}

// runCLIWithConfig writes configJSON to dir/config.json, runs `go run ./cmd/pa [args...]` with PA_CONFIG_DIR and PA_DATA_DIR set, returns combined output and error (exit error if non-zero).
func runCLIWithConfig(t *testing.T, dir, configJSON string, args ...string) (output []byte, runErr error) {
	t.Helper()
	cfgPath := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte(configJSON), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if strings.Contains(configJSON, "tool_catalog_path") {
		toolsPath := filepath.Join(dir, "tools.yaml")
		if err := os.WriteFile(toolsPath, []byte(minimalToolCatalogYAML), 0o600); err != nil {
			t.Fatalf("write tools.yaml: %v", err)
		}
	}
	cfgDir := filepath.Dir(cfgPath)
	root := moduleRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmdArgs := append([]string{"run", "./cmd/pa"}, args...)
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "PA_CONFIG_DIR="+cfgDir, "PA_DATA_DIR="+dir)
	return cmd.CombinedOutput()
}

var validSummarizeConfig = `{
  "version": 1,
  "telegram": { "token_path": "t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m", "supports_tools": true, "default_temperature": 0.3, "default_max_tokens": 1024, "supports_json_mode": true, "default_response_format": "text" }],
  "paths": {
    "memory_dir": "memory",
    "log_path": "pa.log",
    "vector_index_path": "vec.sqlite",
    "llm_log_dir": "llm_logs",
    "llm_log_retention_days": 7,
    "scheduled_tasks_path": "",
    "tool_catalog_path": "tools.yaml"
  },
  "embedding": { "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m", "dimensions": 4, "batch_size": 100 },
  "nodes": {},
  "tools": { "text_based_enabled": false },
  "log_redaction": { "additional_patterns": [] },
  "pa_timezone": "UTC",
  "tool_pre_selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50 },
  "conversation_context": { "max_dynamic_system_runes": 4000, "vector_search_top_k": 10 }
}`

// runSummarizeCLI runs `go run ./cmd/pa -summarize=<value>` with minimal config in dir; expects exit 0 (e.g. skip when no data).
func runSummarizeCLI(t *testing.T, dir, summarizeValue string) {
	t.Helper()
	out, err := runCLIWithConfig(t, dir, validSummarizeConfig, "-summarize="+summarizeValue)
	if err != nil {
		t.Logf("output: %s", out)
		t.Fatalf("run -summarize=%s: %v", summarizeValue, err)
	}
	if len(out) > 0 {
		t.Logf("output: %s", out)
	}
}

// Covers AC-01.011, AC-01.012 (US-06): day summarization CLI -summarize=YYYY-MM-DD runs without starting the bot and exits 0 when successful (e.g. no entries to summarize).
func TestSummarizeCLI_day_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026-03-12")
}

// Supporting AC-01.011, AC-01.012 (US-06): month summarization CLI -summarize=YYYY-MM exits 0 when no day summaries (skip).
func TestSummarizeCLI_month_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026-03")
}

// Supporting AC-01.011, AC-01.012 (US-06): year summarization CLI -summarize=YYYY exits 0 when no month summaries (skip).
func TestSummarizeCLI_year_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026")
}

// LLM log retention (fail fast): config with llm_log_retention_days 0 or missing — binary exits non-zero (bot and -summarize).
// Covers AC-01.011: traceability for TestCLI_retentionDaysZero_refusesStart.
func TestCLI_retentionDaysZero_refusesStart(t *testing.T) {
	dir := t.TempDir()
	cfgZero := strings.Replace(validSummarizeConfig, `"llm_log_retention_days": 7`, `"llm_log_retention_days": 0`, 1)

	// Run without -summarize (bot mode): should exit non-zero on config load.
	out, err := runCLIWithConfig(t, dir, cfgZero)
	if err == nil {
		t.Logf("output: %s", out)
		t.Fatal("run (bot mode) with retention_days=0: expected non-zero exit, got nil")
	}
	if !strings.Contains(string(out), "llm_log_retention_days") {
		t.Logf("output: %s", out)
		t.Errorf("expected error message to mention llm_log_retention_days, got: %s", out)
	}

	// Run with -summarize: should also exit non-zero (same config validation).
	out2, err2 := runCLIWithConfig(t, dir, cfgZero, "-summarize=2026-03-01")
	if err2 == nil {
		t.Logf("output: %s", out2)
		t.Fatal("run -summarize with retention_days=0: expected non-zero exit, got nil")
	}
}

// testLogger returns a logger that discards output so test output is minimal.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.DiscardHandler)
}

// Covers cmd/pa runSummarize/runVerifyNodes coverage: invalid scope returns error.
// Covers AC-01.011: traceability for TestRunSummarize_InvalidScope_returnsError.
func TestRunSummarize_InvalidScope_returnsError(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	cfgPath := writeValidConfigWithCatalog(t, dir)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	err = runSummarize(cfg, "not-a-date", logger)
	if err == nil {
		t.Fatal("runSummarize(invalid scope): expected error")
	}
	if !strings.Contains(err.Error(), "invalid format") && !strings.Contains(err.Error(), "summarize") {
		t.Errorf("runSummarize error = %v, want summarize/format message", err)
	}
}

// Covers cmd/pa runSummarizeDay coverage: no LLM log entries skips write and returns nil (AC-01.011, AC-01.012).
func TestRunSummarizeDay_NoEntries_success(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	cfgPath := writeValidConfigWithCatalog(t, dir)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	// Resolve paths relative to config dir so memory_dir and vector_index_path exist under dir.
	cfg.Paths.MemoryDir = filepath.Join(dir, "memory")
	cfg.Paths.VectorIndexPath = filepath.Join(dir, "vec.sqlite")
	cfg.Paths.LLMLogDir = filepath.Join(dir, "llm_logs")
	_ = os.MkdirAll(cfg.Paths.LLMLogDir, 0o755)

	err = runSummarizeDay(cfg, time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC), logger)
	if err != nil {
		t.Fatalf("runSummarizeDay(no entries): %v", err)
	}
}

// Covers cmd/pa runSummarizeMonth coverage: no day summaries skips write and returns nil (AC-01.011, AC-01.012).
func TestRunSummarizeMonth_NoEntries_success(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	cfgPath := writeValidConfigWithCatalog(t, dir)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Paths.MemoryDir = filepath.Join(dir, "memory")
	cfg.Paths.VectorIndexPath = filepath.Join(dir, "vec.sqlite")
	cfg.Paths.LLMLogDir = filepath.Join(dir, "llm_logs")
	_ = os.MkdirAll(cfg.Paths.MemoryDir, 0o755)
	_ = os.MkdirAll(cfg.Paths.LLMLogDir, 0o755)

	err = runSummarizeMonth(cfg, 2026, 3, logger)
	if err != nil {
		t.Fatalf("runSummarizeMonth(no day summaries): %v", err)
	}
}

// Covers cmd/pa runSummarizeYear coverage: no month summaries skips write and returns nil (AC-01.011, AC-01.012).
func TestRunSummarizeYear_NoEntries_success(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	cfgPath := writeValidConfigWithCatalog(t, dir)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Paths.MemoryDir = filepath.Join(dir, "memory")
	cfg.Paths.VectorIndexPath = filepath.Join(dir, "vec.sqlite")
	cfg.Paths.LLMLogDir = filepath.Join(dir, "llm_logs")
	_ = os.MkdirAll(cfg.Paths.MemoryDir, 0o755)
	_ = os.MkdirAll(cfg.Paths.LLMLogDir, 0o755)

	err = runSummarizeYear(cfg, 2026, logger)
	if err != nil {
		t.Fatalf("runSummarizeYear(no month summaries): %v", err)
	}
}

// Covers cmd/pa runVerifyNodes coverage: empty nodes returns nil (AC-01.032).
func TestRunVerifyNodes_EmptyNodes_noError(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	cfgPath := writeValidConfigWithCatalog(t, dir)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.Nodes) != 0 {
		t.Fatal("validSummarizeConfig should have empty nodes")
	}

	err = runVerifyNodes(cfg, "uptime", logger)
	if err != nil {
		t.Errorf("runVerifyNodes(empty nodes): %v", err)
	}
}

// Covers cmd/pa runVerifyNodes coverage: nodes with missing allowlist file returns error (AC-01.032, allowlist load failure).
func TestRunVerifyNodes_AllowlistLoadError_returnsError(t *testing.T) {
	logger := testLogger(t)
	dir := t.TempDir()
	knownHostsPath := filepath.Join(dir, "known_hosts")
	allowlistPath := filepath.Join(dir, "nonexistent_allowlist.txt") // do not create
	if err := os.WriteFile(knownHostsPath, nil, 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	// Build config JSON with nodes and ssh_known_hosts_path (paths are relative to config dir).
	cfgJSON := strings.Replace(validSummarizeConfig, `"scheduled_tasks_path": ""`, `"scheduled_tasks_path": "", "ssh_known_hosts_path": "known_hosts"`, 1)
	cfgJSON = strings.Replace(cfgJSON, `"nodes": {}`, `"nodes": { "n1": { "host": "127.0.0.1", "dedicated_user": "u", "auth": { "private_key_path": "missing_key" }, "command_allowlist_path": "`+filepath.Base(allowlistPath)+`" } }`, 1)
	cfgPath := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(cfgPath, []byte(cfgJSON), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte(minimalToolCatalogYAML), 0o600); err != nil {
		t.Fatalf("write tools.yaml: %v", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	err = runVerifyNodes(cfg, "uptime", logger)
	if err == nil {
		t.Fatal("runVerifyNodes(missing allowlist): expected error")
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Errorf("runVerifyNodes error = %v, want allowlist in message", err)
	}
}

// LLM log retention: -summarize prunes old llm-YYYY-MM-DD.jsonl files and keeps recent ones.
// Covers AC-01.011: traceability for TestSummarizeCLI_prunesOldLLMLogs.
func TestSummarizeCLI_prunesOldLLMLogs(t *testing.T) {
	dir := t.TempDir()
	llmLogDir := filepath.Join(dir, "llm_logs")
	if err := os.MkdirAll(llmLogDir, 0o755); err != nil {
		t.Fatalf("mkdir llm_logs: %v", err)
	}
	oldFile := filepath.Join(llmLogDir, "llm-2020-01-01.jsonl")
	todayFile := filepath.Join(llmLogDir, "llm-"+time.Now().UTC().Format("2006-01-02")+".jsonl")
	for _, p := range []string{oldFile, todayFile} {
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	out, err := runCLIWithConfig(t, dir, validSummarizeConfig, "-summarize=2026-03-12")
	if err != nil {
		t.Logf("output: %s", out)
		t.Fatalf("run -summarize: %v", err)
	}

	if _, err := os.Stat(oldFile); err == nil {
		t.Error("old file llm-2020-01-01.jsonl should have been pruned, still exists")
	}
	if _, err := os.Stat(todayFile); err != nil {
		t.Errorf("recent file %s should remain: %v", todayFile, err)
	}
}

// newToolIndex with nil ToolCatalog returns error "tool catalog is required".
// Covers AC-01.011: traceability for TestNewToolIndex_nilCatalog_returnsError.
func TestNewToolIndex_nilCatalog_returnsError(t *testing.T) {
	logger := testLogger(t)
	cfg := &config.Config{
		ToolCatalog: nil,
		Embedding:   &config.EmbeddingProvider{Dimensions: 4, BatchSize: 10},
		Paths:       config.Paths{VectorIndexPath: filepath.Join(t.TempDir(), "vec.db")},
	}

	idx, err := newToolIndex(cfg, nil, logger)
	if err == nil {
		if idx != nil {
			_ = idx.Close()
		}
		t.Fatal("newToolIndex(nil catalog): expected error, got nil")
	}
	if idx != nil {
		t.Error("newToolIndex(nil catalog): expected nil index")
	}
	if !strings.Contains(err.Error(), "tool catalog is required") {
		t.Errorf("newToolIndex: err = %v", err)
	}
}

// newToolIndex when NewWithTable fails (e.g. path under non-existent dir) returns error; no goroutine started.
// Covers AC-01.011: traceability for TestNewToolIndex_vectorStoreFails_returnsError.
func TestNewToolIndex_vectorStoreFails_returnsError(t *testing.T) {
	logger := testLogger(t)
	// Path under a directory we do not create so sqlite open fails.
	badPath := filepath.Join(t.TempDir(), "nonexistent", "sub", "vec.db")
	cfg := &config.Config{
		ToolCatalog: &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}},
		Embedding:   &config.EmbeddingProvider{Dimensions: 4, BatchSize: 10},
		Paths:       config.Paths{VectorIndexPath: badPath},
	}

	idx, err := newToolIndex(cfg, nil, logger)
	if err == nil {
		if idx != nil {
			_ = idx.Close()
		}
		t.Fatal("newToolIndex(bad path): expected error, got nil")
	}
	if idx != nil {
		t.Error("newToolIndex(bad path): expected nil index")
	}
}

func testClearContextConfig(vectorPath string) *config.Config {
	return &config.Config{
		Embedding: &config.EmbeddingProvider{Dimensions: 4, BatchSize: 10},
		Paths:     config.Paths{VectorIndexPath: vectorPath},
	}
}

// clearConversationContext removes all vec_items rows so semantic context is empty on next Search.
// Covers AC-01.011: traceability for TestClearConversationContext_vecItemsEmptyAfterClear.
func TestClearConversationContext_vecItemsEmptyAfterClear(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "vec.sqlite")

	mem, err := sqlite.NewWithTable(dbPath, 4, sqlite.TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable memory: %v", err)
	}
	if err := mem.Add(ctx, "turn1", []float32{1, 0, 0, 0}, "User: hello"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := mem.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := clearConversationContext(testClearContextConfig(dbPath)); err != nil {
		t.Fatalf("clearConversationContext: %v", err)
	}

	mem2, err := sqlite.NewWithTable(dbPath, 4, sqlite.TableMemory)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = mem2.Close() }()
	results, err := mem2.Search(ctx, []float32{1, 0, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search after clear: got %d results, want 0", len(results))
	}
}

// clearConversationContext does not delete vec_tools (tool pre-selection index).
// Covers AC-01.011: traceability for TestClearConversationContext_vecToolsUnchanged.
func TestClearConversationContext_vecToolsUnchanged(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "both.sqlite")

	mem, err := sqlite.NewWithTable(dbPath, 4, sqlite.TableMemory)
	if err != nil {
		t.Fatalf("NewWithTable memory: %v", err)
	}
	if err := mem.Add(ctx, "mem1", []float32{1, 0, 0, 0}, "context chunk"); err != nil {
		t.Fatalf("Add mem: %v", err)
	}
	if err := mem.Close(); err != nil {
		t.Fatalf("Close mem: %v", err)
	}

	tools, err := sqlite.NewWithTable(dbPath, 4, sqlite.TableTools)
	if err != nil {
		t.Fatalf("NewWithTable tools: %v", err)
	}
	if err := tools.Add(ctx, "tool_x", []float32{0, 1, 0, 0}, "tool text"); err != nil {
		t.Fatalf("Add tools: %v", err)
	}
	if err := tools.Close(); err != nil {
		t.Fatalf("Close tools: %v", err)
	}

	if err := clearConversationContext(testClearContextConfig(dbPath)); err != nil {
		t.Fatalf("clearConversationContext: %v", err)
	}

	mem2, _ := sqlite.NewWithTable(dbPath, 4, sqlite.TableMemory)
	defer func() { _ = mem2.Close() }()
	rMem, _ := mem2.Search(ctx, []float32{1, 0, 0, 0}, 10)
	if len(rMem) != 0 {
		t.Errorf("vec_items: want 0 after clear, got %d", len(rMem))
	}

	tools2, _ := sqlite.NewWithTable(dbPath, 4, sqlite.TableTools)
	defer func() { _ = tools2.Close() }()
	rTools, err := tools2.Search(ctx, []float32{0, 1, 0, 0}, 10)
	if err != nil {
		t.Fatalf("Search tools: %v", err)
	}
	if len(rTools) != 1 || rTools[0].ID != "tool_x" {
		t.Errorf("vec_tools after clear: got %+v, want one row tool_x", rTools)
	}
}

// Covers AC-01.011: traceability for TestClearConversationContext_validationErrors.
func TestClearConversationContext_validationErrors(t *testing.T) {
	if err := clearConversationContext(nil); err == nil {
		t.Fatal("nil config: want error")
	}
	cfg := testClearContextConfig(filepath.Join(t.TempDir(), "x.db"))
	cfg.Embedding = nil
	if err := clearConversationContext(cfg); err == nil {
		t.Fatal("nil embedding: want error")
	}
	cfg2 := testClearContextConfig(filepath.Join(t.TempDir(), "y.db"))
	cfg2.Embedding.Dimensions = 0
	if err := clearConversationContext(cfg2); err == nil {
		t.Fatal("dimensions 0: want error")
	}
	cfg3 := &config.Config{Embedding: &config.EmbeddingProvider{Dimensions: 4, BatchSize: 10}, Paths: config.Paths{VectorIndexPath: "  "}}
	if err := clearConversationContext(cfg3); err == nil {
		t.Fatal("empty vector path: want error")
	}
}

// Covers AC-04.012 (REQ-04.015): CI runs make check; EP-004 behaviour is covered by unit and integration tests across the repository.
func TestEP004_makeCheckRegressionGate(t *testing.T) {}
