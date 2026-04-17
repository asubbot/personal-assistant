# Epic scope — EP-023 Atomic catalog writes for create_tool

| Field | Content |
|-------|---------|
| **ID** | EP-023 |
| **Status** | DONE |
| **Title** | Atomic catalog writes for create_tool |
| **Description** | Make the native create_tool flow write the tool catalog atomically with post-write validation so a failed write never leaves the catalog half-valid, and the in-memory catalog plus tool vector index only advance after a validated write. |
| **First version date** | 2026-04-17 |

## Glossary

- **Catalog file**: the YAML file that holds declarative tool definitions read at startup and by create_tool at runtime.
- **Atomic replace**: write-to-temporary-file, fsync, then rename over the target path on the same filesystem.
- **In-memory catalog**: the parsed catalog structure held by the process and used to build tool definitions for the LLM.
- **Tool vector index**: the searchable index over catalog entries used to pre-select tools for a request.

## Scope (features/capabilities)

- create_tool writes the catalog through a temporary file in the same directory, fsync on the file and directory, then rename over the target path.
- After rename, the new file is re-parsed through the same loader used at startup; parse failure rolls back the in-memory catalog and leaves the file on disk consistent with in-memory state.
- The tool vector index is updated only after a validated write succeeds; failure modes do not leave it pointing at entries missing from the catalog.
- Failure paths are covered by deterministic tests that simulate short writes, rename failure, and invalid post-parse.

## Success criteria

- Simulated failure during write or rename leaves the catalog file byte-identical to its previous state and the in-memory catalog unchanged.
- After a successful create_tool the catalog file, the in-memory catalog, and the tool vector index agree on the new entry.
- Operator documentation explains the atomic replace contract and the fsync guarantees.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Extensibility and reliability focus in [scope.md](../../scope.md).
- **Strategy:** Test pyramid and success criteria in [strategy.md](../../strategy.md).
- **Related:** Recommendations §10.4 and risk R4 in [pa-architecture-review.md](../../pa-architecture-review.md).
