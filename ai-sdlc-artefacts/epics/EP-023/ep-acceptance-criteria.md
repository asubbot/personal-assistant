# EP-023 — Acceptance criteria

## Introduction

This document lists testable acceptance criteria for atomic catalog persistence in `create_tool`, traceable to [ep-requirements.md](ep-requirements.md). Criteria use Gherkin-style Given / When / Then wording and IDs **AC-23.NNN** (epic 23).

---

## Acceptance criteria index

| AC ID | REQ (trace) | Summary |
|-------|---------------|---------|
| [AC-23.001](#ac-23-001) | [REQ-23.001](ep-requirements.md#catalog-file-durability) | Atomic same-directory replace on success |
| [AC-23.002](#ac-23-002) | [REQ-23.002](ep-requirements.md#catalog-file-durability) | File and directory sync participate in persist path |
| [AC-23.003](#ac-23-003) | [REQ-23.003](ep-requirements.md#catalog-file-durability) | Post-rename `toolcatalog.Load` runs on success |
| [AC-23.004](#ac-23-004) | [REQ-23.004](ep-requirements.md#catalog-file-durability) | Failed post-write validation restores bytes and skips memory add |
| [AC-23.005](#ac-23-005) | [REQ-23.005](ep-requirements.md#runtime-catalog-and-tool-index-consistency) | In-memory catalog gains tool only after validated persist |
| [AC-23.006](#ac-23-006) | [REQ-23.006](ep-requirements.md#runtime-catalog-and-tool-index-consistency) | Index upsert runs only after memory contains new tool |
| [AC-23.007](#ac-23-007) | [REQ-23.007](ep-requirements.md#runtime-catalog-and-tool-index-consistency) | Embed failure rolls back file and memory when embedder configured |
| [AC-23.008](#ac-23-008) | [REQ-23.008](ep-requirements.md#verification-and-operator-documentation) | Tests cover short write, rename failure, invalid post-write |
| [AC-23.009](#ac-23-009) | [REQ-23.009](ep-requirements.md#verification-and-operator-documentation) | Operator doc section for replace and sync |
| [AC-23.010](#ac-23-010) | [REQ-23.011](ep-requirements.md#verification-and-operator-documentation) | `make check` passes |

---

## Acceptance criteria

### AC-23.001

**Trace:** [REQ-23.001](ep-requirements.md#catalog-file-durability)

Given an existing valid Catalog file and a valid new tool definition  
When `create_tool` completes successfully  
Then the Catalog file is replaced via a same-directory temporary file and a single rename onto the catalog path.

### AC-23.002

**Trace:** [REQ-23.002](ep-requirements.md#catalog-file-durability)

Given the catalog persistence implementation under test  
When a catalog update is written durably per EP-023  
Then the implementation calls `Sync` on the catalog data file and on the parent directory handle as part of the documented sequence.

### AC-23.003

**Trace:** [REQ-23.003](ep-requirements.md#catalog-file-durability)

Given a successful atomic replace of the Catalog file  
When validation runs  
Then the process invokes `toolcatalog.Load` with the catalog path before marking the write successful.

### AC-23.004

**Trace:** [REQ-23.004](ep-requirements.md#catalog-file-durability)

Given a snapshot of the Catalog file taken before an append attempt  
When post-write validation fails after replace  
Then the Catalog file bytes match the snapshot and the new tool id is absent from the In-memory catalog.

### AC-23.005

**Trace:** [REQ-23.005](ep-requirements.md#runtime-catalog-and-tool-index-consistency)

Given a successful `create_tool` invocation  
When the caller observes the In-memory catalog  
Then the new tool appears only if `toolcatalog.Load` succeeded after the replace.

### AC-23.006

**Trace:** [REQ-23.006](ep-requirements.md#runtime-catalog-and-tool-index-consistency)

Given an embedder and tool index configured for `create_tool`  
When the tool vector index is updated  
Then the upsert runs after the In-memory catalog contains the new tool entry.

### AC-23.007

**Trace:** [REQ-23.007](ep-requirements.md#runtime-catalog-and-tool-index-consistency)

Given an embedder and tool index configured and embedding upsert returns an error  
When `create_tool` returns  
Then the error is non-nil, the Catalog file bytes match the pre-call snapshot, and the new tool id is absent from the In-memory catalog.

### AC-23.008

**Trace:** [REQ-23.008](ep-requirements.md#verification-and-operator-documentation)

Given automated tests for catalog persistence  
When they run  
Then they include deterministic cases for short write, rename failure during replace, and invalid catalog bytes after replace that assert restore behaviour.

### AC-23.009

**Trace:** [REQ-23.009](ep-requirements.md#verification-and-operator-documentation)

Given the repository operator documentation  
When an operator reads the tool catalog section  
Then the documentation explains atomic replace, file and directory sync, and post-write validation.

### AC-23.010

**Trace:** [REQ-23.011](ep-requirements.md#verification-and-operator-documentation)

Given the epic change set merged locally  
When `make check` runs from the repository root  
Then it completes with exit code zero.
