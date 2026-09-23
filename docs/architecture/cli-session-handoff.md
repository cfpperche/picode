# Cross-CLI session handoff (ADR-0088)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Any session of an Agent CLI can continue in another one from that CLI's
Sessions pane ("Continue in <CLI>…") **and** from the terminal that is
running it: the sidebar `⋯` menu, the Agent CLIs Terminals `⋯` menu, and
the pane's right-click menu. The source is the terminal's pinned
conversation (`lastSession`, ADR-0084) **or**, on a managed Pi agent in
the terminal, that agent's `sessionPath` (path-only is enough). It is still
a handoff, never a live switch — the source terminal keeps running and a
new terminal (or a stopped Pi agent) opens on the target. Terminals whose
CLI never registers a native runtime (Muse Code, Antigravity — no PATH
wrapper by design) pin the same way at stop time: the latest session the
vendor store holds for that folder past the terminal's creation, so
Continue in… and resume work there too. A session predating the terminal
never pins (no stealing across terminals in a shared folder).

Continue in from an interactive Pi TUI always takes the live-confirm path
once `liveHolderFor` sees `tmux.HasSession(picode-<agentId>)` (managed RPC
`Runtime.Get` is no longer the only live detector for `cli == "pi"`). The
dialog is the existing 409 `live` + `force=true`. Pin cwd is
`workPath` else the workspace folder else the pane cwd.

`internal/transcript` is the portable model —
ordered events (message, tool call, tool result, thinking, compaction,
context), a header and a manifest of what a reader could not carry.
`internal/clisession` adds optional capabilities next to `Source`:
`Reader` (native → timeline), `Writer` (timeline → a new native session,
create-only, atomic, re-read before it counts) and `Prompter` (launch
arguments for an initial prompt). They are discovered by type assertion,
never by a switch, and `GET /api/clis` advertises them as
`sessions: {list, read, write, prompt}`; the web derives the targets from
that. Every CLI with a full adapter reads and writes; prompters exist for
every CLI but Hermes, which cannot be started with an initial prompt.

**List-only sources** (`list` without `read`) are the other end of that
range: a CLI whose history is on disk before it has an adapter. (Muse Code
and Antigravity both graduated past this in the launch-parity project:
Muse reads its index plus the session logs it points at, Antigravity its
summaries index plus the per-conversation transcripts, so both are full
handoff sources and brief targets; the paragraph stays for the next CLI
that arrives list-only.) Muse Code reads `~/.local/share/muse/session-index.db`
(or `$XDG_DATA_HOME/muse/...`), the index Meta's launcher keeps over its own
session logs — title, first prompt, workspace root, model, prompt count and
timestamps are columns there. Its `ResumeArgs` are `--resume <session-uuid>`.
Antigravity reads `~/.gemini/antigravity-cli/conversation_summaries.db` and
resumes with `--conversation <id>`. The pane list follows the capability,
not the adapter: `cliPanes` adds Sessions when `sessions.list` is true.
Omp reads its own `~/.omp/agent/sessions` JSONL — pi-family schema v3 with
omp's cwd-bucket encoding, `title` records outside the entry chain and
provider-qualified `model_change` records — verified against real files
and against a hand-written minimal file the CLI resumed live (omp 18.2.4,
2026-09-17); its `ResumeArgs` are `--resume <id>`.

Writers publish two ways. Claude Code, Codex, pi (as an adopted managed
agent), Grok, Muse Code, Antigravity and Omp get a new session artifact
created in their own store, atomically and never over an existing one —
Muse as an index row plus a session log of plain records (the exporter
dispatches on the line prefix, so `schema_version` leads and the session
id parses as a UUID), Antigravity as a summaries row plus the brain
transcript plus the (possibly empty) per-conversation store file, Omp as a
schema-v3 JSONL in its cwd bucket at the measured-minimal shape (header,
optional omp-shaped title record, one provider-qualified `model_change`,
chained messages; thinking never written). OpenCode and Hermes are SQLite
stores their CLI holds open, so a writer publishes through the vendor's
own importer instead (ADR-0094): `WriteRequest.Run` executes the CLI in
the session's folder with its configured executable and environment
(`opencode import`, `hermes sessions import --from claude`). Every route
ends in a read-back, block for block where the importer lands what it was
given and by arrival where it rewrites, as Hermes does.

