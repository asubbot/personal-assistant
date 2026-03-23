# EP-009 — Manual verification (operator / environment)

Some acceptance criteria depend on **operator-built Docker images** and a **reachable SSH node** with Docker Engine. They are not fully asserted in CI unless integration tests are enabled and images are present on the target node.

**Related:** [ep-scope.md](ep-scope.md), [ep-implementation-plan.md](ep-implementation-plan.md) §7, [docs/configuration.md](../../../docs/configuration.md).

---

## AC ↔ manual coverage (summary)

| AC | What to verify manually |
|----|-------------------------|
| AC-09.005–AC-09.007 | `pa-sandbox:python`, `pa-sandbox:node`, `pa-sandbox:base` exist on the node (`docker image ls`). |
| AC-09.001–AC-09.004, AC-09.014 | Remote command after substitution contains expected `docker run` flags (network bridge, memory, CPU, timeout); optional timing for 5s startup with warm cache. |
| AC-09.018 | Template with `--network none`: outbound to the public internet from inside the container fails as expected. |
| AC-09.008–AC-09.013, AC-09.017 | Covered primarily by automated tests; manual path below confirms end-to-end operator setup. |

Automated references: comments `Covers AC-09.xxx` / `Supporting AC-09.xxx` in `internal/tools`, `internal/toolcatalog`, `internal/config`, `internal/core`, and `tests/integration/` (when enabled).

---

## Happy path — end-to-end (operator)

Goal: from **empty sandbox images on a build host** to **one new catalog tool created via `create_tool` and executed once** on a configured node.

**Assumptions:**

- One Linux node with Docker, SSH access, and a dedicated user matching `config.json` `nodes.<id>`.
- PersonalAssistant (`pa`) runs where it can reach the Telegram/LLM APIs and load `config.json` + `tools.yaml`.
- Image tags and package versions follow [ep-scope.md](ep-scope.md) (Python 3.14 stack, Node 22, Alpine base).

**Reference Dockerfiles and build instructions** live in **[deploy/pa-sandbox/README.md](../../../deploy/pa-sandbox/README.md)** (this repository). Operators may fork or replace them to match policy.

The steps below remain a **procedure**; image recipes are maintained alongside that folder.

### Phase 1 — Build sandbox images (build host)

1. **Choose a build machine** with Docker (`docker buildx` or classic `docker build`) and network access to pull base images (e.g. `python:3.14-slim`, `node:22-alpine`, `alpine:3.x`).

2. **Create three images** tagged consistently, for example:
   - `pa-sandbox:python` — install runtime deps from [ep-scope.md](ep-scope.md) (e.g. `requests`, `httpx`, `beautifulsoup4`, `lxml`).
   - `pa-sandbox:node` — install `axios`, `node-fetch`, `cheerio` (or lockfile equivalent).
   - `pa-sandbox:base` — install `curl`, `jq` on Alpine.

3. **Build and tag locally** (illustrative; adjust Dockerfiles to your policy):

   ```bash
   docker build -t pa-sandbox:python -f Dockerfile.python .
   docker build -t pa-sandbox:node   -f Dockerfile.node   .
   docker build -t pa-sandbox:base   -f Dockerfile.base   .
   ```

4. **Sanity-check images** on the build host:

   ```bash
   docker run --rm pa-sandbox:python python -c "import json, sys; print(sys.version)"
   docker run --rm pa-sandbox:node   node -e "console.log(process.version)"
   docker run --rm pa-sandbox:base   sh -c 'curl --version && jq --version'
   ```

### Phase 2 — Deploy images to the target node

5. **Transfer images** to the node (pick one):
   - **Registry:** push to your private registry from the build host, then on the node `docker pull <registry>/pa-sandbox:{python,node,base}` and tag as `pa-sandbox:*` if needed.
   - **Tar / airgap:** `docker save pa-sandbox:python ... | ssh user@node docker load`.

6. **On the node**, confirm tags:

   ```bash
   docker image ls | grep pa-sandbox
   ```

   You should see `pa-sandbox` with tags `python`, `node`, `base` (or your chosen naming; templates must reference the same names).

### Phase 3 — Node allowlist (required for execution)

7. **Extend the node allowlist** (`command_allowlist_path` in config) so the **exact** `docker run` prefix and arguments you will use are allowed. PA does not rewrite templates; the substituted command must match an allowlist line (prefix wildcard `*` only as documented in [docs/configuration.md](../../../docs/configuration.md)).

8. **Minimal allowlist pattern example** (one line per logical command shape; adapt to your paths and image tags):

   - Allow a line that matches your sandbox `docker run` after substitution, e.g. a prefix ending with `*` if policy permits, or the full expected string for tests.

9. **Re-load / ensure** the allowlist file is the one referenced by `nodes.<id>.command_allowlist_path` (resolved relative to `PA_CONFIG_DIR` if not absolute).

### Phase 4 — PA configuration and catalog

10. **Set `paths.tool_catalog_path`** to your `tools.yaml` (under `PA_CONFIG_DIR` or absolute).

11. **Configure `nodes.<node_id>`** (host, port, `dedicated_user`, `auth.private_key_path`, `command_allowlist_path`) and **`paths.ssh_known_hosts_path`**.

