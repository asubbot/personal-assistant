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
				Enabled:               true,
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

// Covers AC-18.019: dynamic_selection.enabled does not require text_based_enabled at load;
// full_lite applies the cap at runtime only when text_based_enabled is true.
func TestValidateToolDynamicSelection_enabledOkWithoutTextBased(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			TextBasedEnabled: false,
			DynamicSelection: &ToolDynamicSelection{
				Enabled:               true,
				MaxToolsForLLMRequest: 5,
			},
		},
	}
	if err := validateToolDynamicSelection(c); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

// Covers AC-18.019
func TestValidateToolDynamicSelection_okWhenDisabledFlags(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			DynamicSelection: &ToolDynamicSelection{
				Enabled:               false,
				MaxToolsForLLMRequest: 0,
			},
		},
	}
	if err := validateToolDynamicSelection(c); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

// Covers AC-18.019: when enabled, max_tools_for_llm_request must be explicit (>= 1); no load-time default.
func TestValidateToolDynamicSelection_enabledRequiresPositiveMax(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			TextBasedEnabled: true,
			DynamicSelection: &ToolDynamicSelection{
				Enabled: true,
			},
		},
	}
	if err := validateToolDynamicSelection(c); err == nil {
		t.Fatal("expected error when enabled and max_tools_for_llm_request is zero")
	}
}
