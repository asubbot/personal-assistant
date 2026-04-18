# Architecture Review — EP-025 Test layout cleanup: E2E separation

**Reviewer:** Delegated agent (pipeline stage 7)

---

## Review iteration 1

**Review date:** 2026-04-17  
**Stage 7 iteration:** 1 of max 5  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)  
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0  
**Gate:** Pass

### Overall assessment

The system design aligns with [ep-scope.md](ep-scope.md), states epic intent with an explicit scope reference in Overview, and the [Requirement traceability](ep-system-design.md#requirement-traceability) table maps every requirement **REQ-25.001** through **REQ-25.008** to concrete design sections and components. Structural elements required by stage 7 (overview with scope reference, C4 C2 architecture diagram, module boundaries, components/interfaces, data models, error handling, testing strategy, risks/trade-offs, and requirement traceability) are all present and consistent with [ep-acceptance-criteria.md](ep-acceptance-criteria.md).

**Verdict:** Pass gate

### Strengths

- **REQ-25.001 / REQ-25.002:** Clear split between `//go:build e2e` job flows and `//go:build !e2e` placeholder, with an explicit rule that e2e must not import `cmd/pa` as `main` ([Module boundaries](ep-system-design.md#module-boundaries)).
- **REQ-25.003 / REQ-25.004 / REQ-25.006:** Makefile contracts are named in the components table and reinforced by a dedicated policy test component (`ep025_policy_test.go`), matching **AC-25.003**–**AC-25.006**.
- **REQ-25.005:** CI layer called out in components and testing strategy, aligned with **AC-25.005**.
- **REQ-25.007 / REQ-25.008:** `DeliveryRunner` extraction, error-handling notes (including fail-fast on nil handler), and `make check` / validate hooks are documented and tied to **AC-25.007** and **AC-25.008**.

### Issues and recommendations

#### Blocker

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Major

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| Extract scheduled-job delivery as `DeliveryRunner` in `internal/jobs` | Satisfies **REQ-25.007**, enables e2e and unit tests to share behaviour without compiling full `main` in `tests/e2e` (**REQ-25.001**). |
| Pair `e2e` and `!e2e` files in `tests/e2e` plus policy tests | Reduces build-tag confusion risk called out in risks; supports **REQ-25.002** and **REQ-25.003**–**REQ-25.006**. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Reliability / test pyramid (per requirements NFR section) | E2E gated and separated from default integration runs; CI messaging for coverage layers | OK |
| **REQ-25.008** verification | `make check`, `./bin/validate EP-025`, vet/vuln/lint with `integration,e2e` | OK |

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
