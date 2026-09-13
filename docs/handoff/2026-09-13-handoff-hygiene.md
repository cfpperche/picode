# 2026-09-13 — handoff-hygiene: un-hiding six debts and the board's two defects

Shipped: `docs/handoff/open/snippets.md` had no `## Next` / `## Debts`
heading, so its six debts were **invisible** to every session that reads
the board (`make handoff` only renders bullets under those headings).
It now carries three consolidated debts. Pruned with evidence: bullets
that only restated `docs/plans/snippets-v2.md`, the `desktop-v2` Next
that duplicated `work-browser-tabs` detail for detail, the v1 snippets
note's debt that its topic file already carries, the 0.2.0 release step
(cut; `git tag v0.2.0`), and a stale worktree snapshot the board's own
In flight section renders live. Terminal's "mobile rows offer only
Remove" was rewritten to what is still true (the phone's rows gained
**Send to terminal…** / **Run command…**).

`docs/plans/handoff-board.md` holds the measured diagnosis and the three
options, **awaiting the owner's call**: the board is 97 lines but 12 232
of 12 288 characters, so the byte cap binds ~20 lines before the line cap
and one honest debt can fail a gate every topic shares. Recommended:
**A** (the generator fails on a topic file whose bullets sit outside the
two headings — kills the silent-hole class) + **C** (the board renders a
per-topic count and its `Plan:` line; the bullets live in the topic).
**B** (raise the cap) only if the long-form board is wanted.

Verified: `make handoff` writes 97 lines / 12 232 bytes under both caps;
`make ci-scoped` PASS (docs-only: fmt, vet, hooks, metadata).
Merge: fast-forward ready.