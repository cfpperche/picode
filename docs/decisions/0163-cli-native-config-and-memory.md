# ADR-0163: Native settings and native memory for every agent CLI

- **Status**: accepted (owner approved the plan and its four recommendations, 2026-09-20)
- **Date**: 2026-09-20
- **Boundary**: persistence — PiCode reads and writes other CLIs' native settings files, and reads (and for one tier edits) the files those CLIs use as long-term memory; process — PiCode runs a vendor's own memory command (`grok memory clear`) on the user's behalf; security model — memory content is user data that never leaves the machine, never enters the change feed and is redacted on read.

## Context

Agent CLIs has eight panes. Launch, Terminals, Sessions and Connectors serve
all nine managed CLIs; Providers, Settings, Keyboard and Packages serve Pi
alone (`CLI_SETTINGS`, `CLI_PROVIDERS` are one-entry lists). A user who
opens `#/clis/claude-code/settings` is told the pane is in development,
while the same CLI's MCP servers are fully manageable next door.

ADR-0150 already crossed this boundary for MCP configuration: PiCode edits
the nine CLIs' native config files with merge-by-key, atomic writes,
re-read-before-write and refusal when the file changed underneath. The
drivers in `internal/connectors` located every one of those files. Settings
is the same boundary, the same files and the same nine CLIs — what is
missing is a schema per CLI and a generic reader/writer for scalar keys.

