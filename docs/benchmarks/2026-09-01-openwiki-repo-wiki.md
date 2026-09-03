# Study: OpenWiki (and peers) — an agent-maintained wiki for the codebase

- **Date:** 2026-09-01
- **Sources:** clone of [langchain-ai/openwiki](https://github.com/langchain-ai/openwiki)
  at HEAD `64903f9` (2026-09-01, `openwiki@0.5.0`, MIT, ~16k stars, 61 open
  issues), read file by file — receipts below are paths in that clone. The
  LangChain launch post
  ([Introducing OpenWiki](https://www.langchain.com/blog/introducing-openwiki-an-open-source-agent-for-repo-documentation)).
  Peers from their public READMEs / docs: Cognition
  [DeepWiki](https://docs.devin.ai/work-with-devin/deepwiki),
  [AsyncFuncAI/deepwiki-open](https://github.com/AsyncFuncAI/deepwiki-open),
  [he-yufeng/RepoWiki](https://github.com/he-yufeng/RepoWiki),
  [abhigyanpatwari/GitNexus](https://github.com/abhigyanpatwari/GitNexus),
  Google's [OKF v0.2 spec](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md),
  Ry Walker's [code-intelligence tools comparison](https://rywalker.com/research/code-intelligence-tools),
  and the ETH Zurich paper
  [Evaluating AGENTS.md](https://arxiv.org/abs/2602.11988) (Feb 2026).
  Hosted products (DeepWiki) have no receipts; their internals are marked
  *inference*. Nothing was **run** — no `openwiki --init` on this repo yet
  (see open question 1).
- **Scope:** whether and how PiCode gives its agents a *maintained*
  description of the codebase they work in, and lets the human read it.
  Not a bar for the composer, the editor tab or the dashboard.

## The problem, in PiCode terms

A pi agent starts every session from `AGENTS.md` plus whatever it greps.
`docs/architecture.md` in this repo is hand-maintained by contract
(non-negotiable #1) and is 500+ lines that drift the moment a session
forgets the handoff ritual. Every other workspace a PiCode user opens has
nothing like it. The question this study answers: which of the tools that
generate and *keep current* a codebase wiki fits a pi-controlled ADE, and
what PiCode has to build (as little as possible) to plug one in.

## What OpenWiki ships

| Facet | OpenWiki | Receipt / notes |
|---|---|---|
| Output | `openwiki/` in the repo: Markdown pages with OKF v0.2 front matter (`type` required; `generated`, `verified`, `sources`, `status`, `stale_after` optional), generated `index.md` per directory, `quickstart.md` task-routing page, validated Mermaid fences (a broken diagram degrades to a `text` fence with a comment, repaired on the next update). User brief in `openwiki/INSTRUCTIONS.md`, never rewritten. | `src/okf/*`, `src/mermaid/*`. Two modes: `code` (repo) and `personal` (`~/.openwiki/wiki`, connectors). Only `code` matters here. |
| Lifecycle | `begin → submit_plan → next_page → submit_page … → finish`. Ordered durable page-job queue in `openwiki/.run.json`; `openwiki/.page-manifest.json` per-page source baselines. Interrupted run resumes; a page is "done" only after Markdown + Claims + verification are persisted. | `src/generation/run-state.ts`, `page-jobs.ts`, `repository-run.ts`. |
| Grounded Claims | Each factual page carries Claims: one falsifiable proposition + evidence spans `repo://src/x.ts#L20-L48` with the evidence version seen. Before an update the evidence is rechecked *before* deciding the run is a no-op; a stale Claim forces work on its page even if the planner skipped it. The page worker only receives stale/unresolved Claims and submits sparse decisions (`confirmedClaimIds` / `claims` / `retractedClaimIds`). | `src/claims/*`, sidecars under `openwiki/.claims/`. This is what makes daily runs cheap: an untouched module costs zero model turns. |
| Two engines | **Native**: its own DeepAgents 1.12 agent (`createDeepAgent`, `subagents: []`), 13 providers (OpenAI default, Anthropic, Gemini, Vertex, Bedrock, Copilot, OpenRouter, OpenAI-compatible…), keys in `~/.openwiki/.env`. **Host-driven**: a stdio MCP server (`openwiki mcp --host <id>`) exposing the six lifecycle tools + a `SKILL.md`; the *host coding agent* (Codex, Claude Code, OpenCode, Cursor) does the research and writes each page with its own tools and model; OpenWiki owns queue, validation, Claims, indexes, provenance. | `src/agent/repository-runner.ts`; `src/integrations/mcp/server.ts`, `src/integrations/core/session-manager.ts:225-264`, `integrations/openwiki/SKILL.md`. |
| Host install | `openwiki integrations install codex\|claude\|opencode\|cursor [--project]` copies the skill bundle to the host's skills dir and writes one `mcpServers.openwiki = {command: "openwiki", args: ["mcp","--host",<id>]}` entry into the host's MCP config, with a receipt for safe uninstall. **The registry has four targets; the MCP server itself accepts any host id** matching `[a-z0-9-]{1,64}`. | `src/integrations/install/registry.ts:16-77`, `installer.ts`, `src/cli/commands.ts:686`. |
| Agent hookup | Maintains an `AGENTS.md` and `CLAUDE.md` at the repo root that *point at* the wiki, rewriting only its `<!-- OPENWIKI:START -->…<!-- OPENWIKI:END -->` block. The wiki is referenced, not inlined into the context file. | README "How it stays yours". |
| Update | `openwiki --update`: diff HEAD against the last documented commit (needs full history — the CI example uses `fetch-depth: 0`) + Claims evidence recheck. A clean run skips model work and only touches `.last-update.json`. | `src/ingestion/code-mode.ts`, `examples/openwiki-update.yml`. |
| Keep current | GitHub Actions / GitLab CI / Bitbucket examples: daily cron, `openwiki code --update --print`, then a PR (`peter-evans/create-pull-request`) that carries partial progress on failure. | `examples/openwiki-update.yml`. |
| Human reader | `openwiki visualize` serves a node graph + Markdown reader on `127.0.0.1:4321`; `--export` writes a static site. Loads graph/markdown/diagram libraries from a public CDN. | `src/visualize/*`. |
| Footprint | Node ≥ 22, `better-sqlite3` native (via langgraph checkpoint), full LangChain stack, Ink TUI, PostHog telemetry (`OPENWIKI_TELEMETRY_DISABLED=1`). ~48k lines of TypeScript. | `package.json`, `pnpm-lock.yaml`, `src/telemetry/config.ts`. |

Two things to hold on to. First, **host-driven mode is the product for
us**: the agent the user already configured in PiCode writes the docs,
OpenWiki is just a deterministic state machine over MCP. Second, the
integration surface for a new host is *two files* — a skill directory and
an MCP config entry — and PiCode already writes both kinds of file.

## Peers — two families

| Tool | Family | Runs | Authoring model | Update | Agent delivery | License / cost |
|---|---|---|---|---|---|---|
| **OpenWiki** | prose wiki | local CLI or CI | own DeepAgents *or* the host coding agent (MCP) | git diff + Claims recheck; no-op is free | `AGENTS.md` pointer block; Markdown in repo | MIT; your model tokens |
| **DeepWiki** (Cognition) | prose wiki | hosted | Devin | on connect; cadence undocumented (*inference*: on push) | remote MCP `ask_question`, `read_wiki_structure`, `read_wiki_contents` — **already a preset in PiCode's MCP catalog** (`internal/mcp/mcp.go:99`) | public repos free; private via Devin, Medium 5–10 ACU / High 20–40 ACU per wiki |
| **deepwiki-open** | prose wiki + RAG chat | self-hosted Python API + Next.js, Docker | its own model call (OpenAI/Gemini/Ollama/OpenRouter) + embeddings | regenerate; no incremental story in the README | web UI, "Ask" | MIT, 17.9k stars |
| **RepoWiki** | prose wiki + chat | local Python/FastAPI/SQLite CLI | litellm (100+ providers) | hash-based incremental (`.repowiki-state.json`) | web UI, terminal chat, HTML/JSON/MD export | MIT |
| **GitNexus** | structural index (+ optional wiki) | local tree-sitter → LadybugDB graph, WASM in browser | none for the graph; `gitnexus wiki` needs an LLM | `analyze --watch` incremental (300 ms) | 17 MCP tools, Claude Code skills + hooks | **PolyForm Noncommercial** |
| CodeGraph / Serena / Repomix | structural index / symbol retrieval / context packing | local | none | file watchers | MCP / CLI | MIT |
| Claude Code `/init` + auto memory, Codex `/init` | context file | in the agent | the agent itself | never, unless asked | `CLAUDE.md` / `AGENTS.md` inlined | included |

Where the field agrees: **MCP is how a codebase description reaches an
agent**, local-first wins over hosted for private code, and incremental
update is table stakes (Ry Walker's conclusion for the structural family;
OpenWiki's Claims are the prose-side equivalent). Where it splits: *who
writes the prose*. deepwiki-open and RepoWiki bring their own model
runtime and key store; DeepWiki brings Devin; OpenWiki host mode and
`gitnexus wiki` hand the writing to the coding agent you already have.

The structural family solves a **different problem** (retrieval:
"what calls this?") and complements a wiki rather than replacing it. Any
of them is one stdio MCP server away through `#/mcps`; none needs PiCode
code.

**Caveat that survives all of this.** The ETH Zurich study measured that
LLM-generated `AGENTS.md` files *lowered* success by ~0.5–2 points and
raised inference cost by 20–23 %, while human-written ones gained ~4 points
at similar cost. OpenWiki's design answers this partially — the context
file gets a pointer, the wiki is read on demand — but nobody has measured
a pi agent with and without an OpenWiki. That measurement is the first
thing to do, before any UI (open question 1).

## Why OpenWiki, for PiCode

- **No second model runtime, no second key store.** Host mode uses pi's
  model and pi's tools; ADR-0013's vault stays the only auth. deepwiki-open
  and RepoWiki would each add a Python service with its own provider
  config next to the one PiCode already manages.
- **Output PiCode already renders.** Markdown + Mermaid in the repo; the
  file tab previews both (file-preview roadmap, closed 2026-08-29). No
  viewer to build.
- **Cheap to keep current, and it fits the Automations shape** from the
  Devin study: a scheduled run is an ordinary agent session whose first
  MCP call usually answers `status: "noop"`.
- **MIT**, actively shipped (0.4.0 → 0.5.0 within weeks), and the host
  integration is a documented extension point (`CONTRIBUTING.md`, "Adding
  a coding-agent integration").
- Against: Node 22 + a native module the user must install; a 48k-line
  dependency we do not own; the visualizer needs a CDN; pi is not a
  registered host (but see below — the server does not care).

## What PiCode adapts

| Pattern | From | PiCode version |
|---|---|---|
| Host-driven wiki, pi as the host | OpenWiki `integrations install <host>` | `openwiki mcp --host pi` as a **command** MCP server through `pi-mcp-adapter` (the same file `#/mcps` already writes: `<cwd>/.mcp.json` or `~/.pi/agent/mcp.json`), plus the skill bundle copied to `<workspace>/.pi/skills/openwiki/` (pi loads `SKILL.md` from there today — this repo's own skills live in that layout). PiCode performs the two writes the upstream registry does not know how to do for pi; nothing else is ours. Add an **OpenWiki** preset to `internal/mcp` `Presets()` beside DeepWiki (`Command: "openwiki", Args: ["mcp","--host","pi"]`). |
| The two canonical prompts | `integrations/openwiki/SKILL.md`, `agents/openai.yaml` `default_prompt` | Composer picker entries `/wiki init` and `/wiki update` that insert the exact prompts ("Initialize this repository's OpenWiki from the current source and tests." / "Update … for changes since its last successful run."). Listed only when the skill directory exists in that agent's cwd (ADR-0029 rule: chrome appears because the capability is installed, never hardcoded). |
| Keep current | `examples/openwiki-update.yml` (daily cron → PR) | An **Automations template** (Devin study): schedule → new agent with the update prompt, concurrency 1, cost cap. The daemon on the owner's box is the scheduler, not GitHub Actions; the CI workflow stays the team option and PiCode never writes it. |
| Read the wiki | `openwiki visualize` | Open `openwiki/quickstart.md` / `index.md` in the existing file tab (md + mermaid preview). A sidebar affordance on the workspace card ("Wiki") is enough; no graph in v1. |
| Agent hookup | managed `<!-- OPENWIKI -->` block in `AGENTS.md` | Accept it. pi reads `AGENTS.md` natively; PiCode adds nothing. |
| Upstream | `CONTRIBUTING.md` "Adding a coding-agent integration" | PR a `pi` target to `src/integrations/install/registry.ts` (`skillDirectory: ".pi/skills/openwiki"`, `mcpConfig: {kind: "json", relativePath: ".pi/mcp.json"}` project-level; `~/.pi/agent/mcp.json` user-level). When merged, PiCode's installer shrinks to running `openwiki integrations install pi`. |
| Provenance | OKF `generated.by = <host>` | Pass `--host pi` so pages stamp `pi/<version>`; PiCode-triggered runs are distinguishable from CI runs in the front matter. |

## What PiCode refuses

| Temptation | Why not |
|---|---|
| A wiki generator in the Go binary (deepwiki-open / RepoWiki model) | Second model runtime and key store next to the vault (ADR-0013); ADR-0003 says user-installed tools on the user's machine. OpenWiki is that tool. |
| Vendoring / bundling `openwiki` in PiCode | Same rule as `pi` and `gh`: detect it on `PATH`, one line + one action ("Install openwiki", `www/guide/`), never `npm install` from the binary. |
| OpenWiki **native** mode from the GUI | Its own `~/.openwiki/.env` provider config duplicates Providers. Native stays available from a terminal for people who want it; the GUI path is host mode only. |
| Structural indexes (GitNexus, CodeGraph, Serena) in core | Different problem; GitNexus is PolyForm Noncommercial. Any of them is a user-added MCP server. |
| Hosted DeepWiki for private repos | Requires Devin. The public-repo preset already exists in the catalog. |
| `personal` mode and connectors | Not a codebase concern. |
| Visualizer as an App (ADR-0036) in v1 | Pulls libraries from a public CDN and wants its own origin; the file tab covers reading. Revisit if the static export becomes self-contained. |
| Auto-running `--init` on workspace creation | ADR-0027: workspaces start empty. A wiki costs real tokens; the user asks for it. |

## Open questions for the ADR

1. **Does it help pi at all?** Dogfood on this repo: `openwiki --init` via
   a PiCode agent, then a handful of real tasks with and without the wiki
   pointer, comparing turns, tokens and outcome. The ETH Zurich result says
   the default expectation is *slightly negative*; ship only if the
   measurement says otherwise. Also compare the generated
   `openwiki/architecture*` pages against `docs/architecture.md` — if they
   agree, the hand-written file has a checker; if they disagree, one of
   them is wrong.
2. **Skill scope** — workspace `.pi/skills/openwiki/` (committed, every
   clone gets it) or user-level `~/.pi/agent/skills/`. Proposal: workspace
   by default, matching upstream's `--project` semantics; user-level as
   the option.
3. **Dirty tree** — an automation run leaves `openwiki/` modified in the
   agent's cwd, which collides with the worktree rule (non-negotiable #5)
   in this repo and with any user who keeps `main` clean. Proposal: the
   scheduled agent works in its own worktree and pushes a branch, the way
   the GitHub Action opens a PR; interactive `/wiki update` leaves the
   changes for the user to review in the working-tree diff (ADR-0032).
4. **Cost per run** — unmeasured. One public write-up reports a smaller
   model repeatedly failing on tool calls where a larger one finished; in
   host mode the model is whatever role the automation pins (ADR-0028), so
   the template should pin a strong one and the Automations cost cap is the
   guard.
5. **Upstream or local installer first** — proposal: both. PiCode writes
   the two files today; the `pi` target PR goes upstream in parallel and
   the local code is deleted when it lands.
