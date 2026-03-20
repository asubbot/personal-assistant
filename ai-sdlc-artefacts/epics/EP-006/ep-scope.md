# Epic scope — EP-006 Tool-call reliability and model escalation

| Field | Content |
|-------|---------|
| **ID** | EP-006 |
| **Status** | IN PROGRESS |
| **Title** | Tool-call reliability and model escalation |
| **Description** | Improve automatic recovery when tool invocations fail or the model produces unusable tool output, using the ordered list of configured LLM providers without requiring the user to intervene. Introduce explicit policy (error classification, limits, observability), a **configurable baseline** model (not assumed minimal/cheap), and **rollback at end of user turn** (Option A): the next user message always starts from the configured baseline; mid-turn rollback variants are out of scope. |
| **First version date** | 2026-03-19 (date of first approved write of this document; start of work on the epic idea) |

## Glossary

- **Baseline (default) model:** The LLM provider PA should prefer when starting a user message (or after rollback), as **chosen in configuration**. It is **not** required to be the smallest/cheapest tier; operators may point the baseline at any configured provider (e.g. by explicit provider id/name, or by index into `llm_providers`, or another simple mapping—exact mechanism is defined in requirements/design). Escalation still walks an **ordered** provider list from that baseline’s position (or a defined chain) according to policy.
- **Escalation:** For a given user message handling path, temporarily using a later provider in the ordered escalation chain (higher index or next in list) for one or more `Complete` calls, according to policy, to improve reasoning or tool-call formatting after a qualifying failure.
- **Tool failure (for policy):** Any outcome where a tool invocation does not complete successfully from the core’s perspective (validation error before execution, allowlist denial, SSH/exec error, etc.), or where structured tool flow cannot proceed (e.g. unrecoverable parse error in a defined text-tool path). Exact classification is specified in later requirements; not every error class warrants escalation.
- **Successful tool round:** A processing step in which all tool calls requested in the current assistant turn have been executed and each result appended to the conversation (success or deterministic error text in the tool/user message), and the handler is about to call `Complete` again for the model’s follow-up reply.

## Scope (features/capabilities)

- **Error classification:** Define stable categories for tool-related and tool-flow failures (e.g. policy/security vs transient execution vs model-format). Map categories to allowed actions: no escalation, one repair attempt on the same provider, escalate once to the next provider, or stop.
- **Escalation policy package:** Centralize the mapping from classified causes to escalation allowance in `pa/internal/escalationpolicy` (see [ep-system-design.md](ep-system-design.md), [REQ-06.017](ep-requirements.md#nfr--security-testability-observability)) for testability and clear module boundaries; implementation tracked in [ep-implementation-plan.md](ep-implementation-plan.md) Task 8.
- **Typed escalation inputs:** Represent tool-path outcomes that drive escalation using explicit error types (or equivalent) inspectable via the language error API, not only unstructured message text ([REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) in requirements).
- **Hermes parse and escalation:** When the text-tool (Hermes) parser fails after a `Complete`, apply the same bounded escalation policy as for qualifying tool execution failures where configured ([REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)).
- **Escalation policy:** Bounded behaviour per user message: maximum number of escalations per turn, respect for existing tool-round caps, and no escalation for errors that a stronger model cannot fix (e.g. allowlist denial, unknown tool id in catalog).
- **Multi-provider chain:** Support ordered lists with more than two providers; escalation advances strictly along configuration order until policy stops or the list is exhausted.
- **Exhaustion and stop:** When escalation cannot help or the chain is exhausted, produce a deterministic user-visible outcome and structured logs; do not loop indefinitely.
- **Observability:** Log decisions (classification, escalation yes/no, provider index/label before and after, optional `tried_providers` summary) without secrets; align with existing tool invocation logging where practical.
- **Configuration:** Operator-controlled switches and limits (e.g. enable/disable escalation, max escalations per turn, **which provider is the baseline** for new handling, optional cooldown hints for future use) loaded and validated at startup.
- **Rollback to baseline model:** **Chosen behaviour — end of user turn (Option A).** After the assistant's final text reply is sent for the message, the next user message always starts from the **configured baseline** provider. No mid-turn rollback; one clear rule at message boundary (KISS, predictable state). Alternatives (e.g. rollback after each successful tool round, or configurable enum) are out of scope for this epic.

## Success criteria

- With escalation **disabled**, no escalation occurs; each message is handled according to the configured baseline only.
- With escalation **enabled**, a qualifying simulated failure triggers at most the configured number of provider advances per user message; non-qualifying failures do not advance the provider.
- With three or more providers configured, policy can advance at most along the chain and stops cleanly at the last provider with a defined user-visible or logged outcome.
- Logs allow an operator to answer: which provider served which `Complete`, whether escalation occurred, which **baseline** provider is configured, and that the next user message starts from baseline (rollback at end of turn).
- Unit and/or integration tests cover classification tables (or key branches), escalation limits, exhaustion, and rollback-at-end-of-turn with a mock provider chain.

## Traceability

- **Scope:** [scope.md](../../scope.md) — PersonalAssistant focuses on reliability; the core orchestrates LLM and tools. This epic improves automatic handling when tools or tool-formatting fail, without changing the security model (allowlists, dedicated user, validation before execution).
- **Strategy:** [strategy.md](../../strategy.md) — Testable increments; integration tests for meaningful subsets; security checks remain explicit. Builds on the existing multi-provider fallback and tool pipeline delivered under [EP-004](../EP-004/ep-scope.md).
- **Manual tests:** [ep-manual-tests.md](ep-manual-tests.md) — operator scenarios for real LLM/Telegram/log checks (supplements unit and integration tests).
- **Related epics:** [EP-004](../EP-004/ep-scope.md) (tools, tool-result loop, providers); [EP-001](../EP-001/ep-scope.md) (core and nodes); optional interaction with [EP-005](../EP-005/ep-scope.md) (execution transport) only if execution error classification must distinguish subsystem vs legacy errors in policy.

## Out of scope (this epic)

- **Mid-turn rollback** (e.g. after each successful tool round, sticky-until-success, or a configurable enum of rollback modes); this epic implements **end-of-turn** only.
- Automatic retry of the **same** remote command without a new model decision (no blind re-exec loops).
- Changing tool catalog schema or node allowlist semantics.
- Billing dashboards or external A/B infrastructure beyond logs and config.
- Replacing the existing provider fallback for **pure** LLM API transport errors (that behaviour may be extended in a separate change unless explicitly merged into this epic during requirements).
