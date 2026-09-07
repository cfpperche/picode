# ADR-0087: Cross-CLI session handoff — continue a conversation in another Agent CLI

- **Status**: accepted (owner approved the direction, the scoped exception
  to the never-write rule, the first milestone and the deterministic brief
  on 2026-09-06; live smoke on real binaries passed the same day, see
  `docs/handoff/2026-09-06-session-handoff.md`)
- **Date**: 2026-09-06
- **Extends**: ADR-0069 (Agent CLI terminals), ADR-0079 (sessions under
  Agent CLIs), ADR-0084 (pin and resume)
- **Amends**: ADR-0056's rule that PiCode never writes a guest CLI's home
- **Evidence**: web survey 2026-09-06 (session-migrate, agent-migrator,
  context-bridge, Codex `migrate-to-codex`, Grok Build `grok import`, Zed
  ACP discussions), readers validated against real sessions of all five
  CLIs on the owner's machine

## Context

PiCode lists the sessions of pi, Claude Code, Codex, Grok and Hermes Agent
(ADR-0079) and resumes each in its own CLI (ADR-0084). The owner's request
goes one step further: start a conversation in Claude Code and, for any
reason — quota, model fit, a tool only another agent has — continue it in
Codex, Grok, pi or a CLI that joins later. pi already switches provider
and model mid-session; nothing switches *agent*.

What exists elsewhere converges on one shape. Nobody hands a live process
over; every working tool (session-migrate, agent-migrator, context-bridge)
reads the source's native transcript, projects it into a small canonical
timeline, writes a **new** native session where the target CLI scans for
its own, and starts the target with its resume command. Vendors do the
same one way: Codex imports Claude Code sessions and auto-compacts on the
first turn; Grok Build resumes Claude Code transcripts from its welcome
screen. Every implementation loses the same things — signed or encrypted
reasoning, subagent trees, hooks, permissions, native UI state — and the
good ones count what they dropped instead of hiding it.

Constraints inside PiCode:

- The only per-CLI seam is `clisession.Source` (list). Every other CLI
  behavior is a `switch` on the id. CLIs arrive on demand (Hermes did this
  week); a handoff that needed a central switch would rot with each one.
- ADR-0056 states, and code enforces, that PiCode never writes
  `~/.claude`, `~/.codex`, `~/.grok`, `~/.pi`. A native handoff must
  create a session file exactly there: Claude Code and Codex find
  sessions only by scanning their own roots, and Hermes keeps everything
  in one SQLite file it may hold open.
- Launch arguments are single-line (`clilaunch.Validate`), so a
  multi-line brief cannot ride in `overrides.args`.
- Claude Code rejects a resumed transcript whose `tool_use` has no
  `tool_result` right after it, and any unsigned `thinking`.
- Codex cannot pre-assign a session id; Claude Code, Grok and pi can.
- pi's own resume is PiCode's chat flow (managed agents own their session
  directory, ADR-0040/0053), not a terminal.

## Decision

A session of any Agent CLI can **continue in another Agent CLI** from the
Sessions tab, as a handoff, never as a live switch.

1. **Portable timeline.** `internal/transcript` is the canonical,
   CLI-neutral model: ordered events of kind message, tool call, tool
   result, thinking, compaction and context, plus a header and a manifest
   that counts everything a reader could not carry. It is deliberately a
   conversation, not a runtime: no reasoning signatures, no hooks, no
   permissions, no provider payloads. `Window` cuts at the last compaction
   the way the source replays; `Repair` pairs every call with a result
   (hoisting late ones, synthesizing an error result for unanswered ones);
   `Prepare` strips thinking and injected context and folds a leftover
   compaction into a user message. `Brief` renders a deterministic
   markdown summary — no model involved.
2. **Capabilities, not switches.** `Reader`, `Writer` and `Prompter` are
   optional interfaces a `Source` implements in its own file and the server
   discovers by type assertion. `GET /api/clis` advertises them as
   `sessions: {list, read, write, prompt}`; the web derives the "Continue
   in…" targets from that and keeps no list of its own. A new CLI joins by
   implementing what it can; what it cannot is simply not offered.
3. **Two modes.** *Native*: the target has a Writer; the timeline becomes a
   real session of the target, tool calls in the target's own shape (or
   as plain text on request), preceded by a handoff note that says where
   the history came from and that files may have changed since. *Brief*:
   the target has a Prompter; a markdown brief is written under
   `<dataDir>/handoffs/<id>/brief.md` and the target starts with a
   one-line prompt pointing at it. Native is the default when available.
4. **Create-only exception to ADR-0056.** A Writer may create a new
   session artifact inside the target CLI's own session root — Claude
   Code `~/.claude/projects/<encoded cwd>/<uuid>.jsonl`, Codex
   `~/.codex/sessions/YYYY/MM/DD/rollout-<stamp>-<uuid>.jsonl`, pi the
   adopting agent's private dir — and nothing else: never an existing
   session, never settings, credentials or indexes (Codex's SQLite is not
   touched; the CLI scans). Every write is atomic, refuses to overwrite,
   and is re-read through the same package's Reader before it counts;
   a file that does not read back is removed. Preview never writes.
