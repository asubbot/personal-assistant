package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Covers AC-23.009: root README documents atomic replace, Sync, and post-write Load for operators.
func TestReadme_toolCatalogDurabilitySection(t *testing.T) {
	t.Parallel()
	_, self, _, _ := runtime.Caller(0)
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(self), "..", ".."))
	b, err := os.ReadFile(filepath.Join(repoRoot, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, frag := range []string{
		"Tool catalog durability",
		"atomic replace",
		"Explicit `Sync`",
		"toolcatalog.Load",
	} {
		if !strings.Contains(s, frag) {
			t.Fatalf("README.md missing operator doc fragment %q", frag)
		}
	}
}
