# EP-023 — Implementation plan

**Purpose:** Execute [pipeline.spec.md](../../ai-sdlc/specification/pipeline.spec.md) stage 9 tasks for atomic catalog writes.

**References:** [ep-scope.md](ep-scope.md) · [ep-requirements.md](ep-requirements.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-system-design.md](ep-system-design.md) · [ep-system-design-review.md](ep-system-design-review.md) · [strategy.md](../../strategy.md)

**Checkpoints:** Run `make check` and `./bin/validate EP-023` before declaring the epic complete.

---

## Tasks

- [x] **1** — Extend `internal/toolcatalog` atomic persistence
  - Read snapshot bytes before replace; marshal updated catalog; write temp in same directory; `Sync` temp file; `rename` to target; `Sync` parent directory after rename; call `Load` on catalog path; on `Load` error restore snapshot with the same atomic writer.
  - Optional test hooks: post-marshal body transform, rename hook, sync observation hook for [AC-23.002](ep-acceptance-criteria.md#ac-23-002).
  - _Requirements:_ [REQ-23.001](ep-requirements.md#catalog-file-durability)–[REQ-23.004](ep-requirements.md#catalog-file-durability), [REQ-23.002](ep-requirements.md#catalog-file-durability)
  - _Acceptance Criteria:_ [AC-23.001](ep-acceptance-criteria.md#ac-23-001)–[AC-23.004](ep-acceptance-criteria.md#ac-23-004), [AC-23.002](ep-acceptance-criteria.md#ac-23-002)
  - **Verification:** `go test ./internal/toolcatalog/... -count=1` passes.

- [x] **2** — Reorder and harden `CreateToolTool.lockedCreate` (`internal/tools/create_tool.go`)
  - Read snapshot before append; call `AppendToolToCatalogFile`; only then `c.catalog.Tools[id] = newTool`; then `UpsertToolEmbedding` when embedder and index non-nil; on upsert error delete map entry and restore catalog file from snapshot via `toolcatalog` helper.
  - Stop treating embedding failure as success when embedder is configured ([REQ-23.007](ep-requirements.md#runtime-catalog-and-tool-index-consistency)).
  - _Requirements:_ [REQ-23.005](ep-requirements.md#runtime-catalog-and-tool-index-consistency)–[REQ-23.007](ep-requirements.md#runtime-catalog-and-tool-index-consistency)
  - _Acceptance Criteria:_ [AC-23.005](ep-acceptance-criteria.md#ac-23-005)–[AC-23.007](ep-acceptance-criteria.md#ac-23-007)
  - **Verification:** `go test ./internal/tools/... -count=1 -run CreateTool` passes.

- [x] **3** — Deterministic failure tests
  - `internal/toolcatalog`: short write / corrupt body / rename failure cases with `// Covers AC-23.008` (and supporting ACs as needed).
  - `internal/tools`: fake embedder error path with `// Covers AC-23.007`.
  - Sync hook test with `// Covers AC-23.002` if hooks added.
  - _Requirements:_ [REQ-23.008](ep-requirements.md#verification-and-operator-documentation), [REQ-23.010](ep-requirements.md#verification-and-operator-documentation)
  - _Acceptance Criteria:_ [AC-23.008](ep-acceptance-criteria.md#ac-23-008), [AC-23.002](ep-acceptance-criteria.md#ac-23-002)
  - **Verification:** tests pass; comments satisfy `./bin/validate EP-023`.

- [x] **4** — Operator documentation
  - Add **Tool catalog durability (create_tool)** subsection to repository root [README.md](../../../README.md): atomic replace, `Sync` sequence, post-write `Load`.
  - _Requirements:_ [REQ-23.009](ep-requirements.md#verification-and-operator-documentation)
  - _Acceptance Criteria:_ [AC-23.009](ep-acceptance-criteria.md#ac-23-009)
  - **Verification:** link from README exists; wording matches AC.

- [x] **5** — Quality gate
  - Run `make check` and `make build && ./bin/validate EP-023`.
  - _Requirements:_ [REQ-23.011](ep-requirements.md#verification-and-operator-documentation)
  - _Acceptance Criteria:_ [AC-23.010](ep-acceptance-criteria.md#ac-23-010)
  - **Verification:** both commands exit 0.
