# Agent instructions (AGENTS.md and its kin)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

All nine agent CLIs read `AGENTS.md`, and no two decide the same way which
file wins, how far up the tree they look, which personal file counts, or how
much of a long file survives. The **Instructions** page (`#/instructions/<workspaceId>`)
answers, per workspace, what each CLI reads for a session started there and
why it leaves the rest out. The study that measured every rule is
[docs/benchmarks/2026-09-23-agents-md.md](../benchmarks/2026-09-23-agents-md.md);
the owner's decisions of 2026-09-23 are in `docs/handoff/open/agents-md.md`.

## One rule per CLI, declared

`internal/cliinstructions` holds one rule per CLI (`rules.go`), in the Agent
CLIs catalog's order. Each rule is code, because the nine do not share a
shape a table could hold — Hermes loads one kind of file per session, Omp
keeps one file per folder depth by source priority, Claude Code switches
`AGENTS.md` off when any `CLAUDE.md` exists. Each rule names where it was read
and at which version (`CLIInfo.Source`), and the tests pin every measured case
on sentinel fixtures (`cliinstructions_test.go`): both files in one folder,
the prose pointer, imports and links, `CLAUDE.local.md`, Claude's four modes,
a worktree nested in its main checkout, `.hermes.md`, size limits, Grok's
trust and `.gitignore`, subfolders and the start folder, personal files,
outside a repository.

| Rule reads | From |
|---|---|
| Claude Code's Project instructions mode | `pluginConfigs["agents-md@builtin"].options.instructionFiles` in managed, then user settings |
| Codex's budget and fallback names | `project_doc_max_bytes`, `project_doc_fallback_filenames` in `~/.codex/config.toml` |
| Hermes's cap | `context_file_max_chars` in `~/.hermes/config.yaml`, else 6% of the model's window |
| Grok's trust | `~/.grok/trusted_folders.toml`, exact folder (a trusted parent does not count, measured) |
| git facts | `rev-parse` (repository root, the main checkout of a nested worktree), `ls-files` for subfolders, `check-ignore` |

Cells are `reads`, `on-demand`, `shadowed` (with the winner), `not-read`,
`untrusted` and `unknown`; a file a CLI reads may carry a `cut` in words.
Findings speak only of installed CLIs (the Agent CLIs page's own resolution,
`resolveCLIExecutable`).

## Read-only and on demand

`GET /api/workspaces/{id}/instructions[?start=<folder>]` computes the report
from disk on every call and stores nothing, like the Memory pane (ADR-0163).
It is confined to the workspace folder (`start` must be inside it) and returns
paths, sizes and verdicts — never a file's text. The tab re-reads when it is
shown and on Refresh: instruction files change outside PiCode's store, so no
feed event covers them, and nothing polls.

## What an agent actually read

The matrix is a prediction from the declared rules. Three CLIs also write
down what a session loaded, and `observed.go` reads that record for the
session PiCode pinned on each of the workspace's terminals (ADR-0084,
`TerminalLastSession.Path`):

| CLI | Record |
|---|---|
| Claude Code | the transcript's `Contents of <path> (…instructions…)` entries, before the first answer (later lines are conversation and may quote any path) |
| Codex | the rollout's `# AGENTS.md instructions for <folder>` |
| Grok | `agents_md_files` in the session's `prompt_context.json` |

The report carries them as `agents`; the tab lists them under "What agents
here read". The other six CLIs keep no such record, so they get no line
there rather than the prediction under another name.

## Fixes, written only on confirmation (ADR-0204)

A finding may carry a fix id. `internal/cliinstructions/fix.go` turns the
id into the exact change, computed from the files on disk, and the tab shows
it as a diff. `GET /api/workspaces/{id}/instructions/fix?id=` returns the
change. `POST` writes it only when every file still has the hash the person
saw; otherwise it answers 409 with the fresh change, shown in place. A fix
that no longer applies answers 404.

| Fix id | Offered by | Writes |
|---|---|---|
| `bridge:<CLAUDE.md>` | the prose-pointer finding | `@AGENTS.md` on top (or `@../AGENTS.md` from `.claude/`); a file that holds only the pointer sentence becomes the import line |
| `personal:<file>` | the "not in .gitignore" finding; the tab's **Add personal file** menu | creates `CLAUDE.local.md` or `AGENTS.override.md` empty when missing (never overwrites), and adds its name to the `.gitignore` in the same folder |

Writes are confined to the workspace. A symbolic link, or a folder that
leads out of the workspace, is refused. Files are written atomically. Git is
never run: the change waits in the Git tab.

## Where it shows

| Place | What |
|---|---|
| Workspace `…` menu ▸ **Instructions** | the page `#/instructions/<workspaceId>`, over the tabs like Agent CLIs: findings, then the matrix of files × CLIs; a cell's reason below the table |
| **New agent** dialog | one line for the picked CLI (`createLine`, `web/shared/domain/instructions.js`) |

| CLI page ▸ **Settings** ▸ Instructions | the settings that change what a CLI reads: Claude Code's Project instructions mode (user layer only — Claude ignores it in project settings), Codex's `project_doc_fallback_filenames` and `project_doc_max_bytes`, Hermes's `context_file_max_chars`, OpenCode's `instructions`. Declared in `internal/clisettings/specs.go` like every other row; a finding whose fix is one of them links there |

Desktop only for now; the phone app has no Instructions view yet.
