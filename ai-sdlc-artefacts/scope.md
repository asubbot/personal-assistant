# Scope — PersonalAssistant MVP

## Introduction

PersonalAssistant is a minimal MVP of a personal assistant, inspired by systems like OpenClaw, with a focus on **reliability and security**. The target platform is Synology DS220+ (Docker); the user interacts via a Telegram bot. A Go core in a container manages nodes over SSH under a validated security model, stores long-term memory as markdown files, indexes it for vector search, supports swappable LLM backends (including self-hosted), and provides a task scheduler and extensible tools. Deployment and adding nodes or tools are kept simple; the architecture is designed to evolve without radical redesign.

---

## Glossary

| Term | Definition |
|------|------------|
| **PersonalAssistant (System)** | The set of components: core (Go), Telegram adapter, memory store, vector index, scheduler, LLM providers, and tools. Deployed in Docker, target platform Synology DS220+. |
| **Core** | The main Go service: orchestration of conversations, LLM calls, tool execution, access to memory and scheduler, and SSH-based node management. |
| **Node** | A remote host (e.g. NAS, server) that the core connects to over SSH to run actions; has a defined capability set and credentials in configuration. |
| **Security model** | Explicit definition of which nodes are allowed, which commands/tools are permitted on each node, and how inputs and outputs are validated; validated at load and on configuration change. |
| **Long-term memory** | The assistant's memory: a single store of facts and context held as markdown files in a defined directory structure; read/write by the core and indexer. Structure is calendar-based (year/month/day) with hierarchical summarization (day → month → year). |
| **Vector store** | Index of embeddings from long-term memory for semantic search; provider is pluggable. |
| **LLM provider** | Abstraction for calling a language model (OpenAI-compatible API, Ollama, self-hosted); configuration specifies endpoint and parameters. |
| **Tool** | Extensible module: name, description, validated input schema, and implementation; registered with the core and invoked via a single contract. |
| **Scheduler** | Component that runs tasks on a schedule (cron-like or intervals); tasks are defined in configuration or API. |
| **User** | A person interacting with PersonalAssistant via the Telegram bot; identified by Telegram user ID. |
| **Dedicated PA user** | A dedicated user account on each node used only for PersonalAssistant access. |
| **Logging subsystem** | Component that records LLM interaction events for analysis, debugging, and audit. |

---

## In scope

- Telegram bot for conversation.
- Go core in Docker (target hardware: Synology DS220+).
- SSH interaction with nodes under a clear, validated security model.
- Long-term memory in markdown files.
- Vector indexing for semantic retrieval.
- No vendor lock-in: support for multiple LLMs, including self-hosted.
- Scheduler for time- or interval-based tasks.
- Extensible tools with a single contract.
- Simple deployment and addition of nodes and tools.
- Architecture that allows future evolution without fundamental change.
- Dedicated user account on each node for PersonalAssistant only.
- Logging subsystem for LLM requests and responses (for analysis and audit).
- Version control via an internal git repository for configuration, memory files, and other designated artifacts.

---

## Out of scope / deferred

- Performance, load, or stress testing (covered in test strategy as out of scope for this epic).
- Cross-platform test matrix; E2E targets x86_64 / DS220+.

---

