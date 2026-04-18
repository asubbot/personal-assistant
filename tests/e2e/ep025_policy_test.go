//go:build !e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// Covers AC-25.003. Supporting AC-25.008: policy tests run under default `go test` / `make check`.
func TestMakefileDeclaresTestE2eTarget(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "test-e2e:") {
		t.Fatal("Makefile must declare test-e2e target")
	}
	if !strings.Contains(s, "go test -tags=integration,e2e") {
		t.Fatal("Makefile test-e2e must use integration,e2e build tags")
	}
}

// Covers AC-25.004
func TestMakefileDeclaresCoverageE2eTarget(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "coverage-e2e:") {
		t.Fatal("Makefile must declare coverage-e2e target")
	}
	if !strings.Contains(s, "coverage-e2e.out") {
		t.Fatal("coverage-e2e must write coverage-e2e.out")
	}
}

// Covers AC-25.005
func TestCIWorkflowMentionsE2ELayer(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(strings.ToLower(s), "e2e") {
		t.Fatal("ci.yml should document the e2e test layer")
	}
}

// Covers AC-25.006
func TestDefaultCoverageTargetDoesNotEnableE2eTag(t *testing.T) {
	root := repoRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	start := strings.Index(s, "coverage:\n")
	if start < 0 {
		t.Fatal("coverage: target not found")
	}
	end := strings.Index(s[start:], "coverage-html:")
	if end < 0 {
		t.Fatal("coverage-html: target not found after coverage")
	}
	block := s[start : start+end]
	for _, line := range strings.Split(block, "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "go test") {
			if strings.Contains(trim, "e2e") {
				t.Fatalf("default coverage go test line must not use e2e tag: %s", trim)
			}
			return
		}
	}
	t.Fatal("go test line not found under coverage: target")
}
