# Diff review + browser editor — next roadmap

- **Date:** 2026-08-27
- **Status:** plan. Track D shipped. ADR-0015 accepted.
- **Why:** PiCode without a file in the browser is a mismatch. Chat diffs
  are view-only. Cursor bar item 3 (per-hunk Keep/Undo) never landed.
- **Study:** [Herdr](../benchmarks/2026-08-27-herdr.md) — runtime peer,
  not the editor bar. Editor bar stays Cursor (hunks) + our cwd files.

Worktrees (parallel isolated agents, Orca) are **later**. Do not start
them in this track.

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **E1 — open the file** | **shipped** |
| 2 | **E2 — edit and Save** | **shipped** |
| 3 | **E3 — hunk Keep / Undo** | after E2 (needs the write path) |
| 4 | **E4 — this turn's files** | after E3 |

Do not start llama installer, mobile parity, worktrees, broker, ACP,
LSP, or a file explorer as home.

## Refuse

| Temptation | Why not |
|---|---|
| Monaco / VS Code web | IDE in a box; CodeMirror 6 is the file widget |
| LSP / IntelliSense / outline | we are not a worse IDE |
| Explorer as the product | open from a diff, `@`, or E4. Browse API already exists; it is not the home |
| New hash `#/files` | chat stays; file is a pane on `#/agent/<id>` |
| Auto-Keep every hunk | review is the point |
| Rewrite the file on Undo if it drifted | one line + **Open** (E1) |
| Worktrees / Orca | later; Herdr `worktree.*` is layout, not our unit |
| Replace tmux with Herdr | ADR-0002 stands |

## Where it lives today

| Thing | Path |
|---|---|
| View-only hunks | `web/src/lib/diff.js`, `Conversation.jsx` `DiffHunks` |
| Tool card path | `fileChangeFromTool` on `edit` / `write` |
| List / browse | `GET /api/agents/{id}/files`, `.../browse` |
| GET file | **images only** (`handleAgentFile`) |
| Pins editor | TipTap — stay there; not for code |

## E1 — open the file

Click the path on an `edit` / `write` card. A pane in the workspace
shows the file (read). Empty: "No file open." Click a diff.

E1 **shipped** (2026-08-27). `GET /api/agents/{id}/text?path=` reads under
cwd (absolute paths that stay inside cwd work). Binary or too large: one
line, no editor. Path escape: 400. Pane is absent until you click.

| # | from | file | action |
|---|---|---|---|
| 1 | diff path | text, exists | open pane |
| 2 | diff path | missing | "That file is gone." |
| 3 | diff path | binary / huge | one line, no editor |
| 4 | no file yet | — | "No file open." |
| 5 | path outside cwd | — | refuse (existing `relUnderCwd`) |

## E2 — edit and Save

Same pane becomes CodeMirror 6. **Save** writes the cwd file (`PUT`).
Dirty state on the pane. Conflict if mtime changed since open: confirm
or **Open** again.

E2 **shipped** (2026-08-28). CodeMirror 6. **Save** and Ctrl/Cmd+S. Dirty
dot on the name. Stale disk → confirm **Open**. Close with dirty → Discard.
No format-on-save.

| # | dirty | mtime | action |
|---|---|---|---|
| 1 | no | same | Save no-ops |
| 2 | yes | same | write, clear dirty |
| 3 | yes | changed on disk | confirm; do not clobber |
| 4 | yes | user closes pane | confirm discard |

## E3 — hunk Keep / Undo

Each add/del group on the chat card gets **Keep** (already on disk —
no-op besides marking) and **Undo** (put the old lines back). The
agent already wrote the file; Undo is the review.

If the file no longer contains the new side: "File changed." + **Open**.

| # | hunk | file matches new side | action |
|---|---|---|---|
| 1 | Keep | * | stay; mark kept |
| 2 | Undo | yes | write old side |
| 3 | Undo | no | "File changed." + Open |
| 4 | whole-file `write` | yes | Undo = delete or restore previous; if unknown, Open |

Do not invent git checkout. Disk is the source. Git is later (Orca).

## E4 — this turn's files

A short list of paths the current turn edited. Click opens E1. Empty:
no extra well (the chat already has the cards). No "0 files" badge.

## Primitive

CodeMirror 6. Justification: popular file editor in the browser;
syntax without LSP. Monaco lost in ADR-0015. TipTap stays pins.

## Later (not this track)

- Worktrees / parallel agents (Orca + Herdr `worktree.*`)
- Comment on a hunk (OpenADE)
- Cross-agent change view (Cursor #3 remainder)
- File tree as a sidebar tab
