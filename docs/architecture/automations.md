# Automations (ADR-0045)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`internal/automate` ticks every minute (same shape as the backup loop,
started in `cmd/picode` with the process context, not the HTTP server)
and asks `Due` for each enabled schedule of each enabled automation
(`automation_schedules`, migration 038 — one row per rule with its own
cron, IANA zone (`""` = daemon local), switch and `last_fired_at`; the
amendment of 2026-09-09): a slot fires once at slot + a deterministic
per-schedule jitter (≤ half the interval, ≤ 30 min); a daemon outage
yields at most one `catch-up` run per schedule; boot fails any run left
`running` (`daemon restarted`). Runs carry `schedule_id` (null for
webhook and Run now); the runner receives an `automate.Firing`. Editing
a rule's cron or zone clears its last fire, so it never catches up a slot
it was not yet asked for. `internal/cron` is a stdlib 5-field matcher. The runner lives in `internal/server` (`automations_run.go`):
the decision table (`decideFire` — Pi is required only for `start`, which
creates a Pi agent, and for a `message` whose target is a stopped Pi agent;
a guest agent is reached through its launch terminal while it is open — `Fire` hands an interactive guest to `doorRun`, a stopped one skips as `terminal closed` before any door call; the door is `doorDeliverUnattended`, which refuses an unrecognized composer (`unrecognized`) instead of the blind paste a person at the terminal gets; the Pi-only `agent in terminal` skip does not apply to guests, ADR-0179) then, for `start`, one agent per
automation (created lazily) whose `session_path` is cleared so
`Runtime.Start` mints a fresh session (ADR-0039), `startManaged` +
`SendTurn`, a `RunObserver` on the managed agent for settle / exit /
per-message cost (the cap closes the run on the crossing message), and a
30 s watchdog for the session path, the 2 h timeout and a file-based
cost fallback. `message` enqueues a
`follow_up`. Routes: `GET/POST /api/automations`,
`GET/PATCH/DELETE /api/automations/{id}`, `POST …/secret` (rotate),
`POST …/run` (Run now, 409 when busy), `POST …/fire` (webhook:
`Authorization: Bearer` or `X-Webhook-Secret`, 64 KB cap; 401/404/409/413),
`GET …/runs`, `GET /api/automations/templates` (built-in list, `automate.Templates()`). Inbox items use `sourceKind: automation`. v2: `/automate <text>` in the composer is intercepted client-side (`lib/automateDraft.js`): the current agent gets a fence-shaped request, the settled reply is parsed into a read-once `sessionStorage` draft (`lib/automationDraft.js`) and `#/automations/new` opens pre-filled; template cards use the same draft slot.

## Start runs on guest CLIs (ADR-0217)

An automation's `cli` (`automations.cli`, default `pi`; `store.UnattendedCLIs`)
picks the CLI of its `start` runs. `pi` is the managed runtime above. For
`claude-code`, `codex`, `grok`, `hermes`, `opencode` and `omp`, `cliStartRun`
(`automations_cli.go`) reuses the automation's own agent of that CLI (created
with its terminal through `createCLIAgent` on the first run), closes an open
terminal and starts it again (a new conversation), then `driveCLIRun`:

1. retries `doorDeliverUnattended` every 2 s for 90 s while the refusal is
   one a fresh TUI outgrows (`closed`, `unrecognized`, `unobservable`,
   `working`, `compacting`, `busy`); any other refusal, or the deadline,
   fails the run naming it — a trust question, a login or a vendor menu is
   never pasted into;
2. requires a `verified` receipt, or, when the screen after Enter was not
   readable, a hook state set after the paste within 15 s (`hookSawPrompt`:
   the CLI's own prompt-submit);
3. follows `TermStates` each second: `idle` set after the paste ends the
   turn; each new `needs-you` files one Inbox question and the run waits;
   the session file is priced every 30 s against the cost cap
   (`climetrics.MeterSessionFile` on the pinned last session); 2 h times out;
4. prices the session, finishes the run (Inbox result, notify URL) and
   closes the terminal with the ADR-0085 escalation (`killTerminalPane`).

A `start` whose agent's terminal is open skips as `agent in terminal`, as a
Pi agent's does: that terminal is someone's. The editor offers the CLI
(`START_CLIS`, pinned to the store's list by `TestStartCLIsMatchTheEditor`)
and hides Pi's provider, model and thinking for the others. Measured
2026-09-25 on a scratch instance: Claude Code 2.1.282 3/3 runs done (~$0.15
each), Codex 0.157.0 2/2 (~$0.12), OpenCode 1.18.32 2/2 and Omp 18.2.11 2/2
(~$0.0016); Grok and Hermes not run live. The cost cap holds only where the
session file is priced (`climetrics.Metered`: Claude Code, Codex, Omp); for
OpenCode, Grok and Hermes the runs table shows "—" and the editor's hint says
the limit does not stop them (`METERED_CLIS`, pinned by
`TestMeteredCLIsMatchTheEditor`).
