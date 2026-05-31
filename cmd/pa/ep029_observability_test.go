package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/llm"
	"pa/internal/testutil"
	"pa/internal/toolindex"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stubLLMProvider implements llm.Provider for readiness tests (EP-029).
type stubLLMProvider struct{}

func (stubLLMProvider) Complete(ctx context.Context, messages []llm.Message, opts *llm.CompletionOptions) (*llm.CompletionResult, error) {
	_ = ctx
	_ = messages
	_ = opts
	return &llm.CompletionResult{Content: "ok"}, nil
}

// Covers AC-29.002, AC-29.003, AC-29.004: readiness aggregates subsystem checks.
func TestEvalReadiness_AllOKWithoutJobsOrMemoryWorker(t *testing.T) {
	idx := toolindex.NewIndex(noopVectorStore{})
	idx.SetReady(true)
	app := &paApplication{
		Cfg: &config.Config{
			Paths: config.Paths{},
			ObservabilityHTTP: &config.ObservabilityHTTPConfig{
				ListenAddress: "unused",
				HealthPath:    "/health",
				ReadinessPath: "/ready",
				ProbeLLM:      false,
			},
		},
		LLMProviders: []llm.Provider{stubLLMProvider{}},
		Infra: paInfrastructure{
			MemVec: &core.MemoryVectors{
				Summaries: noopVectorStore{},
				Turns:     noopVectorStore{},
				Notes:     noopVectorStore{},
			},
			ToolIndex: idx,
		},
	}
	body := app.EvalReadiness(context.Background())
	if !body.Ready {
		t.Fatalf("expected ready, got %+v", body)
	}
}

// Covers AC-29.002: health returns liveness JSON independent of readiness.
func TestObservabilityHTTPHandler_HealthAndReadiness(t *testing.T) {
	idx := toolindex.NewIndex(noopVectorStore{})
	idx.SetReady(true)
	app := &paApplication{
		Cfg: &config.Config{
			Paths: config.Paths{},
			ObservabilityHTTP: &config.ObservabilityHTTPConfig{
				ListenAddress: "unused",
				HealthPath:    "/health",
				ReadinessPath: "/ready",
				ProbeLLM:      false,
			},
		},
		LLMProviders: []llm.Provider{stubLLMProvider{}},
		Infra: paInfrastructure{
			MemVec: &core.MemoryVectors{
				Summaries: noopVectorStore{},
				Turns:     noopVectorStore{},
				Notes:     noopVectorStore{},
			},
			ToolIndex: idx,
		},
	}
	srv := httptest.NewServer(observabilityHTTPHandler(app.Cfg, app, slog.New(slog.DiscardHandler)))
	t.Cleanup(srv.Close)

	client := &http.Client{Timeout: 5 * time.Second}
	reqCtx := context.Background()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, srv.URL+"/health", http.NoBody)
	if err != nil {
		t.Fatalf("health request: %v", err)
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	func() {
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("health status = %d", res.StatusCode)
		}
		b, errRead := io.ReadAll(res.Body)
		if errRead != nil {
			t.Fatalf("read body: %v", errRead)
		}
		var h map[string]string
		if err := json.Unmarshal(b, &h); err != nil {
			t.Fatalf("health json: %v", err)
		}
		if h["status"] != "alive" {
			t.Fatalf("health body = %#v", h)
		}
	}()

	req2, err := http.NewRequestWithContext(reqCtx, http.MethodGet, srv.URL+"/ready", http.NoBody)
	if err != nil {
		t.Fatalf("ready request: %v", err)
	}
	res2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("GET ready: %v", err)
	}
	defer func() { _ = res2.Body.Close() }()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("readiness status = %d", res2.StatusCode)
	}
}

// Covers AC-29.007: operator observability documentation is present and mentions key operator concepts.
func TestEP029_operatorObservabilityDoc(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "docs", "observability-http.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, needle := range []string{
		"observability_http",
		"listen_address",
		"health_path",
		"readiness_path",
		"probe_llm",
		"HEALTHCHECK",
		"lifecycle_event",
		"subsystem",
		"lifecycle_phase",
		"duration_ms",
		"Bind failures",
		"Duplicate log lines",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("docs/observability-http.md missing %q", needle)
		}
	}
}

// Covers AC-29.008. Entrypoint: ./bin/validate EP-029 (via testutil.EnsureValidator).
func TestEP029_validateCommandExitZero(t *testing.T) {
	root := repoRoot(t)
	testutil.RunValidateEpic(t, root, "EP-029")
}

// Covers AC-29.003: readiness is not OK while the tool index is still building.
func TestEvalReadiness_ToolIndexNotReady(t *testing.T) {
	idx := toolindex.NewIndex(noopVectorStore{})
	app := &paApplication{
		Cfg: &config.Config{
			Paths: config.Paths{},
			ObservabilityHTTP: &config.ObservabilityHTTPConfig{
				ListenAddress: "unused",
				HealthPath:    "/health",
				ReadinessPath: "/ready",
				ProbeLLM:      false,
			},
		},
		LLMProviders: []llm.Provider{stubLLMProvider{}},
		Infra: paInfrastructure{
			MemVec: &core.MemoryVectors{
				Summaries: noopVectorStore{},
				Turns:     noopVectorStore{},
				Notes:     noopVectorStore{},
			},
			ToolIndex: idx,
		},
	}
	body := app.EvalReadiness(context.Background())
	if body.Ready {
		t.Fatal("expected not ready when tool index is not ready")
	}
}
