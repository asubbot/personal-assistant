# Epic scope — EP-031 Vector Memory Search Tool

| Field | Content |
|-------|---------|
| **ID** | EP-031 |
| **Status** | DONE |
| **Title** | Vector Memory Search Tool |
| **Description** | Add a native, read-only tool for on-demand semantic retrieval from vector memory so the assistant can fetch relevant memory only when needed instead of always injecting auto-RAG context into the system prompt. The epic focuses on controllable retrieval, bounded output, and safe integration with existing skills/tool-calling flow. |
| **First version date** | 2026-04-21 |

## Glossary

- **Vector memory**: Semantic index stored in SQLite vector tables (`vec_notes`, `vec_summaries`, `vec_turns`) used for similarity search.
- **Auto-RAG**: Automatic retrieval path that injects retrieved chunks into the dynamic system tail for full-tier turns.
- **On-demand retrieval**: Retrieval executed via an explicit native tool call during a turn, only when needed.
- **Memory lanes**: Logical retrieval sources: notes, summaries, and conversation turns.
- **Tool output budget**: Prompt-side byte cap for tool messages used to prevent context explosion.

## Scope (features/capabilities)

- Add a new native read-only tool `search_vector_memory` for semantic retrieval from vector memory, callable by the LLM through native tool-calling.
- Support query-based retrieval across selectable memory lanes (`notes`, `summaries`, `turns`) with bounded `top_k`.
- Return compact, structured snippets suitable for prompt usage (not full large payloads), including stable source identifiers.
- Enforce strict guardrails for request and response size (argument validation, max results, output-size constraints, deterministic truncation strategy where applicable).
- Integrate the tool into native tool registry and runtime validation/allowlist paths so it can be referenced by runtime skills.
- Add/update runtime skill guidance so memory retrieval can happen on demand instead of relying only on auto-RAG system-tail injection.
- Preserve existing `read_memory` and `write_memory` behavior; this epic adds semantic retrieval capability and does not replace calendar-based memory reads.
- Keep retrieval behavior deterministic and observable (clear logs/telemetry attributes for invocation and lane usage).
- Provide tests across unit/integration levels for validation, retrieval behavior, bounds, and error paths.
- Update operator/developer documentation for configuration and usage expectations of the new tool.

## Tool naming decision

- **Final tool id:** `search_vector_memory`
- **Rationale:** Explicitly indicates vector-memory semantic retrieval and preserves `search_memory` for planned calendar-memory search.

## Success criteria

- The assistant can call the new native vector-memory search tool in normal tool-calling flow and receive relevant results from configured lanes.
- Tool responses remain within bounded prompt budget and do not cause uncontrolled context growth.
- Validation rejects invalid arguments (empty query, unsupported lane, out-of-range limits) with clear deterministic errors.
- Runtime skills can reference and use the new tool through existing skill loading/validation paths.
- Existing memory tools (`read_memory`, `write_memory`) and memory indexing flows continue to work without regression.
- `make check` passes with new tests.
- `make validate` (without parameters) passes after adding this epic changes.
- E2E scenario passes: user asks a question that requires prior semantic context, the assistant invokes the new vector-memory tool, receives bounded relevant snippets, and produces a grounded final answer without enabling auto-RAG system-tail retrieval.

## Traceability

- **Scope:** Supports reliability and maintainability goals in [scope.md](../../scope.md) by reducing unnecessary context injection and keeping memory retrieval explicit, bounded, and testable.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) testability and incremental delivery goals by introducing a controlled retrieval mechanism with clear unit/integration coverage.
- **Related epics:** Extends memory foundations from [EP-002](../EP-002/ep-scope.md) and [EP-016](../EP-016/ep-scope.md), and complements runtime-skills/tool-selection behavior established in [EP-013](../EP-013/ep-scope.md) and [EP-018](../EP-018/ep-scope.md).
