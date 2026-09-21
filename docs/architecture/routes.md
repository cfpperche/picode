# Application routes

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

The Go binary serves **two independent React applications** (ADR-0072):
`web/browser` at `/browser/` (the desktop surface) and `web/mobile` at
`/mobile/`. `web/desktop` at `/desktop/` is the shell's own bundle (ADR-0122),
including the Management page — never a surface the launcher sends a browser
to. Each has its own
entry, dependencies, UI, styles and build output. `web/shared` exports
contracts, client adapters, domain helpers and theme tokens through explicit
subpaths; it contains no React presentation. Source and resolved build checks
prevent imports between the applications. Tailwind scans only each app's
sources. A root launcher chooses once by legacy query, saved preference or
viewport (`max-width: 767px`), preserving the hash and other query parameters.
Explicit app paths win at every width. Rotation does not replace the app or
its connections. The desktop remains responsive, with a navigation disclosure
above the canvas on narrow screens. Both outputs ship atomically in one binary.

TUI agents and Agent CLIs use one terminal engine:
`web/shared/client/terminalRuntime.js` owns input, wheel/touch translation,
clipboard, resizing and socket wiring. Each independent app supplies xterm
and its own presentation, never an agent-specific terminal implementation.
`agentTerminal.js` resolves the runtime identity and HTTP owner for every
CLI. A live legacy Pi process wins over a newly allocated binding after a
failed restart (ADR-0162); only its address differs, not its renderer or
controls. Missing bound records never invent a legacy session.

Mobile agent terminal views render the same `TerminalScreen` as Agent CLIs:
toolbar, attachments, Files/Git, prompt snippets, keyboard accessory, loading,
retry and stopped states. Bound terminal links canonicalize to the owning
agent view. Pi Chat versus Terminal is chosen from the Work list agent menu
(Open chat / Open terminal); managed Pi stays chat-only and non-Pi agents
do not acquire a managed composer (ADR-0091). Both old `TerminalDock`
implementations are removed. Browser and desktop
use the same browser bundle; Canvas and tabs resolve the same runtime key
while keeping containing-tab and agent lifecycle ownership separate.
See the [acceptance matrix](../plans/agent-tui-unification.md).

Mobile owns four tabs: **Now** (needs-you queue, activity and results),
**Inbox**, **Work** (workspaces, agents, terminals) and **More** (settings and
Apps). Pushed agent, terminal, inspector and app-detail screens retain the
existing API behavior and compatible agent/terminal hashes. Conversation,
composer, terminal, settings and preview components are mobile-owned copies;
secondary screens load on demand and offer retry on loading failure. Mobile
has no desktop sidebar or Pin Studio. ADR-0095 adds mobile-owned Files and Git
tools, one full-screen view at a time. The Inspector (ADR-0078's rail
re-shaped for the phone) is one pushed screen per owner — Changes
(folder-grouped sums, the `All | This agent` session scope, multi-worktree
following), Files and PR — and, on the agent screen (chat and terminal
views) and the terminal screen, a header toggle opens
it as a right drawer over the content (Back or the overlay returns to
the agent); the change-shape logic is shared in
`@picode/shared/domain/inspector.js`, and the legacy `#/changes` screen
parses onto it. Its dialogs
are always sheets, including wide previews; desktop keeps responsive dialogs.
The v2 composer keeps its primary message and Send/Stop row compact; message
options (kind, attachments, voice and expansion) open on demand. Per-agent
in-memory drafts retain text, kind and images across screen changes, clear
only acknowledged content, and keep failed submissions available for retry.
Enter inserts a newline; Ctrl/Cmd+Enter sends. Settings opens over the mounted
conversation. Work uses searchable rows and a compact view selector; More
groups and searches tools. Sessions under Agent CLIs and Automations have
mobile-owned list/detail/editor flows using the existing server contracts.
Fleet reads retain successful sources after partial failures and replay feed
events received while a read is pending. Sessions and automation runs discard
outdated responses; initial errors offer retry instead of a missing-resource
claim or endless loading. Files supports owner-scoped browsing, bounded folder
search, text editing and media previews. Its document controller preserves
unsaved edits across reads and writes, handles mtime conflicts and guards
navigation with Save/Discard/Cancel. Git provides Changes, History and PR
views, branch/remote filtering, commit and sibling-worktree details, and the
existing Prepare, Run when idle and Ask agent actions. Its folder row is where
the reading workspace is named and switched: a sheet of the user's Git
workspaces, the rule itself in `web/shared/domain/workspacePicker.js` because the
browser's graph toolbar asks the same question (ADR-0022 amendment). Command delivery only
falls back after a confirmed conflict; an unknown network outcome is not
replayed. Both tools pin the owner folder and require explicit Follow after
a root mismatch. Compatible `#/file/`, `#/tree/` and `#/git/` links retain
owner identity and an optional root precondition. Workspace graph/commit
reads now match the agent and terminal APIs; supplied roots are checked
before all three perform their Git reads. See the
[v2 acceptance table](../plans/mobile-v2.md).
`#m-app` is pinned to `visualViewport` so the composer and a one-row
terminal extra-keys accessory stay above the software keyboard
(ADR-0044); the accessory follows a user tap, not attach-time focus,
and is not a second QWERTY.
On iOS home-screen installs (standalone), WebKit parks a status-bar-sized
strip below the layout viewport that no element can reach and still reports
`env(safe-area-inset-bottom)` inside it (WebKit 313800/254868); the mobile
shell detects that mode at bootstrap (`navigator.standalone`), treats the
bottom inset as already reserved — Safari and Android keep real insets —
and measures the gap as `--letterbox` (opaque status bar: half of
`screen.height - innerHeight`, so the status bar is not counted twice).
Auto on a letterboxed install docks `#m-app` into the strip (edge) so the
tab bar sits on the physical bottom; a pushed screen (terminal, agent) does
the same even when Layout is "low". "Low" keeps the tab buttons in the
layout viewport if the system clips the strip.
The status bar itself is opaque (`black` in dark, `default` in light): iOS 26
Liquid Glass frosts `black-translucent` over the header. Heads still pad with
`env(safe-area-inset-top)`, which is 0 when the bar is opaque.
Both mobile settings paths save via the agent
PATCH endpoint; changed tool mode restarts the same runtime, as on desktop.
The model catalog parser accepts capability rows only when both final columns
are yes/no; CLI setup diagnostics cannot become provider/model choices.
The mobile agent socket uses `web/mobile/src/lib/agentEvents.js`. Both clients
consume the change feed; presence follows the mounted app rather than width.

