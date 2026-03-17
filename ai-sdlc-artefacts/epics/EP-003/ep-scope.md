# Epic scope — EP-003 Agent security hardening

## Introduction

This document is the epic scope for EP-003 (Agent security hardening). It builds on the system delivered in EP-001 (PersonalAssistant MVP) and is independent of EP-002 (Automatic memory summarization). It is aligned with project [scope.md](../../scope.md) and [strategy.md](../../strategy.md). The epic is based on analysis of agent security practices (Habr “Халява уходит из разработки Агентов”, CrowdStrike AI Tool Poisoning, AuthZed MCP breaches timeline, Invariant Labs guardrails/WhatsApp MCP, Anthropic writing tools for agents). Requirements, acceptance criteria, and implementation details are produced in later pipeline stages.

## Epic ID, title, short description

| Field | Content |
|-------|---------|
| **ID** | EP-003 |
| **Status** | NEW |
| **Title** | Agent security hardening |
| **Description** | Harden the PersonalAssistant agent so that tool execution, human-in-the-loop, observability, and constraints align with “reliable systems from unreliable parts”: limited tool set, confirmation at tool level, no arbitrary shell, traceability, and E2E tests with mocked tools. |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:

| Term | Definition |
|------|-------------|
| **Human-in-the-loop (HITL)** | For sensitive or destructive actions, the tool implementation requires explicit user confirmation before performing the action; the decision is enforced in code, not delegated to the LLM. |
| **Tool-level confirmation** | The decision to ask the user (and what to show) is implemented inside the tool or its caller in the core; the LLM does not decide whether to ask. |
| **Destructive action** | An action that removes or irreversibly alters user data (e.g. delete file, overwrite, send message to external party). Where feasible, default to soft delete (e.g. move to trash) rather than irreversible delete. |
| **Session/trace ID** | A single identifier for a user request that is propagated through the entire pipeline (adapter → core → LLM, tool calls, scheduler, logs) so that all related events can be correlated. |
| **E2E test with mocked tools** | An end-to-end test where the flow is real (user message → core → LLM → tool dispatch) but tools are replaced by mocks; the test asserts on the sequence and parameters of tool invocations (and optionally LLM output). |
| **Allowlist (this epic)** | The existing per-node command allowlist (REQ-01.004, REQ-01.005, AC-01.007, AC-01.008): only full commands on the allowlist are executed; no general shell or arbitrary command execution. |

## Scope (features/capabilities)

- **Confirmation at tool level:** For any tool that performs a sensitive or destructive action (today: `run_on_node` is the only execution tool; future tools such as “delete”, “send email” will be in scope when added), the core or tool enforces a confirmation step. What is shown to the user (e.g. full command, recipient, body) is determined by the implementation, not by the LLM. The LLM does not decide whether to ask the user.
- **Transparent confirmation UI:** When the user is asked to confirm, the full parameters of the action (e.g. full command string, full recipient, full message body) are visible without requiring scroll or hidden fields, so that “hidden payload” attacks (e.g. exfiltration hidden in a long string) cannot rely on UI truncation.
- **Session/trace ID:** Every user message triggers a session or trace ID that is passed through the pipeline and included in logs (adapter, core, LLM logger, tool execution, scheduler). This enables “why did this tool run?” and “what was the full context for this reply?” analysis.
- **E2E tests with mocked tools:** At least one E2E scenario runs the full path (incoming message → core → LLM → tool dispatch) with tools mocked; the test asserts on the order and parameters of tool calls (and optionally on the final reply). This supports safe evolution of prompts and models while guarding behaviour.
- **No arbitrary shell:** The system SHALL NOT expose a general “execute shell command” or “run terminal” tool to the agent. Execution is only via defined tools (e.g. `run_on_node` with allowlist). This is a constraint preserved and documented, not a new feature.
- **Secrets and context:** No secrets (passwords, API keys, tokens) are included in the text sent to the LLM or in log output visible to operators; redaction or out-of-band handling is already in scope from EP-001; this epic reinforces and verifies it where it touches new code.
- **Destructive actions (future-ready):** If the epic or a later epic adds tools that “delete” or “overwrite”, the default behaviour SHALL be soft delete (e.g. move to trash) where applicable; irreversible delete only via an explicit, separate capability or parameter.

## Success criteria

- **Tool-level confirmation:** For `run_on_node` (and any future sensitive tool), confirmation is required and implemented in code; the full command (and other parameters) is available to the user when confirming.
- **Trace ID:** Each request has a trace/session ID in logs across adapter, core, LLM, and tool execution.
- **E2E with mocks:** At least one E2E test runs the agent loop with mocked tools and asserts on tool call sequence and parameters.
- **Documented constraints:** The “no arbitrary shell” and “confirmation at tool level” constraints are documented in the epic and in the codebase (e.g. architecture or security notes).
- **Tests:** New or changed behaviour is covered by unit and/or integration tests; existing tests continue to pass.

## Out of scope / deferred

- **MCP integration:** Introducing or securing MCP (Model Context Protocol) servers is out of scope. If MCP is introduced later, a separate epic will define controls (e.g. no trust in third-party tool descriptions, tool poisoning mitigations).
- **Guardrails based on regex/classifiers:** Building safety primarily on content classifiers (e.g. “detect PII”, “detect dangerous command”) is out of scope; this epic relies on architectural constraints (limited tools, allowlist, confirmation at tool level).
- **Sandboxing the agent process:** Isolating the agent in a separate sandbox (e.g. container, VM) is out of scope; the focus is on tool contracts, confirmation, and observability.
- **New tools:** Implementing new tools (e.g. read file, send email) is out of scope unless explicitly added to this epic; the epic may define how such tools SHALL behave (confirmation, soft delete) when they are added later.

## Traceability

- **Scope:** This epic reinforces the Security model and Tool contract from [scope.md](../../scope.md): explicit definition of permitted actions, validation at tool level, and no reliance on the LLM for security decisions.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) test strategy: security checks (allowlists, no secrets in context/logs), and E2E coverage for agent behaviour.
- **Dependency:** Builds on EP-001 (PersonalAssistant MVP). Independent of EP-002; can be developed in parallel or after EP-002.
- **Sources:** [Habr: «Халява уходит из разработки Агентов»](https://habr.com/ru/articles/1010236/), CrowdStrike AI Tool Poisoning, AuthZed “Timeline of MCP Security Breaches”, Invariant Labs (Guardrails, WhatsApp MCP), Anthropic “Writing effective tools for agents”.
