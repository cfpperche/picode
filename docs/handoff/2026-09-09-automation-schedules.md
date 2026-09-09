# 2026-09-09 — feat/automation-schedules: many schedules per automation

Shipped (ADR-0045 amendment 2026-09-09): the schedule leaves the
`automations` row into `automation_schedules` (migration 038) — one row
per rule with cron, IANA zone (`""` = daemon local), label, switch and
`last_fired_at`; `automation_runs.schedule_id` names the rule that fired.
Engine iterates enabled schedules with per-row jitter and catch-up;
`Runner.Fire` takes an `automate.Firing`. API returns `schedules` (each
with `nextFireAt`; the automation's is the earliest) and takes
`schedules` on POST/PATCH; `cron` stays as the one-rule shorthand.
Editing a rule's cron or zone clears its last fire (starts over, as
ADR-0100). Editor (desktop + mobile): a list of rules with presets,
label, switch, Remove, *Add a schedule*, browser zone stamped on new
rows; list/detail summarise every rule; Runs table shows the rule.
Shared `domain/automationSchedule.js` (+ tests); guide, architecture, CHANGELOG.

Verified: go tests (store, automate, server) incl. a two-zone tick test
and an API round-trip; migration dogfooded on a copy of `~/.picode/picode.db`
(3 automations → 3 rows, last fire kept); web tests + build; ci-scoped
PASS; qa-scratch `automation-schedules` :8473 — desktop editor with 3
rows, added a 4th through the UI, saved, API shows 4 with next fires;
detail line and mobile editor read from screenshots. visual-review: PASS.
Not done / debts: no schedule-triggered run observed live on scratch (it
would spend on a real pi session) — the runs-table label cell is unit-tested only.
Merge: fast-forward ready after `make close`.
