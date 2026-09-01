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

**Adaptation rule** (same as Cursor): borrow a pattern only if it makes
*Pi agent control* better. We are not a multi-runtime harness (t3code)
and not a pairing-relay fleet (paseo). ADR-0003 stands: user-installed
`pi`, no vendored agent SDKs.

## Studies

- [2026-09-01 — Supervising coding agents from a phone](2026-09-01-mobile-agent-supervision.md)
- [2026-09-01 — LLM observability dashboards](2026-09-01-llm-observability-dashboards.md)
- [2026-08-30 — Copying out of a browser terminal backed by tmux](2026-08-30-web-terminal-clipboard.md)
- [2026-08-27 — Herdr](2026-08-27-herdr.md)
- [2026-08-24 — Adopt t3code / paseo / Cursor](2026-08-24-adopt-t3code-paseo-cursor.md)
