package main

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

// paApplication is the composition root output: runnable handler wiring and coordinated teardown (EP-027).
type paApplication struct {
	cfg        *config.Config
	configPath string
	logger     *slog.Logger

	infra paInfrastructure

	llmProviders []llm.Provider
	llmLabels    []string
	summarizeLLM llm.Provider

	memJob    *memoryjob.Runner
	jobsState *jobsRuntimeState
}

func newPAApplication(cfg *config.Config, configPath string, logger *slog.Logger) (*paApplication, error) {
	infra, err := buildPAInfrastructure(cfg, configPath, logger)
	if err != nil {
		return nil, err
	}
	return &paApplication{
		cfg:        cfg,
		configPath: configPath,
		logger:     logger,
		infra:      infra,
	}, nil
}

// Close releases infrastructure acquired during construction (vector stores, indices).
func (a *paApplication) Close() {
	if a == nil {
		return
	}
	a.infra.close(a.logger)
}

func (a *paApplication) startLLMProviders() error {
	providers, labels, summarizeLLM, err := buildAppLLM(a.cfg, a.logger)
	if err != nil {
		return err
	}
	a.llmProviders = providers
	a.llmLabels = labels
	a.summarizeLLM = summarizeLLM
	return nil
}

func (a *paApplication) maybeStartMemorySummarization(ctx context.Context) error {
	if a.infra.MemoryStore == nil || a.cfg.Paths.LLMLogDir == "" || a.infra.Embedder == nil ||
		a.infra.MemVec == nil || a.infra.MemVec.Summaries == nil {
		return nil
	}
	paLoc, locErr := config.PALocation(a.cfg)
	if locErr != nil {
		return fmt.Errorf("memory summarization pa_timezone: %w", locErr)
	}
	a.memJob = memoryjob.Start(ctx, memoryjob.Deps{
		Cfg:         a.cfg,
		Loc:         paLoc,
		Memory:      a.infra.MemoryStore,
		Vector:      a.infra.MemVec.Summaries,
		Embedder:    a.infra.Embedder,
		LLMProvider: a.summarizeLLM,
		Logger:      a.logger,
	})
	a.logger.Info("memory summarization worker started")
	return nil
}

func (a *paApplication) stopMemorySummarization() {
	if a.memJob != nil {
		a.memJob.Stop()
		<-a.memJob.Done()
	}
}

func (a *paApplication) buildToolRegistry() (*tools.Registry, error) {
	toolRegistry := tools.NewRegistry()
	if a.cfg.Paths.JobsDBPath != "" {
		a.jobsState = &jobsRuntimeState{}
		toolRegistry.Register(jobs.NewCreateScheduledJobToolWithRuntimeLookup(func() (*jobs.Manager, bool, bool) {
			if a.jobsState == nil {
				return nil, false, false
			}
			return a.jobsState.snapshot()
		}))
	}
	if a.infra.NodeRunner != nil {
		toolRegistry.Register(tools.NewRunOnNode(a.infra.NodeRunner))
	}
	absCatalog, err := filepath.Abs(a.cfg.Paths.ToolCatalogPath)
	if err != nil {
		return nil, fmt.Errorf("tool catalog path: %w", err)
	}
	var createToolMu sync.Mutex
	if a.cfg.ToolCatalog != nil {
		toolRegistry.Register(tools.NewCreateTool(&createToolMu, a.cfg.ToolCatalog, absCatalog, a.cfg, a.infra.Embedder, a.infra.ToolIndex, a.logger))
	}
	if err := registerWebToolsIfEnabled(a.cfg, toolRegistry, a.logger); err != nil {
		return nil, err
	}
	if err := registerMemoryToolsIfEnabled(a.cfg, toolRegistry, a.infra.MemoryStore, a.infra.MemVec, a.infra.Embedder); err != nil {
		return nil, err
	}
	registerKnowledgeToolsIfEnabled(a.cfg, toolRegistry, a.infra.ToolIndex, a.infra.SkillIndex, a.infra.Embedder)
	return toolRegistry, nil
}

func (a *paApplication) buildMessageHandler(ctx context.Context, toolRegistry *tools.Registry) (core.MessageHandler, error) {
	classifier := buildIntentClassifier(a.cfg, a.logger)
	var ti core.ToolIndex = a.infra.ToolIndex
	var si core.SkillIndex = a.infra.SkillIndex
	router, err := llmrouter.New(a.llmProviders, a.llmLabels, llmrouter.Config{}, a.logger)
	if err != nil {
		return nil, err
	}
	baseHandler, err := core.BuildMessageHandler(
		a.cfg,
		a.logger,
		core.BuildLogRedactor(a.cfg),
		router,
		a.infra.MemVec,
		a.infra.Embedder,
		a.infra.NodeRunner,
		ti,
		si,
		toolRegistry,
		classifier,
	)
	if err != nil {
		return nil, err
	}
	handler := core.MessageHandler(baseHandler)
	if a.cfg.Paths.JobsDBPath != "" {
		state := a.jobsState
		if state == nil {
			state = &jobsRuntimeState{}
			a.jobsState = state
		}
		handler = &jobsCommandHandler{
			base:  baseHandler,
			state: state,
		}
		if sender, ok := a.infra.Adapter.(chatSender); ok {
			initJobsRuntimeAsync(ctx, state, a.cfg.Paths.JobsDBPath, a.cfg.JobsStoreReliabilityPolicy(), a.cfg.PATimezone, sender, baseHandler, a.logger)
		} else {
			err := fmt.Errorf("adapter does not support direct chat sending")
			a.logger.Warn("jobs runtime delivery disabled", "error", err)
			state.setInitError(err)
		}
	}
	return handler, nil
}
