---
artefact: ep-code-review
epic_id: EP-035
status: draft
source_of_truth: true
gate: pass
latest_iteration: 2
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
non_blocking_counts:
  nit: 0
  suggestion: 0
next_action: proceed_to_stage_11
updated_at: 2026-05-30
---

# Code review — EP-035 Consolidate small internal packages

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 0 | Suggestion 0
Open findings: none
Resolved this iteration:
- F-001 (Medium) RESOLVED — anchor independently confirmed load-bearing; iteration-1 finding was factually incorrect.
- F-002 (Minor) RESOLVED — AC index "Test level" reconciled with MANUAL ONLY status.
- F-003 (Minor) RESOLVED — wrap helpers now have golden exact-equality + empty-inner assertions.
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-035-consolidate-small-packages` vs `main` (`git diff main...HEAD`); fresh delegated reviewer, readonly on product code.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 2 | Nit: 1 | Suggestion: 1
**Gate:** Fail

### Summary

Request changes (minor). The high-risk security invariants are fully preserved: `TrustPolicy` and all six `<<<PA_BEGIN_*>>>`/`<<<PA_END_*>>>` marker constants are **byte-identical** to `main`, and the importer rewrites are pure package-path renames with no logic change. No behavioural change is observable. The only substantive issue is the newly-added `tests/integration/anchor.go`, which iteration 1 judged unnecessary. The remaining items are documentation/test-rigor polish. No Blockers or Majors.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| **Medium** | `tests/integration/anchor.go` | Iteration-1 claim: file unnecessary, comment misleading (premise later proven incorrect — see iteration 2). | Remove anchor.go (later rejected). |
| **Minor** | `ep-acceptance-criteria.md` index vs status | AC index "Test level" for AC-35.002/013/014/016 listed Integration while status said MANUAL ONLY. | Reconcile labels. |
| **Minor** | `internal/prompt/wrap_test.go` | Wrap-helper tests asserted only marker presence, not exact equality required by AC-35.011. | Add golden exact-equality assertions incl. empty-inner. |
| **Nit** | working tree leftover dirs | Empty dirs may remain locally; not tracked by Git. Informational. | Optional `rmdir`. |
| **Suggestion** | `internal/prompt` | Clean merge; good non-tautological golden literals. | Optional. |

### Focus-area conclusions

1. **Security invariants — PASS** (TrustPolicy + six markers byte-identical to `main`).
2. **No behavioural change — PASS** (pure renames).
3. **anchor.go — flagged** (revisited and rejected in iteration 2).
4. **AC coverage integrity — OK**, no hidden test gaps.
5. **AC-22.010 — PASS** (relocated test retains trace).
6. **Import hygiene / boundaries — PASS** (zero dead references).

---

## Review iteration 2

**Review date:** 2026-05-30
**Stage 10 iteration:** 2 of max 5
**Scope:** Branch `epic/EP-035-consolidate-small-packages` vs `main` (`git diff main...HEAD`); fresh delegated reviewer, readonly on product code. Re-verification of iteration-1 findings F-001/F-002/F-003 and all carry-over invariants.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 0
**Gate:** Pass

### Summary

Approve. All three iteration-1 findings are resolved. The reviewer independently re-verified F-001's rejection and agrees it is correct: `tests/integration/anchor.go` is genuinely load-bearing, not dead weight, and its rewritten comment is accurate. F-002 and F-003 are fixed and now satisfy their ACs. All carry-over invariants still hold. No new findings. Gate flips to pass; proceed to stage 11.

### Verification run (readonly)

| Command / check | Result |
|-----------------|--------|
| `go vet -tags=integration ./tests/integration/...` (anchor present) | exit 0 |
| **F-001 isolated reproduction** — minimal module with `doc.go` (untagged), `config_helpers.go` + `concurrent_write_test.go` (both `//go:build integration`, `package integration_test`), **no anchor** | **fails**: `found packages integration (concurrent_write_test.go) and integration_test (config_helpers.go)` |
| Same module **with** `anchor.go` (sorts first) | exit 0 |
| Same module, anchor renamed `zzz_anchor.go` (sorts *after* `concurrent`) | **fails** again (proves it is sort-order, not filename magic) |
| Same module, file renamed `aaa.go` (any non-test file sorting before `concurrent`) | exit 0 (confirms mechanism) |
| `TrustPolicy` const literal: `git show main:internal/systemprompt/systemprompt.go` vs `HEAD:internal/prompt/wrap.go` | **byte-identical** |
| Six marker constant values: `git show main:internal/promptmarkers/markers.go` vs `HEAD:internal/prompt/markers.go` | **byte-identical** (only package name + doc comment differ) |
| Importer hunks (`handler.go`, `handler_test.go`, `system_tail.go`, `runtimeskills/package.go`, `tools/write_memory.go`) | pure `promptmarkers`/`systemprompt` → `prompt` renames; identical call sites |
| `go test ./internal/prompt/...` | `ok` (golden exact-equality + empty-inner tests pass) |
| `go test -race -tags=integration -run TestConcurrentWrites_NoBusyErrors ./tests/integration/` | `ok 1.400s` |
| Dead-reference grep `internal/(promptmarkers|systemprompt)` and `pa/internal/logging` in `*.go` | zero matches |
| `git diff main...HEAD -- config.json .config/config.json internal/config` | empty (config untouched) |

### Finding resolutions

| ID | Severity (iter 1) | Status | Evidence |
|----|-------------------|--------|----------|
| F-001 | Medium | **RESOLVED (finding was incorrect)** | The iteration-1 claim that `config_helpers.go` is untagged is false — it carries `//go:build integration`. With `-tags=integration`, the loader scans files in sorted order; `concurrent_write_test.go` (XTest file, `package integration_test`) sorts before `config_helpers.go`/`doc.go`. Go's `go/build` strips the `_test` suffix and sets the directory package name to `integration`, then collides with `config_helpers.go`'s `integration_test`. Reproduced in an isolated module: removing the anchor (or renaming it to sort after `concurrent`) re-triggers the error; any non-test `integration_test` file sorting first resolves it. `anchor.go` is load-bearing and its comment is accurate. |
| F-002 | Minor | **RESOLVED** | `ep-acceptance-criteria.md` index now lists AC-35.002/013/014 as `Manual (build/grep)` and AC-35.016 as `Manual (make check)`, consistent with each AC's MANUAL ONLY status line. |
| F-003 | Minor | **RESOLVED** | `internal/prompt/wrap_test.go` now asserts exact byte-equality golden strings for `WrapRetrievedContext`, `WrapToolInstructions`, and `WrapRuntimeSkills`, plus `TestWrapHelpers_emptyInner` covering `""`/whitespace-only inner → `""`. Fully satisfies AC-35.011. |

### Carry-over re-confirmation

1. **Security invariants — PASS.** `TrustPolicy` and all six marker constants byte-identical to `main`.
2. **No behavioural change — PASS.** Every importer edit is a pure path rename; call sites unchanged.
3. **Dead references — PASS.** Zero `*.go` imports of `internal/promptmarkers`, `internal/systemprompt`, or `pa/internal/logging`.
4. **Config — PASS.** Untouched.
5. **Relocated test — PASS.** Passes under `-race -tags=integration`; retains `// Covers AC-22.010`.

### Gate decision

All Blocker/Major/Medium/Minor open counts are **0**. The §2.2 9↔10 loop is complete. Approve and proceed to stage 11 (audit).
