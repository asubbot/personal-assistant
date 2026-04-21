package runtimeskills

import (
	"os"
	"pa/internal/toolcatalog"
	"path/filepath"
	"runtime"
	"testing"
)

func writeSkill(t *testing.T, root, skillName, content string) {
	t.Helper()
	dir := filepath.Join(root, skillName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-13.011: traceability for TestLoadDir_valid.
func TestLoadDir_valid(t *testing.T) {
	// Covers AC-13.011 happy path
	root := t.TempDir()
	writeSkill(t, root, "demo", "---\nname: Demo\ndescription: A demo\ntools:\n  - t1\n---\n\nBody here.\n")
	pkgs, err := LoadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].ID != "demo" || pkgs[0].Body != "Body here." {
		t.Fatalf("%+v", pkgs)
	}
}

func TestLoadDir_missingFrontmatter(t *testing.T) {
	// Covers AC-13.011
	root := t.TempDir()
	writeSkill(t, root, "bad", "no frontmatter\n")
	_, err := LoadDir(root)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadDir_forbiddenMarker(t *testing.T) {
	// Covers AC-13.001 (loader)
	root := t.TempDir()
	bad := "---\nname: X\ndescription: Y\n---\n\n<<<PA_BEGIN_CONTEXT>>>\n"
	writeSkill(t, root, "bad", bad)
	_, err := LoadDir(root)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateToolRefs_unknown(t *testing.T) {
	// Covers AC-13.002
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{"t1": {ID: "t1"}}}
	pkgs := []*Package{{ID: "s", Tools: []string{"nope"}}}
	err := ValidateToolRefs(pkgs, cat, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateToolRefs_native(t *testing.T) {
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	pkgs := []*Package{{ID: "s", Tools: []string{"run_on_node"}}}
	if err := ValidateToolRefs(pkgs, cat, []string{"run_on_node"}); err != nil {
		t.Fatal(err)
	}
}

// Covers AC-02.012: memory retrieval skill may list read_memory in frontmatter; EP-013 validation accepts it when id is on native allowlist.
func TestValidateToolRefs_readMemoryNative(t *testing.T) {
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	pkgs := []*Package{{ID: "memory-retrieval", Tools: []string{"read_memory"}}}
	allow := []string{"run_on_node", "create_tool", "read_memory"}
	if err := ValidateToolRefs(pkgs, cat, allow); err != nil {
		t.Fatal(err)
	}
}

// Covers EP-021 AC-21.007: example scheduled-jobs skill lists create_scheduled_job and validates against native allowlist.
func TestEP021_exampleScheduledJobsSkill_validateRefs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(repoRoot, "config.examples", "skills", "scheduled-jobs", "SKILL.md"))
	if err != nil {
		t.Fatalf("read example skill: %v", err)
	}
	root := t.TempDir()
	writeSkill(t, root, "scheduled-jobs", string(raw))
	pkgs, err := LoadDir(root)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].ID != "scheduled-jobs" {
		t.Fatalf("pkgs = %+v", pkgs)
	}
	cat := &toolcatalog.Catalog{Tools: map[string]*toolcatalog.Tool{}}
	allow := []string{"run_on_node", "create_tool", "read_memory", "write_memory", "create_scheduled_job"}
	if err := ValidateToolRefs(pkgs, cat, allow); err != nil {
		t.Fatal(err)
	}
}
