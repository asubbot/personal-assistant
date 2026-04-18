# Architecture Review — EP-024 Operator documentation and safe logging defaults

**Reviewer:** Delegated agent (pipeline stage 7)

---

## Review iteration 1

**Review date:** 2026-04-17
**Stage 7 iteration:** 1 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 2 | Minor: 2
**Gate:** Fail (any Blocker/Major/Medium/Minor > 0)

### Overall assessment

The system design is structurally complete (overview, C4 container diagram, module boundaries, components/interfaces, data models, error handling, testing strategy, risks, and a requirement traceability table). All REQ-24.001–REQ-24.010 appear in the traceability table with plausible design anchors. As a strict review against the requirements text and glossary, verification scope for Compose `include:` and the full `PA_ENV` predicate in REQ-24.008 are under-specified relative to the normative REQ wording, leaving avoidable implementation and test gaps.

**Verdict:** Fail gate

### Strengths

- **Full REQ inventory in traceability:** The traceability table explicitly lists REQ-24.001 through REQ-24.010 with mapped design sections ([ep-system-design.md](ep-system-design.md) § Requirement traceability).
- **Clear bounded change:** Module boundaries cleanly separate `docs/`, container defaults, and `cmd/pa` startup policy without spurious Go coupling ([ep-system-design.md](ep-system-design.md) § Module boundaries).
- **Operational realism in risks:** Explicitly distinguishes application `slog` `debug` behaviour from `paths.llm_log_dir` JSONL audit logging, reducing false assurance ([ep-system-design.md](ep-system-design.md) § Risks and trade-offs).
- **REQ-24.008 sequencing:** Error handling states the warning runs after the effective level is chosen, matching the REQ’s “effective application log level” notion ([ep-system-design.md](ep-system-design.md) § Error handling).

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
| 1 | REQ-24.008 trigger not fully reflected in the verification plan | REQ-24.008 requires a WARN when `debug` and (`PA_ENV` unset **or** `PA_ENV` ≠ `development` under ASCII case-folding). The testing strategy cites [AC-24.009](ep-acceptance-criteria.md#ac-24-009), whose Given clause only states `PA_ENV` unset ([ep-system-design.md](ep-system-design.md) § Testing strategy; [ep-acceptance-criteria.md](ep-acceptance-criteria.md) AC-24.009). | Extend the design’s testing strategy (and/or push an AC update) to require explicit cases where `PA_ENV` is set to non-`development` values (including case variants) while `PA_LOG_LEVEL=debug`, asserting exactly one WARN; keep the “zero WARN when `PA_ENV=development`” case. |
| 2 | Compose “production-oriented” scope vs validation target is ambiguous | [ep-requirements.md](ep-requirements.md) glossary defines production-oriented Docker artefacts as root `Dockerfile`, `docker-compose.yml`, **and files included via Compose `include:`**. The design and tests anchor on root `Dockerfile` and `docker-compose.yml` only ([ep-system-design.md](ep-system-design.md) § Components and interfaces; § Testing strategy). | State whether `include:` targets exist in-repo for `pa`, and whether file-content tests recurse or otherwise cover included compose fragments; if none, document that explicit assumption so REQ-24.006/007 interpretation is unambiguous. |

#### Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| 1 | Overview does not point to the epic scope artefact | Stage 7 structural checklist expects an overview with scope reference; the overview references requirements IDs but not [ep-scope.md](ep-scope.md) ([ep-system-design.md](ep-system-design.md) § Overview). | Add a single sentence + link to [ep-scope.md](ep-scope.md) for operator-visible intent and glossary alignment. |
| 2 | REQ-24.010 traceability row omits listed verification mechanics | Components table ties [REQ-24.010](ep-requirements.md#verification) to `make check` **and** `./bin/validate EP-024` ([ep-system-design.md](ep-system-design.md) § Components and interfaces), but the traceability row only lists “Overview; Testing strategy” ([ep-system-design.md](ep-system-design.md) § Requirement traceability). | Amend the REQ-24.010 row to include validate CLI (or mark validate as optional with justification) so the traceability table matches components. |

### Architectural decisions (optional)

| Decision | Justification |
|----------|---------------|
| — | No additional ADRs recorded this iteration. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Security / sensitive logging | Defaults to `info`, explicit WARN for `debug` without dev acknowledgement; docs distinguish JSONL audit path | OK |
| Deployability | Notes `.env` overrides vs explicit compose baseline | OK |
| Operability | Single doc entry point + configuration link | OK |

### Project rules compliance (optional)

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ⚠️ (WARN path clear; extend tests for full REQ predicate — see Medium #1) |
| Security | ✅ |
| Testability | ⚠️ (strong for cited ACs; gaps vs full REQ-24.008 / compose `include:` scope — see Medium #1–2) |

### Traceability (this iteration)

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)

---

## Review iteration 2

**Review date:** 2026-04-17
**Stage 7 iteration:** 2 of max 5
**Document reviewed:** [ep-system-design.md](ep-system-design.md)
**Iteration summary — open counts:** Blocker: 0 | Major: 0 | Medium: 0 | Minor: 0
**Gate:** Pass (Blocker/Major/Medium/Minor all zero)

### Overall assessment

Iteration 1’s Medium findings on REQ-24.008 / AC-24.009 coverage and Compose `include:` scope, and Minor findings on overview scope linkage and REQ-24.010 traceability, are reflected in the current `ep-acceptance-criteria.md` and `ep-system-design.md` without regression elsewhere. The design remains structurally complete against the stage 7 checklist, and requirement traceability still covers REQ-24.001 through REQ-24.010 with acceptance criteria alignment.

**Verdict:** Pass gate

### Strengths

- **REQ-24.008 testability:** AC-24.009 now encodes the full predicate (unset or non-`development` `PA_ENV` with ASCII case-folding, plus negative cases for `development` / `DEVELOPMENT` and `info`), and the testing strategy explicitly mirrors those cases ([ep-acceptance-criteria.md](ep-acceptance-criteria.md) AC-24.009; [ep-system-design.md](ep-system-design.md) § Testing strategy).
- **Compose `include:` vs REQ-24.007:** The architecture section documents overlay files that use `include:` and do not redefine `pa` `environment`, closing the glossary scope gap from iteration 1 ([ep-system-design.md](ep-system-design.md) § Architecture).
- **Overview and traceability polish:** Overview links operator intent to [ep-scope.md](ep-scope.md), and the REQ-24.010 row now matches the components table by naming both `make check` and `./bin/validate EP-024` ([ep-system-design.md](ep-system-design.md) § Overview; § Requirement traceability).

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
| — | No new ADRs beyond iteration 1; prior assumptions (overlay compose, startup WARN sequencing) remain coherent. |

### NFR coverage (optional)

| NFR | Coverage | Status |
|-----|----------|--------|
| Security / sensitive logging | `info` defaults, single WARN for `debug` without dev acknowledgement; risk note on JSONL vs `slog` | OK |
| Deployability | Compose baseline vs optional `.env` | OK |
| Operability | Single doc entry point and configuration link | OK |

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
