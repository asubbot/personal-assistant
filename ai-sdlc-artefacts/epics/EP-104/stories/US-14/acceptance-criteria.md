# Acceptance criteria — US-14

**Story:** [08-user-stories.md](../../08-user-stories.md#us-14--architecture-boundaries)

---

## AC-025 ([US-14](../../08-user-stories.md#us-14--architecture-boundaries))

**Given** the codebase, **When** an architect or developer reviews the module boundaries, **Then** ingestion adapters (e.g. Telegram), core, memory store, vector index, LLM abstraction, scheduler, and tools are clearly separated so that replacing or extending one part does not require a full redesign.
