# EP-004 Manual test scenarios

**Purpose:** Document manual verification steps for acceptance criteria that are weakly covered by automated tests or need a real environment (node, Sonos CLI, Hermes provider). See [ep-audit-report.md](ep-audit-report.md) for the REQ/AC test coverage matrix.

**AC-04.010 (Sonos):** The scenario [Sonos tool end-to-end](ep-manual-tests.md#sonos-tool-end-to-end) is the **documented manual test** for AC-04.010. Complete it on a real deployment and record pass/fail (e.g. in test log or release checklist); no automated test uses tool id `sonos`.

**Reference:** [strategy.md](../../strategy.md) §2.3 (Manual testing), [ep-acceptance-criteria.md](ep-acceptance-criteria.md), [ep-scope.md](ep-scope.md).

*Headings are short titles. Traceability is on the **Trace** line below each scenario (clickable links to AC and REQ).*

---

## Sonos tool end-to-end

**Trace:** [AC-04.010](ep-acceptance-criteria.md#ac-04-010) · [REQ-04.013](ep-requirements.md#sonos-support)

**Criterion:** At least one Sonos-related tool is definable in the catalog, included in the tool list when the user message matches, executed via **run_on_node** on the configured node with valid arguments; validation/errors same as other tools.

**Prerequisites:**

- Tool catalog includes a tool with id **`sonos`** (or equivalent) bound to a node where the Sonos control command exists and is allowlisted (see operator [config.examples/tools.yaml](../../../config.examples/tools.yaml) as an example).
- Chat uses a provider with **supports_tools** true (native tool-calling) unless you run the Hermes variant under [AC-04.022](ep-acceptance-criteria.md#ac-04-022).

**Steps:**

1. Start the service with valid config and catalog.
2. Send a user message that should pre-select the Sonos tool (e.g. *"set "Living Room" volume to 25"* or *"stop Sonos in Bedroom"* — use real room names from your Sonos setup).
3. Confirm the assistant either asks for a missing room name or returns a tool call for **`sonos`** with **`speaker`** and **`tail`** arguments.
4. After execution, confirm the physical Sonos behaviour matches **`tail`** (e.g. volume change, pause).
5. Send a message that should trigger **invalid** arguments (e.g. empty room if required) and confirm the user sees a validation error in chat and **no** shell command runs with unsafe args.

**Pass:** Tool id **`sonos`** appears in the LLM tool list when relevant; valid calls execute on the node; invalid calls do not execute arbitrary commands; errors surface in chat like other node tools.

**Note:** Requires a real NAS/node with Sonos CLI (or equivalent) and allowlisted template path.

---

## Tool index build logging

**Trace:** [AC-04.021](ep-acceptance-criteria.md#ac-04-021) · [REQ-04.025](ep-requirements.md#tool-index-and-pre-selection)

**Criterion:** On successful tool index build, an **INFO** log records completion (e.g. number of tools indexed). On failure, an **ERROR** log includes the reason.

**Steps:**

1. Start the service with a **valid** catalog and embedding config. Inspect logs during startup.
2. **Pass:** An informational line appears indicating tool index build success (and tool count if implemented).
3. **Failure path (staging only):** Temporarily break embedding (invalid API key, wrong URL) or make the vector path unusable; restart.
4. **Pass:** Startup fails or logs **ERROR** with an explicit failure reason (embedding error, store error, etc.), consistent with [AC-04.018](ep-acceptance-criteria.md#ac-04-018).

**Note:** Exact log strings depend on implementation; capture a sample line when signing off.

---

## system_prompt in system message

**Trace:** [AC-04.026](ep-acceptance-criteria.md#ac-04-026) · [REQ-04.032](ep-requirements.md#prompt-text-for-selected-tools)

**Criterion:** When selected tools define non-empty **system_prompt**, the system message for that request includes that text per tool id (e.g. marked section per id).

**Steps:**

1. Enable LLM request/response logging (e.g. `data/llm_logs/` or project-documented debug).
2. Send a message that pre-selects a tool with a distinctive **system_prompt** (e.g. **`node_time`** or **`sonos`** in the sample catalog).
3. Inspect the logged **system** (or first system message) for the completion request.
4. **Pass:** Content includes the tool’s **system_prompt** text (or an unambiguous per-tool block tied to that tool id, as implemented).

---

## Hermes tool list in prompt

**Trace:** [AC-04.027](ep-acceptance-criteria.md#ac-04-027) · [REQ-04.033](ep-requirements.md#prompt-text-for-selected-tools)

**Criterion:** For text-based tool mode, each tool line in the Hermes-style list uses **hermes_prompt** when non-empty, else **index_text**; parameters schema appears when arguments are defined.

**Prerequisites:** Provider with **supports_tools** false and **tools.text_based_enabled** true.

**Steps:**

1. Configure a Hermes-capable provider path as above.
2. Trigger a request that includes tools with different **hermes_prompt** vs **index_text** (e.g. **`sonos`** in sample catalog).
3. Inspect the prompt sent to the LLM (logs or capture).
4. **Pass:** Hermes tool list shows **hermes_prompt** lines where set; tools with arguments show parameter/schema lines as designed.

---

## Text-based tool flow

**Trace:** [AC-04.022](ep-acceptance-criteria.md#ac-04-022) · [AC-04.023](ep-acceptance-criteria.md#ac-04-023) · [AC-04.024](ep-acceptance-criteria.md#ac-04-024) · [REQ-04.026](ep-requirements.md#tool-invocation-without-tool-calling-api)–[REQ-04.029](ep-requirements.md#tool-invocation-without-tool-calling-api)

**Criterion:** Without native tools, the model is instructed to emit **`<tool_call>`** JSON; valid calls execute; malformed or unparseable output does not run commands and user sees text or a clear error.

**Steps:**

1. Set **supports_tools** false and **tools.text_based_enabled** true for the chat provider.
2. Ask for something that should invoke a simple tool (e.g. current time on NAS → **`node_time`**).
3. **Pass:** Assistant emits tool markup, system executes, follow-up uses tool result.
4. If the model returns plain text only (no valid tool_call), **Pass:** No node command is run for that turn unless a valid parse exists.
5. Optionally simulate or observe a malformed **`<tool_call>`** response (harder without mock); automated tests cover much of this — manual check confirms behaviour with your real model.

---

## Shell metacharacter rejection

**Trace:** [AC-04.029](ep-acceptance-criteria.md#ac-04-029) · [REQ-04.031](ep-requirements.md#validation-and-execution)

**Criterion:** After substitution, the command string must not contain shell metacharacters that could alter execution (rejected before **run_on_node**).

**Steps:**

1. Use a tool with string arguments (e.g. **`sonos`** **`tail`**).
2. Attempt to get the model to pass a **`tail`** value containing metacharacters (e.g. **`; rm -rf`**), or use a controlled test harness if available.
3. **Pass:** System rejects the call with a deterministic validation error; **no** such string is executed on the node.

**Note:** Automated coverage: **internal/cmdsafe**, **noderunner** `TestRunOnNode_shellMetacharacters_rejected`, **core** `TestExecuteOneToolCall_substitutedCommandWithMetachar_noRunOnNode`. This scenario is an optional live spot-check with the real model.

---

## Tool invocation in logs

**Trace:** [AC-04.013](ep-acceptance-criteria.md#ac-04-013) · [REQ-04.016](ep-requirements.md#nfr--security-testability-observability-consistency)

**Criterion:** Tool id, arguments, and result or error are traceable in logs when logging is enabled.

**Steps:**

1. Run a successful tool call and a failed validation path.
2. Search logs for the tool id and redacted or full args per project policy.
3. **Pass:** Success and failure paths are distinguishable and auditable.

---

## Optional manual checks (strategy §2.3)

- [AC-04.017](ep-acceptance-criteria.md#ac-04-017) / [REQ-04.021](ep-requirements.md#tool-index-and-pre-selection): Large catalog (many tools): confirm index build completes within ~20 s or documented fallback until ready.
- [AC-04.015](ep-acceptance-criteria.md#ac-04-015) / [AC-04.016](ep-acceptance-criteria.md#ac-04-016): Spot-check that irrelevant user messages yield a small tool subset or fallback list, not the full 1000-tool dump.
- [AC-04.028](ep-acceptance-criteria.md#ac-04-028) / [REQ-04.034](ep-requirements.md#provider-interface): With **supports_tools** false, confirm via HTTP proxy or logs that the provider request body omits the native **tools** array.
- **Native tool loop:** Two-step completion (tool_calls → second request with tool results) works with your primary OpenAI-compatible provider.
