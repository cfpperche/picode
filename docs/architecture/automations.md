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
a guest agent is reached through its launch terminal, ADR-0179) then, for `start`, one agent per
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
