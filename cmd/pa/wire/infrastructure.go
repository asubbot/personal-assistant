package wire

import (
	"fmt"
	"log/slog"
	"os"
	"pa/internal/allowlist"
	"pa/internal/config"
	"pa/internal/core"
	"pa/internal/embedding"
	"pa/internal/memory"
	"pa/internal/noderunner"
	"pa/internal/skillindex"
	"pa/internal/telegram"
	"pa/internal/toolindex"
	"path/filepath"
)

// Infrastructure holds subsystem handles constructed by the composition root (EP-027, EP-042).
type Infrastructure struct {
	Adapter     core.Adapter
	MemoryStore *memory.Store
	MemVec      *core.MemoryVectors
	Embedder    embedding.Embedder
	NodeRunner  core.NodeRunner
	ToolIndex   *toolindex.Index
	SkillIndex  *skillindex.Index
}

// Close releases infrastructure acquired during construction (vector stores, indices).
func (i *Infrastructure) Close(logger *slog.Logger) {
	if i == nil {
		return
	}
	if i.MemVec != nil {
		if closeErr := i.MemVec.Close(); closeErr != nil {
			logger.Error("close memory vector stores", "error", closeErr)
		}
	}
	if i.ToolIndex != nil {
		if closeErr := i.ToolIndex.Close(); closeErr != nil {
			logger.Error("close tool index", "error", closeErr)
		}
	}
	if i.SkillIndex != nil {
		if closeErr := i.SkillIndex.Close(); closeErr != nil {
			logger.Error("close skill index", "error", closeErr)
		}
	}
}

func setupTelegramAdapter(cfg *config.Config, configPath string) (core.Adapter, error) {
	return telegram.NewAdapter(cfg, configPath)
}

func setupMemoryStoreIfConfigured(cfg *config.Config) (*memory.Store, error) {
	if cfg.Paths.MemoryDir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(cfg.Paths.MemoryDir, 0o755); err != nil {
		return nil, err
	}
	loc, err := config.PALocation(cfg)
	if err != nil {
		return nil, fmt.Errorf("memory store timezone: %w", err)
	}
	return memory.NewStore(cfg.Paths.MemoryDir, loc)
}

func setupEmbedder(cfg *config.Config) (embedding.Embedder, error) {
	return embedding.NewEmbedder(cfg.Embedding)
}

func ensureVectorDir(cfg *config.Config) error {
	vecDir := filepath.Dir(cfg.Paths.VectorIndexPath)
	return os.MkdirAll(vecDir, 0o755)
}

func setupNodeRunnerIfConfigured(cfg *config.Config, logger *slog.Logger) (core.NodeRunner, error) {
	if len(cfg.Nodes) == 0 {
		return nil, nil
	}
	al, err := allowlist.NewChecker(cfg)
	if err != nil {
		return nil, err
	}
	nr := noderunner.New(cfg, al, logger)
	nr.SetLogRedactor(core.BuildLogRedactor(cfg))
	return nr, nil
}

func setupToolIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*toolindex.Index, error) {
	return NewToolIndex(cfg, embedder, logger)
}

func setupSkillIndex(cfg *config.Config, embedder embedding.Embedder, logger *slog.Logger) (*skillindex.Index, error) {
	return newSkillIndex(cfg, embedder, logger)
}

// BuildInfrastructure constructs adapter, optional memory store, vectors, embedder, indices, and optional node runner.
func BuildInfrastructure(cfg *config.Config, configPath string, logger *slog.Logger) (Infrastructure, error) {
	var out Infrastructure
	adapter, err := setupTelegramAdapter(cfg, configPath)
	if err != nil {
		return out, err
	}
	out.Adapter = adapter

	memoryStore, err := setupMemoryStoreIfConfigured(cfg)
	if err != nil {
		return out, err
	}
	out.MemoryStore = memoryStore

	embedder, err := setupEmbedder(cfg)
	if err != nil {
		return out, err
	}
	out.Embedder = embedder

	if err := ensureVectorDir(cfg); err != nil {
		return out, err
	}
	memVec, err := openMemoryVectorBundle(cfg)
	if err != nil {
		return out, err
	}
	out.MemVec = memVec

	toolIndex, err := setupToolIndex(cfg, embedder, logger)
	if err != nil {
		out.Close(logger)
		return out, err
	}
	out.ToolIndex = toolIndex

	skillIndex, err := setupSkillIndex(cfg, embedder, logger)
	if err != nil {
		out.Close(logger)
		return out, err
	}
	out.SkillIndex = skillIndex

	nodeRunner, err := setupNodeRunnerIfConfigured(cfg, logger)
	if err != nil {
		out.Close(logger)
		return out, err
	}
	out.NodeRunner = nodeRunner

	return out, nil
}
