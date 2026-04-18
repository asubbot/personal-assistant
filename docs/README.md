# PersonalAssistant — user documentation

Operator-focused guides for installing, configuring, running, and troubleshooting PersonalAssistant. For SDLC and epic artefacts, see the repository root [README.md](../README.md) and [ai-sdlc/](../ai-sdlc/).

| Document | Description |
|----------|-------------|
| [installation.md](installation.md) | Prerequisites (Go, CGO/SQLite), clone, dependencies, first build. |
| [configuration.md](configuration.md) | Environment variables, `config.json`, path resolution, secrets, key JSON sections. |
| [llm-provider-roles-and-logging.md](llm-provider-roles-and-logging.md) | How `llm_providers` maps to chat, summarization, escalation, intent classifier; `PA_LOG_LEVEL` / `PA_ENV`. |
| [docker.md](docker.md) | Docker Compose, volumes, secrets, optional `TZ`, in-container summarization cron. |
| [operations.md](operations.md) | Running the binary, CLI flags, logs, LLM log files, scheduler. |
| [observability-http.md](observability-http.md) | Optional health/readiness HTTP (`observability_http`), Docker probes, lifecycle log fields (EP-029). |
| [troubleshooting.md](troubleshooting.md) | Common failures and checks. |
| [Threat model (artefact)](../ai-sdlc-artefacts/threat-model.md) | Code-grounded security overview for operators (not a pentest report). |
