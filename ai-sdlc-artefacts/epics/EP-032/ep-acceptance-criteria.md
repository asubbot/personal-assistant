# EP-032 — Specialized Knowledge Search Tools — Acceptance criteria

This document defines acceptance criteria for [ep-scope.md](ep-scope.md), traced to [ep-requirements.md](ep-requirements.md).

---

## Acceptance criteria index

| AC | REQ (trace) | Summary |
|----|-------------|---------|
| [AC-32.001](#ac-32-001) | REQ-32.001 | Register native tool id `search_vector_tool` |
| [AC-32.002](#ac-32-002) | REQ-32.002 | Register native tool id `search_vector_skill` |
| [AC-32.003](#ac-32-003) | REQ-32.003 | Keep `search_vector_memory` domain unchanged |
| [AC-32.004](#ac-32-004) | REQ-32.004 | Unified config block exists for all vector-search tools |
| [AC-32.005](#ac-32-005) | REQ-32.005 | Invalid unified config is rejected at load |
| [AC-32.006](#ac-32-006) | REQ-32.006 | `search_vector_memory` reads limits from unified config |
| [AC-32.007](#ac-32-007) | REQ-32.007 | `search_vector_tool` reads limits from unified config |
| [AC-32.008](#ac-32-008) | REQ-32.008 | `search_vector_skill` reads limits from unified config |
| [AC-32.009](#ac-32-009) | REQ-32.009 | Reject empty query for specialized tools |
| [AC-32.010](#ac-32-010) | REQ-32.010 | Enforce top_k bounds and deterministic order |
| [AC-32.011](#ac-32-011) | REQ-32.011 | Return compact bounded snippets with identifiers |
| [AC-32.012](#ac-32-012) | REQ-32.012 | Specialized retrieval is read-only |
| [AC-32.013](#ac-32-013) | REQ-32.013 | Invocation logs include metadata with redaction |
| [AC-32.014](#ac-32-014) | REQ-32.014 | Runtime skills allow new tool IDs |
| [AC-32.015](#ac-32-015) | REQ-32.015 | `make check` passes |
| [AC-32.016](#ac-32-016) | REQ-32.016 | `make validate` passes |
| [AC-32.017](#ac-32-017) | REQ-32.017 | E2E specialized retrieval flow passes |

---

## Acceptance criteria

### AC-32.001

**AC-32.001** (Trace: [REQ-32.001](ep-requirements.md#tool-contract))

Given tool-index store and embedding provider are available  
When native tools are registered for conversation flow  
Then tool registry SHALL contain id `search_vector_tool`.

---

### AC-32.002

**AC-32.002** (Trace: [REQ-32.002](ep-requirements.md#tool-contract))

Given skill-index store and embedding provider are available  
When native tools are registered for conversation flow  
Then tool registry SHALL contain id `search_vector_skill`.

---

### AC-32.003

**AC-32.003** (Trace: [REQ-32.003](ep-requirements.md#tool-contract))

Given EP-032 changes are applied  
When `search_vector_memory` executes  
Then supported lanes SHALL remain limited to `notes`, `summaries`, and `turns`.

---

### AC-32.004

**AC-32.004** (Trace: [REQ-32.004](ep-requirements.md#unified-config-block))

Given config JSON includes `tools.vector_search_tools`  
When config is loaded  
Then runtime settings for `search_vector_memory`, `search_vector_tool`, and `search_vector_skill` SHALL be read from that single block.

---

### AC-32.005

**AC-32.005** (Trace: [REQ-32.005](ep-requirements.md#unified-config-block))

Given `tools.vector_search_tools` contains invalid bounds (`default_top_k > max_top_k` or non-positive limits)  
When config loading validates the block  
Then load SHALL fail with deterministic field-level error.

---

### AC-32.006

**AC-32.006** (Trace: [REQ-32.006](ep-requirements.md#unified-config-block))

Given `tools.vector_search_tools.search_vector_memory` defines custom limits  
When `search_vector_memory` is registered and invoked  
Then runtime behavior SHALL use configured limits instead of hardcoded values.

---

### AC-32.007

**AC-32.007** (Trace: [REQ-32.007](ep-requirements.md#unified-config-block))

Given `tools.vector_search_tools.search_vector_tool` defines custom limits  
When `search_vector_tool` runs  
Then query validation, top_k bounds, and output budget SHALL follow configured values.

---

### AC-32.008

**AC-32.008** (Trace: [REQ-32.008](ep-requirements.md#unified-config-block))

Given `tools.vector_search_tools.search_vector_skill` defines custom limits  
When `search_vector_skill` runs  
Then query validation, top_k bounds, and output budget SHALL follow configured values.

---

### AC-32.009

**AC-32.009** (Trace: [REQ-32.009](ep-requirements.md#retrieval-behavior))

Given `search_vector_tool` or `search_vector_skill` receives missing or whitespace-only `query`  
When the tool validates input  
Then tool SHALL return deterministic validation error and SHALL NOT run embedding or vector search.

---

### AC-32.010

**AC-32.010** (Trace: [REQ-32.010](ep-requirements.md#retrieval-behavior))

Given valid request and repeated calls with same query and index data  
When specialized tool returns hits  
Then result order SHALL be deterministic by score with stable tie-breaking by source id.

---

### AC-32.011

**AC-32.011** (Trace: [REQ-32.011](ep-requirements.md#retrieval-behavior))

Given successful specialized retrieval call  
When results are formatted  
Then each line SHALL include source identifier and compact snippet, and result payload SHALL remain within configured output budget.

---

### AC-32.012

**AC-32.012** (Trace: [REQ-32.012](ep-requirements.md#safety-and-observability))

Given successful `search_vector_tool` or `search_vector_skill` call  
When retrieval completes  
Then no write operation SHALL be executed for memory markdown files or vector rows.

---

### AC-32.013

**AC-32.013** (Trace: [REQ-32.013](ep-requirements.md#safety-and-observability))

Given specialized vector tool is invoked in tool-calling loop  
When invocation logging is emitted  
Then logs SHALL include tool id and invocation metadata and SHALL apply existing sensitive-value redaction.

---

### AC-32.014

**AC-32.014** (Trace: [REQ-32.014](ep-requirements.md#runtime-skills-integration))

Given runtime skill package references `search_vector_tool` or `search_vector_skill`  
When runtime skill validation loads package  
Then validation SHALL accept both tool ids as allowed native references.

---

### AC-32.015

**AC-32.015** (Trace: [REQ-32.015](ep-requirements.md#verification))

Given EP-032 changes are present on clean working tree  
When `make check` runs  
Then command SHALL exit with status `0`.

---

### AC-32.016

**AC-32.016** (Trace: [REQ-32.016](ep-requirements.md#verification))

Given EP-032 changes are present on clean working tree  
When `make validate` runs without parameters  
Then command SHALL exit with status `0`.

---

### AC-32.017

**AC-32.017** (Trace: [REQ-32.017](ep-requirements.md#verification))

Given prior indexed tool and skill knowledge exists  
When user asks a tool-focused or skill-focused question and model calls corresponding specialized vector tool  
Then assistant SHALL produce grounded final answer in the same turn using bounded snippets from that tool.
