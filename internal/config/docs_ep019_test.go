package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Covers AC-19.020: docs and examples reference only jobs_db_path (no legacy scheduled_tasks_path).
func TestDocsAndExamples_EP019_NoLegacyScheduledTasksPath(t *testing.T) {
	root := filepath.Join("..", "..")
	paths := []string{
		filepath.Join(root, "docs", "configuration.md"),
		filepath.Join(root, "docs", "operations.md"),
		filepath.Join(root, "docs", "installation.md"),
		filepath.Join(root, "config.examples", "config.example.json"),
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		s := string(data)
		if strings.Contains(s, "scheduled_tasks_path") {
			t.Fatalf("%s still contains legacy scheduled_tasks_path", p)
		}
		if !strings.Contains(s, "jobs_db_path") {
			t.Fatalf("%s does not mention jobs_db_path", p)
		}
	}
}
