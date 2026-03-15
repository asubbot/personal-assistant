# Story scope — US-07 Vector search

**Story:** US-07  
**Title:** Vector index and semantic search

---

## Formulation

As the assistant (system), I want to index long-term memory in a vector store and run semantic search, so that relevant context can be retrieved for replies.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-013](../../ep-acceptance-criteria.md#ac-013) | [REQ-007](../../ep-requirements.md#memory-and-indexing) | Memory store indexed → vector index maintained |
| [AC-014](../../ep-acceptance-criteria.md#ac-014) | [REQ-007](../../ep-requirements.md#memory-and-indexing) | Semantic search → relevant context from index returned |
| [AC-037](../../ep-acceptance-criteria.md#ac-037) | [REQ-025](../../ep-requirements.md#llm-and-logging) | Embedding provider error → handled, no crash |
