# EP-014 — Implementation plan

**Pipeline:** Stage 8.  
**Related:** [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-requirements.md](ep-requirements.md), [ep-system-design.md](ep-system-design.md), [ep-system-design-review.md](ep-system-design-review.md), [strategy.md](../../strategy.md)

**Status:** Implemented on branch `epic/EP-014-sliding-session-memory` (2026-04-10).

---

## Tasks (completed)

- [x] **1. Config** — Add `conversation_session` to `config.Config`; `validateConversationSession` when `enabled`.  
  - _Requirements:_ [REQ-14.001](ep-requirements.md#configuration), [REQ-14.002](ep-requirements.md#configuration)  
  - _Acceptance Criteria:_ [AC-14.001](ep-acceptance-criteria.md#ac-14-001--config-keys-when-section-present), [AC-14.002](ep-acceptance-criteria.md#ac-14-002--invalid-cap-fails-load)  
  - **Verification:** `TestLoad_ConversationSession_*`; `make check`.

- [x] **2. Session store** — `sessionWindowStore` with per-key mutex, snapshot, append with cap.  
  - _Requirements:_ [REQ-14.004](ep-requirements.md#session-store), [REQ-14.005](ep-requirements.md#session-store)  
  - _Acceptance Criteria:_ [AC-14.005](ep-acceptance-criteria.md#ac-14-005--concurrent-updates-safe), [AC-14.008](ep-acceptance-criteria.md#ac-14-008--append-after-successful-turn), [AC-14.010](ep-acceptance-criteria.md#ac-14-010--chronological-order)  
  - **Verification:** `session_window_test.go`; handler session tests.

- [x] **3. Handler** — Extend `HandleMessage` with `sessionKey`; inject history; append after `handleLLMSuccess`.  
  - _Requirements:_ [REQ-14.006](ep-requirements.md#prompt-assembly)–[REQ-14.011](ep-requirements.md#interaction-with-vector-memory)  
  - _Acceptance Criteria:_ [AC-14.006](ep-acceptance-criteria.md#ac-14-006--message-order-with-history)–[AC-14.011](ep-acceptance-criteria.md#ac-14-011--vector--session-both-possible)  
  - **Verification:** `handler_test.go` session tests; `make check`.

- [x] **4. Telegram** — Pass chat id string as session key.  
  - _Requirements:_ [REQ-14.003](ep-requirements.md#session-identifier-and-adapter)  
  - _Acceptance Criteria:_ [AC-14.003](ep-acceptance-criteria.md#ac-14-003--telegram-supplies-session-id)  
  - **Verification:** Code review + unit tests for distinct keys; integration mocks updated.

- [x] **5. Wiring** — `newRunConversationHandler` creates store when enabled; integration test helper params.  
  - **Verification:** `core.Run` path; integration tests compile.

- [x] **6. Docs & tests** — `docs/configuration.md`; `Covers AC-14.*` / `Supporting AC-14.012`; defer AC-14.004 in AC doc.  
  - _Requirements:_ [REQ-14.012](ep-requirements.md#nfr--security-testability-operations)–[REQ-14.014](ep-requirements.md#nfr--security-testability-operations)  
  - **Verification:** `./bin/validate EP-014`; `TestDocs_configuration_mentionsConversationSession`.

- [x] **7. SDLC artefacts** — `ep-system-design.md`, `ep-system-design-review.md`, diagrams, this plan, audit.  
  - **Verification:** Links under `ai-sdlc-artefacts/` resolve.

---

## Checkpoints

- [x] `make check` green before audit.
- [x] `./bin/validate EP-014` exit 0 (with AC-14.004 deferred).

---

## Traceability

All in-scope REQ-14.xxx mapped to tasks above; AC-14.004 deferred per [ep-acceptance-criteria.md](ep-acceptance-criteria.md).
