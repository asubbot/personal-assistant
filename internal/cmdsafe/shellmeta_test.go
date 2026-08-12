package cmdsafe

import (
	"strings"
	"testing"
)

// Covers AC-04.029 (REQ-04.031): safe command strings pass rejection check.
func TestRejectShellMetacharacters_safeCommands(t *testing.T) {
	for _, cmd := range []string{
		"",
		"uptime",
		"echo hello",
		"/volume1/homes/x/.local/bin/sonos Kitchen volume 30",
	} {
		if err := RejectShellMetacharacters(cmd); err != nil {
			t.Errorf("RejectShellMetacharacters(%q): %v, want nil", cmd, err)
		}
	}
}

// Covers AC-04.029 (REQ-04.031): forbidden shell sequences rejected before execution.
func TestRejectShellMetacharacters_rejectsForbidden(t *testing.T) {
	cases := []string{
		"echo hi; rm -rf /",
		"foo & bar",
		"a | b",
		"line1\nline2",
		"a\rb",
		"echo $(id)",
		"echo `id`",
	}
	for _, cmd := range cases {
		err := RejectShellMetacharacters(cmd)
		if err == nil {
			t.Fatalf("RejectShellMetacharacters(%q): want error", cmd)
		}
		if !strings.Contains(err.Error(), "REQ-04.031") {
			t.Errorf("RejectShellMetacharacters(%q): error %q", cmd, err.Error())
		}
	}
}
