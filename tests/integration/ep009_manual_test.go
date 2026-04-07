//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-009 operator verification in ai-sdlc-artefacts/epics/EP-009/ep-manual-tests.md.

// manual Covers AC-09.005, AC-09.006, AC-09.007 — sandbox images on node (see summary table)
func TestManual_AC09005_09007_SandboxImagesOnNode(t *testing.T) {
	t.Skip("manual: docker image ls for pa-sandbox:python, pa-sandbox:node, pa-sandbox:base; ep-manual-tests.md")
}

// manual Covers AC-09.001, AC-09.002, AC-09.003, AC-09.004, AC-09.014 — docker run flags / remote command substitution
func TestManual_AC09001_09004_09014_DockerRunFlags(t *testing.T) {
	t.Skip("manual: remote command after substitution (network, memory, CPU, timeout, startup); ep-manual-tests.md")
}

// manual Covers AC-09.018 — network isolation template
func TestManual_AC09018_NetworkNoneIsolation(t *testing.T) {
	t.Skip("manual: template with --network none; outbound blocked; ep-manual-tests.md")
}

// manual Covers AC-09.008, AC-09.009, AC-09.010, AC-09.011, AC-09.012, AC-09.013, AC-09.017 — E2E operator confirmation (primarily automated elsewhere)
func TestManual_AC09008_09017_EndToEndOperatorSetup(t *testing.T) {
	t.Skip("manual: end-to-end operator setup per happy path; ep-manual-tests.md § Happy path — end-to-end")
}