`POST /api/clis/{cli}/sessions/handoff/preview` and `…/handoff` share one
plan: window (since the last compaction, or all), repair (every tool call
answered), prepare (no thinking, no injected context), a handoff note as
the first user message, then the target's Writer — or, in brief mode, a
deterministic markdown brief under `<dataDir>/handoffs/<id>/brief.md` and
the target started with a one-line prompt. Lineage lives in
`session_handoffs` (event `session.handoff`) and is shown on both session
listings. The installed target version comes from the setup check (run on
demand); an unknown format refuses native and keeps brief. A source over
the 64 MB read cap drops native and offers a brief of the last 64 MB of
the file (the recent turns); executing native on that source is still 413.

Where the handoff opens is a choice when the target is also a managed
agent — `sessions.agent` in `GET /api/clis`, pi only today (ADR-0040).
`landing: "agent"` adopts a stopped managed agent: for native it owns the
written session, for brief it starts with the brief queued as its first
prompt. `landing: "terminal"` opens a CLI terminal; for pi native the
session is written into pi's own cwd bucket and the terminal resumes it
through `--session`. Omitting `landing` keeps the shipped default (pi
native → agent, everything else → terminal), and a CLI without an agent
side refuses `landing: "agent"` with 400.

A long-term memory server in the same folder (hooks that inject a next-
session brief) is a different layer: it does not replace Native session,
and Continue does not wrap launches through it. Study:
[2026-09-13-ai-memory](../benchmarks/2026-09-13-ai-memory.md).

## Fork

**Fork agent…** (sidebar `⋯` of a CLI agent whose terminal has a pinned
conversation) starts a new agent of the **same** CLI on a copy of that
conversation, with a task of its own, while the source keeps running. It
never goes through the portable timeline: `clisession.Forker` composes the
vendor's own fork, and `GET /api/clis` advertises it as `sessions.fork`.

| CLI | Launch | Copy's id |
|---|---|---|
| Claude Code | `--resume <id> --fork-session --session-id <new> <task>` | pre-assigned, pinned at once |
| Grok | `--resume <id> --fork-session --session-id <new> <task>` | pre-assigned, pinned at once |
| Codex | `fork <id> <task>` | pinned on its first turn |
| OpenCode | `--session <id> --fork --prompt <task>` | pinned on its first turn |
| Omp | `--fork <file\|id> <task>` (parsed, not in `--help`) | pinned on its first turn |

Hermes, Muse Code and Antigravity fork only inside their TUI; Pi agents own
their session file. None of them advertises a fork yet
(`docs/handoff/open/agent-fork.md`).

`POST /api/agents/{id}/fork-agent` takes `{name, prompt, workPath, files,
paths}`. The source's launch overrides carry over with the fork recipe as
the arguments (resume's rule). Attachments are staged in the fork's folder
(`.picode/drop/`, as the attach bar does) and named in the task as `@path`;
the task is one launch argument, so line breaks become spaces. The row lands
in `session_handoffs` with `mode = "fork"`, its manifest naming the source
agent (`sourceAgentId`, `sourceAgentName`). The agent lists
(`GET /api/workspaces`, `GET /api/agents`) carry `forkedFrom {agentId,
name, gone}` from those rows — the source's current name while it exists,
the fork-time name once removed. The desktop row gives it a line of its
own under the title block ("fork of <name>", the checklist line's inset —
beside the status chip the subtitle has ~100px); the phone's Work list
appends it to the CLI line. Both refetch when a
`session.handoff` event with mode `fork` arrives, since the row lands just
after the agent.

The dialog's task field is the terminal's attach composer
(`AttachComposer.jsx`, shared with `TermAttachBar`). **New worktree** is the
default in a repository: the dialog composes `create-worktree-branch`
(`git worktree add -b <slug> <root>/.worktrees/<slug> HEAD`), hands it to
the git door (ADR-0096, typed into a visible shell, run only when the
ADR-0078 interlock allows — and ADR-0202 lets this exact command past
the busy-repository check, since it touches no file another agent is
working on), waits for the folder in the graph, then forks into it. When
the command is only typed and waits for the person's Enter (a door refused
it for another reason), the dialog closes — a modal
over that terminal would hide the key it asks for — a toast says what to
press, and the fork starts in the background as soon as the worktree
appears (ten minutes at most; a failure after that is a toast). The git
shell is never an agent's terminal: agent-owned and `launchCli` terminals
are skipped, and an "Agent CLI" refusal falls back to a fresh shell.
**Same folder** shares the source's files. The task composer takes pasted
screenshots and files like the terminal's attach bar.
