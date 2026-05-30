---
artefact: ep-code-review
epic_id: EP-035
status: draft
source_of_truth: true
gate: fail
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 1
  minor: 2
non_blocking_counts:
  nit: 1
  suggestion: 1
next_action: return_to_stage_9
updated_at: 2026-05-30
---

# Code review — EP-035 Consolidate small internal packages

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 1 | Minor 2
Non-blocking counts: Nit 1 | Suggestion 1
Open findings:
- F-001 Medium: `tests/integration/anchor.go` is unnecessary and documents an incorrect Go rationale (violates KISS).
- F-002 Minor: AC index "Test level" column disagrees with "MANUAL ONLY" status lines (AC-35.002/013/014/016).
- F-003 Minor: AC-35.011 wrap-helper tests assert framing/`Contains`, not exact equality to pre-EP-035 output as the AC requires.
Next action: Return to stage 9

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-035-consolidate-small-packages` vs `main` (`git diff main...HEAD`); fresh delegated reviewer, readonly on product code.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 2 | Nit: 1 | Suggestion: 1
**Gate:** Fail

### Summary

Request changes (minor). The high-risk security invariants are fully preserved: `TrustPolicy` and all six `<<<PA_BEGIN_*>>>`/`<<<PA_END_*>>>` marker constants are **byte-identical** to `main` (verified by direct diff against `git show main:internal/systemprompt/systemprompt.go` and `git show main:internal/promptmarkers/markers.go`), and the importer rewrites are pure package-path renames with no logic change. No behavioural change is observable: forbidden-marker detection, wrap helpers, prompt assembly ordering, runtime-skills load rejection, and memory-index rejection all pass their tests. The only substantive issue is the newly-added `tests/integration/anchor.go`, which is unnecessary and ships a misleading rationale. The remaining items are documentation/test-rigor polish in the AC artefact. No Blockers or Majors.

### Verification run (readonly)

| Command | Result |
|---------|--------|
| `git diff` of `TrustPolicy` const (main vs HEAD) | **byte-identical** |
| `git diff` of six marker constants (main vs HEAD) | **byte-identical** |
| `go vet -tags=integration ./tests/integration/...` | exit 0 |
| `go test -race -tags=integration -run TestConcurrentWrites_NoBusyErrors ./tests/integration/` | `ok 1.632s` (passes under `-race`) |
| `go test ./internal/prompt/... ./internal/core/... ./internal/tools/... ./internal/runtimeskills/...` | all `ok` |
| `go test -tags=integration -run TestRuntimeSkills ./tests/integration/` | `ok` |
| Dead-reference grep for removed package paths (excl. artefacts) | zero matches |
| `git diff main...HEAD -- config.json/.config/config.json/internal/config` | empty (config untouched) |

> Note: full `make check` was not run (govulncheck/lint need network + write cache in this readonly sandbox). Build, vet, and the relevant test subsets all pass; AC-35.016 should still be confirmed by `make check` in stage 9/11.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| **Medium** | `tests/integration/anchor.go:1-6` | The file is **unnecessary** and its comment is **technically incorrect**. The directory's package name is already fixed by the pre-existing untagged `doc.go` and `config_helpers.go` (`package integration_test`, no build constraint, always compiled). `main` built/tested fine **with no anchor file**. The relocated `concurrent_write_test.go` is an internal `_test.go` file that never determines the primary package name, so it cannot introduce the "first `*_test.go` … when `p.Name` is still empty" condition the comment describes. This adds dead weight and a misleading rationale, contrary to the repo KISS principle in `AGENTS.md`. | Remove `tests/integration/anchor.go`. Confirm `make check` (incl. `go vet -tags=integration`) stays green without it. |
| **Minor** | `ep-acceptance-criteria.md` index vs status lines | AC index "Test level" column lists AC-35.002/013/014/016 as **Integration**, while each AC's Status line marks it **MANUAL ONLY**. The document states two different test levels for the same AC, which will confuse the stage-11 audit. (Substantively, the deleted-package import bans *are* enforced automatically because importing a removed package fails compilation under `make check`; the labels just need reconciling.) | Reconcile the index "Test level" with the per-AC status (mark as Manual/Build-verified consistently), or add a one-line note that `make check` provides the automated enforcement. |
| **Minor** | `internal/prompt/wrap_test.go` | AC-35.011 requires each wrap helper result to **equal** the pre-EP-035 output. The tests only assert marker presence (`strings.Contains`) and non-emptiness — weaker than the AC's exact-equivalence wording. (The implementation is byte-identical to `main`'s `systemprompt.go` logic, verified by reading both, so functional risk is low; the test simply under-implements its own AC.) | Add a golden/exact-equality assertion for `WrapRetrievedContext`/`WrapToolInstructions`/`WrapRuntimeSkills` (including the empty-inner → `""` case) to fully satisfy AC-35.011. |
| **Nit** | working tree leftover dirs | After file deletion, empty `internal/logging/` etc. may remain on the local working tree. Not a committed-branch problem: Git does not track empty directories; a fresh checkout will not contain them. Informational only. | Optional `rmdir` of stale local dirs. |
| **Suggestion** | `internal/prompt` | The merge cleanly unifies markers + trust policy + wrap helpers; the independent golden literals are a good non-tautological choice. Consider folding the F-003 exact-output assertions into these existing golden tests rather than adding new files. | Optional. |

### Focus-area conclusions

1. **Security invariants — PASS.** `TrustPolicy` and all six marker constants are byte-identical to `main`. The `internal/prompt/markers.go` package doc comment was reworded — a comment-only change.
2. **No behavioural change — PASS.** Importer edits are pure `promptmarkers`/`systemprompt` → `prompt` renames. `TextContainsForbiddenMarkerLine`, `ForbiddenMarkerLines`, wrap helpers, redaction, runtime-skills load rejection, and memory-index rejection are unchanged and pass.
3. **`tests/integration/anchor.go` — SMELL (F-001).** Not minimal; the untagged `doc.go`/`config_helpers.go` already anchor the package. The relocated test runs under `-race` (`test-race` uses `-tags=integration`).
4. **AC coverage integrity — OK, no hidden test gaps.** The 9 MANUAL-ONLY ACs are genuinely structural/grep/config-diff/process-gate checks; behavioural ACs (35.007–011, 017–020) each have real automated tests. Only the cosmetic index/status drift (F-002) needs fixing.
5. **AC-22.010 handling — PASS.** The relocated test retains `// Covers AC-22.010`; EP-022 files untouched.
6. **Import hygiene / boundaries / KISS — PASS** except F-001. Zero dead references to removed packages.

### Residual risks / follow-ups

- Confirm full `make check` (lint + govulncheck + `check-boundaries`) exits zero in an environment with network/write access (AC-35.016).
- F-003 is low functional risk because the wrap implementation is byte-identical to `main`; treat it as test-completeness hardening.
