# 2026-09-21 — d2-deployment: Delivery's Deployment lens

Shipped: D2 (ADR-0170). Delivery has two lenses inside its card. The Deployment
lens reports the connected local instance's running revision, the integrated
changes that revision does not contain, and the last recorded deploy attempt —
from a per-workspace binding (migration 065, `delivery.observer.changed`) and
deployment receipts under `<data>/var/delivery/` written by `picode deploy`
(started, then its result). The read carries `environments[]` plus a per-change
`publication`; browser and phone encode the lens as
`?view=delivery&lane=deployment`. Contract and evidence:
`docs/plans/delivery-publication.md`; guide `docs-site/guide/delivery.md`
§ Deployment (the lane's View setup destination); architecture
`docs/architecture/delivery.md` § Deployment observation.

Verified: affected Go packages (`internal/server` sharded, delivery, install,
store, version — 31 new tests) and the JS suites (shared domain 10, mobile 364),
then `make close`. Scratch (`qa-scratch d2` on :8474, fixtures in `/tmp/d2-qa`)
covered the lane's states: not connected, known with one unpublished change,
last attempt passed / failed (previous version still responding) / unknown
outcome / none, a revision this repository lacks, a binding naming another
repository, a conflict and a busy row (both fault-injected), and the Integration
lens's `Not published` marker and detail tile; overlay audit `ok` with every
control row at 36px. Screenshots read in a subagent (verdict below). Blind spot:
Linux headless Chromium, and the deploy producer's real path runs only on the
owner's `picode deploy` (its tests fake the systemctl boundary); remote
environments stay unknown by design.

visual-review: PASS (14 captures read in a subagent: every state's copy
contract-exact, no safe/ready/approved/queued wording, the integration row
marks Not published and its detail carries three verdict tiles, dark theme
legible, phone without overflow; overlayAudit ok, every control row 36px. The
read's one suggestion — a failed attempt in plain body text — was fixed with
the danger tint before closing.)

Reverted the same day: the owner tested it live and removed the lens from the
product (ADR-0177). With `picode-self` as the only environment it answered a
question only PiCode's own repository has, so every other project saw an inert
lane; environments must be project-owned markers if the capability returns. The
receipts the two deploys wrote stay on disk in `<data>/var/delivery/` — evidence
is not pruned — with nothing reading them.
Not done / debts: none carried — the lens is gone (ADR-0177), and
`docs/handoff/open/delivery-flow.md` keeps the D1b list unchanged.
Merge: fast-forward ready.
