# Servers panel v2 — what a listener is, and what you can do with it

- **Status**: landed 2026-09-17 (`feat/servers-v2`)
- **Benchmark**: [`docs/benchmarks/2026-09-05-inspector-rail.md`](../benchmarks/2026-09-05-inspector-rail.md)
  (the rail's panel bar: name the owner, one verb per row, no invented facts)
- **ADR**: [ADR-0151](../decisions/0151-devservers-stop.md) (the Stop capability)
- **Docs**: [`docs/architecture/devservers.md`](../architecture/devservers.md)

## Why now — the two rows that were wrong

The owner's machine, 2026-09-17. The panel showed:

```
45683 agy    http://localhost:45683/    Open
             terminal "sidebar" · PiCode
46579 agy    http://localhost:46579/    Open
             terminal "sidebar" · PiCode
```

Measured, not guessed:

| Claim | Evidence |
|---|---|
| both rows are one process | `ss -ltnp`: pid 3257366 (`agy --dangerously-skip-permissions`) holds 45683 and 46579 |
| there is no terminal named "agy" | the store has `sidebar-6fabbd` (name `sidebar`) in workspace PiCode; `agy` is the binary's name, read from its argv |
| "Open" could not work | `curl http://127.0.0.1:45683/` → `400 Client sent an HTTP request to an HTTPS server`; `curl -k https://…` → `HTTP/2 404`, cert `CN=localhost · O=ENABLES HTTP2 · OU=bundled on purpose`; `curl http://127.0.0.1:46579/` → `404 page not found`, `Vary: Origin` |
| the port is legitimately the panel's business | the process lives in `picode-sh-sidebar-6fabbd`, a PiCode pane |
| there is nothing to do with a row | the row was one `<button onClick={open}>` |

Three design failures, not one: the row was **named by a binary** instead of the
thing the human knows; **"a server I can open" was decided by "answered
something below 500"**; and **the only verb was Open**.

## The shape

A row is a listener PiCode can name, with the truth about what it is and the
verbs to act on it.

```
[globe] 4321  [not a page]                        [⋯]
        http://localhost:4321/
        terminal "api" · QA · 2 min
```

- **Name**: the page's `<title>`, else the tool, else nothing (a port that is
  not a page has no name to give and must not borrow "unnamed page" while a
  *not a page* badge sits beside it).
- **Note** (pill): `HTTPS` when the answer came over TLS, `not a page` for an
  API, `starting` / `not answering` for a port that has not spoken yet.
- **Meta**: the owner in the panel's own words (`terminal "sidebar" · PiCode`),
  plus how long the process has been up. `started outside PiCode` when PiCode
  cannot see the process.
- **Open** only exists where the URL leads somewhere a browser can show; the
  name + URL are a button when it does, a plain block when it does not.
- **Verbs** live in the row's menu: Open in PiCode · Open in browser · Copy
  address · Show terminal · Stop server… · Hide from the list. A hidden row
  offers *Show again*.

### What a port is (`kind`) and what answered (`scheme`)

| the probe found | kind | scheme | row |
|---|---|---|---|
| HTML (a declared `text/html`, or a title in an undeclared document) | `page` | http/https | title + **Open** |
| any other HTTP answer — JSON, plain text, 404, 5xx | `api` | http/https | pill *not a page*, no Open |
| plain HTTP answered Go's `Client sent an HTTP request to an HTTPS server` | re-probe over TLS | https | judge the TLS answer |
| nothing on http, nothing on a TLS attempt | `opaque` | — | pill *starting* / *not answering* |

The TLS attempt is bounded and never trusts what it reads: a dev server's
certificate is self-signed by definition (Vite `--https`, mkcert, a CLI that
bundles one), and the client reads a content type and a title and closes. The
work browser still shows the browser's own warning.

### What the panel shows first

