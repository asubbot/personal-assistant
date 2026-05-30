---
artefact: ep-audit-report
epic_id: EP-037
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-037 — Audit report

**Date and time of creation:** 2026-05-31 (UTC)

**Purpose:** Stage 11 audit for [ep-implementation-plan.md](ep-implementation-plan.md) on branch `epic/EP-037-consolidate-tool-selection`.

**Pipeline reference:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)

**Related artefacts:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-code-review.md](ep-code-review.md) · [ep-system-design-review.md](ep-system-design-review.md)

---

## Summary

**PASS.** All implementation-plan tasks (Phases 1–6) are complete. Stage 7 system design review iteration 2 and stage 10 code review iteration 2 gates are **Pass** (zero open Blocker/Major/Medium/Minor). `make check` passed with **76.0%** total statement coverage and module boundaries OK. `./bin/validate EP-037` reports **in-scope 15/15 traced (100.0% automated)**; eight ACs are intentionally **deferred (manual/process)** per acceptance-criteria annotations. `./bin/validate ears EP-037` reports **0 errors** (19 style warnings). `./bin/validate pipeline EP-037` reports no gate violations after this report. EP-037 is ready for merge.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| **Phase 1** (1.1–1.3) `ToolsSelection` schema + validators | Done | `ToolsSelection` struct; split `validateToolsSelectionBounds` (pre-catalog) and `validateToolsSelectionAlwaysIncludeFloor` (post-catalog); `tools_selection_test.go` with `// Covers AC-37.00x`. |
| **Phase 2** (2.1–2.3) Legacy rejection + `tools` whitelist | Done | Root `tool_pre_selection` and `tools.dynamic_selection` rejected; `allowedToolsKeys` includes `tool_output_artifacts`; negative testdata fixtures. |
| **Phase 3** (3.1–3.4) `internal/core` wiring | Done | Handler reads `tools.selection` only; integration export wired; core tests updated. |
| **Phase 4** (4.1) Remove legacy types | Done | `ToolPreSelection` / `ToolDynamicSelection` removed; grep clean for legacy symbols in product Go. |
| **Phase 5** (5.1–5.4) Config migration | Done | Operator `.config/config.json`, examples, 62× testdata fixtures, integration helpers migrated; `TestLoad_AllFixturesLoad` green. |
| **Phase 6** (6.1–6.3) Parity, docs, gates | Done | Golden parity tests in `tools_selection_parity_test.go`; `docs/configuration.md` migration table; `make check` and validate green. |

Reference: [ep-implementation-plan.md](ep-implementation-plan.md)

**Delivered change set (`git diff --stat main...HEAD`):** 98 files changed, 3220 insertions(+), 561 deletions(−) (includes epic artefacts, 62 testdata migrations, `tools_selection_parity_test.go`, and `docs/configuration.md`).

---

## Test results and coverage

| Command | Result | Notes |
|---------|--------|-------|
| `make check` | **Pass** (exit 0) | fmt, vet, golangci-lint, govulncheck, race tests, coverage, **module boundaries OK** |
| `./bin/validate EP-037` | **Pass** (exit 0) | in-scope **15/15** traced, **100.0%** automated; deferred 8, total ACs 23 |
| `./bin/validate EP-037 --json` | **Pass** (exit 0) | `traceability_ratio: 1`, `automated_ratio: 1` for in-scope ACs |
| `./bin/validate ears EP-037` | **Pass** (exit 0) | 24 requirements, **0 errors**, 19 EARS style warnings |
| `./bin/validate pipeline EP-037` | **Pass** (exit 0) | Stages 3–10 present with gates pass; stage 11 report present after this artefact |

**Total statement coverage:** `total: (statements) 76.0%`

