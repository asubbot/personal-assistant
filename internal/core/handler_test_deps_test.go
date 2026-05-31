package core

import (
	"log/slog"
	"pa/internal/config"
	"pa/internal/embedding"
	"pa/internal/intent"
	"pa/internal/llmlog"
	"pa/internal/llmrouter"
	"pa/internal/runtimeskills"
	"pa/internal/toolcatalog"
	"pa/internal/tools"
	"time"
)

// testHandlerDeps holds flat dependency fields for building conversationHandler in unit tests (EP-040).
type testHandlerDeps struct {
	catalog                    *toolcatalog.Catalog
	toolIndex                  ToolIndex
	skillIndex                 SkillIndex
	nativeRegistry             *tools.Registry
	skillPackagesByID          map[string]*runtimeskills.Package
	toolsCfg                   *config.ToolsConfig
	toolsSelection             *config.ToolsSelection
	toolSearchTopK             int
	toolMinCount               int
	toolFallbackCap            int
	nodeRunner                 NodeRunner
	runtimeSkillsCfg           *config.RuntimeSkillsConfig
	memVec                     *MemoryVectors
	embedder                   embedding.Embedder
	memoryVectorTopK           config.MemoryVectorConfig
	paLoc                      *time.Location
	sessionCfg                 *config.ConversationSessionConfig
	sessionStore               *sessionWindowStore
	router                     *llmrouter.Router
	llmLog                     llmlog.Writer
	model                      string
	firstProviderSupportsTools bool
	logRedactor                func(string) string
	logger                     *slog.Logger
	classifier                 intent.Classifier
	maxMessageLength           int
	maxDynamicSystemRunes      int
	toolResultPromptBytes      int
}

func (d testHandlerDeps) handler() *conversationHandler {
	return &conversationHandler{
		tools: handlerToolDeps{
			catalog:           d.catalog,
			toolIndex:         d.toolIndex,
			skillIndex:        d.skillIndex,
			nativeRegistry:    d.nativeRegistry,
			skillPackagesByID: d.skillPackagesByID,
			toolsCfg:          d.toolsCfg,
			toolsSelection:    d.toolsSelection,
			toolSearchTopK:    d.toolSearchTopK,
			toolMinCount:      d.toolMinCount,
			toolFallbackCap:   d.toolFallbackCap,
			nodeRunner:        d.nodeRunner,
			runtimeSkillsCfg:  d.runtimeSkillsCfg,
		},
		memory: handlerMemoryDeps{
			memVec:           d.memVec,
			embedder:         d.embedder,
			memoryVectorTopK: d.memoryVectorTopK,
			paLoc:            d.paLoc,
		},
		session: handlerSessionDeps{
			sessionCfg:   d.sessionCfg,
			sessionStore: d.sessionStore,
		},
		llm: handlerLLMDeps{
			router:                     d.router,
			llmLog:                     d.llmLog,
			model:                      d.model,
			firstProviderSupportsTools: d.firstProviderSupportsTools,
			logRedactor:                d.logRedactor,
			logger:                     d.logger,
			classifier:                 d.classifier,
			maxMessageLength:           d.maxMessageLength,
			maxDynamicSystemRunes:      d.maxDynamicSystemRunes,
		},
		toolResultPromptBytes: d.toolResultPromptBytes,
	}
}
