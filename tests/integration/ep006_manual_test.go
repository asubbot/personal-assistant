//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-006 operator scenarios in ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md (Navigation table / #mt-* sections).

// manual Covers AC-06.011 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-esc-off
func TestManual_MT_EscOff(t *testing.T) {
	t.Skip("manual: escalation disabled — no provider advance; ep-manual-tests.md#mt-esc-off")
}

// manual Covers AC-06.005, AC-06.006 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-esc-on
func TestManual_MT_EscOn(t *testing.T) {
	t.Skip("manual: escalation enabled — qualifying failure advances; ep-manual-tests.md#mt-esc-on")
}

// manual Covers AC-06.007 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-max-esc
func TestManual_MT_MaxEsc(t *testing.T) {
	t.Skip("manual: max escalations per user message; ep-manual-tests.md#mt-max-esc")
}

// manual Covers AC-06.008 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-baseline-reset
func TestManual_MT_BaselineReset(t *testing.T) {
	t.Skip("manual: baseline reset on next user message; ep-manual-tests.md#mt-baseline-reset")
}

// manual Covers AC-06.013 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-hermes
func TestManual_MT_HermesParseEscalation(t *testing.T) {
	t.Skip("manual: Hermes parse failure and escalation; ep-manual-tests.md#mt-hermes")
}

// manual Covers AC-06.009 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-no-secrets
func TestManual_MT_NoSecretsInLogs(t *testing.T) {
	t.Skip("manual: no secrets in escalation and LLM logs; ep-manual-tests.md#mt-no-secrets")
}

// manual Covers AC-06.004 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-non-qual
func TestManual_MT_NonQualifyingNoEscalation(t *testing.T) {
	t.Skip("manual: non-qualifying tool failures — no escalation; ep-manual-tests.md#mt-non-qual")
}

// manual Covers AC-06.002 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-invalid-config
func TestManual_MT_InvalidEscalationConfig(t *testing.T) {
	t.Skip("manual: invalid escalation config at startup; ep-manual-tests.md#mt-invalid-config")
}

// manual Covers AC-06.010 — see ai-sdlc-artefacts/epics/EP-006/ep-manual-tests.md#mt-operator
func TestManual_MT_OperatorChecklist(t *testing.T) {
	t.Skip("manual: operator checklist after deploy; ep-manual-tests.md#mt-operator")
}
