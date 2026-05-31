---
artefact: ep-system-design-review
epic_id: EP-043
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
updated_at: 2026-05-31
---

# Architecture Review — EP-043 Test suite organization

**Reviewer:** AI Agent (delegated pipeline stage 7)

---

## Current Gate Summary

Gate: Pass
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 0 | Minor 0
Open findings: None (Nit/Suggestion items below do not block the gate)
Next action: Proceed to stage 8

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

The system design is appropriately scoped for a test-only maintainability epic: it names concrete target files, a handler split map aligned with existing `handler_epNNN_test.go` precedent, a config fixture helper contract, and consolidation of per-epic traceability guards. Branch baseline confirms the pain points (`handler_test.go` ~1826 LOC, `config_test.go` ~802 LOC with multiple inline JSON blocks, four `epNNN_traceability_test.go` files in `internal/core`). Requirements REQ-43.001–010 and acceptance criteria AC-43.001–007 are addressable from the design without product behaviour changes. The document is intentionally minimal versus full product epics; for zero-runtime-impact test reorganization that is sufficient for stage 8. No Blocker/Major/Medium/Minor findings; stage 8 may proceed.

**Verdict:** Pass gate

### Strengths

- **Accurate baseline:** `internal/core/handler_test.go` is 1826 lines with domain-clustered `TestHandleMessage_*` groups (session ~L1601+, tools ~L711+, LLM/router ~L183+, memory/vector ~L463+); the four-file split map matches natural test themes and EP-038 precedent (`handler_ep017_test.go`, `handler_ep034_regression_test.go`).
- **Shared-deps pattern exists:** `handler_test_deps_test.go` already centralizes `testHandlerDeps` → `conversationHandler` construction (EP-040); design correctly extends to `handler_test_helpers.go` only when cross-file sharing is needed (REQ-43.003).
- **Config migration target is feasible:** `config_test.go` contains ≥7 large inline `content := \`{...\}\`` blocks (e.g. L169, L293, L460, L496, L630, L695, L762) plus many `testdata/` loads; extracting ten blocks via `loadConfigFixture` satisfies REQ-43.005 without rewriting the entire file.
- **Traceability guard scope:** Design names `ep034_traceability_test.go`, `ep038_traceability_test.go`, and `ep040_traceability_test.go` for table-driven merge into `architecture_guards_test.go`; ep034 alone carries eight invariant tests suitable for table rows (REQ-43.006, REQ-43.007).
- **Verification contract:** Design and [ep-implementation-plan.md](ep-implementation-plan.md) both gate on `make check` (coverage ≤0.5% drop) and `make validate` (REQ-43.008–010, AC-43.005–007).
- **Zero product scope:** Overview and scope explicitly forbid assertion or behaviour changes; file moves within `package core` / `package config_test` preserve test semantics (REQ-43.001, AC-43.002).

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| N-001 | Nit | Stage 7 structural checklist sections (module boundaries, components table, error handling, risks) are omitted; design is file-map only. | `ep-system-design.md` (35 lines) vs stage 7 template. | Acceptable for test-only epic; stage 8 plan may add a one-row risks note (tempDir path interpolation in migrated fixtures). |
| N-002 | Nit | `loadConfigFixture` signature in design returns `( *Config, string path )`; REQ-43.004 specifies `*config.Config` only. | `ep-system-design.md` Config helpers; `ep-requirements.md` REQ-43.004. | Stage 8: pick one signature; returning loaded path is optional for debugging. |
| N-003 | Nit | `ep-acceptance-criteria.md` index lists AC-43.005 but has no `### AC-43.005` body (coverage floor). | AC index L21 vs sections ending at AC-43.004 then AC-43.006. | Add AC-43.005 body in stage 5 polish or map explicitly in stage 8 verification checklist. |
| S-001 | Suggestion | `ep041_traceability_test.go` exists (3 tests, tier-assembler invariants) but is not listed for merge; scope mentions only ep034/ep038. | `internal/core/ep041_traceability_test.go`. | Keep ep041 separate unless operator wants full consolidation; document decision in stage 8 task 1.3. |
| S-002 | Suggestion | Inline JSON tests using `t.TempDir()` and string-interpolated paths may need parameterized fixtures rather than static `testdata/` files. | `config_test.go` L460–538, L630–788. | Stage 8: migrate static blocks first; leave path-interpolated cases on helper-wrapped temp files or add fixture template + replace. |
| S-003 | Suggestion | C4 C1 diagram lives in `ep-requirements.md` only; design has no architecture diagram reference. | `diagrams/c4-context.puml`; design Overview. | Optional cross-link in design Overview; not required for implementation. |

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

_None._

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Domain-file split over epic-file split | Follows EP-038 `handler_ep017_test.go` precedent; groups by behaviour theme not delivery epic (KISS, navigability). |
| `handler_test_helpers.go` only when shared | Avoid premature abstraction; `handler_test_deps_test.go` already covers constructor wiring (REQ-43.003). |
| Table-driven `architecture_guards_test.go` | Reduces four small traceability files to one maintainable invariant list (REQ-43.006). |
| Incremental config JSON migration (≥10) | Scope defers full `config_test.go` rewrite; meets REQ-43.005 without blocking epic. |
| Delete merged traceability files after merge | Prevents duplicate guard tests; validator must still resolve `// Covers AC-xx.xxx` on moved functions (REQ-43.007). |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-43.008 Coverage floor | Implementation plan baseline ~76%; `make check` diff | OK |
| REQ-43.009 make check | Design + plan task 1.4 | OK |
| REQ-43.010 make validate | Design + plan task 1.4; EP-039–043 registration | OK |
| Security | Test-only; no prod surface change | OK |
| Testability | Moves preserve package-level test access | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ File moves and one helper; no new subsystems |
| Fail fast | ✅ `make check` / `make validate` gates |
| Security | ✅ No config or runtime behaviour change |
| Testability | ✅ Split improves maintainability without dropping tests |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md) — REQ-43.001–010
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — AC-43.001–007 (see N-003 for AC-43.005 body gap)
- **Scope:** [ep-scope.md](ep-scope.md)

### Requirement traceability verification (this iteration)

| REQ | Design coverage | Code baseline aligned |
|-----|-----------------|------------------------|
| REQ-43.001 | File split map (4 domain files) | `handler_test.go` 1826 LOC, ~49 Test* functions |
| REQ-43.002 | Overview ≤600 LOC target | Main file oversized; split required |
| REQ-43.003 | `handler_test_helpers.go` | `handler_test_deps_test.go` exists |
| REQ-43.004 | `config_test_helpers.go` + `loadConfigFixture` | No helper file yet |
| REQ-43.005 | Migrate ≥10 inline JSON | ≥7 large inline blocks in `config_test.go` |
| REQ-43.006 | `architecture_guards_test.go` merge | `ep034/038/040_traceability_test.go` present |
| REQ-43.007 | Preserve `// Covers AC-xx.xxx` on moves | Comments throughout `handler_test.go` |
| REQ-43.008 | make check coverage floor | Plan baseline ~76% |
| REQ-43.009 | make check | Standard repo gate |
| REQ-43.010 | make validate | Standard repo gate |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-043/ep-system-design-review.md [gate=pass, iteration 1, blocker:0 major:0 medium:0 minor:0]`
