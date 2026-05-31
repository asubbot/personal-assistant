package wire

import (
	"context"
	"fmt"
	"pa/internal/llm"
	"time"
)

const llmReadinessProbeTimeout = 5 * time.Second

// ReadinessCheck is one subsystem verdict for the readiness HTTP body (EP-029).
type ReadinessCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

// ReadinessBody is the JSON envelope for GET readiness.
type ReadinessBody struct {
	Ready  bool             `json:"ready"`
	Checks []ReadinessCheck `json:"checks"`
}

// EvalReadiness aggregates subsystem readiness for the observability HTTP handler.
func (a *Application) EvalReadiness(ctx context.Context) ReadinessBody {
	if a == nil || a.Cfg == nil {
		return ReadinessBody{Ready: false, Checks: []ReadinessCheck{{Name: "application", OK: false, Detail: "nil application"}}}
	}
	var checks []ReadinessCheck
	allOK := true

	add := func(name string, ok bool, detail string) {
		checks = append(checks, ReadinessCheck{Name: name, OK: ok, Detail: detail})
		if !ok {
			allOK = false
		}
	}

	a.appendLLMReadinessChecks(ctx, add)
	a.appendVectorReadinessChecks(add)
	a.appendToolIndexReadinessChecks(add)
	a.appendJobsReadinessChecks(add)
	a.appendMemorySummarizationReadinessChecks(add)

	return ReadinessBody{Ready: allOK, Checks: checks}
}

func (a *Application) appendLLMReadinessChecks(ctx context.Context, add func(string, bool, string)) {
	if len(a.LLMProviders) == 0 {
		add("llm", false, "no providers loaded")
		return
	}
	probe := a.Cfg.ObservabilityHTTP != nil && a.Cfg.ObservabilityHTTP.ProbeLLM
	if !probe {
		add("llm", true, "providers loaded (probe_llm false)")
		return
	}
	pctx, cancel := context.WithTimeout(ctx, llmReadinessProbeTimeout)
	defer cancel()
	if err := pingLLMProvider(pctx, a.LLMProviders[0]); err != nil {
		add("llm", false, fmt.Sprintf("probe failed: %v", err))
		return
	}
	add("llm", true, "provider responded to probe")
}

func (a *Application) appendVectorReadinessChecks(add func(string, bool, string)) {
	mv := a.Infra.MemVec
	if mv == nil || mv.Summaries == nil || mv.Turns == nil || mv.Notes == nil {
		add("vector_stores", false, "memory vector bundle incomplete")
		return
	}
	add("vector_stores", true, "")
}

func (a *Application) appendToolIndexReadinessChecks(add func(string, bool, string)) {
	switch {
	case a.Infra.ToolIndex == nil:
		add("tool_index", false, "tool index not constructed")
	case !a.Infra.ToolIndex.Ready():
		add("tool_index", false, "index build not finished")
	default:
		add("tool_index", true, "")
	}
}

func (a *Application) appendJobsReadinessChecks(add func(string, bool, string)) {
	if a.Cfg.Paths.JobsDBPath == "" {
		return
	}
	if a.JobsState == nil {
		add("scheduled_jobs", false, "runtime state not initialized")
		return
	}
	_, phase := a.JobsState.Snapshot()
	switch phase {
	case JobsRuntimeFailed:
		add("scheduled_jobs", false, "initialization failed")
	case JobsRuntimeReady:
		add("scheduled_jobs", true, "")
	default:
		add("scheduled_jobs", false, "initializing")
	}
}

func (a *Application) appendMemorySummarizationReadinessChecks(add func(string, bool, string)) {
	memConfigured := a.Infra.MemoryStore != nil && a.Cfg.Paths.LLMLogDir != "" && a.Infra.Embedder != nil &&
		a.Infra.MemVec != nil && a.Infra.MemVec.Summaries != nil
	if !memConfigured {
		return
	}
	if a.MemJob == nil {
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
