# Security model (ADR-0007)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

- **HTTPS always** (bind 0.0.0.0): mkcert-issued cert via
  `scripts/setup-cert.sh` (SANs: localhost, picode.local, 127.0.0.1, ::1 +
  LAN + tailscale; CA exported to
  the Windows trust store on WSL) or a generated self-signed cert as the
  zero-config bootstrap. `PICODE_INSECURE=1` disables TLS (dev only).
- **Port and bind**: default range `8445-8455` on `0.0.0.0`, first free
  port wins; both **editable in the Settings UI at runtime** (graceful
  rebind: bind-new-first, revert on failure — ADR-0007, ADR-0050).
  Precedence: UI/DB > `PICODE_PORT`/`PICODE_HOST` env > default. A
  **public URL** setting (advisory, no env) is the origin other machines
  use: pairing links, the phone drawer and `server.json` carry it.
  Discovery: `~/.picode/server.json` (`url` for clients on this machine,
  `bind`, `publicUrl`); a client on another machine reads
  `~/.picode/remote.json` (`url`, `token`, `caFile`) instead.
- **Trust boundary**: a paired device (ADR-0049), no longer a network.
  What `Wrap` inspects at all is `guarded()`: `/api/`, `/ws/` and
  `/mcp/communication`. Everything else — the UI and its assets, `/pair`,
  and the ticketed `/preview/**` routes (ADR-0136/0137, where the capability
  token is the gate because the sandbox sends no cookie) — passes straight
  through, and so never reaches the `Host` and `Origin` checks either. Whether
  `/pair` should instead be guarded-and-exempt is an open question in
  `docs/handoff/open/process.md`; it changes who can reach pairing.
  `internal/auth` gates every `/api` and `/ws` request: principal from
  the `picode_session` cookie or `Authorization: Bearer` (install token
  at `<data>/token`, or a token session); `Host` and `Origin` checked in
  every mode; modes `off | remote (default: loopback auto-pairs) | all`.
  **Host allowlist (ADR-0215)**: IP literals, `localhost` / `*.localhost`,
  `picode.local`, the hostname bare or as the first label (Tailscale's
  `name-N` twin included) under `local`, `lan`, `home`, `home.arpa`,
  `localdomain`, `internal` or `<tailnet>.ts.net`, every DNS name the
  served certificates cover (`tlsutil.CertNames`, re-read on change), and
  the public URL — nothing else. **Fetch metadata (ADR-0215)**: a
  `Sec-Fetch-Site: cross-site` request is refused on every guarded route
  except the exempt ones (`/api/health` carries CORS for the `:8470` trust
  page), and only a first-party request (no fetch metadata, `same-origin`
  or `none`) is handed a loopback session — a `same-site` page on another
  localhost port gets 401 — and over plain HTTP only on a loopback Host
  name (`localhost`, `*.localhost`, a loopback IP), since an untrustworthy
  `http://picode.local` carries no fetch metadata and mDNS can be spoofed.
  The public URL the gate admits is the Settings value, else
  `PICODE_PUBLIC_URL` (how a gateway member gets it).
  A browser-like first-party loopback visit with no cookie reuses the newest live
  session with its user-agent label (secret rotated in place, presence
  asked first so an active browser keeps its cookie — ADR-0049
  amendment 2026-09-03) instead of minting a duplicate row per launch.
  Route registration is unconditional, including the auth surface itself:
  `Routes()` records `registerAll` against a zero `Deps`, so a helper that
  registered only when its dependency was present wrote its routes out of
  the published OpenAPI document. `registerAuthRoutes` did exactly that and
  the nine `/api/auth` and `/pair` patterns were missing from the public API
  reference while the daemon served them all; `docs-check` compares the
  committed JSON against the same generator, so nothing failed.
  `TestRoutesCoverEveryRegisteredPattern` now holds the line — a nil
  dependency answers at request time, the way `Deps` already documents
  ("nil-safe = 503 on the routes").

  Pairing codes (`/pair?code=`, ten minutes, one use, lockout after five
  failures) mint browser sessions; Preferences → Server lists and revokes
  them. Expired sessions stop listing; a daily sweep prunes
  revoked/expired rows after 7 days. PiCode executes with the user's
  permissions, like Pi itself.
  Roadmap for tailnet, shared and public servers:
  `docs/design/remote-modes-roadmap.md`.
- **HTML previews are a capability route, not the session** (ADR-0136,
  ADR-0137): `POST /api/previews` mints an hour-long, session-bound ticket for
  one `.html`/`.htm`; the path form `GET /preview/<ticket>/<path>` serves only
  allowlisted web assets under the owner's folder (dotfiles and symlinks out,
  `no-referrer`, `no-store`) to a frame sandboxed by response header
  (`sandbox`, never `allow-same-origin`), and the host form serves the same
  ticket from `<label>.localhost` — a real origin, reached before the auth
  gate, that serves the preview namespace and answers 404 for `/api` and the
  app shell. A sandboxed page sends no cookie and reads no `/api` answer
  (CORS); the host form cannot either: `Origin == r.Host` fails for it, its
  `SameSite=Strict` host-only cookie is not sent (measured), and no route
  answers CORS except the deliberate `/api/health` probe. Revoking the session
  kills both forms; the daemon restart drops them all.
