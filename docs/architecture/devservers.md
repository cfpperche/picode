# Dev servers: what is running, and one click to see it

> Part of PiCode's architecture (ADR-0105: one file per subsystem). Edit here; the index only links.

A dev server (`npm run dev`, `uvicorn`, `python3 dev.py`) is the one thing the
file preview cannot serve: it exists only while its process does, on a port,
and its own machinery (HMR, websockets, absolute URLs) breaks under any proxy.
So PiCode does not proxy it. **The panel finds it; PiCode's own browser
surface shows it, talking to the server directly.**

Pieces:

| Piece | Where |
|---|---|
| The read: what is listening, who owns it, what it calls itself | `GET /api/devservers` (`internal/server/devservers.go`) |
| The list with one **Open** per row | Inspector rail → **Servers** (`web/browser/src/components/InspectorServers.jsx`) |
| The surface that shows the page | the work-browser tab (`openWebTab`): native WebView2 in the desktop app, a frame in a plain browser |
| The door from the terminal | a loopback `http(s)` link in a pane opens in PiCode instead of the system browser (`App.jsx`, `open-link`) |

## The read

`GET /api/devservers` answers `{servers: [{port, url, title, tool, ownerKind,
ownerId, ownerName, workspace}]}`. It is an ordinary guarded API — the browser
that fills the panel asked for it — and it costs:

- **What listens.** `/proc/net/tcp` + `/proc/net/tcp6`, LISTEN rows on the
  loopback addresses and the wildcard binds (Linux; elsewhere the list falls
  back to the candidate ports alone).
- **Who owns a port.** Each PiCode terminal's and agent's tmux pane, its
  process tree, and `/proc/<pid>/fd` — the socket inode a process holds is the
  join key between "which port" and "which pane". The same walk ADR-0062
  already does for CLI presence; the tool name comes from that process's own
  command line (`node …/node_modules/.bin/vite` → `vite`, `python3 dev.py` →
  `dev`).
- **What the page is.** One HTTP `GET http://127.0.0.1:<port>/` for a `<title>`
  and a `<meta name="generator">`. This is the only reason the daemon ever
  speaks to the user's servers: once per port, cached for two minutes (a port
  that answered nothing is retried after ten seconds, so a server still
  starting gets its title soon), and `?refresh=1` — the panel's own Refresh —
  asks again. Polls inside the TTL cost one `/proc` read and nothing else, so
  a panel left open does not fill a dev server's log. The probes of one read
  run in parallel (16 at a time): a loopback port that silently drops a SYN
  costs its whole 900 ms timeout, and twenty of those in series measured 17 s
  before this was parallel.

**What is reported stays narrow.** A port is a row when a PiCode pane owns its
socket, or when it is one of the usual dev ports (`devServerPorts`) *and*
answered the probe. Everything else listening on the machine — a database, a
private API, PiCode's own daemon — is none of this panel's business; the first
version reported every listener and listed Postgres on 5432 and PiCode on 8445.

## The surface

**Open** hands the URL to `openWebTab`: the work-browser tab the desktop app
already had (a native WebView2 child). In a **plain browser** there is no
webview to host a page, so that same tab renders a loopback URL in a frame
with one line saying so — that is what makes the panel usable outside the
desktop app, and it is the only reason the app shell's CSP grew
`frame-src http://localhost:* http://127.0.0.1:*` (plus the https forms): the
app shell may frame this machine, nothing else. The URL lives in the tab's
address bar (editable, Enter reloads), and each web tab's address is kept in
`picode-webtab-urls` so a reload restores it — the desktop shell reads the
address back from its own webview instead.

Two smaller things the same door needed: the tab strip now receives `webTabs`
(it never had, so every web tab read "New tab"), and opening a row carries the
page's title so the tab is named before anything reports back.

## The terminal door

Ctrl+click (or the pane menu's **Open <host>**) on an `http(s)` link in a
terminal whose host is this machine — `localhost`, `127.0.0.0/8`, `::1`,
`*.localhost` (`isLoopbackUrl`, `web/shared/client/devservers.js`) — opens the
page in PiCode's own browser surface. Anything else keeps going to the browser
the system would pick, exactly as before.

## What this does not do

- **No proxy, no rewrite.** The daemon never serves another process's pages;
  HMR, devtools and service workers are the browser's business, unchanged.
- **No `/proc`, no attribution.** On macOS and Windows-native daemons the pane
  walk finds nothing, so only ports from the candidate list appear, and only if
  they answer. A dev server on an unusual port is then invisible until it is
  typed into the browser tab's address bar.
- **Mobile has no web tab**, so the panel is a desktop/browser-shell surface
  (the rail exists only in the desktop shell).
- **HTTPS with a self-signed certificate** shows the browser's warning; the
  title probe does not follow it, so the row may show the port and owner
  without a title.
- The daemon now makes HTTP requests to the user's own servers (see the TTL
  above). That is a real, small behaviour change, and it lives only here.

The panel's poll is 5 s (`InspectorServers.jsx`) — a `setInterval` against
`/api/*`, which ADR-0048 asks to justify: what is listening is machine state,
not a store mutation, so the feed cannot cover it. The route is a read and
answers from cache; the refresh control is the escape hatch.
