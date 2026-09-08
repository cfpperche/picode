# ADR-0089: User-initiated prompt door for Agent CLI terminals

- **Status**: accepted (owner, 2026-09-06: execute the 2026-09-06 study's
  recommended path)
- **Date**: 2026-09-06
- **Amends**: ADR-0078 (Inspector still must not type into a CLI TUI;
  the *user's* Send on that terminal's attach bar may paste a prompt);
  ADR-0002 (PasteText is an owner-accepted exception, as in ADR-0060,
  now also on `picode-sh-*` — not a send-keys control protocol)
- **Does not change**: ADR-0069 / 0079 / 0003 (CLIs are not managed
  agents, chat replay stays Pi-only, no vendor SDKs)
- **Study**: [2026-09-06 — CLI terminal attach](../benchmarks/2026-09-06-cli-terminal-attach.md)

## Context

Agent CLIs (Pi, Claude Code, Codex, Grok, Hermes Agent) run as terminals.
A phone cannot paste an image into xterm (`clipboard.readText` only).
Claude Code Remote Control already sends photos into a local TUI; Claude
and Codex treat a **file path in the prompt** as the portable attach
method. Promoting those CLIs to managed agents (ACP) is a separate
project. The Inspector must still never steal a live CLI prompt with git
`type`/`run`.

## Decision

A **prompt door** exists on Agent CLI terminals only (plain shells wait).

1. **Stage.** `POST /api/terminals/{id}/drop` writes a regular file under
   the terminal's recorded cwd, in `.picode/drop/`, cwd-confined. Images
   are png/jpeg/gif/webp; other artifacts are allowed as files. Caps: 4
   attachments per prompt, 4 MB each. A nested, uncommitted
   `.picode/.gitignore` (`drop/`) keeps the folder out of git — never the
   project's own tracked `.gitignore` (2026-09-07 amendment below). Return
   a cwd-relative path. Bytes are not stored in SQLite. Anything older
   than 7 days is swept on the next drop into the same project — no
   daemon; nothing else ever removes a staged attachment.
2. **Send.** `POST /api/terminals/{id}/prompt` `{message, paths}` builds
   one bracketed paste (caption plus `@path` lines) and `PasteText`s it
   into `picode-sh-<id>`, then Enter. Proof is honest: tmux accepted the
   paste. The UI says "Sent to the terminal", never "the model saw it".
3. **Who may type.** Only this user-initiated Send. Inspector Ask, type
   and run stay refused on a pane with CLI presence (ADR-0078).
4. **UI.** Desktop: attach bar on `TermSurface` when the terminal is an
   Agent CLI. Mobile: header paperclip → sheet (Photos / files /
   workspace), not a second composer on the extra-keys row (ADR-0044).
5. **Managed Pi composer** separately opens a device file picker
   (Photos / camera / files) into the existing `images[]` / chip path.
   That picker does not use this door.

### Decision table

| Conditions | Action |
|---|---|
| Terminal missing / not running / pane dead | 409; no write |
| Path escapes cwd | 400; no paste |
| Over 4 MB, over 4 attachments, or not a regular file | 400; keep what fits |
| Live CLI TUI, user hit Send, no in-flight prompt | Stage if needed, PasteText + Enter |
| In-flight prompt for this terminal | 409 busy |
| Inspector type/run/Ask on a CLI pane | Still refused |
| Managed Pi composer, user picks from Photos | Existing `prompt.images`; no terminal paste |

## Consequences

Easier: a phone can attach a screenshot to Claude/Codex/Grok/Hermes/Pi
without a fake chat. Harder: paste can land in an open TUI draft (the
same cost ADR-0060 accepted). Wrong if we treat tmux accept as model
delivery, or if Inspector git starts using this door.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Tell the user to paste into xterm | Text-only; this is today's failure |
| Replace the TUI with PiCode chat (ACP) | First-class CLI agents; separate ADR |
| Swap a Pi CLI for managed RPC | Owner rejected that for Inbox (0059→0060); useless for Claude |
| OSC 52 clipboard read | Guest CLIs must not read the system clipboard |

## Amendment (2026-09-07): the project's own `.gitignore` stops being touched

The owner noticed `.picode/drop/` sitting in a project's file tree and
asked whether it should live in PiCode's global data dir instead
(`~/.picode`). It cannot: the door pastes a **path**, not bytes, and the
CLI reads that path itself, typically confined to its own cwd — a path
outside the project risks landing outside whatever sandbox the CLI reads
files under. The folder stays in the repo on purpose.

Two real defects surfaced chasing that question, both in the original
"touch `.gitignore` for that folder when a `.gitignore` already exists"
line above: a project with **no** root `.gitignore` left every staged
attachment fully untracked and visible to `git status` forever, and a
project that had one got a silent, uncommitted `.picode/drop/` line
appended to a file the user tracks — every attach dirtied `git status`
until someone committed or reverted it. Fix: a nested `.picode/.gitignore`
(`drop/`), created once, unconditionally, never touching the project's
own file.

Also: nothing in this ADR gave a staged attachment a lifetime. The CLI
reads its path once, at paste time, and nothing deletes the file after —
a project accumulates every image ever attached, forever. `dropMaxAge`
(7 days) is now swept opportunistically on the next drop into the same
project; no daemon, no ticker to forget to wire up.

## Amendment (2026-09-08): a pi terminal can be asked, through its receiver

Owner's decision, 2026-09-08, after ADR-0096 shipped the graph's three doors:
the owner works in pi **as an Agent CLI terminal** for now (the managed chat
still has bugs), and the graph's *ask* door had nobody to ask there — it
reached only agents of the store.

Rule 3 above ("Only this user-initiated Send") gains one named exception,
and only one: **a terminal hosting pi whose ADR-0060 receiver has said hello
within the TTL and named the session it is showing** may be asked by the git
graph and the Inspector — `POST /api/terminals/{id}/ask {text, root}`. The
prompt travels as a one-shot reply file under `tui-inbox/term-<id>/`, the TUI
submits it through `pi.sendUserMessage`, the receiver acks, and the JSONL
row is the proof, exactly as for an interactive agent. Nothing is ever pasted
or typed into the pane: no receiver, no session, another repository, a
message already in flight, pi not running — each is a 409 that names itself
(`no-receiver`, `no-session`, `moved`, `busy`, `stopped`), and a terminal
hosting any other CLI, or a plain shell, answers `cli` as before.

What made it possible without a new mechanism: the receiver extension now
takes its identity from `PICODE_TERM_ID` when it has no `PICODE_AGENT_ID`,
and the pi wrapper injects it (`-e`) beside the activity extension, so a pi
launched as a terminal carries the same receiver an interactive agent does.
Its hello carries the session file — the one fact a terminal, unlike an
agent, has no record of. Provenance is an event (`terminal_ask_delivered` /
`terminal_ask_failed`), not a task: `tasks.agent_id` references `agents`,
and a terminal is not one.

The graph lists such terminals as occupants of their worktree
(`kind: "terminal"`, with the server's own presence as `live`), offers
**Open \<name\>** into the pane, and **Ask \<name\>** in the action form.
Claude Code, Codex, Grok, Hermes and OpenCode terminals stay exactly where
rule 3 left them: no receiver, no ask.
