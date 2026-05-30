---
artefact: ep-implementation-plan
epic_id: EP-035
status: draft
source_of_truth: true
updated_at: 2026-05-30
---

# EP-035 — Implementation plan

Pipeline stage 8 output for **Consolidate small internal packages**. Ordered tasks follow [ep-system-design.md](ep-system-design.md) migration sequencing so **`make check` stays green** after each step. Stage 7 gate: **Pass** ([ep-system-design-review.md](ep-system-design-review.md)).

**Related artefacts**

- Scope: [ep-scope.md](ep-scope.md)
- Requirements: [ep-requirements.md](ep-requirements.md)
- Acceptance criteria: [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- System design: [ep-system-design.md](ep-system-design.md)
- Design review: [ep-system-design-review.md](ep-system-design-review.md)
- Test strategy: [strategy.md](../../strategy.md)

**Execution notes (stage 7 non-blocking)**

- **S-001:** Byte-identity tests MUST compare production constants to **independent** expected literals in the test file (golden strings captured once from pre-merge `internal/systemprompt` / `internal/promptmarkers`), not to themselves or only to the same in-package symbol.
- **N-001:** Inside package `prompt`, tests and `wrap.go` use **unqualified** marker identifiers (`BeginContext`, not `prompt.BeginContext`).

---

## Tasks

### Phase 1 — Remove `internal/logging`

- [x] **1. Delete `internal/logging` stub package**
  - Confirm zero importers: `rg 'pa/internal/logging' --glob '*.go' cmd internal tests` → no matches.
  - Remove `internal/logging/` (only `doc.go` today). LLM logging stays in `internal/llmlog` and `internal/logredact`.
  - _Requirements:_ [REQ-35.001](ep-requirements.md#req-35-001--delete-logging-stub-package), [REQ-35.002](ep-requirements.md#req-35-002--no-logging-package-imports)
  - _Acceptance Criteria:_ [AC-35.001](ep-acceptance-criteria.md#ac-35-001), [AC-35.002](ep-acceptance-criteria.md#ac-35-002)
  - **Verification:** `make check` exits 0; `test ! -d internal/logging`; `rg 'pa/internal/logging' --glob '*.go' cmd internal tests` → zero matches.

**Checkpoint:** Phase 1 complete — logging path gone; quality gate still green.

---

### Phase 2 — Merge prompt packages (`internal/prompt`)

- [x] **2. Add `internal/prompt` production sources (`markers.go`, `wrap.go`)**
  - Create `internal/prompt` with package comment: `// Package prompt defines EP-013 canonical system-block markers, trust policy, and wrap helpers.`
  - Copy `internal/promptmarkers/markers.go` → `internal/prompt/markers.go` **verbatim** (package name `prompt` only).
  - Copy `internal/systemprompt/systemprompt.go` → `internal/prompt/wrap.go` **verbatim** except: package `prompt`; replace `promptmarkers.BeginContext` (etc.) with unqualified `BeginContext` (same package); drop the `promptmarkers` import.
  - Legacy packages remain; no production file imports `pa/internal/prompt` yet.
  - _Requirements:_ [REQ-35.007](ep-requirements.md#req-35-007--provide-merged-internalprompt-package), [REQ-35.008](ep-requirements.md#req-35-008--byte-identical-trust-policy), [REQ-35.009](ep-requirements.md#req-35-009--byte-identical-marker-constants), [REQ-35.010](ep-requirements.md#req-35-010--equivalent-forbidden-marker-validation), [REQ-35.011](ep-requirements.md#req-35-011--equivalent-block-wrap-helpers)
  - _Acceptance Criteria:_ [AC-35.007](ep-acceptance-criteria.md#ac-35-007), [AC-35.008](ep-acceptance-criteria.md#ac-35-008), [AC-35.009](ep-acceptance-criteria.md#ac-35-009)
  - **Verification:** `go build ./internal/prompt/...` succeeds; `go vet ./internal/prompt/...` clean.

- [x] **3. Add `internal/prompt` unit tests with independent byte-identity goldens (S-001)**
  - Before deleting legacy packages, capture **once** into the test file(s) as literal `expected…` constants (not references to `TrustPolicy` / `BeginContext` exports):
    - Full `TrustPolicy` string from `internal/systemprompt/systemprompt.go`.
    - All six marker line strings from `internal/promptmarkers/markers.go`.
  - Add tests asserting `TrustPolicy == expectedTrustPolicy` and each marker const equals its `expected…` literal (**non-tautological** per S-001 / [AC-35.008](ep-acceptance-criteria.md#ac-35-008), [AC-35.009](ep-acceptance-criteria.md#ac-35-009)).
  - Move `internal/promptmarkers/markers_test.go` → `internal/prompt/markers_test.go` (package `prompt`; keep table-driven forbidden-line cases).
  - Move `internal/systemprompt/systemprompt_test.go` → `internal/prompt/wrap_test.go` (package `prompt`; use **unqualified** `BeginContext`, `WrapToolInstructions`, etc. — **N-001**, no `prompt.` prefix in same-package tests).
  - Preserve wrap golden / empty-inner cases; behaviour must match pre-merge `systemprompt` outputs ([AC-35.010](ep-acceptance-criteria.md#ac-35-010), [AC-35.011](ep-acceptance-criteria.md#ac-35-011)).
  - _Requirements:_ [REQ-35.010](ep-requirements.md#req-35-010--equivalent-forbidden-marker-validation), [REQ-35.011](ep-requirements.md#req-35-011--equivalent-block-wrap-helpers), [REQ-35.020](ep-requirements.md#req-35-020--ep-013-prompt-tests-retain-intent)
  - _Acceptance Criteria:_ [AC-35.008](ep-acceptance-criteria.md#ac-35-008), [AC-35.009](ep-acceptance-criteria.md#ac-35-009), [AC-35.010](ep-acceptance-criteria.md#ac-35-010), [AC-35.011](ep-acceptance-criteria.md#ac-35-011), [AC-35.020](ep-acceptance-criteria.md#ac-35-020)
  - **Verification:** `go test ./internal/prompt/...` passes (legacy packages still present).

- [x] **4. Rewrite all seven importers to `pa/internal/prompt`**
  - Single import `pa/internal/prompt` per file; update qualified calls (`promptmarkers.*` / `systemprompt.*` → `prompt.*` at call sites outside package `prompt`).
  - Files (complete list per design):
    1. `internal/core/handler.go`
    2. `internal/core/system_tail.go`
    3. `internal/core/handler_test.go`
    4. `internal/tools/write_memory.go`
    5. `internal/runtimeskills/package.go`
    6. `tests/integration/runtime_skills_handler_test.go`
    7. `tests/integration/runtime_skills_config_test.go`
  - Do **not** leave any file importing both legacy and new prompt packages.
  - _Requirements:_ [REQ-35.014](ep-requirements.md#req-35-014--update-prompt-package-importers), [REQ-35.017](ep-requirements.md#req-35-017--preserve-system-prompt-assembly), [REQ-35.018](ep-requirements.md#req-35-018--preserve-runtime-skills-marker-rejection), [REQ-35.019](ep-requirements.md#req-35-019--preserve-memory-indexing-marker-rejection)
  - _Acceptance Criteria:_ [AC-35.014](ep-acceptance-criteria.md#ac-35-014), [AC-35.017](ep-acceptance-criteria.md#ac-35-017), [AC-35.018](ep-acceptance-criteria.md#ac-35-018), [AC-35.019](ep-acceptance-criteria.md#ac-35-019)
  - **Verification:** `go test ./internal/core/... ./internal/tools/... ./internal/runtimeskills/...` passes; `go test -tags=integration ./tests/integration/... -run 'Runtime|Skills|Handler'` (or full integration package) passes.

- [x] **5. Delete legacy `internal/promptmarkers` and `internal/systemprompt`**
  - **Depends on Task 4** — land in the **same commit/PR slice** as Task 4 so the tree never imports removed packages.
  - Remove directories `internal/promptmarkers/` and `internal/systemprompt/` entirely.
  - _Requirements:_ [REQ-35.012](ep-requirements.md#req-35-012--delete-legacy-prompt-packages), [REQ-35.013](ep-requirements.md#req-35-013--no-legacy-prompt-package-imports)
  - _Acceptance Criteria:_ [AC-35.012](ep-acceptance-criteria.md#ac-35-012), [AC-35.013](ep-acceptance-criteria.md#ac-35-013)
  - **Verification:** `test ! -d internal/promptmarkers && test ! -d internal/systemprompt`; `rg 'pa/internal/(promptmarkers|systemprompt)' --glob '*.go' cmd internal tests` → zero matches; `make check` exits 0.

**Checkpoint:** Phase 2 complete — single `internal/prompt` package; legacy prompt paths gone; `make check` green.

---

### Phase 3 — Relocate reliability test

- [x] **6. Add `tests/integration/concurrent_write_test.go`**
  - Copy body from `internal/reliability/concurrent_write_test.go` unchanged (helpers `runVectorWriter`, `runJobsWriter`, `isBusyOrLocked`, `iterations` const, test name `TestConcurrentWrites_NoBusyErrors`).
  - File header: `//go:build integration`; package `integration_test` (match `tests/integration/doc.go`).
  - Keep trace comment `// Covers AC-22.010`; add `// Covers AC-35.005`.
  - Imports: `pa/internal/jobs`, `pa/internal/sqlitepragma`, `pa/internal/vector/sqlite` (alias `vectorsqlite` as today), plus stdlib as in source.
  - _Requirements:_ [REQ-35.004](ep-requirements.md#req-35-004--relocate-concurrent-write-test), [REQ-35.005](ep-requirements.md#req-35-005--preserve-ac-22-010-race-test-intent)
  - _Acceptance Criteria:_ [AC-35.004](ep-acceptance-criteria.md#ac-35-004), [AC-35.005](ep-acceptance-criteria.md#ac-35-005)
  - **Verification:** `go test -race -tags=integration ./tests/integration/... -run TestConcurrentWrites_NoBusyErrors` exits 0.

- [x] **7. Delete `internal/reliability`**
  - **Depends on Task 6** (relocated test must exist first).
  - Remove `internal/reliability/` directory.
  - _Requirements:_ [REQ-35.003](ep-requirements.md#req-35-003--delete-reliability-test-package)
  - _Acceptance Criteria:_ [AC-35.003](ep-acceptance-criteria.md#ac-35-003)
  - **Verification:** `test ! -d internal/reliability`; `rg 'pa/internal/reliability' --glob '*.go' cmd internal tests` → zero matches; `make check` exits 0.

- [x] **8. Update `docs/configuration.md` reliability test path**
  - In § Local SQLite stores, replace reference to `internal/reliability` with `tests/integration` (and note `-race` / `integration` tag as appropriate).
  - _Requirements:_ [REQ-35.006](ep-requirements.md#req-35-006--update-reliability-test-documentation-path)
  - _Acceptance Criteria:_ [AC-35.006](ep-acceptance-criteria.md#ac-35-006)
  - **Verification:** `rg 'internal/reliability' docs/configuration.md` → zero matches; `rg 'tests/integration' docs/configuration.md` matches the concurrent-writer sentence.

**Checkpoint:** Phase 3 complete — race test under integration tag; reliability package removed.

---

### Phase 4 — Final verification

- [x] **9. Epic quality gates and removal grep**
  - Confirm EP-035 did not touch product `config.json`, examples, or `internal/config` validation ([REQ-35.015](ep-requirements.md#req-35-015--no-configjson-changes)).
  - Run full gate: `make check` from repo root ([REQ-35.016](ep-requirements.md#req-35-016--quality-gate-passes)).
  - Explicit race smoke (also covered by `make check` via `test-race`): `go test -race -tags=integration ./tests/integration/... -run TestConcurrentWrites_NoBusyErrors`.
  - Final grep — zero references to all removed packages:
    ```bash
    rg 'pa/internal/(logging|reliability|promptmarkers|systemprompt)' --glob '*.go' cmd internal tests
    ```
  - _Requirements:_ [REQ-35.015](ep-requirements.md#req-35-015--no-configjson-changes), [REQ-35.016](ep-requirements.md#req-35-016--quality-gate-passes), [REQ-35.020](ep-requirements.md#req-35-020--ep-013-prompt-tests-retain-intent)
  - _Acceptance Criteria:_ [AC-35.015](ep-acceptance-criteria.md#ac-35-015), [AC-35.016](ep-acceptance-criteria.md#ac-35-016), [AC-35.020](ep-acceptance-criteria.md#ac-35-020)
  - **Verification:** `make check` and explicit `go test -race -tags=integration ./tests/integration/... -run TestConcurrentWrites_NoBusyErrors` exit 0; grep → zero matches; `git diff` shows no config schema/validation changes attributable to EP-035.

---

## Dependencies and order

| Task | Depends on | Notes |
|------|------------|-------|
| 1 | — | Independent first step |
| 2 | 1 | New package; legacy still present |
| 3 | 2 | Tests target `internal/prompt` only |
| 4 | 2, 3 | Importer rewrite |
| 5 | 4 | **Same change set as 4** — avoid broken imports |
| 6 | 1–5 (recommended) | Can run after prompt merge; keep `make check` green |
| 7 | 6 | Delete old reliability only after copy |
| 8 | 6 or 7 | Doc-only; any time after 6 |
| 9 | 1–8 | Final gate |

**Out of scope (no tasks):** `internal/lifecyclelog` — unchanged per [ep-scope.md](ep-scope.md).

---

## Checkpoints

1. **After Task 1:** `make check` green; no `internal/logging`.
2. **After Task 3:** `go test ./internal/prompt/...` green with independent byte goldens (S-001).
3. **After Tasks 4+5 (single slice):** no legacy prompt imports; `make check` green.
4. **After Task 7:** no `internal/reliability`; relocated race test passes with `-tags=integration`.
5. **After Task 9:** ready for stage 9 implementation / stage 10 review — all 20 ACs satisfied via tasks above.

**HITL:** Ask the operator if an unexpected importer appears outside the seven-file list or if `make check` fails for reasons outside EP-035 scope.
