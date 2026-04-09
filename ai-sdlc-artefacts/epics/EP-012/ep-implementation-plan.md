# EP-012 — Implementation plan

**Pipeline:** Stage 7.

---

## Task order

1. **Converter** — Add `internal/telegram/format.go` and `format_test.go` implementing `MarkdownToTelegramHTML`; tests with `Covers AC-12.001`, `Covers AC-12.002`.
2. **Outbound send helper** — Implement `sendOutboundText` with HTML mode and fallback; wire into `SendMessage` (notifier) and `handleUpdate` for all outbound texts; tests `Covers AC-12.003`, `Covers AC-12.004`, `Covers AC-12.005`.
3. **Typing indicator** — Goroutine + ticker in `handleUpdate` for allowed-user text path; extend mock with `SendChatAction`; tests `Covers AC-12.006`, `Covers AC-12.007`.
4. **Verification** — `make check`; `./bin/validate EP-012`.
5. **Audit** — Update `ep-audit-report.md` (stage 9).

---

## Verification

- `make check`
- `./bin/validate EP-012`

---

## Notes

- Update existing adapter tests that assert raw message text to expect HTML where the converter changes content, or assert on `ParseMode` + structured checks.
- Disallowed-user and some error messages are plain English without Markdown; converter should still produce safe HTML (escaped plain text).
