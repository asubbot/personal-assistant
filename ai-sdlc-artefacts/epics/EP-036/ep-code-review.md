---
artefact: ep-code-review
epic_id: EP-036
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
  nit: 2
  suggestion: 1
next_action: return_to_stage_9
updated_at: 2026-05-30
---

# Code review — EP-036 (Simplify intent classification)

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 1 | Minor 2
Non-blocking counts: Nit 2 | Suggestion 1
Open findings:
- F-001 Medium: Four in-scope Unit ACs (36.006/013/014/022) are silently excluded as DEFERRED by `bin/validate` because their codes appear inside the "MANUAL ONLY" status sentences of AC-36.009/36.018; the hard AC↔test gate stops enforcing them.
- F-002 Minor: EP-018 `ep-requirements.md` REQs for `full_lite`/model-stage are not annotated as superseded by EP-036, while their ACs are marked Obsolete (forward-traceability asymmetry).
- F-003 Minor: EP-018 AC-18.005/006/007/008/017 still describe the removed `full_lite` tier in their text while their tests now exercise the `full` tier.
Next action: Return to stage 9

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-036-simplify-intent-tiers` vs `main` — `git diff main...HEAD` (product code: `cmd/`, `internal/`; docs; epic artefacts incl. cross-epic EP-018 AC edit). Readonly review.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 2 | Nit: 2 | Suggestion: 1
**Gate:** Fail — Medium=1, Minor=2 remain open (§2.2 requires Blocker=Major=Medium=Minor=0).

### Summary

Request changes (non-blocking on correctness). The product change is clean, idiomatic, and faithful to the epic: the model stage is fully removed (`internal/intent/model.go` + `model_test.go` deleted), the cascade is now heuristic → default `full`, `TierFullLite`/`full_lite` is gone from production code and tier dispatch, config structs are shrunk, and `load.go` fails fast on the removed keys with explicit, named errors. `go build ./...`, the targeted test packages, and `./bin/validate EP-036` / `EP-018` all pass. There are **no Blocker or Major** findings. The blocking issues are about traceability evidence integrity, not product behaviour.

### Verification performed (readonly)

- `go build ./...` → success.
- `go test ./internal/config/... ./internal/intent/... ./internal/core/... ./cmd/pa/...` → all `ok`.
- `./bin/validate EP-036` → exit 0 (in-scope 11/11 traced, automated 11, deferred 10, obsolete 1, total 22).
- `./bin/validate EP-018` → exit 0 (in-scope 16/16 traced, obsolete 5, total 21).
- `.config/config.json`: `intent_classifier.enabled=true`, heuristic-only, no `model_stage`/`full_lite_patterns` → AC-36.018 manual claim valid.
- `config.examples/config.example.json`: `intent_classifier: null` (top-level key retained, nullable) → explicit-JSON principle intact.
- grep `full_lite|TierFullLite|ModelStage|ModelClassifier|model_stage` in `cmd/`+`internal/`: production matches limited to config-rejection logic and tests that intentionally name rejected keys (plus two unrelated test string literals — F-005). No dead code.

### Findings

| Severity | Location | Issue | Recommendation |
|----------|----------|-------|----------------|
| **Medium** | `ep-acceptance-criteria.md` AC-36.009 and AC-36.018 status lines | The two **MANUAL ONLY** sentences name other Unit AC codes (AC-36.006; AC-36.022/013/014) on the same physical line. `bin/validate`'s exclusion heuristic marks those codes DEFERRED, so AC-36.006/013/014/022 — each `Test level: Unit` with real `// Covers` tests — are dropped from the hard AC↔test gate. Coverage evidence is misleading (a future deletion of those tests would not be caught). | Reword AC-36.009/36.018 status sentences so referenced Unit AC codes are NOT on the same line as "MANUAL ONLY" (drop the explicit tokens or move cross-refs to a separate line). Re-run `./bin/validate EP-036` and confirm 36.006/013/014/022 count as automated. |
| **Minor** | EP-018 `ep-requirements.md` (REQ-18.004/009/010/011 + full_lite FR rows) | ACs marked Obsolete with EP-036 cross-refs, but the REQs still describe full_lite/model-stage as live with no superseded note. Asymmetric forward traceability. | Add a `Superseded by EP-036` note to the affected REQ-18.xxx lines. |
| **Minor** | EP-018 `ep-acceptance-criteria.md` AC-18.005/006/007/008/017; `internal/core/handler_ep018_coverage_test.go` | These ACs still describe the removed `full_lite` tier while their tests were repointed to `full` (incl. lingering `fullLite` test names). Inconsistent with obsoleting 18.004/009/010/011/020. | Obsolete these too or reword to the surviving `full`-tier behaviour, and rename lingering `fullLite` test functions. |
| **Nit** | `internal/llm/openai_test.go`; `internal/telegram/outbound_chunk_test.go` | Pre-existing unmodified tests use `"full_lite"` as arbitrary literal content (not the intent tier). Stale sample label. | Optional: replace with `full`/`simple`. Out of strict scope. |
| **Nit** | `internal/config/intent_classifier_test.go` heuristic-only testdata test | A `Test*` calls another `Test*` to attach a second `// Covers AC-36.022` trace — slightly hacky. | Optional: extract a shared non-`Test` helper. |
| **Suggestion** | Cross-epic EP-018 edit | Assessed as **legitimate traceability hygiene**: feature deliberately removed by EP-036, REQ links retained, ACs annotated (not deleted), validator supports `Obsolete`, `validate EP-018` passes. No objection. | Record a one-line operator decision acknowledging the cross-epic obsoleting (done in chat). |

### Focus-area conclusions

1. **Config strictness / explicit-JSON:** PASS. Fail-fast on `model_stage` and `heuristic.full_lite_patterns` with named errors; strict unknown-key rejection unchanged; top-level `intent_classifier` required-but-nullable; new testdata + example load successfully.
2. **Behavioural safety:** PASS. `assembleTierMainLLMParams` only has `TierFull` + `default`(simple); `buildTierFullLiteMainPrompt` removed; `full` path unchanged; ambiguous → `full` (no LLM call).
3. **Complete removal:** PASS. No stray symbols; `model.go`/`model_test.go` deleted; no dead code.
4. **Cross-epic EP-018 edit:** Legitimate; REQ traceability retained; `validate EP-018` passes. Minor asymmetries (F-002/F-003).
5. **Test integrity:** Largely PASS; new tests carry `// Covers AC-36.xxx`; AC-36.018 genuinely MANUAL, AC-36.022 automated. F-001 wording causes validator to under-enforce 4 ACs.
6. **KISS / imports / boundaries:** PASS.

### Residual risks / follow-ups

- Stale `full_lite` literals in unrelated `llm`/`telegram` tests (harmless).
- Capture cross-epic decision as operator record (done).
