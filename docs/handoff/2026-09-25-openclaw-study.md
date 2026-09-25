# 2026-09-25 — feat/openclaw-study: the OpenClaw 2.0+ study, and why PiCode stays out

One commit, docs only (54dce326b): `docs/benchmarks/2026-09-25-openclaw.md`, plus a row and a list entry in `docs/benchmarks/README.md`. The owner asked whether PiCode should integrate OpenClaw 2.0+.

**Finding.** With 2.0 (v2026.8.1), OpenClaw stopped being an agent PiCode could host and became a peer control plane: a project sidebar, managed worktrees, a full web terminal, a Changes panel, an approvals Inbox, Swarm, and ACP harnesses driving the same CLIs PiCode hosts, headless. Its own runtime replaced Pi's agent core entirely — only the `pi-tui` toolkit remains. Recommendation: no catalog row, no managed runtime; the only door worth opening is a no-code `openclaw mcp serve` connector recipe, and only if the owner wants it.

**Gap found.** PiCode's webhooks send fixed headers only (`internal/webhooks/engine.go`) and cannot authenticate to OpenClaw's hooks, which require a bearer header and reject query-string tokens. Fixing it means a stored secret and an ADR-0075 amendment.

**Honesty.** OpenClaw was not installed or run — this machine's Node 24.11.1 is below the required 24.16. Door costs C3–C5 are extrapolated from the Hermes footprint, not measured.

## Next up

- Owner: publish the C1 `openclaw mcp serve` connector recipe in docs-site, or not.
- Owner: is reach into WhatsApp/Telegram wanted — a bearer on webhooks (ADR-0075 amendment) vs a PiCode-owned channel.

Verified: `make close` green, scoped (metadata); docs-check ok. visual-review: n/a — no UI. Nothing deployed.
