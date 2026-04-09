# Epic scope — EP-012 Telegram HTML formatting and typing indicator

| Field | Content |
|-------|---------|
| **ID** | EP-012 |
| **Status** | DONE |
| **Title** | Telegram HTML formatting and typing indicator |
| **Description** | Improve Telegram UX: send assistant replies and scheduler notifications as **HTML** (Bot API `parse_mode`) after converting common LLM **Markdown-style** text to Telegram-safe HTML, with **fallback** to plain text if the API rejects entities; show **typing** (`sendChatAction`) while the conversation handler processes an allowed user message. |
| **First version date** | 2026-04-09 |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:

| Term | Definition |
|------|------------|
| **Telegram adapter** | The `internal/telegram` component that runs long polling, filters users, forwards text to the core handler, and sends outbound messages. |
| **Telegram HTML (parse mode)** | The subset of HTML supported by the Telegram Bot API for `sendMessage` (`<b>`, `<i>`, `<code>`, `<pre>`, `<a href="...">`, etc.). |
| **Assistant-oriented Markdown** | Typical model output: headings (`##`), bold (`**`), italic (`*`), inline and fenced code, links, bullet lines, and pipe tables—not full CommonMark. |
| **Typing indicator** | Client UI state triggered by `sendChatAction` with action `typing` for a chat; must be refreshed approximately every five seconds while work continues. |

## Scope (features/capabilities)

- **HTML outbound text:** All outbound user-visible text messages from the Telegram adapter (reply to allowed user and scheduler notifier `SendMessage`) are sent with `parse_mode` **HTML** after deterministic conversion from assistant-oriented Markdown to Telegram-safe HTML (escape unsafe characters; map supported constructs; tables and wide layouts as monospace `<pre>` where needed).
- **Fallback:** If Telegram rejects the message due to entity/HTML parsing, the adapter resends the **same logical content** as plain text **without** `parse_mode` so the user still receives a message.
- **Typing:** For an incoming text message from an allowed user, after allowlist check and **before** `HandleMessage` returns, the adapter sends `typing` and **refreshes** it on an interval (~4–5 s) until `HandleMessage` completes or the request context is cancelled; failures of `sendChatAction` do not block the handler.
- **No config toggle** for parse mode: behaviour is always HTML with fallback (per product decision).
- **Scheduler path** uses the same formatting as conversational replies.
- **Tests:** Unit tests for the converter; adapter tests for `ParseMode`, typing calls, and fallback; AC comments for traceability; `./bin/validate EP-012` passes.

## Success criteria

- Allowed user receives replies with visible formatting (e.g. bold, code) when the model emits corresponding Markdown markers.
- Long-running handler runs show a typing indicator in Telegram (verified manually or via integration; automated tests assert `SendChatAction` usage).
- Scheduler notify messages use HTML parse mode with the same conversion.
- `make check` and `./bin/validate EP-012` succeed.

## Out of scope / deferred

- Native rendered tables as rich Telegram layouts (not supported by the API).
- Splitting messages longer than 4096 characters.
- Typing indicator for scheduler-only outbound messages (no user turn).
- User-configurable parse mode (`plain` vs HTML).

## Traceability

- **Scope:** Extends Telegram interaction in [scope.md](../../scope.md) (user ↔ PersonalAssistant via Telegram bot) without changing the core LLM contract.
- **Strategy:** Aligns with [strategy.md](../../strategy.md): incremental delivery, automated tests, fail-safe fallback for API errors.
