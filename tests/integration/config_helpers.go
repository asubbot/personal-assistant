//go:build integration

package integration_test

import (
	"pa/internal/config"
	"strings"
)

// ensureCoreRunConfigRequiredSections fills required config sections when tests bypass config.Load.
// Production configs must include these keys explicitly in config.json; the loader does not inject them.
// This helper only wires the same explicit values tests would otherwise repeat on every &config.Config{}.
func ensureCoreRunConfigRequiredSections(cfg *config.Config) {
	if cfg == nil {
		return
	}
	if cfg.Tools == nil {
		cfg.Tools = &config.ToolsConfig{}
	}
	if cfg.LogRedaction == nil {
		cfg.LogRedaction = &config.LogRedaction{}
	}
	if strings.TrimSpace(cfg.PATimezone) == "" {
		cfg.PATimezone = "UTC"
	}
	if cfg.ToolPreSelection == nil {
		cfg.ToolPreSelection = &config.ToolPreSelection{
			ToolSearchTopK: 10, ToolMinCount: 1, ToolFallbackCap: 50,
		}
	}
	if cfg.ConversationContext == nil {
		cfg.ConversationContext = &config.ConversationContextConfig{
			MaxDynamicSystemRunes: 4000,
			MemoryVector: config.MemoryVectorConfig{
				NotesTopK: 10, SummariesTopK: 10, TurnsTopK: 10,
			},
		}
	}
	if cfg.ReadMemory == nil {
		cfg.ReadMemory = &config.ReadMemoryConfig{
			MaxSpanDays:    31,
			MaxOutputBytes: 256 * 1024,
		}
	}
	if cfg.WriteMemory == nil {
		cfg.WriteMemory = &config.WriteMemoryConfig{
			MaxAppendBytes: 65536,
			MaxFileBytes:   5 * 1024 * 1024,
		}
	}
}
