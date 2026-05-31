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

// VectorSearchToolOverride is a per-tool override; omitted fields inherit from defaults (EP-039).
type VectorSearchToolOverride struct {
	Enabled        *bool `json:"enabled,omitempty"`
	DefaultTopK    *int  `json:"default_top_k,omitempty"`
	MaxTopK        *int  `json:"max_top_k,omitempty"`
	MaxOutputBytes *int  `json:"max_output_bytes,omitempty"`
	SnippetRunes   *int  `json:"snippet_runes,omitempty"`
}

// VectorSearchToolsConfig holds defaults plus per-tool overrides for vector-search native tools (EP-039).
type VectorSearchToolsConfig struct {
	Defaults           VectorSearchToolConfig   `json:"defaults"`
	SearchVectorMemory VectorSearchToolOverride `json:"search_vector_memory"`
	SearchVectorTool   VectorSearchToolOverride `json:"search_vector_tool"`
	SearchVectorSkill  VectorSearchToolOverride `json:"search_vector_skill"`
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

func mergeVectorSearchTool(defaults VectorSearchToolConfig, override VectorSearchToolOverride) VectorSearchToolConfig {
	merged := defaults
	if override.Enabled != nil {
		merged.Enabled = *override.Enabled
	}
	if override.DefaultTopK != nil {
		merged.DefaultTopK = *override.DefaultTopK
	}
	if override.MaxTopK != nil {
		merged.MaxTopK = *override.MaxTopK
	}
	if override.MaxOutputBytes != nil {
		merged.MaxOutputBytes = *override.MaxOutputBytes
	}
	if override.SnippetRunes != nil {
		merged.SnippetRunes = *override.SnippetRunes
	}
	return merged
}

// VectorSearchToolSettings resolves one tool settings from tools.vector_search_tools with defaults when block is absent.
func (c *Config) VectorSearchToolSettings(toolID string) VectorSearchToolConfig {
	d := defaultVectorSearchToolConfig()
	if c == nil || c.Tools == nil || c.Tools.VectorSearchTools == nil {
		return d
	}
	vst := c.Tools.VectorSearchTools
	switch toolID {
	case "search_vector_memory":
		return mergeVectorSearchTool(vst.Defaults, vst.SearchVectorMemory)
	case "search_vector_tool":
		return mergeVectorSearchTool(vst.Defaults, vst.SearchVectorTool)
	case "search_vector_skill":
		return mergeVectorSearchTool(vst.Defaults, vst.SearchVectorSkill)
	default:
		return d
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

func validateVectorSearchToolOverride(field string, defaults VectorSearchToolConfig, override VectorSearchToolOverride) error {
	if err := validateVectorSearchToolOverrideFields(field, override); err != nil {
		return err
	}
	merged := mergeVectorSearchTool(defaults, override)
	if override.DefaultTopK != nil || override.MaxTopK != nil {
		if merged.DefaultTopK > merged.MaxTopK {
			return fmt.Errorf("config: %s.default_top_k must be <= %s.max_top_k", field, field)
		}
	}
	return nil
}

func validateVectorSearchToolOverrideFields(field string, override VectorSearchToolOverride) error {
	if override.DefaultTopK != nil && *override.DefaultTopK < 1 {
		return fmt.Errorf("config: %s.default_top_k must be >= 1", field)
	}
	if override.MaxTopK != nil {
		if *override.MaxTopK < 1 {
			return fmt.Errorf("config: %s.max_top_k must be >= 1", field)
		}
		if *override.MaxTopK > 500 {
			return fmt.Errorf("config: %s.max_top_k must be <= 500", field)
		}
	}
	if override.MaxOutputBytes != nil {
		if *override.MaxOutputBytes < 256 || *override.MaxOutputBytes > 1024*1024 {
			return fmt.Errorf("config: %s.max_output_bytes must be in 256..1048576", field)
		}
	}
	if override.SnippetRunes != nil {
		if *override.SnippetRunes < 32 || *override.SnippetRunes > 2000 {
			return fmt.Errorf("config: %s.snippet_runes must be in 32..2000", field)
		}
	}
	return nil
}

func validateVectorSearchTools(vst *VectorSearchToolsConfig) error {
	if vst == nil {
		return nil
	}
	if err := validateVectorSearchToolConfig("tools.vector_search_tools.defaults", vst.Defaults); err != nil {
		return err
	}
	tools := []struct {
		name string
		o    VectorSearchToolOverride
	}{
		{"search_vector_memory", vst.SearchVectorMemory},
		{"search_vector_tool", vst.SearchVectorTool},
		{"search_vector_skill", vst.SearchVectorSkill},
	}
	for _, t := range tools {
		field := "tools.vector_search_tools." + t.name
		if err := validateVectorSearchToolOverride(field, vst.Defaults, t.o); err != nil {
			return err
		}
		merged := mergeVectorSearchTool(vst.Defaults, t.o)
		if err := validateVectorSearchToolConfig(field, merged); err != nil {
			return err
		}
	}
	return nil
}
