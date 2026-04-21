---
name: Memory retrieval
description: "When the user asks about memory, tools, or skills, choose the matching retrieval tool: read_memory (date/range), search_vector_memory (memory semantics), search_vector_tool (tool knowledge), search_vector_skill (skill knowledge)."
tools: ["read_memory", "search_vector_memory", "search_vector_tool", "search_vector_skill"]
---

## Policy

- Use **`read_memory`** with **`date`** (YYYY-MM-DD) for a single day, or **`from`** and **`to`** (inclusive) for a bounded range.
- Use **`search_vector_memory`** for semantic lookup when the user does not know exact dates and asks by topic or meaning.
- Use **`search_vector_tool`** for questions about available tools, tool intent, and tool selection.
- Use **`search_vector_skill`** for questions about skills, when to apply them, and skill capabilities.
- Dates are interpreted in the assistant **`pa_timezone`** from config (same as automatic summarization).
- Do not invent paths; only structured ISO arguments are allowed.
- For vague requests (“last week”), ask the user for concrete dates or offer a small range within configured limits.
