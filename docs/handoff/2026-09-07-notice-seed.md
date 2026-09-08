# 2026-09-07 — feat/notice-seed: a reload stops announcing the backlog

Found by the first live test against a **real pi dialog** (scratch agent with
`pi-roles` installed by path, `/role` raising `ui.select`): the needs-you card
appeared on a fresh page load with the agent already waiting — the one thing
the seed rule was written to prevent, and which the merged docs already
claimed. Cause: the effect could not tell "nothing is waiting" from "nothing
has been read yet", so it seeded from the first render's empty fleet and the
first real answer read as an arrival. Both shells now gate the pass on having
read the fleet once — desktop sets `fleetLoaded` in `loadWorkspaces`, mobile
uses `useFleet`'s `loaded`.

Verified live on a scratch (8474) with a real pi `select` dialog: page load
with the question already open → **0 cards**, sidebar "Needs you" still on;
answered through `POST /api/agents/{id}/ui` → nothing; `/role` again while the
browser sat on `#/` → **1 card** with the real title, `overlayAudit ok`.
`make ci-scoped` PASS.

visual-review: PASS (live-arrival.png, real dialog; audit ok; card 5/5)

Note for whoever tests a dialog next: pi-roles installed from npm made the
managed agent exit at spawn (`rpc: write command: file already closed`) in an
isolated scratch HOME; installing it by absolute path worked. `/model` raises
no dialog.

Debt unchanged: the *finish* card still only fires for the agent whose socket
is open. Merge: `git merge --ff-only feat/notice-seed && make ci` from main.
This wants a deploy — the bug is live on the instance deployed at 00:55Z.
