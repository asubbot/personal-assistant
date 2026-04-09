# EP-012 — Acceptance criteria

**Pipeline:** Stage 5.  
**Inputs:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md)

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-12.001](#ac-12-001) | REQ-12.001, REQ-12.003 | Converter maps `**bold**` and `` `code` `` to Telegram HTML |
| [AC-12.002](#ac-12-002) | REQ-12.002 | Raw `<script>` in source is escaped, not executed as markup |
| [AC-12.003](#ac-12-003) | REQ-12.001 | Adapter sends user reply with `ParseModeHTML` |
| [AC-12.004](#ac-12-004) | REQ-12.004 | On HTML parse rejection from API, adapter resends plain text without parse mode |
| [AC-12.005](#ac-12-005) | REQ-12.001 | Scheduler notifier `SendMessage` uses `ParseModeHTML` |
| [AC-12.006](#ac-12-006) | REQ-12.005 | While handler runs, adapter issues `typing` chat action (initial + refresh) |
| [AC-12.007](#ac-12-007) | REQ-12.006 | `sendChatAction` failure does not prevent handler from running |

---

## Acceptance criteria

<a id="ac-12-001"></a>**AC-12.001** ([REQ-12.001](ep-requirements.md#req-12-001), [REQ-12.003](ep-requirements.md#req-12-003))

Given a source string containing `**bold**` and inline `` `x` ``, When the converter runs, Then the output contains `<b>` / `</b>` around bold text and `<code>` / `</code>` around `x`, with HTML metacharacters in plain text escaped.

---

<a id="ac-12-002"></a>**AC-12.002** ([REQ-12.002](ep-requirements.md#req-12-002))

Given a source string containing literal `<script>`, When the converter runs, Then the output does not contain an unescaped literal tag sequence that Telegram would interpret as an HTML tag opening for `script`.

---

<a id="ac-12-003"></a>**AC-12.003** ([REQ-12.001](ep-requirements.md#req-12-001))

Given an allowed user message and a handler that returns non-empty text, When `handleUpdate` completes, Then the outbound `sendMessage` call uses `ParseMode` HTML and non-empty text.

---

<a id="ac-12-004"></a>**AC-12.004** ([REQ-12.004](ep-requirements.md#req-12-004))

Given the first `sendMessage` fails with a Telegram bad request whose description indicates entity/HTML parse failure, When the adapter handles outbound text, Then a second `sendMessage` is issued without `parse_mode` with the original source string.

---

<a id="ac-12-005"></a>**AC-12.005** ([REQ-12.001](ep-requirements.md#req-12-001))

Given a configured notifier chat and a running bot handle, When `Adapter.SendMessage` is called with non-empty text, Then the `sendMessage` call uses `ParseMode` HTML.

---

<a id="ac-12-006"></a>**AC-12.006** ([REQ-12.005](ep-requirements.md#req-12-005))

Given an allowed user sends text and the handler blocks longer than one typing refresh interval, When the update is processed, Then `sendChatAction` with `typing` is invoked at least twice for that chat (initial and refresh).

---

<a id="ac-12-007"></a>**AC-12.007** ([REQ-12.006](ep-requirements.md#req-12-006))

Given `sendChatAction` always returns an error, When an allowed user message is processed, Then the handler still runs and a reply is still sent when the handler returns successfully.
