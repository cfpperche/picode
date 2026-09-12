# 2026-09-11 — feat/inbox-reply-hardening: adversarial review of the inbox terminal-reply fix

Reviewing 520864e4 (the deepseek-flash session's fix, merged to main): two
real flaws found and fixed on top.

Shipped: a sessionless hello (a nested `pi -p`, a print-mode run — the
terminal id and reply directory are inherited environment) no longer takes
over the recorded session or its pid in the reply registry; a
session-bearing hello always wins, so the reply file keeps reaching the
terminal's own receiver (`internal/server/tui_reply.go`, `terminal_ask.go`).
`accept` on a channelless terminal approval now refuses visibly instead of
closing as "accepted" while nothing was sent — the same visible-failure
rule the ignore fix restored (`internal/store/inbox.go`). ADR-0060's
2026-09-11 amendment and the changelog fragment updated.

Verified: new tests — sessionless-hello takeover (server, end-to-end with a
reply file), channelless accept refusal + local ignore (store); ci-scoped PASS.
visual-review: n/a (no UI change).

Debts: the incident's trigger (why the terminal's receiver answered
"different session" at 15:37:47Z) stays unproven — every path in that class
now delivers or refuses honestly. A foreign session-bearing hello still
flips the preflight to "different session" until the pi re-hellos (≤5 min).
The receiver TS has no committed harness (manual stub-host verification).
Merge: fast-forward ready; the previous fix is on main but NOT deployed
(production runs 0.2.0+89f1209, built pre-fix).
