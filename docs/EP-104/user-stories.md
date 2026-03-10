# EP-104: User Stories (project source of truth)

**Epic:** EP-104  
**Related:** [REQUIREMENTS.md](REQUIREMENTS.md), [acceptance-criteria.md](acceptance-criteria.md), [test-strategy.md](test-strategy.md)

This document is the project’s canonical list of user stories. IDs: **US-01** … **US-16**.

| ID    | Title |
|-------|--------|
| US-01 | Telegram bot — receive and reply to messages |
| US-02 | Docker deploy — run core on DS220+ |
| US-03 | Node config — define and validate at startup |
| US-04 | Per-node allowlist — security model |
| US-05 | Dedicated PA user per node for SSH |
| US-06 | Memory store — long-term markdown files |
| US-07 | Vector index and semantic search |
| US-08 | Pluggable LLM provider |
| US-09 | LLM request/response logging |
| US-10 | Configurable log destination and format |
| US-11 | Scheduled tasks (time or interval) |
| US-12 | Extensible tools with single contract |
| US-13 | Add nodes and tools without image rebuild |
| US-14 | Clear architecture boundaries |
| US-15 | Version control for config and memory (git) |
| US-16 | Secret leakage protection (prompt injection) |

---

## US-01 — Telegram bot

As a user, I want to send text messages to the assistant via a Telegram bot and receive text replies, so that I can interact without installing a separate app.  
**Requirements:** REQ-001. **AC:** AC-001, AC-002.

---

## US-02 — Docker deploy

As an operator, I want to run the PersonalAssistant core as a single Docker container (including on Synology DS220+), so that I can deploy with one command.  
**Requirements:** REQ-002. **AC:** AC-003, AC-004.

---

## US-03 — Node config

As an operator, I want to define nodes (host, SSH user, authentication) in configuration and have the system validate at startup, so that configuration errors are caught before serving.  
**Requirements:** REQ-003, REQ-004. **AC:** AC-005, AC-006.

---

## US-04 — Per-node allowlist

As an operator, I want a documented security model that defines, per node, which commands or tools are allowed, so that only permitted actions run on each node.  
**Requirements:** REQ-005. **AC:** AC-007, AC-008.

---

## US-05 — Dedicated PA user per node

As an operator, I want to configure one dedicated user account per node for PersonalAssistant SSH access, so that all actions are attributed to that identity.  
**Requirements:** REQ-013. **AC:** AC-009, AC-010.

---

## US-06 — Memory store

As the assistant (system), I want to store long-term memory in markdown files in a defined directory structure, so that data is human-readable and easy to back up.  
**Requirements:** REQ-006, REQ-018, REQ-019, REQ-020. **AC:** AC-011, AC-012.

---

## US-07 — Vector search

As the assistant (system), I want to index long-term memory in a vector store and run semantic search, so that relevant context can be retrieved for replies.  
**Requirements:** REQ-007. **AC:** AC-013, AC-014.

---

## US-08 — Pluggable LLM provider

As an operator, I want to choose and configure the LLM provider via configuration without code changes, so that I can avoid vendor lock-in.  
**Requirements:** REQ-008. **AC:** AC-015, AC-016.

---

## US-09 — LLM logging

As an operator or developer, I want a logging subsystem that records each LLM request and response, so that I can analyse usage and perform audits.  
**Requirements:** REQ-014. **AC:** AC-017.

---

## US-10 — Log destination and format

As an operator, I want to configure where LLM logs are written and in what parseable format, so that I can control retention and analysis.  
**Requirements:** REQ-015. **AC:** AC-018, AC-019.

---

## US-11 — Scheduled tasks

As an operator, I want to define scheduled tasks (time or interval) in configuration, so that the assistant can run periodic actions within the security model.  
**Requirements:** REQ-009. **AC:** AC-020, AC-021.

---

## US-12 — Extensible tools

As a developer, I want to add new tools via a single contract without changing core orchestration code, so that capabilities can be extended in a modular way.  
**Requirements:** REQ-010. **AC:** AC-022, AC-023.

---

## US-13 — Add nodes/tools without rebuild

As an operator, I want to add new nodes and register new tools through configuration, so that I can scale without rebuilding the core image.  
**Requirements:** REQ-011. **AC:** AC-024.

---

## US-14 — Architecture boundaries

As an architect or developer, I want the system to clearly separate adapters, core, memory, vector, LLM, scheduler, and tools, so that we can evolve or replace each part.  
**Requirements:** REQ-012. **AC:** AC-025.

---

## US-15 — Version control (git)

As an operator, I want the assistant to use a git repository to track configuration, memory, and designated data, so that I can review history and roll back if needed.  
**Requirements:** REQ-016. **AC:** AC-026, AC-027.

---

## US-16 — Secret leakage protection

As an operator or security-conscious user, I want the assistant to never expose secret values in LLM context, user-facing responses, or logs, so that credentials cannot be extracted via crafted prompts.  
**Requirements:** REQ-017. **AC:** AC-028, AC-029, AC-030.
