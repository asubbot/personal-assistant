# Architecture Review — EP-016

**Reviewer:** AI Agent

---

## Review iteration 1

**Review date:** 2026-04-14  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 5  
**Gate:** Pass (Blocker/Major/Medium all zero)

### Overall assessment

The system design is coherent with [ep-scope.md](ep-scope.md) (including split-query rollout), covers the main component boundaries, data models, error paths, risks, and a full requirement traceability table for **REQ-16.001** through **REQ-16.027**. Acceptance criteria are largely supported by concrete design hooks (headings, table names, id formulas, legacy prefix filtering, and validation entry points). Follow-up polish (link text, first-line notes format alignment with AC, explicit AC ownership pointer, doc path for REQ-16.027) was applied in `ep-system-design.md` after this review draft.

**Verdict:** Pass gate

### Strengths

- **Complete REQ table:** All **REQ-16.001**–**REQ-16.027** appear in [ep-system-design.md](ep-system-design.md) § Requirement traceability with concrete design anchors (paths, tables, tools, behaviours).
- **Scope alignment:** Three dedicated vector stores plus **legacy `vec_items` summary-only** path matches variant 1 in [ep-scope.md](ep-scope.md); turn path avoids legacy duplication per **REQ-16.019**.
- **Testability:** Stable ids, upsert semantics, explicit headings (`### Automatic summary`, `### Manual notes`), and **SHA-256** canonicalisation give clear targets for **AC-16.009**–**AC-16.015** and **AC-16.016**–**AC-16.018**.
- **NFR hooks:** Logging parity for **REQ-16.024**, `./bin/validate EP-016` and **make check** for **REQ-16.025**–**REQ-16.026**, and **docs/configuration.md** for **REQ-16.027** are explicitly wired in the design narrative.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | *(none — open count 0)* | | |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | *(none — open count 0)* | | |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | *(none — open count 0)* | | |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Components table REQ link inconsistency | `vector.Store` row linked only to REQ-16.016 | Fixed in design: cite **REQ-16.015** and **REQ-16.016** |
| 2 | First-line `notes.md` format vs **AC-16.003** | Design previously used `timestamp=` prefix | Fixed: line 1 is raw RFC3339 per **REQ-16.004** |
| 3 | **REQ-16.025** matrix | Testing strategy lacked explicit AC owner pointer | Fixed: implementation plan must mirror AC index |
| 4 | **AC-16.021** / **REQ-16.027** doc surface | Doc path ambiguity | Fixed: **docs/configuration.md** named as operator source |
| 5 | Vector failure after successful append | Optional test gap | Defer unless product wants integration test in this epic |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Upsert (**Delete** + **Add**) for turns | Matches summary behaviour and satisfies **REQ-16.022** / **AC-16.015** bounded-growth expectation. |
| Legacy summary retrieval via **post-filter** on **`vec_items`** | Achieves **REQ-16.018** result set (only `summary:*` ids) without requiring sqlite-vec prefix SQL features. |
| Fallback clock for **REQ-16.021** | **`time.Now()` in `pa_timezone`** at indexing call is explicit, testable, and documented. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Security / redaction (**REQ-16.024**) | Same hooks as **read_memory** | OK |
| Verification (**REQ-16.025**) | Tests + `./bin/validate EP-016` | OK |
| Quality (**REQ-16.026**) | **make check** | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | OK |
| Fail fast | OK |
| Security | OK |
| Testability | OK |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)  
- **Requirements:** [ep-requirements.md](ep-requirements.md)  
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)  
- **Scope:** [ep-scope.md](ep-scope.md)
