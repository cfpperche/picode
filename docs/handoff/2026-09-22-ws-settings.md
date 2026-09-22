# 2026-09-22 — ws-settings: workspace settings (name + how delivered work lands)

Shipped: **Settings…** item in the workspace card menu opens `components/WorkspaceSettings.jsx` (ResponsiveDialog): **Name** field
(PATCH `/api/workspaces/{id}` → `store.RenameWorkspace`, event `workspace.updated`, patched by `applyFleet`); **Landing work** =
ADR-0182 integration declaration as one choice (Same as this machine / Only this workspace; fast-forward checkbox, up to 8 checks);
Communication link. Save follows the decision table in `web/shared/domain/workspaceSettings.js` (tested). Backend: `PUT /api/delivery/integration`
now enforces `expectedVersion` → 409; new `DELETE /api/delivery/integration?workspace=` (workspace required; machine layer never deleted
there); `RemoveWorkspace` drops the workspace's declaration. Also fixed: narrow-window navigation drawer closed on Escape bubbling from
a portaled dialog (`App.jsx` `#desktop-navigation` onKeyDown now checks `currentTarget.contains(target)`). Wire quirk: the machine layer's
`fromScope ""` is dropped by `omitempty`; the client treats anything not `"default"` as the machine. No ADR (extends ADR-0182's existing
doors). Verified: `make ci-scoped` PASS; visual-review PASS on scratch after four rounds (defects: machine rules labelled as default,
"./...." four dots, mobile focus, Escape behavior, focus loss; all fixed); overlayAudit ok every capture; card 5/5. Blind spots: loading
skeleton never seen (localhost too fast); the machine layer itself has no UI — "Same as this machine" shows it read-only.

## Next up

- Workspace menu: Open in editor (VS Code, `code --remote wsl+<distro>`)
- Machine-level integration rules have no UI yet (Preferences?) — the owner's call where it lives
