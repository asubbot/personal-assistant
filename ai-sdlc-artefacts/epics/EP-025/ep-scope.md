# Epic scope — EP-025 Test layout cleanup: E2E separation

| Field | Content |
|-------|---------|
| **ID** | EP-025 |
| **Status** | DONE |
| **Title** | Test layout cleanup: E2E separation |
| **Description** | Move end-to-end tests out of the main binary package into a dedicated end-to-end directory, gate them with a build tag, and split the make targets so unit and end-to-end coverage are reported separately. |
| **First version date** | 2026-04-17 |

## Glossary

- **End-to-end test**: a test that exercises the full binary path (configuration load, adapter wiring, memory and vector stores) in a controlled environment.
- **Build tag gate**: a Go build tag that excludes a test file from default builds.
- **Coverage scope**: the set of packages reported by the coverage run.

## Scope (features/capabilities)

- End-to-end test files currently rooted in the main binary package move under the existing end-to-end test directory.
- A build tag excludes end-to-end tests from the default unit test run; a dedicated make target runs them under the tag.
- Coverage configuration keeps unit coverage as the default metric and exposes end-to-end coverage under a separate target or suffix.
- CI is updated so unit and end-to-end runs are distinguishable in the pipeline output.

## Success criteria

- The default test target runs without touching the moved end-to-end tests.
- Running the dedicated end-to-end target exercises the moved tests under the build tag.
- Coverage output separates unit and end-to-end contributions.
- Full quality gate passes on the change set.

## Traceability

- **Scope:** Reliability focus and test pyramid in [scope.md](../../scope.md).
- **Strategy:** Test pyramid and levels in [strategy.md](../../strategy.md) §2.2.
- **Related:** Recommendations §10.8 in [pa-architecture-review.md](../../pa-architecture-review.md).
