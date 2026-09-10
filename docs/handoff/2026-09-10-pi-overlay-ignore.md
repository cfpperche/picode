# 2026-09-10 — pi-overlay-ignore: git ignores per-agent pi overlays

Shipped: `.gitignore` hides `.pi/roles/` and `.pi/compact/` — the per-agent
overlays (PI_ROLES_AGENT / PI_COMPACT_AGENT, ADR-0033/0061) that every managed
agent rewrites on `/roles` or `/compact-edit` and that surfaced as `??` in
every `git status` (cf. 302e7797, a commit spent just dropping `.pi/roles.json`).
The workspace layer (`.pi/roles.json`, `.pi/compact.json`) stays tracked —
policy as code per a2cbad66; ADR-0033 untouched (overlay was never meant to
be committed; `.claude/` is the precedent).

Verified: `git check-ignore` proves overlays ignored, workspace files not;
`make ci-scoped` green (first run FAILed inside go test — cold-cache flake
already on the debt list; immediate rerun green); `make close` green,
main fast-forwards.

visual-review: n/a (no UI surface).

Not done / debts: edits from a raw terminal `pi` (no agent env) still write
the tracked workspace files and dirty the repo by design — owner chose this
option deliberately; if that residue still annoys, untracking + templates
means amending ADR-0033 (owner call). No changelog fragment: repo hygiene,
not product-visible.

Merge: fast-forward ready.
