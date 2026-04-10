package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_tools_badAlwaysInclude(t *testing.T) {
	// Covers AC-13.003
	path := filepath.Join("testdata", "tools_bad_always_include.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "tools.always_include") {
		t.Fatalf("error = %v", err)
	}
}
