# EP-008 LLM Parameters Enhancement — Acceptance criteria

This document defines testable acceptance criteria for EP-008, traceable to [ep-requirements.md](ep-requirements.md) and aligned with [ep-scope.md](ep-scope.md).

## Introduction

EP-008 adds provider defaults and per-request controls for LLM **temperature**, **max_tokens**, and **response_format** (including JSON mode for text-based tools). These criteria verify that the OpenAI-compatible provider builds HTTP request bodies that match the priority rules in the requirements.

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|-------------|---------|
| [AC-08.001](#ac-08001) | [REQ-08.001](ep-requirements.md#temperature-configuration) | Default temperature appears in HTTP body when configured |
| [AC-08.002](#ac-08002) | [REQ-08.002](ep-requirements.md#temperature-configuration) | Request temperature overrides provider default |
| [AC-08.003](#ac-08003) | [REQ-08.003](ep-requirements.md#max-tokens-configuration) | Default max_tokens appears when provider default &gt; 0 |
| [AC-08.004](#ac-08004) | [REQ-08.004](ep-requirements.md#max-tokens-configuration) | Request max_tokens overrides provider default when &gt; 0 |
| [AC-08.005](#ac-08005) | [REQ-08.005](ep-requirements.md#json-response-format) | **Obsolete:** `SupportsJSONMode` / `ForceJSONOutput` removed; product uses text `response_format` only. |
| [AC-08.006](#ac-08006) | [REQ-08.006](ep-requirements.md#json-response-format) | **Obsolete:** Per-request JSON shaping via removed flags. |
| [AC-08.007](#ac-08007) | [REQ-08.007](ep-requirements.md#json-response-format) | **Obsolete:** Default `json_object` path removed; default is text-only. |

## Acceptance criteria

### AC-08.001

**Trace:** [REQ-08.001](ep-requirements.md#temperature-configuration)

```gherkin
Given the LLMProvider has DefaultTemperature configured
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include a temperature field with the provider default value
```

### AC-08.002

**Trace:** [REQ-08.002](ep-requirements.md#temperature-configuration)

```gherkin
Given the LLMProvider has DefaultTemperature configured
And CompletionOptions.Temperature is set
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL use the CompletionOptions.Temperature value instead of the provider default
```

### AC-08.003

**Trace:** [REQ-08.003](ep-requirements.md#max-tokens-configuration)

```gherkin
Given the LLMProvider has DefaultMaxTokens greater than zero
And CompletionOptions.MaxTokens is zero or unset
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include max_tokens with the provider default value
```

### AC-08.004

**Trace:** [REQ-08.004](ep-requirements.md#max-tokens-configuration)

```gherkin
Given the LLMProvider has DefaultMaxTokens greater than zero
And CompletionOptions.MaxTokens is greater than zero
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL use the CompletionOptions.MaxTokens value instead of the provider default
```

### AC-08.005 **Obsolete:** JSON-mode completion shaping (`json_object` via `ForceJSONOutput` / `SupportsJSONMode`) was removed from the product; see text-only `response_format` tests in `internal/llm` and config validation.

**Trace:** [REQ-08.005](ep-requirements.md#json-response-format)

```gherkin
Given the LLMProvider has SupportsJSONMode enabled
And CompletionOptions.ForceJSONOutput is true
And CompletionOptions.ResponseFormat is unset
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include response_format with type "json_object"
```

### AC-08.006 **Obsolete:** Same as AC-08.005; explicit per-request JSON overrides are out of scope for the current provider contract.

**Trace:** [REQ-08.006](ep-requirements.md#json-response-format)

```gherkin
Given CompletionOptions.ResponseFormat is set to a valid type
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include response_format with the type from CompletionOptions.ResponseFormat
And the type SHALL not be taken from ForceJSONOutput or from DefaultResponseFormat when those sources would differ from CompletionOptions.ResponseFormat
```

### AC-08.007 **Obsolete:** Default response format is fixed to `text`; REQ-08.007 gherkin below describes the removed JSON-default behaviour.

**Trace:** [REQ-08.007](ep-requirements.md#json-response-format)

```gherkin
Given the LLMProvider has DefaultResponseFormat configured
And CompletionOptions.ResponseFormat is unset
And CompletionOptions.ForceJSONOutput is false
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include response_format with the configured default type
```

---

## Automated test traceability (Go)

The table below maps each AC to **primary** unit tests that assert the HTTP request body (or equivalent provider behaviour). Paths are relative to the repository root.

| AC | Primary test(s) | Package / file |
|----|-----------------|----------------|
| AC-08.001 | `TestOpenAICompatible_buildRequest_withDefaultTemperature` | `internal/llm/openai_test.go` |
| AC-08.002 | `TestOpenAICompatible_buildRequest_withOverrideTemperature` | `internal/llm/openai_test.go` |
| AC-08.003 | `TestOpenAICompatible_buildRequest_withDefaultMaxTokens` | `internal/llm/openai_test.go` |
| AC-08.004 | `TestOpenAICompatible_buildRequest_withOverrideMaxTokens` | `internal/llm/openai_test.go` |
| AC-08.005 | **OBSOLETE** (see criterion row) | — |
| AC-08.006 | **OBSOLETE** (see criterion row) | — |
| AC-08.007 | **OBSOLETE** (see criterion row) | — |

**Additional coverage (edges, not one-to-one with a single AC):**

| Tests | Intent |
|-------|--------|
| `TestOpenAICompatible_buildRequest_withForceJSONOutput_false` | `SupportsJSONMode=false`: `ForceJSONOutput` does not set `json_object`; default format still applied (relates to [REQ-08.005](ep-requirements.md#json-response-format) / [Non-functional requirements](ep-requirements.md#non-functional-requirements)). |
| `TestOpenAICompatible_buildRequest_explicitJSONObject_ignoredWithoutJSONMode` | Explicit `json_object` with unsupported JSON mode falls back (implementation guard). |
| `TestOpenAICompatible_buildRequest_emptyExplicitResponseFormatType_usesDefault`; `TestOpenAICompatible_buildRequest_emptyExplicitType_forceJSONStillApplies` | Whitespace/empty explicit `ResponseFormat.Type` and interaction with `ForceJSONOutput`. |
| `TestLoad_LLMProviderDefaults_boundaryTemperature_loads`; `TestLoad_InvalidConfig` rows for `llm_default_*` fixtures | Config load / fail-fast for temperature, `default_max_tokens`, `default_response_format`, `supports_json_mode` (`internal/config/config_test.go`). |
| Text-only `response_format` / rejection of removed JSON-mode config | `internal/llm/openai_test.go`, `internal/config/config_test.go` |

**Code comments:** `internal/llm/openai_test.go` tests for temperature, max tokens, and text `response_format` carry EP-008 / REQ-08 trace lines where they still apply (AC-08.001–AC-08.004). JSON-mode ACs AC-08.005–AC-08.007 are **Obsolete** in the index above.
