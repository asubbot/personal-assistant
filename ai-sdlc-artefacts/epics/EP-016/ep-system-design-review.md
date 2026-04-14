# EP-016 — System design review

**Pipeline:** [pipeline.spec.md](../../ai-sdlc/specification/pipeline.spec.md) §2.1

---

## Review iteration 1

**Review date:** 2026-04-14  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 4 | Minor: 2  
**Gate:** Fail (any Blocker/Major/Medium > 0)

### Overall assessment

The design document is structurally complete for stage 7 (overview, C4 C2 container diagram, module boundaries, components/interfaces, data models, retrieval sequence for split queries, migration messaging, error handling, testing strategy, risks, and a full REQ-16.001–REQ-16.028 traceability table). Scope alignment for variant 1 split retrieval, merge order, and filesystem tools is good. Remaining gaps are mainly traceability precision, observability placement for DEBUG class counts, and a few incomplete contracts in error handling (notably `read_memory` caps and `write_memory`/vector partial failure).

**Verdict:** Fail gate — return to stage 6 to revise `ep-system-design.md`, then re-run stage 7.

### Strengths

- Clear **split-query** retrieval story with a sequence diagram and explicit merge ordering consistent with [ep-scope.md](ep-scope.md) variant 1.
- Solid **data model** separation (`vec_summaries`, `vec_turns`, `vec_notes`) with example ids and stored text shapes supporting REQ-16.010–REQ-16.012, REQ-16.018, REQ-16.022.
- **Requirement traceability** includes all 28 requirement IDs from [ep-requirements.md](ep-requirements.md).

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| | None | | |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| | None | | |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | REQ-16.005 traced to the wrong conceptual area | Trace row points to “Event-aligned date derivation”, which is turn-centric | Add/adjust a subsection for `write_memory` default calendar-day resolution; update the trace row |
| 2 | REQ-16.025 observability not designed where implemented | Trace row points to “Testing strategy” | Specify DEBUG logging after merge (per-class counts, field names); move trace to retrieval/observability |
| 3 | `write_memory` disk vs vector partial failure undecided | Error handling lists rollback vs repair without a chosen contract | Pick one behaviour and document user/operator-visible outcomes |
| 4 | `read_memory` bound/cap errors underspecified | REQ-16.009 / AC-16.008 not reflected in the error-handling table | Add explicit rows for span/byte limits and structured error semantics |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | REQ-16.009 is thin for design-only readers | Trace says “unchanged” via `ReadMemoryTool` only | Add an explicit one-line pointer to EP-002 noon anchoring and caps |
| 2 | Traceability table omits component column | Pipeline skill Step 3 | Add a component (or component + section) column per REQ |

---

## Review iteration 2

**Review date:** 2026-04-14  
**Stage 7 iteration:** 2 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 1 | Minor: 1  
**Gate:** Fail (Blocker/Major/Medium not all zero)

### Overall assessment

Prior iteration Medium items on REQ-16.005 vs turn dating, REQ-16.025 DEBUG placement and content, write_memory disk-then-vector failure contract, read_memory vs REQ-16.009 in the error table, and EP-002 / component traceability are addressed in the updated design. One Medium remains: the `max_span_days` error-handling row cites AC-16.008, which only covers the output byte cap.

**Verdict:** Fail gate — resolve Medium (AC trace for span) then re-run stage 7 or accept operator decision per pipeline cap.

### Issues and recommendations

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M-2.1 | Wrong AC pointer for `max_span_days` | Error handling row links AC-16.008, which tests only `max_output_bytes` | Remove AC from that row until a span AC exists, or add an AC for span violation and link it here |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| m-2.1 | Risk mitigation vs chosen write path | Risks and trade-offs still lists write-after-vector / temp-file options | Align bullet with append + truncate-on-vec-failure contract in Error handling |

---

## Review iteration 3

**Review date:** 2026-04-14  
**Stage 7 iteration:** 3 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0  
**Gate:** Pass

### Overall assessment

Design updates remove the erroneous AC pointer on the `max_span_days` error row, align risks with the append-and-truncate contract, and retain split-query retrieval, merge order, and full REQ traceability with a **Component** column. No open Blocker, Major, or Medium findings remain.

**Verdict:** Pass — proceed to stage 8 (implementation planning).

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
