package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_runtimeSkills_badAlwaysInclude(t *testing.T) {
	// Covers AC-13.003
	path := filepath.Join("testdata", "runtime_skills_bad_always_include.json")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "always_include") {
		t.Fatalf("error = %v", err)
	}
}
