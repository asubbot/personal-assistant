// Package integration_test holds integration tests for PersonalAssistant.
// Test files use the build tag "integration". Run with: go test -tags=integration ./tests/integration/...
//
// Runtime skills and system-prompt marker tests live in runtime_skills_*.go and use
// pa/internal/core helpers built with //go:build integration (integration_export.go).
package integration_test
