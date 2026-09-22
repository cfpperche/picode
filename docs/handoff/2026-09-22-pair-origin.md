# 2026-09-22 — feat/pair-origin: half a gate for the pairing door, by the owner's call

One commit. `/pair` was the one route outside every check — an artefact of how `guarded()` was written, not a decision, and an unreachable `case p == "/pair"` in `exempt()` had made it read like one until `feat/audit-fixes` removed the dead line and left the question.

**What was open.** A page in another tab could post codes at the daemon. It still needed a code off the owner's screen and five wrong ones lock it out, so the hole was small — but the two halves of the gate have very different costs here, and that is what made it the owner's call rather than a tidy-up.

**What was decided.** The cross-site check applies to a `/pair` submission; `HostAllowed` deliberately does not. Pairing is the door you use *before* you are in, so refusing the address someone is pairing a new device from would lock them out of the only page that lets them in — a worse failure than the one being closed, and one that shows up exactly when it hurts most.

Nothing else moves: the page is readable from anywhere, a script and `picode pair` send no `Origin` and pass, and a device on an address PiCode has never seen still reaches it. Six rows in `internal/auth`'s decision table cover the change and the four ways it must *not* lock anyone out; the two that matter fail without it.

Also here: ADR-0154 and ADR-0156 were accepted by the owner and their records now say so (`1eee1973`, on main), closing the other half of that topic's debt.

Verified: `make close` green, scoped. visual-review: n/a — no UI. Nothing deployed.
