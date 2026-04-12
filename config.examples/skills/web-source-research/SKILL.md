---
name: Web and GitHub source research
description: When the user asks to investigate, summarize, or explore a public website or an open-source project on GitHub. Guides bounded use of web_fetch, web_search, and run_on_node so responses stay within useful LLM context.
tools: ["web_fetch", "web_search", "run_on_node"]
---

## Goals

- Answer the question with the **smallest** amount of fetched text.
- Avoid dumping **multi‑hundred‑KB HTML** (e.g. GitHub repository UI pages) into the conversation.

## GitHub and code hosts

- Prefer **`raw.githubusercontent.com`** (or the host’s raw file URL) for a **single file** (README, `go.mod`, license) instead of the HTML repo homepage.
- Prefer the **GitHub HTTP API** for metadata (description, topics, default branch) when a tiny JSON response is enough.
- If a **git clone** exists on a configured node, use **`run_on_node`** with **bounded** commands: `git show`, `git grep`, `git ls-tree`, always pipe or limit output (e.g. `head`) so stdout stays small.

## Generic websites

- Prefer **one URL per step**: landing page, then `/docs` or `/about` if needed—not the whole site at once.
- **`web_fetch`** returns UTF‑8 body text truncated by **`max_body_bytes`** in config; treat that as a hard ceiling and still prefer smaller URLs when possible.
- Use **`web_search`** for discovery, then **`web_fetch`** on **specific** pages, not open-ended crawling.

## Tool hygiene

- Prefer **narrow** `web_fetch` URLs over “everything” pages.
- After a large or noisy fetch, **summarize** in your reply and avoid repeating the full raw body in follow‑up turns unless the user asks.
