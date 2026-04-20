---
name: Web and GitHub output hygiene
description: When the user asks about the web, GitHub, URLs, fetching pages, searching online, digests from news sites, or any workflow that may use web_search or web_fetch. Prefer small tool outputs over large HTML.
tools:
  - web_fetch
  - web_search
  - run_on_node
---

## Playbook

- For web or GitHub research, keep tool outputs small (prefer raw files, API, or git on nodes over huge HTML pages).
