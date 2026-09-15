# 2026-09-15 · feat/tmux-reap-foreign — one PiCode does not reap another's sessions

**The gap, closed.** ADR-0139 merged every instance's reads, so a scratch
instance saw production's twelve terminals in *Not in PiCode's records* — one
click from **Remove**. ADR-0133 had accepted that ambiguity in its copy
("no read can settle it", measured 2026-09-14 with seven live sessions of a
second instance); ADR-0141 settles it: the session itself carries the answer.

**What shipped.** The Manager stamps `PICODE_INSTANCE` = its data directory
into every session it creates — one funnel (`NewSessionEnvSize`, after the
caller's env, so a caller cannot claim another identity; a real-tmux test
pins it against a forged value), derived from `filepath.Dir(socket)`, so
`New()` on the shared default server stamps nothing. `SessionReceipt` reads
it beside `PICODE_TERM_URL`. An unclaimed session stamped by another
directory — or, for sessions created before the stamp, carrying another
instance's port — is named on screen as another PiCode's work and gets **no
removal door**; `reap` refuses it behind the receipt, so the rail holds
without a human watching. A session stamped by this instance keeps the
leftover copy and the door; an agent session from before the stamp (no
stamp, no URL) keeps ADR-0133's honest "cannot tell them apart".

**Evidence.** `internal/tmux/instance_test.go` (real tmux: stamped, forged
value loses, default manager stamps nothing); `server_test.go` receipt
fields; `TestTmuxReapDecisionTable` gained four rows (foreign refused and
named, own stamped reaped, pre-stamp port fallback both ways) and a new view
test (foreign screen has no actions block; own leftover keeps it). Visual on
scratch `:8474`, screenshots read in `var/screenshots/`: production's
`picode-sh-antigravity-c1c4d7` shows *"belongs to another PiCode instance —
https://localhost:8445"* and no button; the scratch's own `picode-sh-fakeleftover`
keeps **Remove this session**. Overlay audit `ok:true`. End-to-end rail:
`POST /api/apps/tmux/action` with the live receipt answered *refused* and the
production session was still running (`$1440`).

**Not done, on purpose.** The list still carries no per-row foreign badge
(one tmux call per screen, ADR-0133 rule 1); the detail screen is where the
verdict lives. Moving a data directory (or a port change) makes an instance's
own old leftovers look foreign — refused with the reason, still removable
from tmux itself; recorded in ADR-0141's consequences.

**Observed, not mine.** Production's default socket moved 12 → 10 sessions
during the session: no `tmux.session.reaped` event today (last: 2026-09-14),
server pid unchanged since 11:40 — the owner's terminals closed normally.

**Gates.** `make ci-scoped: PASS`.
