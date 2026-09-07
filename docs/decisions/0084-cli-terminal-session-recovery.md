# ADR-0084: CLI terminal session recovery — pin and resume

- **Status**: accepted
- **Date**: 2026-09-06
- **Extends**: ADR-0069 (CLI terminals), ADR-0079 (sessions under Agent CLIs)

## Context

Two incidents on 2026-09-06 (13:41 and 14:08) ended every running Agent CLI
terminal at once. Both were ordinary `make deploy` restarts — the 14:08 one
was run by the codex agent working on llama delivery 2 from the root
checkout (reflog `70edb214` 14:08:07, daemon-reload journal line 14:08:40).
The investigation established, from tmux listings, transcript mtimes, the
event feed and journal:

- Plain interactive-shell terminals (pane root `bash`, which ignores
  SIGHUP) survive every restart; CLI and agent panes (pane root
  `/bin/sh`) do not. The exact signal chain that ends the `sh` pane roots
  is still unproven — that gap is tracked as instrumentation debt.
- The CLI *processes* often survive headless for minutes (the codex wrote
  its rollout until 14:16:46, eight minutes after its pane died), so
  checklist/state events kept flowing and masked the breakage.
- No conversation was lost on disk: claude/codex/grok/pi all keep full
  transcripts. Recovery existed only as a manual, disconnected flow —
  Sessions tab → "Open in terminal" — which creates a NEW terminal each
  time and leaves the dead terminal a dead end ("This CLI terminal is
  stopped. Start from Agent CLIs").
- Deploys are frequent (seven on the incident day) because agents ship
  work; a deploy must not cost a conversation.

## Decision

Every CLI terminal with a launch records the native conversation it is
running, and a stopped terminal offers one-click resume of that
conversation.

- **Pin while alive.** The wrapper runtime protocol already reports
  run start/end (`picode-hook runtime-start/end`). At run end, at every
  CLI state report while the run is live, and at explicit stop/restart,
  the server resolves the newest native session of that CLI in the
  terminal's folder written during the current run
  (`clisession.Latest`) and pins it on the launch record
  (`terminal_launches.last_session`, store event `terminal.last_session`).
  The run-start floor prevents one terminal from stealing another
  terminal's conversation in a shared folder. Writes are idempotent per
  (session, updatedAt) so state reports do not flood the feed.
- **Resume when stopped.** `POST /api/terminals/{id}/launch/start` with
  `{"resume": true}` relaunches the CLI with the pinned session's
  server-verified `ResumeArgs` (`claude --resume <id>`,
  `codex resume <id>`, `grok --resume <id>`, pi `--session <file>`),
  replacing default arguments for that one launch — the same contract the
  Sessions tab already uses. The stopped state in the terminal surface
  gains "Resume last session" next to the existing "Start from Agent
  CLIs" link. Refusals: no launch (400), nothing pinned yet (400),
  already running (409). A plain start stays exactly as before.
- **Never auto-restart work** (ADR-0069 unchanged): resume is explicit.
- Known gap, recorded as debt: the first release of this feature only
  pins sessions from its deployment onward; terminals stopped before it
  ship show no Resume button (their conversations remain reachable via
  the Sessions tab).

## Consequences

Easier: a deploy, crash or daemon restart costs at most a click; the
session listing and the terminal agree on what was running; post-incident
forensics get a durable record (`terminal.last_session`) of what each
terminal was doing.

Harder: the pin is heuristic — newest session per (cli, folder, run
window) — so a CLI that writes no transcript (never accepted a turn)
leaves the previous pin, shown in the UI before clicking. The store gains
one more column and one more event; the feed gains one low-frequency
event type.

If we are wrong about the session-death mechanism, nothing here breaks:
resume does not depend on *why* the session died. The instrumentation
debt (shutdown snapshot of live sessions + boot diff, and a SIGHUP-immune
pane-root experiment) remains open in `docs/handoff.md`.

## Decision table

| Conditions | Action / observable result |
|---|---|
| Run ends (wrapper runtime-end) | Pin newest session of the run; event only when the pin changes |
| State report while run is live | Refresh pin; idempotent write, no feed flood |
| Stop/Restart confirmed | Pin before the pane is killed |
| Stopped terminal, Resume, pin exists | Launch with `ResumeArgs`; terminal live; defaults replaced for that launch |
| Stopped terminal, Resume, nothing pinned | 400 "No previous session recorded yet." |
| Stopped terminal, no launch, Resume | 400 "This terminal has no CLI launch to resume." |
| Live terminal, Resume | 409 "This terminal is already running." |
| Start without `resume` | Unchanged behavior (CLI defaults) |
| CLI wrote no transcript during the run | Previous pin kept; UI shows it before the click |

Coverage: `TestCLITerminalResumeDecisionTable` (real tmux; refusal rows,
pin via runtime protocol, argv-level proof that resume args replace
defaults, plain-start parity), `TestLatestFiltersByRunStart`
(clisession floor), `TestEveryMutationAppendsAnEvent` row
`SetTerminalLastSession` (event + idempotency).

## Alternatives considered

- **Auto-resume after restart**: rejected — contradicts ADR-0069 ("never
  automatically restart work"); an unattended relaunch can replay prompts
  into a changed world.
- **Pin from the wrapper at launch time**: rejected — the native session
  file does not exist until the CLI's first turn, so the pin would be
  empty exactly when it matters; resolution from the CLI's own index is
  the honest source.
- **Kill-the-killers: forbid deploys with live CLIs**: rejected as the
  primary fix — deploys are the delivery path for this very work; a
  hard block would deadlock agents. A soft warning belongs to the deploy
  guard (ADR next step, tracked in handoff), not to resume.

## References and adaptation

- [Cursor checkpoints](../../docs/benchmark-cursor.md): every turn leaves
  a visible restore point — adapted to terminal conversations as the
  pinned session shown before the click.
- tmux-sentinel (system service) and `KillMode=process` (ADR-0002/0018)
  already protect the tmux server; this ADR protects the *conversation*,
  which is what users actually lost.
