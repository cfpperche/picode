# 2026-09-14 — title-idle-signal: measured, not adopted

Shipped: a measurement, deliberately without code. The Orca-style terminal-title signal (pane title says ready/working) does not exist on our vendor versions: Codex sets the folder name, Claude keeps a constant topic label through turns (adopting Orca's ✳-idle rule would produce a false idle), Grok/OpenCode echo the last prompt. Recorded in the attention section of `docs/architecture/direct-session-communication.md` with a re-measure condition.
Verified: live scratch, six real TUIs, titles sampled at rest, mid-turn (2 s cadence through a long turn) and after; `var/qa/title-signal/summary.json`. Scratch stopped, six terminals removed.
visual-review: n/a (measurement)
Not done / debts: revisit the title tiers if a vendor version starts emitting status titles; our native hooks remain the first-party evidence.
Merge: fast-forward ready after this close.
