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
| [0006](0006-run-modes.md) | Agent run modes — one live pi process per agent | accepted, session-visibility clause amended by 0039 |
| [0007](0007-https-mkcert-runtime-port.md) | HTTPS by default with mkcert trust; port configurable at runtime | accepted — "no app-level auth" superseded by 0049 |
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
| [0036](0036-extensions-host-and-apps-tab.md) | Extensions host — apps on schema-driven primitives, Apps sidebar tab | accepted, amended 2026-08-31 (iframe first-class in marketplace era; primitives frozen) |
| [0037](0037-inbox-async-agent-human-messages.md) | Inbox — async agent↔human messages; core data plane, view as first app | accepted |
| [0038](0038-git-graph-v2.md) | Git graph v2 — inline detail, uncommitted row, search, token auto-refresh | accepted, amends 0022, supersedes 0030 on refresh for the graph |
| [0039](0039-per-agent-session-ownership.md) | Per-agent session ownership, tracked in PiCode (`--session-id` for chat attribution; 0040 adds a private dir for pi's own TUI) | accepted, amends 0006, amended by 0040 |
| [0040](0040-per-agent-session-dir.md) | Per-agent `--session-dir` — extends ownership into pi's own native TUI picker | accepted, amends 0039 |
| [0041](0041-session-observability-dashboard.md) | Session observability dashboard (spend/activity/fleet) replaces the no-tabs-open home | accepted, amended by 0042 |
| [0042](0042-dashboard-v2-breakdowns.md) | Dashboard v2 — model/workspace/token/tool/reliability breakdowns, live refresh, fingerprint cache | accepted, amends 0041 |
| [0043](0043-browser-extension-native-host.md) | Chrome extension is a native-messaging client of existing agents | accepted |
| [0044](0044-mobile-supervision-shell.md) | Mobile shell is a supervision console (Now / Inbox / Work / More + agent and terminal screens), not desktop parity | accepted, amended (2026-09-02: Safari-tab sticky heads) |
| [0045](0045-automations.md) | Automations — daemon scheduler + webhook fire ordinary agent sessions; bounds, runs log, Inbox; v2: `/automate` drafts from the current agent, built-in templates | accepted, amends 0037 (source kind), amended 2026-09-01 (v2), 2026-09-02 (webhook through the gateway, notify URL, message runs deliver now) |
| [0046](0046-responsive-dialogs.md) | One modal primitive: Radix dialog ≥720px, Vaul bottom sheet below, enforced by a test | accepted |
| [0047](0047-web-push.md) | Web Push over VAPID in the standard library; presence-aware; per-device prefs | accepted, amended by 0048 (consumes the feed) |
| [0048](0048-change-feed.md) | The change feed — every mutation is a durable event, served over SSE with replay; polls become fallback | accepted, amends 0037 and 0047 |
| [0049](0049-auth-pairing.md) | Authentication — paired devices, install token, Host/Origin gate; modes off / remote / all | accepted, supersedes 0007 on "no app-level auth" |
| [0050](0050-tailnet-server.md) | A PiCode you own on a tailnet server — host/public URL settings, env drop-in, verified updates, provision server checks, off-box clients (`remote.json`); `tailscale cert` by SNI deferred to B.2 | accepted (B.1 + B.2), amends 0007, 0018/0020, 0043, 0049 |
| [0051](0051-shared-tailnet-server.md) | Shared tailnet box — a daemon per Linux user behind `picode gateway`, identity from `tailscale whois`, `/etc/picode/gateway.json`, `provision --shared` | accepted, amends 0049, 0050, 0020 |
| [0052](0052-public-access.md) | Public access — Google/GitHub login at the gateway (stdlib OIDC/OAuth), a plain listener behind a TLS proxy, signed session, hardening; a systemd-nspawn container per member | accepted, amends 0051, 0007 |
| [0053](0053-pending-session-resolves-at-spawn.md) | A pending session adopts its file at spawn (chat and TUI share one thread); the picker unions the private dir; the status bar follows the selected agent | accepted, amends 0039, 0040 |
| [0054](0054-extension-actuator.md) | Extension actuator — granted, visible act loop on the current tab (`picode-act` batches) | accepted, fills the hole ADR-0043 left |
| [0055](0055-internal-checklist.md) | Internal checklist — `pi-checklist` package: a `checklist` tool, a gate on the first change per task, a per-agent level (`changes` / `always` / `never`), one operator line per agent on both shells and a card in the chat | accepted, builds on 0010, 0037, 0048 |
