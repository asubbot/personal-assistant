package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-43.003
func TestEP043_loadConfigFixtureUsedInTests(t *testing.T) {
	root := filepath.Join("..", "..", "internal", "config")
	var count int
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		count += strings.Count(string(raw), "loadConfigFixture(")
		return nil
	})
	if count < 10 {
		t.Fatalf("loadConfigFixture call count = %d, want >= 10", count)
	}
}
