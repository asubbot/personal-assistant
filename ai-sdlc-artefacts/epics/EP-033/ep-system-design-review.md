# Architecture Review — EP-033 Memory Summarization Retry

**Reviewer:** AI Agent (delegated stage 7 reviewer)

---

## Review iteration 1

**Review date:** 2026-04-21  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 1 | Medium: 2 | Minor: 1  
**Gate:** Fail (any Blocker/Major/Medium/Minor > 0)

### Overall assessment

The design document is well-structured and covers all required sections, with full REQ/AC traceability and a clear narrow scope aligned to EP-033.  
Core architecture choices (single existing queue, bounded retries, dedupe intent) are directionally correct and consistent with KISS.  
However, key retry contracts are still underspecified for implementation and verification, so the stage 7 gate cannot pass yet.

**Verdict:** Fail gate

### Strengths

- Complete stage-7 structure is present: overview, architecture diagram, module boundaries, interfaces, data models, error handling, testing strategy, risks/trade-offs, and requirement traceability.
- Requirement traceability includes all REQ-33.001 through REQ-33.014 with linked AC coverage.
- Scope control is strong: day-job retries are isolated, month/year behavior is explicitly preserved, and no extra worker subsystem is introduced.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| - | None | - | - |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Retryable vs non-retryable decision contract is not specified | `Error handling` states retryable failures are retried and non-retryable are not, but no classification rules/source-of-truth are defined; this leaves REQ-33.004/REQ-33.005 behavior ambiguous and hard to test consistently. | Add a concrete retry-classification contract (decision table or policy section) mapping failure categories to actions (`retry`, `no retry`, `terminal`) and define default fail-fast behavior for unknown errors. |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Dedupe semantics are incomplete for race/concurrency scenarios | Design defines dedupe key and duplicate prevention intent, but does not specify atomicity guarantees when enqueue attempts for the same day arrive close together (e.g., from different producers). | Specify where dedupe state is owned and how atomic enqueue+dedupe is enforced in the queue path; include expected behavior for simultaneous enqueue attempts of the same day target. |
| 2 | Deterministic retry policy is not concretized enough for implementation/test reproducibility | `Data models` mentions `max_attempts` and `backoff` sequence, but not the authoritative source (constants/config) or clock source contract for `not_before` calculations tied to REQ-33.011. | Define exact policy source (constants/config keys), retry delay sequence shape, and clock abstraction used for scheduling/tests so deterministic behavior is verifiable. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Security/observability guardrails for retry logs are implicit, not explicit | Logging fields are listed, but there is no explicit note that retry logs must avoid sensitive payload leakage from underlying errors. | Add a short logging safety note in `Error handling` or `Risks and trade-offs` specifying structured fields and redaction/sanitization expectations for error details. |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Keep retries inside existing `memoryjob` queue/worker loop | Minimizes architecture changes, preserves existing deferral and execution model, aligns with KISS and REQ-33.008. |
| Limit retry scope to day jobs only | Prevents scope creep and protects month/year behavior guarantees in REQ-33.003. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Deterministic retry timing | Mentioned via fixed policy + tests, but policy source/clock contract needs clarification | Needs work |
| Testability | Unit/integration strategy present and aligned to retry behaviors | OK |
| Quality gates | `make check` and `./bin/validate EP-033` included | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ⚠️ |
| Security | ⚠️ |
| Testability | ✅ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)

---

## Review iteration 2

**Review date:** 2026-04-21  
**Stage 7 iteration:** 2 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0  
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

Iteration 2 resolves all open findings from iteration 1 while preserving EP-033 scope boundaries and KISS architecture decisions.  
The design now provides concrete retry-classification rules, deterministic retry policy source/clock contracts, explicit dedupe concurrency semantics, and logging safety expectations, with full REQ-to-AC traceability.

**Verdict:** Pass gate

### Strengths

- Retry classification is now explicit, including unknown-error fallback behavior and terminal exhaustion handling, making REQ-33.004/005 testable.
- Dedupe and concurrency behavior is concretely defined (`Runner.mu` atomicity, pop/reinsert semantics, simultaneous enqueue outcome), closing the prior race-gap.
- Determinism contract is implementation-ready (constants as policy source, explicit backoff sequence, `Runner.now()` with `Deps.Now`/`time.Now` mapping).
- Security-conscious observability is documented (structured retry logs and redaction/sanitization expectations).
- Structural completeness and traceability remain intact across all required stage-7 sections and REQ-33.001..014.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| - | None | - | - |

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| - | None | - | - |

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| - | None | - | - |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| - | None | - | - |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Keep retries in existing `memoryjob` queue loop | Minimizes architecture change, preserves existing scheduling/deferral behavior, and satisfies REQ-33.008. |
| Limit retries to day jobs only | Enforces EP-033 scope and protects unchanged month/year behavior (REQ-33.003). |
| Use deterministic constants + injected clock contract | Enables reproducible retry scheduling behavior and verifiable tests (REQ-33.011/012). |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Determinism | Fixed policy source, explicit backoff, and clock abstraction are defined | OK |
| Testability | Unit/integration strategy maps to retry timing, exhaustion, dedupe, and regression checks | OK |
| Quality gates | `make check` and `./bin/validate EP-033` are captured in design testing strategy | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ✅ |
| Security | ✅ |
| Testability | ✅ |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
