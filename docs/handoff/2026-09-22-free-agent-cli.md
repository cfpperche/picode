# 2026-09-22 — feat/free-agent-cli

Shipped: the free **New agent** runs any launchable CLI, the third branch of `docs/handoff/open/multi-cli-ade.md` (ADR-0179). Server: `handleAddFreeAgent` takes `cli` and calls `AddAgentWithCLI(FreeWorkspaceID, …)`; `attachAgentTerminal(deps, workspaceID, cwd, agent)` is now shared by the workspace, free and principals paths. `TestAddFreeAgentCLICreatesCLIAgent` covers the rows: guest → free launch terminal on the folder + managed start 400; default → Pi with provider; unknown → 400. Web (both twins): `NewCliPrincipal` gains a free mode (`workspace = { free: true }`, `freeAgentPickSchema` cli + name + optional path, `FolderField`, POST `/api/agents`); the sidebar, Browser, Computer and mobile entry points route to it; the old Pi-only free branch of `submitNew` is removed; `catalogForAgent` no longer forces Pi installed (tests updated). Docs: `docs/architecture/managed-principals.md`, `docs-site/guide/getting-started.md`.

Deliberate simplification: a free Pi agent's provider/model/thinking are no longer chosen at creation — they are set afterwards in Settings, as the workspace path already does.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, go, test-js, build, docs). Live on a scratch instance: creating "Scout" as Claude Code from the sidebar dialog produced `cli=claude-code` and a free launch terminal (`workspaceId ws_free`, cwd the derived work folder) that opened Claude Code's TUI, mode interactive.

Visual review: two passes, desktop 1440×900 and mobile 390×844 (`var/screenshots/free-agent-cli/`). Pass 1 PASS with one debt — the Folder placeholder truncated mid-word — fixed by shortening the placeholder to "Folder (optional)" and moving the hint into the description. Pass 2 PASS, `window.__picodeOverlayAudit()` ok on both.
visual-review: PASS (v2-new-free-agent-desktop.png, v2-new-free-agent-mobile.png; overlayAudit ok; card 5/5)

Pre-existing, not this branch's: the Inspector "⎇ fea…" tab truncates; the sidebar count and the Inspector "Changes" count differ.

The remaining branches and debts live in `docs/handoff/open/multi-cli-ade.md`; nothing is repeated here.
