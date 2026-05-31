package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-39.008
func TestConfigRootJSONKeys_IncludesSQLiteStoreDefaults(t *testing.T) {
	keys := ConfigRootJSONKeys()
	found := false
	for i, k := range keys {
		if k == "sqlite_store_defaults" {
			found = true
			if i > 0 && keys[i-1] != "runtime_skills" {
				t.Fatalf("sqlite_store_defaults should follow runtime_skills, got %q before it", keys[i-1])
			}
			if i+1 < len(keys) && keys[i+1] != "telegram" {
				t.Fatalf("sqlite_store_defaults should precede telegram, got %q after it", keys[i+1])
			}
		}
	}
	if !found {
		t.Fatal("configRootJSONKeys must include sqlite_store_defaults (EP-039)")
	}
}

// Covers AC-39.002, AC-39.010
func TestLoad_LegacyVectorSearchToolsShape_Rejected(t *testing.T) {
	path := filepath.Join("testdata", "vector_search_tools_legacy_rejected.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tools.vector_search_tools") || !strings.Contains(err.Error(), "legacy per-tool shape") {
		t.Fatalf("Load: error = %v, want legacy vector_search_tools rejection", err)
	}
}

// Covers AC-39.010
func TestLoad_LegacySQLiteReliabilityShape_Rejected(t *testing.T) {
	path := filepath.Join("testdata", "sqlite_reliability_legacy_rejected.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sqlite_store_defaults") || !strings.Contains(err.Error(), "legacy full duplicate PRAGMA") {
		t.Fatalf("Load: error = %v, want legacy sqlite reliability rejection", err)
	}
}
