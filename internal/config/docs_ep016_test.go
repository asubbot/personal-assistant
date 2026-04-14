package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-16.021: operator configuration doc lists write_memory next to read_memory for the memory-tool profile.
func TestDocs_configuration_listsWriteMemoryWithReadMemory(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "configuration.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := string(data)
	if !strings.Contains(s, "**`read_memory`**") || !strings.Contains(s, "**`write_memory`**") {
		t.Fatalf("expected read_memory and write_memory bullets in configuration.md")
	}
}
