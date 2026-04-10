# Architecture Review — EP-014 Sliding session memory window

**Review date:** 2026-04-10  
**Reviewer:** AI Agent  
**Document reviewed:** [ep-system-design.md](ep-system-design.md)

---

## 1. Overall Assessment

The design is minimal, aligns with KISS: optional config, in-process store, per-key locking, and a single extension point on `MessageHandler`. Traceability to all REQ-14.xxx entries is explicit in the design and in code.

**Verdict:** Ready

---

## 2. Strengths

### 2.1 Simplicity

- No persistence layer in MVP; bounded memory via `max_session_exchanges`.
- Session append only after successful LLM completion (`handleLLMSuccess`), avoiding pollution from Hermes parse short-circuits.

### 2.2 Concurrency

- Per-session mutex prevents lost updates under concurrent Telegram updates for the same chat.

### 2.3 Testability

- Store and handler behaviour covered with `Covers AC-14.*` comments; `./bin/validate EP-014` passes (one deferred AC for manual restart).

---

## 3. Issues and Recommendations

### 3.1 Critical

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| — | None | — | — |

### 3.2 Medium

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| M1 | Empty `sessionKey` falls back to `uid:userID` | Design / tests | Document for non-Telegram adapters; require explicit key in future channels if needed. |

### 3.3 Minor

| # | Issue | Context | Recommendation |
|---|-------|---------|----------------|
| n1 | Unbounded session keys | `map[string]*sessionKeyBuf` | Accept for MVP; consider TTL or max keys if abuse becomes a concern. |

---

## 4. Architectural Decisions

### 4.1 Justified Trade-offs

| Decision | Justification |
|----------|---------------|
| Interface change on `MessageHandler` | Clean contract for any adapter; all call sites updated. |
| No persistence | Matches scope; EP-002 / future work for long-term memory. |

### 4.2 Potential Improvements (post-MVP)

1. Disk-backed or Redis session store for multi-instance deployments.
2. Optional `/clear` command to reset a session without restart.

---

## 5. NFR Coverage

| NFR | Coverage | Status |
|-----|----------|--------|
| REQ-14.012 | Unit/handler/config tests | OK |
| REQ-14.013 | Shared `logRedactor` on DEBUG request logs | OK |
| REQ-14.014 | `docs/configuration.md` | OK |

---

## 6. Project Rules Compliance

| Rule | Compliance |
|------|------------|
| KISS | ✅ |
| Fail fast | ✅ |
| Security | ✅ (no new secret paths; redaction unchanged) |
| Testability | ✅ |

---

## 7. Summary

**Ready** with optional follow-up: document `sessionKey` fallback for adapters other than Telegram.

1. **Manual** — Run process restart check for AC-14.004 when convenient.

---

## Traceability

- **Architecture:** [ep-system-design.md](ep-system-design.md)
- **Requirements:** [ep-requirements.md](ep-requirements.md)
- **Acceptance criteria:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- **Scope:** [ep-scope.md](ep-scope.md)
