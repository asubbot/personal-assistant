package config

import "fmt"

const (
	defaultVectorSearchTopK       = 5
	defaultVectorSearchMaxTopK    = 10
	defaultVectorSearchOutputByte = 4096
	defaultVectorSearchSnippet    = 200
)

// VectorSearchToolConfig configures one native vector-search tool runtime behavior.
type VectorSearchToolConfig struct {
	Enabled        bool `json:"enabled"`
	DefaultTopK    int  `json:"default_top_k"`
	MaxTopK        int  `json:"max_top_k"`
	MaxOutputBytes int  `json:"max_output_bytes"`
	SnippetRunes   int  `json:"snippet_runes"`
}

// VectorSearchToolsConfig holds one unified block for all vector-search native tools (EP-032).
type VectorSearchToolsConfig struct {
	SearchVectorMemory VectorSearchToolConfig `json:"search_vector_memory"`
	SearchVectorTool   VectorSearchToolConfig `json:"search_vector_tool"`
	SearchVectorSkill  VectorSearchToolConfig `json:"search_vector_skill"`
}

func defaultVectorSearchToolConfig() VectorSearchToolConfig {
	return VectorSearchToolConfig{
		Enabled:        true,
		DefaultTopK:    defaultVectorSearchTopK,
		MaxTopK:        defaultVectorSearchMaxTopK,
		MaxOutputBytes: defaultVectorSearchOutputByte,
		SnippetRunes:   defaultVectorSearchSnippet,
	}
}

func defaultVectorSearchToolsConfig() VectorSearchToolsConfig {
	d := defaultVectorSearchToolConfig()
	return VectorSearchToolsConfig{
		SearchVectorMemory: d,
		SearchVectorTool:   d,
		SearchVectorSkill:  d,
	}
}

// VectorSearchToolSettings resolves one tool settings from tools.vector_search_tools with defaults when block is absent.
func (c *Config) VectorSearchToolSettings(toolID string) VectorSearchToolConfig {
	d := defaultVectorSearchToolsConfig()
	if c == nil || c.Tools == nil || c.Tools.VectorSearchTools == nil {
		switch toolID {
		case "search_vector_memory":
			return d.SearchVectorMemory
		case "search_vector_tool":
			return d.SearchVectorTool
		case "search_vector_skill":
			return d.SearchVectorSkill
		default:
			return defaultVectorSearchToolConfig()
		}
	}
	switch toolID {
	case "search_vector_memory":
		return c.Tools.VectorSearchTools.SearchVectorMemory
	case "search_vector_tool":
		return c.Tools.VectorSearchTools.SearchVectorTool
	case "search_vector_skill":
		return c.Tools.VectorSearchTools.SearchVectorSkill
	default:
		return defaultVectorSearchToolConfig()
	}
}

func validateVectorSearchToolConfig(field string, cfg VectorSearchToolConfig) error {
	if cfg.DefaultTopK < 1 {
		return fmt.Errorf("config: %s.default_top_k must be >= 1", field)
	}
	if cfg.MaxTopK < 1 {
		return fmt.Errorf("config: %s.max_top_k must be >= 1", field)
	}
	if cfg.DefaultTopK > cfg.MaxTopK {
		return fmt.Errorf("config: %s.default_top_k must be <= %s.max_top_k", field, field)
	}
	if cfg.MaxTopK > 500 {
		return fmt.Errorf("config: %s.max_top_k must be <= 500", field)
	}
	if cfg.MaxOutputBytes < 256 || cfg.MaxOutputBytes > 1024*1024 {
		return fmt.Errorf("config: %s.max_output_bytes must be in 256..1048576", field)
	}
	if cfg.SnippetRunes < 32 || cfg.SnippetRunes > 2000 {
		return fmt.Errorf("config: %s.snippet_runes must be in 32..2000", field)
	}
	return nil
}