5. **One plan, two endpoints.** `POST /api/clis/{cli}/sessions/handoff
   /preview` and `…/handoff` share `planHandoff` (ADR-0070's rule): the
   preview shows the counts, the manifest, whether the source is still
   live and which modes exist; the execution commits the same plan.
6. **Lineage is PiCode's.** Each handoff is a `session_handoffs` row with
   a `session.handoff` event (ADR-0048): source, target, mode, window,
   tool shape, manifest, and the terminal or agent it continues in. Both
   session listings show it ("from Claude Code", "continued in Codex").
   Codex in brief mode has no id up front; its row resolves through the
   terminal's pinned session (ADR-0084).
7. **pi as a target is an agent.** The session is written into a new
   managed agent's private directory and the agent is created stopped in
   the workspace that owns the folder — the product's own way to run pi.
8. **Format drift is version or refusal.** The installed target version
   comes from the setup check (run on demand when missing or stale); when
   neither the check nor a local artifact tells the version, native is
   refused with a clear message and brief remains.

## Consequences

Easier: a conversation is no longer captive to the CLI that started it;
adding a CLI to the handoff surface is one file with two or three methods;
the user sees what will be lost before anything is written; every handoff
leaves an audit row and a visible lineage.

Harder and accepted: PiCode now creates files in guest CLI homes — the
narrowest exception (create-only, session roots only, round-trip
verified) and the first. Native formats are undocumented and move; the
round-trip and the manual smoke on real binaries are the guard, and a
change upstream shows up as a refused write, not a corrupt session. Thinking
never travels, so the receiving model starts without the previous one's
reasoning. The brief is deliberately dumb: honest, cheap, reproducible.

If we are wrong about a format, the target refuses to resume a file
PiCode wrote; nothing of the user's is modified, the source stays intact,
and the brief path still works. If we are wrong about the invariant, the
line is easy to audit: `writeNewFile` is the only way a Writer touches disk.

## Decision table

| Source Reader | Target Writer | Target Prompter | Mode asked | Result |
|---|---|---|---|---|
| no | – | – | any | 409 "Sessions of <src> cannot be read yet." |
| yes | yes | – | native / "" | native session |
| yes | no | yes | native | 409, preview lists `modes: ["brief"]` |
| yes | no | yes | brief / "" | brief |
| yes | yes | yes | brief | brief |
| yes | no | no | any | 409 "<dst> cannot receive a handoff yet." |
| yes | yes, version unknown | – | native | 409 unknown format; brief stays offered |
| source live, `force=false` | – | – | any | 409 with `live` {kind, id, name} |
| source live, `force=true` | – | – | any | proceeds; manifest warning |
| target not installed (non-pi) | – | – | any | 400 before anything is created |
| source over 64 MiB | – | – | any | 413 |
| `to == cli` | – | – | any | 400 "Choose a different CLI." |
| `window=recent`, no compaction | – | – | – | whole timeline, no summary |
| target pi | yes | – | native | stopped managed agent, no terminal |
| unknown `{cli}` / unknown `to` | – | – | – | 404 |
| body has unknown keys | – | – | – | 400 |

Coverage: `TestHandoffDecisionTable`, `TestHandoffNativeClaudeToCodex
CreatesRolloutAndTerminal` (argv-level proof of `resume <id>`, lineage on
both listings), `TestHandoffBriefWritesFileAndPromptArgs` (brief under the
data dir, pre-assigned id for Grok, none for Codex, target home untouched),
`TestHandoffToPiAdoptsAgent`, `TestHandoffLiveSourceNeedsForce` (real
tmux, runtime lease and pin), `TestCLIsAdvertiseSessionCapabilities`;
`internal/clisession`: reader tests per CLI, `TestReadRefusesPathOutside
Root`, `TestReadRefusesOversize`, writer round trips, `TestWriteTextTools
ModeAndNeverOverwrites`, `TestCrossFormatChainKeepsCounts`, `TestPrompt
Args`; `internal/transcript`: window, repair, prepare, note wording, brief;
store: `TestAddSessionHandoffRoundTrip` and the `AddSessionHandoff` row of
`TestEveryMutationAppendsAnEvent`; web: `sessionHandoff.test.js`.

## Alternatives considered

- **Live switch (share one process or protocol between agents)**: no
  agent exposes it; ACP editors (Zed) keep sessions isolated and hand
  off by summary. Rejected as fiction.
- **Summary-only (brief) for every target**: never breaks on a format
  change, but throws away tool history the receiving agent can use.
  Kept as the fallback, not the product.
- **Shell out to session-migrate**: covers 18 CLIs today, but is Python
  3.11, Linux-only, CLI-only, and would put the create-only invariant in
  someone else's hands. Its format research informed the writers.
- **Writer overlay directories only (never the CLI's root)**: works for
  pi (`--session <path>`) and Grok (`GROK_HOME`), not for Claude Code or
  Codex, which the owner named first. Rejected for the milestone.
- **A jobs table with progress (llama pattern)**: a translation of a
  10 MB file finishes well under a second; the plan/commit pair plus a
  30-second timeout is enough. Revisit if reads move off the local disk.
- **Hermes as a native target**: means inserting rows into a SQLite file
  Hermes may hold open, and Hermes has no interactive launch with an
  initial prompt. Deferred to its own decision; Hermes is a source.
- **Grok as a native target**: the per-session directory is understood
  for reading; whether `grok --resume` needs `updates.jsonl` is not.
  A spike against the real binary decides; brief serves meanwhile.

## References and adaptation

- [session-migrate](https://github.com/xhluca/session-migrate): the
  "native → validated timeline → native → resume" pipeline, the manifest
  of omissions and the rule that writers never invent tool output.
- [context-bridge](https://github.com/conrad621/context-bridge): the
  provenance fingerprint idea, adapted here as the lineage row.
- Codex `migrate-to-codex` and Grok Build `grok import`: vendors confirm
  the transcript-not-workspace framing; the handoff note tells the model
  the same thing in words.
- [Cursor checkpoints](../../docs/benchmark-cursor.md): show the user
  what a restore point contains before acting — the preview.
