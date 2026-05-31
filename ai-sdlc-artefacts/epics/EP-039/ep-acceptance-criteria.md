---
artefact: ep-acceptance-criteria
epic_id: EP-039
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-039 — Config surface simplification — Acceptance criteria

## Introduction

Testable acceptance criteria for **EP-039**: DRY `tools.vector_search_tools`, typed and wired `tools.tool_output_artifacts`, and DRY SQLite reliability configuration. Criteria trace to [ep-requirements.md](ep-requirements.md) and [ep-scope.md](ep-scope.md).

**Git branch:** `epic/EP-039-config-surface-simplification`

---

## Acceptance criteria index

| AC ID | REQ (trace) | Test level | Summary |
|-------|-------------|------------|---------|
| [AC-39.001](#ac-39-001) | [REQ-39.001](ep-requirements.md#req-39-001--require-defaults-and-per-tool-overrides) | Unit | Load requires `defaults` + three tool override objects |
| [AC-39.002](#ac-39-002) | [REQ-39.002](ep-requirements.md#req-39-002--reject-legacy-vector_search_tools-shape), [REQ-39.020](ep-requirements.md#req-39-020--negative-legacy-fixtures) | Unit | Legacy flat shape rejected |
| [AC-39.003](#ac-39-003) | [REQ-39.003](ep-requirements.md#req-39-003--validate-defaults-bounds), [REQ-39.004](ep-requirements.md#req-39-004--validate-per-tool-overrides) | Unit | Bounds validation on defaults and overrides |
| [AC-39.004](#ac-39-004) | [REQ-39.005](ep-requirements.md#req-39-005--resolve-merged-settings), [REQ-39.006](ep-requirements.md#req-39-006--runtime-parity-for-vector-tools), [REQ-39.019](ep-requirements.md#req-39-019--equivalent-config-parity-tests) | Unit | Equivalent config → same resolved vector tool settings |
| [AC-39.005](#ac-39-005) | [REQ-39.007](ep-requirements.md#req-39-007--typed-tooloutputartifactsconfig), [REQ-39.008](ep-requirements.md#req-39-008--validate-artifact-fields) | Unit | Typed artifact config validates at load |
| [AC-39.006](#ac-39-006) | [REQ-39.009](ep-requirements.md#req-39-009--wire-tool_result_prompt_bytes) | Unit | Truncation uses `tool_result_prompt_bytes` from config |
| [AC-39.007](#ac-39-007) | [REQ-39.011](ep-requirements.md#req-39-011--reject-unknown-artifact-keys) | Unit | Unknown nested artifact keys fail load |
| [AC-39.008](#ac-39-008) | [REQ-39.012](ep-requirements.md#req-39-012--require-sqlite_store_defaults), [REQ-39.013](ep-requirements.md#req-39-013--per-store-override-blocks) | Unit | `sqlite_store_defaults` required; stores are overrides |
| [AC-39.009](#ac-39-009) | [REQ-39.014](ep-requirements.md#req-39-014--effective-pragma-parity), [REQ-39.019](ep-requirements.md#req-39-019--equivalent-config-parity-tests) | Unit | Equivalent config → same effective SQLite policy |
| [AC-39.010](#ac-39-010) | [REQ-39.015](ep-requirements.md#req-39-015--reject-redundant-legacy-reliability-only-shape), [REQ-39.020](ep-requirements.md#req-39-020--negative-legacy-fixtures) | Unit | Legacy full duplicate reliability blocks rejected |
| [AC-39.011](#ac-39-011) | [REQ-39.016](ep-requirements.md#req-39-016--update-repository-configs) | Unit | Examples and testdata load under new schema |
| [AC-39.012](#ac-39-012) | [REQ-39.017](ep-requirements.md#req-39-017--document-migration) | Manual | configuration.md documents migration |
| [AC-39.013](#ac-39-013) | [REQ-39.021](ep-requirements.md#req-39-021--make-check-passes) | Manual (make check) | `make check` exits zero |
| [AC-39.014](#ac-39-014) | [REQ-39.023](ep-requirements.md#req-39-023--limit-core-changes), [REQ-39.024](ep-requirements.md#req-39-024--no-toolsselection-changes) | Manual | Scope guards: no selection schema or handler refactor |
| [AC-39.015](#ac-39-015) | [REQ-39.025](ep-requirements.md#req-39-025--operator-config-loads) | Manual | Operator config loads after migration |

---

## Acceptance criteria

<a id="ac-39-001"></a>

### AC-39.001

**Trace:** [REQ-39.001](ep-requirements.md#req-39-001--require-defaults-and-per-tool-overrides)  
**Test level:** Unit

Given a valid post-EP-039 `config.json` fixture with `tools.vector_search_tools.defaults` and three per-tool objects  
When config load runs in `internal/config` tests  
Then the loaded config SHALL expose resolved settings for `search_vector_memory`, `search_vector_tool`, and `search_vector_skill`.

<a id="ac-39-002"></a>

### AC-39.002

**Trace:** [REQ-39.002](ep-requirements.md#req-39-002--reject-legacy-vector_search_tools-shape), [REQ-39.020](ep-requirements.md#req-39-020--negative-legacy-fixtures)  
**Test level:** Unit

Given a fixture using the pre-EP-039 flat repeated `vector_search_tools` shape  
When config load runs  
Then load SHALL fail with an error naming `tools.vector_search_tools` as unsupported.

<a id="ac-39-004"></a>

### AC-39.004

**Trace:** [REQ-39.005](ep-requirements.md#req-39-005--resolve-merged-settings), [REQ-39.006](ep-requirements.md#req-39-006--runtime-parity-for-vector-tools), [REQ-39.019](ep-requirements.md#req-39-019--equivalent-config-parity-tests)  
**Test level:** Unit

Given a table of equivalent pre- and post-epic `vector_search_tools` configs  
When `VectorSearchToolSettings` is called for each tool id  
Then resolved fields SHALL match the pre-epic baseline for each row.

<a id="ac-39-006"></a>

### AC-39.006

**Trace:** [REQ-39.009](ep-requirements.md#req-39-009--wire-tool_result_prompt_bytes)  
**Test level:** Unit

Given a handler built from config with `tool_result_prompt_bytes` set to a value distinct from the historical hardcoded default  
When tool results are truncated for the main LLM prompt in unit tests  
Then truncation SHALL occur at the configured byte limit.

<a id="ac-39-008"></a>

### AC-39.008

**Trace:** [REQ-39.012](ep-requirements.md#req-39-012--require-sqlite_store_defaults), [REQ-39.013](ep-requirements.md#req-39-013--per-store-override-blocks)  
**Test level:** Unit

Given a valid config with `sqlite_store_defaults` and minimal per-store `foreign_keys` overrides  
When config load runs  
Then both store reliability blocks SHALL resolve to policies with correct per-store `foreign_keys` and shared defaults for other fields.

<a id="ac-39-011"></a>

### AC-39.011

**Trace:** [REQ-39.016](ep-requirements.md#req-39-016--update-repository-configs)  
**Test level:** Unit

Given every migrated JSON under `config.examples/`, `internal/config/testdata/`, and integration test configs  
When batch config load tests run  
Then each file SHALL load successfully under the EP-039 schema.

<a id="ac-39-012"></a>

### AC-39.012

**Trace:** [REQ-39.017](ep-requirements.md#req-39-017--document-migration)  
**Test level:** Manual

Given the merged `docs/configuration.md` on the epic branch  
When an operator follows the migration section  
Then they SHALL be able to transform a pre-EP-039 config to the new schema without undocumented steps.

<a id="ac-39-013"></a>

### AC-39.013

**Trace:** [REQ-39.021](ep-requirements.md#req-39-021--make-check-passes)  
**Test level:** Manual (make check)

Given the epic branch implementation  
When `make check` runs from the repository root  
Then the command SHALL exit zero.

<a id="ac-39-014"></a>

### AC-39.014

**Trace:** [REQ-39.023](ep-requirements.md#req-39-023--limit-core-changes), [REQ-39.024](ep-requirements.md#req-39-024--no-toolsselection-changes)  
**Test level:** Manual

Given the epic branch diff  
When inspected for scope  
Then `tools.selection` schema SHALL be unchanged and `internal/core` changes SHALL be limited to config field wiring.

<a id="ac-39-015"></a>

### AC-39.015

**Trace:** [REQ-39.025](ep-requirements.md#req-39-025--operator-config-loads)  
**Test level:** Manual

Given the operator `.config/config.json` migrated per docs  
When PersonalAssistant starts  
Then config load SHALL succeed without legacy keys.

---

**Source:** [ep-requirements.md](ep-requirements.md) · [ep-scope.md](ep-scope.md)
