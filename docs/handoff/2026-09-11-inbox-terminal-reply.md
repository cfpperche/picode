# 2026-09-11 — feat/inbox-terminal-reply: terminal Inbox questions could not be answered or ignored

Shipped: `ignore` on a terminal-sourced item closes it locally, like an
agent-sourced ignore (`internal/apps/inbox.go`, `internal/server/inbox.go`);
a fresh receiver hello that names no session is refused before parking with
the truth (`internal/server/terminal_ask.go`); the reply file carries the
accepted hello's `pid`, and a receiver with another pid (or no session of
its own) leaves it alone (`internal/server/tui_reply.go`,
`peer_onboarding.go`, `peer_attention.go`,
`internal/server/intercept/pi-inbox-reply.ts`). ADR-0060 has the
2026-09-11 amendment; routes.md updated.

Verified: `make ci-scoped` PASS; Go tests cover the local ignore, the
empty-session refusal and the address in the file. The receiver's
consumption rows run in the committed node harness (`peer_receiver_test.go`):
the original decision table plus the pid/session skip rules (foreign pid
left; no session left; matching pid + session submits and acks ok; matching
pid + other session acks "showing a different session"). This branch also
addresses the peer setup/attention files.
visual-review: n/a (no UI change).

Not done / debts: running receivers are per-launch copies — terminals opened
before this builds keep the old rule until relaunched. The daemon keeps one
hello per terminal (last writer wins): a wrong writer stays possible until
hellos are tracked per process — the fix makes it harmless, not impossible.
Merge: fast-forward ready (owner merges, then `make ci` on main).
