# Dev servers: what is running, what it is, and what PiCode may do about it

> Part of PiCode's architecture (ADR-0105: one file per subsystem). Edit here; the index only links.

A dev server (`npm run dev`, `uvicorn`, `python3 dev.py`) is the one thing the
file preview cannot serve: it exists only while its process does, on a port,
and its own machinery (HMR, websockets, absolute URLs) breaks under any proxy.
So PiCode does not proxy it. **The panel finds it; PiCode's own browser
surface shows it, talking to the server directly — and, since the v2 rework of
2026-09-17 (ADR-0151), the panel names what each port is and can stop or hide
the listeners PiCode started.**

Pieces:

| Piece | Where |
|---|---|
| The read: what is listening, who owns it, what it is | `GET /api/devservers` (`internal/server/devservers.go`) |
| The writes: stop a listener, hide one, bring it back | `POST /api/devservers/{stop,hide,unhide}` (`devservers_actions.go`; ADR-0151) |
| The list, the row verbs and the states | Inspector rail → **Servers** (`web/browser/src/components/InspectorServers.jsx`) |
| The panel's decisions (pure, tested) | `web/browser/src/lib/devServers.js` |
| Hides | `dev_server_hides` (migration 052, `internal/store/dev_server_hides.go`) |
| The surface that shows the page | the work-browser tab (`openWebTab`): native WebView2 in the desktop app, a frame in a plain browser |
| The door from the terminal | a loopback `http(s)` link in a pane opens in PiCode instead of the system browser (`App.jsx`, `open-link`) |
| The design record | `docs/plans/servers-panel-v2.md` |

## The read

`GET /api/devservers` answers
`{servers: [{port, url, scheme, kind, state, title, tool, ownerKind, ownerId,
ownerName, workspace, pid, startKey, startedAt, hidden, hideId}], hidden, total,
truncated, readable}`. It is an ordinary guarded API — the browser that fills
the panel asked for it — and it costs:

- **What listens.** `/proc/net/tcp` + `/proc/net/tcp6`, LISTEN rows on the
  loopback addresses and the wildcard binds (Linux; elsewhere the list falls
  back to the candidate ports alone).
- **Who owns a port.** Each PiCode terminal's and agent's tmux pane, its
  process tree, and `/proc/<pid>/fd` — the socket inode a process holds is the
  join key between "which port" and "which pane". The same walk ADR-0062
  already does for CLI presence; the tool name comes from that process's own
  command line (`node …/node_modules/.bin/vite` → `vite`, `python3 dev.py` →
  `dev`). The walk also learns **which process** holds the socket (`pid`) and
  its start token (`startKey`, `/proc/<pid>/stat` field 22 — the PID-reuse guard
  the peer-stop path already uses), plus the wall clock of that start
  (`startedAt`: boot time + start ticks).
- **What a port is.** One HTTP `GET http://127.0.0.1:<port>/` for the content
  type and `<title>`, and — when the plain attempt got nothing, or Go's own
  `Client sent an HTTP request to an HTTPS server` — **one TLS attempt** on the
  same port with certificate verification off. This is the only reason the
  daemon ever speaks to the user's servers: once per port, cached for two
  minutes (a port that answered nothing is retried after ten seconds, so a
  server still starting gets its answer soon), and `?refresh=1` — the panel's
  own Refresh — asks again. Polls inside the TTL cost one `/proc` read and
  nothing else, so a panel left open does not fill a dev server's log. The
  probes of one read run in parallel (16 at a time): a loopback port that
  silently drops a SYN costs its whole 900 ms timeout, and twenty of those in
  series measured 17 s before this was parallel.

**`kind` is the difference v2 added.** `page` is HTML — what the browser tab can
show, whatever the status code. `api` is any other HTTP answer: JSON, plain
text, a 404, a 5xx. `opaque` answered nothing at all. The first version treated
"anything below 500" as a page, which put an agent CLI's internal control
channel — an HTTPS API with a bundled certificate — in the list as a row with an
**Open** that could never work (measured on the owner's machine: plain HTTP to
the TLS port answers `400 Client sent an HTTP request to an HTTPS server`, and
the plain API port answers `404 page not found`). `scheme` says which attempt
answered, so the URL the row offers is the one that works.

