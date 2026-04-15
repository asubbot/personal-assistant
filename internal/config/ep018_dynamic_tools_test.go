package config

import (
	"pa/internal/toolcatalog"
	"testing"
)

// Covers AC-18.019
func TestValidateToolDynamicSelection_maxToolsBelowAlwaysIncludeFails(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			TextBasedEnabled: true,
			AlwaysInclude:    []string{"a", "b"},
			DynamicSelection: &ToolDynamicSelection{
				EnabledForFull:        true,
				MaxToolsForLLMRequest: 1,
			},
		},
		ToolCatalog: &toolcatalog.Catalog{
			Tools: map[string]*toolcatalog.Tool{
				"a": {ID: "a"},
				"b": {ID: "b"},
			},
		},
	}
	if err := validateToolDynamicSelection(c); err == nil {
		t.Fatal("expected error when max_tools_for_llm_request < distinct valid always_include")
	}
}

// Covers AC-18.019
func TestValidateToolDynamicSelection_fullLiteRequiresTextBased(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			TextBasedEnabled: false,
			DynamicSelection: &ToolDynamicSelection{
				EnabledForFullLite:    true,
				EnabledForFull:        false,
				MaxToolsForLLMRequest: 5,
			},
		},
	}
	if err := validateToolDynamicSelection(c); err == nil {
		t.Fatal("expected error when enabled_for_full_lite without text_based_enabled")
	}
}

// Covers AC-18.019
func TestValidateToolDynamicSelection_okWhenDisabledFlags(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			DynamicSelection: &ToolDynamicSelection{
				EnabledForFullLite:    false,
				EnabledForFull:        false,
				MaxToolsForLLMRequest: 0,
			},
		},
	}
	if err := validateToolDynamicSelection(c); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