- **Installed webapps fetch is a bounded, user-initiated retrieval, not a
  proxy** (ADR-0147): `/api/webapps/resolve` and install fetch the page the
  user named — 4 s timeout, 256 KB caps (64 KB manifest), four redirects, no
  proxy, no credentials forwarded — with a DNS-rebind guard: every resolved
  address of a non-local hostname must be public (checked at dial, the
  validated IP dialed directly), redirects may not cross from a public origin
  onto a local name or private address, and explicit loopback URLs stay
  first-class. Metadata (PWA manifest, then same-origin `<link rel=icon>`,
  then `/favicon.ico`) is same-origin only. `GET /api/webapps/{id}/icon`
  serves stored bytes behind `Content-Security-Policy: default-src 'none'`
  and `nosniff` (a hostile SVG stays inert). The webapp itself renders in
  the shell's WebView2 child: no daemon token, no `host` object, no approval
  UI; cookies are engine-isolated per origin. Since ADR-0153 an installed
  app's webview uses **its own user-data folder**
  (`WebView2\webapps\<webappId>`, derived by the shell from the tab id):
  its logins and its "clear data" are contained — one account per service
  becomes possible, and legacy installs (column `partitioned = 0`) keep
  the shared work profile, whose "Clear browsing data" wipes them too.
- **App-shell CSP** (ADR-0052): HTML responses for `/`, `/browser/`, `/desktop/`
  and `/mobile/` carry `appCSP`. `connect-src` is `'self'` plus the request
  host's `ws://`/`wss://`. The Windows shell's pages (`/desktop/`,
  `/desktop/management.html`) also name Tauri 2's IPC
  (`ipc: http://ipc.localhost https://ipc.localhost`) so WebView2 can fetch
  plugin commands; the browser and mobile shells do not, because they never
  talk to that host. Assets and API answers carry no policy.
- **Which origin reaches the shell's IPC**: the capability files
  (`desktop-shell/capabilities/default.json`, `management.json`) list the
  daemon's default origin, `https://localhost:8445` and `127.0.0.1:8445`,
  under `remote`. When the shell finds the daemon on another port (its
  8445–8455 range, or a port set in Settings / `PICODE_PORT`), it adds the
  same two files again at runtime with exactly that origin
  (`desktop-shell/src/daemon_acl.rs`): same permissions, same windows, one
  more origin. The files are not widened to `localhost:*`, which would hand
  IPC to any loopback server a webview ever showed. Two costs, accepted:
  Tauri has no revoke, so an origin once granted stays granted until the
  shell restarts (a daemon that moved back to 8445 leaves its old port
  trusted, and a page later served there — a dev server in a work tab —
  would get the main window's commands); and the origin comes from
  `server.json`, which any process running as the same WSL user can
  rewrite. That user can already run Windows programs through interop,
  so it is not an escalation.
- **The waiting page's commands are local-only**: the main window opens on
  the bundled `ui/waiting.html` until the daemon answers
  (`desktop-shell/src/waiting.rs`). Its buttons — Start PiCode, View logs,
  Trust certificate — run through `waiting_state` / `waiting_action`,
  granted by `capabilities/waiting.json`, which has no `remote` list: the
  daemon's `/desktop/`, loaded later in the same webview, cannot call them.
  Trust certificate imports the distro's mkcert root into the Windows
  user's `CurrentUser\Root` with `certutil -user` (no administrator rights;
  Windows asks the human to confirm the root), the same file the installer
  puts in `LocalMachine\Root`. Whether Windows trusts the daemon is probed
  with `curl.exe` *without* `-k` (schannel reads the store WebView2 uses),
  so the window never navigates onto a certificate error page.

## Handing a target to the operating system (the desktop shell)

One command hands a target to Windows: `btab_open_external`, which runs
`cmd /C start "" <target>`. Whatever reaches it is executed by the OS, so the
guard is an allowlist of target *shapes* and refuses every character that
could turn a target into a second command (quotes, spaces, `&`, `|`, `<`,
`>`, `^`, backtick). Two classes are allowed: `http(s)://` — links that leave
the app (ADR-0122) — and `ms-settings:` screens, which the v2d row uses for
what PiCode deliberately does not reimplement (Windows Hello, passkeys,
saved passwords). The allowlist and its decision table live in
`desktop-shell/src/external.rs`, where the rows run as tests without cargo.
Everything else — `file:`, `javascript:`, `data:`, bare `cmd:` — is refused
with a message, never silently ignored.
