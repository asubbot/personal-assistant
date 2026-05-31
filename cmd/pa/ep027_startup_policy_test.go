package main

import (
	"os"
	"strings"
	"testing"
)

// Covers AC-27.003, AC-42.001
func TestEP027_MainRunServerDelegatesToApplication(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	s := string(raw)
	if !strings.Contains(s, "wire.Build(") {
		t.Fatal("main.go runServer should construct application via wire.Build")
	}
	if !strings.Contains(s, "app.BuildToolRegistry()") || !strings.Contains(s, "app.BuildMessageHandler(") {
		t.Fatal("main.go runServer should delegate to application wiring methods")
	}
}

// Covers AC-27.005. Supporting AC-27.006: exercised under full make check.
func TestEP027_StartupSourcesHaveNoGocycloNolint(t *testing.T) {
	for _, name := range []string{"main.go", "wire/build.go", "wire/application.go", "wire/infrastructure.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(raw), "//nolint:gocyclo") {
			t.Fatalf("%s must not contain //nolint:gocyclo (EP-027)", name)
		}
	}
}

// Covers AC-42.004
func TestEP042_ApplicationCloseMethod(t *testing.T) {
	raw, err := os.ReadFile("wire/application.go")
	if err != nil {
		t.Fatalf("read wire/application.go: %v", err)
	}
	if !strings.Contains(string(raw), "func (a *Application) Close()") {
		t.Fatal("wire.Application must expose Close for coordinated teardown")
	}
}

// Covers AC-27.001, AC-27.002, AC-27.003. Supporting AC-27.006: full gate via make check.
func TestEP027_CompositionTypesLinkedInPackage(t *testing.T) {
	var _ *paApplication
	var _ paInfrastructure
}
