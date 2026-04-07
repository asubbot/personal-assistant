# Epic scope — EP-009 Dynamic Tool Creation with Docker Sandbox


| Field                  | Content                                                                                                                                                                                                                            |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ID**                 | EP-009                                                                                                                                                                                                                             |
| **Status**             | DONE                                                                                                                                                                                                                               |
| **Title**              | Dynamic Tool Creation with Docker Sandbox                                                                                                                                                                                          |
| **Description**        | Enable the LLM to create new tools at runtime by generating code (Python, Node.js, or shell), executing it in an isolated Docker sandbox on a configured node, and persisting the tool definition to the catalog for future reuse. |
| **First version date** | 2026-03-23                                                                                                                                                                                                                         |


## Glossary

- **Dynamic tool creation**: LLM-driven process where the assistant generates a new tool definition (id, template, arguments) and saves it to the catalog without operator intervention.
- **Docker sandbox**: Isolated execution environment on a node using Docker containers with resource limits (memory, CPU, timeout) and controlled network access.
- **pa-sandbox image**: Pre-built Docker images on the node containing language runtimes and common packages for executing generated code.
- **create_tool**: Native tool that validates and persists a new tool definition to the catalog (tools.yaml).
- **Template whitelist**: Security constraint allowing only specific command patterns (e.g., `docker run ...`) in dynamically created tool templates.

## Scope (features/capabilities)

- **Docker sandbox execution on node**: Run code in an isolated Docker container on any configured node with:
  - Network access (`--network bridge`) for API calls
  - Memory limits (e.g., 256MB)
  - CPU limits (e.g., 0.5 cores)
  - Execution timeout (e.g., 30 seconds)
- **Pre-built sandbox images**: Docker images maintained on the node:
  - `pa-sandbox:python` — Python 3.14 with requests, httpx, beautifulsoup4, lxml, json, re, datetime, math
  - `pa-sandbox:node` — Node.js 22 LTS with axios, node-fetch, cheerio
  - `pa-sandbox:base` — Minimal Alpine with curl, jq for simple HTTP/shell tasks
- **Native create_tool tool**: New tool registered in the tools registry that:
  - Accepts tool definition parameters (id, index_text, template, node_id, arguments, system_prompt)
  - Validates the template against whitelist (only `docker run` patterns allowed)
  - Checks for duplicate tool IDs
  - Appends the tool to tools.yaml
  - Adds the tool to the runtime catalog immediately
  - Returns success message to the LLM
- **Template security whitelist**: Only templates starting with allowed Docker commands are accepted:
  - `docker run --rm --network bridge ...`
  - `docker run --rm --network none ...`
- **Automatic tool reuse**: After creation, the new tool is immediately available for the current and future conversations through the standard tool catalog.
- **Tool catalog hot-reload**: Runtime catalog update without service restart (in-memory addition after file write).

## Success criteria

- LLM can create a new tool via `create_tool` call with valid parameters
- Created tool is persisted to tools.yaml and available for immediate use
- Template validation rejects non-whitelisted commands (e.g., direct shell commands, arbitrary paths)
- Docker sandbox executes Python and Node.js code with network access and resource limits
- Example scenario works: user asks "What's the weather in Barcelona?" → LLM creates `get_weather` tool → executes it → returns result
- Created tools persist across PA restarts (saved to tools.yaml)
- Unit tests cover create_tool validation logic
- Integration test covers end-to-end flow: create tool → execute → verify result

## Out of scope / deferred

- **Custom Docker image building at runtime**: Use pre-built images only; dynamic Dockerfile generation and `docker build` are not in scope.
- **Approval workflow**: User confirmation before tool creation is not required; tools are created and saved automatically.
- **Sandbox service (HTTP/gRPC)**: Uses direct Docker CLI execution via SSH; dedicated sandbox HTTP service is not in scope.
- **Code interpreter for other languages**: Only Python, Node.js, and shell (via base image) are supported initially.
- **Session-scoped tools**: All created tools are persisted; temporary/session-only tools are not in scope.

## Traceability

- **Scope:** Extends [scope.md](../../scope.md) Tool extensibility — from static catalog-only tools to LLM-driven dynamic creation while maintaining security model.
- **Strategy:** Aligns with [strategy.md](../../strategy.md) MVP evolution — extends capabilities without breaking core; testable with unit and integration tests.
- **Dependencies:**
  - Builds on [EP-004](../EP-004/ep-scope.md) (tool catalog, validation, execution path)
  - Uses [EP-001](../EP-001/ep-scope.md) node model for sandbox execution on any configured node