| answers | is a page | PiCode sees the process | default view |
|---|---|---|---|
| yes | yes | either | visible |
| yes | no | yes | visible (it is the human's: Stop/Hide apply) |
| yes | no | no | **behind "Show N more"** |
| nothing | — | yes | visible as *starting* (**< 60 s since the process began**) or *not answering* |
| nothing | — | no | not a row at all |

The server reports every row it can see — including the ones behind the
disclosure — so revealing the rest costs no round trip, and the "N" the button
promises is the number of rows it actually adds.

### States

| state | when | one line + one action |
|---|---|---|
| first paint | `loaded === false` | three skeleton lines (never "Looking…") |
| empty | nothing at all | *Nothing is listening. Start a dev server in a terminal — it shows up here.* + **Open a URL…** |
| empty because it is all hidden/disclosed | rows exist, none shown | *Nothing you started is listening.* + the disclosure buttons |
| blocked | `/proc` is unreadable here (`readable: false`) | *Port owners are not readable on this machine, so only the usual dev ports that answer are listed.* |
| error | the read failed | the message + **Try again**; the last good list stays on screen |
| stale from the read | nothing listens any more | the row is gone on the next poll (a cached title is not evidence) |
| mid-stop | the signal is out | *Stopping…* on the row, then the row leaves; *Still running.* + **Force stop** when it did not land |
| truncated | more listeners than the 40-row cap | *Showing the first 40 of N listening ports.* |

## Decisions this design had to make

| question | answer | why |
|---|---|---|
| Who may be stopped? | a process PiCode re-derives inside one of its own panes, with the pid **and** start token matched at action time | a kill without the join is a guess with a destructive outcome; the token makes PID reuse a refusal, not an accident |
| Where does Stop's confirmation say what? | the process, its terminal, and the cost (*Anything it was running stops too.*) | the process is a shell the human may be working in |
| Force? | only after a SIGTERM that did not land | teaches the escalation instead of offering a kill up front |
| Hide, keyed by what? | port + pid + start token | the port is not the listener; a new server on 5173 must be born visible |
| Does a hidden row survive a reload? | yes (SQLite, `dev_server_hides`), and it is pruned when its process is gone | the "N hidden" count must not lie |
| Who decides the disclosure? | the client (the payload carries `kind` + owner) | one read, instant disclosure |
| Where is the audit? | `devserver.stopped` and `devserver.hidden` events | ADR-0048: a destructive write leaves a row |

## Refused

- A **proxy** or URL rewriting: HMR, devtools and service workers stay the
  browser's business (unchanged from v1).
- **Restart**: PiCode does not know the command that started a server, and a
  process restarted outside its pane is a lie about ownership.
- **Killing a port nobody owns**, even with a confirmation (ADR-0151).
- **Remembering the Servers tab**: unchanged — the remembered rail tabs stay
  `files | changes | pr`, and Servers is a look, not a preference.

## Where it lives

| Piece | File |
|---|---|
| the read, the classification, the row decision | `internal/server/devservers.go` |
| the two writes and their guards | `internal/server/devservers_actions.go` |
| the signal (Linux) and the refusal (elsewhere) | `devservers_signal_linux.go` / `_other.go` |
| hides | `internal/store/migrations/052_dev_server_hides.sql`, `dev_server_hides.go` |
| panel decisions (pure, tested) | `web/browser/src/lib/devServers.js` |
| rows, menu, states | `web/browser/src/components/InspectorServers.jsx` |
| the door to the owning terminal | `App.jsx` → `onOpenOwner` (agent tab, or `openTermTab`) |

## Evidence (scratch instance, 2026-09-17)

Fixtures: a page in a PiCode terminal (5173), an API in one (4321), a silent
listener in one (9000), a page started outside PiCode (3000) and an API outside
it (8080). Screenshots read: default panel, row menu, stop confirmation, the
terminal printing `Terminated`, the hidden section, "Show N more", and the empty
state; `window.__picodeOverlayAudit()` `ok: true` on every captured frame, with
no console errors. The stop was verified at the process level (`ss` shows the
port gone; the owning pane printed `Terminated python3 -m http.server 5173`).

Not reproduced visually: the blocked line for a daemon without `/proc` (this
machine is Linux) — covered by `TestDevServerUnreadableOwners`.
