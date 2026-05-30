---
artefact: ep-audit-report
epic_id: EP-035
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-035 — Audit report

**Date and time of creation:** 2026-05-30 (UTC)

**Purpose:** Stage 11 audit for [ep-implementation-plan.md](ep-implementation-plan.md) on branch `epic/EP-035-consolidate-small-packages`.

**Pipeline reference:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)

**Related artefacts:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-code-review.md](ep-code-review.md) · [ep-system-design-review.md](ep-system-design-review.md)

---

## Summary

**PASS.** All nine implementation-plan tasks are complete. Stage 7 system design review gate is **Pass** (iteration 1, zero open Blocker/Major/Medium/Minor). Stage 10 code review iteration 2 gate is **Pass** (zero open Blocker/Major/Medium/Minor). `make check` passed with **75.9%** total statement coverage. `./bin/validate EP-035` reports **11/11** in-scope ACs traced (100.0% automated); nine structural ACs are **MANUAL ONLY** and were verified by audit inspection (removed directories, zero dead imports, config/doc diff). `./bin/validate pipeline EP-035` reports no gate violations after this report is written. EP-035 is ready for merge.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| 1. Delete `internal/logging` stub package | Done | `internal/logging/` removed from committed tree; zero `pa/internal/logging` imports. |
| 2. Add `internal/prompt` production sources | Done | `markers.go` and `wrap.go` created; byte-identical marker constants and `TrustPolicy`. |
| 3. Add `internal/prompt` unit tests with independent goldens (S-001) | Done | Non-tautological golden literals; forbidden-line and wrap tests moved from legacy packages. |
| 4. Rewrite all seven importers to `pa/internal/prompt` | Done | `handler.go`, `system_tail.go`, `handler_test.go`, `write_memory.go`, `runtimeskills/package.go`, two integration test files. |
| 5. Delete legacy `internal/promptmarkers` and `internal/systemprompt` | Done | Landed with task 4; zero legacy import paths in `*.go`. |
| 6. Add `tests/integration/concurrent_write_test.go` | Done | `TestConcurrentWrites_NoBusyErrors` relocated with `//go:build integration`; retains `// Covers AC-22.010` and `// Covers AC-35.005`. |
| 7. Delete `internal/reliability` | Done | Package removed; test lives under `tests/integration`. |
| 8. Update `docs/configuration.md` reliability test path | Done | Cites `tests/integration`; no `internal/reliability` reference. |
| 9. Epic quality gates and removal grep | Done | `make check` exit 0; config untouched; final grep zero matches for removed packages. |

Reference: [ep-implementation-plan.md](ep-implementation-plan.md)

**Change set (branch vs `main`):** 30 files changed, 1637 insertions(+), 110 deletions(−) — product changes: merged `internal/prompt`, seven importer rewrites, relocated concurrent-write test, `tests/integration/anchor.go` (load-bearing package-name anchor), `docs/configuration.md` path update, plus epic SDLC artefacts.

---

## Test results and coverage

| Command | Result | Notes |
|---------|--------|-------|
| `make check` | **Pass** (exit 0) | fmt, vet, golangci-lint (0 issues), govulncheck clean, unit + integration tests, `test-race`, module boundaries OK |
| `./bin/validate EP-035` | **Pass** (exit 0) | in-scope 11/11 traced (100.0% automated); 9 deferred MANUAL ONLY ACs |
| `./bin/validate EP-035 --json` | **Pass** (exit 0) | `traceability_ratio: 1`, `automated_ratio: 1`, `gaps: []` for in-scope ACs |
| `./bin/validate pipeline EP-035` | **Pass** (exit 0) | No gate violations; stage 11 artefact present after this write |

**Total statement coverage:** `total: (statements) 75.9%`

**Notable package coverage (from `make check`):**

| Package / area | Coverage | EP-035 relevance |
|----------------|----------|------------------|
| `pa/internal/prompt` | 100% on exported helpers (`ForbiddenMarkerLines`, `TextContainsForbiddenMarkerLine`, all three `Wrap*` functions) | New merged package; full unit coverage on security-sensitive API |
| `pa/tests/integration` | Relocated race test passes under `-race -tags=integration` | AC-35.004/005; preserves EP-022 intent |

**Manual verification (deferred ACs):**

