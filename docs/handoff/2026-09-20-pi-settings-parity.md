# 2026-09-20 — pi-settings-parity: Pi's own pane catches up with the guests'

Why this branch exists: the owner opened Pi's Settings beside the eight guest CLIs' and asked why Pi had been left out. Half the complaint was a misreading — Pi's "Defaults" row *is* model and effort, "Tools" is its approvals equivalent, and it has no Memory row because pi keeps no memory. The other half was right and worse than it looked: pi persists about forty settings keys and the pane showed eight, so the owner's own machine had `theme: dark` and `hideThinkingBlock` set with no row for either.
The cause, for the record: ADR-0163 gave the guests a schema-driven pane where a row is one line in a table; Pi kept the hand-written form from ADR-0101 and nobody revisited it. The newest CLIs ended up with a better mechanism than the oldest one, which on screen is indistinguishable from favouritism.
What landed: five rows on the This machine layer, the only place pi writes them — `theme`, `hideThinkingBlock`, `quietStartup`, `defaultProjectTrust`, `shellPath`. Each name and value domain was read out of the installed pi 0.86.1 bundle before it was declared; `defaultProjectTrust` carries exactly `ask | always | never` because pi's getter resolves anything else to `ask` and the row would never look set. `npmCommand` stayed out: pi stores it as an argv array and this pane writes scalars only.
A machine-only key sent to the workspace or agent layer is refused by name before trust or path resolution answers, so the message names the key instead of blaming the folder.
Two defects the live check caught that the unit tests could not: `web/shared/domain/resolveLayer.js` is a named list, so the Theme row rendered empty while the file said `dark` until the key was added there too; and a boolean was about to ship as a checkbox two rows below the switch Auto-compact has always used. All three are switches now.
Verified against a copy of the owner's real `~/.pi/agent/settings.json` on a scratch instance: reading `theme: dark`, writing and resetting two keys, and the file coming back byte-identical with `packages` and `lastChangelogVersion` untouched.
State: `make close` green, `main` can fast-forward. Durable items stay in `docs/handoff/open/agent-clis-native.md`, including the one this branch created — a boolean still looks different in the two panes, because the guests need an unset state a switch cannot show.

## Next up

- Decide the boolean control once: either the guest pane gains a tri-state, or the unset state moves entirely into the source line and both panes use the switch.
