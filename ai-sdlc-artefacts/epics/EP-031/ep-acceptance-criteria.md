# EP-031 — Vector Memory Search Tool — Acceptance criteria

This document defines acceptance criteria for [ep-scope.md](ep-scope.md), traced to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC | REQ (trace) | Summary |
|----|-------------|---------|
| [AC-31.001](#ac-31-001) | REQ-31.001 | Register native tool id `search_vector_memory` |
| [AC-31.002](#ac-31-002) | REQ-31.002 | Reject empty or missing query |
| [AC-31.003](#ac-31-003) | REQ-31.003 | Omitted lanes means search all lanes |
| [AC-31.004](#ac-31-004) | REQ-31.004 | Reject unknown lane value |
| [AC-31.005](#ac-31-005) | REQ-31.005 | Enforce top_k and deterministic result order |
| [AC-31.006](#ac-31-006) | REQ-31.006 | Return compact snippets with source and lane |
| [AC-31.007](#ac-31-007) | REQ-31.007 | Retrieval is read-only and has no write side effects |
| [AC-31.008](#ac-31-008) | REQ-31.008 | Runtime skill validation allows tool reference |
| [AC-31.009](#ac-31-009) | REQ-31.009 | Structured invocation logs with redaction |
| [AC-31.010](#ac-31-010) | REQ-31.010 | Tool retrieval works when auto-RAG lanes are zero |
| [AC-31.011](#ac-31-011) | REQ-31.011 | `make check` passes |
| [AC-31.012](#ac-31-012) | REQ-31.012 | `make validate` passes |
| [AC-31.013](#ac-31-013) | REQ-31.013 | E2E on-demand retrieval scenario passes |

---

## Acceptance criteria

### AC-31.001

**AC-31.001** (Trace: [REQ-31.001](ep-requirements.md#tool-contract))

Given runtime dependencies for vector memory retrieval are available  
When native tools are registered for the main conversation path  
Then the registry SHALL contain tool id `search_vector_memory`.

---

### AC-31.002

**AC-31.002** (Trace: [REQ-31.002](ep-requirements.md#tool-contract))

Given a tool call to `search_vector_memory` with `query` missing or only whitespace  
When the tool validates input  
Then the tool SHALL return a deterministic validation error and SHALL NOT perform embedding or vector search.

---

### AC-31.003

**AC-31.003** (Trace: [REQ-31.003](ep-requirements.md#retrieval-lanes))

Given a tool call to `search_vector_memory` with `query` and no `lanes` field  
When the tool executes retrieval  
Then the tool SHALL search all configured lanes (`notes`, `summaries`, `turns`) that are available at runtime.

---

### AC-31.004

**AC-31.004** (Trace: [REQ-31.004](ep-requirements.md#retrieval-lanes))

Given a tool call to `search_vector_memory` that includes lane `foo`  
When the tool validates `lanes`  
Then the tool SHALL reject the request with an error that names lane `foo`.

---

### AC-31.005

**AC-31.005** (Trace: [REQ-31.005](ep-requirements.md#limits-and-output-shaping))

Given a valid query and `top_k` larger than configured maximum  
When `search_vector_memory` validates arguments  
Then the request SHALL be rejected with a deterministic bounds error.

And given a valid query and valid `top_k`  
When the tool returns matches  
Then match ordering SHALL be deterministic according to documented lane-and-score policy.

---

### AC-31.006

**AC-31.006** (Trace: [REQ-31.006](ep-requirements.md#limits-and-output-shaping))

Given a successful `search_vector_memory` call  
When the tool formats output  
Then each returned item SHALL include lane label and source identifier, and output SHALL be compact enough for tool-result prompt usage.

---

### AC-31.007

**AC-31.007** (Trace: [REQ-31.007](ep-requirements.md#safety-and-integration))

Given a successful `search_vector_memory` call  
When retrieval completes  
Then no write operation SHALL be performed against memory markdown files or vector rows.

---

### AC-31.008

**AC-31.008** (Trace: [REQ-31.008](ep-requirements.md#safety-and-integration))

Given a runtime skill package that references `search_vector_memory` in frontmatter  
When runtime skill validation loads the package  
Then validation SHALL accept the tool reference as allowed native tool id.

---

### AC-31.009

**AC-31.009** (Trace: [REQ-31.009](ep-requirements.md#safety-and-integration))

Given `search_vector_memory` is invoked from the tool-calling loop  
When invocation logging is emitted  
Then logs SHALL contain tool id and invocation metadata and SHALL apply existing redaction policy to sensitive fragments.

---

### AC-31.010

**AC-31.010** (Trace: [REQ-31.010](ep-requirements.md#retrieval-policy))

Given configuration sets `conversation_context.memory_vector.notes_top_k`, `summaries_top_k`, and `turns_top_k` to `0`  
When the model invokes `search_vector_memory`  
Then semantic retrieval SHALL still return relevant bounded snippets for matching data.

---

### AC-31.011

**AC-31.011** (Trace: [REQ-31.011](ep-requirements.md#verification))

Given EP-031 changes are present on a clean working tree  
When `make check` runs  
Then the command SHALL exit with status `0`.

---

### AC-31.012

**AC-31.012** (Trace: [REQ-31.012](ep-requirements.md#verification))

Given EP-031 changes are present on a clean working tree  
When `make validate` runs without parameters  
Then the command SHALL exit with status `0`.

---

### AC-31.013

**AC-31.013** (Trace: [REQ-31.013](ep-requirements.md#verification))

Given memory contains semantically relevant prior notes or summaries and auto-RAG lane top_k values are zero  
When a user asks a question that needs that prior context and the model calls `search_vector_memory`  
Then the tool SHALL return bounded relevant snippets and the assistant SHALL produce a grounded final answer in the same turn.