**What is reported stays narrow.** A port is a row when a PiCode pane owns its
socket, or when it is one of the usual dev ports (`devServerPorts`) *and* it is
listening right now — a cached probe answer is not evidence, so a server that
stopped cannot linger for the rest of the TTL (the defect the first QA pass of
v2 found minutes after a Stop). A pane-owned port that answers nothing is still
a row: it is the thing the user just launched, and it says *starting* (the
process is younger than 60 s) or *not answering*. Everything else listening on
the machine — a database, a private API, PiCode's own daemon — is none of this
panel's business; the first version reported every listener and listed Postgres
on 5432 and PiCode on 8445.

**The read reports more than the panel shows first.** A page, and any listener
PiCode can attribute, is visible. A port that only answers an API, outside
PiCode, on a port the candidate list merely guesses at, waits behind "Show N
more" — the row is in the payload, so revealing it costs no round trip. `hidden`
counts the saved hides, `total`/`truncated` report the 40-row cap, and
`readable` says whether the owner walk could run at all (the blocked line on a
machine without `/proc`).

## The writes (ADR-0151)

`POST /api/devservers/stop {port, pid, startKey, force}` re-runs the owner walk
at action time and stops only a process it can still re-derive as the holder of
that loopback socket inside a PiCode pane: the caller's `pid` and the process's
current start token must match what the route just read, so a stale row can
never signal a process that reused the id. SIGTERM first; SIGKILL only on a
second, forced request. The route waits briefly (1.5 s) for the signal to take
effect and answers `{stopped, pid}`, so the panel's *Stopping…* ends with the
truth; a process that survived answers `stopped: false` and the row offers
**Force stop**. Every stop appends a `devserver.stopped` event (port, pid, tool,
owner, forced, stopped) — the events table is the audit log for a write that
reaches outside PiCode's own records.

`POST /api/devservers/hide {port, pid, startKey}` and `/unhide {hideId}` persist
a hide in `dev_server_hides`, keyed by **port + pid + start token**, and announce
`devserver.hidden` (ADR-0048). The key is the whole point: a hide means *this
listener*, so a new process on the same port is a new row, never a silent hole.
A hide whose process is gone can never match a listener again, so the next hide
prunes it in one transaction — the panel's "N hidden" line cannot lie. Hide needs
the same identity Stop needs, so a port PiCode cannot attribute has no Hide
either: it has Copy address, and the disclosure.

## The surface

Each row is: the port and what the page calls itself (or nothing, when the port
is not a page and has no name), the note pill (*HTTPS*, *not a page*,
*starting*, *not answering*), the address, and the owner line — `terminal
"sidebar" · PiCode · 2 h`, or `started outside PiCode`. The name + address are a
button when the URL is a page (that is the **Open**), and a plain block when it
is not: a control channel that answers an API gets no button that leads nowhere.
The row's menu is the verb set: Open in PiCode, Open in browser, Copy address,
Show terminal (the agent's tab, or the terminal's), Stop server…, Hide from the
list, and Show again on a hidden row. Stop confirms with the process and its
terminal in the sentence, and the row shows *Stopping…* immediately — the state
is keyed by the process identity, so a new server on the same port never
inherits a stale "Stopping…".

**Open** hands the URL to `openWebTab`: the work-browser tab the desktop app
already had (a native WebView2 child). In a **plain browser** there is no
webview to host a page, so that same tab renders a loopback URL in a frame —
that is what makes the panel usable outside the desktop app, and it is the
only reason the app shell's CSP grew
`frame-src http://localhost:* http://127.0.0.1:*` (plus the https forms): the
app shell may frame this machine, nothing else. The frame used to carry a line
saying so ("The desktop app shows this page in its own window…"); the owner
had it removed — it spent a row of a 1080p window on every local page, and the
frame is self-evident. The non-local case still explains itself when a URL is
typed, and the empty state still says what the surface is for. The policy now reaches the URLs the shells are served at — `/browser/`,
`/desktop/`, `/mobile/` — not only `/`, `/index.html` and `*.html`: the
coverage gap this feature first ran into was its own doing in the sense that
nothing restricted the frame, and it hid the fact that the app ran with no
policy at all. The hash is computed from the file each path serves (the
launcher has no inline script; every shell carries the theme bootstrap), and
`frame-src` is why the frame still opens under it. Amendment of 2026-09-15 in
`docs/decisions/0052-public-access.md`. The URL lives in the tab's address bar (editable, Enter
reloads), and each web tab's address is kept in `picode-webtab-urls` so a
reload restores it — the desktop shell reads the address back from its own
webview instead.

