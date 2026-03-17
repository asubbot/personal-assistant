# EP-003 Agent security hardening — Requirements (EARS / INCOSE)

This document contains the product requirements for EP-003 (Agent security hardening) in EARS form, aligned with INCOSE semantic quality rules (active voice, one thought per requirement, explicit and measurable criteria, defined terminology, solution-free where applicable).

**Total: 11 requirements (8 FR, 3 NFR)**

**Contents**

- [Introduction](#introduction)
- [Glossary](#glossary)
- [EARS patterns used](#ears-patterns-used)
- [Requirement index](#requirement-index)
- [Requirements](#requirements)
  - [No arbitrary shell](#no-arbitrary-shell)
  - [Confirmation at tool level](#confirmation-at-tool-level)
  - [Transparent confirmation](#transparent-confirmation)
  - [Session/trace ID](#sessiontrace-id)
  - [E2E tests with mocked tools](#e2e-tests-with-mocked-tools)
  - [Secrets and context](#secrets-and-context)
  - [Destructive actions (future-ready)](#destructive-actions-future-ready)
  - [Non-functional](#non-functional)

---

## Introduction

This document is derived from [ep-scope.md](ep-scope.md). EP-003 builds on EP-001 (PersonalAssistant MVP) and hardens the agent so that tool execution, human-in-the-loop, and observability follow the principle of “reliable systems from unreliable parts”: limited tool set, confirmation enforced at tool level, no arbitrary shell, traceability via a single request ID, and E2E tests that assert on tool usage with mocked tools.

**Epic scope in brief:**

- No general “execute shell” or “terminal” tool; execution only via defined tools (e.g. `run_on_node` with allowlist).
- For sensitive tools (e.g. `run_on_node`), confirmation is required and implemented in code; the LLM does not decide whether to ask the user.
- When the user is asked to confirm, full parameters of the action are visible (no hidden payload via UI truncation).
- Every user request has a session/trace ID propagated through the pipeline and logs.
- At least one E2E test runs the agent loop with mocked tools and asserts on tool call order and parameters.
- No secrets in LLM context or in logs; destructive actions (when added) default to soft delete where applicable.

---

## Glossary

Terms from the project [scope.md](../../scope.md) and [ep-scope.md](ep-scope.md) apply. Epic-specific terms:

| Term | Definition |
|------|-------------|
| **Human-in-the-loop (HITL)** | For sensitive or destructive actions, the tool implementation requires explicit user confirmation before performing the action; the decision is enforced in code, not delegated to the LLM. |
| **Tool-level confirmation** | The decision to ask the user (and what to show) is implemented inside the tool or its caller in the core; the LLM does not decide whether to ask. |
| **Destructive action** | An action that removes or irreversibly alters user data (e.g. delete file, overwrite, send message to external party). |
| **Session/trace ID** | A single identifier for a user request propagated through the entire pipeline (adapter → core → LLM, tool calls, logs) for correlation. |
| **E2E test with mocked tools** | An end-to-end test where the flow is real (user message → core → LLM → tool dispatch) but tools are replaced by mocks; the test asserts on the sequence and parameters of tool invocations. |

---

## EARS patterns used

- **Ubiquitous:** THE \<system\> SHALL \<response\>
- **Event-driven:** WHEN \<trigger\>, THE \<system\> SHALL \<response\>
- **Unwanted event:** IF \<condition\>, THEN THE \<system\> SHALL \<response\>
- **Optional feature:** WHERE \<option\>, THE \<system\> SHALL \<response\>

In the following, *System* = PersonalAssistant (or the relevant component as stated).

---

## Requirement index

| Id | Type | Section | Summary |
|----|------|---------|---------|
| REQ-03.001 | NFR | No arbitrary shell | System SHALL NOT expose a general shell/terminal tool to the agent |
| REQ-03.002 | FR | Confirmation at tool level | Sensitive tools require confirmation implemented in code |
| REQ-03.003 | FR | Confirmation at tool level | LLM SHALL NOT decide whether to ask user for confirmation |
| REQ-03.004 | FR | Transparent confirmation | Full parameters of the action SHALL be visible when user confirms |
| REQ-03.005 | FR | Session/trace ID | Each user request SHALL have a session/trace ID |
| REQ-03.006 | FR | Session/trace ID | Trace ID SHALL be propagated to adapter, core, LLM logger, tool execution, scheduler logs |
| REQ-03.007 | FR | E2E tests with mocked tools | At least one E2E scenario with mocked tools asserting on tool call order/params |
| REQ-03.008 | NFR | Secrets and context | No secrets in text sent to LLM or in operator-visible logs |
| REQ-03.009 | FR | Destructive actions (future-ready) | New destructive tools SHALL default to soft delete where applicable |
| REQ-03.010 | FR | Destructive actions (future-ready) | Irreversible delete only via explicit, separate capability or parameter |
| REQ-03.011 | NFR | Non-functional | New or changed behaviour covered by tests; existing tests pass |

---

## Requirements

### No arbitrary shell

*REQ-03.001*

**REQ-03.001** (NFR)  
THE system SHALL NOT expose a general “execute shell command” or “run terminal” tool to the agent. Execution on nodes SHALL occur only via defined tools (e.g. `run_on_node`) subject to the existing allowlist (REQ-01.004, REQ-01.005). This constraint SHALL be documented in the architecture or security documentation.

---

### Confirmation at tool level

*REQ-03.002, REQ-03.003*

**REQ-03.002** (Ubiquitous)  
THE system SHALL require explicit user confirmation for any tool that performs a sensitive or destructive action (e.g. executing a command on a node). The requirement to ask for confirmation SHALL be implemented in the tool or in the core that invokes the tool (tool-level confirmation), not in the LLM.

**REQ-03.003** (Unwanted event)  
THE system SHALL NOT delegate to the LLM the decision of whether to ask the user for confirmation before performing a sensitive or destructive action. The implementation SHALL enforce confirmation in code regardless of LLM output.

---

### Transparent confirmation

*REQ-03.004*

**REQ-03.004** (Ubiquitous)  
WHEN the user is asked to confirm an action (e.g. run_on_node command), THE system SHALL present the full parameters of that action (e.g. full command string, node_id) in a way that the user can see them without relying on scrolling or truncated fields, so that hidden payloads (e.g. exfiltration in a long string) cannot rely on UI truncation.

---

### Session/trace ID

*REQ-03.005, REQ-03.006*

**REQ-03.005** (Ubiquitous)  
THE system SHALL assign a unique session or trace ID to each user request (e.g. each incoming message from the adapter that triggers the agent loop).

**REQ-03.006** (Ubiquitous)  
THE system SHALL propagate the session/trace ID through the pipeline and SHALL include it in logs produced by the adapter, core, LLM logging subsystem, tool execution, and scheduler (where applicable), so that all events related to a single user request can be correlated.

---

### E2E tests with mocked tools

*REQ-03.007*

**REQ-03.007** (Ubiquitous)  
THE system SHALL have at least one end-to-end test that runs the full agent flow (incoming message → core → LLM → tool dispatch) with tools replaced by mocks, and SHALL assert on the order and parameters of tool invocations (and optionally on the final reply), so that changes to prompts or models can be validated against expected tool usage.

---

### Secrets and context

*REQ-03.008*

**REQ-03.008** (NFR)  
THE system SHALL NOT include secrets (passwords, API keys, tokens, or equivalent) in the plain text sent to the LLM or in log output that is visible to operators. Redaction or out-of-band handling (as in EP-001) SHALL be applied; any new code paths introduced in this epic that handle user or node data SHALL preserve this property.

---

### Destructive actions (future-ready)

*REQ-03.009, REQ-03.010*

**REQ-03.009** (Ubiquitous)  
WHERE the system gains a tool that deletes or overwrites user data, THE system SHALL default to soft delete (e.g. move to trash or archive) where applicable, unless the tool is explicitly defined as irreversible.

**REQ-03.010** (Ubiquitous)  
WHERE the system provides irreversible delete or overwrite, THE system SHALL do so only via an explicit, separate capability or parameter (e.g. a dedicated “permanent delete” tool or an explicit “irreversible” flag), not as the default behaviour of a generic delete tool.

---

### Non-functional

*REQ-03.011*

**REQ-03.011** (NFR)  
New or changed behaviour introduced in this epic SHALL be covered by unit and/or integration tests; all existing tests SHALL continue to pass.