| Check | Result |
|-------|--------|
| Removed dirs absent from `HEAD` tree (`internal/logging`, `internal/reliability`, `internal/promptmarkers`, `internal/systemprompt`) | **Pass** — not in committed tree (`git ls-tree HEAD`) |
| Dead-import grep `pa/internal/(logging\|reliability\|promptmarkers\|systemprompt)` in `*.go` | **Pass** — zero matches |
| `git diff main...HEAD -- config.json .config/config.json internal/config` | **Pass** — empty diff |
| `docs/configuration.md` concurrent-writer path | **Pass** — cites `tests/integration`; no `internal/reliability` |

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-35.001](ep-acceptance-criteria.md#ac-35-001) | [REQ-35.001](ep-requirements.md#req-35-001--delete-logging-stub-package) | — | — | — | ✓ | Audit: `internal/logging/` absent from committed tree |
| [AC-35.002](ep-acceptance-criteria.md#ac-35-002) | [REQ-35.002](ep-requirements.md#req-35-002--no-logging-package-imports) | — | — | — | ✓ | Audit grep: zero `pa/internal/logging` in `*.go`; `make check` build pass |
| [AC-35.003](ep-acceptance-criteria.md#ac-35-003) | [REQ-35.003](ep-requirements.md#req-35-003--delete-reliability-test-package) | — | — | — | ✓ | Audit: `internal/reliability/` absent from committed tree |
| [AC-35.004](ep-acceptance-criteria.md#ac-35-004) | [REQ-35.004](ep-requirements.md#req-35-004--relocate-concurrent-write-test) | — | ✓ | — | — | `tests/integration/concurrent_write_test.go::TestConcurrentWrites_NoBusyErrors` |
| [AC-35.005](ep-acceptance-criteria.md#ac-35-005) | [REQ-35.005](ep-requirements.md#req-35-005--preserve-ac-22-010-race-test-intent) | — | ✓ | — | — | `tests/integration/concurrent_write_test.go::TestConcurrentWrites_NoBusyErrors` (also `// Covers AC-22.010`) |
| [AC-35.006](ep-acceptance-criteria.md#ac-35-006) | [REQ-35.006](ep-requirements.md#req-35-006--update-reliability-test-documentation-path) | — | — | — | ✓ | Audit: `docs/configuration.md` cites `tests/integration`; no `internal/reliability` |
| [AC-35.007](ep-acceptance-criteria.md#ac-35-007) | [REQ-35.007](ep-requirements.md#req-35-007--provide-merged-internalprompt-package) | ✓ | — | — | — | `internal/prompt/markers_test.go::TestForbiddenMarkerLines_allSixCanonical`; `internal/prompt/wrap_test.go::TestMergedPromptAPI_surface` |
| [AC-35.008](ep-acceptance-criteria.md#ac-35-008) | [REQ-35.008](ep-requirements.md#req-35-008--byte-identical-trust-policy) | ✓ | — | — | — | `internal/prompt/wrap_test.go::TestTrustPolicy_byteIdentical`; `TestTrustPolicyPrefix` |
| [AC-35.009](ep-acceptance-criteria.md#ac-35-009) | [REQ-35.009](ep-requirements.md#req-35-009--byte-identical-marker-constants) | ✓ | — | — | — | `internal/prompt/markers_test.go::TestMarkerConstants_byteIdentical` |
| [AC-35.010](ep-acceptance-criteria.md#ac-35-010) | [REQ-35.010](ep-requirements.md#req-35-010--equivalent-forbidden-marker-validation) | ✓ | — | — | — | `internal/prompt/markers_test.go::TestTextContainsForbiddenMarkerLine_*`, `TestForbiddenMarkerLines_allSixCanonical` |
| [AC-35.011](ep-acceptance-criteria.md#ac-35-011) | [REQ-35.011](ep-requirements.md#req-35-011--equivalent-block-wrap-helpers) | ✓ | — | — | — | `internal/prompt/wrap_test.go::TestWrapRetrievedContext_nonEmpty`, `TestWrapToolInstructions`, `TestWrapRuntimeSkills`, `TestWrapHelpers_emptyInner` |
| [AC-35.012](ep-acceptance-criteria.md#ac-35-012) | [REQ-35.012](ep-requirements.md#req-35-012--delete-legacy-prompt-packages) | — | — | — | ✓ | Audit: `internal/promptmarkers/`, `internal/systemprompt/` absent from committed tree |
| [AC-35.013](ep-acceptance-criteria.md#ac-35-013) | [REQ-35.013](ep-requirements.md#req-35-013--no-legacy-prompt-package-imports) | — | — | — | ✓ | Audit grep: zero legacy prompt imports; `make check` build pass |
| [AC-35.014](ep-acceptance-criteria.md#ac-35-014) | [REQ-35.014](ep-requirements.md#req-35-014--update-prompt-package-importers) | — | — | — | ✓ | Audit: seven listed importers compile via `make check`; all use `pa/internal/prompt` |
| [AC-35.015](ep-acceptance-criteria.md#ac-35-015) | [REQ-35.015](ep-requirements.md#req-35-015--no-configjson-changes) | — | — | — | ✓ | Audit: empty diff for `config.json`, `.config/config.json`, `internal/config` |
| [AC-35.016](ep-acceptance-criteria.md#ac-35-016) | [REQ-35.016](ep-requirements.md#req-35-016--quality-gate-passes) | — | — | — | ✓ | Audit: `make check` exit 0 (2026-05-30) |
| [AC-35.017](ep-acceptance-criteria.md#ac-35-017) | [REQ-35.017](ep-requirements.md#req-35-017--preserve-system-prompt-assembly) | ✓ | ✓ | — | — | `internal/core/handler_test.go::TestHandleMessage_passesSystemAndUserMessages`; `tests/integration/runtime_skills_handler_test.go::TestRuntimeSkills_handler_trustAndRetrievedMarkers` |
| [AC-35.018](ep-acceptance-criteria.md#ac-35-018) | [REQ-35.018](ep-requirements.md#req-35-018--preserve-runtime-skills-marker-rejection) | — | ✓ | — | — | `tests/integration/runtime_skills_config_test.go::TestRuntimeSkills_configLoad_rejectsForbiddenMarkerInSkill` |
| [AC-35.019](ep-acceptance-criteria.md#ac-35-019) | [REQ-35.019](ep-requirements.md#req-35-019--preserve-memory-indexing-marker-rejection) | ✓ | ✓ | — | — | `internal/tools/ep016_memory_tools_test.go::TestWriteMemoryTool_rejectsForbiddenMarkerLine`; `tests/integration/runtime_skills_handler_test.go::TestRuntimeSkills_handler_indexTurnRejectsForbiddenMarkerLine` |
| [AC-35.020](ep-acceptance-criteria.md#ac-35-020) | [REQ-35.020](ep-requirements.md#req-35-020--ep-013-prompt-tests-retain-intent) | ✓ | ✓ | — | — | Marker/wrap/handler/runtime-skills tests (see validator: 4 tests across `internal/prompt`, integration packages) |

### Notes

- AC mapping primary source: `./bin/validate EP-035 --json` (2026-05-30 audit run).
- **Unit:** tests under `internal/` per [strategy.md](../../strategy.md) §2.
- **Integration:** tests under `tests/integration` with `//go:build integration`; `-race` coverage included in `make check` via `test-race`.
- **Manual (deferred):** nine ACs marked MANUAL ONLY in [ep-acceptance-criteria.md](ep-acceptance-criteria.md); validator treats them as deferred (not automated gaps). Audit verified each via tree inspection, grep, doc read, and `make check`.
- **E2E:** not used for EP-035 ACs.
- Stage 10 gate: [ep-code-review.md](ep-code-review.md) iteration 2 — Pass, zero open Blocker/Major/Medium/Minor.

---

## Quality gate

| Check | Result |
|-------|--------|
| `make check` | **Pass** — format, vet, lint (0 issues), tests with race detector, govulncheck, module boundaries |
| `./bin/validate EP-035` | **Pass** — in-scope 11/11 ACs traced (100.0% automated); all ACs covered for audit |
| `./bin/validate pipeline EP-035` | **Pass** — no gate violations |
| System design review (stage 7) | **Pass** — iteration 1, open counts all zero |
| Code review (stage 10) | **Pass** — iteration 2, open counts all zero |

---

## Gaps, risks, recommendations

### Gaps

None. All implementation-plan tasks are complete. All 20 acceptance criteria are satisfied (11 automated + 9 manual-only verified).

### Risks

- **Local empty directories (low):** Untracked empty dirs for removed packages may remain on developer machines after checkout; they are not in Git and do not affect build. Code review iteration 1 noted this as informational (Nit).
- **`tests/integration/anchor.go` coupling (low, accepted):** A non-test file sorting first is required so `integration_test` package name resolves when `-tags=integration` is set. Code review iteration 2 confirmed this is load-bearing, not dead code; future integration files must preserve sort-order invariant or an equivalent anchor.
- **`internal/lifecyclelog` deferred (informational):** Out of epic scope per [ep-scope.md](ep-scope.md); no regression risk from EP-035.

### Recommendations

- After merge: update [ep-scope.md](ep-scope.md) status from `NEW` to `DONE`.
- Optional local hygiene: remove leftover empty dirs (`internal/logging`, etc.) if present on disk after merge.
- No follow-up epic required for EP-035 delivery.

---

## Overall verdict

**PASS — ready for merge.** Requirements (REQ-35.001–020), acceptance criteria (AC-35.001–020), system design, implementation plan, code review, and quality gates are satisfied. Security-sensitive invariants (`TrustPolicy`, six canonical marker lines, forbidden-line validation, wrap helpers) are byte-identical to pre-EP-035 baseline. No config schema changes. Residual risks are low and documented above.
