---
artefact: ep-code-review
epic_id: EP-039
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
  nit: 1
  suggestion: 1
next_action: proceed_to_stage_11
updated_at: 2026-05-31
---

# Code review — EP-039 Config surface simplification

---

## Current Gate Summary

Gate: Pass
Latest iteration: 2
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Non-blocking counts: Nit 1 | Suggestion 1
Open findings: (none)
Next action: Proceed to stage 11

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 10 iteration:** 1 of max 5
**Scope:** Branch `epic/EP-039-config-surface-simplification` vs `main` (104 files, +2594/−592 lines). Product scope: config DRY (`vector_search_tools`, `sqlite_store_defaults`, `tool_output_artifacts`), core truncation wiring, docs and testdata migration.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 1 | Nit: 1 | Suggestion: 2
**Gate:** Fail

### Summary

The EP-039 implementation is well-executed and aligns with the approved system design and implementation plan. All six plan phases are marked complete: legacy rejection scaffolding, SQLite DRY merge, vector-search defaults+overrides, typed `tool_output_artifacts` with nested-key whitelist, core truncation wiring, docs migration tables, and bulk testdata/example updates. `make check` passes cleanly (race, vet, lint, vuln, module boundaries, integration). Scope guards hold: `tools.selection` schema unchanged; `internal/core` changes limited to handler truncation wiring; `cmd/pa` updates only switch SQLite policy access to merged `*Policy()` methods.

One minor finding blocks the gate: REQ-39.019 / plan task 4.3 call for table-driven pre/post parity tests (≥3 rows per workstream per design); the branch has merge and spot-check unit tests but no explicit parity table. Recommend a small stage-9 addition, then re-run stage 10.

### What was done well

- **Legacy rejection:** `rejectLegacyEP039Shapes` runs before unmarshal; vector and SQLite legacy fixtures (`vector_search_tools_legacy_rejected.json`, `sqlite_reliability_legacy_rejected.json`) assert actionable error text (AC-39.002, AC-39.010).
- **Vector DRY:** `VectorSearchToolsConfig` defaults + pointer overrides, `mergeVectorSearchTool`, and `VectorSearchToolSettings` preserve the public API; bounds validation on defaults and merged overrides (REQ-39.001–006).
- **SQLite DRY:** `sqlite_store_defaults` in sorted `configRootJSONKeys`; `mergeSQLiteStoreReliability` + `VectorStoreReliabilityPolicy` / `JobsStoreReliabilityPolicy`; `cmd/pa` call sites updated consistently.
- **Artifacts:** `ToolOutputArtifactsConfig` typed on `ToolsConfig`; nested whitelist in `validateToolOutputArtifactsObjectKeys`; load-time field validation; `ResolvePaths` resolves `directory` under `PA_DATA_DIR`; `ArtifactDirectory(cfg)` helper with unit tests (design-accepted resolver-only for REQ-39.010).
- **Core wiring:** `toolResultPromptBytes` on `conversationHandler`, set in `newRunConversationHandler` via `toolResultPromptBytesFromConfig`; `truncateToolResultForPrompt` uses handler field with fallback to `maxToolResultPromptBytes` (AC-39.006).
- **Migration:** `docs/configuration.md` EP-039 section with field-for-field tables and JSON examples (AC-39.012); `config.examples/config.example.json` and ~60 testdata files migrated; integration `minimal_ok/config.json` updated.
- **Traceability:** AC comments on new tests (`ep039_legacy_test.go`, `sqlite_reliability_test.go`, `tool_output_artifacts_test.go`, `vector_search_tools_test.go`, `handler_test.go`).

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-001 | **Minor** | REQ-39.019; [ep-system-design.md](ep-system-design.md) Testing strategy; [ep-implementation-plan.md](ep-implementation-plan.md) task 4.3 | Design and plan require table-driven parity tests (≥3 rows per workstream) proving equivalent pre/post configs yield identical resolved vector settings, truncation limits, and SQLite policies. Implementation has merge tests (`TestMergeVectorSearchTool_InheritsDefaults`, `TestMergeSQLiteStoreReliability_*`) and handler truncation test with a hardcoded limit, but no explicit parity table mapping baseline resolved values to post-EP-039 configs. Task 4.3 is marked complete without this deliverable. | Add `TestEP039_Parity_*` table tests: for vector search, SQLite merge/`ToPolicy()`, and `tool_result_prompt_bytes`, define ≥3 rows each with expected resolved values; load post-EP-039 JSON (or call merge helpers with constructed inputs) and assert equality with documented baselines. Tag `// Covers AC-39.004`, `AC-39.009`, `REQ-39.019`. |
| F-002 | **Nit** | `internal/core/run.go`, `internal/core/handler_llm.go` | Design said remove package constant `maxToolResultPromptBytes`; constant retained as fallback when `tool_output_artifacts` is absent or `tool_result_prompt_bytes` is unset. | Acceptable deviation; optional one-line comment in `run.go` that fallback preserves pre-EP-039 behaviour when the artifacts block is omitted. |
| F-003 | **Suggestion** | `internal/core/run.go:186-191` | `toolResultPromptBytesFromConfig` has no unit test; AC-39.006 covered only by direct handler field assignment in `TestTruncateToolResultForPrompt_usesConfiguredLimit`. | Add a small table test for `toolResultPromptBytesFromConfig` (nil config, nil artifacts, zero bytes, valid bytes) or one `BuildMessageHandler` test asserting propagated limit. |
| F-004 | **Suggestion** | `internal/config/ep039_traceability_test.go` | `TestEP039_MigratedValidTestdataLoads` spot-checks 3 fixtures; AC-39.011 wording implies broader batch coverage. Mitigated by existing `config_test.go` table tests loading many migrated fixtures and green `make check`. | Optionally glob `testdata/valid_*.json` (and other known-good fixtures) or document that existing load tests satisfy AC-39.011. |

