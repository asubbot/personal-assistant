---
artefact: ep-code-review
epic_id: EP-036
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
  nit: 2
  suggestion: 2
next_action: proceed_to_stage_11
updated_at: 2026-05-30
---

# Code review — EP-036 (Simplify intent classification)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 2 | Suggestion 2
Open findings: none (all blocking findings F-001/F-002/F-003 resolved in iteration 2)
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-036-simplify-intent-tiers` vs `main` — `git diff main...HEAD`. Readonly review.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 2 | Nit: 2 | Suggestion: 1
**Gate:** Fail

### Summary

Request changes (non-blocking on correctness). Product change is clean and faithful: model stage removed (`internal/intent/model.go`/`model_test.go` deleted), cascade = heuristic → default `full`, `TierFullLite`/`full_lite` gone from production code, config structs shrunk, `load.go` fails fast on removed keys. Build, targeted tests, and both validators pass. No Blocker/Major. Blocking issues are traceability evidence integrity, not behaviour.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| **Medium** | `ep-acceptance-criteria.md` AC-36.009 / AC-36.018 status lines | MANUAL ONLY sentences name other Unit AC codes (AC-36.006; AC-36.022/013/014) on the same line → validator marks AC-36.006/013/014/022 DEFERRED despite real tests. | Reword so referenced AC codes are not on the MANUAL ONLY line; re-run validate. |
| **Minor** | EP-018 `ep-requirements.md` (REQ-18.004/009/010/011) | ACs marked Obsolete but REQs not annotated superseded. Asymmetric traceability. | Add `Superseded by EP-036` notes. |
| **Minor** | EP-018 `ep-acceptance-criteria.md` AC-18.005/006/007/008/017; `handler_ep018_coverage_test.go` | ACs still describe removed `full_lite` while tests repointed to `full` (lingering `fullLite` test names). | Reword to `full` tier; rename tests. |
| **Nit** | `internal/llm/openai_test.go`; `internal/telegram/outbound_chunk_test.go` | Pre-existing literal `"full_lite"` (not intent tier). | Optional. |
| **Nit** | `internal/config/intent_classifier_test.go` | `Test*` calls another `Test*` for a second trace. | Optional helper extraction. |
| **Suggestion** | Cross-epic EP-018 edit | Legitimate traceability hygiene; no objection. | Record operator decision (done in chat). |

### Focus-area conclusions

1. Config strictness / explicit-JSON: PASS. 2. Behavioural safety: PASS. 3. Complete removal: PASS. 4. Cross-epic EP-018 edit: legitimate. 5. Test integrity: PASS except F-001 under-enforcement. 6. KISS / boundaries: PASS.

---

## Review iteration 2

**Review date:** 2026-05-30
**Stage 10 iteration:** 2 of max 5
**Scope:** Branch `epic/EP-036-simplify-intent-tiers` vs `main` — full `git diff main...HEAD`, focused re-review of fix commit `1bc3e93` (`git diff 72f72f7..HEAD`). Fresh readonly reviewer. Verified resolution of F-001/F-002/F-003.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 2 | Suggestion: 2
**Gate:** Pass — all blocking severities 0 (§2.2 exit satisfied).

### Summary

Approve. The iteration-2 commit is documentation/test-naming only (EP-036 + EP-018 artefacts and three EP-018 test-function renames); it touches no product code under `cmd/`/`internal/` (non-test). All three blocking findings resolved and verified by re-running validators and the build/test suite. Iteration-1's behavioural PASS conclusions remain valid. No new regressions.

### Verification performed (readonly)

- `git diff 72f72f7..HEAD`: only `EP-018/ep-acceptance-criteria.md`, `EP-018/ep-requirements.md`, `EP-036/ep-acceptance-criteria.md`, `internal/core/handler_ep018_coverage_test.go` (3 test renames). No product (non-test) code changed.
- `go build ./...` → success. `go test ./internal/core/... ./internal/config/... ./internal/intent/... ./cmd/pa/...` → all `ok`.
- `./bin/validate EP-036` → exit 0. **in-scope 15/15 traced (100%), automated 15 (100%), manual-only 0 | deferred 6 | obsolete 1 | total 22.** AC-36.006/013/014/022 now `✓` automated (were DEFERRED in iteration 1). F-001 effect confirmed.
- `./bin/validate EP-018` → exit 0. in-scope 16/16 traced (100%), automated 15, manual-only 1, deferred 0, obsolete 5, total 21. AC-18.005/006/007/008/017 `✓`.
- grep: no `fullLite`/`full_lite` remain in `handler_ep018_*` tests; `// Covers AC-18.xxx` traces intact.

### Resolution verification

| Finding | Severity | Status | Evidence |
|---------|----------|--------|----------|
| F-001 | Medium | **Resolved** | AC-36.009/36.018 status lines reworded; cross-refs moved to `Related coverage:` lines with no AC tokens. validate EP-036 counts AC-36.006/013/014/022 as automated (15/15, 100%). |
| F-002 | Minor | **Resolved** | REQ-18.004/009/010/011 annotated `Superseded by EP-036` in table + detailed entries; retained for historical traceability. |
| F-003 | Minor | **Resolved** | AC-18.005/006/007/008/017 reworded to `full` tier (`Amended by EP-036`); three `fullLite` test functions renamed to `TestEP018_fullTier_*`; traces intact; validate EP-018 passes. |

### Findings (open)

None blocking. Carried-over non-blocking: two Nits (stale `full_lite` literals in unrelated `llm`/`telegram` tests; test-calls-test wrapper) and two Suggestions (optional REQ-18.005–008/017 wording polish; operator record of cross-epic edit — acknowledged).

### Gate decision

All Blocker/Major/Medium/Minor open counts are 0. The §2.2 9↔10 loop is complete. Approve and proceed to stage 11 (audit).
