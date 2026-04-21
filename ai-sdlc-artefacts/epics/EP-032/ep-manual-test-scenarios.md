# EP-032 — Manual E2E test scenarios

This document defines operator-driven end-to-end scenarios for EP-032.

Related artefacts:

- [ep-scope.md](ep-scope.md)
- [ep-requirements.md](ep-requirements.md)
- [ep-acceptance-criteria.md](ep-acceptance-criteria.md)
- [ep-system-design.md](ep-system-design.md)

---

## Preconditions

- Application is running with EP-032 changes.
- Config contains `tools.vector_search_tools` for:
  - `search_vector_memory`
  - `search_vector_tool`
  - `search_vector_skill`
- Vector DB has non-empty `vec_tools` and `vec_skills`.
- Operator has access to:
  - Telegram chat with the assistant
  - application log (`pa.log`)

---

## Scenario M-32-01 — Tool knowledge retrieval happy path

**Trace:** AC-32.001, AC-32.011, AC-32.017

Given the assistant is online and tool knowledge index is ready  
When the operator asks in chat: "Which tool should I use for web search?"  
Then the assistant returns a grounded answer that references tool knowledge from `search_vector_tool` output  
And the answer does not invent non-existing tool ids.

---

## Scenario M-32-02 — Skill knowledge retrieval happy path

**Trace:** AC-32.002, AC-32.011, AC-32.017

Given the assistant is online and skill knowledge index is ready  
When the operator asks in chat: "Which skill should I use for semantic memory lookup?"  
Then the assistant returns a grounded answer that references skill knowledge from `search_vector_skill` output  
And the answer does not invent non-existing skill ids.

---

## Scenario M-32-03 — Memory retrieval remains memory-only

**Trace:** AC-32.003

Given EP-032 is enabled with specialized tools  
When the operator asks about prior personal context (not tool/skill metadata)  
Then the assistant keeps memory retrieval on `search_vector_memory` lanes (`notes`, `summaries`, `turns`)  
And does not mix tool/skill domain results into this retrieval path.

---

## Scenario M-32-04 — Unified limits applied for specialized tool

**Trace:** AC-32.004, AC-32.007, AC-32.008, AC-32.010, AC-32.011

Given `tools.vector_search_tools.search_vector_tool.max_top_k` is set to a low value (for example `2`)  
And assistant is restarted with updated config  
When the operator asks a broad tool-discovery question that usually yields many hits  
Then result size is bounded according to configured limits  
And behavior remains deterministic on repeated calls with same query and unchanged index data.

---

## Scenario M-32-05 — Per-tool disable switch

**Trace:** AC-32.008

Given `tools.vector_search_tools.search_vector_skill.enabled` is set to `false`  
And assistant is restarted with updated config  
When the operator asks a question that would normally use skill knowledge retrieval  
Then no successful invocation of `search_vector_skill` is observed in logs  
And assistant responds without process crash.

---

## Scenario M-32-06 — Fail-fast config validation

**Trace:** AC-32.005

Given config contains invalid values, for example `default_top_k > max_top_k` under `search_vector_tool`  
When the operator starts the assistant  
Then startup fails with deterministic field-specific validation error  
And process does not continue in partially valid state.

---

## Scenario M-32-07 — Redaction in invocation logs

**Trace:** AC-32.013

Given redaction policy is enabled  
When the operator sends a prompt containing sensitive-looking fragment (for example `secret-12345`) that triggers specialized vector tool call  
Then tool invocation log entry masks sensitive fragment according to configured redaction  
And raw secret fragment is not present in emitted log line.

---

## Scenario M-32-08 — Runtime skill allowlist compatibility

**Trace:** AC-32.014

Given runtime skills include `search_vector_tool` and `search_vector_skill` in frontmatter  
When assistant loads runtime skills at startup  
Then runtime skill validation accepts both tool ids  
And startup does not fail on unknown native tool reference.

---

## Operator checklist

- [ ] M-32-01 passed
- [ ] M-32-02 passed
- [ ] M-32-03 passed
- [ ] M-32-04 passed
- [ ] M-32-05 passed
- [ ] M-32-06 passed
- [ ] M-32-07 passed
- [ ] M-32-08 passed

If any scenario fails, record:

- scenario id
- observed behavior
- expected behavior
- relevant log excerpt
- config snapshot used for the run
