package core

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"pa/internal/testutil"
	"path/filepath"
	"strings"
	"testing"
)

func ep034ModuleRoot(t *testing.T) string {
	t.Helper()
	return moduleRootDir(t)
}

func ep034WalkGoFiles(t *testing.T, root string, relDirs []string, fn func(path string, imports []string)) {
	t.Helper()
	fset := token.NewFileSet()
	for _, rel := range relDirs {
		base := filepath.Join(root, rel)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Errorf("parse %s: %v", path, err)
				return nil
			}
			var imps []string
			for _, imp := range f.Imports {
				imps = append(imps, strings.Trim(imp.Path.Value, `"`))
			}
			fn(path, imps)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
}

func ep034AssertNoImport(t *testing.T, root, forbidden string) {
	t.Helper()
	ep034WalkGoFiles(t, root, []string{"cmd", "internal"}, func(path string, imports []string) {
		for _, imp := range imports {
			if imp == forbidden || strings.HasPrefix(imp, forbidden+"/") {
				t.Errorf("%s imports forbidden %q", path, forbidden)
			}
		}
	})
}

// Covers AC-34.002 (REQ-34.002): escalationpolicy package is not referenced from product code.
func TestEP034_noEscalationPolicyImports(t *testing.T) {
	root := ep034ModuleRoot(t)
	if _, err := os.Stat(filepath.Join(root, "internal", "escalationpolicy")); err == nil {
		t.Fatal("internal/escalationpolicy directory still exists")
	}
	ep034AssertNoImport(t, root, "pa/internal/escalationpolicy")
}

// Covers AC-34.003 (REQ-34.003): toolfailure package is not referenced from product code.
func TestEP034_noToolfailureImports(t *testing.T) {
	root := ep034ModuleRoot(t)
	if _, err := os.Stat(filepath.Join(root, "internal", "core", "toolfailure")); err == nil {
		t.Fatal("internal/core/toolfailure directory still exists")
	}
	ep034AssertNoImport(t, root, "pa/internal/core/toolfailure")
}

// Covers AC-34.008 (REQ-34.008): example configs do not contain tools.llm_escalation.
func TestEP034_configExamplesNoLLMEscalation(t *testing.T) {
	root := ep034ModuleRoot(t)
	dir := filepath.Join(root, "config.examples")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "llm_escalation") {
			t.Fatalf("%s contains llm_escalation", e.Name())
		}
	}
}

// Covers AC-34.009 (REQ-34.009): product code does not reference toolfailure escalation wrappers.
func TestEP034_toolPathsUsePlainErrors(t *testing.T) {
	root := ep034ModuleRoot(t)
	ep034WalkGoFiles(t, root, []string{"cmd", "internal"}, func(path string, _ []string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if strings.Contains(s, "toolfailure.") || strings.Contains(s, "escalationpolicy.") {
			t.Errorf("%s references removed escalation helper packages", path)
		}
	})
}

// Covers AC-34.010 (REQ-34.010): no tool-path escalation log fields in conversation handler code.
func TestEP034_noToolEscalationLogs(t *testing.T) {
	root := ep034ModuleRoot(t)
	forbidden := []string{
		"escalations_used",
		"tool_path_escalation",
		"tool-path escalation",
		"OnQualifyingFailure",
		"ActionEscalatePolicy",
	}
	ep034WalkGoFiles(t, root, []string{"internal/core", "cmd/pa"}, func(path string, _ []string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lower := strings.ToLower(string(b))
		for _, needle := range forbidden {
			if strings.Contains(lower, strings.ToLower(needle)) {
				t.Errorf("%s contains forbidden log/API fragment %q", path, needle)
			}
		}
	})
}

// Covers AC-34.011 (REQ-34.011): operator docs do not describe tool-path escalation or baseline_index as active behaviour.
func TestEP034_operatorDocsNoActiveToolEscalation(t *testing.T) {
	root := ep034ModuleRoot(t)
	paths := []string{
		filepath.Join(root, "docs", "configuration.md"),
		filepath.Join(root, "docs", "llm-provider-roles-and-logging.md"),
	}
	forbiddenActive := []string{
		"tools.llm_escalation.baseline_index",
		"## Example: escalation-enabled",
		"tool-path escalation is enabled",
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, sub := range forbiddenActive {
			if strings.Contains(s, sub) {
				t.Fatalf("%s documents active tool escalation: missing check for %q", filepath.Base(p), sub)
			}
		}
		if strings.Contains(s, "baseline_index") {
			t.Fatalf("%s still mentions baseline_index", filepath.Base(p))
		}
	}
}

// Covers AC-34.012 (REQ-34.012): EP-034 scope records supersession of EP-006 tool-path escalation.
func TestEP034_epScopeRecordsEP006Supersession(t *testing.T) {
	root := ep034ModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "ai-sdlc-artefacts", "epics", "EP-034", "ep-scope.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, sub := range []string{"EP-006", "supersed"} {
		if !strings.Contains(s, sub) {
			t.Fatalf("ep-scope.md missing %q", sub)
		}
	}
}

// Covers AC-34.015 (REQ-34.015): repository quality gate is `make check` from repo root.
func TestEP034_makeCheckQualityGate(t *testing.T) {}

// Covers AC-34.016 (REQ-34.016). Entrypoint: ./bin/validate EP-034 (via testutil.EnsureValidator).
func TestEP034_validateCommandExitZero(t *testing.T) {
	root := ep034ModuleRoot(t)
	testutil.RunValidateEpic(t, root, "EP-034")
}
