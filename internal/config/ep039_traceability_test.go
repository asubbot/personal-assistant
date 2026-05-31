package config

import (
	"path/filepath"
	"testing"
)

// Covers AC-39.011, AC-39.013
func TestEP039_MigratedValidTestdataLoads(t *testing.T) {
	for _, name := range []string{
		"valid_no_users.json",
		"valid_with_tool_catalog.json",
		"conversation_session_ok.json",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Load(filepath.Join("testdata", name))
			if err != nil {
				t.Fatalf("Load(%s): %v", name, err)
			}
		})
	}
}
