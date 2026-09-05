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
| [herdrdev/herdr](https://github.com/herdrdev/herdr) | Rust PTY runtime for guest CLIs | Semantic idle/working/blocked, socket API, later worktrees. **Not** the editor bar | Open repo — file-path receipts |
| [Agent Client Protocol](https://agentclientprotocol.com) | Open editor↔agent standard (Zed) — JSON-RPC over stdio | The guest-CLI escape hatch: Claude, Codex, Gemini, OpenCode, Kimi, Qwen, Droid, Cursor — and **Pi** (pi-acp) — already speak it | Public docs + registry — study [2026-09-03](2026-09-03-guest-tui-agent-state.md) |
| [Devin](https://devin.ai) | Hosted autonomous engineer (Cognition) | **Automations** (triggers → session, ACU/rate caps, activity log, NL-generated config), blocked-and-wake sessions (ADR-0037). Not a runtime or editor bar | Public docs + owner's org UI — hosted, no clone |
| [OpenWiki / docs platforms](2026-09-03-docs-harness.md) | Docs harness study: Diátaxis, Scalar, Vale, Mintlify, Remotion license, HyperFrames, D2 | Public docs completeness/beauty: theme, screenshots pipeline, API reference, prose gate, tutorial videos | Live pages + local receipts, 2026-09-03 |
| [Provider/account managers](2026-09-03-providers-view-v2.md) | Providers-view study: agent IDEs (Kilo, Roo, Zed, Cursor), account switchers (cc-switch, claude-swap), quota monitors (ccusage, CodexBar), credential dashboards (OpenRouter, Vercel, Stripe, Zapier) | Roster row spec, quota inline, credential origin, Verify, blast radius, fallback order | Open repos + live docs, 2026-09-03 |

**Adaptation rule** (same as Cursor): borrow a pattern when it improves agent
or CLI-terminal control. ADR-0069 records the owner's multi-CLI direction:
managed agents remain Pi, while other CLIs get managed terminal launches.
Their protocols and packages need a separate decision before first-class
agent support. ADR-0003 still governs managed Pi agents: user-installed `pi`,
no vendored agent SDKs. The current terminal manager adapts the dedicated
configuration and availability patterns cited in ADR-0069.

## Studies

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
