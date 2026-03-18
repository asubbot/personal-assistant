# EP-004 Structured tools and Tool-calling API — Acceptance criteria

**Pipeline:** Stage 5 (Acceptance criteria).

**Contents**

- [Introduction](#introduction)
- [Acceptance criteria index](#acceptance-criteria-index)
- [Acceptance criteria](#acceptance-criteria)

---

## Introduction

This document defines epic-level acceptance criteria for **EP-004 Structured tools and Tool-calling API**. It contains testable conditions (Gherkin-style) that apply to the epic as a whole. Traceability to [ep-requirements.md](ep-requirements.md) is given per AC below.

**Scope:** Single source of truth for tools (catalog), tool index at startup (same vector DB as memory, dedicated table; scale up to 1000 tools; load within 20 s via batching or background with fallback) and tool pre-selection per request, tool list (pre-selected subset) in every LLM request, Tool-calling API integration, validation and execution via run_on_node, errors surfaced in chat, provider interface extension, and Sonos tools in the same catalog. Optional: tool invocation for providers that do not support the Tool-calling API (text-based: tools in prompt, parse assistant output in a defined format, same validation and execution path; configurable enable/disable). Scheduler contract unchanged; no MCP or dynamic tool loading.

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-04.001](#ac-04-001) | [REQ-04.001](ep-requirements.md#tool-catalog-and-source-of-truth), [REQ-04.002](ep-requirements.md#tool-catalog-and-source-of-truth) | Tool catalog defines all invocable tools with id, template, node_id, argument rules |
| [AC-04.002](#ac-04-002) | [REQ-04.003](ep-requirements.md#tool-catalog-and-source-of-truth) | Catalog parsed at startup; used for LLM payload and for validation/execution |
| [AC-04.003](#ac-04-003) | [REQ-04.004](ep-requirements.md#tool-calling-api), [REQ-04.005](ep-requirements.md#tool-calling-api), [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection) | Every completion request includes pre-selected tool list in provider format; at least one provider supported |
| [AC-04.004](#ac-04-004) | [REQ-04.006](ep-requirements.md#tool-calling-api) | LLM returns tool_calls → core validates, substitutes, executes via run_on_node, returns tool results in loop |
| [AC-04.005](#ac-04-005) | [REQ-04.007](ep-requirements.md#validation-and-execution) | Tool-call arguments validated (types, allowed_values, pattern, min/max) before any execution |
| [AC-04.006](#ac-04-006) | [REQ-04.008](ep-requirements.md#validation-and-execution) | Unknown tool id or invalid arguments → no command executed; deterministic error returned |
| [AC-04.007](#ac-04-007) | [REQ-04.009](ep-requirements.md#validation-and-execution), [REQ-04.010](ep-requirements.md#validation-and-execution) | Valid tool call → arguments substituted into template; command executed via run_on_node under allowlist and SSH |
| [AC-04.008](#ac-04-008) | [REQ-04.011](ep-requirements.md#errors-to-chat) | Validation or execution failure surfaced to user in chat |
| [AC-04.009](#ac-04-009) | [REQ-04.012](ep-requirements.md#provider-interface) | Provider interface accepts optional tools payload and returns tool_calls so core can drive tool-result loop |
| [AC-04.010](#ac-04-010) | [REQ-04.013](ep-requirements.md#sonos-support) | At least one Sonos tool definable in catalog, exposed to LLM, executable on configured node with same validation/errors as other tools |
| [AC-04.011](#ac-04-011) | [REQ-04.014](ep-requirements.md#nfr--security-testability-observability-consistency), [REQ-04.017](ep-requirements.md#nfr--security-testability-observability-consistency) | No command executed without known tool and schema-passing arguments; Sonos uses same catalog and execution path |
| [AC-04.012](#ac-04-012) | [REQ-04.015](ep-requirements.md#nfr--security-testability-observability-consistency) | New or changed behaviour covered by unit/integration tests; existing tests pass |
| [AC-04.013](#ac-04-013) | [REQ-04.016](ep-requirements.md#nfr--security-testability-observability-consistency) | Tool invocations (id, arguments, result or error) traceable in logs where supported |
| [AC-04.014](#ac-04-014) | [REQ-04.018](ep-requirements.md#tool-index-and-pre-selection) | Tool index built at startup from catalog and stored in vector store |
| [AC-04.015](#ac-04-015) | [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection) | Tool list for each request built via pre-selection (embed message, search index, bounded subset) |
| [AC-04.016](#ac-04-016) | [REQ-04.020](ep-requirements.md#tool-index-and-pre-selection) | Fallback when pre-selection returns no or too few tools yields non-empty bounded tool list |
| [AC-04.017](#ac-04-017) | [REQ-04.021](ep-requirements.md#tool-index-and-pre-selection) | Tool index load completes within 20 s or runs in background with fallback; batching used where applicable |
| [AC-04.018](#ac-04-018) | [REQ-04.022](ep-requirements.md#tool-index-and-pre-selection) | Startup fails if tool catalog missing or tool index store cannot be created |
| [AC-04.019](#ac-04-019) | [REQ-04.023](ep-requirements.md#tool-index-and-pre-selection) | When index not ready, fallback yields non-empty bounded tool list (same as empty pre-selection) |
| [AC-04.020](#ac-04-020) | [REQ-04.024](ep-requirements.md#tool-index-and-pre-selection) | Embedding batch_size configurable (e.g. 1–1000); tool index build uses it for chunking |
| [AC-04.021](#ac-04-021) | [REQ-04.025](ep-requirements.md#tool-index-and-pre-selection) | Tool index build success logged (INFO); build failure logged (ERROR with reason) |
| [AC-04.022](#ac-04-022) | [REQ-04.026](ep-requirements.md#tool-invocation-without-tool-calling-api) | When provider lacks Tool-calling API, system MAY support text-based tool invocation (tools in prompt, defined format) |
| [AC-04.023](#ac-04-023) | [REQ-04.027](ep-requirements.md#tool-invocation-without-tool-calling-api), [REQ-04.028](ep-requirements.md#tool-invocation-without-tool-calling-api) | Text-based: prompt describes tools and format; parsed tool calls use same validation and execution path |
| [AC-04.024](#ac-04-024) | [REQ-04.029](ep-requirements.md#tool-invocation-without-tool-calling-api) | Parse failure or invalid format → no execution; plain text or deterministic error to user |
| [AC-04.025](#ac-04-025) | [REQ-04.030](ep-requirements.md#tool-invocation-without-tool-calling-api) | Configurable enable/disable of text-based tool invocation per provider or globally |

---

## Acceptance criteria

<a id="ac-04-001"></a>**AC-04.001** (Trace: [REQ-04.001](ep-requirements.md#tool-catalog-and-source-of-truth), [REQ-04.002](ep-requirements.md#tool-catalog-and-source-of-truth))

Given the operator has configured a tool catalog (e.g. YAML) as the single source of truth,  
When the catalog is valid,  
Then every invocable tool is defined there with: a stable id, a template (command string with placeholders), node_id (or binding to a node), and argument rules (e.g. allowed_values, pattern, type, min/max).  
And there are no duplicate or ad-hoc tool definitions elsewhere.

---

<a id="ac-04-002"></a>**AC-04.002** (Trace: [REQ-04.003](ep-requirements.md#tool-catalog-and-source-of-truth))

Given the service is started with a valid tool catalog path,  
When the service starts,  
Then the catalog is parsed at startup.  
And when the core builds the tool list for an LLM completion request, it uses the parsed catalog.  
And when the core validates and executes a tool call, it uses the same parsed catalog.

---

<a id="ac-04-003"></a>**AC-04.003** (Trace: [REQ-04.004](ep-requirements.md#tool-calling-api), [REQ-04.005](ep-requirements.md#tool-calling-api))

Given the user sends a message that can trigger tools,  
When the core calls the LLM provider to get a completion,  
Then the request includes the pre-selected list of tools (from tool pre-selection) in the format required by the provider's Tool-calling API (name/id, short description, parameters schema or example).  
And at least one supported provider (e.g. OpenAI-compatible or Ollama) accepts this format and can return tool_calls in the response.

---

<a id="ac-04-004"></a>**AC-04.004** (Trace: [REQ-04.006](ep-requirements.md#tool-calling-api))

Given the LLM response contains one or more tool_calls (tool id and arguments),  
When the core processes the response,  
Then the core parses the arguments, validates them against the tool's schema from the source of truth, substitutes validated arguments into the tool's template, and executes the resulting command via the existing run_on_node path.  
And the core returns tool results (or errors) so that the request–response–tool-result loop continues without parsing JSON from assistant text.

---

<a id="ac-04-005"></a>**AC-04.005** (Trace: [REQ-04.007](ep-requirements.md#validation-and-execution))

Given a tool call with arguments from the LLM,  
When the core validates the arguments before executing any command,  
Then the core checks types, allowed_values, pattern, and min/max (where defined in the tool's schema).  
And no command is executed until validation passes.

---

<a id="ac-04-006"></a>**AC-04.006** (Trace: [REQ-04.008](ep-requirements.md#validation-and-execution))

Given a tool call with an unknown tool id or with arguments that fail validation (wrong type, value not in allowed_values, pattern mismatch, or out of min/max range),  
When the core processes the tool call,  
Then the system SHALL NOT execute any command on any node.  
And the system SHALL produce a deterministic error response (e.g. tool result or message to the user).

---

<a id="ac-04-007"></a>**AC-04.007** (Trace: [REQ-04.009](ep-requirements.md#validation-and-execution), [REQ-04.010](ep-requirements.md#validation-and-execution))

Given a tool call with a known tool id and arguments that pass validation,  
When the core executes the tool,  
Then the core substitutes the validated arguments into the tool's template and obtains a single command string.  
And the core executes that command on the node via the existing run_on_node path.  
And the executed command remains subject to the node's allowlist and SSH security model (no execution if the substituted command is not allowlisted).

---

<a id="ac-04-008"></a>**AC-04.008** (Trace: [REQ-04.011](ep-requirements.md#errors-to-chat))

Given a validation failure (unknown tool, invalid arguments) or an execution failure (run_on_node error, node-returned error),  
When the user is in a chat session and the failure occurs during processing of their message,  
Then the system SHALL surface the error to the user in chat (e.g. as the assistant's reply or as a tool result conveyed to the user).

---

<a id="ac-04-009"></a>**AC-04.009** (Trace: [REQ-04.012](ep-requirements.md#provider-interface))

Given an LLM provider implementation that supports the Tool-calling API,  
When the core passes an optional tools payload in the completion request and receives a response,  
Then the provider interface accepts the optional tools payload.  
And the provider returns tool_calls and related metadata in the response so the core can drive the request–response–tool-result loop without parsing JSON from assistant text.

---

<a id="ac-04-010"></a>**AC-04.010** (Trace: [REQ-04.013](ep-requirements.md#sonos-support))

Given the operator has defined at least one Sonos-related tool (e.g. volume or play) in the tool catalog, bound to a configured node that runs the Sonos control interface (e.g. sonos-cli),  
When the catalog is loaded and the user sends a message that triggers that tool (e.g. "set kitchen volume to 30"),  
Then the tool is included in the tool list sent to the LLM.  
And when the LLM returns a tool call for that tool with valid arguments, the system executes it via run_on_node on the configured node.  
And validation and error behaviour for the Sonos tool are the same as for other node tools (no separate Sonos API in the core).

---

<a id="ac-04-011"></a>**AC-04.011** (Trace: [REQ-04.014](ep-requirements.md#nfr--security-testability-observability-consistency), [REQ-04.017](ep-requirements.md#nfr--security-testability-observability-consistency))

Given any tool call,  
When the core decides whether to execute a command on a node,  
Then the system SHALL NOT execute any command unless the tool id is known and the arguments pass the schema defined in the source of truth.  
And Sonos tools SHALL use the same catalog format, validation, and execution path as other node tools.

---

<a id="ac-04-012"></a>**AC-04.012** (Trace: [REQ-04.015](ep-requirements.md#nfr--security-testability-observability-consistency))

Given the changes introduced in this epic,  
When the test suite is run,  
Then new or changed behaviour is covered by unit and/or integration tests.  
And all existing tests continue to pass.

---

<a id="ac-04-013"></a>**AC-04.013** (Trace: [REQ-04.016](ep-requirements.md#nfr--security-testability-observability-consistency))

Given a tool invocation (tool id, arguments, and result or error),  
When the existing logging subsystem is configured and supports it,  
Then the tool id, arguments, and result or error are traceable in logs (e.g. for debugging and audit).

---

<a id="ac-04-014"></a>**AC-04.014** (Trace: [REQ-04.018](ep-requirements.md#tool-index-and-pre-selection))

Given the service is started with a valid tool catalog and vector store (and embedding provider) configured,  
When the service starts,  
Then the system builds a tool index from the catalog (tool id, short_description, optional triggers).  
And each index entry is embedded and stored in the **same vector database as memory, in a dedicated table** (e.g. vec_tools).  
And the design supports catalogs of up to 1000 tools.  
And the index is available for tool pre-selection on subsequent completion requests (or a defined fallback applies until the index is ready when built in background).

---

<a id="ac-04-015"></a>**AC-04.015** (Trace: [REQ-04.019](ep-requirements.md#tool-index-and-pre-selection))

Given the tool index is built and the user sends a message that can trigger tools,  
When the core builds the tool list for the LLM completion request,  
Then the core uses the user message (and optionally conversation context) to query the tool index (e.g. embed message, search by similarity, take top-k).  
And only the tools in the selected subset are included in the request to the provider.  
And the number of tools sent is bounded (e.g. by a configured top-k or cap).

---

<a id="ac-04-016"></a>**AC-04.016** (Trace: [REQ-04.020](ep-requirements.md#tool-index-and-pre-selection))

Given tool pre-selection is performed for a completion request,  
When the pre-selection returns no tools or fewer than the configured minimum,  
Then the system SHALL apply a defined fallback (e.g. include a default subset of tools or all tools up to a cap).  
And the request to the provider SHALL include a non-empty, bounded tool list so the LLM can still choose to use tools.

---

<a id="ac-04-017"></a>**AC-04.017** (Trace: [REQ-04.021](ep-requirements.md#tool-index-and-pre-selection))

Given the service is started with a tool catalog that may contain up to 1000 tools and a configured embedding provider,  
When the tool index is built at startup,  
Then the index load (embedding all tools and writing to the vector store) SHALL complete **within 20 seconds**, or the build SHALL run in the background with a defined fallback (e.g. default tool subset or "index not ready" handling) until the index is ready.  
And the implementation SHALL use batched requests to the embedding API where supported and/or background index build to meet the 20-second constraint.

---

<a id="ac-04-018"></a>**AC-04.018** (Trace: [REQ-04.022](ep-requirements.md#tool-index-and-pre-selection))

Given the service is configured with a tool catalog path and an embedding provider,  
When the tool catalog fails to load (e.g. missing or invalid) or the tool index vector store cannot be created (e.g. path invalid),  
Then startup SHALL fail with a clear error and the service SHALL NOT start.

---

<a id="ac-04-019"></a>**AC-04.019** (Trace: [REQ-04.023](ep-requirements.md#tool-index-and-pre-selection))

Given the tool index is not ready (e.g. background build in progress or build has failed),  
When the core builds the tool list for a completion request that can trigger tools,  
Then the system SHALL apply the defined fallback (e.g. all tools up to a cap or a default subset) so that the request receives a non-empty, bounded tool list.  
And the behaviour SHALL be the same as when pre-selection returns no or too few tools.

---

<a id="ac-04-020"></a>**AC-04.020** (Trace: [REQ-04.024](ep-requirements.md#tool-index-and-pre-selection))

Given the embedding provider is configured with a batch_size in the allowed range (e.g. 1–1000),  
When the tool index is built at startup,  
Then embedding API calls for the index SHALL be chunked so that no single request exceeds batch_size texts.  
And the batch_size SHALL be configurable and validated at config load (e.g. required and within bounds).

---

<a id="ac-04-021"></a>**AC-04.021** (Trace: [REQ-04.025](ep-requirements.md#tool-index-and-pre-selection))

Given the tool index is built at startup (in background or synchronously),  
When the build completes successfully,  
Then the system SHALL log an informational message (e.g. "tool index built" with the number of tools indexed).  
When the build fails,  
Then the system SHALL log an error message that includes the failure reason (e.g. embedding error, store error).

---

<a id="ac-04-022"></a>**AC-04.022** (Trace: [REQ-04.026](ep-requirements.md#tool-invocation-without-tool-calling-api))

Given the configured LLM provider does not support the Tool-calling API (e.g. returns an error indicating tools are not supported),  
When the feature is enabled (per provider or globally),  
Then the system MAY support tool invocation by describing the pre-selected tools in the prompt and parsing the assistant's text response for tool calls in a defined, documented format (e.g. Hermes-style `<tool_call>` with JSON).  
And the format SHALL be documented so operators and implementers know what the model is expected to output.

---

<a id="ac-04-023"></a>**AC-04.023** (Trace: [REQ-04.027](ep-requirements.md#tool-invocation-without-tool-calling-api), [REQ-04.028](ep-requirements.md#tool-invocation-without-tool-calling-api))

Given text-based tool invocation is enabled and the provider does not support the Tool-calling API,  
When the core sends a completion request that can trigger tools,  
Then the prompt SHALL include a description of the available tools and instructions for the model to output tool calls in the defined format.  
And when the assistant message is received, the core SHALL parse it to extract tool id (or name) and arguments.  
And parsed tool calls SHALL undergo the same validation (tool id in catalog, arguments conforming to source-of-truth schema) and execution path (template substitution, run_on_node) as tool_calls received via the Tool-calling API.  
And the system SHALL NOT execute any command for an unknown tool id or for arguments that fail validation.

---

<a id="ac-04-024"></a>**AC-04.024** (Trace: [REQ-04.029](ep-requirements.md#tool-invocation-without-tool-calling-api))

Given the assistant text is received and text-based tool invocation is enabled,  
When the text cannot be parsed to obtain a valid tool call (malformed format, missing required fields, or unrecoverable parse error),  
Then the system SHALL NOT execute any command based on that output.  
And the system SHALL either treat the response as plain assistant text or surface a deterministic error to the user in chat.

---

<a id="ac-04-025"></a>**AC-04.025** (Trace: [REQ-04.030](ep-requirements.md#tool-invocation-without-tool-calling-api))

Given the system supports text-based tool invocation for providers that do not support the Tool-calling API,  
When the operator configures the system,  
Then configuration SHALL allow enabling or disabling text-based tool invocation (e.g. per provider or globally).  
And changing this setting SHALL NOT require code changes (config or env only).
