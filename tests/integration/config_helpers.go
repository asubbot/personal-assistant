//go:build integration

package integration_test

import (
	"pa/internal/config"
	"strings"
)

// ensureCoreRunConfigRequiredSections fills required config sections when tests bypass config.Load.
// Production configs must include these keys explicitly; this helper only avoids duplicating literals in tests.
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
			InjectedContextMaxChars: 4000,
			VectorSearchTopK:        10,
		}
	}
}
