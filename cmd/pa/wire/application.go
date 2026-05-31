package wire

import (
	"context"
	"fmt"
	"log/slog"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/jobs"
	"pa/internal/llm"
	"pa/internal/llmrouter"
	"pa/internal/memoryjob"
	"pa/internal/tools"
	"path/filepath"
	"sync"
)

// Application is the composition root output: runnable handler wiring and coordinated teardown (EP-027, EP-042).
type Application struct {
	Cfg        *config.Config
	ConfigPath string
	Logger     *slog.Logger

	Infra Infrastructure

	LLMProviders []llm.Provider
	LLMLabels    []string
	SummarizeLLM llm.Provider

	MemJob    *memoryjob.Runner
	JobsState *JobsRuntimeState
}

// Close releases infrastructure acquired during construction (vector stores, indices).
func (a *Application) Close() {
	if a == nil {
		return
	}
	a.Infra.Close(a.Logger)
}

// StartLLMProviders loads LLM provider handles from configuration.
func (a *Application) StartLLMProviders() error {
	providers, labels, summarizeLLM, err := BuildAppLLM(a.Cfg, a.Logger)
	if err != nil {
		return err
	}
	a.LLMProviders = providers
	a.LLMLabels = labels
	a.SummarizeLLM = summarizeLLM
	return nil
}

// MaybeStartMemorySummarization starts the background memory summarization worker when configured.
func (a *Application) MaybeStartMemorySummarization(ctx context.Context) error {
	if a.Infra.MemoryStore == nil || a.Cfg.Paths.LLMLogDir == "" || a.Infra.Embedder == nil ||
		a.Infra.MemVec == nil || a.Infra.MemVec.Summaries == nil {
		return nil
	}
	paLoc, locErr := config.PALocation(a.Cfg)
	if locErr != nil {
		return fmt.Errorf("memory summarization pa_timezone: %w", locErr)
	}
	a.MemJob = memoryjob.Start(ctx, memoryjob.Deps{
		Cfg:         a.Cfg,
		Loc:         paLoc,
		Memory:      a.Infra.MemoryStore,
		Vector:      a.Infra.MemVec.Summaries,
		Embedder:    a.Infra.Embedder,
		LLMProvider: a.SummarizeLLM,
		Logger:      a.Logger,
	})
	a.Logger.Info("memory summarization worker started")
	return nil
}

// StopMemorySummarization stops the background memory summarization worker when running.
func (a *Application) StopMemorySummarization() {
	if a.MemJob != nil {
		a.MemJob.Stop()
		<-a.MemJob.Done()
	}
}

// BuildToolRegistry registers native tools for the running application.
func (a *Application) BuildToolRegistry() (*tools.Registry, error) {
	toolRegistry := tools.NewRegistry()
	if a.Cfg.Paths.JobsDBPath != "" {
		a.JobsState = NewJobsRuntimeState()
		toolRegistry.Register(jobs.NewCreateScheduledJobToolWithRuntimeLookup(func() (*jobs.Manager, bool, bool) {
			if a.JobsState == nil {
				return nil, false, false
			}
			return a.JobsState.SnapshotLegacy()
		}))
	}
	if a.Infra.NodeRunner != nil {
		toolRegistry.Register(tools.NewRunOnNode(a.Infra.NodeRunner))
	}
	absCatalog, err := filepath.Abs(a.Cfg.Paths.ToolCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("tool catalog path: %w", err)
	}
	var createToolMu sync.Mutex
	if a.Cfg.ToolCatalog != nil {
		toolRegistry.Register(tools.NewCreateTool(&createToolMu, a.Cfg.ToolCatalog, absCatalog, a.Cfg, a.Infra.Embedder, a.Infra.ToolIndex, a.Logger))
	}
	if err := RegisterWebToolsIfEnabled(a.Cfg, toolRegistry, a.Logger); err != nil {
		return nil, err
	}
	if err := RegisterMemoryToolsIfEnabled(a.Cfg, toolRegistry, a.Infra.MemoryStore, a.Infra.MemVec, a.Infra.Embedder); err != nil {
		return nil, err
	}
	RegisterKnowledgeToolsIfEnabled(a.Cfg, toolRegistry, a.Infra.ToolIndex, a.Infra.SkillIndex, a.Infra.Embedder)
	return toolRegistry, nil
}

// BuildMessageHandler assembles the core conversation handler (jobs wrapper is applied in cmd/pa).
func (a *Application) BuildMessageHandler(_ context.Context, toolRegistry *tools.Registry) (core.MessageHandler, error) {
	classifier := BuildIntentClassifier(a.Cfg, a.Logger)
	var ti core.ToolIndex = a.Infra.ToolIndex
	var si core.SkillIndex = a.Infra.SkillIndex
	router, err := llmrouter.New(a.LLMProviders, a.LLMLabels, llmrouter.Config{}, a.Logger)
	if err != nil {
		return nil, err
	}
	return core.BuildMessageHandler(
		a.Cfg,
		a.Logger,
		core.BuildLogRedactor(a.Cfg),
		router,
		a.Infra.MemVec,
		a.Infra.Embedder,
		a.Infra.NodeRunner,
		ti,
		si,
		toolRegistry,
		classifier,
	)
}
