package main

import (
	"os"
	"strings"
	"testing"
)

// Covers AC-27.003
func TestEP027_MainRunServerDelegatesToApplication(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "newPAApplication(") {
		t.Fatal("main.go runServer should construct paApplication via newPAApplication")
	}
	if !strings.Contains(s, "app.buildToolRegistry()") || !strings.Contains(s, "app.buildMessageHandler(") {
		t.Fatal("main.go runServer should delegate to paApplication wiring methods")
	}
}

// Covers AC-27.005. Supporting AC-27.006: exercised under full make check.
func TestEP027_StartupSourcesHaveNoGocycloNolint(t *testing.T) {
	for _, name := range []string{"main.go", "application.go", "setup_infra.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(raw), "//nolint:gocyclo") {
			t.Fatalf("%s must not contain //nolint:gocyclo (EP-027)", name)
		}
	}
}

// Covers AC-27.001, AC-27.002, AC-27.003. Supporting AC-27.006: full gate via make check.
func TestEP027_CompositionTypesLinkedInPackage(t *testing.T) {
	var _ *paApplication
	var _ paInfrastructure
}
