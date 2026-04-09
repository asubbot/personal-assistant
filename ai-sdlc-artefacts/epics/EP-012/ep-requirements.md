# EP-012 — Requirements

**Pipeline:** Stage 4 (Requirements).  
**Inputs:** [ep-scope.md](ep-scope.md)

---

## Introduction

This document specifies requirements for **EP-012 Telegram HTML formatting and typing indicator**. Requirements use EARS patterns and stable IDs **REQ-12.NNN**.

---

## Glossary

See [ep-scope.md](ep-scope.md) glossary and project [scope.md](../../scope.md).

---

## Requirements

### Outbound message formatting

<a id="req-12-001"></a>**REQ-12.001** — HTML parse mode for outbound text  
WHEN the Telegram adapter sends a text message to a Telegram chat (user reply or scheduler notifier), THE Telegram adapter SHALL set Bot API `parse_mode` to HTML and SHALL set message text to the result of converting the source string with the project’s Markdown-to-Telegram-HTML converter.

<a id="req-12-002"></a>**REQ-12.002** — Converter safety  
THE Markdown-to-Telegram-HTML converter SHALL escape characters `&`, `<`, and `>` in text nodes so that user or model content cannot inject arbitrary HTML outside allowed tags.

<a id="req-12-003"></a>**REQ-12.003** — Supported Markdown mappings  
WHERE the source string contains assistant-oriented Markdown, THE converter SHALL map at minimum: fenced code blocks to `<pre>`; inline code (backticks) to `<code>`; lines starting with `##` or `###` to a bold line equivalent; `**...**` to bold; single-asterisk emphasis to italic where unambiguous; `http`/`https` links `[label](url)` to `<a href="url">label</a>`; consecutive table-style lines (pipe-separated) to a single `<pre>` block.

<a id="req-12-004"></a>**REQ-12.004** — Fallback on entity error  
IF Telegram returns an error indicating that the message text cannot be parsed as HTML entities, THEN THE Telegram adapter SHALL send the same source string in a follow-up `sendMessage` call without `parse_mode`.

### Typing indicator

<a id="req-12-005"></a>**REQ-12.005** — Typing for user turns  
WHEN an allowed user sends a text message that reaches the conversation handler, THE Telegram adapter SHALL call `sendChatAction` with action `typing` for that chat before awaiting the handler result and SHALL refresh that action at least every five seconds until the handler returns or the request context is cancelled.

<a id="req-12-006"></a>**REQ-12.006** — Typing errors non-blocking  
IF `sendChatAction` fails, THEN THE Telegram adapter SHALL continue processing the conversation without failing the turn solely for that reason.

---

## Traceability

| REQ | ep-scope capability |
|-----|---------------------|
| REQ-12.001 – REQ-12.004 | HTML outbound text, fallback |
| REQ-12.005 – REQ-12.006 | Typing indicator |
