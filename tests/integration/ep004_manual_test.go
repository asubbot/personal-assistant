//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-004 scenarios in ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md.
// Each test maps to a Trace section in that document.

// manual Covers AC-04.010, AC-04.011 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#sonos-tool-end-to-end (same catalog path as other tools)
func TestManual_AC04010_SonosEndToEnd(t *testing.T) {
	t.Skip("manual: Sonos tool E2E on real deployment; steps in ep-manual-tests.md")
}

// manual Covers AC-04.012 — see ai-sdlc-artefacts/epics/EP-004/ep-audit-report.md (make check / regression)
func TestManual_AC04012_RegressionSuite(t *testing.T) {
	t.Skip("manual: AC-04.012 — run make check before release; ep-audit-report.md")
}

// manual Covers AC-04.021 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#tool-index-build-logging (failure path relates to AC-04.018)
func TestManual_AC04021_ToolIndexBuildLogging(t *testing.T) {
	t.Skip("manual: INFO/ERROR logs on index build success/failure; see ep-manual-tests.md")
}

// manual Covers AC-04.026 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#system_prompt-in-system-message
func TestManual_AC04026_SystemPromptInSystemMessage(t *testing.T) {
	t.Skip("manual: verify system_prompt in LLM log; see ep-manual-tests.md")
}

// manual Covers AC-04.027 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#hermes-tool-list-in-prompt
func TestManual_AC04027_HermesToolListInPrompt(t *testing.T) {
	t.Skip("manual: Hermes provider tool list in prompt; see ep-manual-tests.md")
}

// manual Covers AC-04.022, AC-04.023, AC-04.024 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#text-based-tool-flow
func TestManual_AC04022_04024_TextBasedToolFlow(t *testing.T) {
	t.Skip("manual: text-based tool flow E2E; see ep-manual-tests.md")
}

// manual Covers AC-04.029 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#shell-metacharacter-rejection
func TestManual_AC04029_ShellMetacharacterRejection(t *testing.T) {
	t.Skip("manual: shell metacharacter rejection in chat; see ep-manual-tests.md")
}

// manual Covers AC-04.013 — see ai-sdlc-artefacts/epics/EP-004/ep-manual-tests.md#tool-invocation-in-logs
func TestManual_AC04013_ToolInvocationInLogs(t *testing.T) {
	t.Skip("manual: tool invocation fields in logs; see ep-manual-tests.md")
}
