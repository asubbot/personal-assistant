# Observability HTTP — health and readiness (EP-029)

The main `pa` binary can expose a small **HTTP** listener for operators (Kubernetes/Docker health probes, load balancers, or manual `curl`). The listener is **opt-in**: it binds only when **`observability_http`** is a JSON **object** with every field set. Use JSON **`null`** for **`observability_http`** when no HTTP server should run (the top-level key must still be present; omitting it fails config load). See [configuration.md](configuration.md) for the exhaustive root-key rule.

## Configuration

All fields are **required** when `observability_http` is set (the loader rejects missing values, identical paths, or paths not starting with `/`):

| Field | Meaning |
|-------|---------|
| `listen_address` | TCP address for `Listen` (e.g. `127.0.0.1:9090`). Bind only to interfaces you intend to expose. |
| `health_path` | Path for **liveness** (process is up and serving this HTTP stack). Example: `/health`. |
| `readiness_path` | Path for **readiness** (aggregated subsystem state). Example: `/ready`. |
| `probe_llm` | When `true`, readiness includes a **short** completion call to the **first** `llm_providers` entry (bounded timeout). When `false`, readiness treats LLM as ready once providers are loaded without an outbound probe. |

Example:

```json
  "observability_http": {
    "listen_address": "127.0.0.1:9090",
    "health_path": "/health",
    "readiness_path": "/ready",
    "probe_llm": false
  }
```

## Endpoints

- **`GET {health_path}`** — Always **200** when the listener is running. JSON body includes `"status":"alive"`. Suitable for Docker **`HEALTHCHECK`** liveness: if the process is wedged but the HTTP stack still answers, you still get 200; combine with process supervision as needed.
- **`GET {readiness_path}`** — **200** when all checks pass; **503** with JSON listing checks when something is still initializing or failed. Checks typically cover LLM configuration (and optional probe), memory vector tables, tool vector index build completion, optional scheduled jobs runtime, and the memory summarization worker when that subsystem is configured to run.

## Bind failures

If `listen_address` is invalid, already in use, or the process lacks permission to bind, the observability server logs an error and **the Telegram bot continues to run**. Health and readiness URLs will not respond until the address is fixed and the process is restarted. This is intentional so a misconfigured sidecar port does not take down messaging.

## Duplicate log lines

Some subsystems emit both a **legacy** operational line (for example `memory job failed`) and a **lifecycle** record with the same underlying error. Operators migrating to structured queries should prefer the `lifecycle_event=true` records.

## Docker HEALTHCHECK example

The health endpoint is intentionally minimal so operators can probe liveness without credentials:

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9090/health || exit 1
```

Adjust host, port, and path to match your `listen_address` and `health_path`. Readiness is usually polled by an orchestrator hook or manually during rollout (`curl -sf http://127.0.0.1:9090/ready`).

## Lifecycle log schema

Background subsystems emit structured records tagged for operators and log pipelines:

| Attribute | When set | Meaning |
|-----------|----------|---------|
| `lifecycle_event` | always | `true` for lifecycle-tagged records. |
| `subsystem` | always | Logical owner, e.g. `memory_job`, `jobs_runtime`, `tool_index`. |
| `lifecycle_phase` | always | Boundary name, e.g. `job_start`, `job_complete`, `init`, `build`. |
| `duration_ms` | when applicable | Wall time for the completed phase in milliseconds. |
| `job` | memory jobs | Memory worker job name. |
| `tool_count` | tool index | Number of catalog tools indexed on success. |

Other attributes (for example `error`, `jobs_loaded`) may appear on specific messages. Treat logs as sensitive in production; see [configuration.md](configuration.md) and [llm-provider-roles-and-logging.md](llm-provider-roles-and-logging.md).
