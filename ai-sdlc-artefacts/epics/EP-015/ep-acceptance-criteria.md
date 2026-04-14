# EP-015 — Acceptance criteria

**Pipeline:** Stage 5.  
**Inputs:** [ep-scope.md](ep-scope.md), [ep-requirements.md](ep-requirements.md)

---

## Introduction

This document defines testable acceptance criteria for the Telegram token usage footer (EP-015). Each AC maps to one or more requirements. Automated tests SHALL declare coverage using `// Covers AC-15.NNN` (or equivalent per project validation rules).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-15.001](#ac-15-001) | REQ-15.001, REQ-15.002, REQ-15.004, REQ-15.006 | Multi-completion turn sums usage and appends a correctly formatted footer |
| [AC-15.002](#ac-15-002) | REQ-15.005 | All completions report zero usage → outbound string has no token footer |
| [AC-15.003](#ac-15-003) | REQ-15.007 | Long reply splits into multiple chunks; footer appears only on the last `sendMessage` payload |
| [AC-15.004](#ac-15-004) | REQ-15.008 | Empty assistant body with non-zero usage → no outbound send (or no send containing only the footer) |
| [AC-15.005](#ac-15-005) | REQ-15.009 | Session memory stores assistant text without the token footer line |
| [AC-15.006](#ac-15-006) | REQ-15.007 | Token footer line contains no HTML angle brackets |
| [AC-15.007](#ac-15-007) | REQ-15.010, REQ-15.012 | `./bin/validate EP-015` passes |

---

## Acceptance criteria

<a id="ac-15-001"></a>**AC-15.001** ([REQ-15.001](ep-requirements.md#req-15-001), [REQ-15.002](ep-requirements.md#req-15-002), [REQ-15.004](ep-requirements.md#req-15-004), [REQ-15.006](ep-requirements.md#req-15-006))

Given a user turn that performs at least two successful LLM completions with API usage `prompt_tokens` 10 and 20 and `completion_tokens` 5 and 7 respectively, When the handler returns the Telegram-bound reply string, Then the string ends with a single new line followed by `Tokens 42 (in: 30 / out: 12)`.

---

<a id="ac-15-002"></a>**AC-15.002** ([REQ-15.005](ep-requirements.md#req-15-005))

Given a user turn where every completion returns zero for both `prompt_tokens` and `completion_tokens` in API usage, When the handler returns the Telegram-bound reply string, Then the string does not contain the substring `Tokens ` as a token footer (no footer line).

---

<a id="ac-15-003"></a>**AC-15.003** ([REQ-15.007](ep-requirements.md#req-15-007))

Given an assistant reply body that requires more than one Telegram outbound chunk after Markdown-to-HTML length splitting, and non-zero aggregated usage, When the Telegram adapter sends the reply, Then the token footer substring appears in the final chunk payload and does not appear in any earlier chunk payload.

---

<a id="ac-15-004"></a>**AC-15.004** ([REQ-15.008](ep-requirements.md#req-15-008))

Given the assistant reply body is empty or whitespace-only at end of turn, and aggregated usage is non-zero, When the Telegram adapter finishes processing the reply, Then the adapter does not send a message whose payload consists only of the token footer line.

---

<a id="ac-15-005"></a>**AC-15.005** ([REQ-15.009](ep-requirements.md#req-15-009))

Given sliding session memory is enabled and a user turn returns a non-empty reply with a token footer, When the session store is inspected after the turn, Then the stored assistant text for that exchange does not end with the token footer pattern.

---

<a id="ac-15-006"></a>**AC-15.006** ([REQ-15.007](ep-requirements.md#req-15-007))

Given a non-zero usage footer is produced, When the footer line is inspected, Then the line contains no `<` and no `>` characters.

---

<a id="ac-15-007"></a>**AC-15.007** ([REQ-15.010](ep-requirements.md#req-15-010), [REQ-15.012](ep-requirements.md#req-15-012))

Given the EP-015 acceptance criteria are present in this file, When `./bin/validate EP-015` is executed from the repository root after a successful `make build`, Then the command exits with code zero.
