package ssh

import (
	"pa/internal/config"
	"strings"
	"testing"
)

// Covers AC-009 (US-05): DedicatedUser yields the single user from node config.
func TestDedicatedUser_singleNode(t *testing.T) {
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"nas": {
				Host:                 "192.168.1.10",
				DedicatedUser:        "pa",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/key"},
				CommandAllowlistPath: "/allowlist.txt",
			},
		},
	}
	user, err := DedicatedUser(cfg, "nas")
	if err != nil {
		t.Fatalf("DedicatedUser: %v", err)
	}
	if user != "pa" {
		t.Errorf("DedicatedUser(nas) = %q, want %q", user, "pa")
	}
}

// Covers AC-010 (US-05): DedicatedUser yields correct user per node (multi-node).
func TestDedicatedUser_multiNode(t *testing.T) {
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"nas": {
				Host: "h1", DedicatedUser: "pa_nas",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k1"},
				CommandAllowlistPath: "/a1.txt",
			},
			"server": {
				Host: "h2", DedicatedUser: "pa_server",
				Auth:                 config.NodeAuth{PrivateKeyPath: "/k2"},
				CommandAllowlistPath: "/a2.txt",
			},
		},
	}
	tests := []struct {
		nodeID string
		want   string
	}{
		{"nas", "pa_nas"},
		{"server", "pa_server"},
	}
	for _, tt := range tests {
		user, err := DedicatedUser(cfg, tt.nodeID)
		if err != nil {
			t.Errorf("DedicatedUser(%q): %v", tt.nodeID, err)
			continue
		}
		if user != tt.want {
			t.Errorf("DedicatedUser(%q) = %q, want %q", tt.nodeID, user, tt.want)
		}
	}
}

// Covers AC-009 (US-05): DedicatedUser for unknown node returns error (config-only; no other identity used).
func TestDedicatedUser_unknownNode(t *testing.T) {
	cfg := &config.Config{
		Nodes: map[string]config.Node{
			"nas": {Host: "h1", DedicatedUser: "pa", Auth: config.NodeAuth{PrivateKeyPath: "/k1"}, CommandAllowlistPath: "/a.txt"},
		},
	}
	_, err := DedicatedUser(cfg, "nonexistent")
	if err == nil {
		t.Fatal("DedicatedUser(nonexistent): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("DedicatedUser(nonexistent): error = %v (expect 'not found')", err)
	}
}
