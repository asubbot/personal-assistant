# EP-006 Tool-call reliability and model escalation — Acceptance criteria

**Pipeline:** Stage 5 (Acceptance criteria).

**Contents**

- [Introduction](#introduction)
- [Acceptance criteria index](#acceptance-criteria-index)
- [Acceptance criteria](#acceptance-criteria)

---

## Introduction

This document defines epic-level acceptance criteria for **EP-006 Tool-call reliability and model escalation**. It contains testable conditions (Gherkin-style) for the epic. Traceability to [ep-requirements.md](ep-requirements.md) is given per AC below.

**Purpose:** Historical acceptance criteria for EP-006 tool-path LLM escalation. Tool-path escalation behaviour was removed by [EP-034](../EP-034/ep-scope.md); most criteria are retained for REQ traceability. **AC-06.015** remains active for catalog `ValidateKind` typing (REQ-06.018).

---

## Acceptance criteria index

| AC ID | REQ | Summary |
|-------|-----|---------|
| [AC-06.001](#ac-06-001) | [REQ-06.001](ep-requirements.md#baseline-and-configuration) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.002](#ac-06-002) | [REQ-06.002](ep-requirements.md#baseline-and-configuration) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.003](#ac-06-003) | [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.004](#ac-06-004) | [REQ-06.005](ep-requirements.md#error-classification) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.005](#ac-06-005) | [REQ-06.006](ep-requirements.md#escalation-policy-and-chain) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.006](#ac-06-006) | [REQ-06.007](ep-requirements.md#escalation-policy-and-chain) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.007](#ac-06-007) | [REQ-06.008](ep-requirements.md#exhaustion-and-stop) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.008](#ac-06-008) | [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.009](#ac-06-009) | [REQ-06.010](ep-requirements.md#observability), [REQ-06.011](ep-requirements.md#observability), [REQ-06.012](ep-requirements.md#nfr--security-testability-observability) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.010](#ac-06-010) | [REQ-06.013](ep-requirements.md#nfr--security-testability-observability) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.011](#ac-06-011) | [REQ-06.014](ep-requirements.md#nfr--security-testability-observability) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.012](#ac-06-012) | [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.013](#ac-06-013) | [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | **Obsolete:** Text-markup tool parse + escalation path removed; native `tool_calls` only. |
| [AC-06.014](#ac-06-014) | [REQ-06.017](ep-requirements.md#nfr--security-testability-observability) | **Obsolete:** `internal/escalationpolicy` removed by [EP-034](../EP-034/ep-scope.md) |
| [AC-06.015](#ac-06-015) | [REQ-06.018](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs) | Catalog validate errors: policy uses typed `ValidateKind` / `errors.As`, not `Error()` substring rules |

---

## Acceptance criteria

<a id="ac-06-001"></a>**AC-06.001** (Trace: [REQ-06.001](ep-requirements.md#baseline-and-configuration)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given PersonalAssistant is configured with a baseline provider **B** distinct from the first list entry (where supported),  
When a new user message begins handling,  
Then the first Complete call for that user message uses provider **B** (the configured baseline).

---

<a id="ac-06-002"></a>**AC-06.002** (Trace: [REQ-06.002](ep-requirements.md#baseline-and-configuration)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given the operator sets enable/disable escalation, maximum escalations per user message, and baseline provider in configuration,  
When the service starts,  
Then the configuration is loaded and validated.  
And invalid or missing required values result in startup failure or a defined documented default consistent with [ep-scope.md](ep-scope.md).

---

<a id="ac-06-003"></a>**AC-06.003** (Trace: [REQ-06.003](ep-requirements.md#error-classification), [REQ-06.004](ep-requirements.md#error-classification)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given a tool-related or tool-flow failure occurs during handling of a user message,  
When the core applies error classification,  
Then the failure is assigned to a stable category (e.g. policy/security, transient execution, model-format).  
And that category maps to exactly one allowed action among: no escalation, one repair on same provider, escalate once to next provider, or stop.

---

<a id="ac-06-004"></a>**AC-06.004** (Trace: [REQ-06.005](ep-requirements.md#error-classification)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given escalation is enabled and a tool failure is an allowlist denial or an unknown tool id in the catalog,  
When the core processes that failure,  
Then the provider index does not advance for escalation because of that failure alone.

---

<a id="ac-06-005"></a>**AC-06.005** (Trace: [REQ-06.006](ep-requirements.md#escalation-policy-and-chain)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given escalation is enabled, the maximum escalations per user message is **N**, and a qualifying tool failure occurs,  
When the core handles that failure according to policy,  
Then the next Complete call uses the next provider in the configured ordered list.  
And at most **N** escalations occur for that single user message (subject to existing tool-round caps).

---

<a id="ac-06-006"></a>**AC-06.006** (Trace: [REQ-06.007](ep-requirements.md#escalation-policy-and-chain)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given at least three LLM providers are configured in list order **P0**, **P1**, **P2**,  
When escalation advances the active provider within one user message,  
Then each advance moves to the immediate next entry in that list (no skipping entries and no order other than the configured list order) until the last entry is reached or policy stops escalation.

---

<a id="ac-06-007"></a>**AC-06.007** (Trace: [REQ-06.008](ep-requirements.md#exhaustion-and-stop)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given escalation is enabled and the provider chain is exhausted or policy dictates stop for the current user message,  
When the core finishes handling for that user message,  
Then the user receives a deterministic visible outcome (reply or error as designed).  
And structured logs record the terminal state.  
And the core does not issue further escalation attempts for that same user message.

---

<a id="ac-06-008"></a>**AC-06.008** (Trace: [REQ-06.009](ep-requirements.md#rollback-at-end-of-turn)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given the assistant has sent the final text reply for user message **M** while using an escalated provider **P_k** (not the baseline),  
When the user sends a new user message **M+1**,  
Then the first Complete call for **M+1** uses the configured baseline provider.

---

<a id="ac-06-009"></a>**AC-06.009** (Trace: [REQ-06.010](ep-requirements.md#observability), [REQ-06.011](ep-requirements.md#observability), [REQ-06.012](ep-requirements.md#nfr--security-testability-observability)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given a user message is handled with escalation decisions,  
When an operator inspects logs for that path,  
Then each relevant decision includes classification result, whether escalation occurred, and provider index or label before and after (where applicable).  
And log lines for escalation or provider selection do not contain API keys, tokens, or other configured secrets.  
And where **tried_providers** (or equivalent) is implemented, an optional summary may appear without violating the no-secrets rule.

---

<a id="ac-06-010"></a>**AC-06.010** (Trace: [REQ-06.013](ep-requirements.md#nfr--security-testability-observability)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given the implementation is complete for this epic,  
When the test suite runs,  
Then unit and/or integration tests exist that cover classification (tables or key branches), escalation limits, exhaustion behaviour, and rollback-at-end-of-turn using a mock provider chain.

---

<a id="ac-06-011"></a>**AC-06.011** (Trace: [REQ-06.014](ep-requirements.md#nfr--security-testability-observability)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given escalation is disabled in configuration,  
When a tool failure that would qualify for escalation if escalation were enabled occurs,  
Then all Complete calls for that user message use only the configured baseline provider.  
And the provider does not advance along the list solely because of that failure.

---

<a id="ac-06-012"></a>**AC-06.012** (Trace: [REQ-06.015](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)) **Obsolete:** Tool-path escalation removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given tool execution or node execution returns an error wrapped with the typed escalation policy used by the core,  
When the core evaluates whether to escalate after that failure,  
Then the decision uses that type (or equivalent inspection API) and not only substring matching on `Error()` text.  
And given an otherwise similar failure returned as a plain error without that typing,  
When the core evaluates escalation qualification,  
Then that plain error does not qualify for escalation (fail closed).

---

<a id="ac-06-013"></a>**AC-06.013** (Trace: [REQ-06.016](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs)) **Obsolete:** No text-markup tool parse path; escalation on that failure class is not applicable. This AC is retained for historical REQ traceability only.

Given escalation is enabled, the text-based Hermes tool path is active, and the assistant's completion content fails Hermes parsing after a Complete call (first completion or follow-up in the tool loop),  
When the core applies escalation policy and a next provider exists and the per-message escalation limit allows another advance,  
Then the next Complete call uses the next provider in the configured ordered list.  
And when no further escalation is allowed,  
Then the user receives the same deterministic invalid-format or parse-error outcome as defined for that path before this requirement (no infinite loop).

---

<a id="ac-06-014"></a>**AC-06.014** (Trace: [REQ-06.017](ep-requirements.md#nfr--security-testability-observability)) **Obsolete:** `internal/escalationpolicy` removed by [EP-034](../EP-034/ep-scope.md); retained for historical REQ traceability.

Given the repository layout defined in [ep-system-design.md](ep-system-design.md),  
When an engineer inspects the implementation of escalation-allowance mapping for tool-path failures used by the conversation handler,  
Then that mapping is implemented in the Go package `pa/internal/escalationpolicy` (not only ad hoc inside the handler).  
And when unit tests for that package run,  
Then they exercise the mapping table or equivalent without starting the full conversation handler, Telegram adapter, or real LLM client.

---

<a id="ac-06-015"></a>**AC-06.015** (Trace: [REQ-06.018](ep-requirements.md#typed-tool-failures-and-hermes-parse-escalation-inputs))

Given a failure returned from `toolcatalog.ValidateToolCall` (including unknown tool and invalid arguments),  
When `internal/escalationpolicy.WrapCatalogValidateError` maps that failure to `toolfailure.NoEscalate` or `MayEscalate`,  
Then the decision uses `errors.As` on the dedicated catalog validation error type and its `ValidateKind` (or equivalent), and does not classify catalog validation outcomes by scanning substrings of `Error()` text alone.  
And given an error that is not a catalog validation error of that type but resembles validate message text,  
When `WrapCatalogValidateError` is applied,  
Then the outcome fails closed to non-qualifying escalation (same as [AC-06.012](ep-acceptance-criteria.md#ac-06-012) for untyped errors).

---

**Traceability:** [ep-requirements.md](ep-requirements.md) · [ep-scope.md](ep-scope.md)
