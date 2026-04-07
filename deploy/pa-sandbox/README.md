# PA sandbox images (`pa-sandbox:*`)

Reference **operator-built** Docker images for EP-009 dynamic tools. PersonalAssistant does not build these at runtime; the bot runs workloads on a configured node via `docker run` over SSH.

**Requirements (packages and runtimes):** [ep-requirements.md](../../ai-sdlc-artefacts/epics/EP-009/ep-requirements.md) (REQ-09.005–09.007), [ep-scope.md](../../ai-sdlc-artefacts/epics/EP-009/ep-scope.md).

**Manual end-to-end:** [ep-manual-tests.md](../../ai-sdlc-artefacts/epics/EP-009/ep-manual-tests.md).

## Images

| Tag | Purpose |
|-----|---------|
| `pa-sandbox:python` | Python sandbox (stdlib + listed pip packages). |
| `pa-sandbox:node` | Node.js 22 LTS + npm packages for HTTP/HTML. |
| `pa-sandbox:base` | Minimal Alpine: `curl`, `jq`, shell. |

Expected names match tool templates (e.g. `docker run ... pa-sandbox:base ...`). You may use a registry prefix (e.g. `registry.example/pa-sandbox:python`) if templates reference the same image name.

## Build (from this directory)

```bash
docker build -f Dockerfile.python -t pa-sandbox:python .
docker build -f Dockerfile.node   -t pa-sandbox:node   .
docker build -f Dockerfile.base   -t pa-sandbox:base   .
```

## Deploy to the node

- Push to a private registry and `docker pull` on the node, or  
- `docker save … | ssh user@node docker load` (airgap).

On the node: `docker image ls | grep pa-sandbox`.

## Allowlist

Each new `docker run …` shape must be allowed in the node **command allowlist** (`command_allowlist_path` in config). PA does not rewrite templates. See [docs/configuration.md](../../docs/configuration.md).

## Python base image version

REQ-09.005 specifies **Python 3.14**. If `python:3.14-slim` is unavailable on your registry, pin a nearby tag (e.g. `3.13-slim`) until 3.14 is published, and align with your compliance policy.

Build steps install OS/pip/npm packages as **root** where required (standard for Dockerfiles). All three images run the default process **non-root**: **`app` (UID 1000)** in `base` and `python`, built-in **`node` (UID 1000)** in `node`. `PIP_ROOT_USER_ACTION=ignore` is set in the Python Dockerfile so pip does not print the “running as root” warning during the build-time `pip install`.

## Security

- Treat these images as **trusted supply chain**: pin digests or tags in production.  
- Sandboxes still rely on Docker limits (`--memory`, `--cpus`, timeout) and network mode from the **tool template** (`bridge` vs `none`).
- Prefer **non-root** default users in images where feasible; override with `docker run --user …` only when required.
