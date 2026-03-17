package allowlist

import (
	"pa/internal/config"
	"path/filepath"
	"testing"
)

// Covers AC-01.007 (US-04): Allow returns allowed for allowlisted commands.
func TestChecker_Allow_allowlistedCommands(t *testing.T) {
	_, checker := mustNewChecker(t)

	tests := []struct {
		nodeID  string
		command string
	}{
		{"n1", "/usr/bin/rsync -avz /src /dst"},
		{"n1", "/usr/bin/rsync"},
		{"n1", "/usr/bin/systemctl status nginx"},
		{"n1", "/usr/bin/systemctl start foo"},
		{"n1", "/usr/bin/systemctl stop bar"},
		{"n1", "/exact/command"},
	}
	for _, tt := range tests {
		if !checker.Allow(tt.nodeID, tt.command) {
			t.Errorf("Allow(%q, %q) = false, want true (allowlisted)", tt.nodeID, tt.command)
		}
	}
}

// Covers AC-01.008 (US-04): Allow returns denied when command not in allowlist.
func TestChecker_Allow_deniedWhenNotInAllowlist(t *testing.T) {
	_, checker := mustNewChecker(t)

	denied := []struct {
		nodeID  string
		command string
	}{
		{"n1", "/usr/bin/rm -rf /"},
		{"n1", "/usr/sbin/rsync"},
		{"n1", "/usr/bin/systemctl reboot"},
		{"n1", "/exact/command/extra"},
		{"n1", "/other"},
		{"unknown", "/usr/bin/rsync"},
	}
	for _, tt := range denied {
		if checker.Allow(tt.nodeID, tt.command) {
			t.Errorf("Allow(%q, %q) = true, want false (denied)", tt.nodeID, tt.command)
		}
	}
}

// Supporting AC-01.007, AC-01.008 (US-04): Allow(unknown node) returns false.
func TestChecker_Allow_unknownNode(t *testing.T) {
	_, checker := mustNewChecker(t)
	if checker.Allow("nonexistent", "any") {
		t.Error("Allow(unknown node, any) = true, want false")
	}
}

// Supporting AC-01.007 (US-04): NewChecker shares allowlist by path when multiple nodes use same file.
func TestNewChecker_sameFileSharedByNodes(t *testing.T) {
	cfg := &config.Config{
		Version:  1,
		Telegram: config.Telegram{TokenPath: "/t", UsersPath: ""},
		LLMProviders: []config.LLMProvider{
			{Type: "ollama", Endpoint: "http://x", Model: "m"},
		},
		Paths: config.Paths{
			MemoryDir: "/d", LogPath: "/d", VectorIndexPath: "/d", LLMLogDir: "/d", ScheduledTasksPath: "",
		},
		Nodes: map[string]config.Node{
			"n1": {
				Host: "h1", DedicatedUser: "u1",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k1"},
				CommandAllowlistPath: filepath.Join("testdata", "allowlist.txt"),
			},
			"n2": {
				Host: "h2", DedicatedUser: "u2",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k2"},
				CommandAllowlistPath: filepath.Join("testdata", "allowlist.txt"),
			},
		},
	}
	checker, err := NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	if !checker.Allow("n1", "/usr/bin/rsync") {
		t.Error("n1: allowlisted command denied")
	}
	if !checker.Allow("n2", "/usr/bin/rsync") {
		t.Error("n2: allowlisted command denied (shared file)")
	}
}

func mustNewChecker(t *testing.T) (*config.Config, *Checker) {
	t.Helper()
	cfg := &config.Config{
		Version:  1,
		Telegram: config.Telegram{TokenPath: "/t", UsersPath: ""},
		LLMProviders: []config.LLMProvider{
			{Type: "ollama", Endpoint: "http://x", Model: "m"},
		},
		Paths: config.Paths{
			MemoryDir: "/d", LogPath: "/d", VectorIndexPath: "/d", LLMLogDir: "/d", ScheduledTasksPath: "",
		},
		Nodes: map[string]config.Node{
			"n1": {
				Host: "h1", DedicatedUser: "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k1"},
				CommandAllowlistPath: filepath.Join("testdata", "allowlist.txt"),
			},
		},
	}
	// Paths in config are relative to project root (CWD); test runs from package dir so testdata/allowlist.txt resolves
	checker, err := NewChecker(cfg)
	if err != nil {
		t.Fatalf("NewChecker: %v", err)
	}
	return cfg, checker
}
