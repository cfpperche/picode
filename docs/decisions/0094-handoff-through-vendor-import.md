# ADR-0094: A handoff writer may publish through the target CLI's own import command

- **Status**: accepted (owner asked for OpenCode to join the handoff
  surface, 2026-09-07; the mechanism was chosen after finding that both
  OpenCode and Hermes Agent ship import as a first-class command, and was
  proven against both binaries the same day)
- **Date**: 2026-09-07
- **Extends**: ADR-0088 (cross-CLI session handoff)
- **References**: ADR-0093 (install a missing CLI by orchestrating the
  vendor's own command), ADR-0079, ADR-0069

## Context

ADR-0088 gave a handoff writer one way to publish: create a new session
artifact inside the target CLI's own store, atomically, never touching
what is already there. That fits Claude Code, Codex and pi, whose stores
are directories of JSONL files discovered by scanning.

OpenCode is the sixth Agent CLI and does not fit it. Its store is a single
SQLite database at `~/.local/share/opencode/opencode.db` that the running
CLI holds open, with sessions spread over `session`, `message` and `part`
rows plus an event log. Writing those rows behind the CLI's back is
exactly the class of write ADR-0088 refused to make: not a new file in a
scanned directory, but an edit inside a live database whose invariants we
would be guessing at.

Hermes Agent has the same store shape: one SQLite database the running
agent holds open, which is why ADR-0088 left it a source only.

Neither needs the exception. Both ship the command we want, verified on
real installs (OpenCode 1.18.29, Hermes Agent 0.21.0):

- `opencode export [sessionID]` prints a session as JSON, and
  `opencode import <file>` reads that JSON back and reports the id it
  landed.
- `hermes sessions import [--from claude|codex] <path>` pulls a
  conversation started in Claude Code or Codex into the Hermes store "so
  it can be resumed with `hermes --resume <id>`", and says in its own help
  that "the foreign files are only read, never modified".

Probes against isolated stores confirmed the behavior each writer needs.
OpenCode keeps the ids the file declares, keeps the title, and files the
session under the folder the command runs in rather than the `directory`
recorded in the file. Hermes assigns its own id and reports it, stamps
`source = claude-code` on the row, and folds tool calls into the
assistant's text — it rewrites what it imports.

## Decision

A handoff writer may publish by invoking the target CLI's own import
command instead of creating a file in its store.

`clisession.WriteRequest` gains `Run`, a bounded way to execute the target
CLI in the session's folder with the executable and environment the CLI's
launch settings already define. The server supplies it; a writer that
needs it and does not get it refuses with `ErrNoRunner` rather than
falling back to writing the store. `Run` never starts a conversation: it
runs the subcommand the writer names, with a timeout and bounded output,
and its result is checked.

The OpenCode writer builds the export envelope (`{info, messages:[{info,
parts}]}`); the Hermes writer renders the timeline as a Claude Code
transcript, which is the shape Hermes' importer reads. Both write to a
temporary file outside every CLI's store, run the command, delete the
file, and then read the session back. The verification is what makes the
handoff true: the file having been accepted is not evidence that the
conversation is there.

**Verification matches what the importer promises.** An importer that
lands the conversation as given is checked block for block, the round-trip
every ADR-0088 writer owes. An importer that deliberately rewrites — as
Hermes does, folding tool calls into text — is checked for arrival
instead: the session exists at the id the command reported, in the right
folder, carrying the conversation's turns. Claiming a stricter check than
the vendor offers would only make the writer lie.

Tool calls map to OpenCode's single `tool` part, which carries the call
and its result together, so a call and the result that answered it become
one part rather than two records. For Hermes they are rendered as turns,
since its importer would flatten them anyway.

A session Hermes imported is a coding session like any other: it is
resumable with `hermes --resume <id>`, so the sessions listing shows rows
stamped `claude-code` and `codex` alongside its own `cli` and `tui`.

## Consequences

Easier: a CLI with an import command joins the handoff surface without any
exception to the never-write rule, and the vendor owns its own store
invariants. The mechanism generalizes: any future CLI that ships an
importer gets a writer with no new policy.

Harder and accepted: publishing now depends on the target binary being
installed, launchable and fast enough, so a native handoff to OpenCode can
fail for reasons a file write never could. Those failures are reported as
they are, with the command's own output clipped into the message. The
import binds the session to the invoking folder, so the runner must set
the working directory rather than trust the file, and PiCode does now run
a guest CLI outside a terminal, which it did not before. That run is
bounded, non-interactive, and named by the writer.

If we are wrong about the envelope, the import refuses or the round-trip
disagrees, and the handoff fails with nothing created; the source is never
touched either way.

## Decision table

| Conditions | Action / observable result |
|---|---|
| Target writer needs `Run`, server supplied it | File written to a temp path, import run in the session's folder, file deleted |
| Target writer needs `Run`, none supplied | `ErrNoRunner`; the brief mode stays offered |
| Import command exits non-zero | Handoff fails with the command's clipped output; nothing recorded |
| Import succeeds, round-trip disagrees | Handoff fails; the session is not claimed |
| Importer rewrites what it lands (Hermes) | Checked for arrival at the reported id, not block for block |
| Importer reports no id (Hermes owns ids) | Handoff fails rather than guessing which session it made |
| Target CLI not installed | 400 before anything runs, as for every other target |
| OpenCode or Hermes as source | Reader over the store, read-only, unchanged by this ADR |
| Tool call and its result | One `tool` part carrying both; `tools: text` renders them as turns |
| Store has no version and no setup check | `ErrUnknownFormat`; brief remains |

Coverage: `TestOpenCodeWritePublishesThroughItsOwnImport` and
`TestHermesWriteImportsThroughItsOwnCommand` (runners that apply the file
exactly as each CLI's importer does, including Hermes' flattening and its
own id, then read the session back), `TestHermesImportedIDFromOutput`,
`TestOpenCodeReadMapsParts`, `TestCapabilitiesOf`, `TestPromptArgs`, and
the live smoke on both binaries recorded in the session note.

## Alternatives considered

- **Insert into `opencode.db` or `state.db` directly**: rejected. Each CLI
  holds its database open, the schemas span several tables plus an event
  log, and ADR-0088's create-only rule was written precisely to avoid
  this.
- **Brief only for OpenCode and Hermes**: rejected as the ceiling. It is
  the fallback, and it throws away history the receiving agent can use
  when the vendor hands us a supported importer. Hermes has no way to
  start with a prompt at all, so a brief was never even available there.
- **A generic "import file" field instead of `Run`**: rejected. The
  subcommand and its arguments are the CLI's own vocabulary and belong in
  that CLI's file, not in a shared struct.
- **Reuse the lifecycle job lane (ADR-0093) to run the import**: rejected
  for now. A handoff is one short foreground step the user is waiting on,
  not a durable operation with streamed output; the job lane would add a
  state machine the interaction does not need.
