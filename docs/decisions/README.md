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
| [0022](0022-git-graph-per-repository.md) | Git graph per repository — read-only, opened from any cwd | accepted, clone exception carved by 0034, amended by 0038; write refusal amended by 0078 for the Inspector's Git actions (user's shell, behind an interlock) |
| [0023](0023-built-ui-is-not-committed.md) | Built UI is not committed; embedding moves behind a build tag | accepted |
| [0024](0024-terminal-settings.md) | Terminal settings — global defaults, per-terminal overrides, user presets | accepted, amended in part by 0025 |
| [0025](0025-open-tmux-catalog.md) | The whole tmux catalog is a settings surface | accepted |
| [0026](0026-sidebar-tabs-and-workspace-terminals.md) | Four sidebar tabs; workspaces own terminals | accepted |
| [0027](0027-workspaces-start-empty.md) | Workspaces start empty | accepted |
| [0028](0028-model-roles.md) | Model roles as an opt-in MIT pi package (`packages/pi-roles`) | accepted |
| [0029](0029-composer-extension-commands.md) | Composer `/` lists commands from the running agent | accepted |
| [0030](0030-file-tree-per-owner.md) | File tree per folder, read-only, changed files highlighted | accepted, diff refusal amended by 0032, graph-refresh precedent superseded by 0038, file navigation amended by 0074 |
| [0031](0031-provider-usage-dialog.md) | Live provider usage dialog on `#/providers` | accepted |
| [0032](0032-working-tree-diff-and-reveal.md) | Change dots expand into working-tree diffs; Reveal in the host file manager | accepted, Open file navigation amended by 0074; write refusal amended by 0078 |
| [0033](0033-roles-per-agent-overlay.md) | Model roles per-agent overlay (`PI_ROLES_AGENT` → `.pi/roles/<id>.json`) | accepted |
| [0034](0034-clone-remote-repository-workspace.md) | Clone a remote repository into a new workspace | accepted |
| [0035](0035-remove-workspace-delete-local-data.md) | Remove workspace can delete the local folder — opt-in, typed confirmation | accepted |
| [0036](0036-extensions-host-and-apps-tab.md) | Extensions host — apps on schema-driven primitives, Apps sidebar tab | accepted, amended 2026-08-31 (iframe first-class in marketplace era; primitives frozen) |
| [0037](0037-inbox-async-agent-human-messages.md) | Inbox — async agent↔human messages; core data plane, view as first app | accepted, amended 2026-09-02 (consented switch), 2026-09-09 (terminal identity on items; replies ride the terminal's receiver; channelless blocking replies refuse visibly) |
| [0038](0038-git-graph-v2.md) | Git graph v2 — inline detail, uncommitted row, search, token auto-refresh | accepted, amends 0022, supersedes 0030 on refresh for the graph; write refusal amended by 0078 |
| [0039](0039-per-agent-session-ownership.md) | Per-agent session ownership, tracked in PiCode (`--session-id` for chat attribution; 0040 adds a private dir for pi's own TUI) | accepted, amends 0006, amended by 0040 |
| [0040](0040-per-agent-session-dir.md) | Per-agent `--session-dir` — extends ownership into pi's own native TUI picker | accepted, amends 0039 |
| [0041](0041-session-observability-dashboard.md) | Session observability dashboard (spend/activity/fleet) replaces the no-tabs-open home | accepted, amended by 0042 |
| [0042](0042-dashboard-v2-breakdowns.md) | Dashboard v2 — model/workspace/token/tool/reliability breakdowns, live refresh, fingerprint cache | accepted, amends 0041 |
| [0043](0043-browser-extension-native-host.md) | Chrome extension is a native-messaging client of existing agents | accepted |
| [0044](0044-mobile-supervision-shell.md) | Mobile shell is a supervision console (Now / Inbox / Work / More + agent and terminal screens), not desktop parity | accepted, amended (Safari-tab sticky heads; file/Git scope superseded by 0095) |
| [0045](0045-automations.md) | Automations — daemon scheduler + webhook fire ordinary agent sessions; bounds, runs log, Inbox; v2: `/automate` drafts from the current agent, built-in templates | accepted, amends 0037 (source kind), amended 2026-09-01 (v2), 2026-09-02 (webhook through the gateway, notify URL, message runs deliver now), 2026-09-09 (many schedules per automation: `automation_schedules`, zone per row, `schedule_id` on runs) |
| [0046](0046-responsive-dialogs.md) | One modal primitive: Radix dialog ≥720px, Vaul bottom sheet below, enforced by a test | accepted |
| [0047](0047-web-push.md) | Web Push over VAPID in the standard library; presence-aware; per-device prefs | accepted, amended by 0048 (consumes the feed) |
| [0048](0048-change-feed.md) | The change feed — every mutation is a durable event, served over SSE with replay; polls become fallback | accepted, amends 0037 and 0047 |
| [0049](0049-auth-pairing.md) | Authentication — paired devices, install token, Host/Origin gate; modes off / remote / all; loopback mints reused (2026-09-03) and ephemeral — revoked 10 min after the last request (2026-09-06) | accepted, supersedes 0007 on "no app-level auth" |
| [0050](0050-tailnet-server.md) | A PiCode you own on a tailnet server — host/public URL settings, env drop-in, verified updates, provision server checks, off-box clients (`remote.json`); `tailscale cert` by SNI deferred to B.2 | accepted (B.1 + B.2), amends 0007, 0018/0020, 0043, 0049 |
| [0051](0051-shared-tailnet-server.md) | Shared tailnet box — a daemon per Linux user behind `picode gateway`, identity from `tailscale whois`, `/etc/picode/gateway.json`, `provision --shared` | accepted, amends 0049, 0050, 0020 |
| [0052](0052-public-access.md) | Public access — Google/GitHub login at the gateway (stdlib OIDC/OAuth), a plain listener behind a TLS proxy, signed session, hardening; a systemd-nspawn container per member | accepted, amends 0051, 0007 |
| [0053](0053-pending-session-resolves-at-spawn.md) | A pending session adopts its file at spawn (chat and TUI share one thread); the picker unions the private dir; the status bar follows the selected agent | accepted, amends 0039, 0040 |
| [0054](0054-extension-actuator.md) | Extension actuator — granted, visible act loop on the current tab (`picode-act` batches) | accepted, fills the hole ADR-0043 left |
| [0055](0055-internal-checklist.md) | Internal checklist — `pi-checklist` package: a `checklist` tool, a gate on the first change per task, a per-agent level (`changes` / `always` / `never`), one operator line per agent on both shells and a card in the chat | accepted, builds on 0010, 0037, 0048 |
| [0056](0056-guest-tui-state-sensors.md) | Coding-CLI state in two tiers — tier 1 (built): guest CLIs and manual Pi TUIs report through scoped sensors on the terminal change feed (`working` / `needs-you` / no signal); tier 2 (deferred): promote guests to observe-only agents. Screen-scraping refused; ACP/control deferred | accepted 2026-09-03, tier split fixed in review; builds on 0017, 0024, 0048 |
| [0057](0057-tool-live-preview.md) | Tool live preview — a tool-agnostic `details.preview` contract rendered inline in the tool pill; no tool names in core, no second package system (pi's already exists, ADR-0010) | accepted |
| [0058](0058-providers-view-v2.md) | Providers view v2 — plan windows on the roster from a cache in three honest states, active-slot-only refresh, vendor identity in the vault, credential source, `pi auth check` as Verify, Pause beside Sign out, blast radius on the confirm | accepted, extends 0031, 0013 |
| [0059](0059-transient-rpc-burst-for-tui-replies.md) | Inbox replies to idle TUI agents borrow one exact-session RPC turn behind the unchanged terminal tab | superseded by 0060; had superseded ADR-0037's 2026-09-02 consented-switch amendment and narrowed 0006 |
| [0060](0060-inbox-replies-into-the-tui.md) | Inbox replies land directly in the running TUI via an injected receiver extension (`pi.sendUserMessage`), with a tmux bracketed-paste fallback and JSONL-row proof | proposed, supersedes 0059; its receiver/paste door is reused by 0078 stage 3 for Inspector asks |
| [0061](0061-compaction-policy-package.md) | Compaction policy as an opt-in MIT pi package (`packages/pi-compact`); `/compact-edit|model|on|off` wizard family; dormant until a config file exists (no defaults) | accepted |
| [0062](0062-terminal-cli-presence.md) | Authoritative ephemeral CLI presence in project terminals via wrapper leases, run IDs, process validation, and exact tmux/PID fallback; presence stays separate from lifecycle activity | accepted, amends 0056 |
| [0063](0063-whats-new-release-highlights.md) | Curated in-product release highlights, stamped-build auto-open, per-browser acknowledgement | accepted |
| [0064](0064-release-cadence.md) | Official release cadence and source/Stable release lanes | proposed |
| [0065](0065-docker-sysadmin.md) | Local Docker App and optional sysadmin tools share bounded, audited operations | accepted |
| [0066](0066-docker-project-groups.md) | Docker project groups inside the App, with saved disclosures and search | accepted |
| [0067](0067-docker-maintenance-plans.md) | Reviewed project operations, selected resource removal, shared jobs and supervised procedures | accepted |
| [0068](0068-docker-health-monitoring.md) | Opt-in project sampling, deduplicated incidents and supervised diagnosis | accepted |
| [0069](0069-agent-cli-terminals.md) | Dedicated CLI terminal control, inherited launch settings and invocation-scoped integration | accepted |
| [0070](0070-cli-launch-inspection.md) | Inspect launch defaults, copy terminal profiles and separate setup from observed activity | accepted |
| [0071](0071-desktop-task-reliability.md) | Explicit resident Windows task policy, read-only startup checks and scoped repair | accepted, extends 0020 |
| [0072](0072-independent-web-applications.md) | Independent desktop/mobile builds and presentation; explicit shared contracts and stable PWA identity | accepted; file/Git scope superseded by 0095 |
| [0073](0073-git-graph-worktrees.md) | The git graph shows every worktree's working tree — one dirty row per worktree, ref-addressed sibling reads, detached chips | accepted; write refusal amended by 0078 |
| [0074](0074-file-tree-inline-details.md) | Files and Changes share a local detail pane with document guards and root preconditions | accepted |
| [0075](0075-integrations.md) | Integrations: generic outbound webhooks and externally implemented MCP connectors | accepted |
| [0076](0076-bounded-tool-captures.md) | Bounded historical tool captures and transcript reconciliation | accepted |
| [0077](0077-tui-diff-panel.md) | Side diff panel for the pi TUI — an MIT extension drawn as a non-capturing overlay, never bytes into the PTY | accepted |
| [0078](0078-inspector-rail.md) | An Inspector rail follows the selected tab's owner and opens content in the center; PR through the host's gh; Git actions in the user's shell, run behind an interlock, or asked of a running agent through its prompt channel | accepted, amends the write refusals of 0022/0032/0038/0073 for the rail's Git actions |
| [0079](0079-sessions-under-agent-clis.md) | Sessions are a CLI capability: `#/clis/sessions(/<wsId>)` replaces the top-level route; old links redirect | accepted, extends 0069 |
| [0080](0080-llama-manager.md) | Dedicated llama.cpp manager and reliable connection results | accepted |
| [0081](0081-terminal-checklists.md) | The internal checklist follows the agent into its terminal: publish target falls back to PICODE_TERM_ID, terminal cards and panes carry the same line | accepted, extends 0055 and 0069 |
| [0082](0082-browser-capture-sidecar.md) | Browser capture as a standalone sidecar extension: bounded frames over RPC, no patched agent | accepted, supersedes the emitter placement of 0076 |
| [0083](0083-llama-operation-jobs.md) | Durable model operations, progress, cancellation and reconnect | accepted |
| [0084](0084-cli-terminal-session-recovery.md) | CLI terminal session recovery — pin the native conversation while alive, resume it in one click after a stop or restart | accepted, extends 0069 and 0079 |
| [0085](0085-session-forensics.md) | Session forensics — shutdown snapshot + boot diff, SIGHUP-immune pane roots, deploy log | accepted, extends 0084 |
| [0086](0086-process-cost.md) | The rite around a change costs less than the change: batched guarded deploys, scoped gates, `make close`, a 100-line handoff with per-session files, advisory capture parity | accepted, amends 0018 and 0084/0085; amended 2026-09-08 (docs site path `www/` renamed to `docs-site/`); amended by 0105 (deploy on request, no timer; captures at deploy; changelog fragments; byte cap) |
| [0087](0087-cli-lifecycle.md) | CLI lifecycle — update check, update, reinstall, uninstall by orchestrating the vendors' own commands as durable jobs | accepted, extends 0069 and 0083 |
| [0088](0088-cross-cli-session-handoff.md) | Cross-CLI session handoff — a conversation of one Agent CLI continues in another as a new native session (or a brief), with a preview of what travels and a lineage row; create-only exception to 0056 | accepted, extends 0069/0079/0084, amends 0056 |
| [0089](0089-cli-terminal-prompt-door.md) | User-initiated prompt door for Agent CLI terminals: stage files in the cwd, paste paths into the TUI; Inspector still must not type into a CLI | accepted, amends 0078 and 0002; amended (nested .gitignore, 7-day sweep; 2026-09-08: a pi terminal with a live receiver can be asked) |
| [0090](0090-llama-service-ownership.md) | Explicit llama.cpp service ownership, reviewed lifecycle and cache boundaries | accepted |
| [0091](0091-pi-only-agents-acp-waits.md) | Pi-only agents hold — guest CLIs stay TUI + tier-1 sensors; no agent-protocol client (ACP, app-server, SDKs) until the market converges on one standard, named re-measure trigger | accepted, closes 0056's ACP deferral as deliberate |
| [0092](0092-absent-checklist-renders-silence.md) | An absent checklist renders as silence — no line, no "No checklist"; the counter loses its accent | accepted, amends 0055 and 0081 |
| [0093](0093-cli-install-missing.md) | Install a missing npm-backed CLI through the same lifecycle lane; vendor curl installers stay guided | accepted, extends 0087 |
| [0094](0094-handoff-through-vendor-import.md) | A handoff writer may publish through the target CLI's own import command (OpenCode `import`, `hermes sessions import`); verification matches what the importer promises | accepted, extends 0088 |
| [0095](0095-mobile-files-git.md) | Mobile files, editing and Git workflows | accepted; amended 2026-09-08 (the graph's write vocabulary as a sheet) |
| [0096](0096-git-graph-write-actions.md) | The git graph acts on what you point at: target-bound actions in risk tiers, delivered through ADR-0078's three doors; git never runs in the service process | accepted, amends 0022/0032/0038/0073, extends 0078; amended 2026-09-08 after an adversarial review |
| [0097](0097-cross-cli-dashboard-metrics.md) | The dashboard measures every agent CLI, not just pi, and every metric carries a state saying whether that CLI could report it | proposed, amends 0041/0042 |
| [0098](0098-windows-clean-install.md) | Windows clean-machine install — the desktop exe installs the Linux binary and the runtime (tmux, Node, pi) itself; first run through `install.ps1` (no SmartScreen), winget as a second door, no paid signing; imported PiCode distro as a later opt-in | accepted, extends 0020 and 0093, amends 0003's first-run boundary; amended 2026-09-08 |
| [0099](0099-package-configuration-gui.md) | Packages with a known adapter get a config GUI that edits the extension's own files (workspace layer + agent overlay); reset is scoped, never one button | accepted, extends 0010/0028, supersedes 0033 §5 "No PiCode GUI page" |
| [0100](0100-pin-reminders.md) | Pin reminders — a `pin_reminders` row (once / interval with schedule-or-completion anchor / cron in a named zone) fired by the one-minute engine into an Inbox item that is the acknowledgement state; sticky notice + push as projections; missed slot fires once | accepted 2026-09-08, builds on 0037, 0045, 0048 |
| [0101](0101-settings-under-agent-clis.md) | Native Settings under Agent CLIs, contextual URLs and legacy redirects | accepted; supersedes 0012 navigation |
| [0102](0102-packages-under-agent-clis.md) | Native Packages under Agent CLIs, explicit scopes and compatibility redirects | accepted; supersedes 0010/0099 navigation |
| [0103](0103-providers-under-agent-clis.md) | Native Providers under Agent CLIs, machine scope and app-preserving OAuth return | accepted; supersedes 0058 navigation |
| [0104](0104-peer-communication.md) | Embedded MCP for direct session messages | accepted |
| [0105](0105-process-cost-second-review.md) | The rite runs in a fresh context, the living docs stop conflicting, deploy is the owner's call: one branch one session, changelog fragments, handoff without shipped work (100 lines and 8 KB), no deploy timer, captures at deploy, sharded server tests, `make adr` with a boundary line, architecture split per subsystem | accepted, amends 0086 |
| [0106](0106-conversation-launch-setup.md) | Private communication setup on conversation resume | accepted |
| [0108](0108-matrix-persistence.md) | Matrix persistence: one row per panel, six feed events, a subset layout patch under ifUpdatedAt | accepted |
