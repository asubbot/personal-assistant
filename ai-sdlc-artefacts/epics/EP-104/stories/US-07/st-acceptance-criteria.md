# Acceptance criteria — US-07

**Story:** [08-user-stories.md](../../08-user-stories.md#us-07--vector-search)

---

## AC-013 ([US-07](../../08-user-stories.md#us-07--vector-search))

**Given** content in the long-term memory store, **When** the store is indexed, **Then** a vector index is maintained for that content.

---

## AC-014 ([US-07](../../08-user-stories.md#us-07--vector-search))

**Given** a user query, **When** semantic search is performed, **Then** the system returns relevant context from the index (e.g. top-k or threshold-based).

---

## AC-037 ([US-07](../../08-user-stories.md#us-07--vector-search))

**Given** the embedding provider returns an error or invalid response (e.g. 4xx, empty data, invalid JSON, context canceled, unreachable server), **When** the core uses the embedder, **Then** the system handles the error and does not crash.
