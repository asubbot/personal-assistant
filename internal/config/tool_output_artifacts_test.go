package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validToolOutputArtifactsBlock = `{
      "enabled": true,
      "directory": "tool_artifacts",
      "tool_result_prompt_bytes": 8192,
      "max_artifact_bytes": 10485760,
      "omission_marker": "...[omitted]...",
      "preview_min_tail_bytes": 128,
      "max_stderr_bytes_in_prompt": 2048,
      "max_reads_per_turn": 32,
      "max_read_bytes_per_turn": 524288,
      "max_bytes_per_read": 4096,
      "retention_max_total_bytes": 1073741824,
      "retention_max_files": 1000
    }`

func testConfigWithToolOutputArtifacts(toolsInner string) string {
	return testConfigWithVectorSearchTools(`{
    "selection": { "tool_search_top_k": 10, "tool_min_count": 1, "tool_fallback_cap": 50, "enabled": false, "max_tools_for_llm_request": 0 },
    "tool_output_artifacts": ` + toolsInner + `
  }`)
}

// Covers AC-39.005
func TestLoad_ToolOutputArtifacts_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigAndCatalog(t, dir, testConfigWithToolOutputArtifacts(validToolOutputArtifactsBlock))
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Tools == nil || cfg.Tools.ToolOutputArtifacts == nil {
		t.Fatal("expected typed ToolOutputArtifacts on ToolsConfig")
	}
	a := cfg.Tools.ToolOutputArtifacts
	if !a.Enabled || a.ToolResultPromptBytes != 8192 || a.MaxArtifactBytes != 10485760 {
		t.Fatalf("ToolOutputArtifacts = %+v, want operator defaults", a)
	}
}

// Covers AC-39.005
func TestLoad_ToolOutputArtifacts_InvalidBounds(t *testing.T) {
	dir := t.TempDir()
	block := strings.Replace(validToolOutputArtifactsBlock, `"tool_result_prompt_bytes": 8192`, `"tool_result_prompt_bytes": 0`, 1)
	path := writeConfigAndCatalog(t, dir, testConfigWithToolOutputArtifacts(block))
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error for tool_result_prompt_bytes=0, got nil")
	}
	if !strings.Contains(err.Error(), "tool_result_prompt_bytes") {
		t.Fatalf("Load: error = %v, want tool_result_prompt_bytes validation", err)
	}
}

// Covers AC-39.007
func TestLoad_ToolOutputArtifacts_UnknownNestedKey(t *testing.T) {
	path := filepath.Join("testdata", "tool_output_artifacts_unknown_key.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tools.tool_output_artifacts") || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("Load: error = %v, want unknown nested key rejection", err)
	}
}

// Covers AC-39.006, AC-39.010
func TestArtifactDirectory_ResolvesRelativeToDataDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PA_DATA_DIR", dataDir)
	path := writeConfigAndCatalog(t, dir, testConfigWithToolOutputArtifacts(validToolOutputArtifactsBlock))
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := ArtifactDirectory(cfg)
	want := filepath.Join(dataDir, "tool_artifacts")
	if got != want {
		t.Fatalf("ArtifactDirectory = %q, want %q", got, want)
	}
}

// Covers AC-39.006
func TestArtifactDirectory_DisabledReturnsEmpty(t *testing.T) {
	block := strings.Replace(validToolOutputArtifactsBlock, `"enabled": true`, `"enabled": false`, 1)
	dir := t.TempDir()
	path := writeConfigAndCatalog(t, dir, testConfigWithToolOutputArtifacts(block))
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := ArtifactDirectory(cfg); got != "" {
		t.Fatalf("ArtifactDirectory when disabled = %q, want empty", got)
	}
}
