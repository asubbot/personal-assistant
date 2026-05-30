---
artefact: ep-audit-report
epic_id: EP-036
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-036 — Audit report

**Date and time of creation:** 2026-05-30 (UTC)

**Purpose:** Stage 11 audit for [ep-implementation-plan.md](ep-implementation-plan.md) on branch `epic/EP-036-simplify-intent-tiers`.

**Pipeline reference:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md)

**Related artefacts:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-code-review.md](ep-code-review.md) · [ep-system-design-review.md](ep-system-design-review.md)

---

## Summary

**PASS.** All implementation-plan tasks (Phases 1–7) are complete. Stage 7 system design review iteration 3 and stage 10 code review iteration 2 gates are **Pass** (zero open Blocker/Major/Medium/Minor). `make check` passed with **76.0%** total statement coverage and module boundaries OK. `./bin/validate EP-036` reports **in-scope 15/15 traced (100.0% automated)**; six ACs are intentionally **deferred (manual/process)** and one **obsolete** (AC-36.019 inventory criterion, validator classification). `./bin/validate pipeline EP-036` reports no gate violations after this report. Cross-epic EP-018 artefact updates (obsolete/amended tier ACs and superseded REQs) are acknowledged as legitimate traceability hygiene. EP-036 is ready for merge.

---

## Implementation vs plan

| Task | Status | Notes |
|------|--------|-------|
| **Phase 1** (1.1–1.5) Config rejection | Done | `rejectRemovedIntentClassifierKeys`; testdata fixtures; enabled-heuristic positive load; checkpoint `make check` green. |
| **Phase 2** (2.1–2.5) `internal/intent` | Done | `model.go` / `model_test.go` deleted; two tiers; heuristic-only cascade; tests migrated. |
| **Phase 3** (3.1) `cmd/pa` wiring | Done | No classification LLM; `NewHeuristicClassifier` / `NewCascadeClassifier` 3-arg / 2-arg. |
| **Phase 4** (4.1–4.6) `internal/core` | Done | `full_lite` dispatch removed; EP-017/018 tests migrated; former `full_lite` → `full` integration test. |
| **Phase 5** (5.1–5.4) Config shrink + operator config | Done | Structs shrunk; `.config/config.json` updated (manual verify); examples/testdata clean. |
| **Phase 6** (6.1–6.6) Docs + doc-content tests | Done | `configuration.md`, `llm-provider-roles-and-logging.md`, `architecture-ru.md`; atomic doc/test commit. |
| **Phase 7** (7.1–7.4) Quality gates + manual AC closure | Done | `make check`, `./bin/validate ears EP-036`, residual-symbol grep, manual checklist signed in plan. |

Reference: [ep-implementation-plan.md](ep-implementation-plan.md)

**Delivered change set (`git diff --stat main...HEAD`):** 41 files changed, 2387 insertions(+), 875 deletions(−) (includes new epic artefacts and diagrams; product code net reduction includes deletion of `internal/intent/model.go` / `model_test.go` and removal of three-tier / model-stage paths).

---

## Test results and coverage

| Command | Result | Notes |
|---------|--------|-------|
| `make check` | **Pass** (exit 0) | fmt, vet, golangci-lint, govulncheck, race tests, coverage, **module boundaries OK** |
| `./bin/validate EP-036` | **Pass** (exit 0) | in-scope **15/15** traced, **100.0%** automated; deferred 6, obsolete 1, total ACs 22 |
| `./bin/validate EP-036 --json` | **Pass** (exit 0) | `traceability_ratio: 1`, `automated_ratio: 1` for in-scope ACs |
| `./bin/validate EP-018` | **Pass** (exit 0) | Cross-epic traceability after EP-036 amendments (5 obsolete ACs; amended full-tier ACs) |
| `./bin/validate pipeline EP-036` | **Pass** (exit 0) | Stages 3–10 present with gates pass; stage 11 report present after this artefact |

**Total statement coverage:** `total: (statements) 76.0%`

**EP-036–relevant packages (from `make check` coverage output):** `internal/intent`, `internal/config`, `internal/core`, `cmd/pa` — tier/cascade/config rejection and handler integration tests carry `// Covers AC-36.xxx` traces.

---

## REQ/AC test coverage matrix

