//go:build integration

package integration_test

import "testing"

// Manual traceability for EP-022 acceptance criteria that are not exercised by
// Go tests because they describe either documentation content or an
// operator-run command. These tests intentionally do not run automated checks;
// they anchor ACs for ./bin/validate.

// manual Covers AC-22.009 — operator docs describe PRAGMA policy, single-writer
// expectation, and HTTP timeout fields.
// See: docs/configuration.md §"Local SQLite stores: reliability policy (EP-022)"
// and §"Outbound HTTP timeouts (EP-022)".
func TestManual_AC22009_OperatorDocsDescribePRAGMAAndTimeouts(t *testing.T) {
	t.Skip("manual: read docs/configuration.md; confirm PRAGMA fields, single-writer note, and http_timeout table for llm/embedding/web_tools.")
}

// manual Covers AC-22.011 — `make check` succeeds on the epic branch.
// Verified by running `make check` from the repository root; expected to exit 0.
func TestManual_AC22011_MakeCheckPasses(t *testing.T) {
	t.Skip("manual: from repo root, run `make check`; command must exit 0 on the EP-022 branch.")
}
