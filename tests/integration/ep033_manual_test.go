//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-033 acceptance criteria that are validated by
// operator checks or command execution rather than deterministic unit logic.

// manual Covers AC-33.003 — month/year catch-up and rollup behavior remains unchanged.
// Verify by running EP-002/EP-033 memoryjob tests and checking no retry policy
// is applied to month/year jobs.
func TestManual_AC33003_MonthYearRetryBehaviorUnchanged(t *testing.T) {
	t.Skip("manual: review memoryjob logs/tests; confirm retry policy is only for catchup_day and summarize_yesterday.")
}

// manual Covers AC-33.013 — `make check` passes on EP-033 branch.
func TestManual_AC33013_MakeCheckPasses(t *testing.T) {
	t.Skip("manual: from repo root run `make check`; command must exit 0 for EP-033.")
}

// manual Covers AC-33.014 — `./bin/validate EP-033` passes.
func TestManual_AC33014_ValidateEP033Passes(t *testing.T) {
	t.Skip("manual: build validator then run `./bin/validate EP-033`; command must exit 0.")
}
