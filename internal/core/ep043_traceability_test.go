package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-43.001
// Covers AC-43.002
func TestEP043_HandlerTestFilesSplit(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "core")
	for _, name := range []string{
		"handler_session_test.go",
		"handler_tools_test.go",
		"handler_llm_test.go",
		"handler_memory_test.go",
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing split test file %s: %v", name, err)
		}
	}
	main, err := os.ReadFile(filepath.Join(root, "handler_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(main), "\n") + 1
	if lines > 600 {
		t.Fatalf("handler_test.go lines=%d want <=600", lines)
	}
}
