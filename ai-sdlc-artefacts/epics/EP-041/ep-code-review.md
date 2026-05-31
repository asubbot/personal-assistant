---
artefact: ep-code-review
epic_id: EP-041
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
non_blocking_counts:
  nit: 2
  suggestion: 2
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-041 Full-tier prompt pipeline

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 2 | Suggestion 2
Open findings: none
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** All changes on branch `epic/EP-041-full-tier-pipeline` vs `main` (7 files, +215/−26 lines): `handler_full_tier_pipeline.go` (new), `handler_tier_main_prompt.go` (delegate refactor), `ep041_traceability_test.go` (new), parity traceability comments on existing tier tests, EP-041 design/plan artefacts.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 2 | Suggestion: 2
**Gate:** Pass

### Summary

The implementation cleanly delivers the planned structural refactor: `fullTierAssembler` with five named steps in fixed order, single entry from `buildTierFullMainPrompt`, and a thin `mergeTailMergedToolsAndOptions` delegate for pre-selected skills. Logic moved from `mergeTailMergedToolsAndOptions` into the assembler preserves error handling, tail mutation, and completion-option wrapping. All four implementation-plan tasks are complete. `make check` passes (exit 0). No config schema changes. Simple tier dispatch is untouched. Approved for stage 11.

### What was done well

- Clear pipeline extraction in `handler_full_tier_pipeline.go`: step methods map 1:1 to requirements and system design (skills → merge tools → dynamic cap → tail budget → completion options).
- `fullTierStepOrder` constants plus numbered step comments satisfy REQ-41.002 and REQ-41.006; `TestEP041_fullTierStepOrderConstants` adds regression safety.
- `buildTierFullMainPrompt` is a one-line delegate to `newFullTierAssembler(...).run()` (REQ-41.003).
- `mergeTailMergedToolsAndOptions` correctly reuses `runFromSkills()` when skills are pre-selected — no duplicated tail logic.
- Behaviour parity preserved: same call sequence, same `WrapUserError` on completion-options failure, same system-message tail mutation via `buildDynamicTailString`.
- `runFromSkills()` / `stepSelectSkills()` split is a sensible design for the pre-selected-skills entry without duplicating steps 2–5.
- Traceability tests verify struct shape, delegate behaviour, and step-order constants.
- Existing tier tests tagged `// Covers AC-41.003` on representative parity paths (`handler_tier_main_prompt_test.go`, `handler_ep018_coverage_test.go`).
- `TestTierMainPromptBuilders_simpleTierUnchanged` continues to guard simple-tier behaviour (REQ-41.005 / AC-41.004 intent).
- No `//nolint:gocyclo`; KISS refactor with minimal surface-area change.

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Nit** | `ai-sdlc-artefacts/epics/EP-041/ep-system-design.md:9` | H1 title reads `# EP-040 — Full-tier prompt pipeline` while YAML front matter and epic id are EP-041. | Fix heading to `EP-041`. |
| F-002 | **Nit** | `internal/core/handler_tier_main_prompt_test.go:15` | `TestTierMainPromptBuilders_simpleTierUnchanged` covers AC-41.004 intent but lacks `// Covers AC-41.004` traceability comment (plan task 1.3 added AC-41.003 only). | Add `// Covers AC-41.004` above the test for consistency with other epic traceability markers. |
| F-003 | **Suggestion** | `ai-sdlc-artefacts/epics/EP-041/ep-acceptance-criteria.md` | Index lists AC-41.002 and AC-41.004 but detailed anchor sections are missing (only AC-41.001, AC-41.003, AC-41.005 are written out). | Add missing AC sections in a follow-up artefact pass; not blocking merge because behaviour is covered in code/tests. |
| F-004 | **Suggestion** | `internal/core/handler_full_tier_pipeline.go:52` | `runFromSkills()` purpose (pre-selected skills entry for `mergeTailMergedToolsAndOptions`) is implicit. | Add a one-line comment on `runFromSkills` explaining the alternate entry point. Optional polish. |

### Plan alignment

| Plan task | Status | Notes |
|-----------|--------|-------|
| 1.1 Create `handler_full_tier_pipeline.go` | Done | Type + five step methods present |
| 1.2 Wire `buildTierFullMainPrompt`; thin delegate | Done | Duplicate logic removed from `mergeTailMergedToolsAndOptions` |
| 1.3 Handler tier tests + AC-41.003 markers; `make check` | Done | Markers on two parity tests; check passes |
| 1.4 `ep041_traceability_test.go` | Done | Step order, type shape, delegate smoke test |

### REQ / AC coverage (spot check)

| Id | Verdict |
|----|---------|
| REQ-41.001 / AC-41.001 | Pass — `fullTierAssembler` type with handler/turn inputs; sole `buildTierFullMainPrompt` entry |
| REQ-41.002 / AC-41.002 | Pass — five steps in fixed order via `run()` / `runFromSkills()` and documented step methods |
| REQ-41.003 | Pass — `buildTierFullMainPrompt` → `newFullTierAssembler(...).run()` |
| REQ-41.004 / AC-41.003 | Pass — existing tier tests pass with AC-41.003 markers |
| REQ-41.005 / AC-41.004 | Pass — simple tier path and tests unchanged |
| REQ-41.006 | Pass — step names/comments + order constants |
| REQ-41.007 | Pass — no config files in diff |
| REQ-41.008 / AC-41.005 | Pass — `make check` exit 0 |

### Test / verification

- `make check` — **PASS** (exit 0). All packages pass including `pa/internal/core`. Race detector enabled. golangci-lint 0 issues. govulncheck clean. Module boundaries OK. Coverage 76.1%.
- New tests: `TestEP041_fullTierStepOrderConstants`, `TestEP041_mergeTailMergedToolsAndOptions_delegate`, `TestEP041_fullTierAssemblerType`.
- Parity: `TestTierMainPromptBuilders_fullNilCatalog`, `TestEP018_fullTier_dynamicDisabled_preservesMoreToolsThanWhenEnabled` (AC-41.003).

### Residual risks / follow-ups

- F-001 and F-002 are quick polish items; address at team discretion before or after merge.
- F-003 (incomplete AC artefact sections) is a documentation gap from earlier stages; does not affect runtime behaviour.
- `mergeTailMergedToolsAndOptions` remains a public handler method (required by EP-038); correctly documented as thin delegate rather than removed.
