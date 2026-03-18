# Epic scope — EP-004 Structured tools and Tool-calling API

## Introduction

This document is the epic scope for EP-004 (Structured tools and Tool-calling API). It builds on EP-001 (PersonalAssistant MVP) and extends the tool contract so that the LLM invokes tools via the provider's Tool-calling API, with a single source of truth for tool definitions (id, short description, node, schema), compact context per request, validation and template-based execution (e.g. run_on_node). It is aligned with project [scope.md](../../scope.md) and [strategy.md](../../strategy.md). Requirements and acceptance criteria are produced in later pipeline stages.

## Epic ID, title, short description

| Field | Content |
|-------|---------|
| **ID** | EP-004 |
| **Status** | NEW |
| **Title** | Structured tools and Tool-calling API |
| **Description** | Introduce a single source of truth for tools (id, short_description, node_id, schema/expected_json), a vector-based tool index and pre-selection so each LLM request receives a bounded subset of tools, Tool-calling API integration, validation and execution via run_on_node (template substitution and allowlist), and errors to the user in chat. Includes support for Sonos control (e.g. volume, play) as invocable tools on a configured node. |

## Glossary

Terms from the project [scope.md](../../scope.md) glossary apply. Epic-specific terms:

| Term | Definition |
|------|------------|
| **Tool (invocable)** | A single invokable capability presented to the LLM: identified by a stable id, has a short_description for model choice, is bound to a node (or global), and has a schema (expected_json) for arguments. The LLM returns a tool call with id and arguments; PA validates and executes. |
| **Single source of truth (tools)** | One configuration (e.g. YAML) per node or shared base_commands plus per-node overrides, parsed at startup. Defines for each tool: id, template (command string with placeholders), node_id, and argument rules (allowed_values, pattern, type, min/max). Used both to build the payload sent to the LLM and to validate and run the command. |
| **Tool-calling API** | The provider-native mechanism (OpenAI, Anthropic, Ollama-compatible) to pass a list of tools in the request and receive structured tool_calls in the response (name/id and arguments as JSON). PA uses this so the model does not embed JSON in free text. |
| **short_description** | One or two sentences per tool included in every LLM request so the model can choose which tool to call without receiving full argument enumerations. |
| **expected_json** | The shape of arguments for a tool (e.g. object with tool_id, args or command_id, args). Sent to the LLM as a schema or example; full validation (allowed_values, patterns) is done in PA from the source of truth. |
| **Template (command)** | A string with placeholders (e.g. `systemctl status {{service}}`) from the source of truth. After validating arguments, PA substitutes placeholders and runs the resulting command via run_on_node. |
| **Sonos (node / tool)** | A node or device type that exposes Sonos control (e.g. volume, play, track) via allowlisted commands (e.g. sonos-cli). Tools for Sonos are defined in the same source of truth as other node tools (id, template, args) and are invoked through the Tool-calling API. |
| **Tool index (vector)** | A searchable index built at startup from the tool catalog: each tool is represented by text (e.g. id, short_description, optional triggers) and stored with its embedding. The index lives in the **same vector database** as memory, in a **dedicated table** (e.g. vec_tools), so one DB file can hold multiple logical stores (memory, tools, and future ones such as skills). Used to select a subset of tools per user request. |
| **Tool pre-selection** | The process of selecting which tools to send to the LLM for a given request. The core embeds the user message (and optionally conversation context), searches the tool index, and obtains a bounded subset (e.g. top-k by similarity); only that subset is included in the Tool-calling API request. |
| **Tool call (text-based)** | When the provider does not support the Tool-calling API, tool invocation by instructing the model (via prompt) to output tool calls in a defined text format (e.g. `<tool_call>` tags with JSON). The system parses the assistant message, validates and executes as for native tool_calls. |

## Scope (features/capabilities)

