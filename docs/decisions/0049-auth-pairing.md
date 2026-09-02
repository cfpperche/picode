# ADR-0049: Authentication — paired devices, an install token, and a Host/Origin gate

- **Status**: accepted (supersedes ADR-0007's "no app-level auth"; first step of the remote-modes roadmap, `docs/design/remote-modes-roadmap.md`)
- **Date**: 2026-09-01

## Context

ADR-0007 bound PiCode to `0.0.0.0` with HTTPS and no application-level
authentication, on the argument that the trust boundary is the personal
machine and the tailnet, and recorded "token auth if ever exposed beyond
the tailnet" as a debt. The inventory behind this ADR found ~190 routes
reachable by anyone who can open the port, among them `POST
/api/agents/{id}/bash`, `/ws/term` (a shell), file writes, backup
restore, and the provider credential routes; WebSocket upgraders that
accept any Origin; no cookies; and a plaintext listener on `:8470`
serving the mkcert root CA. A page in any browser on the machine could
also drive `localhost:8445` (DNS rebinding, simple POSTs).

The owner's direction: the product must run as a server on a tailnet
(own or shared) and, later, on the public internet. That needs identity
per client, not per network — and because an authenticated caller gets a
shell, the mechanism has to be as strong as the ones that guard SSH.

## Decision

1. **Every request under `/api/` and `/ws/` passes one gate**
   (`internal/auth`, wrapping the mux in `server.New`). It resolves a
   principal from the `picode_session` cookie (HttpOnly, Secure under TLS,
   `SameSite=Strict`) or an `Authorization: Bearer` header (the install
   token, or a token session), and enforces `Host` and `Origin` in every
   mode: a `Host` that is not loopback, an IP literal, `localhost`,
   `picode.local`, `*.local`, `*.ts.net`, this machine's hostname or the
   configured public URL is refused (403); a state-changing request, a
   WebSocket upgrade or the event stream with a foreign `Origin` (or
   `Sec-Fetch-Site: cross-site`) is refused (403).
2. **Modes** (`auth.mode` setting, Preferences → Server): `remote`
   (default) — a browser on loopback with no session is minted one
   silently, so the local experience does not change; every other client
   must present a session or token, or gets 401 `pairing required`.
   `all` — loopback pairs too (shared / public servers). `off` — no
   principal required (dev, or behind a proxy the operator trusts); the
   Host/Origin gate still applies.
3. **Pairing** turns a visit into a browser session: a paired device (or
   `picode pair` on the daemon's machine, with the install token) mints
   a one-time code valid ten minutes; `GET /pair?code=` spends it, records
   the device (user-agent label, IP, device id) and sets the cookie.
   Five failures per minute per IP lock that IP out for ten minutes. The
   "Open on phone" QR carries a pairing link, so the phone flow is scan →
   trust page (mkcert setups) → paired.
4. **Principals are rows** (`auth_sessions`, `auth_pairings`, migration
   019) with secrets stored as sha256; browser sessions expire after 90
   days unseen, token sessions never; every change is an event
   (`session.created / session.revoked / pairing.created / pairing.used`)
   so the Devices list updates live and the log is the audit.
5. **The install token** (`<data>/token`, 0600) is the bearer for
   scripts and the daemon's own non-browser clients: the Chrome native
   host and `pi-inbox` read it beside `server.json`; `PICODE_DATA` now
   reaches every spawned pi so they find both. `picode token rotate`
   replaces it. Those clients verify TLS unless the host is loopback.
6. **Devices, in Preferences → Server**: paired sessions with label, IP,
   last seen, Forget; Pair a device (QR + link); the mode; the token
   path with Rotate. A 401 in the SPA shows the pairing screen instead
   of a broken page. The presence "host" flag now means "a desk browser"
   (desktop shell), not "loopback", since the daemon may be elsewhere.

Decision table (each row is a test in `internal/auth`):

| condition | outcome |
|---|---|
| `GET /api/health`, `/pair`, `POST …/fire`, the UI's own files | pass, no principal |
| Host not allowed | 403 |
| mutating / upgrade / `/api/events` with foreign Origin or cross-site | 403 |
| valid cookie or bearer | pass with principal |
| mode off, no principal | pass, anonymous |
| mode remote, loopback (no proxy header), no principal | session minted, cookie set |
| mode remote, non-loopback, no principal | 401 (stale cookie cleared) |
| mode all, loopback, no principal | 401 |
| revoked or expired session | 401 |
| pairing code reused / expired / unknown | 410 / 410 / 400 page |
| 6th failure in a minute | 429 for ten minutes, even for a valid code |
| token rotated | old bearer 401, new 200 |

## Consequences

- **Other devices on the LAN or tailnet stop working until paired.** The
  changelog says so; the QR and `picode pair` are the two ways in.
- **Loopback stays frictionless** in the default mode; `all` is a
  deliberate switch for servers.
- **The Chrome extension and `pi-inbox` keep working unchanged** on the
  same machine (they read the token). Off-box clients arrive with Track B.
- **Cookies need HTTPS** (`Secure`) except in `PICODE_INSECURE` mode.
- **The `:8470` trust listener is unchanged in this step** — it hands out
  a CA, not access; Track B binds it to the LAN and Track D disables it.
- **A proxy hop hides loopback** (`X-Forwarded-For` present): behind a
  proxy on the same machine, use `off` or pair — never trust a forwarded
  header for the loopback shortcut.
- **Session listing shows IPs and user-agent labels**, nothing else; the
  secret exists only in the cookie or the file.

## Alternatives considered

- **A single bearer token for everything** — no per-device revocation,
  no expiry, and a copied string is a shell; kept only as the install
  token for local clients, rotatable.
- **Passwords / passkeys** — a second factor worth adding for public
  access (Track D); pairing is the bootstrap that makes a device known.
- **mTLS** — correct, but issuing client certificates to phones is the
  friction this product exists to avoid.
- **Keep ADR-0007 and only publish the webhook through a tunnel** — the
  cheapest way to receive webhooks, and still recommended for that; it
  does nothing for using the app from another machine.
