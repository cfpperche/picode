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
| [0011](0011-workspaces-and-agents.md) | Workspaces contain many agents; unbound agents in `ws_free` | accepted |
| [0012](0012-settings-vs-preferences.md) | `#/preferences` = PiCode; `#/settings` = pi GUI | accepted |
| [0013](0013-provider-accounts.md) | Extra logins in `~/.picode/accounts.json`; `auth.json` is the active slot | accepted |
| [0014](0014-local-backup.md) | Local directory snapshots of the PiCode environment | accepted |
| [0015](0015-browser-file-editor.md) | Browser file editor for the agent cwd (not an IDE) | accepted |
| [0016](0016-project-shells.md) | Project shells in tmux, as editor tabs | superseded by 0017 |
| [0017](0017-first-class-terminals.md) | First-class terminals (sidebar + main tabs) | accepted |
| [0018](0018-systemd-user-install.md) | `picode install` — systemd user unit, not Windows | accepted |
| [0019](0019-terminal-file-tabs.md) | Ctrl+click a path in the terminal → editor tab | accepted |
