package scheduler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTasks_emptyPath(t *testing.T) {
	tasks, err := LoadTasks("")
	if err != nil {
		t.Fatalf("LoadTasks(\"\") = %v", err)
	}
	if tasks != nil {
		t.Errorf("LoadTasks(\"\") = %v, want nil", tasks)
	}
}

func TestLoadTasks_missingFile(t *testing.T) {
	tasks, err := LoadTasks(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("LoadTasks(missing) = %v", err)
	}
	if tasks != nil {
		t.Errorf("LoadTasks(missing) = %v, want nil", tasks)
	}
}

func TestLoadTasks_validJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	content := `[
  { "schedule": "0 9 * * *", "action": "notify", "params": {} },
  { "schedule": "@every 1h", "action": "run_on_node", "params": { "node_id": "nas", "command": "uptime" } }
]`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	tasks, err := LoadTasks(path)
	if err != nil {
		t.Fatalf("LoadTasks = %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	if tasks[0].Schedule != "0 9 * * *" || tasks[0].Action != "notify" {
		t.Errorf("task[0] = %+v", tasks[0])
	}
	if tasks[1].Schedule != "@every 1h" || tasks[1].Action != "run_on_node" {
		t.Errorf("task[1] = %+v", tasks[1])
	}
}

func TestLoadTasks_invalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTasks(path)
	if err == nil {
		t.Error("LoadTasks(invalid JSON) want error")
	}
}
