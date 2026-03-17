package toolcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-04.001, AC-04.002: catalog defines tools with id, template, node_id, argument rules; parse and validate at load.
func TestLoad_ValidCatalog_ParsesAndValidates(t *testing.T) {
	path := filepath.Join("testdata", "valid_catalog.yaml")
	cat, err := Load(path)
	if err != nil {
		t.Fatalf("Load(valid catalog): %v", err)
	}
	if cat == nil || cat.Tools == nil {
		t.Fatal("Load(valid catalog): got nil catalog or Tools")
	}
	if len(cat.Tools) != 1 {
		t.Errorf("Load(valid catalog): len(Tools) = %d, want 1", len(cat.Tools))
	}
	tool, ok := cat.Tools["example_cmd"]
	if !ok {
		t.Fatal("Load(valid catalog): tool example_cmd not found")
	}
	if strings.TrimSpace(tool.ID) != "example_cmd" || strings.TrimSpace(tool.Template) == "" || strings.TrimSpace(tool.NodeID) != "nas" {
		t.Errorf("Load(valid catalog): tool = id=%q template=%q node_id=%q", tool.ID, tool.Template, tool.NodeID)
	}
	if len(tool.Arguments) != 1 || tool.Arguments[0].Name != "message" {
		t.Errorf("Load(valid catalog): arguments = %v", tool.Arguments)
	}
}

// Covers AC-04.001, AC-04.002: invalid catalog (missing required fields) returns error; startup fails fast.
func TestLoad_MissingRequiredFields_ReturnsError(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{"missing id", "tools:\n  - short_description: x\n    template: t\n    node_id: n\n", "id is required"},
		{"missing short_description", "tools:\n  - id: x\n    template: t\n    node_id: n\n", "short_description is required"},
		{"missing template", "tools:\n  - id: x\n    short_description: d\n    node_id: n\n", "template is required"},
		{"missing node_id", "tools:\n  - id: x\n    short_description: d\n    template: t\n", "node_id is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "catalog.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatal("Load: expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load: error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

// Covers AC-04.001: invalid YAML or nonexistent path returns error; startup fails fast.
func TestLoad_InvalidYAMLOrMissingFile_ReturnsError(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "nonexistent.yaml"))
	if err == nil {
		t.Fatal("Load(nonexistent): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "tool catalog") {
		t.Errorf("Load(nonexistent): error = %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("not: valid: yaml: [[["), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = Load(path)
	if err == nil {
		t.Fatal("Load(invalid YAML): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Load(invalid YAML): error = %v", err)
	}
}
