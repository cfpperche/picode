# 2026-09-22 — ws-menu-utilities: the workspace menu's ways out of PiCode
Shipped: the workspace "…" menu is regrouped (inside PiCode / outside / order / Remove) by a pure module,
`web/shared/domain/workspaceRowMenu.js`, with a decision-table test. New rows: **Show in Explorer** (the file manager
off WSL; the existing `POST /api/workspaces/{id}/reveal`), **Open on <host>** (the list now carries `remote` from
`internal/gitinfo/remote.go`: origin else the first remote, as a web URL with credentials and ports dropped, `kind`
github/gitlab/bitbucket/azure, `defaultBranch`), a **pull request** row (GitHub asks `GET /api/workspaces/{id}/pr`
when the menu opens → "Open pull request #N" or "Create pull request"; GitLab/Bitbucket get the create link; never
on the default branch) and **Copy path** (Linux, plus Windows under WSL via the new `osopen.WindowsPath`; textarea
fallback for insecure contexts). Move up/down gained arrow icons, which also aligns the agent-row menus.
No ADR (no boundary crossed). Docs: `docs/architecture/agent-manager.md`, `docs-site/guide/files.md`,
`docs/changelog.d/ws-menu-utilities.md`. Mobile has no workspace row menu; nothing changed there.
Verified: `make ci-scoped` PASS; `make close` PASS before main was merged in (7d24efba), not re-run after it here.
visual-review: PASS on a scratch instance (desktop menu, copy toast; `__picodeOverlayAudit()` ok on every capture;
card 5/5). Defect found and fixed in-session: Move up/down had no icon, so the label column was misaligned.
Blind spots: the "Checking pull request…" pending row was only seen through a MutationObserver (gh answered in < 1 s);
"Open pull request #N" was never shown against a real PR (the scratch HOME has no gh login); Show in Explorer and
Open on GitHub were not clicked (they open real windows on the owner's desktop); the agent-row arrows were not seen
(the scratch had no agents). These need a live look on the owner's instance.
Merge: fast-forward ready once `make close` passes on the merged tree.

## Next up

- Workspace menu, from the approved proposal: Open in editor (VS Code, `code --remote wsl+<distro>`), Rename workspace, Workspace settings (door to the delivery queue's per-workspace integration declaration)
