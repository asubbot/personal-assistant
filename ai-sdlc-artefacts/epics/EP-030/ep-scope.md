# Epic scope — EP-030 Remove Hermes text-based tool path

| Field | Content |
|-------|---------|
| **ID** | EP-030 |
| **Status** | NEW |
| **Title** | Remove Hermes text-based tool path |
| **Description** | Remove the product feature that lets the main LLM invoke tools by emitting Hermes-style `<tool_call>` blocks in free text when the first configured provider does not support native tool calling. After this epic, tool execution for the conversation path is available only through the provider tool API (native tool calls), simplifying prompts, handler logic, and observability. |
| **First version date** | 2026-04-19 |

## Glossary

- **Hermes text path**: Conversation flow where the system instructs the model to output tool invocations as marked-up JSON inside `<tool_call>…</tool_call>` in assistant message content; the core parses that text when native tool calls are not used for the first provider.
- **Native tool calling**: Provider and HTTP path where tool definitions are sent in the completion request and the model returns structured tool calls in the API response (not parsed from assistant prose).
- **Text-based tools flag**: Configuration field `tools.text_based_enabled` that today gates whether the Hermes instructions and parsing path may activate together with provider capability flags.

## Scope (features/capabilities)

- **Remove Hermes parsing and follow-up loop** from the conversation handler: no parsing of assistant content into tool calls via the Hermes markup convention; no second completion solely to recover from Hermes parse failures (e.g. escalation class tied to that path).
- **Remove Hermes instructions from dynamic system tail** wherever they are assembled for main-turn prompts (all intent tiers and shared helpers that currently inject Hermes format text when the text path would apply).
- **Remove or repurpose the `internal/tooltext` package** (Hermes format description, parsers, helpers used only for this path), including catalog tool helpers that exist only for Hermes prompt bodies, unless a small stable subset is still required for unrelated features (if none, delete the package).
- **Configuration and load-time validation**: `tools.text_based_enabled` is removed from the explicit product configuration contract, or retained only if repurposed with a documented no-op or replacement semantics agreed in requirements stage—default outcome is **removal** so operators cannot enable a removed path. All documented top-level keys in `config.json` remain explicit per product rules; unknown keys stay rejected; invalid combinations fail at load.
- **Provider contract**: Document that conversation tools require `supports_tools: true` on the first (or primary) provider used for the main turn when tools are enabled—exact wording and any fallback behaviour (e.g. clear error vs. silent omission) are fixed in requirements and acceptance criteria stages.
- **Dynamic tool selection interaction**: Where tier or dynamic-tool logic today depends on `text_based_enabled` (for example, gating dynamic cap behaviour), behaviour is simplified or re-specified so full_lite and full tiers remain coherent without the Hermes path (parity rules for tiers defined in follow-on requirements).
- **Observability**: Remove or replace log attributes and failure classes that refer to Hermes or `invoked_via=hermes` so traces remain accurate and grep-clean.
- **Tests and validation**: Delete or rewrite unit and integration tests that assert Hermes behaviour; update `./bin/validate` wiring for this epic’s acceptance criteria when added in stage 5; `make check` passes at epic completion.
- **Documentation**: Update operator-facing configuration docs and any architecture notes that describe the Hermes text path so they match the post-epic product.

## Out of scope (deferred)

- **Adding a replacement text protocol** for models without native tools (e.g. a new delimiter format)—treat as a separate epic if needed later.
- **Changing third-party servers** (Ollama, vLLM, etc.) to add tool APIs—outside this repository.
- **Classifier or tier naming** beyond what is strictly required to remove Hermes dependencies (large EP-017 or EP-018 refactors without linkage to this removal).

## Success criteria

- With tools enabled for conversations and a valid configuration, the core never injects Hermes tool-format instructions into the system message and never attempts to parse `<tool_call>` blocks from assistant content for tool execution.
- Configuration that previously relied on `text_based_enabled: true` is either rejected with a clear load error (if the key is removed) or documented migration is available; no silent partial behaviour.
- All automated tests pass (`make check`); epic acceptance criteria validated with `./bin/validate EP-030` once criteria and traceability comments are added in the pipeline.
- No remaining product references to the removed path in code paths exercised by tests (dead code in comments only is not sufficient—implementation and tests align with “native tools only” for tool execution on the main conversation turn).

## Traceability

- **Scope:** Supports reliability and maintainability goals in [scope.md](../../scope.md) (simpler core behaviour, fewer model-dependent parsing paths). Aligns with “swappable LLM backends” when combined with documented requirement for native tool support where tools are used.
- **Strategy:** Fits [strategy.md](../../strategy.md) delivery increment **v0.01** and the rule that existing behaviour stays testable: removal is covered by updated tests and clear operator docs.
- **Related epics:** Builds on tool and tier work described in [EP-004](../EP-004/ep-scope.md) (structured tools) and [EP-018](../EP-018/ep-scope.md) (tiers and prompt assembly that referenced Hermes gating); EP-030 narrows the product by removing one execution path rather than extending tiers.