| AC | REQ | Unit | Integration | E2E | Manual | Link |
|----|-----|------|-------------|-----|--------|------|
| [AC-36.001](ep-acceptance-criteria.md#ac-36-001) | [REQ-36.001](ep-requirements.md#req-36-001--two-complexity-tiers) | ✓ | — | — | — | `internal/intent/cascade_test.go::TestTierConstants_twoTiersOnly` |
| [AC-36.002](ep-acceptance-criteria.md#ac-36-002) | [REQ-36.002](ep-requirements.md#req-36-002--remove-full_lite-tier) | ✓ | — | — | — | `internal/intent/cascade_test.go::TestTierConstants_twoTiersOnly` |
| [AC-36.003](ep-acceptance-criteria.md#ac-36-003) | [REQ-36.003](ep-requirements.md#req-36-003--one-tier-per-turn-when-enabled) | — | ✓ | — | — | `internal/core/handler_ep017_test.go::TestHandleMessage_SimpleTier_NoToolsNoRAG` |
| [AC-36.004](ep-acceptance-criteria.md#ac-36-004) | [REQ-36.004](ep-requirements.md#req-36-004--heuristic-evaluation-order) | ✓ | — | — | — | `internal/intent/heuristic_test.go` (GreetingSimple, ToolIntentFull, AmbiguousMessage) |
| [AC-36.005](ep-acceptance-criteria.md#ac-36-005) | [REQ-36.005](ep-requirements.md#req-36-005--no-full_lite-patterns-in-heuristic) | ✓ | — | — | — | `internal/intent/heuristic_test.go::TestHeuristic_fullPatternsOnly_noFullLiteStep` |
| [AC-36.006](ep-acceptance-criteria.md#ac-36-006) | [REQ-36.006](ep-requirements.md#req-36-006--ambiguous-defaults-to-full), [REQ-36.023](ep-requirements.md#req-36-023--classification-and-config-load-tests) | ✓ | — | — | — | `internal/intent/cascade_test.go` (AmbiguousDefaultsToFull, ModelDisabled, BothNil) |
| [AC-36.007](ep-acceptance-criteria.md#ac-36-007) | [REQ-36.007](ep-requirements.md#req-36-007--confident-heuristic-stage-label), [REQ-36.011](ep-requirements.md#req-36-011--stage-values-heuristic-or-default) | ✓ | — | — | — | `internal/intent/cascade_test.go::TestCascade_HeuristicConfident`; `observability_test.go::TestCascadeClassifier_ResultContainsStageAndLen` |
| [AC-36.008](ep-acceptance-criteria.md#ac-36-008) | [REQ-36.008](ep-requirements.md#req-36-008--delete-model-stage-code), [REQ-36.009](ep-requirements.md#req-36-009--remove-modelclassifier-type) | — | — | — | ✓ | MANUAL ONLY — `model.go` / `model_test.go` absent; grep `ModelClassifier` zero (plan §7.3–7.4) |
| [AC-36.009](ep-acceptance-criteria.md#ac-36-009) | [REQ-36.010](ep-requirements.md#req-36-010--no-classification-llm-wiring) | — | — | — | ✓ | MANUAL ONLY — `cmd/pa` grep: no classification LLM / `ModelClassifier` (plan §3.1, §7.4) |
| [AC-36.010](ep-acceptance-criteria.md#ac-36-010) | [REQ-36.012](ep-requirements.md#req-36-012--dispatch-simple-and-full-only), [REQ-36.013](ep-requirements.md#req-36-013--remove-full_lite-prompt-builder) | ✓ | — | — | — | `internal/core/handler_tier_main_prompt_test.go` (3 cases) |
| [AC-36.011](ep-acceptance-criteria.md#ac-36-011) | [REQ-36.014](ep-requirements.md#req-36-014--parity-for-simple-and-full-assembly) | — | ✓ | — | — | `internal/core/handler_ep017_test.go` (SimpleTier, FullTier) |
| [AC-36.012](ep-acceptance-criteria.md#ac-36-012) | [REQ-36.015](ep-requirements.md#req-36-015--former-full_lite-uses-full-path) | — | ✓ | — | — | `internal/core/handler_ep018_test.go::TestHandleMessage_formerFullLitePattern_usesFullAssemblyWithRAG` |
| [AC-36.013](ep-acceptance-criteria.md#ac-36-013) | [REQ-36.016](ep-requirements.md#req-36-016--reject-model_stage-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests) | ✓ | — | — | — | `internal/config/intent_classifier_test.go::TestLoad_RejectRemovedIntentClassifier_model_stage` |
| [AC-36.014](ep-acceptance-criteria.md#ac-36-014) | [REQ-36.017](ep-requirements.md#req-36-017--reject-full_lite_patterns-config-key), [REQ-36.024](ep-requirements.md#req-36-024--reject-removed-keys-in-tests) | ✓ | — | — | — | `internal/config/intent_classifier_test.go::TestLoad_RejectRemovedIntentClassifier_full_lite_patterns` |
| [AC-36.015](ep-acceptance-criteria.md#ac-36-015) | [REQ-36.018](ep-requirements.md#req-36-018--enabled-heuristic-schema), [REQ-36.019](ep-requirements.md#req-36-019--validate-heuristic-at-load) | ✓ | — | — | — | `internal/config/intent_classifier_test.go` (enabled heuristic, invalid regex, max_simple_len) |
| [AC-36.016](ep-acceptance-criteria.md#ac-36-016) | [REQ-36.020](ep-requirements.md#req-36-020--keep-intent_classifier-root-key), [REQ-36.021](ep-requirements.md#req-36-021--null-intent_classifier-disables-classification) | ✓ | — | — | — | `internal/config/intent_classifier_test.go` (enabled + null root) |
| [AC-36.017](ep-acceptance-criteria.md#ac-36-017) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | ✓ | — | — | ✓ | Unit: `handler_ep018_coverage_test.go::TestEP018_configurationDoc_containsTierMatrix`; `cmd/pa/ep024_operator_logging_test.go::TestEP024_ProviderRolesDocContent`. Manual: operator doc read (plan §6.1–6.3, §7.4) |
| [AC-36.018](ep-acceptance-criteria.md#ac-36-018) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | — | — | — | ✓ | MANUAL ONLY — live `.config/config.json` load / startup (plan §5.3) |
| [AC-36.019](ep-acceptance-criteria.md#ac-36-019) | [REQ-36.025](ep-requirements.md#req-36-025--retire-obsolete-tier-tests) | — | — | — | ✓ | **Obsolete** (validator) / satisfied by test refresh + inventory (plan §7.4); no dedicated automated trace |
| [AC-36.020](ep-acceptance-criteria.md#ac-36-020) | [REQ-36.026](ep-requirements.md#req-36-026--make-check-passes) | — | — | — | ✓ | MANUAL ONLY — `make check` exit 0 (this audit run) |
| [AC-36.021](ep-acceptance-criteria.md#ac-36-021) | [REQ-36.027](ep-requirements.md#req-36-027--epic-validation-passes) | ✓ | — | — | ✓ | `internal/core/handler_ep036_test.go::TestEP036_validateCommandExitZero`; MANUAL `./bin/validate ears EP-036` (plan §7.2) |
| [AC-36.022](ep-acceptance-criteria.md#ac-36-022) | [REQ-36.022](ep-requirements.md#req-36-022--update-configs-and-operator-docs) | ✓ | — | — | — | `internal/config/intent_classifier_test.go` (example + testdata, no removed keys) |

### Notes

- Primary mapping source: `./bin/validate EP-036 --json` (audit run 2026-05-30).
- **In-scope** (15 ACs): all have automated test traces per validator. **Deferred** (6): MANUAL ONLY process/inspection gates (AC-36.008, 009, 017, 018, 020, 021) — closed per implementation plan §7.4 and this audit’s `make check` / validate runs.
- **Obsolete** (1): AC-36.019 — validator marks obsolete (index summary contains “Obsolete”); criterion satisfied by Phases 2–5 test rewrites and §7.4 inventory.
- **Unit / Integration** per [strategy.md](../../strategy.md) §2: handler tests with mocked LLM/Telegram count as integration in the matrix when they exercise end-to-end tier dispatch per turn.
- Stage 10 gate: [ep-code-review.md](ep-code-review.md) iteration 2 — Pass.

---

## Quality gate

| Check | Result |
|-------|--------|
| `make check` | **Pass** — format, vet, lint, tests (race), govulncheck, **76.0%** statement coverage, module boundaries OK |
| `./bin/validate EP-036` | **Pass** — in-scope 15/15, 100.0% automated traceability |
| `./bin/validate EP-018` | **Pass** — cross-epic amendments consistent |
| Code review (stage 10) | **Pass** — iteration 2; Blocker/Major/Medium/Minor 0 |
| System design review (stage 7) | **Pass** — iteration 3; Blocker/Major/Medium/Minor 0 |

---

## Gaps, risks, recommendations

### Gaps

None for in-scope automated ACs. Six deferred MANUAL ONLY ACs and AC-36.019 (obsolete/inventory) are closed by plan checkpoints and audit verification (`make check`, validate, grep, code review sign-off).

### Risks

- **Cross-epic EP-018 edits (low, accepted):** Branch modifies `ai-sdlc-artefacts/epics/EP-018/ep-acceptance-criteria.md` and `ep-requirements.md` (obsolete model-stage / `full_lite` ACs; amended full-tier ACs; superseded REQ notes). Legitimate traceability hygiene; operator and stage-10 reviewer acknowledged. `./bin/validate EP-018` passes.
- **Live operator config (low):** `.config/config.json` is not loaded by automated tests (AC-36.018); verified manually per plan.
- **Stale literals (negligible):** Pre-existing `"full_lite"` strings in `internal/llm/openai_test.go` and `internal/telegram/outbound_chunk_test.go` (not intent tier); code review Nit — optional cleanup.

### Recommendations (non-blocking)

- **Nit (code review):** Optional helper extraction in `internal/config/intent_classifier_test.go` where one `Test*` invokes another for a second AC trace.
- **Nit:** Optional rename/remove unrelated `full_lite` literals in llm/telegram test fixtures.
- **Suggestion:** Optional polish of EP-018 REQ wording for amended tier criteria (stage 10 carried suggestion).
- After merge: set [ep-scope.md](ep-scope.md) **Status** to `DONE`.

### Cross-epic change record

EP-036 delivery includes updates to **EP-018** artefacts: AC-18.004/009/010/011/020 **Obsolete**; AC-18.005/006/007/008/017 **amended** to full tier; REQ-18.004/009/010/011 **superseded** annotations. Recorded here for audit traceability; no product behaviour regression in EP-018 in-scope ACs (`./bin/validate EP-018` exit 0).

---

## Overall verdict

**PASS** — Ready for merge on `epic/EP-036-simplify-intent-tiers`.
