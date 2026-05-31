---
artefact: ep-system-design-review
epic_id: EP-040
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

# Architecture Review — EP-040 Handler dependency grouping

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

The system design correctly describes a mechanical dependency grouping refactor: four unexported `handler*Deps` structs, direct grouped access (`h.tools.catalog`), top-level `toolResultPromptBytes` from EP-039, and constructor grouping in `run.go` / `integration_export.go`. All ten REQs are covered in the traceability table; scope guards (no config, no public API, no behaviour change) are explicit. Verification on branch `epic/EP-040-handler-dependency-grouping` shows production handler files and constructors already match the design; remaining work is test struct-literal migration (called out in the design Components table and implementation plan). The document is appropriately minimal for a structural refactor; optional stage 6 depth (TOC, module boundaries, error-handling stub) is noted as Nit/Suggestion only. Stage 8 may proceed.

**Verdict:** Pass gate

### Strengths

- **Accurate target layout:** Four groups plus `toolResultPromptBytes` match [ep-scope.md](ep-scope.md) field lists and REQ-40.001–004; implemented `handler.go` defines `handlerToolDeps`, `handlerMemoryDeps`, `handlerSessionDeps`, `handlerLLMDeps` with the required fields (`internal/core/handler.go:27-74`).
- **KISS access pattern:** Design forbids getters; production code uses `h.llm.router`, `h.session.sessionCfg`, etc., with no remaining flat `h.router` / `h.catalog` in `handler*.go`.
- **Constructor contract:** `newRunConversationHandler` builds four struct literals (`run.go:113-152`); post-build `h.tools.catalog` assignment preserves existing catalog wiring — compatible with design sequencing.
- **Integration parity:** `NewIntegrationConversationHandler` uses the same grouping (`integration_export.go:174-208`), satisfying REQ-40.007 scope.
- **Full REQ traceability:** All REQ-40.001–010 referenced; testing strategy and risks appropriate for compile/test-gated mechanical rename.
- **C4 artefact:** `diagrams/c4-container.png` exists and is embedded; container diagram matches `internal/core` boundary.

### Findings

| Id | Severity | Description | Evidence | Recommendation |
|----|----------|-------------|----------|----------------|
| N-001 | Nit | Struct layout snippet names session fields `cfg` / `store`; requirements and code use `sessionCfg` / `sessionStore`. | `ep-system-design.md` Struct layout vs `handlerSessionDeps` in `handler.go:49-52` | Align design snippet field names in a stage 6 polish pass (cosmetic). |
| N-002 | Nit | Stage 6 recommends an **Error handling** section; design omits it. | `06-system-design.skill.md` §3; `ep-system-design.md` has Risks but no error-handling subsection | One sentence: validation/runtime errors unchanged; existing `checkUserMessage` and LLM error paths retained. |
| N-003 | Nit | No first-level TOC (stage 6 optional but listed in skill). | `ep-system-design.md` | Add short TOC if validating structure; not required for this epic size. |
| S-001 | Suggestion | Post-constructor `catalog` assignment not documented. | `run.go:150-152` sets `h.tools.catalog` after literal | Stage 8 task 1.3: note in plan or design that catalog may be nil in literal then filled from `cfg.ToolCatalog`. |
| S-002 | Suggestion | `ep-acceptance-criteria.md` index lists AC-40.002–004, AC-40.006 but bodies only for AC-40.001, 005, 007. | AC index vs `### AC-…` sections | Stage 8 maps verification to indexed ACs; optional stage 5 fix for missing bodies. |
| S-003 | Suggestion | Optional `ep040_traceability_test.go` in design Testing strategy not yet present. | `ep-implementation-plan.md` task 1.5 | Add during implementation if desired; not required for REQ coverage. |
| S-004 | Suggestion | Many test files still use flat `conversationHandler{ catalog: … }` literals. | `make check` vet failure (`dynamic_tool_selection_test.go:28`); ~90 literals across `handler*_test.go` | Stage 8 task 1.4: migrate to nested literals (`tools: handlerToolDeps{…}`, `llm: handlerLLMDeps{…}`); run `go test ./internal/core/...` then `make check`. |
| S-005 | Suggestion | `integration_export.go` grouped literal omits `toolResultPromptBytes` and `classifier` in visible block — confirm defaults acceptable for integration tests. | `integration_export.go:174-208` | Verify integration tests still set needed deps via params or zero-value behaviour. |

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
| Direct grouped field access (no getters) | KISS; scope and REQ-40.005 |
| `toolResultPromptBytes` stays top-level | Single int from EP-039; avoids a one-field struct |
| Unexported sub-structs only | No new public API; REQ-40.007 |
| Four groups in `handler.go` (not separate package) | Scope guard; matches EP-038 file split |
| Post-assign `tools.catalog` in `run.go` | Preserves existing conditional catalog wiring |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-40.009 no config schema change | Overview + scope | OK |
| REQ-40.010 `make check` | Implementation sequencing §4 | OK (post test migration) |
| REQ-40.008 test parity | Testing strategy | OK (assertions unchanged; literals only) |
| Security | No new surfaces; redactor/logger stay in `handlerLLMDeps` | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ Grouping only; no DI framework |
| Fail fast | ✅ Unchanged error paths; compile catches missed fields |
| Security | ✅ No weakening of redaction or allowlists |
| Testability | ✅ Same test package; struct literal updates mechanical |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md) — REQ-40.001–010
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md) — see S-002 for partial AC bodies
- **Scope:** [ep-scope.md](ep-scope.md)

### Requirement traceability verification (this iteration)

| REQ | Design coverage | Branch alignment |
|-----|-----------------|------------------|
| REQ-40.001 | Struct layout `handlerToolDeps` | `handler.go:27-40` — all listed fields present |
| REQ-40.002 | `handlerMemoryDeps` | `handler.go:42-47` |
| REQ-40.003 | `handlerSessionDeps` | `handler.go:49-52` (`sessionCfg`, `sessionStore`) |
| REQ-40.004 | `handlerLLMDeps` | `handler.go:54-64` |
| REQ-40.005 | File table + access pattern | `handler*.go` use grouped access; no flat dep access in handlers |
| REQ-40.006 | `run.go` in Components | `run.go:113-152` four literals |
| REQ-40.007 | Scope + Components `integration_export.go` | Grouped constructor; public signatures unchanged |
| REQ-40.008 | Testing strategy | Tests not fully migrated (S-004); design expects assertion parity |
| REQ-40.009 | Overview | No config package changes in epic scope |
| REQ-40.010 | Sequencing step 4 | Pending full `make check` after test literal pass |

---

**Signal:** `STAGE_7_COMPLETE: ai-sdlc-artefacts/epics/EP-040/ep-system-design-review.md [gate=pass, iteration 1, blocker:0 major:0 medium:0 minor:0]`
