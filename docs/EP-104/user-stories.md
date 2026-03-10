# EP-104: User Stories (project source of truth)

**Epic:** EP-104  
**Related:** [REQUIREMENTS.md](REQUIREMENTS.md), [acceptance-criteria.md](acceptance-criteria.md), [test-strategy.md](test-strategy.md)

This document is the project’s canonical list of user stories. IDs: **US-01** … **US-18**.

| ID | Title |
|----|--------|
| [US-01](#us-01--telegram-bot) | Telegram bot — receive and reply to messages |
| [US-02](#us-02--docker-deploy) | Docker deploy — run core on DS220+ |
| [US-03](#us-03--node-config) | Node config — define and validate at startup |
| [US-04](#us-04--per-node-allowlist) | Per-node allowlist — security model |
| [US-05](#us-05--dedicated-pa-user-per-node) | Dedicated PA user per node for SSH |
| [US-06](#us-06--memory-store) | Memory store — long-term markdown files |
| [US-07](#us-07--vector-search) | Vector index and semantic search |
| [US-08](#us-08--pluggable-llm-provider) | Pluggable LLM provider |
| [US-09](#us-09--llm-logging) | LLM request/response logging |
| [US-10](#us-10--log-destination-and-format) | Configurable log destination and format |
| [US-11](#us-11--scheduled-tasks) | Scheduled tasks (time or interval) |
| [US-12](#us-12--extensible-tools) | Extensible tools with single contract |
| [US-13](#us-13--add-nodestools-without-rebuild) | Add nodes and tools without image rebuild |
| [US-14](#us-14--architecture-boundaries) | Clear architecture boundaries |
| [US-15](#us-15--version-control-git) | Version control for config and memory (git) |
| [US-16](#us-16--secret-leakage-protection) | Secret leakage protection (prompt injection) |
| [US-17](#us-17--debug-llm-logging) | Debug-level LLM conversation logging |
| [US-18](#us-18--verify-node-availability) | Verify node availability via CLI parameter |

---

## US-01 — Telegram bot

As a user, I want to send text messages to the assistant via a Telegram bot and receive text replies, so that I can interact without installing a separate app.

**Requirements:** [REQ-001](REQUIREMENTS.md#interface-and-deployment).  
**Acceptance criteria:** [AC-001](acceptance-criteria.md#ac-001-us-01), [AC-002](acceptance-criteria.md#ac-002-us-01).

---

## US-02 — Docker deploy

As an operator, I want to run the PersonalAssistant core as a single Docker container (including on Synology DS220+), so that I can deploy with one command.

**Requirements:** [REQ-002](REQUIREMENTS.md#interface-and-deployment).  
**Acceptance criteria:** [AC-003](acceptance-criteria.md#ac-003-us-02), [AC-004](acceptance-criteria.md#ac-004-us-02).

---

## US-03 — Node config

As an operator, I want to define nodes (host, SSH user, authentication) in configuration and have the system validate at startup, so that configuration errors are caught before serving.

**Requirements:** [REQ-003](REQUIREMENTS.md#nodes-and-ssh), [REQ-004](REQUIREMENTS.md#nodes-and-ssh).  
**Acceptance criteria:** [AC-005](acceptance-criteria.md#ac-005-us-03), [AC-006](acceptance-criteria.md#ac-006-us-03).

---

## US-04 — Per-node allowlist

As an operator, I want a documented security model that defines, per node, which commands or tools are allowed, so that only permitted actions run on each node.

**Requirements:** [REQ-005](REQUIREMENTS.md#nodes-and-ssh).  
**Acceptance criteria:** [AC-007](acceptance-criteria.md#ac-007-us-04), [AC-008](acceptance-criteria.md#ac-008-us-04).

---

## US-05 — Dedicated PA user per node

As an operator, I want to configure one dedicated user account per node for PersonalAssistant SSH access, so that all actions are attributed to that identity.

**Requirements:** [REQ-013](REQUIREMENTS.md#nodes-and-ssh).  
**Acceptance criteria:** [AC-009](acceptance-criteria.md#ac-009-us-05), [AC-010](acceptance-criteria.md#ac-010-us-05).

---

## US-06 — Memory store

As the assistant (system), I want to store long-term memory in markdown files in a defined directory structure, so that data is human-readable and easy to back up.

**Requirements:** [REQ-006](REQUIREMENTS.md#memory-and-indexing), [REQ-018](REQUIREMENTS.md#memory-and-indexing), [REQ-019](REQUIREMENTS.md#memory-and-indexing), [REQ-020](REQUIREMENTS.md#memory-and-indexing).  
**Acceptance criteria:** [AC-011](acceptance-criteria.md#ac-011-us-06), [AC-012](acceptance-criteria.md#ac-012-us-06).

---

## US-07 — Vector search

As the assistant (system), I want to index long-term memory in a vector store and run semantic search, so that relevant context can be retrieved for replies.

**Requirements:** [REQ-007](REQUIREMENTS.md#memory-and-indexing).  
**Acceptance criteria:** [AC-013](acceptance-criteria.md#ac-013-us-07), [AC-014](acceptance-criteria.md#ac-014-us-07).

---

## US-08 — Pluggable LLM provider

As an operator, I want to choose and configure the LLM provider via configuration without code changes, so that I can avoid vendor lock-in.

**Requirements:** [REQ-008](REQUIREMENTS.md#llm-and-logging).  
**Acceptance criteria:** [AC-015](acceptance-criteria.md#ac-015-us-08), [AC-016](acceptance-criteria.md#ac-016-us-08).

---

## US-09 — LLM logging

As an operator or developer, I want a logging subsystem that records each LLM request and response, so that I can analyse usage and perform audits.

**Requirements:** [REQ-014](REQUIREMENTS.md#llm-and-logging).  
**Acceptance criteria:** [AC-017](acceptance-criteria.md#ac-017-us-09).

---

## US-10 — Log destination and format

As an operator, I want to configure where LLM logs are written and in what parseable format, so that I can control retention and analysis.

**Requirements:** [REQ-015](REQUIREMENTS.md#llm-and-logging).  
**Acceptance criteria:** [AC-018](acceptance-criteria.md#ac-018-us-10), [AC-019](acceptance-criteria.md#ac-019-us-10).

---

## US-11 — Scheduled tasks

As an operator, I want to define scheduled tasks (time or interval) in configuration, so that the assistant can run periodic actions within the security model.

**Requirements:** [REQ-009](REQUIREMENTS.md#scheduler-and-tools).  
**Acceptance criteria:** [AC-020](acceptance-criteria.md#ac-020-us-11), [AC-021](acceptance-criteria.md#ac-021-us-11).

---

## US-12 — Extensible tools

As a developer, I want to add new tools via a single contract without changing core orchestration code, so that capabilities can be extended in a modular way.

**Requirements:** [REQ-010](REQUIREMENTS.md#scheduler-and-tools).  
**Acceptance criteria:** [AC-022](acceptance-criteria.md#ac-022-us-12), [AC-023](acceptance-criteria.md#ac-023-us-12).

---

## US-13 — Add nodes/tools without rebuild

As an operator, I want to add new nodes and register new tools through configuration, so that I can scale without rebuilding the core image.

**Requirements:** [REQ-011](REQUIREMENTS.md#extensibility-and-architecture).  
**Acceptance criteria:** [AC-024](acceptance-criteria.md#ac-024-us-13).

---

## US-14 — Architecture boundaries

As an architect or developer, I want the system to clearly separate adapters, core, memory, vector, LLM, scheduler, and tools, so that we can evolve or replace each part.

**Requirements:** [REQ-012](REQUIREMENTS.md#extensibility-and-architecture).  
**Acceptance criteria:** [AC-025](acceptance-criteria.md#ac-025-us-14).

---

## US-15 — Version control (git)

As an operator, I want the assistant to use a git repository to track configuration, memory, and designated data, so that I can review history and roll back if needed.

**Requirements:** [REQ-016](REQUIREMENTS.md#version-control-and-audit).  
**Acceptance criteria:** [AC-026](acceptance-criteria.md#ac-026-us-15), [AC-027](acceptance-criteria.md#ac-027-us-15).

---

## US-16 — Secret leakage protection

As an operator or security-conscious user, I want the assistant to never expose secret values in LLM context, user-facing responses, or logs, so that credentials cannot be extracted via crafted prompts.

**Requirements:** [REQ-017](REQUIREMENTS.md#secret-protection-prompt-injection--exfiltration).  
**Acceptance criteria:** [AC-028](acceptance-criteria.md#ac-028-us-16), [AC-029](acceptance-criteria.md#ac-029-us-16), [AC-030](acceptance-criteria.md#ac-030-us-16).

---

## US-17 — Debug-level LLM logging

As a developer or operator, I want to enable debug logging for LLM conversations via `PA_LOG_LEVEL=debug`, so that I can inspect the full request (including memory and vector context) and response when troubleshooting. By default (INFO level) only metadata is logged.

**Requirements:** [REQ-021](REQUIREMENTS.md#llm-and-logging).  
**Acceptance criteria:** [AC-031](acceptance-criteria.md#ac-031-us-17).

---

## US-18 — Verify node availability

As an operator, I want to run the PersonalAssistant binary with a dedicated parameter to verify that SSH access to all configured nodes works, so that I can confirm credentials and allowlist without starting the bot.

**Requirements:** [REQ-022](REQUIREMENTS.md#nodes-and-ssh).  
**Acceptance criteria:** [AC-032](acceptance-criteria.md#ac-032-us-18).
