---
artefact: ep-code-review
epic_id: EP-037
status: draft
source_of_truth: true
gate: fail
latest_iteration: 1
open_counts:
  blocker: 0
  major: 1
  medium: 1
  minor: 0
non_blocking_counts:
  nit: 1
  suggestion: 1
next_action: return_to_stage_9
updated_at: 2026-05-30
---

# Code review — EP-037 Consolidate tool pre-selection configuration

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 1 | Medium 1 | Minor 0
Non-blocking counts: Nit 1 | Suggestion 1
Open findings:
- F-001 Major: `TestLoad_ToolsDynamicSelectionRejected` references a non-existent fixture and passes vacuously — AC-37.006 has no genuine automated coverage.
- F-002 Medium: Parity tests (AC-37.010 / AC-37.011) compare equivalent post-EP-037 handlers / a function against itself; they do not encode a pre-EP-037 baseline as the ACs claim.
Next action: Return to stage 9

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-037-consolidate-tool-selection` vs `main` (`git diff main...HEAD`, HEAD = a22465b). Readonly review.
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 1 | Minor: 0 | Nit: 1 | Suggestion: 1
**Gate:** Fail

### Summary

Request changes. The consolidation itself is clean and low-risk: the new required `tools.selection` block, the strict `allowedToolsKeys` whitelist, fail-fast rejection of `tool_pre_selection` / `tools.dynamic_selection`, validator ordering, and the 64-file migration are all correctly implemented, and the runtime tool-selection algorithms are **structurally unchanged** (so true behaviour parity holds). However, the test suite meant to *prove* the high-risk properties has two integrity defects: one legacy-key rejection test passes vacuously against a missing fixture (AC-37.006 effectively untested), and the "parity" tests are tautological rather than baseline-anchored. Both block §2.2 exit until fixed.

### Findings

#### F-001 — Major — `internal/config/tools_selection_test.go` (`TestLoad_ToolsDynamicSelectionRejected`)

The test calls `Load(filepath.Join("testdata", "tools_dynamic_selection_rejected.json"))`, but that fixture does NOT exist at HEAD. `Load` fails at `os.ReadFile` and returns `read config: open testdata/tools_dynamic_selection_rejected.json: no such file or directory`. The assertion `strings.Contains(err.Error(), "dynamic_selection")` matches the file PATH in that OS error, so the test passes for the wrong reason. AC-37.006 ("Load rejects `tools.dynamic_selection`") is counted covered but the production rejection path in `rejectRemovedToolsConfigKeys` (`internal/config/load.go`) is never exercised. (The `tool_pre_selection` case is fine: `tool_pre_selection_rejected.json` exists and genuinely contains the block.)

Recommendation: Add `internal/config/testdata/tools_dynamic_selection_rejected.json` (valid config that also includes a `tools.dynamic_selection` object) and tighten the assertion to the rejection message (e.g. `"tools.dynamic_selection is not supported"`) rather than the bare substring, so a missing/renamed fixture fails loudly.

#### F-002 — Medium — `internal/core/tools_selection_parity_test.go`

The parity tests do not encode a pre-EP-037 baseline, contrary to AC-37.010 / AC-37.011:
- `TestToolsSelectionParity_mergeEquivalentPreSelection` builds two handlers with identical inputs and asserts `merged == merged2` — tautological (proves determinism, not equivalence to prior behaviour).
- `TestToolsSelectionParity_capEquivalentDynamicSelection` compares `mergedAfterDynamicToolCap` against the function it delegates to with the same argument — proves plumbing, not a captured baseline.

Mitigating fact (why Medium not Major): the underlying algorithms (`mergeSelectedToolIDs`, `pickToolsForMainRequest`, the `min(toolSearchTopK, runtime_skills.tool_vector_top_k_cap)` cap) are NOT in the diff, so real behaviour parity is structurally guaranteed. The risk is test adequacy.

Recommendation: Anchor at least one row of each test to concrete expected tool-id slices (golden values) for a representative config. `TestToolsSelectionParity_runtimeTopKCap` (AC-37.009) is already concrete and fine.

#### F-003 — Nit — `internal/core/handler.go` (`toolsSelection` field comment)

Comment reads `// toolsSelection is optional EP-018 main-LLM tool cap; nil = disabled for both tiers.` After EP-037 `tools.selection` is required; nil is only the narrow nil-config test path. Update to reflect EP-037. Non-blocking.

#### F-004 — Suggestion — `tool_output_artifacts` untyped

`tool_output_artifacts` is whitelisted and present in `.config/config.json` but has no `ToolsConfig` field, so it is parsed-and-dropped. Intended per design (typed modelling deferred). Follow-up reminder only; out of scope.

### Verification of highest-risk items

1. **Behaviour parity (CRITICAL):** PASS (structural). `min(...)` cap and `mergedAfterDynamicToolCap` byte-for-byte unchanged; only field source changed (`h.toolsDynamic` → `h.toolsSelection`). `enabled=false`/`nil` → cap not applied preserved. Test quality gap in F-002.
2. **Validator ordering:** PASS. Bounds pre-catalog; `validateToolsSelectionAlwaysIncludeFloor` post-`toolcatalog.Load` (same site as old dynamic validation); floor not skipped.
3. **Config strictness / explicit-JSON:** PASS. `allowedToolsKeys` = {always_include, selection, vector_search_tools, create_tool_secret_patterns, tool_output_artifacts}; legacy keys rejected fail-fast; unknown `tools` keys rejected; `tools.selection` required; bounds at least as strict. Live/example/testdata/integration configs load.
4. **Complete migration:** PASS. Go legacy refs intentional only; only negative fixture `tool_pre_selection_rejected.json` carries a legacy key (plus the missing dynamic fixture — F-001).
5. **AC coverage integrity:** MOSTLY PASS; MANUAL ONLY status lines name only their own AC code (no under-count trap). Defect is AC-37.006 (F-001).
6. **KISS / boundaries / imports:** PASS. No broad handler refactor; minimal change.

### Test / verification

Readonly review; `make check` not run. Caution: `make check` currently passes despite F-001 because the vacuous test is green — green CI is not evidence AC-37.006 is covered.
