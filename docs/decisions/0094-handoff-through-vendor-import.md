# ADR-0094: A handoff writer may publish through the target CLI's own import command

- **Status**: accepted (owner asked for OpenCode to join the handoff
  surface, 2026-09-07; the mechanism was chosen after finding that
  OpenCode ships `import`/`export` as first-class commands)
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

OpenCode also does not need it. The CLI ships the pair we want as
supported commands, verified on a real install (OpenCode 1.18.29):

- `opencode export [sessionID]` prints a session as JSON.
- `opencode import <file>` reads that JSON back and reports the id it
  landed.

A probe against an isolated store confirmed the behavior the writer needs:
the import keeps the ids the file declares, keeps the title, and files the
session under the folder the command runs in rather than the `directory`
recorded in the file. Hermes Agent has the same store shape and no such
command, which is why it stays a source.

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
parts}]}`), writes it to a temporary file outside every CLI's store, runs
`import`, deletes the file, and then re-reads the session through
`OpenCodeSource.Read` — the same round-trip every ADR-0088 writer owes.
The verification is what makes the handoff true: the file having been
accepted is not evidence that the conversation is there.

Tool calls map to OpenCode's single `tool` part, which carries the call
and its result together, so a call and the result that answered it become
one part rather than two records.

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
| Target writer needs `Run`, server supplied it | Envelope written to a temp file, import run in the session's folder, file deleted |
| Target writer needs `Run`, none supplied | `ErrNoRunner`; the brief mode stays offered |
| Import command exits non-zero | Handoff fails with the command's clipped output; nothing recorded |
| Import succeeds, round-trip disagrees | Handoff fails; the session is not claimed |
| Target CLI not installed | 400 before anything runs, as for every other target |
| OpenCode as source | Reader over the store, read-only, unchanged by this ADR |
| Tool call and its result | One `tool` part carrying both; `tools: text` renders them as turns |
| Store has no version and no setup check | `ErrUnknownFormat`; brief remains |

Coverage: `TestOpenCodeWritePublishesThroughItsOwnImport` (a runner that
applies the envelope exactly as the CLI's importer does, then reads the
session back), `TestOpenCodeReadMapsParts`, `TestCapabilitiesOf`,
`TestPromptArgs`, and the live smoke recorded in the session note.

## Alternatives considered

- **Insert into `opencode.db` directly**: rejected. The CLI holds the
  database open, the schema spans three tables plus an event log, and
  ADR-0088's create-only rule was written precisely to avoid this.
- **Brief only for OpenCode**: rejected as the ceiling. It is the
  fallback, and it throws away tool history the receiving agent can use
  when the vendor hands us a supported importer.
- **A generic "import file" field instead of `Run`**: rejected. The
  subcommand and its arguments are the CLI's own vocabulary and belong in
  that CLI's file, not in a shared struct.
- **Reuse the lifecycle job lane (ADR-0093) to run the import**: rejected
  for now. A handoff is one short foreground step the user is waiting on,
  not a durable operation with streamed output; the job lane would add a
  state machine the interaction does not need.
