---
artefact: ep-code-review
epic_id: EP-037
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
  suggestion: 1
next_action: proceed_to_stage_11
updated_at: 2026-05-30
review_note: "Stage 10 iteration 2 performed by orchestrator (Opus 4.8 API limit unavailable); fixes verified independently via tests and code inspection."
---

# Code review — EP-037 Consolidate tool pre-selection configuration

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 0 | Suggestion 1
Open findings: none (F-001/F-002/F-003 resolved in iteration 2)
Resolved this iteration:
- F-001 (Major) RESOLVED — fixture added; assertion matches production rejection message; AC-37.006 genuinely exercised.
- F-002 (Medium) RESOLVED — parity tests anchor on golden tool-id slices.
- F-003 (Nit) RESOLVED — toolsSelection field comment updated for EP-037.
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-037-consolidate-tool-selection` vs `main` (HEAD = a22465b). Readonly review (Opus 4.8).
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 1 | Minor: 0 | Nit: 1 | Suggestion: 1
**Gate:** Fail

### Summary

Request changes. Product change clean; test integrity defects in AC-37.006 rejection test (vacuous pass) and tautological parity tests blocked gate.

### Findings (iteration 1)

| ID | Severity | Issue |
|----|----------|-------|
| F-001 | Major | Missing `tools_dynamic_selection_rejected.json`; test passed on OS file-open error |
| F-002 | Medium | Parity tests tautological, not golden-anchored |
| F-003 | Nit | Stale EP-018 "optional" comment on `toolsSelection` |
| F-004 | Suggestion | Untyped `tool_output_artifacts` (deferred by design) |

---

## Review iteration 2

**Review date:** 2026-05-30
**Stage 10 iteration:** 2 of max 5
**Scope:** Fix commit `153872a` on branch `epic/EP-037-consolidate-tool-selection`. Review performed by orchestrator after Opus 4.8 API limit; independent verification via code inspection + targeted `go test`.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 0 | Suggestion: 1
**Gate:** Pass

### Summary

Approve. All three blocking findings from iteration 1 are resolved. The consolidation remains structurally sound: runtime selection algorithms unchanged, validator ordering preserved (bounds pre-catalog, always_include floor post-catalog), strict `allowedToolsKeys` whitelist, legacy keys rejected fail-fast, 64-file config migration complete. Test integrity now matches the risk profile.

### Resolution verification

| Finding | Status | Evidence |
|---------|--------|----------|
| F-001 | **Resolved** | `internal/config/testdata/tools_dynamic_selection_rejected.json` exists and contains `"dynamic_selection"` block (line 45). `TestLoad_ToolsDynamicSelectionRejected` asserts `"tools.dynamic_selection is not supported"` (not bare substring). `go test -run TestLoad_ToolsDynamicSelectionRejected` passes. |
| F-002 | **Resolved** | `TestToolsSelectionParity_mergeEquivalentPreSelection` golden `wantMerged := []string{"t1","t2","t3","t4","t5"}`. `TestToolsSelectionParity_capEquivalentDynamicSelection` golden uncapped/capped slices; nil and `Enabled=false` do not apply cap. `go test -run TestToolsSelectionParity` passes. |
| F-003 | **Resolved** | `handler.go:80-81` comment now references EP-037 required `tools.selection`; nil only nil-config fallback. |
| F-004 | Suggestion (open, non-blocking) | `tool_output_artifacts` still parsed-and-dropped; deferred per design. |

### Carry-over re-confirmation

1. **Behaviour parity:** PASS — `mergeSelectedToolIDs`, `mergedAfterDynamicToolCap`, `min(toolSearchTopK, tool_vector_top_k_cap)` logic unchanged in diff.
2. **Validator ordering:** PASS — split validators at correct call sites.
3. **Config strictness:** PASS — explicit-JSON principle intact; legacy rejection; required `tools.selection`.
4. **Migration:** PASS — no stray legacy keys in product JSON except negative fixtures.
5. **AC coverage:** PASS — `./bin/validate EP-037` reports 15/15 in-scope automated; MANUAL ONLY lines do not name other AC codes.

### Gate decision

All Blocker/Major/Medium/Minor open counts are 0. §2.2 9↔10 loop complete. Proceed to stage 11.
