# Epic scope — EP-014 Sliding session memory window

| Field | Content |
|-------|---------|
| **ID** | EP-014 |
| **Status** | DONE |
| **Title** | Sliding session memory window |
| **Description** | Keep a bounded, ordered list of recent user and assistant **text** exchanges per conversation session and pass it to the LLM on each inbound message so multi-turn clarifications (e.g. room name after “turn on music”) retain intent without relying on vector similarity alone. Complements existing vector memory (conversation turns in `vec_items`) and future EP-002 long-term summarization; does not replace them. |
| **First version date** | 2026-04-10 |

## Glossary

- **Session:** A logical conversation thread identified by a stable key derived from the adapter (e.g. Telegram **chat ID** for private and group chats). Multiple users in a group may share one session key unless a later iteration splits by sender; MVP may document one chosen rule.
- **Sliding window:** A fixed maximum number of **exchanges** (user message + assistant final reply) or **messages** retained in memory; oldest entries are dropped when the limit is exceeded.
- **Exchange (turn pair):** One user text and the assistant’s **final** text reply visible to the user for that `HandleMessage` invocation (after tool rounds complete). Intermediate tool messages remain internal to that invocation and are not duplicated into the session window unless explicitly specified in design.
- **Vector memory:** Existing semantic retrieval from `vec_items` (and future EP-002 dated summaries); injected in the CONTEXT (`PA_BEGIN_CONTEXT` / `PA_END_CONTEXT`) block in the system tail.
- **Working memory:** The session sliding window carried in the LLM `messages` array (roles `user` / `assistant`), distinct from vector-injected snippets.

## Scope (features/capabilities)

- **Session store:** In-process structure keyed by session id; each key holds an ordered deque or ring buffer of exchange records (user text, assistant reply text, optional monotonic sequence or timestamp for debugging). Thread-safe updates for concurrent messages per session if applicable.
- **Configuration:** Feature flag to enable/disable session window; numeric cap (e.g. `max_session_exchanges` or `max_session_messages` with a defined semantics); optional per-adapter defaults documented. Fail fast on invalid bounds (e.g. zero or negative when enabled).
- **Adapter contract:** The core receives enough information to compute the session key (minimum: **chat ID** for Telegram; extend `MessageHandler` or equivalent so private vs group behaviour is correct). Document interaction with existing `userID`-only call sites and tests.
- **Prompt assembly:** When enabled, build `[]llm.Message` as: `system` (unchanged head + dynamic tail: tools, Hermes, retrieved context, runtime skills per existing ordering), then **historical** `user`/`assistant` pairs from the window (oldest to newest), then the **current** user message. No change to tool-round message shape inside a single `HandleMessage` beyond what design specifies.
- **Update rule:** After a successful reply to the user (or after the handler returns the final assistant text), append the current user text and assistant reply to the session store for that key; do not append on early rejection (empty message, over max length) unless design says otherwise.
- **Clear / reset:** Operator or user-visible reset is optional for MVP; if omitted, document that restart clears in-memory state. Optional hook for `/clear`-style behaviour may be a follow-up.
- **Privacy and logs:** Session buffer holds the same categories of data as normal chat; align with log redaction policy for debug logs. Do not persist session window to disk in MVP unless explicitly added in scope (default: memory-only).
- **Interaction with vector memory:** Session window and the CONTEXT marker block may overlap semantically; MVP may include both (acceptable duplication) or a simple de-duplication rule documented in system design. EP-002 (when implemented) remains the owner of dated summaries and chunk-type labels; this epic does not implement summarization of the session window.
- **Tests:** Unit tests for store eviction order, cap, and concurrency; integration test proving two Telegram-equivalent messages where the second depends on the first without relying on vector retrieval (mock vector empty or orthogonal query).
- **Documentation:** Short operator note (config keys, semantics of exchange vs message cap) in existing user-facing doc location per project convention.

## Success criteria

- With session window enabled and cap ≥ 1, a two-step dialogue (user A → assistant → user B that only supplies a missing slot) results in an LLM request whose **messages** include prior user and assistant texts in order, such that the model can complete the original intent without re-asking for the primary action (verified by test or structured assertion on assembled messages).
- With session window disabled, behaviour matches pre-epic assembly (regression: single user message after system).
- Invalid configuration fails at startup with a clear error.
- `./bin/validate EP-014` passes once acceptance criteria and test traces exist per repository convention.

## Out of scope / deferred

- Persisting session history across process restart (disk/sqlite sync).
- Cross-channel shared session (OpenClaw-style `cross_channel`).
- LLM-based summarization or compaction of the session window (may interact with EP-002 later).
- Changing the merged system prompt contract from EP-013 (marker blocks, trust policy) except where order of **message** roles is specified above.
- Replacing vector memory or EP-002 date-based injection.

## Traceability

- **Scope:** Addresses **Core** orchestration and **conversational coherence** from [scope.md](../../scope.md): the assistant should use context across user turns within a session, not only semantic recall.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) testable increments: new behaviour covered by automated tests; KISS and fail fast for configuration.
- **Related epics:** Complements **EP-001** / **REQ-01.006–01.007** (vector context injection); orthogonal to **EP-002** (automatic summarization and date-aware long-term memory). Interacts with **EP-013** (system tail ordering remains; session history is additional `user`/`assistant` messages). **EP-012** (Telegram) is the primary adapter for session key plumbing.
