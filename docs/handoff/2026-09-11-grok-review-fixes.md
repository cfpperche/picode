# 2026-09-11 — feat/grok-review-fixes: adversarial review of the Grok meter work

Shipped: corrections only, no behavior change. The debt left by
feat/grok-usage-meter claimed `clisession.GrokSource` "lists prompts only" —
false: it reads `summary.json` and already shows Grok's model and title on
the session row; only the per-turn cost from `usage.json` is missing. Fixed
in `docs/handoff.md` and that session's note. The changelog fragment dropped
this machine's own numbers ("6 of 785 turns") — a public changelog describes
the mechanism, not one machine's counts. New test pins the multi-file cache
key (`statKey`/`cachedParseKeyed`): an appended events.jsonl or a file that
appears later must re-parse.

Verified: go test on climetrics (including `-race`), `make ci-scoped`,
`make close`, then `make ci` on main after the fast-forward. Checked and
found sound: the edited plan/benchmark tables, the event parser and turn
pairing against the real store, and the removal of three merged branches
from handoff.md's In-flight (changelog-normalize, clis-terminals-section,
codex-subagent-resume — all landed on main).

visual-review: n/a — docs and a test only.

Not done / debts: unchanged from the previous note, minus the wrong claim.

Merge: fast-forward ready.
