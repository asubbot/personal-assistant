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
| [AC-08.005](#ac-08005) | [REQ-08.005](ep-requirements.md#json-response-format) | ForceJSONOutput sets json_object when JSON mode supported |
| [AC-08.006](#ac-08006) | [REQ-08.006](ep-requirements.md#json-response-format) | Explicit ResponseFormat wins over ForceJSONOutput and default |
| [AC-08.007](#ac-08007) | [REQ-08.007](ep-requirements.md#json-response-format) | Default response format when no per-request override |

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

### AC-08.005

**Trace:** [REQ-08.005](ep-requirements.md#json-response-format)

```gherkin
Given the LLMProvider has SupportsJSONMode enabled
And CompletionOptions.ForceJSONOutput is true
And CompletionOptions.ResponseFormat is unset
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include response_format with type "json_object"
```

### AC-08.006

**Trace:** [REQ-08.006](ep-requirements.md#json-response-format)

```gherkin
Given CompletionOptions.ResponseFormat is set to a valid type
When the OpenAICompatible provider builds a chat completion HTTP request body
Then the request body SHALL include response_format with the type from CompletionOptions.ResponseFormat
And the type SHALL not be taken from ForceJSONOutput or from DefaultResponseFormat when those sources would differ from CompletionOptions.ResponseFormat
```

### AC-08.007

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
| AC-08.005 | `TestOpenAICompatible_buildRequest_withForceJSONOutput_true` | `internal/llm/openai_test.go` |
| AC-08.006 | `TestOpenAICompatible_buildRequest_withExplicitResponseFormat_overridesForceJSON`; `TestOpenAICompatible_buildRequest_explicitOverridesDefault` | `internal/llm/openai_test.go` |
| AC-08.007 | `TestOpenAICompatible_buildRequest_withDefaultResponseFormat` (nil opts, default `json_object`); `TestOpenAICompatible_buildRequest_withoutForceJSONOutput_usesDefault` (explicit `ForceJSONOutput: false`, default `text`) | `internal/llm/openai_test.go` |

**Additional coverage (edges, not one-to-one with a single AC):**

| Tests | Intent |
|-------|--------|
| `TestOpenAICompatible_buildRequest_withForceJSONOutput_false` | `SupportsJSONMode=false`: `ForceJSONOutput` does not set `json_object`; default format still applied (relates to [REQ-08.005](ep-requirements.md#json-response-format) / [Non-functional requirements](ep-requirements.md#non-functional-requirements)). |
| `TestOpenAICompatible_buildRequest_explicitJSONObject_ignoredWithoutJSONMode` | Explicit `json_object` with unsupported JSON mode falls back (implementation guard). |
| `TestOpenAICompatible_buildRequest_emptyExplicitResponseFormatType_usesDefault`; `TestOpenAICompatible_buildRequest_emptyExplicitType_forceJSONStillApplies` | Whitespace/empty explicit `ResponseFormat.Type` and interaction with `ForceJSONOutput`. |
| `TestLoad_LLMProviderDefaults_boundaryTemperature_loads`; `TestLoad_InvalidConfig` rows for `llm_default_*` fixtures | Config load / fail-fast for temperature, `default_max_tokens`, `default_response_format`, `supports_json_mode` (`internal/config/config_test.go`). |
| `TestHandleMessage_textBasedHermes_twoToolRounds_preservesForceJSONOnEachComplete` (and related Hermes tests in `handler_test.go`) | Integration: handler keeps `ForceJSONOutput=true` across tool rounds so the provider can apply AC-08.005 on each completion. |

**Code comments:** Primary `internal/llm/openai_test.go` tests carry `// EP-008 AC-08.xxx / REQ-08.xxx` (or EP-008-only) comments above each function. Hermes integration tests in `internal/core/handler_test.go` and config validation in `internal/config/config_test.go` include EP-008 trace lines where applicable.
