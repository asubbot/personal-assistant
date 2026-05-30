---
artefact: ep-system-design
epic_id: EP-037
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-037 — Consolidate tool pre-selection configuration — System design

**Contents**

- [Overview](#overview)
- [Architecture](#architecture)
- [Components and interfaces](#components-and-interfaces)
- [Data models](#data-models)
- [Error handling](#error-handling)
- [Testing strategy](#testing-strategy)
- [Implementation sequencing](#implementation-sequencing)
- [Requirement traceability](#requirement-traceability)

---

## Overview

EP-037 is a **config-shape refactor** (Refactoring **0.02**, direction B in [strategy.md](../../strategy.md)): collapse top-level `tool_pre_selection` and nested `tools.dynamic_selection` into one required **`tools.selection`** object with five explicit JSON fields. Selection algorithms (`mergeSelectedToolIDs`, `pickToolsForMainRequest`), merge order, and runtime `min(tool_search_top_k, runtime_skills.tool_vector_top_k_cap)` stay unchanged for equivalent settings ([REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap)–[REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)).

Explicit JSON rules are **not** weakened: `configRootJSONKeys` drops `tool_pre_selection`; legacy keys fail load via the existing **raw-JSON rejection** entry point `rejectRemovedUnsupportedConfigKeys` ([REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key)–[REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys), [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules)). `tools.vector_search_tools` schema is untouched ([REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry)); `internal/core` changes are wiring-only ([REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes)).

**Operator migration (1:1):**

| Removed | New |
|---------|-----|
| `tool_pre_selection.tool_search_top_k` | `tools.selection.tool_search_top_k` |
| `tool_pre_selection.tool_min_count` | `tools.selection.tool_min_count` |
| `tool_pre_selection.tool_fallback_cap` | `tools.selection.tool_fallback_cap` |
| `tools.dynamic_selection.enabled` | `tools.selection.enabled` |
| `tools.dynamic_selection.max_tools_for_llm_request` | `tools.selection.max_tools_for_llm_request` |

---

## Architecture

<p align="center"><img src="diagrams/c4-container.png" alt="C4 C2 — EP-037 Containers" /></p>

**Source:** [diagrams/c4-container.puml](diagrams/c4-container.puml). Regenerate PNG: `plantuml -tpng diagrams/c4-container.puml` from this directory.

### Config → runtime data flow

```mermaid
flowchart LR
  JSON["config.json tools.selection"]
  Load["config.Load / prepareConfig"]
  Val["validateToolsSelection"]
  Run["core.Run → conversationHandler"]
  Merge["mergeSelectedToolIDs"]
  Cap["mergedAfterDynamicToolCap"]
  JSON --> Load --> Val --> Run
  Run --> Merge
  Merge --> Cap
```

### Module boundaries

| Module | Responsibility | EP-037 change |
|--------|----------------|---------------|
| `internal/config` | Parse, validate, reject legacy keys | New `ToolsSelection`; remove `ToolPreSelection` / `ToolDynamicSelection`; extend raw rejection |
| `internal/core` | Handler construction, merge, cap | Read `cfg.Tools.Selection` only; no algorithm changes |
| `docs/configuration.md` | Operator contract | Document `tools.selection`; migration table |
| Repo JSON fixtures | Explicit-JSON examples | Replace legacy blocks in all tracked configs |

`runtime_skills.tool_vector_top_k_cap` stays under `runtime_skills` ([REQ-37.021](ep-requirements.md#req-37-021--keep-tool_vector_top_k_cap-location)); handler still applies `min` in `mergeSelectedToolIDs` ([REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap)).

---

## Components and interfaces

| Component | Responsibility | Key contract |
|-----------|----------------|--------------|
| `rejectRemovedUnsupportedConfigKeys` | Pre-unmarshal legacy key scan | Reject `tool_pre_selection` at root; delegate `tools` to `rejectRemovedToolsConfigKeys` ([REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key), [REQ-37.006](ep-requirements.md#req-37-006--reject-toolsdynamic_selection-key)) |
| `rejectRemovedToolsConfigKeys` | Reject removed `tools.*` keys | Add `dynamic_selection`; keep `text_based_enabled`, `llm_escalation` (EP-034) |
| `validateToolsObjectKeys` (new) | Whitelist nested `tools` keys | Fail on unknown keys e.g. `legacy_selection_stub` ([REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys)) |
| `validateToolsSelection` | Bounds + dynamic cap rules | Same limits as today’s `validateToolPreSelection` + `validateToolDynamicSelection` ([REQ-37.002](ep-requirements.md#req-37-002--validate-pre-selection-bounds)–[REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)) |
| `toolPreSelectionParams` → `toolsSelectionParams` | Extract ints for handler | `(topK, minCount, fallbackCap)` from `cfg.Tools.Selection` |
| `conversationHandler` | Runtime merge/cap | `toolSearchTopK` / `toolMinCount` / `toolFallbackCap` ints; `toolsSelection *config.ToolsSelection` for cap gate ([REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)) |
| `mergeSelectedToolIDs` | Vector pre-selection + merge | Unchanged logic; `topK` capped by `runtime_skills.tool_vector_top_k_cap` ([REQ-37.010](ep-requirements.md#req-37-010--preserve-merge-semantics)) |
| `mergedAfterDynamicToolCap` | EP-018 cap after merge | `toolsSelection.Enabled` + `MaxToolsForLLMRequest` on tier `full` ([REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap)) |

---

## Data models

### `ToolsSelection` (new) — lives in `internal/config/config.go`

Placed on **`ToolsConfig`** (not top-level `Config`) because all five fields belong under the required `tools` object ([REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block)).

```go
// ToolsSelection configures catalog vector pre-selection and per-request main-LLM tool cap (EP-037).
// All fields are required in JSON when tools is present; validated at load; no implicit defaults.
type ToolsSelection struct {
	ToolSearchTopK          int  `json:"tool_search_top_k"`           // >= 1, <= 500
	ToolMinCount            int  `json:"tool_min_count"`              // >= 1, <= 500
	ToolFallbackCap         int  `json:"tool_fallback_cap"`           // >= 1, <= 1000
	Enabled                 bool `json:"enabled"`                     // dynamic cap gate (tier full)
	MaxToolsForLLMRequest   int  `json:"max_tools_for_llm_request"`   // when enabled: >= 1 and >= always_include count; when disabled: may be 0
}

// ToolsConfig — EP-037 field change:
type ToolsConfig struct {
	AlwaysInclude            []string               `json:"always_include,omitempty"`
	Selection                *ToolsSelection        `json:"selection"` // required (non-nil after load)
	VectorSearchTools        *VectorSearchToolsConfig `json:"vector_search_tools,omitempty"`
	CreateToolSecretPatterns []string               `json:"create_tool_secret_patterns,omitempty"`
}
```

**Removed types/fields:**

- `Config.ToolPreSelection` / JSON `tool_pre_selection`
- `ToolsConfig.DynamicSelection` / JSON `dynamic_selection`
- Types `ToolPreSelection`, `ToolDynamicSelection` (delete; no aliases)

### Validation (`internal/config/load.go`)

| Rule | Function | Error prefix |
|------|----------|--------------|
| `tools.selection` present | `validateTools` → `validateToolsSelection` | `config: tools.selection is required` |
| `tool_search_top_k`, `tool_min_count` ∈ [1, 500] | `validateToolsSelection` | `config: tools.selection.<field> ...` (reuse `maxToolSearchTopK`, `maxToolMinCount`) |
| `tool_fallback_cap` ∈ [1, 1000] | `validateToolsSelection` | `config: tools.selection.tool_fallback_cap ...` (reuse `maxToolFallbackCap`) |
| `enabled == true` → `max_tools_for_llm_request >= 1` and `>= countValidAlwaysIncludeTools(c)` | `validateToolsSelection` | same messages as today’s `validateToolDynamicSelection` with `tools.selection` path |
| `enabled == false` → `max_tools_for_llm_request` may be `0` | `validateToolsSelection` | no error on zero max |

Replace calls: `validateToolPreSelection` / `validateToolDynamicSelection` → single `validateToolsSelection(c *Config)`.

### `enabled` vs `max_tools_for_llm_request` (load + runtime)

| Phase | `enabled == false` | `enabled == true` |
|-------|-------------------|-------------------|
| **Load** | `max_tools_for_llm_request` may be **0**; no always_include floor check ([REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled)) | `max_tools_for_llm_request` **≥ 1** and **≥** distinct valid `tools.always_include` count ([REQ-37.003](ep-requirements.md#req-37-003--validate-dynamic-cap-when-enabled)) |
| **Runtime** (`mergedAfterDynamicToolCap`) | Cap **skipped**; merged ids unchanged (same as `toolsDynamic == nil` or `!Enabled` today) | `pickToolsForMainRequest(ctx, merged, max_tools_for_llm_request)` after merge on tier **`full`** ([REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap)) |

The numeric cap field is **ignored at runtime** when `enabled` is false, even if non-zero in JSON.

### Legacy key rejection (EP-034 / EP-036 pattern)

**Entry point:** `prepareConfig` → `rejectRemovedUnsupportedConfigKeys(rootJSON)` (before `validateConfigRootObjectKeys`).

**Additions:**

```go
// In rejectRemovedUnsupportedConfigKeys, after unmarshaling root:
if _, has := root["tool_pre_selection"]; has {
    return errors.New("config: tool_pre_selection is not supported; use tools.selection (EP-037)")
}

// In rejectRemovedToolsConfigKeys:
if _, has := tools["dynamic_selection"]; has {
    return errors.New("config: tools.dynamic_selection is not supported; use tools.selection (EP-037)")
}
```

**Root keys:** remove `"tool_pre_selection"` from `configRootJSONKeys` in `root_keys.go` ([REQ-37.007](ep-requirements.md#req-37-007--omit-tool_pre_selection-from-root-keys)).

**Unknown `tools` keys:** add `validateToolsObjectKeys(rawTools)` called from `rejectRemovedToolsConfigKeys` (or immediately after), whitelisting documented keys:

`always_include`, `selection`, `vector_search_tools`, `create_tool_secret_patterns`

(Plus any other keys already present in operator `.config/config.json` and documented in `docs/configuration.md` at implementation time — e.g. if `tool_output_artifacts` remains in live config, either add it to the whitelist and struct in a follow-up or document removal; **EP-037 does not change** `vector_search_tools` shape.)

### Core wiring (`internal/core`)

**`run.go` — `Run` / handler construction:**

```go
// Before (remove):
toolTopK, toolMin, toolCap := toolPreSelectionParams(cfg)
toolDynSel = tc.DynamicSelection

// After:
sel := cfg.Tools.Selection // validated non-nil
toolTopK, toolMin, toolCap = sel.ToolSearchTopK, sel.ToolMinCount, sel.ToolFallbackCap
// handler field:
toolsSelection: sel,
```

Delete `toolPreSelectionParams`; add `toolsSelectionParams` only if a helper improves clarity (optional one-liner inline is fine).

**`handler.go`:**

- Replace `toolsDynamic *config.ToolDynamicSelection` with `toolsSelection *config.ToolsSelection`.
- `mergedAfterDynamicToolCap`: `if h.toolsSelection == nil || !h.toolsSelection.Enabled || len(merged) == 0` → return merged; else `pickToolsForMainRequest(..., h.toolsSelection.MaxToolsForLLMRequest)`.

**`integration_export.go`:**

- `NewConversationHandlerParams`: keep explicit `ToolSearchTopK` / `ToolMinCount` / `ToolFallbackCap` ints for tests; add optional `ToolsSelection *config.ToolsSelection` **or** `DynamicEnabled` + `DynamicMax` fields mirroring cap — prefer passing `*config.ToolsSelection` when non-nil so integration tests match production wiring ([REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection)).

**Unchanged:** `mergeSelectedToolIDs` top-K cap block (lines 318–321 today); `pickToolsForMainRequest` implementation.

---

## Error handling

| Failure | When | Message shape |
|---------|------|-----------------|
| Legacy `tool_pre_selection` | `rejectRemovedUnsupportedConfigKeys` | `tool_pre_selection is not supported; use tools.selection (EP-037)` |
| Legacy `tools.dynamic_selection` | `rejectRemovedToolsConfigKeys` | `tools.dynamic_selection is not supported; use tools.selection (EP-037)` |
| Missing `tools.selection` | `validateToolsSelection` | `tools.selection is required` |
| Out-of-range pre-selection ints | `validateToolsSelection` | `tools.selection.<field> must be >= 1` / `<= N` |
| Enabled cap violations | `validateToolsSelection` | same semantics as current dynamic_selection errors |
| Unknown `tools.*` key | `validateToolsObjectKeys` | `config: unknown tools key %q` |
| Unknown top-level key | `validateConfigRootObjectKeys` | unchanged ([REQ-37.018](ep-requirements.md#req-37-020--preserve-explicit-json-rules) / AC-37.018) |

---

## Testing strategy

### New / renamed unit tests (`internal/config`)

| Test | AC / REQ |
|------|----------|
| `TestLoad_ToolPreSelectionRejected` | AC-37.005 |
| `TestLoad_ToolsDynamicSelectionRejected` | AC-37.006 |
| `TestLoad_ToolsSelectionRequired` / missing block | AC-37.001 |
| `TestLoad_ToolsSelectionBounds` | AC-37.002 |
| `TestValidateToolsSelection_*` (port from `ep018_dynamic_tools_test.go`) | AC-37.003, AC-37.004 |
| `TestLoad_ToolsUnknownNestedKey` | AC-37.008 |
| `TestConfigRootJSONKeys_ExcludesToolPreSelection` | AC-37.007 |
| `TestLoad_AllFixturesLoad` (existing table) — update every fixture | AC-37.013 |

### Core parity tests (`internal/core`)

| File | Change |
|------|--------|
| `handler_ep018_coverage_test.go` | Build handler with `toolsSelection: &config.ToolsSelection{...}`; update doc needle from `dynamic_selection` → `tools.selection` where applicable |
| `handler_test.go` | Keep explicit `toolSearchTopK` fields; set `toolsSelection` when testing cap |
| `run_test.go` | Replace `ToolPreSelection` on `&config.Config{}` with `Tools: &config.ToolsConfig{Selection: &config.ToolsSelection{...}}}` |

### Parity fixtures ([REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity))

Add table-driven tests (config load + handler):

1. **Merge parity:** legacy top-level pre-selection values → same `mergeSelectedToolIDs` output as `tools.selection` with copied fields (AC-37.010).
2. **Cap parity:** legacy `dynamic_selection` → same post-cap ids as `tools.selection` with `enabled` + `max_tools_for_llm_request` (AC-37.011).
3. **Runtime cap:** `tool_search_top_k` > `tool_vector_top_k_cap` → effective top-K = min (AC-37.009).

### Manual gates

- `make check` ([REQ-37.018](ep-requirements.md#req-37-018--make-check-passes))
- `./bin/validate ears EP-037` ([REQ-37.019](ep-requirements.md#req-37-019--validate-ears-ep-037-passes))
- Read `docs/configuration.md` (AC-37.014, AC-37.015)
- Operator `.config/config.json` (AC-37.023)

---

## Implementation sequencing

Single epic branch; **one atomic config migration** per commit step to keep `make check` green:

1. **`internal/config` schema + validation + rejection** — add `ToolsSelection`, `validateToolsSelection`, extend `rejectRemovedUnsupportedConfigKeys` / `rejectRemovedToolsConfigKeys`, add `validateToolsObjectKeys`; remove old types/validators; update `root_keys.go`.
2. **Bulk JSON fixtures** — all `internal/config/testdata/*.json`, inline JSON in `config_test.go`, `intent_classifier_test.go`, `vector_search_tools_test.go`, `cmd/pa/main_test.go`.
3. **`internal/config` tests** — rename `tool_pre_selection_zero` case → `tools_selection_*`; port `ep018_dynamic_tools_test.go` → `tools_selection_test.go`; add legacy rejection tests.
4. **`internal/core` wiring** — `run.go`, `handler.go`, `handler_tier_main_prompt.go` (field rename only), `integration_export.go`, core tests.
5. **Operator artefacts** — `.config/config.json`, `config.examples/config.example.json`, `tests/integration/testdata/runtime_skills/minimal_ok/config.json`, `tests/integration/config_helpers.go`.
6. **`docs/configuration.md`** — migration table, intent-tier row uses `tools.selection.enabled`.
7. **Parity tests** — merge/cap/top-K tables (step 7 can merge with step 3–4 if preferred).

Do **not** refactor `vector_search_tools` ([REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry)) or decompose the handler ([REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes)).

---

## Files to update (implementation checklist)

### Product Go

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `ToolsSelection`; `ToolsConfig.Selection`; remove `ToolPreSelection`, `ToolDynamicSelection`, `Config.ToolPreSelection`, `ToolsConfig.DynamicSelection` |
| `internal/config/load.go` | `validateToolsSelection`; legacy rejection; remove `validateToolPreSelection` / `validateToolDynamicSelection`; wire `prepareConfig` |
| `internal/config/root_keys.go` | Drop `tool_pre_selection` |
| `internal/config/ep018_dynamic_tools_test.go` | Rename/replace → `tools_selection_test.go` |
| `internal/config/config_test.go` | Inline JSON + table cases |
| `internal/config/intent_classifier_test.go` | Inline JSON |
| `internal/config/vector_search_tools_test.go` | Inline JSON |
| `internal/core/run.go` | Read `cfg.Tools.Selection`; drop `toolPreSelectionParams` |
| `internal/core/handler.go` | `toolsSelection` field |
| `internal/core/handler_tier_main_prompt.go` | Use `toolsSelection` in `mergedAfterDynamicToolCap` |
| `internal/core/integration_export.go` | Wire selection into handler |
| `internal/core/run_test.go` | Config struct in tests |
| `internal/core/handler_test.go` | Handler literals |
| `internal/core/handler_ep018_coverage_test.go` | Handler literals + doc strings |
| `cmd/pa/main_test.go` | Inline config JSON |

### Config JSON (replace `tool_pre_selection` + move `dynamic_selection` → `tools.selection`)

| Path |
|------|
| `.config/config.json` |
| `config.examples/config.example.json` |
| `tests/integration/testdata/runtime_skills/minimal_ok/config.json` |
| **All** `internal/config/testdata/*.json` containing `tool_pre_selection` (52 files per repo grep) |

**New testdata (suggested names):**

- `tool_pre_selection_rejected.json`
- `tools_dynamic_selection_rejected.json`
- `tools_selection_missing.json`
- `tools_unknown_nested_key.json`
- `tools_selection_enabled_max_zero.json` (rename from `tools_dynamic_selection_enabled_max_zero.json`)

### Docs / integration

| File | Change |
|------|--------|
| `docs/configuration.md` | `tools.selection` section; remove `tool_pre_selection` / `dynamic_selection`; intent-tier table ([REQ-37.014](ep-requirements.md#req-37-014--document-migration-in-configurationmd), [REQ-37.015](ep-requirements.md#req-37-015--document-tool_vector_top_k_cap-interaction)) |
| `tests/integration/config_helpers.go` | `ensureCoreRunConfigRequiredSections`: set `cfg.Tools.Selection` instead of `cfg.ToolPreSelection` |

### Out of scope (no edits)

- `internal/config/vector_search_tools.go` and per-tool JSON shape ([REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry))
- Broad `internal/core` handler decomposition (EP-038)

---

## Test inventory (grep baseline)

Sites matching `tool_pre_selection|ToolPreSelection|dynamic_selection|DynamicSelection|ToolSearchTopK|ToolFallbackCap|ToolMinCount` under `internal/`, `cmd/`, `tests/` (update all on implementation):

### `internal/config`

- **Go sources:** `config.go`, `load.go`, `root_keys.go`, `config_test.go`, `intent_classifier_test.go`, `vector_search_tools_test.go`, `ep018_dynamic_tools_test.go`
- **testdata (52 JSON files):** `conversation_memory_vector_notes_negative.json`, `llm_log_retention_zero.json`, `llm_default_temperature_above_two.json`, `tool_pre_selection_zero.json`, `tools_llm_escalation_rejected.json`, `missing_read_memory.json`, `unknown_root_key.json`, `llm_default_response_format_invalid.json`, `llm_default_response_format_empty.json`, `missing_tool_catalog_path.json`, `missing_log_redaction.json`, `tools_bad_always_include.json`, `conversation_context_zero.json`, `invalid_auth.json`, `invalid_observability_http_empty_listen.json`, `llm_default_temperature_two.json`, `missing_dedicated_user.json`, `valid_with_users.json`, `intent_classifier_model_stage_rejected.json`, `openai_missing_api_key.json`, `missing_write_memory.json`, `missing_supports_tools.json`, `llm_supports_json_mode_rejected.json`, `missing_command_allowlist.json`, `valid_pa_timezone.json`, `nodes_missing_ssh_known_hosts_path.json`, `invalid_embedding_batch_size.json`, `valid_observability_http.json`, `log_redaction_invalid_regex.json`, `invalid_version.json`, `llm_json_object_without_supports_json_mode.json`, `missing_embedding_batch_size.json`, `valid_with_good_users.json`, `conversation_memory_vector_notes_over_max.json`, `valid_tools_text_based_enabled.json`, `invalid_host.json`, `intent_classifier_enabled_heuristic_only.json`, `tools_dynamic_selection_enabled_max_zero.json`, `valid_no_users.json`, `missing_llm_type.json`, `conversation_session_ok.json`, `missing_tools.json`, `invalid_observability_http_same_paths.json`, `empty_llm_providers.json`, `missing_llm_endpoint.json`, `missing_paths_memory_dir.json`, `conversation_memory_vector_all_zero.json`, `create_tool_bad_regex.json`, `missing_token_path.json`, `conversation_session_bad_max.json`, `log_redaction_invalid_regex.json`, `tools_text_based_enabled_rejected.json`, `llm_default_temperature_negative.json`, `valid_max_message_length.json`, `missing_pa_timezone.json`, `missing_embedding.json`, `intent_classifier_full_lite_patterns_rejected.json`, `invalid_pa_timezone.json`

### `internal/core`

- `run.go`, `handler.go`, `handler_tier_main_prompt.go`, `integration_export.go`
- `run_test.go`, `handler_test.go`, `handler_ep018_coverage_test.go`

### `cmd/`

- `cmd/pa/main_test.go`

### `tests/integration`

- `config_helpers.go`
- `testdata/runtime_skills/minimal_ok/config.json`

### Repo configs / docs

- `.config/config.json`
- `config.examples/config.example.json`
- `docs/configuration.md`

---

## Requirement traceability

| REQ | Design section |
|-----|----------------|
| [REQ-37.001](ep-requirements.md#req-37-001--require-toolsselection-block) | `ToolsSelection` on `ToolsConfig`; `validateToolsSelection` required block |
| [REQ-37.002](ep-requirements.md#req-37-002--validate-pre-selection-bounds) | Bounds in `validateToolsSelection`; reuse `maxToolSearchTopK` / `maxToolMinCount` / `maxToolFallbackCap` |
| [REQ-37.003](ep-requirements.md#req-37-003--validate-dynamic-cap-when-enabled) | Enabled branch in `validateToolsSelection` |
| [REQ-37.004](ep-requirements.md#req-37-004--allow-zero-max-when-disabled) | Disabled branch in `validateToolsSelection` |
| [REQ-37.005](ep-requirements.md#req-37-005--reject-tool_pre_selection-root-key) | `rejectRemovedUnsupportedConfigKeys` root check |
| [REQ-37.006](ep-requirements.md#req-37-006--reject-toolsdynamic_selection-key) | `rejectRemovedToolsConfigKeys` |
| [REQ-37.007](ep-requirements.md#req-37-007--omit-tool_pre_selection-from-root-keys) | `root_keys.go` |
| [REQ-37.008](ep-requirements.md#req-37-008--reject-unknown-tools-nested-keys) | `validateToolsObjectKeys` |
| [REQ-37.009](ep-requirements.md#req-37-009--apply-runtime-skills-top-k-cap) | Unchanged `mergeSelectedToolIDs` cap |
| [REQ-37.010](ep-requirements.md#req-37-010--preserve-merge-semantics) | Parity tests; unchanged merge |
| [REQ-37.011](ep-requirements.md#req-37-011--preserve-ep-018-dynamic-cap) | `mergedAfterDynamicToolCap` + `toolsSelection` |
| [REQ-37.012](ep-requirements.md#req-37-012--wire-handler-from-toolsselection) | `run.go` / `handler.go` / `integration_export.go` |
| [REQ-37.013](ep-requirements.md#req-37-013--update-repository-configs) | Files checklist |
| [REQ-37.014](ep-requirements.md#req-37-014--document-migration-in-configurationmd) | `docs/configuration.md` |
| [REQ-37.015](ep-requirements.md#req-37-015--document-tool_vector_top_k_cap-interaction) | `docs/configuration.md` |
| [REQ-37.016](ep-requirements.md#req-37-016--tests-reject-legacy-keys) | New rejection tests |
| [REQ-37.017](ep-requirements.md#req-37-017--tests-prove-equivalent-config-parity) | Parity table tests |
| [REQ-37.018](ep-requirements.md#req-37-018--make-check-passes) | Sequencing gate |
| [REQ-37.019](ep-requirements.md#req-37-019--validate-ears-ep-037-passes) | Artefact gate |
| [REQ-37.020](ep-requirements.md#req-37-020--preserve-explicit-json-rules) | Root keys + examples |
| [REQ-37.021](ep-requirements.md#req-37-021--keep-tool_vector_top_k_cap-location) | No move to `tools.selection` |
| [REQ-37.022](ep-requirements.md#req-37-022--defer-vector_search_tools-dry) | Out of scope |
| [REQ-37.023](ep-requirements.md#req-37-023--limit-core-handler-changes) | Wiring-only core diff |
| [REQ-37.024](ep-requirements.md#req-37-024--no-new-selection-features) | No algorithm changes |