The PWA keeps the root worker registration and scope. Manifest identity
`/?mobile=1` preserves existing installations while `/mobile/` becomes the
start URL. Hashed assets have separate launcher/desktop/mobile caches; HTML
and APIs remain fresh. Cache failures fall back to the network; persistence
never blocks a successful asset response. Parsed source imports and transitive
shared build dependencies enforce the headless package boundary. See the [migration inventory and decision table](../plans/mobile-decoupling.md).
**Web Push (ADR-0047):** `internal/push` (stdlib VAPID + RFC 8291) posts
encrypted messages to each subscribed browser's push service; the store
holds subscriptions (`/api/push/*`), `sw.js` shows them and routes a tap
to `#/agent/<id>` or `#/inbox/<id>`. Triggers: a blocking inbox item, a
finished run filed to the inbox, a managed agent's dialog with no socket
open. Suppressed while any host-machine browser is online (presence).

Hash routes (ADR-0012). **Preferences** is PiCode-the-product.
**Settings** lives under Agent CLIs (ADR-0101), initially editing Pi configuration. Packages follows the same area (ADR-0102). Auth and MCP
stay on their own routes.

Mobile may add `?view=terminal` to an agent route. An explicit Pi
`?view=chat` is respected instead of being reset to Terminal on every
render. Bound terminals resolve to their owning agent; unbound terminals
remain `#/term/<id>`. Changing a view never starts a second writer.

