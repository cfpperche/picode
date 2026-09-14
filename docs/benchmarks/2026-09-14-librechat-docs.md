# Study: LibreChat docs as the public-docs product bar

- **Date:** 2026-09-14
- **Sources:** live [librechat.ai/docs](https://www.librechat.ai/docs)
  (2026-09-14); repo [LibreChat-AI/librechat.ai](https://github.com/LibreChat-AI/librechat.ai)
  (`content/docs/meta.json`, `features/meta.json`, `configuration/meta.json`,
  `configuration/librechat_yaml/object_structure/meta.json`,
  `mcp_servers/meta.json`, `quick_start/local_setup.mdx`,
  `configuration/librechat_yaml/index.mdx`, `features/upload_as_text.mdx`,
  README). ClickHouse acquisition context:
  [clickhouse.com/blog/librechat-open-source-agentic-data-stack](https://clickhouse.com/blog/librechat-open-source-agentic-data-stack).
- **Scope:** *content architecture and page rhythm* of a self-hosted agent
  product's public docs. Not the LibreChat app as an ADE, not a docs-engine
  swap. The harness (theme, shots, Scalar, Vale, videos) stays the 2026-09-03
  [docs-harness](2026-09-03-docs-harness.md) study.

## Why now

The owner asked for LibreChat's documentation to be a PiCode benchmark.
LibreChat is the chat layer of ClickHouse's Agentic Data Stack: a
self-hosted, multi-user agent product with the same class of reader we
write for (install it, configure models and tools, use agents, do not
become a contributor). Stripe remains the *docs-as-product* bar; Diátaxis
remains the *mode* bar; LibreChat is the *self-hosted agent IA* bar.

## What it is (for us)

A separate Next.js 16 + [Fumadocs](https://fumadocs.dev) site. MDX in
`content/docs/`, sidebar from per-directory `meta.json`. Also: blog,
changelog, in-page Ask AI (Vercel AI SDK + OpenRouter), Orama search, 14
languages, Copy Markdown. The product repo is `danny-avila/LibreChat`;
docs issues go to `LibreChat-AI/librechat.ai`.

PiCode keeps VitePress in-repo (`docs-site/`, bar #1). We steal IA and
page shape, not the generator, the extra repo, Ask AI, or i18n.

## Their IA (receipt: `content/docs/meta.json`)

Audience groups, not a flat feature list:

| Group | Pages | Reader question |
|---|---|---|
| **Deploy** | `quick_start`, `local` (Docker / npm / Helm), `remote` (DO, Railway, nginx, Cloudflare, ngrok, Traefik) | How do I get it running? |
| **Configure** | `.env`, `librechat.yaml`, auth, Docker override, Langfuse, object-structure reference | How do I turn a thing on without breaking the rest? |
| **Use** | `features` (catalog), `mcp_servers` (cookbook), `user_guides` | What exists, and which click? |
| **Tools / Contributing** | toolkit, translation, `development` | I am extending it. |

`features/meta.json` further groups the catalog: Agentic AI, Search &
Knowledge, Media, Chat, Security. `object_structure/meta.json` is one page
per YAML object (interface, agents, mcp_servers, memory, …).
`mcp_servers/` is a cookbook (index + Google Workspace + Salesforce) —
not a single MCP essay.

## Page rhythm (receipts)

**Feature** (`features/upload_as_text.mdx`): one-line promise → UI steps
(paperclip → “Upload as Text”) → what it is *not* (not RAG, not vision
upload) → the flag that hides it (`context` capability) → troubleshooting.
The “not this” table is the page.

**Config** (`configuration/librechat_yaml/index.mdx`): the callout
**“If you only remember one thing”** — for Docker, editing the file is not
enough; it must exist, be mounted, and the process restarted. Then a
numbered path (create → mount → restart → verify) with Docker vs local
tabs, then one worked example (OpenRouter), then the object-structure
index.

**First run** (`quick_start/local_setup.mdx`): download ZIP or `git clone`
→ install Docker Desktop → copy `.env` → `docker compose up -d`. Three
steps. No Node, no from-source toolchain. From-source lives under Local
Installation (`npm`, Helm).

## PiCode today

`docs-site/` already has Diátaxis-ish groups (Start / Guides / Run it
somewhere / Reference), `llms.txt`, Scalar `/api`, generated shots. The
gaps against this bar:

| LibreChat | PiCode now |
|---|---|
| First run is Docker Desktop | `guide/getting-started.md` opens on Go 1.26, Node 22, Pi, tmux, `make build` |
| Features catalog, grouped | One flat Guides list of 20+ named surfaces |
| Config overview + per-object tables | One `guide/settings.md` |
| MCP cookbook (one server per page) | One `guide/mcp.md` |
| “If you only remember one thing” | Buried in prose |
| “What this is not” | Rare; MCP does it once (“Pi does not speak MCP itself”) |
| Changelog + blog on the docs site | Fragments in `docs/changelog.d/`; the site has no changelog page |

## Take

| Pattern | Adapt as |
|---|---|
| Audience IA: Deploy / Configure / Use | Keep Diátaxis names if they still teach; split **Use** (catalog) from **Configure** (reference). Run stays its own group. |
| Feature page rhythm | Every user-visible capability: what it is → UI path → how to enable → what it is not. |
| Config one-liner | How-tos that can brick a deploy open with one sentence that, remembered alone, does not. |
| First run ≠ from source | Getting started is `picode install` / Docker. `make build` moves to a from-source page. |
| MCP cookbook | `guide/mcp.md` becomes the index; one page per connector we actually ship a path for. |
| Worked example after the workflow | One concrete “add this provider / connector” on the config overview, then the tables. |

## Refuse

| LibreChat does | Why we do not |
|---|---|
| Fumadocs / Next.js / MDX components (`Steps`, `Cards`, `DocsHub`) | Bar #1: VitePress only. Markdown + tables + existing VitePress containers. |
| Separate docs repo | Code and docs in the same commit (AGENTS.md). |
| Ask AI on the page | New runtime, API key, rate limit. `llms.txt` + local search stay. |
| 14 languages | English only. |
| Copy Markdown button | Nice; not a bar. Agents already have `llms.txt`. |
| Good/Bad page widget | Skip. |
| Blog as a docs-site section | Changelog fragments are the product history; a public changelog page is optional later, not a blog engine. |
| Their YAML object model, screenshots, or copy | Ours. |

No ADR. This crosses no protocol, persistence, security, or process
boundary.

## Map (LibreChat group → PiCode page)

Use this when rewriting `docs-site/`; do not clone their catalog.

| Their group | Our pages (today) | Gap |
|---|---|---|
| Quick Start / Local Docker | `guide/getting-started.md` | Contributor toolchain is the first path |
| Remote | `windows-desktop`, `remote-server`, `shared-server`, `public-access`, `mobile`, `ssh-terminals` | Already the closest match |
| Configuration overview | — | Missing. Closest: `settings.md` |
| Features / Agentic | `agent-clis`, `packages`, `mcp`, `automations`, `snippets`, `canvas`, `communication` | No landing; mixed how-to and reference |
| Features / Chat & files | `files`, `inbox-tools`, `diff-panel`, `compact`, `checklist` | Same |
| MCP servers cookbook | `mcp.md` | Index only |
| Object structure | `settings.md`, `providers.md` | Not key/type/default/example tables |
| Development | README + `docs/` (internal) | Keep internal; do not publish ADRs |

PiCode-only pages stay (tmux, llama, browser-tool, browser-extension,
keyboard, roles, terminal-status, integrations, docker app, security).
They get the feature rhythm; they do not get a LibreChat twin.

## Next (not this session)

1. Sidebar: Start / Use / Run / Configure / Reference.
2. Getting started: user first-run; from-source demoted.
3. Feature rhythm on the existing guides (no new engine).
4. Configuration overview + MCP cookbook.

Owner confirms Take/Refuse before those edits. The docs-harness pipeline
(shots, Scalar, Vale) is unchanged.
