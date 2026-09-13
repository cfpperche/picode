# Snippets — implementation plan

Owner confirmed 2026-09-13 (design session). ADR-0130. Architecture:
[docs/architecture/snippets.md](../architecture/snippets.md).

One worktree `feat/snippets`. Eight stacked commits, one fast-forward.

## Owner calls

| # | Decision |
|---|---|
| 1 | User-facing name **Snippets** (code id `snip`) |
| 2a | Kind `shell` may PasteText+Enter into a plain `isShell` pane (snippet-run door, confirm) |
| 2b | Kind `prompt` into a plain shell is **409 `kind`** in v1 |
| 3 | SQLite-only in v1 |
| 4 | No v1 export to `~/.pi/agent/prompts/` |
| 5 | Placeholders `{{name}}` / `{{name=default}}` + `{{{{` escape |
| 6 | PageFrame 1240px |

Do not amend ADR-0089. Do not write vendor command dirs. No Global chord
in v1.

## Commits

1. **Parser + ADR** (this commit) — `internal/snips` Parse / Expand / Slug.
2. Store + `/api/snips` CRUD + `snip.*` feed + OpenAPI. Kind `shell` 409
   `unimplemented` on `/run` until commit 6.
3. Desktop `#/snippets` studio (Prompt kind only until 6).
4. `/expand` + `/run` into managed agents; both Composers `run === "snip"`;
   palette `kind: "snippets"` vs `kind: "snip-run"`.
5. Kind `prompt` through ADR-0089 when `termHoldsCLI && !isShell`;
   handler-time `isShell` re-check (B6b).
6. Kind `shell` snippet-run door (needs 2a, already yes).
7. Mobile More / read / edit / Terminal actions / composer.
8. `docs-site/guide/snippets.md` + `/snip` in commands.md.

## Chrome labels

| Entry | Label |
|---|---|
| Composer splice | Insert snippet |
| Composer `fireSend(spliced)` / palette → agent | Send snippet |
| Palette / term-menu → CLI TUI | Send to terminal |
| Kind `shell` | Run command |

Fence-runner stays **Run**.
