# PersonalAssistant — user documentation

Operator-focused guides for installing, configuring, running, and troubleshooting PersonalAssistant. For SDLC and epic artefacts, see the repository root [README.md](../README.md) and the nested process clone [ai-sdlc/](../ai-sdlc/) (local checkout per [ai-sdlc.version](../ai-sdlc.version)).

| Document | Description |
|----------|-------------|
| [architecture.md](architecture.md) | Composition root (`cmd/pa/wire`), jobs runtime phases, subsystem insertion checklist (EP-042). |
| [architecture-ru.md](architecture-ru.md) | Architecture overview (Russian): C4 diagrams, message flow, subsystems, security. |
| [installation.md](installation.md) | Prerequisites (Go, CGO/SQLite), clone, dependencies, first build. |
| [configuration.md](configuration.md) | Environment variables, `config.json`, path resolution, secrets, key JSON sections. |
| [llm-provider-roles-and-logging.md](llm-provider-roles-and-logging.md) | How `llm_providers` maps to chat, summarization, transport fallback, intent classifier; `PA_LOG_LEVEL` / `PA_ENV`. |
| [docker.md](docker.md) | Docker Compose, volumes, secrets, optional `TZ`, in-container summarization cron. |
| [operations.md](operations.md) | Running the binary, CLI flags, logs, LLM log files, scheduler. |
| [observability-http.md](observability-http.md) | Optional health/readiness HTTP (`observability_http`), Docker probes, lifecycle log fields (EP-029). |
| [troubleshooting.md](troubleshooting.md) | Common failures and checks. |
| [Threat model (artefact)](../ai-sdlc-artefacts/threat-model.md) | Code-grounded security overview for operators (not a pentest report). |
| [Architecture patterns playbook](../ai-sdlc-artefacts/architecture-patterns-playbook.md) | Product defaults for consulting the ai-sdlc architecture-patterns catalog (ASD hints; not a substitute for card `when_not` / `kiss_default`). |
