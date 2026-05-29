package testutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// EnsureValidator builds bin/validate via make build when the binary is missing.
func EnsureValidator(t *testing.T, root string) {
	t.Helper()

	bin := filepath.Join(root, "bin", "validate")
	if _, err := os.Stat(bin); err == nil {
		return
	}

	modPath := filepath.Join(root, "ai-sdlc", "tools", "validate", "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		t.Fatalf("missing ai-sdlc/: clone https://github.com/asubbot/ai-sdlc at pin in ai-sdlc.version (see README)")
	}

	cmd := exec.CommandContext(context.Background(), "make", "build")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build: %v\n%s", err, out)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("bin/validate missing after make build: %v", err)
	}
}

// RunValidateEpic runs ./bin/validate for one epic from the product repository root.
func RunValidateEpic(t *testing.T, root, epicID string) {
	t.Helper()

	EnsureValidator(t, root)

	bin := filepath.Join(root, "bin", "validate")
	cmd := exec.CommandContext(context.Background(), bin, epicID)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate %s: %v\n%s", epicID, err, out)
	}
}