Memory is new. Six of the nine CLIs now keep long-term memory of their own,
in three incompatible shapes (probed on the owner's machine 2026-09-20):

| CLI | Store | Shape |
|---|---|---|
| Claude Code | `~/.claude/projects/<slug>/memory/` | `MEMORY.md` index + one topic file per memory, YAML frontmatter |
| Hermes | `~/.hermes/memories/` (present, empty here) | `MEMORY.md`, `USER.md`, capped by `memory_char_limit` / `user_char_limit` |
| Omp | agent dir, backend `local` (off here, folder not documented) | `MEMORY.md`, `learned.md`, `skills/<n>/SKILL.md` per the vendor |
| Muse Code | `<ws>/.agents/memory/` | markdown |
| Grok | `~/.grok/memory-v2/{global,workspaces/<slug>}/` | generated `MEMORY.md` + `topics/` + `observations/` + two SQLite files |
| Codex | `~/.codex/memories/` | git-backed `MEMORY.md`, `memory_summary.md`, `raw_memories.md`, with `memories_*.sqlite` beside the folder |

Pi and OpenCode have no native memory. Antigravity's CLI keeps conversation
transcripts and knowledge items under `~/.gemini/`, but no vendor
documentation confirms a CLI-managed memory, so PiCode must not claim one.

The switch that turns each store on or off is a key in that CLI's own
config file, so the memory pane's toggle is a settings write, not a second
mechanism.

`docs/benchmarks/2026-09-13-ai-memory.md` already refused a long-term memory
server as a replacement for the native session handoff, and ADR-0150 refused
a PiCode-owned MCP registry injected at launch. Both refusals apply here
unchanged: each CLI's own store stays authoritative.

## Decision

**Settings.** Every agent CLI's Settings pane at `#/clis/<cli>/settings`
manages that CLI's native configuration through a per-CLI declaration behind
one generic file driver in `internal/clisettings`: the config paths per
scope, the document format (JSON, JSONC, TOML, YAML) and a typed field list
(dotted key, kind, options, help). The driver reads the document, reports
which keys each layer sets, and writes a scalar patch by merging into the
parsed document and re-serialising, preserving every key and structure it
does not touch. Unknown keys survive, writes are atomic (tmp + rename), the
file is re-read immediately before the write and a file that changed
underneath is refused. A file the parser rejects is reported and never
overwritten; the API takes a `force` flag for that case and no pane sends it
yet. A CLI declares exactly the layers
it has: a CLI with a single config file gets no layer switcher and no
invented project scope. Pi keeps its existing editor, API and trust rules
(ADR-0101) untouched; this decision amends only the Pi-only registry.

**Memory.** A Memory pane at `#/clis/<cli>/memory` reports each CLI's native
memory in one of four honest tiers declared by a driver in
`internal/climemory`:

- **editable** (Claude Code, Hermes, Omp, Muse Code) — plain markdown the
  vendor expects a human to edit: list, open, edit and delete;
- **read-only** (Grok, Codex) — a generated index backed by a sidecar
  database or a git baseline, which PiCode reads and never writes; clearing
  goes through the vendor's own command;
- **none** (Pi, OpenCode) — one line saying the CLI has no native memory;
- **unknown** (Antigravity) — one line saying PiCode cannot confirm one.

The toggle that enables or disables a store writes that CLI's own config key
through the settings driver above. Memory content is read on demand, never
copied into SQLite, never published on the change feed (ADR-0048 carries the
event, not the text) and redacted for secret-shaped values on read the way
`internal/mcp` redacts them today. A store whose scope PiCode cannot resolve
for the current workspace reports unknown rather than showing another
project's memory.

**Refused in this decision.** PiCode does not keep a memory store of its own
synced into the CLIs; it does not copy a memory from one CLI into another
(ADR-0088's create-only write into a guest store was a narrow, verified
exception for sessions, and a cross-CLI memory transfer would need its own
decision); it does not hand-edit a generated index; and it offers no generic
JSON editor for a CLI without a declared schema (ADR-0099 §5).

## Consequences

Users get one settings surface across the eight guest CLIs and one place to see
what their agents have remembered, without leaving the browser. Coverage is
uneven by design and the pane shows only what each CLI declares: Antigravity
has a single scalar, OpenCode three, Muse four. Every CLI
adds a declaration — paths, format, fields — and a golden-file test, and a
vendor that renames a key breaks that row at read time, visibly, rather than
silently writing a key the CLI ignores. PiCode becomes a second writer of
files other programs write while they run: the re-read-before-write contract
and the HOME-swapped test suite are load-bearing, and because a CLI may be
mid-turn in one of PiCode's own terminals, the panes name the running
terminal before a write instead of racing it.

The tier split means the Memory pane's capability is uneven by design: Grok
and Codex will never gain inline editing here, because their own products
treat those files as generated. That is honest but will read as a gap. The
scope-identity rule (Claude Code derives its directory from the git
repository, Grok from a name-plus-hash) means a workspace PiCode cannot map
shows nothing rather than something plausible; a vendor changing that
derivation turns a working pane into an empty one until the driver follows.

Deleting memory is destructive and `grok memory clear --all` has no undo, so
every removal is scoped, names its blast radius and is confirmed
(ADR-0099 §3).

## Alternatives considered

- **One PiCode memory store synced to every CLI.** Rejected: it repeats the
  refusal in `2026-09-13-ai-memory.md` and ADR-0150 — it breaks every launch
  made outside PiCode and duplicates each vendor's own loader.
- **A generic JSON/YAML/TOML text editor per CLI.** Rejected by ADR-0099 §5
  and the 2026-09-12 study: the file itself is the escape hatch, opened in
  the Files view; a second competing editor in the pane is not.
- **Merging settings into the Launch pane.** Rejected: the 2026-09-17 study
  refused it on purpose. Launch composes one invocation's arguments; Settings
  writes the file every invocation reads, including launches made outside
  PiCode. The panes name each other instead.
- **Treating every memory store as editable.** Rejected: Grok's index says
  "Generated by Grok. Do not edit this file directly" and keeps its truth in
  `index.sqlite`; Codex's directory is a git baseline with a sidecar SQLite.
  Writing them would corrupt state PiCode does not own.
- **Per-CLI bespoke settings panes.** Rejected for the same reason ADR-0150
  rejected it for connectors: nine near-identical surfaces to keep in sync
  for no capability gain.
