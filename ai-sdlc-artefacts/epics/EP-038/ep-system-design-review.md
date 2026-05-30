---
artefact: ep-system-design-review
epic_id: EP-038
status: draft
source_of_truth: true
gate: fail
latest_iteration: 1
open_counts:
  blocker: 0
  major: 0
  medium: 1
  minor: 1
next_action: return_to_stage_6
updated_at: 2026-05-31
---

# Architecture Review — EP-038 Refactor core conversation handler (god handler)

**Reviewer:** AI Agent (delegated pipeline stage 7, fresh context)

---

## Current Gate Summary

Gate: Fail
Latest iteration: 1
Last updated: 2026-05-31
Open counts: Blocker 0 | Major 0 | Medium 1 | Minor 1
Open findings:
- F-001 Medium: `validate ears EP-038` is vacuous — requirements headings not in EARS parser format.
- F-002 Minor: Method→file mapping omits package-level helpers vs receiver methods (AC grep ambiguity).
Next action: Return to stage 6 (revise design testing strategy; coordinate `ep-requirements.md` heading reformat for REQ-38.022)

---

## Review iteration 1

**Review date:** 2026-05-31
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 1
**Gate:** Fail (Medium/Minor > 0)

### Overall assessment

The system design is a strong, implementation-ready structural refactor plan: complete structural sections, accurate C4 container diagram, exhaustive method→file mapping aligned with the current 663-LOC `handler.go` on branch `epic/EP-038-refactor-core-handler`, preserved HandleMessage sequence and two-tier `switch`, full REQ-38.001–038.025 traceability, and sensible extraction order (memory → tools → llm → slim orchestration). Prerequisites EP-035/036/037 are merged on the branch. The gate fails on one Medium cross-artefact verification gap (REQ-38.022 / AC-38.022) and one Minor documentation clarity issue for manual structural ACs.

**Verdict:** Fail gate — return to stage 6 (and reformat `ep-requirements.md` headings before relying on `validate ears` / `validate req`).

### Strengths

- **Accurate as-is inventory:** Method→file mapping matches production code on the epic branch (`handler.go` 663 LOC; `handler_llm.go` / `handler_tools.go` / `handler_memory.go` not yet present — expected pre–stage 9). Listed symbols exist at the documented current locations.
- **Behaviour preservation:** HandleMessage mermaid sequence matches `handler.go` call order; tier dispatch remains `case intent.TierFull` / `default` in `handler_tier_main_prompt.go` (REQ-38.013); no `full_lite` in `internal/core`.
- **Boundary discipline:** Design explicitly leaves `runtime_tools.go`, `dynamic_tool_selection.go`, `system_tail.go`, `vector_merge.go`, and `memory_vectors.go` unchanged; EP-034 transport-only routing and EP-037 `tools.selection` wiring are called out in components and error-handling tables.
- **Full REQ traceability:** All 25 requirements map to design sections; testing strategy names concrete regression files (`handler_ep034_regression_test.go`, `handler_ep036_test.go`, `tools_selection_parity_test.go`, etc.).
- **KISS alignment:** Single unexported `conversationHandler` across new files; no tier-strategy framework; risks table addresses concurrent-edit and accidental-logic-change mitigations.

### Issues and recommendations

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
| F-001 | **REQ-38.022 / AC-38.022 validator gate is vacuous for EP-038.** Design Testing strategy cites `./tools/validate/validate ears EP-038` as a repo gate. `ep-requirements.md` uses bold labels (`**REQ-38.001**`) instead of canonical `### REQ-38.NNN — <summary>` headings (see EP-034, EP-036). `ears_lint.go` and `req_ac_trace.go` only parse `### REQ-…` headings (`reqHeadingRe` / `reqHeadingPattern`). Verified on branch: `validate ears EP-038` → `total_reqs: 0`, exit 0 (no EARS lint on 25 requirements); `validate req EP-038` → exit 1, `0/0 REQs covered`, 25 orphan AC refs. REQ-38.022 cannot be satisfied as written until headings are reformatted. | Reformat `ep-requirements.md` requirement blocks to `### REQ-38.NNN — <summary>` (keep EARS body text and index table). Re-run `validate ears EP-038` and `validate req EP-038` until both pass. Update design Testing strategy with an explicit note that REQ-38.022 depends on canonical REQ headings (or reference the reformat as a stage-8 prerequisite). |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| F-002 | **Package-level helpers vs AC manual grep wording.** REQ-38.005/007/009 and AC-38.005/007/009 list symbols to move; AC-38.005 says “defined (as methods on `conversationHandler`)”. On the epic branch, nine listed symbols are **package-level** (not receivers): `genRequestID`, `truncateToolResultForPrompt`, `parseToolArgumentsJSON`, `remoteCommandFromRunOnNodeArgs`, `retrievalChunkWithLabel`, `eventAlignedTurnDate`, `canonicalizeTurnPair`, `canonicalizeTurnText`, plus `todayCalendarDateInPALocation(h *conversationHandler)` and `paLocationFromConfig` (only referenced from `run.go` for handler wiring). Design Method→file table does not distinguish receiver methods from package helpers. | Add a “Package-level helpers (target file)” subsection to Method→file mapping listing the above symbols. State that MANUAL AC grep verifies symbol definition **in the target file**, not `func (h *conversationHandler)` syntax. Optional: align AC-38.005/007/009 wording in stage 5 artefact to “defined in” rather than “methods on”. |

#### Nit

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| N-001 | Turn-index id wording | AC-38.018 / design cite `turn:<date>:<12-byte-hex-prefix>`; code uses `sum[:12]` with `%x` (24 hex digits). `handler.go` comment documents this. | Harmonize to “first 12 bytes of SHA-256 digest (24 hex chars)” in stage 8 checklist if grep-based verification is added. |

#### Suggestion

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| S-001 | Import split | Today’s `handler.go` has a large import block shared by LLM, tools, memory, and crypto/vector paths. | Stage 8 implementation plan: after each extraction step, run `go test ./internal/core/...` and trim per-file imports to avoid churn. |
| S-002 | `paLocationFromConfig` placement | REQ-38.005 assigns it to `handler_llm.go`; only caller is `run.go` (`BuildMessageHandler` wiring). Same-package move is fine. | One-line note in Components table: construction-time helpers remain callable from `run.go` after move. |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Same-package file split only | Smallest change; no new interfaces; preserves unexported `conversationHandler` (REQ-38.024). |
| Extraction order memory → tools → llm | Keeps `make check` green incrementally; tier file continues calling moved receivers. |
| Leave tier/tail/support files | Matches EP-026 precedent and EP-037 scope boundary. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-38.021 `make check` | Testing strategy | OK (post-implementation) |
| REQ-38.022 `validate ears` | Testing strategy | Needs work (F-001) |
| REQ-38.023 no behaviour change | Overview, error handling, risks | OK |
| REQ-38.024 no export | Overview, data models | OK |
| Security / config | No schema change; cmdsafe/allowlist paths unchanged | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ File split only; no frameworks |
| Fail fast | ✅ No config or behaviour changes |
| Security | ✅ No weakening; structural moves only |
| Testability | ⚠️ EARS/req validator gap (F-001); otherwise regression suites named |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
- **Codebase (branch `epic/EP-038-refactor-core-handler`):** `internal/core/handler.go` (663 LOC), `handler_tier_main_prompt.go`, `run.go`, `adapter.go`, `integration_export.go` — target files not yet created (pre–stage 9)
