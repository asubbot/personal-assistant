---
artefact: ep-acceptance-criteria
epic_id: EP-037
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-037 — Consolidate tool pre-selection configuration — Acceptance criteria

## Introduction

Testable acceptance criteria for **EP-037**: merge catalog tool vector pre-selection and EP-018 per-request tool capping into one required `tools.selection` block, reject legacy `tool_pre_selection` and `tools.dynamic_selection` at config load, and preserve runtime tool-selection outcomes for equivalent settings. Criteria trace to [ep-requirements.md](ep-requirements.md) and [ep-scope.md](ep-scope.md). Test levels follow [strategy.md](../../strategy.md) §2 (Unit / Integration / E2E / Manual).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Test level | Summary |
|-------|-------------|------------|---------|
| [AC-37.001](#ac-37-001) | [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block) | Unit | `tools.selection` required with five explicit fields |
| [AC-37.002](#ac-37-002) | [REQ-37.002](ep-requirements.md#req-37-002--validate-pre-selection-bounds) | Unit | Pre-selection numeric bounds validated at load |
| [AC-37.003](#ac-37-003) | [REQ-37.003](ep-requirements.md#req-37-003--validate-dynamic-cap-when-enabled) | Unit | Dynamic cap rules when `enabled` is true |
| [AC-37.004](#ac-37-004) | [REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled) | Unit | `max_tools_for_llm_request` may be 0 when disabled |
| [AC-37.005](#ac-37-005) | [REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key), [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys) | Unit | Load rejects top-level `tool_pre_selection` |
| [AC-37.006](#ac-37-006) | [REQ-37.006](ep-requirements.md#req-37-006--reject-toolsdynamic_selection-key), [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys) | Unit | Load rejects `tools.dynamic_selection` |
| [AC-37.007](#ac-37-007) | [REQ-37.007](ep-requirements.md#req-37-007--omit-tool_pre_selection-from-root-keys) | Unit | `configRootJSONKeys` excludes `tool_pre_selection` |
| [AC-37.008](#ac-37-008) | [REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys) | Unit | Unknown keys under `tools` fail load |
| [AC-37.009](#ac-37-009) | [REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity) | Unit | Effective top-K uses `min(tool_search_top_k, tool_vector_top_k_cap)` |
| [AC-37.010](#ac-37-010) | [REQ-37.010](ep-requirements.md#req-37-010--preserve-merge-semantics), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity) | Unit | Equivalent config → same merged tool id set |
| [AC-37.011](#ac-37-011) | [REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity) | Unit | Equivalent config → same dynamic-cap tool ids on tier `full` |
| [AC-37.012](#ac-37-012) | [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection) | Unit | Core reads pre-selection and cap from `tools.selection` only |
| [AC-37.013](#ac-37-013) | [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs), [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules) | Unit | Examples, testdata, and integration configs load under new schema |
| [AC-37.014](#ac-37-014) | [REQ-37.014](ep-requirements.md#req-37-014--document-migration-in-configurationmd) | Manual | Operator docs describe `tools.selection` migration |
| [AC-37.015](#ac-37-015) | [REQ-37.015](ep-requirements.md#req-37-015--document-tool_vector_top_k_cap-interaction) | Manual | Docs state `tool_vector_top_k_cap` cross-field interaction |
| [AC-37.016](#ac-37-016) | [REQ-37.018](ep-requirements.md#req-37-018--make-check-passes) | Manual (make check) | `make check` exits zero |
| [AC-37.017](#ac-37-017) | [REQ-37.019](ep-requirements.md#req-37-019--validate-ears-ep-037-passes) | Manual (validate) | `./bin/validate ears EP-037` passes |
| [AC-37.018](#ac-37-018) | [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules) | Unit | Unknown top-level keys and single-occurrence rules preserved |
| [AC-37.019](#ac-37-019) | [REQ-37.021](ep-requirements.md#req-37-021--keep-tool_vector_top_k_cap-location) | Unit | `tool_vector_top_k_cap` remains under `runtime_skills` only |
| [AC-37.020](#ac-37-020) | [REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry) | Manual | `tools.vector_search_tools` schema unchanged |
| [AC-37.021](#ac-37-021) | [REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes) | Manual | Core changes limited to config wiring |
| [AC-37.022](#ac-37-022) | [REQ-37.024](ep-requirements.md#req-37-024--no-new-selection-features) | Manual | No new ranking signals or tier-specific caps |
| [AC-37.023](#ac-37-023) | [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs) | Manual | Operator `.config/config.json` loads without legacy keys |

---

## Scenarios

### AC-37.001 Require tools.selection with five fields (Trace: REQ-37.001)

Given a valid `config.json` fixture on the epic branch  
When config load runs in `internal/config` tests  
Then the loaded `tools.selection` object SHALL expose `tool_search_top_k`, `tool_min_count`, `tool_fallback_cap`, `enabled`, and `max_tools_for_llm_request` with the values from the fixture.

### AC-37.005 Reject tool_pre_selection (Trace: REQ-37.005, REQ-37.016)

Given a `config.json` fixture containing the top-level key `tool_pre_selection`  
When config load runs in `internal/config` tests  
Then load SHALL fail with an explicit error naming `tool_pre_selection` as unsupported.

### AC-37.006 Reject tools.dynamic_selection (Trace: REQ-37.006, REQ-37.016)

Given a `config.json` fixture containing `tools.dynamic_selection`  
When config load runs in `internal/config` tests  
Then load SHALL fail with an explicit error naming `tools.dynamic_selection` as unsupported.

### AC-37.009 Runtime-skills top-K cap (Trace: REQ-37.009, REQ-37.017)

Given equivalent pre-epic and post-epic configuration where `runtime_skills` is enabled with positive `tool_vector_top_k_cap` and `tools.selection.tool_search_top_k` exceeds that cap  
When vector pre-selection runs in unit tests for `mergeSelectedToolIDs`  
Then the effective vector top-K passed to tool index selection SHALL equal `min(tool_search_top_k, runtime_skills.tool_vector_top_k_cap)`.

### AC-37.010 Merge parity (Trace: REQ-37.010, REQ-37.017)

Given a representative fixture table mapping legacy `tool_pre_selection` fields into `tools.selection` one-to-one  
When `mergeSelectedToolIDs` runs for the same user message and catalog state  
Then the merged catalog tool id set SHALL match the baseline captured before EP-037 for that fixture.

### AC-37.011 Dynamic cap parity (Trace: REQ-37.011, REQ-37.017)

Given a representative fixture table mapping legacy `tools.dynamic_selection` into `tools.selection` with `enabled` true  
When assembling tools for tier `full` on the epic branch  
Then the post-merge tool id list after dynamic cap SHALL match the baseline for the same fixture.

### AC-37.013 Repository configs load (Trace: REQ-37.013, REQ-37.020)

Given every JSON file under `config.examples/`, `internal/config/testdata/`, and `tests/integration/` updated on the epic branch  
When config load runs in automated tests  
Then each file SHALL load successfully with `tools.selection` present and without `tool_pre_selection` or `tools.dynamic_selection`.

---

## Acceptance criteria

<a id="ac-37-001"></a>

### AC-37.001

**Trace:** [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block)  
**Test level:** Unit

Given a valid configuration fixture with `tools.selection` on the epic branch  
When config load runs in `internal/config` tests  
Then load SHALL succeed  
And the parsed `tools.selection` SHALL include explicit fields `tool_search_top_k`, `tool_min_count`, `tool_fallback_cap`, `enabled`, and `max_tools_for_llm_request`.

---

<a id="ac-37-002"></a>

### AC-37.002

**Trace:** [REQ-37.002](ep-requirements.md#req-37-002--validate-pre-selection-bounds)  
**Test level:** Unit

Given `tools.selection` fixtures with `tool_search_top_k`, `tool_min_count`, or `tool_fallback_cap` below 1 or above former limits (500 for top-K and min count; 1000 for fallback cap)  
When config load runs  
Then load SHALL fail with explicit validation errors for the out-of-range field.

---

<a id="ac-37-003"></a>

### AC-37.003

**Trace:** [REQ-37.003](ep-requirements.md#req-37-003--validate-dynamic-cap-when-enabled)  
**Test level:** Unit

Given `tools.selection.enabled` is true  
When `max_tools_for_llm_request` is less than 1 or less than the count of distinct valid `tools.always_include` tool ids in the same fixture  
Then config load SHALL fail with an explicit validation error.

---

<a id="ac-37-004"></a>

### AC-37.004

**Trace:** [REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)  
**Test level:** Unit

Given `tools.selection.enabled` is false and `max_tools_for_llm_request` is 0  
When config load runs  
Then load SHALL succeed.

---

<a id="ac-37-005"></a>

### AC-37.005

**Trace:** [REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key), [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys)  
**Test level:** Unit

Given a `config.json` fixture containing the top-level key `tool_pre_selection`  
When config load runs in `internal/config` tests  
Then load SHALL fail  
And the error message SHALL name `tool_pre_selection` as an unsupported key.

---

<a id="ac-37-006"></a>

### AC-37.006

**Trace:** [REQ-37.006](ep-requirements.md#req-37-006--reject-toolsdynamic_selection-key), [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys)  
**Test level:** Unit

Given a `config.json` fixture containing `tools.dynamic_selection`  
When config load runs in `internal/config` tests  
Then load SHALL fail  
And the error message SHALL name `tools.dynamic_selection` as an unsupported key.

---

<a id="ac-37-007"></a>

### AC-37.007

**Trace:** [REQ-37.007](ep-requirements.md#req-37-007--omit-tool_pre_selection-from-root-keys)  
**Test level:** Unit

Given the `configRootJSONKeys` list in `internal/config` after EP-037  
When inspecting the declared root keys  
Then `tool_pre_selection` SHALL NOT appear in the list.

---

<a id="ac-37-008"></a>

### AC-37.008

**Trace:** [REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys)  
**Test level:** Unit

Given a `config.json` fixture with an unknown nested key under `tools` (for example `tools.legacy_selection_stub`)  
When config load runs  
Then load SHALL fail with an explicit unknown-key validation error.

---

<a id="ac-37-009"></a>

### AC-37.009

**Trace:** [REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity)  
**Test level:** Unit

Given `runtime_skills` enabled with `tool_vector_top_k_cap` greater than zero and catalog vector pre-selection active  
When `mergeSelectedToolIDs` runs with `tools.selection.tool_search_top_k` greater than the cap  
Then the effective vector top-K used for `toolindex.SelectToolIDs` SHALL equal `min(tools.selection.tool_search_top_k, runtime_skills.tool_vector_top_k_cap)`.

---

<a id="ac-37-010"></a>

### AC-37.010

**Trace:** [REQ-37.010](ep-requirements.md#req-37-010--preserve-merge-semantics), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity)  
**Test level:** Unit

Given representative equivalent configurations where legacy `tool_pre_selection` values are copied field-for-field into `tools.selection`  
When `mergeSelectedToolIDs` runs for a fixed user message, catalog, and `always_include` / skill-linked ids  
Then the merged catalog tool id set SHALL match the pre-EP-037 baseline for each fixture row.

---

<a id="ac-37-011"></a>

### AC-37.011

**Trace:** [REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap), [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity)  
**Test level:** Unit

Given tier `full` main-LLM assembly with `tools.selection.enabled` true and representative equivalent legacy `tools.dynamic_selection` settings mapped into `tools.selection`  
When dynamic cap is applied after merge (`mergedAfterDynamicToolCap` / `pickToolsForMainRequest`)  
Then the resulting tool id list SHALL match the pre-EP-037 baseline for each fixture row.

---

<a id="ac-37-012"></a>

### AC-37.012

**Trace:** [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)  
**Test level:** Unit

Given the EP-037 change set in `internal/core` (`run.go`, `handler.go`, `integration_export.go`) and `internal/config`  
When inspecting handler construction and tool-selection call sites in unit or compile-time tests  
Then pre-selection and dynamic-cap parameters SHALL be sourced from `tools.selection` via `internal/config` only  
And SHALL NOT read top-level `tool_pre_selection` or nested `tools.dynamic_selection`.

---

<a id="ac-37-013"></a>

### AC-37.013

**Trace:** [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs), [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules)  
**Test level:** Unit

Given updated JSON under `config.examples/`, `internal/config/testdata/`, and `tests/integration/` on the epic branch  
When config load runs in automated tests  
Then each file SHALL load successfully with required `tools.selection` and without legacy keys.

---

<a id="ac-37-014"></a>

### AC-37.014

**Trace:** [REQ-37.014](ep-requirements.md#req-37-014--document-migration-in-configurationmd)  
**Test level:** Manual  
**Status:** AC-37.014 MANUAL ONLY — verified by reading `docs/configuration.md` for `tools.selection`, one-to-one field migration from removed keys, and intent-tier table references to `tools.selection.enabled`.

Given the EP-037 documentation update  
When an operator reads `docs/configuration.md`  
Then the doc SHALL describe `tools.selection` and one-to-one migration from `tool_pre_selection` and `tools.dynamic_selection`  
And the intent-tier tool-shaping table SHALL reference `tools.selection.enabled` for the dynamic cap.

---

<a id="ac-37-015"></a>

### AC-37.015

**Trace:** [REQ-37.015](ep-requirements.md#req-37-015--document-tool_vector_top_k_cap-interaction)  
**Test level:** Manual  
**Status:** AC-37.015 MANUAL ONLY — verified by reading `docs/configuration.md` for `runtime_skills.tool_vector_top_k_cap` remaining under `runtime_skills` and limiting effective vector top-K per REQ-37.009.

Given the EP-037 documentation update  
When an operator reads `docs/configuration.md`  
Then the doc SHALL state that `runtime_skills.tool_vector_top_k_cap` stays under `runtime_skills`  
And SHALL explain that it limits effective vector top-K together with `tools.selection.tool_search_top_k`.

---

<a id="ac-37-016"></a>

### AC-37.016

**Trace:** [REQ-37.018](ep-requirements.md#req-37-018--make-check-passes)  
**Test level:** Manual (make check)  
**Status:** AC-37.016 MANUAL ONLY — verified by running `make check` from the repository root (exit 0); this is a process gate, not a unit test.

Given EP-037 implementation is complete on the epic branch  
When `make check` runs from the repository root  
Then it SHALL exit with status zero.

---

<a id="ac-37-017"></a>

### AC-37.017

**Trace:** [REQ-37.019](ep-requirements.md#req-37-019--validate-ears-ep-037-passes)  
**Test level:** Manual (validate)  
**Status:** AC-37.017 MANUAL ONLY — verified by running `./bin/validate ears EP-037` from the repository root after `make build`; this is an artefact gate, not a product unit test.

Given `ep-requirements.md` for EP-037 on the epic branch  
When `./bin/validate ears EP-037` runs from the repository root  
Then validation SHALL report no EARS format errors for the requirements artefact.

---

<a id="ac-37-018"></a>

### AC-37.018

**Trace:** [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules)  
**Test level:** Unit

Given a fixture with an unknown top-level key  
When config load runs  
Then load SHALL fail with an explicit unknown-key error  
And automated inspection of `config.examples/` fixtures on the epic branch SHALL show each documented top-level key appearing exactly once per file.

---

<a id="ac-37-019"></a>

### AC-37.019

**Trace:** [REQ-37.021](ep-requirements.md#req-37-021--keep-tool_vector_top_k_cap-location)  
**Test level:** Unit

Given a valid configuration loaded on the epic branch  
When inspecting the parsed config struct for runtime skills and tools  
Then `tool_vector_top_k_cap` SHALL be defined only under `runtime_skills`  
And SHALL NOT appear as a field under `tools.selection`.

---

<a id="ac-37-020"></a>

### AC-37.020

**Trace:** [REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry)  
**Test level:** Manual  
**Status:** AC-37.020 MANUAL ONLY — verified by inspecting the EP-037 branch diff for `tools.vector_search_tools` schema and validation (no structural DRY refactor).

Given the EP-037 change set  
When reviewing `internal/config` and example configs for `tools.vector_search_tools`  
Then the per-tool object shape and validation rules SHALL remain unchanged from pre-EP-037.

---

<a id="ac-37-021"></a>

### AC-37.021

**Trace:** [REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes)  
**Test level:** Manual  
**Status:** AC-37.021 MANUAL ONLY — verified by reviewing the EP-037 branch diff under `internal/core` (config field wiring only; no handler decomposition).

Given the EP-037 change set  
When reviewing `internal/core` diffs  
Then changes SHALL be limited to reading `tools.selection` and related wiring in `run.go`, `handler.go`, and `integration_export.go`  
And SHALL NOT include broad handler refactors planned for EP-038.

---

<a id="ac-37-022"></a>

### AC-37.022

**Trace:** [REQ-37.024](ep-requirements.md#req-37-024--no-new-selection-features)  
**Test level:** Manual  
**Status:** AC-37.022 MANUAL ONLY — verified by reviewing the EP-037 branch diff for absence of new ranking signals, tier-specific caps, or selection algorithm changes beyond config consolidation.

Given the EP-037 change set  
When reviewing tool-selection code paths  
Then changes SHALL be restricted to configuration consolidation, documentation, and tests required by this epic  
And SHALL NOT add new ranking signals or tier-specific tool caps.

---

<a id="ac-37-023"></a>

### AC-37.023

**Trace:** [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs)  
**Test level:** Manual  
**Status:** AC-37.023 MANUAL ONLY — operator `.config/config.json` is verified manually and by application startup validation.  
**Related coverage:** automated positive-load coverage for the new schema is provided by example configs, testdata fixtures, and integration configs under automated Unit tests.

Given the operator `.config/config.json` updated on the epic branch  
When the application loads and validates it at startup (or an operator runs config load against it with required secrets)  
Then the live config SHALL load successfully with `tools.selection` and without `tool_pre_selection` or `tools.dynamic_selection`.
