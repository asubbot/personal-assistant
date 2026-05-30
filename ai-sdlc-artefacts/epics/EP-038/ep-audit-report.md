---
artefact: ep-audit-report
epic_id: EP-038
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-038 — Audit report

**Date and time of creation:** 2026-05-31 (UTC)

**Purpose:** Stage 11 audit for [ep-implementation-plan.md](ep-implementation-plan.md) on branch `epic/EP-038-refactor-core-handler`.

**Pipeline reference:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)

**Related artefacts:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-code-review.md](ep-code-review.md) · [ep-system-design-review.md](ep-system-design-review.md)

---

## Summary

**PASS.** All implementation-plan phases (0–7) are complete per stage 10 verification and this audit’s `make check` run. Stage 7 system design review iteration 2 and stage 10 code review iteration 1 gates are **Pass** (zero open Blocker/Major/Medium/Minor). `make check` passed with **76.0%** total statement coverage and module boundaries OK. Epic gates `./bin/validate ears EP-038` (25 reqs, 0 errors) and `./bin/validate req EP-038` (25/25) pass. `./bin/validate EP-038` (AC↔test trace) exits **1** by design: EP-038 adds no `// Covers AC-38.xxx` annotations; twelve in-scope ACs are exercised by existing handler suites (green under `make check`) but lack validator trace comments — see [Gaps](#gaps-risks-recommendations). `./bin/validate pipeline EP-038` passes after this report. EP-038 is ready for merge.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| **Phase 0** (0.1) Prerequisite gate | Done | Merge-base includes EP-035/036/037 on `main` (code review AC-38.001). |
| **Phase 1** (1.1–1.2) `handler_memory.go` | Done | Memory/RAG/turn-index symbols moved; `vector_merge` / `memory_vectors` / `system_tail` unchanged vs `main`. |
| **Phase 2** (2.1–2.2) `handler_tools.go` | Done | Tool merge/execution in `handler_tools.go`; `runtime_tools` / `dynamic_tool_selection` / tier tail ownership unchanged. |
| **Phase 3** (3.1–3.2) `handler_llm.go` | Done | LLM/router/tool-loop methods in `handler_llm.go`; EP-034 regression slice green via `make check`. |
| **Phase 4** (4.1–4.2) Slim `handler.go` | Done | **132** LOC orchestration (≤ ~200); public wiring unchanged (`adapter.go`, `run.go`, `integration_export.go` no diff vs `main`). |
| **Phase 5** (5.1) Tier boundary | Done | `handler_tier_main_prompt.go` unchanged vs `main`; two-tier switch; no `TierFullLite` / strategy framework in production `internal/core`. |
| **Phase 6** (6.1–6.2) Test/config parity | Done | No test assertion churn vs `main`; full `internal/core` and integration slices green. |
| **Phase 7** (7.1–7.3) Manual gates + quality | Done | Grep/LOC/diff/`make check`/`validate ears`/`validate req` verified in code review and re-run in this audit. Plan checkboxes remain unchecked (F-001 suggestion only). |

Reference: [ep-implementation-plan.md](ep-implementation-plan.md)

**Delivered change set (`git diff --stat main...HEAD`):** 16 files changed, 2405 insertions(+), 531 deletions(−) (includes EP-038 SDLC artefacts and diagrams; product: slim `handler.go` + three new `handler_*.go` files, cut/paste from former ~663 LOC god handler).

---

## Test results and coverage

| Command | Result | Notes |
|---------|--------|-------|
| `make check` | **Pass** (exit 0) | fmt, vet, golangci-lint, govulncheck, race tests, coverage, **module boundaries OK** |
| `./bin/validate EP-038` | **Fail** (exit 1) | in-scope **0/12** traced, **0.0%** automated; **13 deferred**, **0 obsolete**, total ACs **25** — no `Covers AC-38.xxx` in codebase (epic design) |
| `./bin/validate EP-038 --json` | **Fail** (exit 1) | `traceability_ratio: 0`, `automated_ratio: 0`; 12 `not_covered` in-scope ACs |
| `./bin/validate ears EP-038` | **Pass** (exit 0) | **25** requirements, **0** errors, 26 EARS weak-pattern warnings |
| `./bin/validate req EP-038` | **Pass** (exit 0) | **25/25** REQs covered, 0 orphan refs |
| `./bin/validate pipeline EP-038` | **Pass** (exit 0) | Stages 3–10 present with gates pass; stage 11 report present after this artefact |

**Total statement coverage:** `total: (statements) 76.0%`

**EP-038–relevant packages (from `make check`):** `internal/core` — handler refactor with zero assertion churn; existing `handler_ep017_test.go`, `handler_ep034_regression_test.go`, `handler_tier_main_prompt_test.go`, and integration suites exercise behaviour parity.

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-38.001](ep-acceptance-criteria.md#ac-38-001) | [REQ-38.001](ep-requirements.md#req-38-001--land-after-ep-035-ep-036-ep-037-merged) | — | — | — | ✓ | MANUAL ONLY — merge-base / branch history (code review §AC-38.001) |
| [AC-38.002](ep-acceptance-criteria.md#ac-38-002) | [REQ-38.002](ep-requirements.md#req-38-002--keep-struct-and-handlemessage-turn-sequence) | — | ✓ | — | — | `internal/core/handler_test.go::TestHandleMessage_passesSystemAndUserMessages`; `handler_ep017_test.go::TestHandleMessage_SimpleTier_NoToolsNoRAG` (no `Covers` trace) |
| [AC-38.003](ep-acceptance-criteria.md#ac-38-003) | [REQ-38.003](ep-requirements.md#req-38-003--reduce-handlergo-to-orchestration-200-loc) | — | — | — | ✓ | MANUAL ONLY — `wc -l handler.go` → **132** LOC |
| [AC-38.004](ep-acceptance-criteria.md#ac-38-004) | [REQ-38.004](ep-requirements.md#req-38-004--retain-shared-turn-constants-in-handlergo) | ✓ | — | — | — | `handler.go` consts; exercised via `handler_ep034_regression_test.go` tool-round cap (no `Covers` trace) |
| [AC-38.005](ep-acceptance-criteria.md#ac-38-005) | [REQ-38.005](ep-requirements.md#req-38-005--extract-llm-completion-and-tool-loop-methods) | — | — | — | ✓ | MANUAL ONLY — grep symbol ownership in `handler_llm.go` (code review) |
| [AC-38.006](ep-acceptance-criteria.md#ac-38-006) | [REQ-38.006](ep-requirements.md#req-38-006--preserve-router-usage-round-cap-message-roles) | — | ✓ | — | — | `internal/core/handler_ep034_regression_test.go::TestHandleMessage_toolFailure_doesNotAdvanceProvider` |
| [AC-38.007](ep-acceptance-criteria.md#ac-38-007) | [REQ-38.007](ep-requirements.md#req-38-007--extract-tool-merge-selection-and-execution) | — | — | — | ✓ | MANUAL ONLY — grep `handler_tools.go` (code review) |
| [AC-38.008](ep-acceptance-criteria.md#ac-38-008) | [REQ-38.008](ep-requirements.md#req-38-008--leave-runtime_tools-and-dynamic_tool_selection-in-place) | — | — | — | ✓ | MANUAL ONLY — grep unchanged modules (code review) |
| [AC-38.009](ep-acceptance-criteria.md#ac-38-009) | [REQ-38.009](ep-requirements.md#req-38-009--extract-rag-chunk-and-turn-index-methods) | — | — | — | ✓ | MANUAL ONLY — grep `handler_memory.go` (code review) |
| [AC-38.010](ep-acceptance-criteria.md#ac-38-010) | [REQ-38.010](ep-requirements.md#req-38-010--leave-vector_merge-memory_vectors-system_tail-ownership) | — | — | — | ✓ | MANUAL ONLY — no diff vs `main` for those files |
| [AC-38.011](ep-acceptance-criteria.md#ac-38-011) | [REQ-38.011](ep-requirements.md#req-38-011--retain-tier-main-prompt-dispatch-in-handler_tier_main_promptgo) | — | ✓ | — | — | `handler_tier_main_prompt_test.go`; `handler_ep017_test.go`; `handler_ep018_test.go` |
| [AC-38.012](ep-acceptance-criteria.md#ac-38-012) | [REQ-38.012](ep-requirements.md#req-38-012--no-new-tier-values-or-full_lite-revival) | ✓ | — | — | — | `internal/core/handler_ep036_test.go::TestEP036_validateCommandExitZero` (two-tier contract) |
| [AC-38.013](ep-acceptance-criteria.md#ac-38-013) | [REQ-38.013](ep-requirements.md#req-38-013--use-simple-tier-switch-no-strategy-framework) | — | — | — | ✓ | MANUAL ONLY — `switch tier` grep in `handler_tier_main_prompt.go` |
| [AC-38.014](ep-acceptance-criteria.md#ac-38-014) | [REQ-38.014](ep-requirements.md#req-38-014--preserve-messagehandlerhandlemessage-signature) | — | ✓ | — | — | `handler_test.go` + full `make check` / core compile |
| [AC-38.015](ep-acceptance-criteria.md#ac-38-015) | [REQ-38.015](ep-requirements.md#req-38-015--preserve-buildmessagehandler-and-run-surfaces) | ✓ | — | — | — | `internal/core/run_test.go` (via `make check`; no config diff) |
| [AC-38.016](ep-acceptance-criteria.md#ac-38-016) | [REQ-38.016](ep-requirements.md#req-38-016--preserve-newintegrationconversationhandler) | — | ✓ | — | — | `tests/integration/runtime_skills_handler_test.go` (via `make check`) |
| [AC-38.017](ep-acceptance-criteria.md#ac-38-017) | [REQ-38.017](ep-requirements.md#req-38-017--no-configjson-schema-change) | ✓ | — | — | — | No `internal/config` diff vs `main`; `go test ./internal/config/...` green |
| [AC-38.018](ep-acceptance-criteria.md#ac-38-018) | [REQ-38.018](ep-requirements.md#req-38-018--behaviour-parity-on-tier-tools-prompts-routing) | — | ✓ | — | — | `tools_selection_parity_test.go`; `handler_ep017_test.go`; `handler_ep018_test.go`; `handler_ep018_coverage_test.go` |
| [AC-38.019](ep-acceptance-criteria.md#ac-38-019) | [REQ-38.019](ep-requirements.md#req-38-019--preserve-ep-013034036037-contracts) | — | ✓ | — | — | `handler_ep034_regression_test.go`; `handler_ep036_test.go`; `tools_selection_parity_test.go`; `internal/config/tools_selection_test.go` |
| [AC-38.020](ep-acceptance-criteria.md#ac-38-020) | [REQ-38.020](ep-requirements.md#req-38-020--existing-handler-tests-pass-without-assertion-changes) | — | ✓ | — | — | Full `go test ./internal/core/...` via `make check`; zero `*_test.go` assertion diff vs `main` |
| [AC-38.021](ep-acceptance-criteria.md#ac-38-021) | [REQ-38.021](ep-requirements.md#req-38-021--make-check-passes) | — | — | — | ✓ | MANUAL ONLY — `make check` exit 0 (this audit run) |
| [AC-38.022](ep-acceptance-criteria.md#ac-38-022) | [REQ-38.022](ep-requirements.md#req-38-022--validate-ears-ep-038-passes) | — | — | — | ✓ | MANUAL ONLY — `./bin/validate ears EP-038` exit 0 (not `./bin/validate EP-038` ac trace) |
| [AC-38.023](ep-acceptance-criteria.md#ac-38-023) | [REQ-38.023](ep-requirements.md#req-38-023--no-product-behaviour-changes) | — | — | — | ✓ | MANUAL ONLY — symmetric diff review; import-only asymmetry (code review) |
| [AC-38.024](ep-acceptance-criteria.md#ac-38-024) | [REQ-38.024](ep-requirements.md#req-38-024--do-not-rename-or-export-conversationhandler) | — | — | — | ✓ | MANUAL ONLY — grep unexported `conversationHandler` |
| [AC-38.025](ep-acceptance-criteria.md#ac-38-025) | [REQ-38.025](ep-requirements.md#req-38-025--optional-namingcomments-cleanup-only-in-tier-prompt-file) | — | — | — | ✓ | N/A — no `handler_tier_main_prompt.go` diff vs `main` |

### Notes

- Primary automated-trace source: `./bin/validate EP-038 --json` (audit run 2026-05-31) — **0/12** in-scope traced because EP-038 deliberately omits `// Covers AC-38.xxx` comments ([ep-implementation-plan.md](ep-implementation-plan.md) AC trace convention).
- **Deferred** (13): ACs with `MANUAL ONLY` status blocks in [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — closed per code review manual table and this audit.
- **In-scope without validator trace** (12): AC-38.002, 004, 006, 011, 012, 014–020 — behaviour verified by existing suites (`make check` green); matrix links are audit mapping, not validator `Covers` refs.
- Epic gate [AC-38.022](ep-acceptance-criteria.md#ac-38-022) specifies `./bin/validate ears EP-038`, not AC↔test `validate EP-038`.
- Stage 10 gate: [ep-code-review.md](ep-code-review.md) iteration 1 — Pass.

---

## Quality gate

| Check | Result |
|-------|--------|
| `make check` | **Pass** — format, vet, lint, tests (race), govulncheck, **76.0%** statement coverage, module boundaries OK |
| `./bin/validate ears EP-038` | **Pass** — 25 reqs, 0 errors |
| `./bin/validate req EP-038` | **Pass** — 25/25 REQ↔AC coverage |
| `./bin/validate EP-038` (AC trace) | **Fail** (exit 1) — 0/12 in-scope traced; acceptable per epic design; see gaps |
| Code review (stage 10) | **Pass** — iteration 1; Blocker/Major/Medium/Minor 0 |
| System design review (stage 7) | **Pass** — iteration 2; Blocker/Major/Medium/Minor 0 |

---

## Gaps, risks, recommendations

### Gaps

- **`./bin/validate EP-038` (AC↔test):** Exits 1 — twelve in-scope ACs lack `Covers AC-38.xxx` comments. Behaviour is covered by existing tests (zero assertion churn); epic AC-38.022 uses `validate ears` instead. Non-blocking for merge given structural-refactor scope and green `make check`.

### Risks

- **Parity reliance (low):** No EP-038-specific tests; residual risk mitigated by full green suite and symmetric cut/paste diff (code review).
- **Validator CI (low):** Project-wide `validate` aggregate may flag EP-038 until ACs are marked **Deferred** for integration parity ACs or trace comments are added.

### Recommendations (non-blocking)

- Optional: add `// Covers AC-38.xxx` on representative handler tests (or mark parity ACs **Deferred** with test hints) so `./bin/validate EP-038` passes.
- After merge: set [ep-scope.md](ep-scope.md) **Status** to `DONE`.

---

## Overall verdict

**PASS** — Ready for merge on `epic/EP-038-refactor-core-handler`.
