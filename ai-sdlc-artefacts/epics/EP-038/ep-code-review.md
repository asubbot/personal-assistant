---
artefact: ep-code-review
epic_id: EP-038
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
  nit: 0
  suggestion: 1
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-038 Refactor core conversation handler

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 0 | Suggestion 1
Open findings: none
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-038-refactor-core-handler` vs `main` (`git diff main...HEAD`). Product focus: `internal/core/handler.go`, `handler_llm.go`, `handler_tools.go`, `handler_memory.go` (commit `32d2117`). SDLC artefacts on branch reviewed for context only.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 1
**Gate:** Pass

### Summary

Approve for stage 11. The change set is a clean structural split of the former god `handler.go` (~663 LOC on `main`) into three focused files plus slim orchestration in `handler.go` (132 LOC). Symmetric diff analysis shows **no logic edits**—only import redistribution remains asymmetric between deleted and added hunks. No `config.json` / `internal/config` / test file diffs vs `main`. `make check` passes; EARS/REQ validate gates pass for EP-038.

### Scope reviewed

| Area | Result |
|------|--------|
| Product diff (`internal/core/`) | 4 files: slim `handler.go` + 3 new handler\_\* files |
| Config / tests | No diff vs `main` |
| Behaviour | Parity via existing suites (`make check` green) |
| Manual AC (LOC, grep) | Verified in this iteration (see below) |

### AC / structural verification (iteration 1)

| AC | Check | Result |
|----|-------|--------|
| AC-38.001 | Merge-base `f997c9b` = merge of EP-037 on `main` | Pass |
| AC-38.002 | `HandleMessage` pipeline in `handler.go` | Pass — `checkUserMessage` → `buildMainTurnMessagesPreTail` → `assembleTierMainLLMParams` → `completeAt` → `finishAfterFirstLLM` (+ usage footer) |
| AC-38.003 | `wc -l handler.go` | Pass — **132** LOC (≤ ~200) |
| AC-38.004 | Turn constants in `handler.go` | Pass — `maxToolRounds = 10`, `logTruncateMaxLen`, `maxToolResultPromptBytes`, `defaultMaxDynamicSystemRunes` |
| AC-38.005 | LLM methods in `handler_llm.go` | Pass — all listed symbols present; absent from `handler.go` |
| AC-38.006 | Router / tool-round behaviour | Pass — covered by `make check` (EP-034 regression slice included) |
| AC-38.007 | Tool methods in `handler_tools.go` | Pass |
| AC-38.008 | `mergeToolIDs`, dynamic cap, tier tail | Pass — unchanged ownership in `runtime_tools.go`, `dynamic_tool_selection.go`, `handler_tier_main_prompt.go` |
| AC-38.009 | Memory methods in `handler_memory.go` | Pass |
| AC-38.010 | `vector_merge` / `memory_vectors` / `system_tail` | Pass — no diff vs `main` |
| AC-38.011–AC-38.013 | Tier dispatch | Pass — direct `switch tier` (`TierFull` / default simple); no `TierFullLite` / strategy framework in production `internal/core` |
| AC-38.014–AC-38.016 | Public wiring | Pass — no diff in `adapter.go`, `run.go`, `integration_export.go`; tests unchanged |
| AC-38.017 | Config schema | Pass — no config paths in diff |
| AC-38.018–AC-38.020 | Parity / handler tests | Pass — `make check` |
| AC-38.021 | `make check` | Pass — run in review (exit 0) |
| AC-38.022 | `validate ears EP-038` | Pass — 25 reqs, 0 errors (26 warnings) |
| AC-38.023 | Structural-only diff | Pass — symmetric diff: 7 unique lines are import lines only |
| AC-38.024 | `conversationHandler` unexported | Pass |
| AC-38.025 | Tier file cleanup | N/A — no `handler_tier_main_prompt.go` diff vs `main` |

### What was done well

- Cut/paste discipline: moved method bodies match `main` `handler.go`; no accidental algorithm edits detected.
- `handler.go` reads as orchestration: struct, constants, session helpers, `HandleMessage`, `redactLogString`, `checkUserMessage`.
- Per-file imports trimmed appropriately (LLM / tools / memory dependencies isolated).
- Zero test assertion churn — strong signal for behaviour parity.
- Prerequisites satisfied at branch base (post EP-037 merge on `main`).

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Suggestion** | [ep-implementation-plan.md](ep-implementation-plan.md) Phase 7 (tasks 7.1–7.3) | Manual LOC/grep/`make check`/validate gates are satisfied by code and this review, but plan checkboxes remain unchecked — no explicit stage-9 sign-off block documenting manual AC completion. | Optional: tick Phase 7 tasks or add a short “manual gates verified” note when closing stage 9; does not block merge. |

No Blocker, Major, Medium, or Minor findings.

### Test / verification

- `make check` — **PASS** (exit 0): tests, vet, lint, vuln, module boundaries.
- `make build` then `./bin/validate ears EP-038` — **PASS** (0 errors, 26 EARS weak-pattern warnings).
- `./bin/validate req EP-038` — **PASS** (25/25 REQ coverage).
- Manual grep/LOC gates — executed in this review (see table above).

### Residual risks / follow-ups

- Behaviour parity relies on existing integration tests; no new EP-038-specific tests (by design). Residual risk is low given zero assertion churn and green full suite.
- Stage 9 commit exists on branch; orchestrator may still want implementation-plan checkbox hygiene (F-001) before audit sign-off.
