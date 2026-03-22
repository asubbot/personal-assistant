# Epic scope — EP-008 LLM Parameters Enhancement

| Field | Content |
|-------|---------|
| **ID** | EP-008 |
| **Status** | DONE |
| **Title** | LLM Parameters Enhancement |
| **Description** | Add temperature, max_tokens, and JSON response format configuration to LLM providers for improved control, reliability, and cost optimization. |
| **First version date** | 2026-03-22 |

## Glossary

- **Temperature**: LLM sampling parameter controlling randomness/creativity (0 = deterministic, 1 = creative)
- **Max Tokens**: Maximum number of tokens in LLM response, used for cost control and response length limiting
- **JSON Mode**: API feature forcing LLM to output valid JSON (response_format: json_object)
- **ResponseFormat**: API parameter specifying desired output format (text, json_object)
- **ForceJSONOutput**: Hint flag in CompletionOptions to request JSON output for text-based tool mode
- **Hermes format**: Text-based tool calling format for models without a native tool API (tool calls encoded in model text with delimiters)

## Scope (features/capabilities)

- Provider-level default configuration for temperature
- Provider-level default configuration for max_tokens
- Per-request override of temperature via CompletionOptions
- Per-request override of max_tokens via CompletionOptions
- Provider-level flag indicating JSON mode support (supports_json_mode)
- Automatic JSON output hint for text-based tools (ForceJSONOutput)
- Flexible ResponseFormat configuration per-request
- Provider-level default response format configuration
- Priority chain: explicit ResponseFormat > ForceJSONOutput > defaultResponseFormat

## Success criteria

- Provider config accepts default_temperature and default_max_tokens; values are applied to API requests
- CompletionOptions allows per-request temperature and max_tokens overrides
- Text-based tool mode (Hermes) uses JSON mode when provider supports it
- ResponseFormat can be explicitly set per-request and overrides defaults
- All existing tests pass after changes
- No breaking changes to existing provider configurations (new fields are optional)

## Traceability

- **Scope:** Extends LLM provider capabilities from [scope.md](../../scope.md) (LLM provider abstraction, tool extensibility)
- **Strategy:** Supports MVP evolution without breaking core per [strategy.md](../../strategy.md)