| Hash | Surface | Owns |
|---|---|---|
| `#/` | Agent workspace | tabs, chat, terminal. Replaced by `#/agent/<id>` when an agent is open. With no tab open (or pinned via the logo — which also brings the workspace route back when the click comes from Agent CLIs, Browser, Preferences or Devices): the session observability dashboard once any workspace/agent/terminal exists — when something is blocked on the reader, one attention line above the numbers (count of Inbox questions from the app's badge + agent-CLI terminals whose hooks said `needs-you`, with one action to the Inbox or that terminal; ADR-0109's 2026-09-14 amendment declares the door), spend/activity/sessions tiles over the chosen range plus a **Fleet** tile that is live and machine-wide (managed agents beside agent-CLI terminals, with shell terminals counted apart; states `working` / `need you` / `idle` / `no signal`, four rows open their agent or terminal tab and `+N more` reveals the rest — ADR-0042's 2026-09-13 amendment), a bar chart of the range — one bar per day, or per hour for `Today`, where the series key states its own granularity (`2026-09-14T14` is 14:00) — and spend by model / workspace, tokens, tools, reliability, top sessions (ADR-0041, ADR-0042; `GET /api/sessions/stats`, fingerprint-cached, meters aggregated in parallel with a background warmup after boot, last good payload replayed from localStorage while revalidating, polled every 60 s while visible; the terminal list it counts comes from the app's fleet state, no extra fetch) — else the first-run blank slate. Not routed, derived from `noTabs && hasData`. |
| `#/agent/<id>` | Agent workspace | same shell; URL is the open agent (wins over saved tabs on load). An Inbox reply lands straight in this terminal (ADR-0060), so the tab never leaves the TUI. |
| `#/file/t/<id>/<path>` | File tab | text editor for a path under that terminal's cwd (Ctrl+click in xterm). `#/file/a/<id>/<path>` is the same for the Pi TUI dock; `#/file/w/<id>/<path>` reads through a workspace (ADR-0030). Preview \| Raw for svg, mermaid, md, html, png, pdf, audio, video, glb/gltf (`GET …/blob`; HTML through `POST /api/previews` → a sandboxed `GET /preview/<ticket>/<path>`, or the ticket's own `<label>.localhost` origin when the browser can reach it, ADR-0136/0137). |
| `#/tree/<w\|t\|a>/<id>` | File tree tab | lazy per-level browse and a **Changes** list from `…/gitstatus`, with changed files and their folders dotted. Tab identity is the canonical root (`d:<root>`), so owners of one folder share a tab. Files and Changes select the editor/preview or diff in one resizable local detail pane (ADR-0074). The toolbar's first item is the workspace the tree reads through — the same picker the graph wears, listing every folder and not only repositories: the same folder swaps the owner in place, another folder renames this tab to it (or selects the tab that already holds it, which takes the pick). The root comes from the picked workspace's own `…/browse`, never from a path in a URL. |
| `#/git/<w\|t\|a>/<id>` | Git graph tab (ADR-0022) | one tab per repository (`g:<git-common-dir>`): lanes and refs read through the owner's folder, a dirty row per worktree with the agents living in it (ADR-0073), commit and sibling-worktree diffs, and the write actions that travel through the owner (ADR-0096). The toolbar's first item is the workspace the history is read through — a picker of the reader's workspaces, each with its branch: a sibling worktree of this repository swaps the owner in place, another repository renames this tab to it (or selects the tab that already holds it, which takes the pick). A repository is never resolved from a path: the key comes from the picked workspace's own `…/git/head`. |
| *(no route)* | Inspector rail | the right-hand Changes/Files rail beside the center (ADR-0078). Per-viewer state (`picode-inspector-open`, `-w`, `-tab`), like `termView` and the sidebar width; it follows the selected tab's owner and opens content as `#/file/…` tabs. |
| `#/clis/<cli>/settings` | Native CLI settings (Pi first) | Pane of the selected CLI. Global + Keys without context; `?agentId=<id>` adds the actual workspace and agent layers. Composer `/settings` includes the agent; `/scoped-models` adds `focus=scoped-models`. Old `#/clis/settings/pi`, `#/settings` and mobile `#/more/settings` rewrite. |
| `#/preferences` | PiCode chrome | **appearance** — the app theme (Light / System / Dark) and nothing else: an app's own settings belong to the app (ADR-0109's 2026-09-11 amendment), which is why the Canvas's ground moved to its `⋯` menu on 2026-09-11 — **terminal** (xterm look), notifications, server (port, bind, public URL, who must pair, install token), **backup** (ADR-0014); tabs `#/preferences/<section>` |
| `#/clis` | Agent CLIs | CLI catalog, installation checks, launch defaults and activity-reporting switches. The strip is **CLIs \| Messages**. A CLI's page hosts **Launch**, **Terminals**, **Sessions**, **Providers**, **Settings**, **Packages** and **Connectors** (a run/setup split). Old Settings/Packages strip addresses and `#/integrations` connectors rewrite onto those panes. `#/clis/new/<cli>` and `#/clis/terminal/<id>` edit launches. Desktop sidebar header / command palette and mobile More open this surface. The old `#/preferences/status` address redirects here. |
| `#/system` | Machine facts | host, network, deps, version (read-only) |
| `#/clis/<cli>/providers` | Native providers pane | Pi: catalog + signed-in state; Sign in; search; **custom provider definitions** (ADR-0129); **plan windows on each account row** from the usage cache, live / stale-with-age / a reason (ADR-0058); vendor identity (email, plan); credential source (vault or an env var); **Verify** via `pi auth check`; **Usage** dialog per vault account (ADR-0031); Pause beside Sign out; 7-day spend per provider; Sign out names the agents and automations that break. `#/clis/<cli>/providers/new` opens Add provider; `#/clis/<cli>/providers/custom[/<id>]` opens the Custom provider page (new / Edit). Old `#/clis/providers*` and `#/providers*` rewrite here. Non-Pi CLIs stay blocked (ADR-0103). |
| `#/clis/<cli>/connectors` | Pi MCP connectors | adapter manager on the selected CLI: list / add / toggle / remove / **Use from…**. `#/integrations`, `#/integrations/connectors` and `#/mcps` rewrite here. |
| `#/integrations/webhooks` | Webhooks (ADR-0075) | signed durable event delivery, tests, pause, removal and secret rotation. PiCode surface, not a CLI pane. |
| `#/clis/<cli>/packages` | Native CLI packages (Pi and the eight guest CLIs, ADR-0167) | Pi: machine / workspace (`pi install`) / this agent (`-e` on start) (ADR-0010) with **Configure** → `#/clis/<cli>/packages/config/<pkg>`. Guests: the CLI's own scopes, plugin verbs and marketplace, never an agent scope; a verb the CLI lacks reads as one line instead of a control. Old `#/clis/packages/pi*` rewrite. |
| `#/automations` | Automations (ADR-0045) | list with enable switch, schedule line (every rule), 30-day runs sparkline, last run, Run now; `#/automations/new` editor (a list of schedules, each presets → cron + label + switch in the browser's zone; webhook, limits); `#/automations/<id>` detail + runs table naming the rule that fired. Polled every 15 s while visible. |
| `#/snippets` | Snippets (ADR-0130) | host library of reusable prompts and commands. List + search + star/archive; `#/snippets/new` and `#/snippets/<id>` editor. Composer `/snip:` and palette **Send snippet** expand into the focused managed agent; terminal menus **Send to terminal…** (CLI panes) and **Run command…** (shells, one confirm that shows the exact command). |
| `#/devices` | Devices (ADR-0043 + ADR-0049) | one surface for identity and liveness: paired sessions (Forget, Forget offline in one confirmed click, Pair a device with QR/link) with an online dot from the presence ping, which carries the session it came from; unpaired-but-online entries appear only in mode `off`. Access rules and the install token are in Preferences → Server. Auto-minted loopback browser sessions are ephemeral: the housekeeping sweep revokes a row once no authenticated request has refreshed it for 10 minutes, so closed headless-QA browsers leave without a manual Forget (ADR-0049 amendment 2026-09-06). |

A tab owns its surface's state for as long as it is open: terminals, file
trees, git graphs and apps each keep one mounted instance per tab, hidden
(not destroyed) while another tab is selected, so expanded folders, scroll
offsets, loaded history, searches and the open item survive a switch and die
only with the tab. A hidden surface takes no part in the window-focus refresh
— revealing it refetches instead, and only when its last read is older than
10s; the git graph refreshes on demand only, so a reveal never refetches it.

Directory browsing adds optional `ignored` metadata using one bounded Git
`check-ignore` call per listing. Desktop renders ignored names in muted italics;
tracked files and ignore exceptions remain normal. Non-Git folders and Git
failures retain ordinary browsing. Existing directory exclusions are unchanged.

The file tree's selection is local to its tab: changing Files/Changes only
changes the navigation list. `FilePane` supplies both standalone and embedded
layouts from one document controller (`web/desktop/src/lib/fileDocument.js`, mirrored in `web/mobile`). Dirty
replacement, switching to a diff and closing the containing tree tab share
Save/Discard/Cancel. A failed write retains the editable draft. An unchanged
refresh preserves editor history/scroll; dirty buffers and edits made while a
read is pending are never overwritten. Preview/Raw keeps CodeMirror mounted.
Hidden tabs keep drafts, and browser unload uses native unsaved protection.

Composer `@` lists files in the agent cwd (`GET /api/agents/{id}/files`), plus other agents and skills (mentions in this prompt, not a message to that agent).
Composer `/` also lists **extension commands** from the running managed agent
(`GET /api/agents/{id}/slash` → RPC `get_commands`, ADR-0029). Picking one
sends `/name` as a prompt. Stopped agents omit that list. Names that collide
with a PiCode command are dropped.
Click a path on an `edit`/`write` card (or the turn's file names) opens a closable card in the thread. **Open in tab** is the same `#/file/a/<id>/<path>` as the terminal. Save writes the file in the tab. A stale mtime is 409 (open again). Keep/Undo on the diff card: Undo rewrites the old lines (or Open if the file moved).
The desktop shell is three columns: the left sidebar, the center (tab strip and one surface at a time) and, since ADR-0078, a right **Inspector** rail. The rail follows the selected tab's owner (agent, terminal or workspace — the same owner that authorises file reads, resolved from the tab id like the git graph and folder tabs do) and keeps its last anchor beside tabs without a folder (apps). **Changes** (default) lists the working tree as a folder tree with per-file and per-folder `+N −M` from `…/gitstatus`, an `Uncommitted · +N −M` total, the branch and worktree, and beside an agent an `All | This agent` scope that intersects the tree with the paths the session's `edit`/`write` tools named; **Files** is the lazy project tree with a filter over loaded rows. The rail hosts no editor: a file opens the existing `#/file/…` tab, a change opens the same tab in a **Diff** view (`WorkingDiff`), and "View diff" / "Open file" swap the two in place — the view is per-tab viewer state, not part of the tab id or the hash. Every read pins the anchor's folder as the `root` precondition (ADR-0074); a background 409 becomes one blocked line — "This terminal moved to …" with **Follow** — that reads the live cwd and never retargets on its own. Live updates: `git.updated` for the pinned root and any followed checkout, feed open/reset, focus/visibility, and the terminal row's live cwd/dirty facts; no interval. **Changes** also follows dirty linked worktrees of the anchor's repository: `…/gitstatus` carries them as `worktrees[]` (dirty only, cap 8, `worktreesTruncated` counts the rest), and the tab shows one clean anchor's single dirty sibling behind a **Following** pill with Back (Back fixes the anchor until Refresh, offering the sibling as a View switcher instead) or one branch-headed group per checkout when several are dirty. While the pill is up the branch chip, Git menu and commit dialog address the followed checkout — the terminal is born in its folder, the ask text names it — and the PR tab stays on the anchor. A worktree file opens the same `#/file/…` tab through `?worktree=<branch-or-commit>` (the gitgraph selection, server-validated against `git worktree list`, now on text/blob/diff/save/preview reads); the tab's checkout rides beside the tab id in persisted viewer state (`picode-file-worktrees`), so a reload never reads one relative path through another tree. The fleet watcher inspects linked worktrees (cap 8 per repo, re-listing the set every 10th tick) and publishes their flips as path-only `git.updated` events — no workspace or agent ids, so the fleet pills keep describing the anchor — and followed groups stay live. Width (260–560, default 320), open state and tab are localStorage preferences (`picode-inspector-*`, no hash route); a viewer who never toggled gets the rail open at ≥1440px; it shrinks before it hides and hides when even 260px would push the conversation under 640px, or in the ≤767px shell. Toggle: the tab-strip button, `Ctrl+.` / `Cmd+.` (`app.inspector.toggle`) or the palette. A fourth tab, **Servers**, lists what is listening on this machine (`GET /api/devservers`): what each port answered (page / api / not answering yet), who holds it and since when, one click to open a page in PiCode's own browser surface, and a row menu with Copy address, Show terminal, Stop server… and Hide — stop and hide need the identity of the process behind the port, so a port PiCode cannot attribute offers neither (ADR-0151) — `docs/architecture/devservers.md`. Servers is the one tab that does not follow the anchor — the machine's listeners exist whether or not something is selected — so the tab row renders even with no owner at all: Files + Servers there, and Files carries the "Open an agent or terminal to inspect its files." line. A third tab, **PR**, shows the branch's pull request through the host's `gh` (`…/pr`): number, state, review decision, checks, `+N −M`, an "Open on GitHub" link; "No pull request" and "not logged in" offer one action that pre-types `gh pr create --fill` or `gh auth login` into the owner's terminal (`POST /api/terminals/{id}/type`), never submitting it. A **Git actions** menu prepares Fetch, Pull, Push, Commit, Commit and push and Create pull request as exact commands in a plain idle terminal of the folder (reused when one exists, never a terminal hosting a CLI), through the same `type` route — the human presses Enter; with the menu's "Run when no agent is working here" checkbox on (per viewer), the `run` route presses Enter itself when its interlock finds the repository idle and otherwise prepares the command with a note saying who is busy; the branch chip shows `↑ahead ↓behind`, `unpublished` or `detached`. For every agent running in that repository (managed or in its own terminal, anchored one first, at most three) the menu adds an "Ask <name>" submenu with the same actions: `POST /api/agents/{id}/ask {text, root}` enqueues a plain-language prompt (the folder, the branch, the action and its rules) through the agent's own channel — a `prompt` task the runtime delivers at once or as pi's `follow_up` once a streaming turn ends, or ADR-0060's receiver/bracketed-paste door into a TUI's pane — so the agent runs git in its own turn, under its own judgment; nothing new runs git here. Its More menu offers Reveal, Open git graph and Open as tab (the ADR-0074 folder tab stays the deep-review host). The rail is the host for later right-hand panels (PR, browser preview, search, tasks): one rail, never two.
**Fullscreen (focus) mode.** One shell-wide, per-viewer mode — not a route and not per tab: `picode-focus` in localStorage, alongside the rail's `picode-inspector-*`, plus `picode-focus-seen` for the one first-run toast. On, the sidebar, the tab strip and the rail leave the layout and turn invisible (`position: fixed` + `visibility: hidden`, never unmounted, so hidden chrome keeps its state exactly as a background tab does) and the selected surface takes the whole window; every live xterm refits on the transition. Three 6px hot zones bring one panel back as an overlay *above* the surface, so a reveal never resizes a pane: **left** the sidebar, **top** the tab strip (with an icon-only **Leave fullscreen** control at its right end, its label and chord in the hint, so tabs can be switched without leaving), **right** the rail — and only while the rail is on screen, which the shell reports as it changes. Each reveal waits 120ms of pointer dwell and closes 250ms after the pointer leaves both the panel and its strip; the top zone wins in a corner. A reveal a *control* asked for — the **Inspector toggle** at the end of the tab strip, whose panel the mode otherwise makes reachable only from its edge, and the same row from `Ctrl+.`, the context menu or the palette — appears at once and is pinned: the pointer never had to enter the panel for it to open, so it leaves on the same toggle, on Escape or with the mode, while a dwell on another strip still replaces it. Showing the rail inside the mode docks it when it was closed (the panel has to exist) and reveals it; hiding it there takes only the overlay down, so the mode never undocks what the viewer set outside it. Entry: the generic context menu, the terminal pane's menu (`buildTermMenu`, since a pane never shows the generic one), the palette, and `app.fullscreen.toggle` (`Ctrl+Shift+Enter` / `Cmd+Shift+Enter` — Enter reads the same on every keyboard layout, and one modifier is already the terminal's newline). The **git-graph menu** keeps its own rows and offers nothing here: it acts on a commit, not on the shell. The mode belongs to the tabs: the row is dropped in the ≤767px shell, which has no sidebar or rail to hide, and on a page route (`#/preferences`, `#/clis/*`, …), where the tab strip lives inside the hidden workspace view and the top reveal would carry no way out — a window narrowing to the column shell, or a navigation to a page, leaves the mode. **The browser window is not part of the mode** (2026-09-11): it hides PiCode's chrome and stops there. Until that day entering also called `requestFullscreen()` on the document element (with `navigator.keyboard.lock()` first, in the same gesture) and the two states followed each other in both directions; the owner removed it because the browser owns Escape in real fullscreen — it ate the first press to leave fullscreen and the sync then ended the mode, so the designed two-step survived only in a browser that refused fullscreen, and the rare case was costing the everyday one. The window is the reader's: `F11` is their gesture, it stacks on the mode with nothing to synchronise, and PiCode neither requests fullscreen nor listens for `fullscreenchange` — a mode entered inside an F11 window leaves without touching it, and a reload restores the whole mode because no half of it was ever the browser's. The price is the **keyboard lock**: the API captures keys only while the *page* asked for fullscreen, so the browser's reserved chords — `Ctrl+T`, `Ctrl+W`, `Ctrl+N`, `Ctrl+Tab`, `Ctrl+1–9` (`web/shared/domain/browserChord.js` is the set) — now stay with the browser in every window, and Codex's `Ctrl+T` or Claude Code's task toggle cannot reach a CLI running in a browser tab; the cancelable class is unaffected (xterm already cancels and encodes it inside panes), and `browserChord.js` survives as the guard that keeps PiCode's own defaults off a chord the page can never receive. **Escape** closes an open reveal and then, on a second press, the mode — both presses reach the app, since nothing is in front of it to eat the first — except inside a terminal pane, where the CLI owns Escape and the way out is the Leave control, the chord or the menu row. Pure logic and its decision table live in `web/shared/domain/focusMode.js` (`node --test`); `web/desktop/src/lib/useFocusMode.js` owns the pointer, Escape and the refit. Adapted from **Zen Mode** in VS Code, Cursor and Zed — their one command that hides every panel, and their double-Escape exit; changed on three points: Zen Mode leaves the chrome unreachable until you leave it (here every panel comes back on its edge), it re-centres the editor (here nothing is re-centred — a terminal must keep every column), and it asks the OS window for real fullscreen (here that is the reader's own `F11`, never ours).

The sidebar has five flat tabs, one kind each (ADR-0026, fifth added by ADR-0036), in order: **Workspaces** (the landing tab — one collapsible card per workspace holding its agents and its terminals; no section-level collapse. The header is one line with two controls that stay visible at rest: **New** — agent, shell terminal or Agent CLI terminal — and a menu holding Files, Git graph, Sessions and Remove. Files and Git graph read through the workspace itself (`#/tree/w/<id>`, `#/git/w/<id>`), so an empty project still opens its files and its history; Git graph is absent on a folder that is not a repository, and Sessions while the workspace has no agents, since the route answers 409 there. Path and branch live on the agent and terminal rows, not repeated on the card above them), **Agents** (free agents, name-sorted, no hierarchy — agent and terminal rows share one flat supervision shape), **Terminals** (free terminals only), **Apps** (a grid of app tiles drawn from `GET /api/apps` manifests — numeric badge for actionable counts, dot for activity, aggregated onto the tab icon; a tile opens the app as a main tab `x:<id>` / `#/app/<id>`. A manifest names its surface (ADR-0109): `""` is the primitives view `AppSurface` renders from `/api/apps/{id}/view`, drawn as a page frame rather than a canvas since 2026-09-14 (the route pages' `settings-wrap` + `settings-head`, Refresh and Close at the head's right, the view's own tabs as an underline nav inside the card, the filter in the card toolbar with the list's own bulk action (an actions block declared `Pane: "list"`, like Inbox's "Clear all done") right-aligned on the same row — detail-pane and unpaned action rows stay in their panes — and one line + one action for empty, blocked and error — `docs/plans/app-surface-parity.md`; it is the host's markup, so every primitives app gains it at once and the phone's own `AppSurface` is untouched); `"native"` is a component compiled into the shell, registered per shell by app id (`web/desktop/src/lib/nativeApps.js`) and mounted with one `host` prop — `{fleet: {workspaces, freeAgents, terminals}, openTabs, openTab, openInteractive, revealAgent, openFileTab, feed}`, the client twin of Go's `apps.Host` and the whole API a native app gets, plus `initialPath` / `onPathChange`, the deep-link pair `AppSurface` has. The desktop registers **Canvas** (`canvas`, `docs/architecture/canvas.md`). The manifest id and its icon key are `canvas` since ADR-0118; the desktop registration, the `#/app/canvas/<canvasId>` hash and the redirect from the old `#/app/matrix[/<id>]` links land with the surface rename. A shell without that surface keeps the tile honest: the desktop dims it as needing a newer PiCode, the phone lists it as Desktop only and answers `#/app/<id>` with one line and Back). Those are the app's whole reach into the shell: ADR-0109's 2026-09-11 amendment makes the doors a closed list — the tile and its badge, that tab, the app's own body, the manifest `icon` key the host's own map draws, the `host` object and this route — so an app never adds a Preferences group, a section inside another surface or a row in a host list (`web/tools/app-boundary.test.mjs` asserts the direction). Beside the first-party tiles the grid carries **user-installed webapps** (ADR-0147): the header's **+** installs from a URL — the daemon resolves the page itself (PWA manifest first, then favicon; the site must answer or the install is refused; a detected manifest also yields `start_url`/`scope`/`display`/`theme_color`, and a **bare** typed address launches from the manifest's `start_url` while a typed deep link wins) and stores name, URL and icon under `/api/webapps`; a tile opens the work browser on the stable tab `w:app-<id>` — since ADR-0153 a partitioned install runs in **its own WebView2 data folder** (own logins, own clear-data blast radius; legacy installs share the work profile), a leading `(N)` in the page title becomes the tile badge while that tab is open, the tile menu offers Open/Rename/Remove (Remove closes its open tab), and in a plain browser the click says web apps need PiCode Desktop and does nothing else. An app-like `display` from the manifest opens the tab **chromeless** — the viewport is entirely the page; Ctrl+F summons the find bar and reload/zoom stay engine accelerators; `display: browser` keeps the full toolbar) and **Pins**. Nothing appears in two tabs. Terminals are first-class shells (ADR-0017): **+** on the Terminals tab creates a free one (`POST /api/terminals` → tmux `picode-sh-<id>` in `$HOME`); the workspace card's **New** menu creates one owned by it, born in the workspace folder (`workspaceId` in the POST body — and every reader, the creation response included, names that folder from the moment the session exists: a live `#{pane_current_path}` read in the same instant answers with the *server's* directory, the daemon's own cwd, until tmux has polled the new pane; `Manager.bornCwd` holds the record for five seconds, measured 2026-09-15). Either opens on the main tab strip (`#/term/<id>`). Closing the tab detaches; Remove kills tmux; removing a workspace kills its terminals with it (the cleanup dialog warns with the count from the preview). Not tied to an agent. A terminal row separates **CLI presence** from **activity** (ADR-0062): a wrapper lease identifies Claude Code, Codex, Grok, Hermes Agent, or Pi with a run id, while lifecycle hooks report `Working`, `Needs you`, or quiet `Ready`; when no wrapper announcement is in memory (daemon restart, unwired sessions), reconciliation revives presence from the pane's process tree — exact pane command, or a `/proc` walk that matches the wrapped CLI through wrapper shells and interpreters, validated by PID plus process-start token, and dropped the moment the CLI exits. No presence or activity is inferred from terminal pixels. Supported CLI badges use each runtime's official mark — the same transparent SVG source the provider faces use, then the vendor's own assets as fallback links — filling the same 22px face slot as agent rows with no chip behind the image; the compact text mark in a boxed badge is only an asset-load fallback. Agent and terminal rows lead with identity/status, keep live path and branch as subdued actions, and put secondary actions behind a menu. The agent's Pi TUI view renders through the **same TermSurface/ShellTerm component** as terminals (same xterm.js options, wheel, keys, links, envelope) — one engine, one look; managed mode shows a one-line hint with an Open TUI action instead. Every `.term-pane` (agent TUI, Agent CLI terminal, plain shell) uses **one** right-click catalog (`buildTermMenu` + `paneCapabilities`): the `/term/` section order, with rows omitted when they have no door. Chat and the composer stay on the generic menu (isolation is DOM — `.term-pane` — not `#/agent/<id>`). `dataset.termKind` is the handler address (`/api/agents/{id}` vs `/api/terminals/{id}`), not a visibility axe. **No terminal surface draws a scrollbar of its own**: the web terminal is a tmux client and tmux attaches on the alternate screen, so xterm has no scrollback — the two one-pixel bars it can still paint at the right edge (its viewport's, and its own, one pixel wide because `overviewRuler.width` is also its scrollbar width) and the overview ruler's outline are hidden, and the fit keeps reserving that one pixel instead of a 14px gutter (`web/desktop/src/styles/app.css`, `web/mobile/src/mobile.css`, `web/shared/domain/termTheme.js`; guard `scripts/term-scrollbar.test.mjs`). The bar a reader sees belongs to whoever holds the scrollback — tmux's copy-mode indicator, or the TUI's own scrollbar (`pi` paints one) — and the wheel keeps scrolling it (`termWheel.js`). Ctrl/Cmd+click a path under the **live** pane cwd (`tmux #{pane_current_path}`, `GET /api/terminals/{id}/cwd`) opens `#/file/…` on the same strip (`GET/PUT /api/terminals/{id}/text`). `cd` then a relative path opens the file in the new folder. http(s) opens in the browser. Paths outside that live cwd are not links. Keys (Preferences → Terminal): Shift+drag select, Ctrl+C copy if selected, Ctrl+V paste. A gear after **+** opens the defaults every terminal inherits; a gear on a row opens that terminal's overrides (ADR-0024).

The pane has one state that is not a terminal: a CLI terminal whose process is gone (`launchCli` with `running: false`), and a pane whose open refused. Both are `TermSurface`'s own message, drawn inside **TermWindow — a picture of a terminal window** (owner call, 2026-09-14, after the ohmyz.sh mock the owner sent): titlebar with the macOS traffic lights, the terminal's own name and folder as the centered title (`codex — <folder>`, full cwd on hover), the content — a mark, what is true (PiCode restarted under it, the last launch failed, or the plain stop), the pinned conversation (ADR-0084) a **Resume last session** would reopen as its meta line — what the button used to keep in a tooltip — and the actions — hugged inside, the whole window centered on the pane (max 520px) instead of filling it. Nothing pinned is stated (`no session to resume`) rather than left as a button that is missing for no reason, and the one action left is the navigation, so the chrome never acks what the record does not hold. **The page follows the app's theme; the window is dark in both** (owner call, 2026-09-14): an empty pane is app chrome — there is no terminal in it to keep a terminal theme for — so the pane carries `is-empty`, the ground becomes `--bg-base`, and the window keeps one fixed palette pinned to the dark theme's own values (`--tw-*` on `.term-window`, `web/browser/src/styles/app.css`, `web/mobile/src/styles/mobile-tools.css`) — it is a picture of a terminal, the same in both themes, and the quiet action inside wears the window's chrome while the primary keeps the accent. A *live* terminal keeps the terminal's own theme (`data-term-theme`, `termTheme.js`), which is independent of the app's. The smallest Canvas panel is 256x224 px, so under 300px the window hides its title text and stacks the actions (`@container`), which is also what a narrow window gets.

An Inbox answer to a TUI agent lands directly in its running terminal
(ADR-0060). Every spawned agent TUI carries PiCode's receiver extension
(`<dataDir>/intercept/pi-inbox-reply.ts`, injected with `-e`); it says hello to
`POST /api/agents/{id}/tui-hello` and consumes one-shot reply files under
`<dataDir>/tui-inbox/<agentID>/`, submitting each through
`pi.sendUserMessage` (queued natively mid-turn) and confirming via
`POST /api/agents/{id}/tui-ack`. Without a fresh hello the daemon types the
reply into the pane itself — tmux bracketed paste plus Enter (a legacy TUI
tradeoff the owner accepted). Delivery truth stays the exact session JSONL:
the reply counts only when the captured file gains the full-payload user row;
otherwise the task fails and the Inbox item reopens with the response
prefilled. Boot reconciliation settles pending replies the same way — no
holders, leases, or fail-closed startup remain. `POST
/api/agents/{id}/open?restart=1` still force-replaces a genuinely dead pane.
Mobile start/stop uses agent-scoped routes inside multi-agent workspaces.
A question filed by `ask_human` from pi running as an Agent CLI terminal
arrives with `sourceKind: "terminal"` (pi-inbox stamps `PICODE_TERM_ID`,
ADR-0037's 2026-09-09 amendment), and its Inbox reply takes the same
receiver door in reverse: `DeliverTerminalReply` preflights the terminal
(is pi, live, receiver fresh, the hello names a session and it is the
item's exact session) and parks done on send, reopening with the response
preserved on every failure — no task row, the queue belongs to agents.
An item that predates per-question session stamping carries no session of
its own; its address is then resolved at reply time from the terminal's
pinned last session (`TerminalLaunch.LastSession`), and delivered only when
that pin names the same file the receiver is showing — two independent
sources, one answer (`docs/plans/inbox-terminal-address.md`). A pin that
disagrees with the pane, a pin whose file is gone, and `system`-sourced
items from pi launched outside the launcher all refuse with the item left
open.
Every pi that inherited the terminal id watches that terminal's reply
directory (a nested `pi -p`, a print-mode run, the TUI itself), so each
reply file names the pid whose hello the daemon accepted and only that
process consumes it; a process with no session of its own leaves the file
for the one that can answer (ADR-0060's 2026-09-11 amendment). **Ignore is
not a reply**: it sends nothing, so it closes the item locally however the
terminal looks right now — exactly as it does for an agent. A source with
no identity at all (`system`) has no channel: `RespondAndForward` refuses
with `ErrNoReplyChannel`, the item stays open, and the UI says to answer
it in the terminal — replying never closes an item while nothing was
sent.

Paste/drop images send `POST /api/agents/{id}/prompt` (live RPC, not the task table).
The composer also opens a device file picker (Photos / camera / files on a
phone) so attach does not depend on clipboard paste.
Agent CLI terminals have no composer. ADR-0089 is a user-initiated prompt
door: `POST /api/terminals/{id}/drop` writes a file under
`<cwd>/.picode/drop/`, and `POST /api/terminals/{id}/prompt` pastes the
caption plus `@path` into `picode-sh-<id>`. Proof is tmux accept ("Sent
to the terminal"), not model delivery. Inspector type/run/Ask still must
not target a CLI TUI (ADR-0078). Plain shells have no attach bar.
Pasting files (Ctrl+V / Ctrl+Shift+V with screenshots or files on the
clipboard, or the PiCode menu's Paste row) opens the same bar seeded with
the staged files, and any accompanying text becomes the message;
text-only pastes keep the native paste. The keydown re-reads the clipboard
and fires an equivalent paste event, because stopping xterm's ^V also stops
the browser's own. The
staging folder stays inside the project deliberately — the CLI reads the
path itself, confined to its own cwd — but is never global-data material
and never touches the project's own tracked `.gitignore`: a nested,
uncommitted `.picode/.gitignore` (`drop/`) covers it even in a project
with no root `.gitignore` at all, and a drop older than 7 days is swept
on the next one into the same project (no daemon; nothing else ever
deletes a staged attachment once the CLI has read its path).
`!cmd` runs in the agent cwd via `POST /api/agents/{id}/bash` (`abort_bash` cancels); output renders in the chat and joins the next prompt.
MCP manager: `GET/POST/PATCH/DELETE /api/mcp` reads and writes the adapter files
(`~/.pi/agent/mcp.json`, `<cwd>/.mcp.json`, `<agent cwd>/.pi/mcp.json`). `?agent=`
or `?workspace=` that is not an agent/workspace (terminal tab `t:…`, stale id)
is ignored — same as packages — so adapter status still comes from machine
packages. A workspace terminal tab still carries that folder as MCP/Packages
context. Add accepts
optional `env`, `headers`, `auth` (`oauth`|`bearer`) and `bearerToken`.
Live status (`idle`/`live`/`failed`/`signin`) comes from the adapter snapshot when
the GUI agent is running (`-e` silent bridge). OAuth rows with tokens in the OS
keyring show **Sign out** (clears the keyring entry). Hide Sign in while signed in. **Sign in** always uses a short
`pi --mode rpc --no-session -e` (not a second agent — no session file, ADR-0006)
running headless adapter `authenticate()` (callback only, no paste). Pi does not
open the browser (WSL would spawn a second tab). Status returns the authorize URL;
the GUI `window.open`s it once so the callback can `window.close()` like Claude/Codex.
If the authorize URL's `redirect_uri` is not localhost, Sign in fails immediately (Linear's hosted callback would never reach PiCode). Authenticate registers `http://127.0.0.1:<port>/callback`.
Success HTML is PiCode's (logo + return to `#/mcps`). Add or On on an OAuth server
starts Sign in immediately. Tokens live in the OS keyring, keyed by server name on
this machine — not per agent. No native MCP.


Sessions are **pi JSONL files** (`~/.pi/agent/sessions/`), bucketed by pi
itself per cwd by default — not per agent. An agent's **Search sessions**
picker (`GET /api/workspaces/{id}/sessions?agent=`) only lists sessions
PiCode has recorded as that agent's own (`agent_sessions`, ADR-0039):
every fresh spawn mints a `--session-id` up front so its session is
attributable from the moment it exists, and every resume/fork/clone/
adopt/import historizes the path it points at. A Terminal running bare
`pi` in the same folder — or another Agent sharing the cwd — never
appears in an unrelated agent's picker. Every agent spawn also carries a
private `--session-dir` (`~/.pi/agent/sessions/<agentID>/`, ADR-0040), so
pi's **own** native "Resume Session" picker inside its interactive TUI —
which reads sessions straight off disk and has no knowledge of PiCode's
API — is scoped the same way, not just PiCode's chat surface. The picker
lists the union of the cwd bucket and the agent's private dir, since
ADR-0040 is where fresh sessions actually land. Before any spawn mints a
fresh `--session-id`, it first **adopts**: pending ids from an earlier
run are matched against the files in the private dir and the newest
match becomes a plain `--session` resume with `agents.session_path`
backfilled (ADR-0053) — so switching between the chat and the agent's
own TUI continues one session instead of minting a competitor each hop.
The composer status bar is per agent too: the desktop app fetches
`/status?agent=<selected>`; without the parameter the endpoint answers
for the workspace's first agent (ADR-0053). The
machine-wide/workspace-wide housekeeping view (`GET
/api/clis/pi/sessions`, below) is unfiltered on purpose and unions in
every agent's private dir alongside the shared cwd bucket when scoped by
workspace: it exists to show and clean up everything, ownership tag or
not.
PiCode lists, switches (`--session`), and **replays** them into the chat
surface. History is not copied into SQLite (ADR-0005). The transcript endpoint serves a
window (`?tail=&skip=`) — the browser holds only the newest slice and
`Load earlier` pages older turns from the server. The window is cut at the
last compaction boundary (what pi itself replays): pre-compaction history
lives only inside the collapsible summary card, and the response reports
`compacted` so the UI can tell "needs /compact" from "already compacted,
file just stays large".

Entry: the last icon in the desktop sidebar header opens Agent CLIs (`#/clis`)
without changing the rail tab; user menu (Tools: Automations, llama.cpp,
Integrations; PiCode: Preferences, Devices, System) and `Ctrl+K`.
QR in the sidebar brand opens a phone-share drawer (`GET /api/share`):
HTTPS + bind + reachable IP + cert SAN + mkcert CA. Missing checks
list the action; a QR is only drawn when every check passes.
