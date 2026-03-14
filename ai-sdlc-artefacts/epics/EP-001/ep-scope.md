# Epic scope — EP-001 PersonalAssistant MVP

## Introduction

This document is the epic scope for EP-001 (PersonalAssistant MVP). It is aligned with project [scope.md](../../scope.md) and [strategy.md](../../strategy.md). Details for requirements, system design, and user stories are produced in later pipeline stages.

## Epic ID, title, short description

| Field | Content |
|-------|---------|
| **ID** | EP-001 |
| **Title** | PersonalAssistant MVP |
| **Description** | A working personal assistant the user can talk to via Telegram. It runs in Docker on Synology DS220+, uses long-term memory and optional remote nodes, supports multiple LLM backends, and is built so the architecture can evolve without breaking the core. |

## Glossary

Terms are defined in the project [scope.md](../../scope.md) glossary (Core, Node, Security model, Long-term memory, Vector store, LLM provider, Tool, Scheduler, User, Dedicated PA user, Logging subsystem). No epic-specific terms added here.

## Scope (features/capabilities)

- Telegram bot for user conversation.
- Go core in Docker; target hardware Synology DS220+.
- SSH interaction with nodes under a validated security model (allowlists, dedicated PA user per node).
- Long-term memory stored as markdown files in a defined structure.
- Vector indexing for semantic retrieval over memory.
- Multiple LLM backends (no vendor lock-in), including self-hosted.
- Scheduler for time- or interval-based tasks.
- Extensible tools with a single contract (name, description, params, run).
- Simple deployment and simple addition of nodes and tools.
- Logging subsystem for LLM requests and responses (analysis and audit).
- Version control (internal git) for configuration, memory files, and other designated artifacts.

## Success criteria

- **Build:** `go build ./...` succeeds; artifact is a single binary.
- **Tests:** All unit and integration tests pass (e.g. `make check` with appropriate build tag for integration).
- **E2E:** At least one scenario passes: start core (locally or in Docker) with minimal config → send one message to Telegram (or via mock) → receive one reply from the assistant.
- **Platform:** Binary or Docker image runs on target platform (DS220+ or x86_64 equivalent).

## Traceability

- **Scope:** This epic covers the full "In scope" set from scope.md: Telegram bot, Go core in Docker, SSH nodes and security model, long-term memory, vector index, multiple LLMs, scheduler, tools, deployment simplicity, evolution-friendly architecture, dedicated PA user per node, logging subsystem, internal git for config and memory.
- **Strategy:** This epic maps to the MVP (0.01) increment in strategy.md: delivery of the working assistant with the capabilities listed above and test strategy as defined in strategy §2.
