# 2026-09-07 — feat/dashboard-cross-cli: the dashboard measures every agent CLI

Shipped (backend): `internal/climetrics`, one `Meter` per agent CLI over the
stores `internal/clisession` already lists, merged into the payload
`/api/sessions/stats` has always served. The surface measured pi and only pi
while PiCode ran six CLIs: on this machine a 7-day window reported **$517.04**
of **$1,916.35** actually spent — 27% of the truth, with nothing on screen to
say so. Every metric now resolves to reported / partial / not-reported /
unavailable rather than to a `0`, a `coverage` payload carries the whole
matrix, and `?scope=machine|picode` narrows to claimed workspaces (default
stays the whole machine, so no row the v1 surface showed disappears).

Per CLI: Claude Code writes cost only on a cumulative snapshot, present on 46
of 340 transcripts and never on a live session — its cost is spread over the
messages that earned it and reported as **partial** with both counts. Codex is
never priced (owner's call); it shows the quota window it does record. OpenCode
prices every message. Hermes' own `billing_mode` outranks the operator's
setting. Grok is activity-only and says so. Study:
`docs/benchmarks/2026-09-07-cross-cli-agent-telemetry.md`; ADR-0097.

Verified: `make ci-scoped` PASS. 34 package tests including a sabotage test per
adapter (no message content in any payload) and a fixture per CLI. Measured
across all six stores (3.1 GB): warm refresh **31 / 46 / 112 ms** (today / 7d /
all) against 2.37 s before the per-file parse cache; cold 1.78 / 3.76 / 4.76 s;
fingerprint 14 ms. The 3 s target for cold `all` was missed and accepted — a
once-per-process cost behind the skeleton the view already renders.

Found and fixed on the way: compactions bucketed by file mtime on a stale
comment (62/62 lines on disk carry their own timestamp); `session.Fingerprint`
returned `""` for a missing root, which would have disabled the cache for every
CLI whenever one was uninstalled; per-type cost was parsed and discarded; and
three tests wrote fixtures into the developer's real `~/.pi/agent/sessions`,
leaving 535 of 593 session directories behind. Those 535 are still on this
machine — cleanup is the owner's call, not a code change.

Shipped (surface): a **By CLI** card with billing badges, a **This machine /
PiCode** scope control beside the range, a **What each CLI reports** matrix,
CLI marks on the model/tool/session rows, and four new panels — Code impact,
Agent time, Limits, Efficiency.

The visual pass earned its keep: seven rendering lies survived a correct
payload, every one of them a `$0.00` or an empty state saying the wrong
thing. The worst was a day whose 260 messages were all live Claude Code
sessions — no snapshot written yet, so the true cost was *unknown* and both
desktop and mobile printed `$0.00`. They now read `—` with the reason. The
scope toggle also exposed a real proration bug: impact counted a file's
in-window tokens without asking whether the scope wanted that folder, so a
picode-scoped window with no matching sessions still reported "+10,724
lines". Fixed with a regression test. Full list in ADR-0097.

Debt: billing mode is `api` for Pi (a fact — PiCode holds its credentials)
and whatever Hermes states for itself; every other CLI stays `unknown` until
an operator setting lands, so those rows carry no badge. Top Sessions marks
its CLI rather than routing to that CLI's list: `#/clis/sessions/<wsId>` is
ADR-0079's shape and widening it belongs to that ADR.

Next: the operator-set billing mode (ADR-0097 §4.4), and the intercept-hook
turn/duration source in ADR-0097's alternatives, which needs its own ADR.
Merge: `git merge --ff-only feat/dashboard-cross-cli && make ci` from main.
