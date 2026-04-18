package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"pa/internal/llm"
	"time"
)

const llmReadinessProbeTimeout = 5 * time.Second

// readinessCheck is one subsystem verdict for the readiness HTTP body (EP-029).
type readinessCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// readinessBody is the JSON envelope for GET readiness.
type readinessBody struct {
	Ready  bool             `json:"ready"`
	Checks []readinessCheck `json:"checks"`
}

func (a *paApplication) evalReadiness(ctx context.Context) readinessBody {
	if a == nil || a.cfg == nil {
		return readinessBody{Ready: false, Checks: []readinessCheck{{Name: "application", OK: false, Detail: "nil application"}}}
	}
	var checks []readinessCheck
	allOK := true

	add := func(name string, ok bool, detail string) {
		checks = append(checks, readinessCheck{Name: name, OK: ok, Detail: detail})
		if !ok {
			allOK = false
		}
	}

	a.appendLLMReadinessChecks(ctx, add)
	a.appendVectorReadinessChecks(add)
	a.appendToolIndexReadinessChecks(add)
	a.appendJobsReadinessChecks(add)
	a.appendMemorySummarizationReadinessChecks(add)

	return readinessBody{Ready: allOK, Checks: checks}
}

func (a *paApplication) appendLLMReadinessChecks(ctx context.Context, add func(string, bool, string)) {
	if len(a.llmProviders) == 0 {
		add("llm", false, "no providers loaded")
		return
	}
	probe := a.cfg.ObservabilityHTTP != nil && a.cfg.ObservabilityHTTP.ProbeLLM
	if !probe {
		add("llm", true, "providers loaded (probe_llm false)")
		return
	}
	pctx, cancel := context.WithTimeout(ctx, llmReadinessProbeTimeout)
	defer cancel()
	if err := pingLLMProvider(pctx, a.llmProviders[0]); err != nil {
		add("llm", false, fmt.Sprintf("probe failed: %v", err))
		return
	}
	add("llm", true, "provider responded to probe")
}

func (a *paApplication) appendVectorReadinessChecks(add func(string, bool, string)) {
	mv := a.infra.MemVec
	if mv == nil || mv.Summaries == nil || mv.Turns == nil || mv.Notes == nil {
		add("vector_stores", false, "memory vector bundle incomplete")
		return
	}
	add("vector_stores", true, "")
}

func (a *paApplication) appendToolIndexReadinessChecks(add func(string, bool, string)) {
	switch {
	case a.infra.ToolIndex == nil:
		add("tool_index", false, "tool index not constructed")
	case !a.infra.ToolIndex.Ready():
		add("tool_index", false, "index build not finished")
	default:
		add("tool_index", true, "")
	}
}

func (a *paApplication) appendJobsReadinessChecks(add func(string, bool, string)) {
	if a.cfg.Paths.JobsDBPath == "" {
		return
	}
	if a.jobsState == nil {
		add("scheduled_jobs", false, "runtime state not initialized")
		return
	}
	_, ready, failed := a.jobsState.snapshot()
	switch {
	case failed:
		add("scheduled_jobs", false, "initialization failed")
	case !ready:
		add("scheduled_jobs", false, "initializing")
	default:
		add("scheduled_jobs", true, "")
	}
}

func (a *paApplication) appendMemorySummarizationReadinessChecks(add func(string, bool, string)) {
	memConfigured := a.infra.MemoryStore != nil && a.cfg.Paths.LLMLogDir != "" && a.infra.Embedder != nil &&
		a.infra.MemVec != nil && a.infra.MemVec.Summaries != nil
	if !memConfigured {
		return
	}
	if a.memJob == nil {
		add("memory_summarization", false, "worker not started")
		return
	}
	add("memory_summarization", true, "")
}

func pingLLMProvider(ctx context.Context, p llm.Provider) error {
	if p == nil {
		return fmt.Errorf("nil provider")
	}
	temp := 0.0
	_, err := p.Complete(ctx, []llm.Message{{Role: "user", Content: "."}}, &llm.CompletionOptions{
		MaxTokens:   1,
		Temperature: &temp,
	})
	return err
}

func writeJSON(logger *slog.Logger, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil && logger != nil {
		logger.Error("observability json encode", "error", err)
	}
}
