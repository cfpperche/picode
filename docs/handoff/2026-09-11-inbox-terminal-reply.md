# 2026-09-11 — feat/inbox-terminal-reply: terminal Inbox questions could not be answered or ignored

Shipped: `ignore` on a terminal-sourced item closes it locally, like an
agent-sourced ignore (`internal/apps/inbox.go`, `internal/server/inbox.go`);
a fresh receiver hello that names no session is refused before parking with
the truth (`internal/server/terminal_ask.go`); the reply file carries the
`pid` whose hello the daemon accepted and a receiver whose pid differs (or
whose own session is unknown) leaves it alone (`internal/server/tui_reply.go`,
`internal/server/intercept/pi-inbox-reply.ts`). ADR-0060 has the
2026-09-11 amendment; routes.md updated.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[5]); new Go tests cover
the local ignore, the empty-session refusal and the address in the file. The
receiver's four consumption rows were exercised ad hoc with a stubbed
extension host (foreign pid leaves the file; own pid + matching session sends
and acks ok; no session leaves it; own pid + another session acks "showing a
different session").
visual-review: n/a (no UI change).

Not done / debts: running receivers are per-launch copies — terminals opened
before this builds keep the old rule until relaunched. The receiver has no
committed harness; those rows stay manual-QA debt. The daemon still keeps one
hello per terminal (last writer wins): the fix makes a wrong writer harmless,
not impossible.
Merge: fast-forward ready (owner merges, then `make ci` on main).
