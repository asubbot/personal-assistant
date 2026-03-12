package main

import (
	"os"
	"os/exec"
	"pa/internal/config"
	"path/filepath"
	"testing"
)

// Covers AC-042 (US-20): config path resolved from PA_CONFIG_DIR when set.
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

// Covers AC-042 (US-20): when PA_CONFIG_DIR unset or empty, documented default is used.
func TestConfigFilePath_PAConfigDirUnsetOrEmpty(t *testing.T) {
	_ = os.Unsetenv("PA_CONFIG_DIR")
	t.Cleanup(func() { _ = os.Unsetenv("PA_CONFIG_DIR") })

	got := configFilePath()
	want := filepath.Join("./config", config.ConfigFileName)
	if got != want {
		t.Errorf("configFilePath() = %q, want %q", got, want)
	}
}

// runSummarizeCLI runs `go run ./cmd/pa -summarize=<value>` with minimal config in dir; expects exit 0 (e.g. skip when no data).
func runSummarizeCLI(t *testing.T, dir, summarizeValue string) {
	t.Helper()
	cfgPath := filepath.Join(dir, config.ConfigFileName)
	cfg := `{
  "version": 1,
  "telegram": { "token_path": "t", "users_path": "" },
  "llm_providers": [{ "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m" }],
  "paths": {
    "memory_dir": "memory",
    "log_path": "pa.log",
    "vector_index_path": "vec.sqlite",
    "llm_log_dir": "llm_logs",
    "scheduled_tasks_path": ""
  },
  "embedding": { "type": "ollama", "endpoint": "http://127.0.0.1:11434", "model": "m", "dimensions": 4 },
  "nodes": {}
}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfgDir := filepath.Dir(cfgPath)

	wd, _ := os.Getwd()
	moduleRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(moduleRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(moduleRoot)
		if parent == moduleRoot {
			t.Skip("could not find go.mod (not in module?)")
		}
		moduleRoot = parent
	}
	cmd := exec.Command("go", "run", "./cmd/pa", "-summarize="+summarizeValue)
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "PA_CONFIG_DIR="+cfgDir, "PA_DATA_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("output: %s", out)
		t.Fatalf("run -summarize=%s: %v", summarizeValue, err)
	}
	if len(out) > 0 {
		t.Logf("output: %s", out)
	}
}

// Covers AC-011, AC-012 (US-06): day summarization CLI -summarize=YYYY-MM-DD runs without starting the bot and exits 0 when successful (e.g. no entries to summarize).
func TestSummarizeCLI_day_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026-03-12")
}

// Supporting AC-011, AC-012 (US-06): month summarization CLI -summarize=YYYY-MM exits 0 when no day summaries (skip).
func TestSummarizeCLI_month_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026-03")
}

// Supporting AC-011, AC-012 (US-06): year summarization CLI -summarize=YYYY exits 0 when no month summaries (skip).
func TestSummarizeCLI_year_exitZero(t *testing.T) {
	runSummarizeCLI(t, t.TempDir(), "2026")
}
