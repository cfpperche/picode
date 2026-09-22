# 2026-09-22 — feat/qa-scratch-owner: qa-scratch starts and seeds only its own daemon
Shipped: `scripts/qa-scratch.sh start` checks the port after the build, removes a stale server.json, and reports up only when `owns_port` holds (the daemon wrote server.json in its data dir for that port and is the listener); otherwise it names who answers, stops what it launched and exits 1. `seed` refuses unless the same check holds. Rule documented in `.pi/skills/agent-browser/SKILL.md`.
Verified: live on scratch instances — normal start + seed + stop; a listener taking the port during the build is refused; seed pointed at a fake instance answering 200 to everything is refused and the fake received zero requests; restart over a stale server.json.
Blind spot: the narrow window between the post-build check and the daemon's bind was not tested directly; it is covered by the same `owns_port` function the seed test exercised.
visual-review: n/a
Not done / debts: none.
Merge: fast-forward ready after `make close`.