12. **Optional — `tools.create_tool_secret_patterns` (REQ-09.017):** operator-defined extra blocking rules for **`create_tool`** before anything is written to `tools.yaml`. Omit the field or use `[]` if you rely only on built-in validation (template whitelist, duplicate id, etc.).

    - **Where:** nested under **`tools`** in `config.json` (same `tools` object as `text_based_enabled` / `llm_escalation`).
    - **Format:** JSON array of **Go `regexp` strings** (RE2). **Invalid regex** → config **fails to load** at startup (fail fast). **Empty string** entries are rejected at load.
    - **What is scanned:** before persist, the product concatenates the proposed tool’s fields into one string and runs each pattern against it:
      - Lines (in order): `id`, `index_text`, `template`, `system_prompt` (each separated by `\n`); if `arguments` is present, a JSON serialization of the argument rules is appended after a newline.
      - If **any** pattern matches, `create_tool` returns an error and **does not** update `tools.yaml` (message like *possible secret content*).
    - **Example** (illustrative patterns — tune for your environment; avoid over-broad regexes that block legitimate tool text):

      ```json
      "tools": {
        "text_based_enabled": false,
        "create_tool_secret_patterns": [
          "api[_-]?key\\s*[:=]",
          "BEGIN (RSA |OPENSSH )?PRIVATE KEY"
        ]
      }
      ```

    - **Manual follow-up:** with patterns configured, run **§24** (secret pattern rejection) in Phase 7.

13. **Start from a valid `tools.yaml`** (at least empty `tools: []` or existing tools). Ensure the catalog loads at startup (PA fails fast on parse errors).

### Phase 5 — Start PersonalAssistant

14. **Export env** as needed: `PA_CONFIG_DIR`, `PA_DATA_DIR`, `PA_SECRETS_DIR`, `PA_LOG_LEVEL`.

15. **Run `pa`** (or your process manager). Confirm in logs: config loaded, tool catalog loaded, no startup error.

16. **Optional:** `pa -verify-nodes` with an allowlisted command to confirm SSH + allowlist (not a substitute for `docker run`, but validates connectivity).

### Phase 6 — Happy path: `create_tool` then execute (LLM or direct test)

17. **Compose a valid template** for the first dynamic tool. It **must**:
    - Start with `docker run --rm --network bridge` **or** `docker run --rm --network none`.
    - (Recommended) Add `--memory=256m` and `--cpus=0.5` in the template for production resource limits; `create_tool` does not require these substrings.
    - Contain a **30s** execution bound (e.g. `timeout 30s` wrapping the `docker` invocation, per your shell — see product validation in code comments).
    - Reference an image that exists on the node, e.g. `pa-sandbox:base`.

18. **Invoke `create_tool`** (via LLM tool call in chat, or a controlled test harness) with parameters:
    - `id` — new unique tool id (e.g. `manual_echo_test`).
    - `index_text` — short description for search / LLM.
    - `template` — full one-line or escaped multi-line string matching §17.
    - `node_id` — must match a key under `config.json` `nodes`.
    - Optional: `arguments`, `system_prompt`.

19. **Expected result:** success message containing the new tool id; **`tools.yaml` on disk** contains a new list entry; **no restart** — the new tool is invokable in the same PA process.

20. **Invoke the new tool** as a normal catalog tool (LLM selects it or you trigger a test call): substitution → SSH → `docker run` on the node. **Expected:** command runs inside the container; stdout/stderr return according to existing noderunner behaviour.

21. **Persistence check:** restart PA once; confirm the tool still loads from `tools.yaml` and runs again.

### Phase 7 — Optional negative / isolation checks

22. **`--network none` (AC-09.018):** define a second tool (or template) with `docker run --rm --network none ...` and run a probe inside the container that attempts outbound HTTP to a public host; expect **failure**.

23. **Reject bad template:** attempt `create_tool` with a template not starting with the whitelisted prefix — expect **error**, no change to `tools.yaml`.

24. **Secret patterns:** if `create_tool_secret_patterns` is set (see **§12**), attempt `create_tool` with a payload that would match one of the patterns (e.g. include `api_key=` in `index_text` if that pattern is listed) — expect **rejection**, no `tools.yaml` change.

---

## Quick checklist (copy for test reports)

- [ ] Images `pa-sandbox:python`, `pa-sandbox:node`, `pa-sandbox:base` present on node (`docker image ls`).
- [ ] Allowlist permits the concrete `docker run` lines used after substitution.
- [ ] `create_tool` succeeds for a valid template; `tools.yaml` updated; tool visible without restart.
- [ ] New tool executes on node via existing SSH path.
- [ ] (Optional) `tools.create_tool_secret_patterns` present in config only when testing §12/§24; invalid regex rejected at startup.
- [ ] (Optional) `network none` blocks outbound; invalid template / secret pattern rejected.

---

## Notes

- **Id collision:** `create_tool` rejects duplicate `id` — use unique ids per manual run.
- **Multi-instance PA:** EP-009 serializes `create_tool` per process; multiple PA instances are not coordinated — avoid concurrent writes to the same `tools.yaml` from several processes.
- **Hermes vs native tools:** If the model uses text-based tools, ensure prompts list `create_tool` when native tools are merged into the session (see implementation).
