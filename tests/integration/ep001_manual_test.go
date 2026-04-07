//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-001 acceptance criteria documented in
// ai-sdlc-artefacts/epics/EP-001/ep-manual-test-scenarios.md (and related audit/plan for gaps).
// These tests intentionally do not run automated checks; they anchor ACs for ./bin/validate.

// manual Covers AC-01.004 — see ai-sdlc-artefacts/epics/EP-001/ep-manual-test-scenarios.md#ac-01-004
func TestManual_AC01004_ImageBuildsOnTarget(t *testing.T) {
	t.Skip("manual: Docker image build/run on x86_64 (e.g. DS220+); steps in ep-manual-test-scenarios.md")
}

// manual Covers AC-01.032 — see ai-sdlc-artefacts/epics/EP-001/ep-manual-test-scenarios.md#ac-01-032
func TestManual_AC01032_VerifyNodes(t *testing.T) {
	t.Skip("manual: pa -verify-nodes with real SSH nodes; steps in ep-manual-test-scenarios.md")
}

// manual Covers AC-01.025 — module boundaries: make check-boundaries / ep-implementation-plan + strategy §2.3
func TestManual_AC01025_ModuleBoundaries(t *testing.T) {
	t.Skip("manual: run ./scripts/check-module-boundaries.sh or make check-boundaries; review strategy.md §2.3")
}

// manual Covers AC-01.043 — LLM provider fallback; see ai-sdlc-artefacts/epics/EP-001/ep-audit-report.md and ep-acceptance-criteria.md#ac-01-043
func TestManual_AC01043_LLMProviderFallback(t *testing.T) {
	t.Skip("manual: multi-provider failure ordering; see ep-audit-report.md and ep-acceptance-criteria.md")
}
