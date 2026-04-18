package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
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
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

// Covers AC-24.007, AC-24.008. Supporting AC-24.010: assertions run inside `go test ./cmd/pa` executed by `make check`.
func TestEP024_ProductionDockerDefaults(t *testing.T) {
	root := repoRoot(t)
	df, err := os.ReadFile(filepath.Join(root, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	dfs := string(df)
	if !strings.Contains(dfs, "ENV PA_LOG_LEVEL=info") {
		t.Fatal("Dockerfile must set ENV PA_LOG_LEVEL=info")
	}
	if strings.Contains(dfs, "ENV PA_LOG_LEVEL=debug") {
		t.Fatal("Dockerfile must not set PA_LOG_LEVEL to debug")
	}

	cf, err := os.ReadFile(filepath.Join(root, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfs := string(cf)
	if !strings.Contains(cfs, "PA_LOG_LEVEL=${PA_LOG_LEVEL:-info}") {
		t.Fatal("docker-compose.yml must default PA_LOG_LEVEL to info via interpolation")
	}
	if strings.Contains(cfs, "PA_LOG_LEVEL=debug") {
		t.Fatal("docker-compose.yml must not hard-code PA_LOG_LEVEL=debug")
	}
}

// Covers AC-24.001, AC-24.002, AC-24.003, AC-24.004, AC-24.005, AC-24.006
func TestEP024_ProviderRolesDocContent(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "llm-provider-roles-and-logging.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	checks := []string{
		"llm_providers",
		"zero-based",
		"baseline_index",
		"SummarizeRouterConfig",
		"intent_classifier",
		"model_stage",
		"not selected by an index",
		"PA_ENV",
		"development",
		"## Example: single-provider",
		"## Example: escalation-enabled",
		"## Example: pool with intent classifier",
	}
	for _, sub := range checks {
		if !strings.Contains(s, sub) {
			t.Fatalf("docs/llm-provider-roles-and-logging.md missing %q", sub)
		}
	}
}

// Covers AC-24.009
func TestEP024_SensitiveLoggingWarning(t *testing.T) {
	cases := []struct {
		name     string
		paEnv    string
		level    slog.Level
		wantWarn int
	}{
		{"debug empty PA_ENV", "", slog.LevelDebug, 1},
		{"debug PA_ENV staging", "staging", slog.LevelDebug, 1},
		{"debug PA_ENV development", "development", slog.LevelDebug, 0},
		{"debug PA_ENV DEVELOPMENT", "DEVELOPMENT", slog.LevelDebug, 0},
		{"info empty PA_ENV", "", slog.LevelInfo, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PA_ENV", tc.paEnv)
			var buf strings.Builder
			h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
			log := slog.New(h)
			warnSensitiveLLMLogging(log, tc.level)
			out := buf.String()
			got := strings.Count(out, "level=WARN")
			if got != tc.wantWarn {
				t.Fatalf("WARN count=%d want=%d log=%q", got, tc.wantWarn, out)
			}
			if tc.wantWarn > 0 && !strings.Contains(out, "full LLM") {
				t.Fatalf("expected sensitive-logging message to mention full LLM payloads, got %q", out)
			}
		})
	}
}