### Plan alignment

| Area | Plan / REQ | Status |
|------|------------|--------|
| Phase 1 legacy rejection + root key | Tasks 1.1–1.2 | ✅ Complete |
| Phase 2 SQLite DRY + fixture migrate | Tasks 2.1–2.2 | ✅ Complete |
| Phase 3 vector_search_tools DRY | Tasks 3.1–3.2 | ✅ Complete |
| Phase 4 tool_output_artifacts + core wire | Tasks 4.1–4.2 | ✅ Complete |
| Phase 4.3 parity tests | Task 4.3 | ⚠️ Partial (F-001) |
| Phase 5 docs | Task 5.1 | ✅ Complete |
| Phase 6 make check | Task 6.1 | ✅ Complete |
| Scope: no tools.selection change | REQ-39.024, AC-39.014 | ✅ Verified |
| Scope: core wiring only | REQ-39.023, AC-39.014 | ✅ Verified |

### Test / verification

- `make check` — **PASS** (exit 0). All packages including `pa/cmd/pa`, `pa/internal/config`, `pa/internal/core`, `pa/tests/integration` pass with race detector; golangci-lint 0 issues; govulncheck clean; module boundaries OK.
- Legacy rejection: `TestLoad_LegacyVectorSearchToolsShape_Rejected`, `TestLoad_LegacySQLiteReliabilityShape_Rejected`.
- Vector DRY: `TestLoad_VectorSearchToolsConfig_Valid`, `TestLoad_VectorSearchToolsConfig_InvalidBounds`, `TestMergeVectorSearchTool_InheritsDefaults`.
- SQLite DRY: `TestMergeSQLiteStoreReliability_InheritsDefaults`, `TestMergeSQLiteStoreReliability_OverrideFields`, `TestValidateVectorStoreReliability_MergedToPolicy`.
- Artifacts: `TestLoad_ToolOutputArtifacts_Valid`, invalid bounds, unknown nested key; `TestArtifactDirectory_*`.
- Truncation: `TestTruncateToolResultForPrompt`, `TestTruncateToolResultForPrompt_usesConfiguredLimit` (AC-39.006).
- AC-39.015 (operator `.config/config.json`): not verified in this review (gitignored); manual per operator procedure.