Two smaller things the same door needed: the tab strip now receives `webTabs`
(it never had, so every web tab read "New tab"), and opening a row carries the
page's title so the tab is named before anything reports back. The tab itself is reachable
with **no anchor at all**: the rail's row used to render only with an owner,
so the empty state ("Open an agent or terminal to inspect its files.") had no
way to Servers. Now the row always renders — the no-owner rail is Files +
Servers, and Files keeps that message — because what listens on this machine
does not depend on what is selected.

**Both shells carry it, because they are one app.** `/browser/` (the desktop
layout a browser opens) and `/desktop/` (the composition the Windows shell
loads) are the same components — `web/desktop/src/main.jsx` is
`boot(root, { shellChrome: true })` over `web/browser`, and `make web` builds
both bundles; the panel ships in both (verified live on both, in a plain
browser). Mobile is the separate bundle: no rail, no web tab, nothing here.
Only the **host** differs, at one line: with `window.__TAURI__` present
(inside the desktop shell) `openWebTab` calls `btab_navigate` and the page is
a native WebView2 child — the path that existed before this feature, reused;
without it, the frame above.

## The terminal door

Ctrl+click (or the pane menu's **Open <host>**) on an `http(s)` link in a
terminal opens it where the human said web links belong. A URL printed in a
terminal is a browsing action inside PiCode, so the two rows Browser settings
already offers decide it — read fresh per link, like the shell's popup door:

| the link | the preference | the action |
| --- | --- | --- |
| loopback (`isLoopbackUrl`) | `localOpenDest` | PiCode's surface, unless "external" |
| any other host | `webOpenDest` | PiCode's surface, unless "external" |
| a path under the terminal's cwd | — | the file pane, unchanged |

Both preferences default to PiCode, so Ctrl+click on a printed link is
normally a **work-browser tab**. Until 2026-09-17 every non-loopback link left
for the system browser from the terminal, which made this the one door that
ignored where the human had said web URLs open (owner's call, 2026-09-17). The
three outcomes are one pure function with its own test —
`web/browser/src/lib/openLink.js` / `openLink.test.js` — and the terminal
wiring hands the URL up (`web/shared/domain/termLinks.js` takes the app's
opener) instead of calling `window.open` itself. The Servers panel's own
**Open** button does not consult it: an explicit control inside PiCode asks for
PiCode, and the preference governs the default, not a click.
The terminal's plain-text link scanner excludes trailing prose punctuation and
unmatched closing brackets, while retaining balanced brackets inside a URL.
The same boundary supplies the Ctrl+click target, underline and pane-menu link.

## What this does not do

- **No proxy, no rewrite.** The daemon never serves another process's pages;
  HMR, devtools and service workers are the browser's business, unchanged.
- **No restart.** PiCode does not know the command that started a server, and a
  process restarted outside its pane would be a lie about ownership. The row
  offers Stop; starting again is the human's shell.
- **No kill without an owner.** A port PiCode cannot attribute is readable,
  openable and hideable, never stoppable (ADR-0151's decision table).
- **No `/proc`, no attribution.** On macOS and Windows-native daemons the pane
  walk finds nothing, the read answers `readable: false`, the panel says so in
  one line, and only ports from the candidate list appear — and only if they
  answer. A dev server on an unusual port is then invisible until it is typed
  into the browser tab's address bar. Stop is refused with a sentence on those
  platforms.
- **Mobile has no web tab**, so the panel is a desktop/browser-shell surface
  (the rail exists only in the desktop shell).
- **HTTPS with a self-signed certificate** shows the browser's warning; the
  probe does not verify the certificate either (it only asks what the port is),
  so the row may show the port and owner with a title taken over TLS.
- The daemon now makes HTTP requests to the user's own servers (see the TTL
  above). That is a real, small behaviour change, and it lives only here.

The panel's poll is 5 s (`InspectorServers.jsx`) — a `setInterval` against
`/api/*`, which ADR-0048 asks to justify: what is listening is machine state,
not a store mutation, so the feed cannot cover it. The route is a read and
answers from cache; the refresh control is the escape hatch, and a Stop
refetches at once rather than waiting for the tick.
