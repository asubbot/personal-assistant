---
artefact: ep-implementation-plan
epic_id: EP-039
status: draft
source_of_truth: true
updated_at: 2026-05-31
---

# EP-039 — Implementation plan

Pipeline stage 8 output for **EP-039 Config surface simplification**.

**Branch:** `epic/EP-039-config-surface-simplification`

**Design review:** [ep-system-design-review.md](ep-system-design-review.md) — gate **pass**, iteration 1

**Out of scope:** `tools.selection` changes; handler file reorganization; new artifact retention sweeper features.

---

## Tasks

### Phase 1 — Root key + legacy rejection scaffolding

- [ ] **1.1** Add `sqlite_store_defaults` to `configRootJSONKeys` (sorted, between `runtime_skills` and `telegram`). Add `SQLiteStoreDefaultsConfig` stub on `Config`.
  - _REQ:_ [REQ-39.012](ep-requirements.md#req-39-012--require-sqlite_store_defaults)
  - _AC:_ [AC-39.008](ep-acceptance-criteria.md#ac-39-008)
  - **Verification:** `go test ./internal/config/... -run ConfigRootJSONKeys -count=1`

- [ ] **1.2** Add `rejectLegacyVectorSearchToolsShape` and `rejectLegacySQLiteReliabilityShape` in `load.go`; call from `prepareConfig` before unmarshal. Add testdata: `vector_search_tools_legacy_rejected.json`, `sqlite_reliability_legacy_rejected.json`.
  - _REQ:_ [REQ-39.002](ep-requirements.md#req-39-002--reject-legacy-vector_search_tools-shape), [REQ-39.015](ep-requirements.md#req-39-015--reject-redundant-legacy-reliability-only-shape), [REQ-39.020](ep-requirements.md#req-39-020--negative-legacy-fixtures)
  - _AC:_ [AC-39.002](ep-acceptance-criteria.md#ac-39-002), [AC-39.010](ep-acceptance-criteria.md#ac-39-010)
  - **Verification:** `go test ./internal/config/... -run Legacy -count=1`

---

### Phase 2 — SQLite reliability DRY

- [ ] **2.1** Implement `SQLiteStoreDefaultsConfig`, `SQLiteStoreReliabilityOverride`, `mergeSQLiteStoreReliability`; update `ValidateVectorStoreReliability` / `ValidateJobsStoreReliability` to merge defaults + override.
  - _REQ:_ [REQ-39.012](ep-requirements.md#req-39-012--require-sqlite_store_defaults)–[REQ-39.014](ep-requirements.md#req-39-014--effective-pragma-parity)
  - _AC:_ [AC-39.008](ep-acceptance-criteria.md#ac-39-008), [AC-39.009](ep-acceptance-criteria.md#ac-39-009)
  - **Verification:** `go test ./internal/config/... -run SQLite -count=1`

- [ ] **2.2** Migrate all `internal/config/testdata/*.json`, `config.examples/`, integration configs: add `sqlite_store_defaults`; shrink store blocks to `foreign_keys` only.
  - _REQ:_ [REQ-39.016](ep-requirements.md#req-39-016--update-repository-configs)
  - _AC:_ [AC-39.011](ep-acceptance-criteria.md#ac-39-011)
  - **Verification:** `go test ./internal/config/... -count=1` (may fail until Phase 3 vector migrate — run again after 3.2)

---

### Phase 3 — vector_search_tools DRY

- [ ] **3.1** Replace `VectorSearchToolsConfig` with defaults + override shape; implement `mergeVectorSearchTool`; update `validateVectorSearchTools` and `VectorSearchToolSettings`.
  - _REQ:_ [REQ-39.001](ep-requirements.md#req-39-001--require-defaults-and-per-tool-overrides)–[REQ-39.006](ep-requirements.md#req-39-006--runtime-parity-for-vector-tools)
  - _AC:_ [AC-39.001](ep-acceptance-criteria.md#ac-39-001), [AC-39.004](ep-acceptance-criteria.md#ac-39-004)
  - **Verification:** `go test ./internal/config/... -run VectorSearch -count=1`

- [ ] **3.2** Migrate `tools.vector_search_tools` in all testdata/examples to defaults+overrides shape.
  - _REQ:_ [REQ-39.016](ep-requirements.md#req-39-016--update-repository-configs)
  - _AC:_ [AC-39.011](ep-acceptance-criteria.md#ac-39-011)
  - **Verification:** `go test ./internal/config/... -count=1`

---

### Phase 4 — tool_output_artifacts typed + core wire

- [ ] **4.1** Add `ToolOutputArtifactsConfig` on `ToolsConfig`; `validateToolOutputArtifacts` + nested key whitelist; reject unknown artifact keys.
  - _REQ:_ [REQ-39.007](ep-requirements.md#req-39-007--typed-tooloutputartifactsconfig)–[REQ-39.008](ep-requirements.md#req-39-008--validate-artifact-fields), [REQ-39.011](ep-requirements.md#req-39-011--reject-unknown-artifact-keys)
  - _AC:_ [AC-39.005](ep-acceptance-criteria.md#ac-39-005), [AC-39.007](ep-acceptance-criteria.md#ac-39-007)
  - **Verification:** `go test ./internal/config/... -run ToolOutput -count=1`

- [ ] **4.2** Core: add `toolResultPromptBytes int` on `conversationHandler`; set from config in `newRunConversationHandler`; convert `truncateToolResultForPrompt` to use handler field; add `ArtifactDirectory(cfg)` helper in config package.
  - _REQ:_ [REQ-39.009](ep-requirements.md#req-39-009--wire-tool_result_prompt_bytes), [REQ-39.010](ep-requirements.md#req-39-010--wire-artifact-directory), [REQ-39.023](ep-requirements.md#req-39-023--limit-core-changes)
  - _AC:_ [AC-39.006](ep-acceptance-criteria.md#ac-39-006)
  - **Verification:** `go test ./internal/core/... -run 'truncat|ToolResult' -count=1`

- [ ] **4.3** Migrate operator block shape in examples (artifact block unchanged fields); add parity + negative tests with `// Covers AC-39.xxx`.
  - _REQ:_ [REQ-39.019](ep-requirements.md#req-39-019--equivalent-config-parity-tests)
  - **Verification:** `make check`

---

### Phase 5 — Documentation

- [ ] **5.1** Update `docs/configuration.md` with migration tables for all three config areas.
  - _REQ:_ [REQ-39.017](ep-requirements.md#req-39-017--document-migration)
  - _AC:_ [AC-39.012](ep-acceptance-criteria.md#ac-39-012)
  - **Verification:** Manual doc review

---

### Phase 6 — Final verification

- [ ] **6.1** Run `make check`; fix any failures; add `ep039_traceability_test.go` if needed for AC coverage.
  - _REQ:_ [REQ-39.021](ep-requirements.md#req-39-021--make-check-passes)
  - _AC:_ [AC-39.013](ep-acceptance-criteria.md#ac-39-013), [AC-39.014](ep-acceptance-criteria.md#ac-39-014), [AC-39.015](ep-acceptance-criteria.md#ac-39-015)
  - **Verification:** `make check`; `./bin/validate EP-039` when registered

**Checkpoint (final):** All tasks checked; ready for stage 10 code review (delegated).
