# ADR-0087: User-initiated prompt door for Agent CLI terminals

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
   attachments per prompt, 4 MB each. Touch `.gitignore` for that folder
   when a `.gitignore` already exists and lacks it. Return a cwd-relative
   path. Bytes are not stored in SQLite.
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