### Residual risks / follow-ups

- Operator config migration (REQ-39.025 / AC-39.015) remains manual; docs migration section is adequate for self-service.
- `ArtifactDirectory` has no production call site yet; accepted per design for REQ-39.010 (future tool persistence paths).
- `./bin/validate EP-039` not registered in Makefile; plan notes “when registered” — optional follow-up.

---

## Review iteration 2

**Review date:** 2026-05-31
**Stage 10 iteration:** 2 of max 5
**Scope:** Branch `epic/EP-039-config-surface-simplification` vs `main` (106 files, +2825/−592 lines). Delta focus: commit `868447f` (REQ-39.019 parity table tests). Full change set re-verified.
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0 | Nit: 1 | Suggestion: 1
**Gate:** Pass

### Summary

Iteration 1 finding **F-001** (Minor) is **resolved** in `868447f`. New table-driven parity tests satisfy REQ-39.019 and plan task 4.3: `TestEP039_Parity_VectorSearchTools` and `TestEP039_Parity_SQLiteReliability` (3 rows each) in `internal/config/ep039_parity_test.go`; `TestEP039_Parity_ToolResultPromptBytes` (4 rows) in `internal/core/ep039_parity_test.go` exercises `toolResultPromptBytesFromConfig` and end-to-end truncation markers. No new Blocker/Major/Medium/Minor findings on re-review of the branch.

**F-003** (Suggestion, iteration 1) is **addressed** by the core parity test. **F-002** (Nit) and **F-004** (Suggestion) remain non-blocking carry-overs.

### Resolved findings (from iteration 1)

| ID | Was | Resolution |
|----|-----|------------|
| F-001 | Minor | `868447f` adds `TestEP039_Parity_*` with ≥3 rows per workstream; golden `want*` baselines document pre-EP-039 resolved values; vector via `VectorSearchToolSettings`, SQLite via `mergeSQLiteStoreReliability` + `ToPolicy()`, truncation via `toolResultPromptBytesFromConfig` + handler truncation. |
| F-003 | Suggestion | Covered by `TestEP039_Parity_ToolResultPromptBytes` (nil artifacts, 8192, 4096, 16384). |

### Findings

| ID | Severity | Location | Issue | Recommendation |
|----|----------|----------|-------|----------------|
| F-002 | **Nit** | `internal/core/run.go`, `internal/core/handler_llm.go` | Unchanged from iteration 1: `maxToolResultPromptBytes` retained as fallback when `tool_output_artifacts` absent. | Optional one-line comment documenting pre-EP-039 fallback behaviour. |
| F-004 | **Suggestion** | `internal/config/ep039_traceability_test.go` | Unchanged from iteration 1: spot-check of 3 fixtures; broader glob optional. | Optional glob of migrated valid fixtures or document existing `config_test.go` coverage. |

### Plan alignment

| Area | Plan / REQ | Status |
|------|------------|--------|
| Phase 4.3 parity tests | Task 4.3, REQ-39.019 | ✅ Complete (was F-001) |
| All other phases | Iteration 1 | ✅ Unchanged — still complete |
| Scope guards | REQ-39.023–024, AC-39.014 | ✅ Re-verified |

### Test / verification

- `make check` — **PASS** (exit 0, 2026-05-31). Race, vet, lint, vuln, module boundaries, integration all green.
- Parity: `go test ./internal/config/... -run TestEP039_Parity -count=1`; `go test ./internal/core/... -run TestEP039_Parity -count=1`.
- Iteration 1 regression suite (legacy rejection, DRY merge, artifacts, truncation) — still present; no regressions observed.

### Residual risks / follow-ups

- Operator config migration (REQ-39.025 / AC-39.015): manual; not verified in-repo (gitignored).
- `ArtifactDirectory` still resolver-only per design.
- Non-blocking: F-002, F-004. Optional: parity test comments say `AC-39.019` but epic trace id is REQ-39.019; SQLite parity could add `// Covers AC-39.009`.
