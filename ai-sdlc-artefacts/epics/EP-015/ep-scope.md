# Epic scope — EP-015 Telegram token usage footer


| Field                  | Content                                                                                                                                                                                                                                                                                   |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ID**                 | EP-015                                                                                                                                                                                                                                                                                    |
| **Status**             | DONE                                                                                                                                                                                                                                                                                      |
| **Title**              | Telegram token usage footer                                                                                                                                                                                                                                                               |
| **Description**        | After processing one user message in Telegram, append a token summary (Markdown `*…*` so Telegram shows **italic**) to the **last** outbound message chunk when the LLM provider reported **non-zero** usage for that turn. All figures come from **API `usage`** fields aggregated across every completion in the turn. |
| **First version date** | 2026-04-14                                                                                                                                                                                                                                                                                |


## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:


| Term               | Definition                                                                                                                                                                                         |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Turn**           | One inbound Telegram text message from an allowed user through `HandleMessage` until the assistant reply string is produced and sent (including intermediate LLM completions such as tool rounds). |
| **API usage**      | Token counts returned by the LLM provider on a completion response (`prompt_tokens`, `completion_tokens`, and optionally `total_tokens` in OpenAI-compatible payloads).                            |
| **Outbound chunk** | One Telegram `sendMessage` payload after splitting assistant text to satisfy the 4096-character limit (see `splitTelegramOutboundSource` / `sendLongOutboundText`).                                |
| **Token footer**   | A single trailing Markdown line summarising aggregated usage for the turn (`*Tokens …*`), converted with the body to Telegram HTML.                                                                 |


## Scope (features/capabilities)

- **Telegram user replies only:** The token footer applies to assistant replies sent to an allowed user on the conversational path (the same path that uses chunked outbound sending today). Scheduler notifier messages and other non-conversational Telegram sends are **out of scope** unless a later epic extends this.
- **Source of numbers:** Only values taken from provider-reported **API usage** on LLM completions invoked during the turn. No client-side token estimation (no tiktoken, character heuristics, or similar).
- **Aggregation:** For the turn, sum **prompt_tokens** across all completions into **in**, and sum **completion_tokens** across all completions into **out**. Treat missing usage on a completion as zero for that call. The displayed total **Tokens** value must equal **in + out** (so the line is arithmetically consistent with the parenthetical).
- **When to show:** Append the footer **only** if, after aggregation, **in > 0 or out > 0** (i.e. at least one non-zero counted token from the API for the turn). If all completions omit usage or all summed values are zero, **do not** append a footer.
- **Where to show:** Append the footer to the **last** outbound chunk only (after chunking). Earlier chunks must remain unchanged. If the reply is empty, there is no chunk to append to—**do not** send a usage-only message.
- **Format (exact inner shape):** One new line at the end of the logical reply, wrapped for Telegram italic as `*Tokens <in+out> (in: <in> / out: <out>)*` with non-negative integers and a single space after `Tokens` and after the colon in `in:` and `out:`. Example: `*Tokens 1801 (in: 1234 / out: 567)*`. The adapter still accepts a legacy footer without asterisks for suffix detection only.
- **Markdown for footer:** The footer uses **only** a single pair of ASCII `*` around the inner pattern so `MarkdownToTelegramHTML` produces `<i>…</i>`. No other Markdown constructs, no raw `<` or `>` inside the inner pattern, no Telegram HTML typed by hand in the footer.
- **Tests:** Automated coverage for aggregation, conditional append, last-chunk-only behaviour, and interaction with splitting near the size limit; `./bin/validate EP-015` passes when AC artefacts exist.

## Success criteria

- For a mocked multi-completion turn with non-zero `usage` on each call, the user sees **one** footer line on the **final** chunk only, with **in** and **out** equal to the **sums** of API fields and **Tokens** equal to their sum; the handler reply ends with `*Tokens …*` (italic after Telegram conversion).
- When all completions return zero or omit usage fields, the outbound text has **no** token footer.
- When the assistant body is split into multiple Telegram messages, only the **last** message includes the footer; prior chunks match the body-only content.
- `make check` succeeds for the delivered implementation.

## Out of scope / deferred

- Token display for scheduler `SendMessage` or other adapters.
- Local token counting or reconciliation when API `total_tokens` disagrees with `prompt_tokens` + `completion_tokens` on a single response (this epic uses summed in/out and **Tokens = in + out** only).
- User-facing configuration toggles for showing or hiding the footer.
- Rich formatting of the footer beyond one Markdown italic pair (bold, monospace, spoiler, raw HTML, etc.).

## Traceability

- **Scope:** Supports the Telegram-centred MVP interaction described in [scope.md](../../scope.md) (user talks to PersonalAssistant via the Telegram bot) by making per-turn LLM cost visible from provider data.
- **Strategy:** Aligns with [strategy.md](../../strategy.md): incremental, testable behaviour; no change to high-level security posture beyond surfacing non-secret numeric usage.

