package config

import (
	"os"
	"pa/internal/toolcatalog"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-37.003
func TestValidateToolsSelectionAlwaysIncludeFloor_maxToolsBelowAlwaysIncludeFails(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			AlwaysInclude: []string{"a", "b"},
			Selection: &ToolsSelection{
				Enabled:               true,
				MaxToolsForLLMRequest: 1,
				ToolSearchTopK:        10,
				ToolMinCount:          1,
				ToolFallbackCap:       50,
			},
		},
		ToolCatalog: &toolcatalog.Catalog{
			Tools: map[string]*toolcatalog.Tool{
				"a": {ID: "a"},
				"b": {ID: "b"},
			},
		},
	}
	if err := validateToolsSelectionAlwaysIncludeFloor(c); err == nil {
		t.Fatal("expected error when max_tools_for_llm_request < distinct valid always_include")
	}
}

// Covers AC-37.003
func TestValidateToolsSelectionAlwaysIncludeFloor_enabledOkWithoutAlwaysInclude(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			Selection: &ToolsSelection{
				Enabled:               true,
				MaxToolsForLLMRequest: 5,
				ToolSearchTopK:        10,
				ToolMinCount:          1,
				ToolFallbackCap:       50,
			},
		},
	}
	if err := validateToolsSelectionAlwaysIncludeFloor(c); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

// Covers AC-37.004
func TestValidateToolsSelectionAlwaysIncludeFloor_okWhenDisabled(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			Selection: &ToolsSelection{
				Enabled:               false,
				MaxToolsForLLMRequest: 0,
				ToolSearchTopK:        10,
				ToolMinCount:          1,
				ToolFallbackCap:       50,
			},
		},
	}
	if err := validateToolsSelectionAlwaysIncludeFloor(c); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

// Covers AC-37.002, AC-37.003
func TestValidateToolsSelectionBounds_enabledRequiresPositiveMax(t *testing.T) {
	c := &Config{
		Tools: &ToolsConfig{
			Selection: &ToolsSelection{
				Enabled:         true,
				ToolSearchTopK:  10,
				ToolMinCount:    1,
				ToolFallbackCap: 50,
			},
		},
	}
	if err := validateToolsSelectionBounds(c); err == nil {
		t.Fatal("expected error when enabled and max_tools_for_llm_request is zero")
	}
}

// Covers AC-37.001
func TestLoad_ToolsSelectionRequired(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_selection_missing.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tools.selection") {
		t.Fatalf("Load: error = %v, want tools.selection mention", err)
	}
}

// Covers AC-37.002
func TestLoad_ToolsSelectionBounds_zeroTopK(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_selection_zero.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tool_search_top_k must be >= 1") {
		t.Fatalf("Load: error = %v, want tool_search_top_k bounds", err)
	}
}

// Covers AC-37.003
func TestLoad_ToolsSelection_enabledMaxZeroRejected(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_selection_enabled_max_zero.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "max_tools_for_llm_request") {
		t.Fatalf("Load: error = %v, want max_tools_for_llm_request mention", err)
	}
}

// Covers AC-37.004
func TestLoad_ToolsSelection_disabledMaxZeroLoads(t *testing.T) {
	// valid_no_users has enabled:false and max 0
	cfg, err := Load(filepath.Join("testdata", "valid_no_users.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Tools == nil || cfg.Tools.Selection == nil || cfg.Tools.Selection.Enabled {
		t.Fatal("expected disabled selection")
	}
}

// Covers AC-37.005
func TestLoad_ToolPreSelectionRejected(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tool_pre_selection_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tool_pre_selection") {
		t.Fatalf("Load: error = %v, want tool_pre_selection rejection", err)
	}
}

// Covers AC-37.006
func TestLoad_ToolsDynamicSelectionRejected(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_dynamic_selection_rejected.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	// Assert the production rejection message (not a bare substring) so a
	// missing/renamed fixture fails loudly instead of passing on the OS
	// "no such file" error path, which also contains "dynamic_selection".
	if !strings.Contains(err.Error(), "tools.dynamic_selection is not supported") {
		t.Fatalf("Load: error = %v, want tools.dynamic_selection rejection", err)
	}
}

// Covers AC-37.008
func TestLoad_ToolsUnknownNestedKey(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "tools_unknown_nested_key.json"))
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown tools key") {
		t.Fatalf("Load: error = %v, want unknown tools key", err)
	}
}

func fixtureShouldLoadOK(name string) bool {
	if name == "valid_with_users.json" {
		return false // invalid users file; covered by TestLoad_invalidUsers
	}
	if strings.HasPrefix(name, "valid_") {
		return true
	}
	switch name {
	case "conversation_memory_vector_all_zero.json", "conversation_session_ok.json",
		"llm_default_temperature_zero.json", "llm_default_temperature_two.json":
		return true
	default:
		return false
	}
}

// Covers AC-37.013
func TestLoad_AllFixturesLoad(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if !fixtureShouldLoadOK(e.Name()) {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			_, err := Load(filepath.Join("testdata", e.Name()))
			if err != nil {
				t.Fatalf("Load(%s): %v", e.Name(), err)
			}
		})
	}
}

// Covers AC-37.007
func TestConfigRootJSONKeys_ExcludesToolPreSelection(t *testing.T) {
	keys := ConfigRootJSONKeys()
	for _, k := range keys {
		if k == "tool_pre_selection" {
			t.Fatal("configRootJSONKeys must not include tool_pre_selection (EP-037)")
		}
	}
}