- **Single source of truth for tools:** One place (e.g. YAML files: base_commands and per-node allowlists) defines all invocable tools. Each tool has: id, template (explicit, no defaults), node_id (or derived from id), and argument rules. Optionally each tool has triggers (example phrases) for better pre-selection. Parsed at service startup; used to build the tool index, the tool list for the LLM, and to validate and execute.
- **Tool index at startup:** At startup, the system builds a tool index from the catalog (tool id, short_description, optional triggers), computes embeddings, and stores them in the **same vector database as memory, in a dedicated table** (multiple tables in one DB). This index is the source for tool pre-selection on each request. The design targets **up to 1000 tools**. **Tool index load (embedding + write) must complete within 20 seconds** at startup; this is achieved by batching requests to the embedding API and/or building or updating the index in the background (e.g. service accepts traffic while the tool index is still populating, with a defined fallback until ready).
- **Tool pre-selection and tool list per request:** For each completion request that can trigger tools, the core selects a subset of tools using the tool index (e.g. embed the user message, search the index, take top-k by similarity). Only this subset is included in the LLM request in the provider's Tool-calling format (name/id, description, parameters schema or example). A defined fallback applies when pre-selection returns no or too few tools.
- **Tool-calling API integration:** The core passes tools to the provider's completion API and handles tool_calls in the response. When the model returns tool_calls, PA parses arguments, validates against the source-of-truth schema, substitutes into the tool's template, and executes via run_on_node (or equivalent). Results and errors are returned as tool results and, when appropriate, surfaced to the user in chat.
- **Validation and execution:** Arguments from the LLM are validated (types, allowed_values, pattern, min/max). Invalid or unknown tool id or arguments produce a clear error; no command is executed. Valid arguments are substituted into the template; the resulting string is executed on the node via the existing run_on_node path (allowlist and SSH model unchanged for the executed command).
- **Errors to chat:** If validation fails, execution fails, or the node returns an error, the user sees the error in chat (e.g. as the assistant's reply or as a tool result conveyed to the user).
- **Provider interface extension:** The LLM provider interface is extended to accept an optional tools payload and to return tool_calls and related metadata so the core can drive the request–response–tool-result loop without parsing JSON from assistant text.
- **Sonos support:** Tools for controlling Sonos (e.g. volume, play/pause, track selection) are definable in the tool source of truth. They target a configured node that runs the Sonos control interface (e.g. sonos-cli). Argument rules (e.g. volume 0–100, service name) follow the same schema (allowed_values, type integer with min/max, or pattern). No separate Sonos-specific API in the core; Sonos is one of the node tool sets described in the catalog.
- **Tool invocation without Tool-calling API (optional):** Where the LLM provider does not support the Tool-calling API (e.g. returns 400 "does not support tools"), the system MAY support tool invocation by describing tools in the prompt and parsing the assistant's text output for tool calls in a defined format (e.g. Hermes-style `<tool_call>` with JSON). Parsed calls use the same validation and execution path as native tool_calls. Configuration SHALL allow enabling or disabling this behaviour per provider or globally.

## Scale and performance

- **Target scale:** The tool catalog and tool index are designed for **up to 1000 tools**.
- **Vector storage:** Tool index uses the **same vector database file** as memory (conversation/summaries), in a **dedicated table** (e.g. vec_tools vs vec_items for memory). One DB, multiple tables — no separate vector store instance for tools.
- **Tool index load at startup:** Building the tool index (embedding all tools and writing to the vector store) **MUST complete within 20 seconds** from service start. To meet this, the implementation SHALL use one or both of: **batched requests to the embedding API** (e.g. embed many tools per API call where supported), and/or **background index build or update** (service may accept traffic while the tool index is still populating, with a defined fallback — e.g. use a default subset of tools or retry pre-selection later — until the index is ready).

## Success criteria

- **Single source of truth:** All invocable tools are defined in the configured YAML (or equivalent); no duplicate or ad-hoc tool definitions.
- **Tool-calling in requests:** Each completion request that can trigger tools includes a pre-selected subset of tools (from the tool index) in the provider's expected format; at least one provider (e.g. OpenAI-compatible or Ollama) is supported.
- **Validation before execution:** No command is run on a node unless the tool call matches a known tool and arguments pass the schema; invalid calls produce a deterministic error response.
- **Execution path:** Valid tool calls result in template substitution and execution via run_on_node; behaviour is consistent with the existing allowlist and SSH model.
- **Errors visible:** Validation and execution errors are returned to the user in chat.
- **Sonos tools:** At least one Sonos-related tool (e.g. volume or play) is definable in the source of truth, exposed to the LLM, and executable via run_on_node on a configured node; validation and errors behave like other tools.
- **Tests:** New or changed behaviour is covered by unit and/or integration tests; existing tests continue to pass.
- **Tool index and pre-selection:** Tool index is built at startup (same vector DB, dedicated table); scale target 1000 tools; load within 20 s (batching or background). Each LLM request receives only a pre-selected subset of tools (with fallback when selection is empty or too small).
- **Text-based tool invocation (optional):** When the provider does not support the Tool-calling API, the system MAY describe tools in the prompt and parse the assistant text for tool calls in a defined format; parsed calls undergo the same validation and execution. Parse failure or invalid format does not trigger execution. Configuration allows enabling or disabling this feature per provider or globally.

## Out of scope / deferred

- **Full tool list in every request:** Sending the entire catalog to the LLM in every request is out of scope; this epic uses semantic tool pre-selection (vector-based) to send a bounded subset per request.
- **Dynamic tool loading:** Tools are fixed at startup from config; no runtime discovery or loading of new tools within a session.
- **MCP or third-party tool servers:** Not in scope; only PA-defined tools from the source of truth.
- **Changing the scheduler contract:** Scheduler continues to use the existing tool registry and params; this epic focuses on chat-driven tool invocation via Tool-calling API.
- **Native Sonos API in core:** The core does not implement a dedicated Sonos HTTP/API client; Sonos is controlled only via node commands (e.g. sonos-cli) defined as tools in the catalog.

## Traceability

- **Scope:** This epic extends the **Tool** and **Security model** from [scope.md](../../scope.md): tools remain explicit and validated; execution is still via defined tools and allowlist, with a clearer contract (Tool-calling API) and a single source of truth for tool definitions.
- **Strategy:** Aligns with [strategy.md](../../strategy.md): testability (unit/integration for validation and execution), security (allowlist, validation at tool level), and evolution without breaking the core (extended provider interface and tool catalog).
- **Dependency:** Builds on EP-001 (PersonalAssistant MVP). Can follow or overlap with EP-002 and EP-003; no hard dependency on their completion.
