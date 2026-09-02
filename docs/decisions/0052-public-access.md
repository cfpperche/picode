# ADR-0052: PiCode from any network — a login at the gateway, and a container per person

- **Status:** accepted (2026-09-02)
- **Amends:** ADR-0051 (the gateway gains a second front door and a second identity source; members may run in containers), ADR-0007 (public TLS is a proxy's, never PiCode's)
- **Roadmap:** `docs/design/remote-modes-roadmap.md`, Track D

## Context

After Track C, a shared box admits anyone the tailnet vouches for. The
last mode is people on **any** network: a colleague on hotel Wi-Fi, a
student on a phone plan. They need a real login, and the box needs a
public name with a certificate a stranger's browser trusts. Two lines
the owner had already drawn: PiCode never speaks ACME (a proxy in front
terminates public TLS), and the list of who may enter stays the admin's.

The owner also asked for the strong-isolation piece now rather than in
a later SaaS track: a Linux user is a fair fence for a team; for people
you do not know, `pi` is a shell on your machine.

## Decision — D.1, login for the internet

1. **Two front doors, one handler.** `listen` (`:443`, Tailscale leaf)
   keeps admitting tailnet peers by `whois`. `plainListen`
   (`127.0.0.1:8480`) sits behind a TLS proxy — Caddy, Cloudflare Tunnel
   — that serves `publicUrl`. `X-Forwarded-For` is believed only when
   the connection comes from a `trustedProxies` CIDR, and only its last
   hop (the proxy's own observation); anything else is a claim.
2. **Identity chain, per request:** tailnet peer → `whois`; else a
   valid gateway session cookie → login; else the login page. Nothing
   else is ever an identity. The members map is the same
   `users add` list for both doors, so `alice@example.com` (Google) and
   `octocat@github` (GitHub, spelled as Tailscale spells it) are one
   line each.
3. **Providers, standard library.** Google is OpenID Connect: discovery,
   Authorization Code + PKCE + `state` + `nonce`, the ID token verified
   against the JWKS (RS256, `crypto/rsa`), `iss`/`aud`/`exp`/`nonce`
   checked, login = verified email. GitHub has no OIDC for users: OAuth
   2 code + `state`, then `GET /user`, login = `<login>@github`.
   Provider endpoints can be set by hand only for a local fake provider
   in tests, and such a config is refused on the tailnet listener.
4. **The gateway's session** is a signed cookie (`picode_gateway`:
   `login|exp` + HMAC-SHA256 with a key in `gateway.secret.json`,
   0600), 30 days, `Secure; HttpOnly; SameSite=Lax`. No session table.
   The daemon's own `picode_session` (a paired device, ADR-0049) remains
   the second factor: after signing in, the first visit still lands on
   "Pair this iPhone".
5. **Routes under `/-/`** (unused by the SPA and the daemon): `/-/login`,
   `/-/auth/start/<p>`, `/-/auth/callback/<p>`, `/-/auth/logout`,
   `/-/healthz`. A navigation that needs a login gets `303 /-/login`;
   an API call gets `401 {"login":"/-/login"}`, which the SPA's pairing
   screen turns into a **Sign in** button.
6. **Hardening.** Per-peer limits on `/-/auth/*` and `POST /pair` (5 a
   minute, then ten minutes out); `state` single-use, ten minutes; every
   gateway response carries HSTS, `Referrer-Policy: same-origin`,
   `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`,
   `Permissions-Policy` (camera, microphone, geolocation off). A
   proxied daemon lists only its public target in `/api/share` — no LAN
   or tailnet addresses leak — and never starts the mkcert trust page.
   CSP shipped the same day (amendment below).
7. **Secrets stay out of `gateway.json`.** `picode gateway oidc set
   <provider> <id> <secret> --public-url https://…` writes the client
   credentials and the cookie key to `gateway.secret.json` (0600) and
   only the provider's presence, the public URL, the plain listener and
   the trusted proxies to the 0644 config.

## Decision — D.2, a container per person

`picode provision --user alice --shared --container` (root) runs
Alice's PiCode inside a `systemd-nspawn` container instead of directly
as her user. Same account, same home (bound into the container), same
gateway, same `users add`; the gateway's `Resolve` is unchanged because
the daemon still publishes `~/.picode/server.json` on the host's
loopback.

- **Root filesystem of her own** (`/var/lib/machines/picode-alice`,
  `debootstrap --variant=minbase` of the host's release with `nodejs`,
  `pi`, `tmux`, `git`): she cannot see the host's `/etc`, other homes,
  or the host's `pi` configuration. The shared `picode` binary is bound
  read-only, so an update is still one copy.
- **A private user namespace** (`--private-users=pick`), capabilities
  dropped to `CAP_NET_BIND_SERVICE`, cgroup limits (`CPUQuota=200%`,
  `MemoryMax=4G`, `TasksMax=512`) in the unit
  (`/etc/systemd/system/picode-alice.service`).
- **Host networking, on purpose:** the daemon binds the host's
  `127.0.0.1:<port>` so the gateway needs no change, and `pi` needs
  outbound access anyway. The kernel and the network stack are shared;
  that is the honest boundary of this tier. Firecracker or a VM per
  client is the next tier, for a SaaS track, not this one.
- `--remove` undoes the unit and the root filesystem; the account and
  home (her data) are `userdel -r`, a deliberate second step.

## Consequences

- Four modes, one binary: laptop, own server, shared box, public.
- A public deployment is `picode gateway` + a proxy recipe (guide) +
  a Google or GitHub OAuth app. PiCode holds no user database: the
  provider says who, the admin says whether.
- The container tier needs `systemd-container` and `debootstrap` on the
  host and a few minutes of network on first provision.

## Alternatives considered

- **Tailscale Funnel** — rejected: still Tailscale identity; no login
  for someone who is not on the tailnet.
- **An auth proxy in front (oauth2-proxy, Authelia)** — rejected:
  identity must stay one code path, and the gateway already terminates
  the request; a second component would have to be trusted by header.
- **A session table for the gateway** — rejected: a signed cookie
  needs no state and no cleanup; revocation is `users remove` (checked
  on every request) plus the 30-day expiry.
- **Docker per member** — rejected for now: nspawn ships with systemd,
  needs no daemon, and the unit is a plain text file `provision` can
  converge on.

## Amendment 2026-09-02 — Content-Security-Policy

The daemon sends the policy itself, on HTML responses, so every mode
gets it — not only the public one. The app shell's one inline script
(the theme bootstrap, which must run before the stylesheet) is named by
**hash**, computed from the served `index.html` (`inlineScriptHashes`,
once for an embedded build, per request for a disk build so `make web`
keeps working under a running daemon); no nonce, no template. The rest
is same-origin plus what the app really uses: `wasm-unsafe-eval`
(model-viewer, excalidraw), `img-src https:` (provider icons from
unpkg, images in chat), `data:`/`blob:` for screenshots and recordings,
inline style attributes (React), `worker-src blob:`, and the request's
own host for `ws://`/`wss://` so every browser's WebSocket passes.
Assets and API answers carry no policy, which keeps a same-origin
worker (excalidraw's font subsetter calls `new Function`) in its own
unrestricted context. `/pair` and the gateway's pages carry a closed
policy: no scripts, inline styles, nothing external, no framing. The
gateway forwards the daemon's headers on proxied pages and adds its own
only to what it renders. Verified in the browser on the desktop shell,
the chat, Preferences, Automations, Devices and the mobile shell with
no violations.
