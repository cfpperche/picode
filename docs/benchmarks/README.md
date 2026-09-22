# Product & architecture benchmarks

Quality *bars* (how we ship) live in [../benchmarks.md](../benchmarks.md).
This folder is **who we study** before designing a feature.

Before building a substantial surface, read how these projects solved
the same problem, then write the PiCode adaptation (or an ADR). Studies
are dated notes with receipts. Closed-source claims are marked inference.

## Benchmarks

| Project | What it is | Why we watch it | Source |
|---|---|---|---|
| [Cursor](https://cursor.com) | Most-shipped agent IDE | Composer, density, diffs, modes — adapted, never copied as an editor | Public docs/changelog/shipped UI. Full bar: [../benchmark-cursor.md](../benchmark-cursor.md) |
| [pingdotgg/t3code](https://github.com/pingdotgg/t3code) | Agent-harness control surface (Claude/Codex/Cursor/Grok/OpenCode) | Runtime normalization, composer depth, URL-routed threads, waiting state | Open repo — file-path receipts |
| [getpaseo/paseo](https://github.com/getpaseo/paseo) | Daemon + clients for Claude/Codex/Copilot/OpenCode/**Pi** | Same ADE mission, different bet (PTY+hooks, task graphs). They already speak Pi | Open repo — file-path receipts |
| [herdrdev/herdr](https://github.com/herdrdev/herdr) | Rust PTY runtime for guest CLIs | Semantic idle/working/blocked, socket API, later worktrees. **Not** the editor bar | Open repo — file-path receipts |
| [cfpperche/tachyon](https://github.com/cfpperche/tachyon) | Owner prior art (discontinued, GPL-3.0): VS Code + persistent engine; **agents spawn/wait/read/kill sub-agents** via an MCP bridge | Fleet-orchestration patterns for a future direction: delegation contracts, lineage governance, shared region reader, diff-confined validation — **patterns only, licences do not cross** — study [2026-09-14](2026-09-14-tachyon-fleet-orchestration.md) | Owner clone — file-path receipts |
| [stablyai/orca](https://github.com/stablyai/orca) | Desktop agent workbench (tasks, automations, projects with worktrees) — **open repo, MIT** | The right-hand Files/Search/Git/Tasks rail beside the center ([2026-09-05](2026-09-05-inspector-rail.md)); **agent messaging + idle evidence for our six CLIs — [2026-09-14](2026-09-14-orca-agent-messaging.md)**: mailbox + pointer delivery, 3-tier idle evidence, per-vendor patterns | Open repo — file-path receipts |
| [Agent Client Protocol](https://agentclientprotocol.com) | Open editor↔agent standard (Zed) — JSON-RPC over stdio | The guest-CLI escape hatch: Claude, Codex, Gemini, OpenCode, Kimi, Qwen, Droid, Cursor — and **Pi** (pi-acp) — already speak it | Public docs + registry — study [2026-09-03](2026-09-03-guest-tui-agent-state.md) |
| [Devin](https://devin.ai) | Hosted autonomous engineer (Cognition) | **Automations** (triggers → session, ACU/rate caps, activity log, NL-generated config), blocked-and-wake sessions (ADR-0037). Not a runtime or editor bar | Public docs + owner's org UI — hosted, no clone |
| [LibreChat docs](https://www.librechat.ai/docs) | Self-hosted agent-chat docs (Fumadocs site; product is now ClickHouse's Agentic Data Stack chat layer) | Public-docs *product* bar: audience IA, feature-page rhythm, first-run ≠ from-source. Engine stays VitePress | Live site + `LibreChat-AI/librechat.ai` `content/docs/*.json` — study [2026-09-14](2026-09-14-librechat-docs.md) |
| [OpenWiki / docs platforms](2026-09-03-docs-harness.md) | Docs harness study: Diátaxis, Scalar, Vale, Mintlify, Remotion license, HyperFrames, D2 | Public docs completeness/beauty: theme, screenshots pipeline, API reference, prose gate, tutorial videos | Live pages + local receipts, 2026-09-03 |
| [Provider/account managers](2026-09-03-providers-view-v2.md) | Providers-view study: agent IDEs (Kilo, Roo, Zed, Cursor), account switchers (cc-switch, claude-swap), quota monitors (ccusage, CodexBar), credential dashboards (OpenRouter, Vercel, Stripe, Zapier) | Roster row spec, quota inline, credential origin, Verify, blast radius, fallback order | Open repos + live docs, 2026-09-03 |
| [Agent CLI credentials](2026-09-20-agent-cli-credentials.md) | Where each of the nine catalog CLIs keeps credentials (probed on this machine), pi vs omp stores, multi-account switchers, keychain-vs-file on WSL2, BYOK GUI benchmarks | The vault shape for step 1 and the injection matrix for step 2: env vs per-account config dir, one rotating refresh token per consumer, `op run`-style injection at launch | Probed `~/.pi`, `~/.omp`, `~/.claude`, `~/.codex`, `~/.grok`, `~/.hermes`, `~/.config/muse`, `~/.gemini` + vendor docs/repos, 2026-09-20 |
| [SSH terminal access](2026-09-07-ssh-terminal.md) | Who reaches dev/agent terminals over SSH: Coder `coder ssh`, Codespaces `gh cs ssh`, Ona/Gitpod (SSH-first), VS Code Remote-SSH, Tailscale SSH; Terminal-Bench/Daytona as the no-SSH contrast | The second door to the PTYs ADR-0002 already owns — who ships it, who refuses it, what users get | Live docs, 2026-09-07 |
| [akitaonrails/ai-memory](https://github.com/akitaonrails/ai-memory) | Long-term memory server + optional `ai-memory run` launcher for many coding CLIs | Wiki brief vs native transcript Continue (ADR-0088). Complementary, not a substitute | Open repo v2.2.1 — study [2026-09-13](2026-09-13-ai-memory.md) |
| [Design mode / annotations](2026-09-18-annotation-design-mode.md) | Who ships element annotate → agent: Orca (live docs), Lovable (live docs), bolt.diy (source), ChatGPT Work (owner-only) | The grammar four products converged on, and the PiCode adaptation for v2c: frozen-page picker, capture pack, staged file + path through the prompt door | Live docs + upstream source, 2026-09-18 |
| [Zed](https://zed.dev) | Rust editor with an ACP-native agent panel | Cross-file **Review Changes** multibuffer, follow-the-agent, tool permissions as a precedence table, Terminal Threads as a first-class kind — and the registry that re-measures ADR-0091's trigger (5/9 of our CLIs now first-party ACP) — study [2026-09-22](2026-09-22-zed.md) | Live zed.dev docs/blog + shipped `default.json` + ACP registry.json, 2026-09-22 |
| [Computer-use agents](2026-09-16-computer-use.md) | Benchmarks (OSWorld/2.0, WAA, WindowsWorld, WebArena-Verified, ScreenSpot-Pro, OS-Harm), vendor tool contracts (Anthropic/OpenAI/Google/Microsoft), open actuators (Cua, Windows-MCP, Terminator, UFO2), Windows agentic surfaces | The `computer` tool beside `browser`: same vocabulary the vendors converged on, shell as the Rust actuator, tiers × window binding × Ask classes, and where the agent's desktop lives — study [2026-09-16](2026-09-16-computer-use.md) | Official OSWorld results workbook + vendor docs + GitHub/crates.io metadata, fetched 2026-09-16 |
| [2026-09-18 — Embedding Windows apps in a PiCode pane (spike)](2026-09-18-embed-windows-apps.md) | reparenting works for Terminal and Electron, not for WinUI Notepad or Explorer; recommendation: mirror first, capture + input later |

**Adaptation rule** (same as Cursor): borrow a pattern when it improves agent
or CLI-terminal control. ADR-0069 records the owner's multi-CLI direction:
managed agents remain Pi, while other CLIs get managed terminal launches.
Their protocols and packages need a separate decision before first-class
agent support. ADR-0003 still governs managed Pi agents: user-installed `pi`,
no vendored agent SDKs. The current terminal manager adapts the dedicated
configuration and availability patterns cited in ADR-0069.

## Studies

- [2026-09-22 — Zed: the agent panel, and whether ACP’s trigger has fired](2026-09-22-zed.md)
- [2026-09-20 — Credentials for every agent CLI: one vault, many accounts](2026-09-20-agent-cli-credentials.md)
- [2026-09-18 — Design mode / annotations: the grammar four products converged on, and what PiCode adapts](2026-09-18-annotation-design-mode.md)
- [2026-09-16 — Computer use for agents: benchmarks, vendor contracts, OS primitives, and the policy around them](2026-09-16-computer-use.md)
- [2026-09-14 — LibreChat docs as the public-docs product bar](2026-09-14-librechat-docs.md)
- [2026-09-13 — Reusable prompt/command templates (Snippets)](2026-09-13-snippets.md)
- [2026-09-13 — ai-memory as a cross-CLI handoff (wiki brief vs native transcript)](2026-09-13-ai-memory.md)
- [2026-09-10 — A canvas of live terminals on @xyflow/react 12.11.6 (Matrix v2, phase C0 spike)](2026-09-10-node-canvas.md)
- [2026-09-09 — A live grid of terminals on react-grid-layout 2.2.4 (Matrix, phase 0 spike)](2026-09-09-matrix-live-grid.md)
- [2026-09-07 — What each agent CLI records about itself (Claude Code OTel, Codex OTel, ccusage, codex-trace, CliDeck, cli-agent-orchestrator)](2026-09-07-cross-cli-agent-telemetry.md) · re-measured 2026-09-11 (Codex request durations, Grok tokens/cost/turns)
- [2026-09-07 — Write actions on a commit graph (Git Graph, GitLens, GitKraken, Tower, GitButler, lazygit, Conductor)](2026-09-07-git-graph-write-actions.md)
- [2026-09-07 — The right-click menu on a terminal pane (VS Code, Windows Terminal, Ghostty, iTerm2, Warp, Cursor)](2026-09-07-terminal-context-menu.md)
- [2026-09-07 — Agent notifications and the toast payload (Superset)](2026-09-07-superset-notifications.md)
- [2026-09-08 — Pin reminders and sticky notifications (Apple, Keep, Todoist, Slack, Things, VS Code, Sonner)](2026-09-08-pins-reminders.md)
- [2026-09-07 — Reaching the agent terminals over SSH (Coder, Codespaces, Ona, VS Code, Tailscale, Terminal-Bench)](2026-09-07-ssh-terminal.md)
- [2026-09-07 — The actions on a workspace card (VS Code, PatternFly, Carbon, WCAG, GitHub Desktop)](2026-09-07-workspace-card-toolbar.md)
- [2026-09-06 — Attaching images and files to Agent CLI terminals](2026-09-06-cli-terminal-attach.md)
- [2026-09-06 — The editor tab strip when tabs overflow (VS Code, Zed, JetBrains, Firefox, Chrome, UI kits)](2026-09-06-tab-strip-overflow.md)
- [2026-09-05 — The right-hand inspector rail (Paseo, Orca, t3code)](2026-09-05-inspector-rail.md)
- [2026-09-05 — Desktop/mobile decoupling](2026-09-05-mobile-decoupling.md)

- [2026-09-04 — Release cadence and release-process documentation](2026-09-04-release-cadence.md)
- [2026-09-03 — Providers view v2 (accounts, quota, model catalog)](2026-09-03-providers-view-v2.md)
- [2026-09-03 — Docs harness (theme, screenshots, API reference, prose gate, videos)](2026-09-03-docs-harness.md)
- [2026-09-03 — Guest TUI agent state (spinner / needs-you for guest CLIs)](2026-09-03-guest-tui-agent-state.md)
- [2026-09-02 — Live browser preview in chat / side panel](2026-09-02-live-browser-preview.md)
- [2026-09-01 — Devin Automations (and peers)](2026-09-01-devin-automations.md)
- [2026-09-01 — Supervising coding agents from a phone](2026-09-01-mobile-agent-supervision.md)
- [2026-09-01 — LLM observability dashboards](2026-09-01-llm-observability-dashboards.md)
- [2026-08-30 — Copying out of a browser terminal backed by tmux](2026-08-30-web-terminal-clipboard.md)
- [2026-08-27 — Herdr](2026-08-27-herdr.md)
- [2026-08-24 — Adopt t3code / paseo / Cursor](2026-08-24-adopt-t3code-paseo-cursor.md)
