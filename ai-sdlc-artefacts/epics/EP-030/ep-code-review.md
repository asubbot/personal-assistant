# Code review — EP-030 (implementation)

**Review date:** 2026-04-19  
**Scope:** Post-implementation review of the native-tool-only change set (`internal/core`, `internal/config`, `internal/llm`, `internal/toolcatalog`, `cmd/pa`, tests, operator docs).  
**Related:** [ep-system-design-review.md](ep-system-design-review.md) (stage 7 architecture review).

---

## Verdict

**Pass with follow-ups addressed in-repo:** Critical paths match REQ-30.001–REQ-30.016; config rejects removed keys; `make check` and `./bin/validate EP-030` were used as the quality gate. Residual items below were triaged; several were fixed in the same maintenance pass (legacy naming, dead helpers, operator-facing messages).

---

## Strengths

- **Single execution path** for conversation tools via native `tool_calls`; text-markup parsing and `internal/tooltext` removed.
- **Fail-fast configuration:** removed JSON keys rejected with explicit errors; `default_response_format` locked to `text`.
- **Tests** retain AC trace comments for `./bin/validate EP-030`; coverage includes native tool defs on the completion path and rejection of removed config keys.
- **Operator documentation** (`docs/configuration.md`) describes baseline `supports_tools`, removed keys, and native-only behaviour.

---

## Findings (historical)

| Severity | Topic | Notes | Resolution |
|----------|--------|-------|------------|
| Low | Duplicated baseline `supports_tools` logic | Similar helper existed in `cmd/pa` and core startup paths. | Acceptable duplication for small surface; watch for drift if logic grows. |
| Low | Startup WARN vs strict AC wording | AC-30.009 expects WARN when baseline omits native tools and catalog has tools; ensure WARN fires once per process after config load. | Verified in wiring; message kept operator-neutral (no epic id in logs). |
| Low | Legacy naming in code/comments | `HermesBody`, `hermes_prompt` struct field, and epic ids in some error strings. | **Done:** removed unused catalog helpers/fields, renamed config rejection helper, stripped epic ids from errors/logs, updated examples and docs. |
| Low | Dead code | `WrapHermesToolFormat` unused after path removal. | **Done:** removed from `internal/systemprompt`. |

---

## Verification commands

- `make check`
- `make build && ./bin/validate EP-030`

---

## Traceability

- **Requirements:** [ep-requirements.md](ep-requirements.md)  
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)  
- **Implementation plan:** [ep-implementation-plan.md](ep-implementation-plan.md)
