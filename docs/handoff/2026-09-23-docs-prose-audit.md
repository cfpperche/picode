# 2026-09-23 — docs-prose-audit: wave 1 of the architecture prose audit
Shipped: `docs/plans/docs-prose-audit.md` (waves, method, tracker) plus
wave 1 — routes.md, agent-manager.md, terminal-bridge.md,
security-model.md. Seven drifts fixed, ~95 claims verified OK: five stale
`web/desktop/src/...` paths corrected to the browser bundle, the
user-menu groups named as they ship, the workspace-menu Sessions row
dropped from the prose, and the per-CLI session index documented for all
nine CLIs. terminal-bridge.md and security-model.md verified clean.
Verified: `make ci-scoped` PASS (fmt,vet,hooks,metadata — docs-only
diff); every fix grounded in a grep of the current tree (constants,
symbols, ADR files, route registrations). Method blind spot: prose
claims about vendor behavior (WebKit letterbox, tmux 3.7 floating
panes) accepted as recorded measurements, not re-measured.
visual-review: n/a
Not done / debts: waves 2–4 remain (tracked in the plan file).
Merge: fast-forward ready.