**EP-037–relevant packages (from `make check` coverage output):** `internal/config` (8.6%), `internal/core` (17.6%), `cmd/pa` (16.2%), `tests/integration` (21.9%) — selection validation, parity tables, and fixture load tests carry `// Covers AC-37.xxx` traces.

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | Manual | Status |
|----|-----|------|-------------|--------|--------|
| AC-37.001 | REQ-37.001 | ✓ | — | — | `TestLoad_ToolsSelectionRequired` |
| AC-37.002 | REQ-37.002 | ✓ | — | — | Bounds validation tests |
| AC-37.003 | REQ-37.003 | ✓ | — | — | Enabled + always_include floor |
| AC-37.004 | REQ-37.004 | ✓ | — | — | Disabled + max 0 |
| AC-37.005 | REQ-37.005 | ✓ | — | — | `TestLoad_ToolPreSelectionRejected` |
| AC-37.006 | REQ-37.006 | ✓ | — | — | `TestLoad_ToolsDynamicSelectionRejected` |
| AC-37.007 | REQ-37.007 | ✓ | — | — | `TestConfigRootJSONKeys_ExcludesToolPreSelection` |
| AC-37.008 | REQ-37.008 | ✓ | — | — | `TestLoad_ToolsUnknownNestedKey` |
| AC-37.009 | REQ-37.009 | ✓ | — | — | `TestToolsSelectionParity_runtimeTopKCap` |
| AC-37.010 | REQ-37.010 | ✓ | — | — | `TestToolsSelectionParity_mergeEquivalentPreSelection` |
| AC-37.011 | REQ-37.011 | ✓ | — | — | `TestToolsSelectionParity_capEquivalentDynamicSelection` |
| AC-37.012 | REQ-37.012 | ✓ | — | — | `TestRun_wiresToolsSelectionFromConfig` |
| AC-37.013 | REQ-37.013 | ✓ | — | — | `TestLoad_AllFixturesLoad` |
| AC-37.014 | REQ-37.014 | — | — | ✓ | MANUAL — `docs/configuration.md` documents `tools.selection` and migration table |
| AC-37.015 | REQ-37.015 | — | — | ✓ | MANUAL — doc states `runtime_skills.tool_vector_top_k_cap` interaction |
| AC-37.016 | REQ-37.018 | — | — | ✓ | MANUAL — `make check` exit 0 (this audit run, 76.0% coverage) |
| AC-37.017 | REQ-37.019 | — | — | ✓ | MANUAL — `./bin/validate ears EP-037` exit 0 |
| AC-37.018 | REQ-37.020 | ✓ | — | — | `TestLoad_UnknownTopLevelKey_ReturnsError` |
| AC-37.019 | REQ-37.021 | ✓ | — | — | `TestToolsSelectionParity_toolVectorTopKCapOnlyUnderRuntimeSkills` |
| AC-37.020 | REQ-37.022 | — | — | ✓ | MANUAL — no `vector_search_tools` schema DRY (stage 10 + plan scope) |
| AC-37.021 | REQ-37.023 | — | — | ✓ | MANUAL — core diff wiring-only (stage 10 re-confirmed) |
| AC-37.022 | REQ-37.024 | — | — | ✓ | MANUAL — no new selection features (stage 10 re-confirmed) |
| AC-37.023 | REQ-37.013 | — | — | ✓ | MANUAL — `.config/config.json` has `tools.selection`, no legacy keys |

### Notes

- Primary mapping source: `./bin/validate EP-037 --json` (audit run 2026-05-31).
- **In-scope** (15 ACs): all have automated test traces per validator. **Deferred** (8): MANUAL ONLY process/inspection gates — closed per implementation plan §6.3, code review iteration 2, and this audit’s `make check` / validate / doc inspection runs.
- Stage 7 gate: [ep-system-design-review.md](ep-system-design-review.md) iteration 2 — Pass.
- Stage 10 gate: [ep-code-review.md](ep-code-review.md) iteration 2 — Pass (F-001/F-002/F-003 resolved; one non-blocking suggestion on `tool_output_artifacts`).

---

## Quality gate

| Check | Result |
|-------|--------|
| `make check` | **Pass** — format, vet, lint, tests (race), govulncheck, **76.0%** statement coverage, module boundaries OK |
| `./bin/validate EP-037` | **Pass** — in-scope 15/15, 100.0% automated traceability |
| `./bin/validate ears EP-037` | **Pass** — 0 EARS format errors |
| Code review (stage 10) | **Pass** — iteration 2; Blocker/Major/Medium/Minor 0 |
| System design review (stage 7) | **Pass** — iteration 2; Blocker/Major/Medium/Minor 0 |

---

## Gaps, risks, recommendations

### Gaps

None for in-scope automated ACs. Eight deferred MANUAL ONLY ACs are closed by plan checkpoints, stage 10 sign-off, documentation inspection, operator config review, and this audit’s verification commands.

### Risks

- **Live operator config (low):** `.config/config.json` is not loaded by automated unit tests (AC-37.023); verified manually — `tools.selection` present with five fields, no `tool_pre_selection` or `tools.dynamic_selection`.
- **Untyped `tool_output_artifacts` (low, accepted):** Parsed-and-ignored in EP-037 whitelist per design; stage 10 suggestion F-004 (non-blocking).
- **Residual legacy strings (negligible):** Only in rejection test fixtures (`tool_pre_selection_rejected.json`, `tools_dynamic_selection_rejected.json`) and migration prose in `docs/configuration.md`; product Go/JSON grep clean.

### Recommendations (non-blocking)

- **Suggestion (code review):** Consider typing `tools.tool_output_artifacts` in a follow-up epic if operators rely on load-time validation for that block.
- After merge: set [ep-scope.md](ep-scope.md) **Status** to `DONE`.

---

## Overall verdict

**PASS** — Ready for merge on `epic/EP-037-consolidate-tool-selection`.
