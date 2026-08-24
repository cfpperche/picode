# ADR index

One architectural decision per file, immutable once accepted. Superseding
an ADR requires a new ADR. Template: [template.md](template.md).

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-browser-go-binary.md) | Browser app served by single Go binary (`go:embed`) | accepted |
| [0002](0002-dual-channel-tmux-rpc.md) | Dual-channel agent control: tmux PTY + pi RPC | accepted |
| [0003](0003-user-installed-pi.md) | Depend on user-installed Pi, no vendoring | accepted |
| [0004](0004-defer-frontend-framework.md) | Defer frontend framework — vanilla ES + vendored xterm.js | accepted |
| [0005](0005-sqlite-store.md) | SQLite (pure Go) store — orchestration data only | accepted |
