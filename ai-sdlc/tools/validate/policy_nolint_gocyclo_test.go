package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindNolintGocycloViolations_emptyTree(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"cmd", "internal", "tests"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := findNolintGocycloViolations(dir)
	if err != nil {
		t.Fatalf("findNolintGocycloViolations: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no violations, got %v", got)
	}
}

func TestFindNolintGocycloViolations_detectsDirective(t *testing.T) {
	dir := t.TempDir()
	internalDir := filepath.Join(dir, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(internalDir, "sample.go")
	content := "package x\n\nfunc f() {\n\t_ = 1\n}\n//nolint:gocyclo // reason\n"
	if err := os.WriteFile(goFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findNolintGocycloViolations(dir)
	if err != nil {
		t.Fatalf("findNolintGocycloViolations: %v", err)
	}
	want := "internal/sample.go:6"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %v, want [%s]", got, want)
	}
}

func TestFindNolintGocycloViolations_ignoresOccurrenceInsideStringLiteral(t *testing.T) {
	dir := t.TempDir()
	internalDir := filepath.Join(dir, "internal")
	if err := os.MkdirAll(internalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(internalDir, "policy.go")
	content := "package x\n\nfunc f() {\n\t_ = strings.Contains(s, \"//nolint:gocyclo\")\n}\n"
	if err := os.WriteFile(goFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findNolintGocycloViolations(dir)
	if err != nil {
		t.Fatalf("findNolintGocycloViolations: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want no violations when only inside string literal, got %v", got)
	}
}

func TestFindNolintGocycloViolations_spaceAfterCommentSlashes(t *testing.T) {
	dir := t.TempDir()
	cmdDir := filepath.Join(dir, "cmd")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(cmdDir, "x.go")
	if err := os.WriteFile(goFile, []byte("// nolint:gocyclo\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := findNolintGocycloViolations(dir)
	if err != nil {
		t.Fatalf("findNolintGocycloViolations: %v", err)
	}
	if len(got) != 1 || got[0] != "cmd/x.go:1" {
		t.Fatalf("got %v", got)
	}
}
