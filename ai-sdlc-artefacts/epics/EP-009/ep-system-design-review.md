# Architecture Review — EP-009 Dynamic Tool Creation with Docker Sandbox

**Review date:** 2026-03-23
**Reviewer:** AI Agent
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

---

## 1. Overall Assessment

The architecture document is well-structured, covers all 18 requirements, and follows the project SDLC process. The design adheres to KISS and fail-fast principles, aligning with project rules.

**Verdict:** Architecture is ready for implementation planning with noted clarifications.

---

## 2. Strengths

### 2.1 Requirement Traceability
- All 18 requirements (REQ-09.001–REQ-09.018) are explicitly covered in the traceability table
- Clear mapping between requirements and design components

### 2.2 Modular Structure
- Responsibility separation by layers (`internal/tools`, `internal/toolcatalog`, `internal/noderunner`, `internal/config`)
- Module boundaries defined with dependencies

### 2.3 Security
- Template whitelist as first line of defense
- Secret detection patterns (REQ-09.017) — protection against accidental credential persistence
- Preservation of existing allowlist model through `noderunner`

### 2.4 Fault Tolerance
- Transactional discipline: "YAML write failure → in-memory catalog unchanged"
- Atomic write (write temp + rename) mentioned in risks section

---

## 3. Issues and Recommendations

### 3.1 Critical

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| C1 | **YAML write race condition**: If two `create_tool` calls occur simultaneously, data loss is possible. Risk table mentions file lock, but strategy is not specified. | Line 133: "Atomic write (write temp + rename) or file lock" — this is "or", not "and". | Add mutex at CreateTool handler level. |
| C2 | **No rollback on indexation failure**: After successful YAML write, if tool index update fails — the tool is unavailable for pre-selection until restart. | Line 134 acknowledges the risk but no explicit solution. | Document as known MVP limitation or add retry mechanism. |

### 3.2 Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M1 | **Validation location**: Template whitelist is checked in CreateTool handler, but resource flags (`--memory`, `--cpus`, `--timeout`) — in template or enforced wrapper. No explicit contract. | Line 67: "enforced by a thin wrapper" — requires clarification in implementation plan. |
| M2 | **Config validation**: Secret patterns are loaded at startup, but behavior for invalid regex is not detailed. | Line 94: "must fail fast at load if invalid" — sufficient, but need format example in configuration. |
| M3 | **Network isolation verification**: REQ-09.018 requires verification of no outbound connectivity, but design delegates this to integration test. | No runtime verification mechanism. | Add healthcheck or verification step at deploy time. |

### 3.3 Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| m1 | **Duplicate Glossary**: Glossary in ep-scope.md and ep-requirements.md has overlaps. | Consolidate for better maintainability. |
| m2 | **C4 diagram source**: `c4-container.puml` is mentioned, but PNG in repo may become outdated. | Add check in CI or generate at docs build. |
| m3 | **Coverage threshold**: REQ-09.016 requires 70%, but automatic verification method is not specified. | Line 161: "`make check` coverage threshold" — requires Makefile clarification. |

---

## 4. Architectural Decisions

### 4.1 Justified Trade-offs

| Decision | Justification |
|----------|---------------|
| Monolith (no separate sandbox service) | Aligns with KISS; uses existing SSH + Docker path |
| Template whitelist instead of full command validation | Balance between security and flexibility; LLM is limited in dangerous commands |
| Runtime catalog update without restart | Satisfies REQ-09.012; minimizes disruption |
| `--network none` probe in integration test | Practical isolation verification without runtime overhead |

### 4.2 Potential Improvements (post-MVP)

1. **Approval workflow** (out of scope, line 59) — consider for production
2. **Tool versioning** — conflicts possible when updating tool definition
3. **Rate limiting** on `create_tool` to prevent abuse

---

## 5. NFR Coverage

| NFR | Coverage in Design | Status |
|-----|-------------------|--------|
| REQ-09.014 (5s sandbox startup) | Integration test environment | OK |
| REQ-09.015 (1s create_tool) | Validation + file write + memory update | OK |
| REQ-09.016 (70% coverage) | `make check` threshold | Requires Makefile verification |
| REQ-09.017 (Secret rejection) | Config-driven regex list | OK |

---

## 6. Project Rules Compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ Minimal changes to existing monolith |
| Fail fast | ✅ Validation before write, explicit errors |
| Security | ✅ Whitelist, secret patterns, allowlist preservation |
| Testability | ✅ Unit + Integration strategy defined |
| Traceability | ✅ Links to requirements and acceptance criteria |

---

## 7. Summary

**Architecture is ready for implementation planning.** The following action items were raised in the initial review and are **closed** in [ep-system-design.md](ep-system-design.md) (see §8):

1. ~~Clarify concurrent writes strategy (mutex vs file lock)~~
2. ~~Document behavior on tool index refresh failure~~
3. ~~Add example format for `create_tool_secret_patterns` in configuration~~

---

## 8. Resolution (2026-03-23)

Items below are **addressed** in [ep-system-design.md](ep-system-design.md) (revision aligning with this review).

| Ref | Resolution |
|-----|------------|
| C1 | **Mutex** on the full `create_tool` critical section + **atomic** YAML write documented in [Concurrency and catalog writes](ep-system-design.md#concurrency-and-catalog-writes) and [Risks](ep-system-design.md#risks-and-trade-offs). |
| C2 | **MVP limitation** for vector index lag + retry recommendation in [Tool vector index after create](ep-system-design.md#tool-vector-index-after-create); error row in [Error handling](ep-system-design.md#error-handling). |
| M1 | **Resource flags contract** table: template-driven flags, optional substring validation, no silent injection—[Resource flags contract](ep-system-design.md#resource-flags-contract-sandbox-templates). |
| M2 | **JSON example** `create_tool_secret_patterns` + compile-at-load behaviour in [Data models](ep-system-design.md#config-extension-secret-detection). |
| M3 | **Integration test** as primary proof for REQ-09.018; deploy-time check explicitly **post-MVP / ops** in [Testing strategy](ep-system-design.md#testing-strategy). |
| m1 | **Glossary** pointer: single source in ep-requirements in [Documentation and diagram maintenance](ep-system-design.md#documentation-and-diagram-maintenance). |
| m2 | **CI / regen** note for diagram PNG in same section. |
| m3 | **Coverage gate** deferred to **stage 7 / Makefile** in [Testing strategy](ep-system-design.md#testing-strategy). |

**C4 diagram:** Embedded PNG restored in [Architecture](ep-system-design.md#architecture).

---

## Traceability

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
