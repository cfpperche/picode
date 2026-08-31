# ADR index

One architectural decision per file, immutable once accepted. Superseding
an ADR requires a new ADR. Template: [template.md](template.md).

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-browser-go-binary.md) | Browser app served by single Go binary (`go:embed`) | accepted |
| [0002](0002-dual-channel-tmux-rpc.md) | Dual-channel agent control: tmux PTY + pi RPC | accepted |
| [0003](0003-user-installed-pi.md) | Depend on user-installed Pi, no vendoring | accepted |
| [0004](0004-defer-frontend-framework.md) | Defer frontend framework — vanilla ES + vendored xterm.js | superseded by 0008 |
| [0008](0008-react-vite-tailwind.md) | React + Vite + Tailwind; tokens stay the design system | accepted |
| [0009](0009-lifecycle-surfaces.md) | Catalog from pi; auth via `/login`; MCP not in wizard | accepted |
| [0010](0010-pi-packages.md) | Packages via `pi install`; no in-app marketplace | accepted |
| [0005](0005-sqlite-store.md) | SQLite (pure Go) store — orchestration data only | accepted |
| [0006](0006-run-modes.md) | Agent run modes — one live pi process per agent | accepted |
| [0007](0007-https-mkcert-runtime-port.md) | HTTPS by default with mkcert trust; port configurable at runtime | accepted |
| [0011](0011-workspaces-and-agents.md) | Workspaces contain many agents; unbound agents in `ws_free` | accepted, amended by 0026, 0027 |
| [0012](0012-settings-vs-preferences.md) | `#/preferences` = PiCode; `#/settings` = pi GUI | accepted |
| [0013](0013-provider-accounts.md) | Extra logins in `~/.picode/accounts.json`; `auth.json` is the active slot | accepted |
| [0014](0014-local-backup.md) | Local directory snapshots of the PiCode environment | accepted |
| [0015](0015-browser-file-editor.md) | Browser file editor for the agent cwd (not an IDE) | accepted, explorer refusal amended by 0030 |
| [0016](0016-project-shells.md) | Project shells in tmux, as editor tabs | superseded by 0017 |
| [0017](0017-first-class-terminals.md) | First-class terminals (sidebar + main tabs) | accepted, amended by 0026 |
| [0018](0018-systemd-user-install.md) | `picode install` — systemd user unit, not Windows | superseded by 0020 |
| [0019](0019-terminal-file-tabs.md) | Ctrl+click a path in the terminal → editor tab | accepted, explorer refusal amended by 0030 |
| [0020](0020-desktop-provisions-wsl.md) | PiCode Desktop — Windows provisions the distro | accepted |
| [0021](0021-adopt-pi-session.md) | Adopt a Pi session by copying the JSONL | accepted |
| [0022](0022-git-graph-per-repository.md) | Git graph per repository — read-only, opened from any cwd | accepted, clone exception carved by 0034, amended by 0038 |
| [0023](0023-built-ui-is-not-committed.md) | Built UI is not committed; embedding moves behind a build tag | accepted |
| [0024](0024-terminal-settings.md) | Terminal settings — global defaults, per-terminal overrides, user presets | accepted, amended in part by 0025 |
| [0025](0025-open-tmux-catalog.md) | The whole tmux catalog is a settings surface | accepted |
| [0026](0026-sidebar-tabs-and-workspace-terminals.md) | Four sidebar tabs; workspaces own terminals | accepted |
| [0027](0027-workspaces-start-empty.md) | Workspaces start empty | accepted |
| [0028](0028-model-roles.md) | Model roles as an opt-in MIT pi package (`packages/pi-roles`) | accepted |
| [0029](0029-composer-extension-commands.md) | Composer `/` lists commands from the running agent | accepted |
| [0030](0030-file-tree-per-owner.md) | File tree per folder, read-only, changed files highlighted | accepted, diff refusal amended by 0032, graph-refresh precedent superseded by 0038 |
| [0031](0031-provider-usage-dialog.md) | Live provider usage dialog on `#/providers` | accepted |
| [0032](0032-working-tree-diff-and-reveal.md) | Change dots expand into working-tree diffs; Reveal in the host file manager | accepted |
| [0033](0033-roles-per-agent-overlay.md) | Model roles per-agent overlay (`PI_ROLES_AGENT` → `.pi/roles/<id>.json`) | accepted |
| [0034](0034-clone-remote-repository-workspace.md) | Clone a remote repository into a new workspace | accepted |
| [0035](0035-remove-workspace-delete-local-data.md) | Remove workspace can delete the local folder — opt-in, typed confirmation | accepted |
| [0036](0036-extensions-host-and-apps-tab.md) | Extensions host — apps on schema-driven primitives, Apps sidebar tab | accepted |
| [0037](0037-inbox-async-agent-human-messages.md) | Inbox — async agent↔human messages; core data plane, view as first app | accepted |
| [0038](0038-git-graph-v2.md) | Git graph v2 — inline detail, uncommitted row, search, token auto-refresh | accepted, amends 0022, supersedes 0030 on refresh for the graph |
