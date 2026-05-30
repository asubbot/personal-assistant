---
artefact: ep-system-design-review
epic_id: EP-035
status: draft
source_of_truth: true
gate: pass
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 0
  minor: 0
next_action: proceed_to_stage_8
updated_at: 2026-05-30
---

# Architecture Review — EP-035 Consolidate small internal packages

**Reviewer:** AI Agent (delegated stage 7 reviewer, fresh context)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-30
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings:
- None (Blocker/Major/Medium/Minor all zero). Two non-blocking items recorded (1 Suggestion, 1 Nit).
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-30
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

This is a tightly-scoped structural refactor and the design is accurate, complete, and implementable. I verified the design against the actual codebase: the seven-file importer-rewrite list is complete, the relocated `-race` test genuinely preserves race coverage under `make check`, and the byte-identical preservation strategy traces correctly to the requirements and AC. No build-breaking gaps, no missed importer, no behavioural drift, and no `config.json` change. Two non-blocking polish items are noted as Suggestion/Nit.

**Verdict:** Pass gate

### Strengths

- **Importer list is complete and verified.** A repo grep for `pa/internal/(promptmarkers|systemprompt)` across `cmd/`, `internal/`, `tests/` returns exactly the seven external importers in the design's table (`internal/core/handler.go`, `internal/core/system_tail.go`, `internal/core/handler_test.go`, `internal/tools/write_memory.go`, `internal/runtimeskills/package.go`, `tests/integration/runtime_skills_handler_test.go`, `tests/integration/runtime_skills_config_test.go`) plus the two legacy files being merged (`internal/systemprompt/systemprompt.go` and `systemprompt_test.go`, which the merge itself absorbs). No importer is missed → no latent build break (REQ-35.013, REQ-35.014; AC-35.013, AC-35.014).
- **Race coverage is genuinely preserved.** The user-flagged risk does not materialise: `Makefile` defines `test-race: go test -race -tags=integration ./...` and `check: ... test-race ...`. Because `make test-race` supplies `-tags=integration`, the relocated file carrying `//go:build integration` in `tests/integration` **is** compiled and run under `-race` as part of `make check` (REQ-35.005, REQ-35.016; AC-35.005, AC-35.016). The design's risk row ("Race test omitted from `integration` tag builds → mandatory `//go:build integration` + run `make test-race` in CI") is correct.
- **Relocation target is sound.** Existing `tests/integration` files use `package integration_test` with `//go:build integration` (confirmed in `doc.go` and `runtime_skills_config_test.go`); the relocated helpers (`runVectorWriter`, `runJobsWriter`, `isBusyOrLocked`, `iterations`) have no name collisions in that package, and the listed imports (`jobs`, `sqlitepragma`, `vector/sqlite`) match the current `internal/reliability/concurrent_write_test.go` body.
- **Security invariants handled by verbatim copy.** `TrustPolicy` (`systemprompt.go`) and the six marker constants (`promptmarkers.go`) are copy-pasted verbatim; the only in-`wrap.go` change is dropping the `promptmarkers.` qualifier (same package after merge). Byte-identity is asserted by unit tests tracing to AC-35.008/009.
- **No behaviour / no config change.** Constraints table and Error-handling table preserve handler assembly order, marker rejection, and SQLite PRAGMA policy; design explicitly scopes out `internal/config` edits (REQ-35.015, REQ-35.017–020).
- **Full traceability.** All 20 requirements (REQ-35.001–020) appear in the traceability table with AC and design-section mappings; structural sections (overview, C4 diagram, module boundaries, components/interfaces, data models, error handling, testing strategy, migration sequencing, risks, traceability) are all present and the referenced diagram assets exist.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

#### Suggestion (non-blocking)

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| S-001 | Byte-identity test reference is not pinned to an *independent* source. | Components/Constraints describe a "unit test compares to frozen pre-EP-035 reference." If implemented as a comparison of `prompt.TrustPolicy` to itself (or to the same in-package literal it copies), the test is tautological and would not catch an accidental edit — the core security invariant of this epic. AC-35.008/009 already permit a golden/VCS-parent snapshot. | In stage 8/9, capture the reference as an *independent* literal (golden string or copied constant in the test file, not a reference to the production constant) so the byte-identity assertion is meaningful. Worth one explicit sentence in the design's Constraints row. |

#### Nit (non-blocking)

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| N-001 | Mildly contradictory qualifier wording for in-package test references. | The `wrap_test.go` row says "qualify markers as `prompt.BeginContext` etc. (same package)"; within package `prompt`, references must be **unqualified** (`BeginContext`). The "Qualified identifier rewrites" list already clarifies this ("or unqualified `BeginContext` in `prompt` tests"), so it is purely cosmetic. | Drop the `prompt.` prefix in the `wrap_test.go` row to avoid implying a same-package self-qualification (which would not compile). |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-35.015 No config.json change | Constraints table; design scopes out `internal/config`; AC-35.015 manual diff check | OK |
| REQ-35.016 `make check` passes | Testing strategy + Migration step 7; Makefile `check` chain verified | OK |
| REQ-35.017 Preserve prompt assembly | Constraints; verbatim move; handler order unchanged | OK |
| REQ-35.018 Runtime skills marker rejection | Error-handling table; `prompt.TextContainsForbiddenMarkerLine` at same call sites | OK |
| REQ-35.019 Memory indexing marker rejection | Error-handling table; `handler` / `write_memory` | OK |
| REQ-35.020 EP-013 tests retain intent | Tests move to `prompt` / `tests/integration` with unchanged intent | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Verbatim move, single merged import path, no new abstractions |
| Fail fast | ✅ Marker-rejection and busy/locked failure paths preserved |
| Security | ✅ Byte-identical TrustPolicy/markers via verbatim copy (see S-001 for test rigor) |
| Testability | ✅ Unit (prompt) + integration (relocated race test) levels defined |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
