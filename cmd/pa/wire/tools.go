package wire

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
	"pa/internal/memory"
	"pa/internal/skillindex"
	"pa/internal/toolindex"
	"pa/internal/tools"
	"pa/internal/vector/sqlite"
	"strings"
	"time"
)

// NewToolIndex creates the tool vector store and starts building the index from the catalog in the background.
func NewToolIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*toolindex.Index, error) {
	if cfg.ToolCatalog == nil {
		return nil, fmt.Errorf("tool catalog is required")
	}
	toolVectorStore, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableTools, cfg.VectorStoreReliabilityPolicy())
	if err != nil {
		return nil, err
	}
	idx := toolindex.NewIndex(toolVectorStore)
	catalog := cfg.ToolCatalog
	go func() {
		t0 := time.Now()
		err := idx.BuildAndSetReady(context.Background(), catalog, embedder)
		toolindex.LogBuildOutcome(logger, len(catalog.Tools), time.Since(t0), err)
	}()
	return idx, nil
}

func newSkillIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*skillindex.Index, error) {
	if cfg.RuntimeSkills == nil || !cfg.RuntimeSkills.Enabled || len(cfg.RuntimeSkillPackages) == 0 {
		return nil, nil
	}
	store, err := sqlite.NewWithTable(cfg.Paths.VectorIndexPath, cfg.Embedding.Dimensions, sqlite.TableSkills, cfg.VectorStoreReliabilityPolicy())
	if err != nil {
		return nil, err
	}
	idx := skillindex.NewIndex(store)
	if err := idx.BuildAndSetReady(context.Background(), cfg.RuntimeSkillPackages, embedder); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("skill index: %w", err)
	}
	logger.Info("runtime skill index ready", "packages", len(cfg.RuntimeSkillPackages))
	return idx, nil
}

func openMemoryVectorBundle(cfg *config.Config) (*core.MemoryVectors, error) {
	dim := cfg.Embedding.Dimensions
	path := cfg.Paths.VectorIndexPath
	vecPolicy := cfg.VectorStoreReliabilityPolicy()
	summ, err := sqlite.NewWithTable(path, dim, sqlite.TableSummaries, vecPolicy)
	if err != nil {
		return nil, err
	}
	turns, err := sqlite.NewWithTable(path, dim, sqlite.TableTurns, vecPolicy)
	if err != nil {
		_ = summ.Close()
		return nil, err
	}
	notes, err := sqlite.NewWithTable(path, dim, sqlite.TableNotes, vecPolicy)
	if err != nil {
		_ = summ.Close()
		_ = turns.Close()
		return nil, err
	}
	return &core.MemoryVectors{
		Summaries: summ,
		Turns:     turns,
		Notes:     notes,
	}, nil
}

// RegisterWebToolsIfEnabled registers web_search and web_fetch when web_tools.enabled.
func RegisterWebToolsIfEnabled(cfg *config.Config, reg *tools.Registry, logger *slog.Logger) error {
	if cfg == nil || cfg.WebTools == nil || !cfg.WebTools.Enabled {
		return nil
	}
	timeout, err := time.ParseDuration(strings.TrimSpace(cfg.WebTools.HTTPTimeout))
	if err != nil || timeout <= 0 {
		return fmt.Errorf("web tools: invalid http_timeout %q (validated at config.Load; unreachable unless bypassed)", cfg.WebTools.HTTPTimeout)
	}
	webHTTP := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	reg.Register(tools.NewWebSearchTool(cfg.WebTools, webHTTP, nil))
	reg.Register(tools.NewWebFetchTool(&cfg.WebTools.Fetch, webHTTP))
	logger.Info("web tools enabled", "search_provider", cfg.WebTools.Search.Provider, "http_timeout", timeout)
	return nil
}

// RegisterMemoryToolsIfEnabled registers read_memory, write_memory, and search_vector_memory.
func RegisterMemoryToolsIfEnabled(cfg *config.Config, reg *tools.Registry, memoryStore *memory.Store, memVec *core.MemoryVectors, embedder embedding.Embedder) error {
	if reg == nil {
		return fmt.Errorf("memory tools: registry is nil")
	}
	if memoryStore == nil {
		return fmt.Errorf("memory tools: memory store is required")
	}
	if cfg.ReadMemory == nil || cfg.WriteMemory == nil {
		return fmt.Errorf("memory tools: read_memory and write_memory are required in config")
	}
	rm := cfg.ReadMemory
	wm := cfg.WriteMemory
	reg.Register(tools.NewReadMemoryTool(memoryStore, rm.MaxSpanDays, rm.MaxOutputBytes))

	if !writeMemoryRuntimeReady(memVec, embedder) {
		return fmt.Errorf("memory tools: write_memory requires notes vector and embedding provider")
	}
	reg.Register(tools.NewWriteMemoryTool(memoryStore, memVec.Notes, embedder, wm.MaxAppendBytes, wm.MaxFileBytes))
	searchMemoryCfg := cfg.VectorSearchToolSettings("search_vector_memory")
	reg.Register(tools.NewSearchVectorMemoryTool(memVec.Notes, memVec.Summaries, memVec.Turns, embedder, searchMemoryCfg.DefaultTopK, searchMemoryCfg.MaxTopK, searchMemoryCfg.MaxOutputBytes, searchMemoryCfg.SnippetRunes))
	return nil
}

func writeMemoryRuntimeReady(memVec *core.MemoryVectors, embedder embedding.Embedder) bool {
	return memVec != nil && memVec.Notes != nil && embedder != nil
}

// RegisterKnowledgeToolsIfEnabled registers search_vector_tool and search_vector_skill when enabled.
func RegisterKnowledgeToolsIfEnabled(cfg *config.Config, reg *tools.Registry, toolIdx *toolindex.Index, skillIdx *skillindex.Index, embedder embedding.Embedder) {
	if cfg == nil || reg == nil || embedder == nil {
		return
	}
	toolCfg := cfg.VectorSearchToolSettings("search_vector_tool")
	if toolCfg.Enabled && toolIdx != nil && toolIdx.Store() != nil {
		reg.Register(tools.NewSearchVectorToolKnowledgeTool(toolIdx.Store(), embedder, toolCfg.DefaultTopK, toolCfg.MaxTopK, toolCfg.MaxOutputBytes, toolCfg.SnippetRunes))
	}
	skillCfg := cfg.VectorSearchToolSettings("search_vector_skill")
	if skillCfg.Enabled && skillIdx != nil && skillIdx.Store() != nil {
		reg.Register(tools.NewSearchVectorSkillKnowledgeTool(skillIdx.Store(), embedder, skillCfg.DefaultTopK, skillCfg.MaxTopK, skillCfg.MaxOutputBytes, skillCfg.SnippetRunes))
	}
}
