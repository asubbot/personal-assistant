//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-002 acceptance criteria where full schedule / catch-up / E2E timing is operator-verified.
// Anchors ./bin/validate; see ai-sdlc-artefacts/epics/EP-002/ep-scope.md Manual E2E scenario.

// manual Covers AC-02.001 — previous-day summarization at 01:00 local pa_timezone (built-in schedule)
func TestManual_AC02001_DaySummarizationSchedule(t *testing.T) {
	t.Skip("manual: wait for 01:00 in pa_timezone or use ep-scope.md E2E; automatic day job in internal/memoryjob")
}

// manual Covers AC-02.002 — month rollup on first local day at 01:00
func TestManual_AC02002_MonthSummarizationSchedule(t *testing.T) {
	t.Skip("manual: first day of month at 01:00 local; ep-scope.md Part D")
}

// manual Covers AC-02.003 — year rollup on first local day at 01:00
func TestManual_AC02003_YearSummarizationSchedule(t *testing.T) {
	t.Skip("manual: first day of year at 01:00 local; ep-scope.md Part D")
}

// manual Covers AC-02.004 — no external cron required for summarization triggers
func TestManual_AC02004_NoExternalCron(t *testing.T) {
	t.Skip("manual: confirm no crontab entry required; behaviour is in-process (ep-scope.md)")
}

// manual Covers AC-02.005 — startup day catch-up when logs exist and summary missing
func TestManual_AC02005_StartupCatchUpDay(t *testing.T) {
	t.Skip("manual: restart bot with yesterday logs and no summary.md; ep-scope.md Part B")
}

// manual Covers AC-02.006 — startup month catch-up
func TestManual_AC02006_StartupCatchUpMonth(t *testing.T) {
	t.Skip("manual: ep-scope.md Part D / catch-up rules")
}

// manual Covers AC-02.007 — startup year catch-up
func TestManual_AC02007_StartupCatchUpYear(t *testing.T) {
	t.Skip("manual: ep-scope.md Part D / catch-up rules")
}
