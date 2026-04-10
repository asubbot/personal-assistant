---
name: Memory retrieval
description: When the user asks about past calendar dates or long-term memory, call read_memory with ISO dates in pa_timezone. Prefer a single date or a short from–to range; clarify ambiguous relative phrases.
tools: ["read_memory"]
---

## Policy

- Use **`read_memory`** with **`date`** (YYYY-MM-DD) for a single day, or **`from`** and **`to`** (inclusive) for a bounded range.
- Dates are interpreted in the assistant **`pa_timezone`** from config (same as automatic summarization).
- Do not invent paths; only structured ISO arguments are allowed.
- For vague requests (“last week”), ask the user for concrete dates or offer a small range within configured limits.
