---
name: user-documentation
description: >-
  Generates or refreshes end-user documentation under docs/ and updates the
  project README.md as the primary entry point. Use when the user asks for user
  docs, operator guide, installation/configuration guide, README refresh for
  end users, or documentation for deployment and daily use (not SDLC artefacts).
---

# End-user documentation (operator / deployer)

**Pipeline:** Cross-cutting (not a numbered stage). Complements contributor-facing material under `ai-sdlc/` and epic artefacts under `ai-sdlc-artefacts/`.

**Outputs (authoritative paths):**

| Output | Path | Role |
|--------|------|------|
| User doc set | **`docs/`** | Detailed guides (markdown). |
| Project front door | **`README.md`** (repository root) | Short overview, quick start, links into `docs/`. |

---

## 1. Goal and audience

**Audience:** Operators and end users who run PersonalAssistant (configure, deploy, operate the bot) — **not** SDLC authors working only in `ai-sdlc-artefacts/`.

**Goal:** Produce accurate, practical documentation derived from the **current codebase and checked-in examples**, with:

- No invented flags, env vars, or config keys — verify against `cmd/pa`, `internal/config`, `docker-compose.yml`, `Makefile`, and **`config.examples/config.example.json`**.
- **No secrets** in examples (placeholders and file-path patterns only; same discipline as project README and AGENTS.md).
- **English** for all generated user-facing prose (titles, body, tables).

**Traceability:** Prefer linking to paths in the repo (e.g. `config.examples/config.example.json`) rather than duplicating large JSON blocks unless a short excerpt helps.

---

## 2. Inputs

Read and reconcile at least:

- **Entrypoints:** `cmd/pa/main.go` (flags, subcommands or modes such as `-verify-nodes`).
- **Configuration:** `config.examples/config.example.json`, `internal/config/` (validation rules, required fields).
- **Deployment:** `docker-compose.yml`, `Dockerfile`, any `scripts/` used in docs today.
- **Existing user surface:** Current **`README.md`** (preserve good structure; fix drift).
- **Optional context:** `ai-sdlc-artefacts/scope.md` for high-level product description (keep user doc solution-oriented, not process-heavy).

If the user requests a **narrow update** (e.g. “document only Docker”), still skim config and README so links and env vars stay consistent.

---

## 3. Recommended layout under `docs/`

Create or update as needed (adjust titles if the user prefers a different split; ask when in doubt):

| File | Typical contents |
|------|------------------|
| **`docs/README.md`** | Index: one-line description of each doc; link back to repo `README.md`. |
| **`docs/installation.md`** | Prerequisites (Go version, CGO/SQLite notes if applicable), clone, `go mod tidy`, build, first run. If `make check` is mentioned: **quality gate only** (fmt, vet, lint, tests, coverage, boundaries) — **not** installation or deployment. |
| **`docs/configuration.md`** | `PA_CONFIG_DIR`, `PA_DATA_DIR`, `PA_SECRETS_DIR`, `PA_LOG_LEVEL`; how paths resolve; copy from `config.examples/` into **`.config/`**; secrets as files; optional nodes / tools / escalation overview with links to example keys. Document every parameter in the configuration file (aligned with `config.examples/` and the live schema). |
| **`docs/docker.md`** | Compose services, volumes, env table, secrets layout, cron/summarization + timezone note if present in code/README. |
| **`docs/operations.md`** | Running the binary, `-verify-nodes`, logs location, safe log levels, where to find LLM logs (if configured). |
| **`docs/troubleshooting.md`** | Common failures (config load, SSH, allowlist, missing secrets) with **symptom → check → fix**; no internal stack traces unless they help the operator. |

Omit or merge files if the user asks for a minimal set; **do not** leave `docs/` without a **`docs/README.md`** index once any other `docs/*.md` exists.

---

## 4. `README.md` (root) responsibilities

When executing this skill, **update `README.md`** so it:

1. Stays a **short** onboarding path: what the project is, prerequisites in one glance, minimal commands to build/run.
2. Points to **`docs/`** for depth (e.g. “Full guides: [docs/README.md](docs/README.md)” or per-topic links).
3. Does not duplicate entire `docs/` pages — link instead.
4. Remains aligned with **checked-in** behaviour (env vars, flags, Docker) after any doc refresh.

Contributor / pipeline pointers may remain in README (e.g. link to `ai-sdlc/`) but must not overwhelm the operator quick start unless the user explicitly wants a contributor-first README.

---

## 5. Workflow

1. **Scope** — Confirm with the user if unclear: full refresh vs single topic; default is **full user-doc sync** when they ask generically.
2. **Inventory** — List existing `docs/*.md` and README sections; note gaps vs code.
3. **Draft** — Produce structured markdown (**English**). Prefer showing **full** draft content **in chat** for `docs/*` and README deltas so the user can review without opening files.
4. **Save** — Write or update files under **`docs/`** and **`README.md`** only after **explicit user approval** (e.g. “save”, “write”, “lgtm”, “apply”). If the user asked to “create the docs” without approval semantics, treat **“create/write the files now”** as explicit only when they clearly asked to write to disk; otherwise default to **draft in chat first**, then save on approval (see [skills README](README.md) — human-in-the-loop).
5. **Verify** — After writes: quick pass for broken relative links from `README.md` and from each `docs/*.md` to repo paths.

**Options:** If multiple valid structures exist (e.g. single long `docs/user-guide.md` vs split files), present **A / B** briefly and ask the user to choose before saving.

---

## 6. Non-goals

- Do not replace or duplicate **epic requirements / AC / design** in `ai-sdlc-artefacts/`; link lightly if useful for “why” background.
- Do not document **internal package APIs** unless needed for operators (rare).
- Do not commit real tokens, API keys, or private keys.

---

## 7. Done when

- [ ] Audience is clearly **operator / end user**; prose is **English**.
- [ ] Outputs are under **`docs/`** (with index) and **`README.md`** is updated as the entry point.
- [ ] Content matches **current** code and **`config.examples/config.example.json`**; no fabricated CLI/config.
- [ ] No secrets in examples.
- [ ] User explicitly approved **save** when following draft-then-save workflow.
