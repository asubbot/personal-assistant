# EP-009 — Manual verification (operator / environment)

Some acceptance criteria depend on **operator-built Docker images** and a **reachable SSH test node** with Docker. They are not fully asserted in CI unless `EP009_SSH_DOCKER=1` (or similar) and images are present.

| AC | Manual scenario |
|----|-----------------|
| AC-09.005–AC-09.007 | Build/tag `pa-sandbox` Python, Node, and Alpine images per [ep-scope.md](ep-scope.md); verify images exist on the target node (`docker image ls`). |
| AC-09.001–AC-09.004, AC-09.014, AC-09.018 | Run integration checks on a node with allowlisted `docker run` lines; measure sandbox start time where applicable; verify `network none` blocks outbound (see [ep-implementation-plan.md](ep-implementation-plan.md) §7). |

Automated unit/integration coverage references: test comments `Covers AC-09.xxx` / `Supporting AC-09.xxx` in `internal/tools`, `internal/toolcatalog`, `internal/config`, `internal/core`, and `tests/integration/` (when enabled).
