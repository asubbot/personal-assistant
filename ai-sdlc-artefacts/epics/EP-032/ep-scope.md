# Epic scope — EP-032 Specialized Knowledge Search Tools

| Field | Content |
|-------|---------|
| **ID** | EP-032 |
| **Status** | NEW |
| **Title** | Specialized Knowledge Search Tools |
| **Description** | Keep `search_vector_memory` focused on personal memory retrieval and add two specialized read-only tools for knowledge about tools and skills. This separates domains, reduces ambiguity, and keeps retrieval behavior explicit and bounded. |
| **First version date** | 2026-04-21 |

## Glossary

- **Tool knowledge**: Semantic knowledge indexed from tool definitions, docs, and operator-facing descriptions (for example `vec_tools`).
- **Skill knowledge**: Semantic knowledge indexed from skill documents and skill metadata (for example `vec_skills`).
- **`search_vector_tool`**: New native read-only tool for semantic retrieval in tool-knowledge domain.
- **`search_vector_skill`**: New native read-only tool for semantic retrieval in skill-knowledge domain.
- **`vector_search_tools` config block**: Single config block with limits and switches for vector-search tools, including `search_vector_memory`, `search_vector_tool`, and `search_vector_skill`.
- **Domain isolation**: Retrieval constraint where each specialized tool only queries its own vector index/domain.
- **Bounded output**: Deterministic output-size and result-count limits for tool responses to avoid prompt bloat.

## Scope (features/capabilities)

- Add native read-only tool `search_vector_tool` for semantic retrieval from tool-knowledge vector domain.
- Add native read-only tool `search_vector_skill` for semantic retrieval from skill-knowledge vector domain.
- Keep `search_vector_memory` unchanged and memory-only (`notes`, `summaries`, `turns`) with no cross-domain widening in this epic.
- Add one unified config block `vector_search_tools` for all vector-search tools (`search_vector_memory`, `search_vector_tool`, `search_vector_skill`) instead of separate per-tool config roots.
- Define strict input validation (`query`, optional `top_k`, optional domain-specific filters where applicable) with deterministic errors.
- Enforce deterministic ordering and bounded output policy for both new tools.
- Move existing `search_vector_memory` runtime limits to the unified `vector_search_tools` block to remove hardcoded registration values.
- Integrate both tool IDs into native tool registry and runtime-skill allowlist/validation.
- Add unit/integration/E2E coverage for positive, negative, and boundary scenarios (including auto-RAG-disabled context path).
- Update docs and skill examples to explain when to use `search_vector_memory` vs `search_vector_tool` vs `search_vector_skill`.
- Preserve backward compatibility and avoid regressions in existing memory tools and runtime skill loading flow.
- Ensure structured invocation logging and existing redaction policy are applied to the new tool calls.

## Success criteria

- Assistant can invoke `search_vector_tool` and return relevant bounded snippets about available tools in one turn.
- Assistant can invoke `search_vector_skill` and return relevant bounded snippets about available skills in one turn.
- `search_vector_memory` behavior remains unchanged and passes existing EP-031 expectations.
- Runtime limits for `search_vector_memory`, `search_vector_tool`, and `search_vector_skill` are read from one shared `vector_search_tools` config block.
- Invalid inputs (empty query, unsupported parameters, out-of-range limits) are rejected with deterministic validation errors.
- Runtime skill validation accepts the two new native tool IDs where configured.
- Structured tool invocation logs are emitted with sensitive-data redaction intact.
- `make check` passes for the full change set.
- `make validate` (without parameters) passes after adding this epic changes.
- E2E scenario passes: user asks about which tool/skill should be used, assistant calls corresponding specialized tool, receives bounded relevant snippets, and provides grounded answer without uncontrolled context growth.

## Traceability

- **Scope:** Extends retrieval capabilities in [scope.md](../../scope.md) by separating knowledge domains and improving reliability of retrieval decisions.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) traceability and testability goals via explicit AC coverage at unit/integration/E2E levels.
- **Related epics:** Builds on [EP-031](../EP-031/ep-scope.md) by preserving memory retrieval as a separate concern and adding tool/skill knowledge retrieval via specialized native tools.
