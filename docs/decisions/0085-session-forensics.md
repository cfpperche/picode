# ADR-0085: Session forensics — flight recorder and SIGHUP-immune pane roots

- **Status**: accepted
- **Date**: 2026-09-06
- **Extends**: ADR-0084 (CLI terminal session recovery), ADR-0069 (CLI terminals)

## Context

The 2026-09-06 mass-detach incidents (13:41, 14:08) took days-of-forensics
to bound: transcript mtimes, reflog, journal scrying and the event feed
eventually showed ordinary agent-run `make deploy` restarts killing every
PiCode-managed tmux session while interactive-bash terminals survived.
What the investigation never got was direct evidence of *when* (shutdown
vs boot vs hours later), *who* (the deployer), and *why* (the signal
chain). ADR-0084 made sessions recoverable; it did not make the next
incident self-reporting. The structural hint — interactive bash ignores
SIGHUP and survived; `/bin/sh` pane roots do not and died — remained
untested.

## Decision

The daemon keeps a flight recorder, and the pane roots get the same
SIGHUP immunity bash always had.

- **Flight recorder.** On SIGTERM/SIGINT, after the HTTP server is down,
  the daemon writes `<data>/var/shutdown-snapshot.json` (atomic rename):
  the PiCode-owned sessions with pane pid, root command, attach state and
  cwd, plus the binary version. On boot, before serving, it diffs the
  snapshot against the live tmux state, logs a plain verdict
  ("N of M sessions from the <at> shutdown did NOT survive: …" or the
  all-clear, or "unclean exit" when no snapshot exists), appends a
  durable `var/restart-report-<bootedAt>.json` (last 10 kept), and keeps
  the lost-session names in memory to badge the affected terminals
  (`lostAtRestart` on the terminal view → the surface says "PiCode
  restarted while this terminal was running" above the Resume button).
  Every step is best-effort with a timeout; forensics never delays boot
  or fails a shutdown.
- **SIGHUP-immune pane roots.** The generated `launch.sh` starts with
  `trap '' HUP` (after the shebang). Interactive bash already ignores
  SIGHUP, which is why plain shell terminals survived every restart;
  trapped `sh` roots now behave the same, and children inherit the
  ignore. **Stop still means stop**: explicit stop/restart/remove reads
  the pane root's pid, kills the session, then SIGTERMs that pid —
  SIGTERM is not trapped, so the pane root dies deterministically.
- **Deploy record.** Every `picode deploy` appends
  `{at, version, termId, cwd}` to `<data>/var/deploy-log.jsonl`. The
  terminal id rides `PICODE_TERM_ID` when the deploy runs inside a
  PiCode terminal — the common case (agents deploying from their own
  panes).

The next deploy is the experiment: with Part 1's evidence, either
sessions survive (mechanism was SIGHUP — now fixed) or the boot report
pins the loss to the restart window with root commands recorded, and the
next hypothesis starts from data.

## Consequences

Easier: session loss becomes a logged fact with a timestamp boundary;
the deployer identity is recorded forever; the stopped-terminal message
tells the truth about what happened; the bash-vs-sh parity removes a
class of "why did only my CLI die" reports.

Harder: SIGHUP-immune roots can linger headless after a stop until their
next stdio write fails (mitigated by the SIGTERM escalation, which does
not reach grandchildren — the same bounded lingering node CLIs already
show today). The snapshot adds ~100ms to shutdown (inside
TimeoutStopSec=30). The boot diff adds one tmux call with a 5s budget
before serving. Deploy-log lines carry cwd — the file is 0600 under the
data dir, same trust domain as the rest of the var/ forensics.

If the SIGHUP theory is wrong, nothing breaks: the trap is one line, the
escalation keeps Stop working, and the flight recorder answers the
timing question regardless of mechanism.

## Decision table

| Event | Condition | Action |
|---|---|---|
| SIGTERM/SIGINT | tmux available | write snapshot (atomic), then exit; never fatal |
| Crash / SIGKILL | no snapshot on disk | boot logs "unclean exit suspected" |
| Boot | snapshot exists | diff → log verdict + report file + `lostAtRestart` flags |
| Boot | no snapshot, nothing alive | silent |
| Reports accumulate | > 10 | prune oldest |
| launch.sh generated | always | `trap '' HUP` after shebang |
| Stop / Restart / Remove confirmed | pane root pid known | KillSession, then SIGTERM the pane root |
| `picode deploy` succeeds | always | append deploy record (version, termId, cwd) |
| Terminal stopped, `lostAtRestart` | from boot diff | surface says "PiCode restarted while this terminal was running" above Resume |

Coverage: `TestShutdownSnapshotAndBootDiffDecisionTable` (fake tmux; all
five rows), `TestListOwnedReportsPaneFacts` (real tmux pane facts),
`TestPaneRootSurvivesSIGHUP` (trapped root survives the exact incident
signal; untrapped control dies — the experiment in miniature),
`TestLaunchScriptIgnoresHUP` (generated script contract),
`TestAppendDeployRecord`.

## Alternatives considered

- **Persist the snapshot in SQLite**: rejected — the recorder must not
  depend on database state surviving (or racing) a shutdown; a single
  atomic JSON file in var/ is the honest medium for ephemeral evidence.
- **Escalate to SIGKILL the whole process tree on stop**: rejected —
  SIGKILL forfeits the CLIs' own graceful exit (transcript flush,
  cleanup); SIGTERM to the pane root plus the already-closed PTY ends
  real usage, and lingering headless CLIs are bounded by their next
  stdio write.
- **Alert/push on session loss at boot**: deferred — the boot report is
  files and logs; wiring it into the Inbox/push pipeline is a separate
  product decision about what deserves an interruption.

## References and adaptation

- [Cursor checkpoints](../../docs/benchmark-cursor.md): state you can go
  back to — adapted to the daemon itself: every restart leaves a
  readable before/after.
- ADR-0084 (resume) supplies the recovery action this ADR explains.
