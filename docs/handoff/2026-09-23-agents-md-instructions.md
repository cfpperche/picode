# 2026-09-23 — feat/agents-md-instructions: which instruction files each CLI reads, per workspace

Slice 1 of the AGENTS.md study, as the owner approved it: read-only, no ADR, a workspace tab.

**Engine.** `internal/cliinstructions`: one rule per CLI in `rules.go`, each naming the version it was measured at, pinned by sentinel-fixture tests (both files in one folder, prose pointer, import and link, `CLAUDE.local.md`, Claude's four modes, a worktree nested in its main checkout, `.hermes.md`, size limits, Grok trust and `.gitignore`, subfolders, personal files, outside a repository). Two rules were settled with no model call: Grok's personal files (`~/.grok/AGENTS.md` and `CLAUDE.md`) and exact-folder trust, via `grok inspect`. One cheap Claude probe showed it reads a `CLAUDE.md` above the repository root; that probe hit the owner's usage limit, so none followed. `GET /api/workspaces/{id}/instructions[?start=]` reads on demand, is confined to the workspace, and never returns file text.

**UI.** Workspace `…` ▸ Instructions: findings (one line + one action), then files × CLIs with a reason for the picked cell; the New agent dialog shows one line for the picked CLI. Two findings were false positives on this repository and are fixed: `docs-videos/` holds the same text as CLAUDE.md and AGENTS.md, and a subfolder's own CLAUDE.md is now named as the reason.

Docs: `docs/architecture/cli-instructions.md`, `docs-site/guide/instructions.md`, changelog fragment.

Verified: `make ci-scoped` and `make close`. Visual review on scratch `instr` (screenshots read in subagents) PASS, overlayAudit ok on the workspace menu and the New agent dialog. At 1440 with the right-hand rail open, 9 CLI columns do not fit the card. The first review found no cue that the table scrolls sideways. A fade and a pinned file column were tried; the pinned column left half-hidden cells that read as stray text and was taken out. What shipped: a line "Scroll the table sideways to see all 9 CLIs." when it overflows, a fade at the edge, and group labels that stay readable while scrolling. Blind spots: the phone app, and the Windows shell. Nothing deployed.
