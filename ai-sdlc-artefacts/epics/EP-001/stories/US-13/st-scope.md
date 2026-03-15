# Story scope — US-13 Add nodes/tools without rebuild

**Story:** US-13  
**Title:** Add nodes and tools without image rebuild

---

## Formulation

As an operator, I want to add new nodes and register new tools through configuration, so that I can scale without rebuilding the core image.

## Traceability to AC/REQ

| AC | REQ | Summary |
|----|-----|---------|
| [AC-024](../../ep-acceptance-criteria.md#ac-024) | [REQ-011](../../ep-requirements.md#extensibility-and-architecture) | New node/tool via config → load after restart/hot-reload without rebuild |
