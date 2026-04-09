# EP-012 — Audit report

**Date and time of creation:** 2026-04-09 (UTC)  
**Pipeline:** [pipeline.spec.md](../../../ai-sdlc/specification/pipeline.spec.md) stage 9  
**Branch:** `epic/EP-012-telegram-html-typing`  
**Related:** [ep-implementation-plan.md](ep-implementation-plan.md) · [ep-acceptance-criteria.md](ep-acceptance-criteria.md) · [ep-requirements.md](ep-requirements.md) · [ep-system-design.md](ep-system-design.md)

## Summary

Telegram outbound messages use **HTML** parse mode via `MarkdownToTelegramHTML` in [`internal/telegram/format.go`](../../../internal/telegram/format.go); on Telegram **bad request** errors that indicate **entity/HTML parse** failure, the adapter **retries once** with plain text and no parse mode ([`internal/telegram/adapter.go`](../../../internal/telegram/adapter.go) `sendOutboundText`). Allowed-user turns run a **typing** refresh loop (`sendChatAction` `typing`) on a **4s** interval (configurable in tests via atomic `typingRefreshNs`); `sendChatAction` errors do not block the handler.

**Verification:** `make check` **passed**; `./bin/validate EP-012` — **7/7** AC traced (100%). Project-wide statement coverage from `make check` output: **~73.4%** (`total: (statements)`).

## Implementation vs plan

| Area | Status | Notes |
|------|--------|--------|
| Converter + tests | **Done** | `format.go`, `format_test.go` |
| `sendOutboundText` + fallback | **Done** | `isEntityParseError`, `errors.Is(bot.ErrorBadRequest)` + description substring |
| User replies + notifier | **Done** | `handleUpdate`, `SendMessage` |
| Typing goroutine | **Done** | `runTypingRefresh`, atomic interval for race-safe tests |
| AC trace comments | **Done** | `Covers AC-12.*` on tests |

## Test results

- **Command:** `make check` (fmt, vet, govulncheck, golangci-lint, `go test -race -tags=integration ./...`, coverage, module boundaries).
- **AC validation:** `./bin/validate EP-012` — exit 0.

## REQ/AC coverage matrix

| AC | Primary tests |
|----|----------------|
| AC-12.001 | `format_test.go` |
| AC-12.002 | `format_test.go` |
| AC-12.003 | `adapter_test.go` `TestHandleUpdate_allowedUser_callsHandlerAndSendsReply` |
| AC-12.004 | `adapter_test.go` `TestSendOutboundText_entityErrorRetriesPlain` |
| AC-12.005 | `adapter_test.go` `TestSendOutboundText_schedulerStyle_usesHTMLParseMode` |
| AC-12.006 | `adapter_test.go` `TestHandleUpdate_typingRefreshedDuringSlowHandler` |
| AC-12.007 | `adapter_test.go` `TestHandleUpdate_chatActionErrorStillSendsReply` |

## Quality gate

- **golangci-lint:** pass  
- **govulncheck:** no vulnerabilities reported  
- **Module boundaries:** OK  
- **`./bin/validate EP-012`:** all AC covered  

## Gaps / recommendations

- **Manual:** Confirm typing indicator and formatted replies in a real Telegram client (not required for automated gate).
- **Deferred (scope):** Message splitting beyond 4096 characters; rich table rendering beyond `<pre>`.
