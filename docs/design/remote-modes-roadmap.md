# Remote modes roadmap — localhost · tailnet (own) · tailnet (shared) · public

Owner direction (2026-09-01): PiCode must run in all four modes, and this
track is followed until it closes. One ADR per track; each track ships
independently and leaves the previous modes working.

| Mode | Daemon | Clients | Identity | Track / ADR |
|---|---|---|---|---|
| M0 localhost | your machine | your browser on it | loopback auto-pairs | A / ADR-0049 ✅ |
| M1 tailnet, own | a box on your tailnet | your PC + phone | device pairing over the tailnet | B / ADR-0050 |
| M2 tailnet, shared | one box, several people | each person's devices | Tailscale identity → per-person daemon, + pairing | C / ADR-0051 |
| M3 public | a box on the internet | whoever is allowed | OIDC login in the gateway, + pairing | D / ADR-0052 |

Decisions taken: shared = one daemon per Linux user behind a gateway
(pi gives a shell, isolation must be OS-level); public = OIDC in the
gateway, stdlib; TLS = `tailscale cert` on the tailnet, an external proxy
terminates in public, never ACME in PiCode; default auth mode = `remote`.

## Track A — authentication core (shipped)

`internal/auth` gate on every `/api` and `/ws` request; Host/Origin
checks in every mode; modes `off | remote | all`; pairing codes and
browser sessions (`auth_sessions`, `auth_pairings`); install token for
scripts, the Chrome native host and `pi-inbox`; Devices in Preferences →
Server; `picode pair`, `picode token [rotate]`; the phone QR carries a
pairing link.

## Track B — tailnet server, one owner

**B.1 (shipped 2026-09-02, ADR-0050):** `server.host` and
`server.public_url` settings with a Reach-this-server block in
Preferences; `server.json` advertises `bind` and `publicUrl`; systemd env
drop-in (`picode install --env`); `picode update` verifies `SHA256SUMS`;
release-binary install path in `www/guide/remote-server.md`; server
checks in `provision --dry-run` (pi, tailnet, reach — no separate
`doctor`); off-box clients via `~/.picode/remote.json` (`picode
extension-install --server --token --ca`) and `PICODE_URL`/`PICODE_TOKEN`
for `pi-inbox`; the phone drawer lists the public target first. The
install token became a token session (Devices shows the extension).
Documented limits: reveal / folder picker / llama / `gh` act on the
server; OAuth loopback callbacks need a port-forward or a device-code
provider; `pi-inbox` needs `NODE_EXTRA_CA_CERTS` for a name.

**B.2 (shipped 2026-09-02):** `tailscale cert` for the MagicDNS name,
served next to the mkcert leaf by the name the client asks for
(`tlsutil.LiveConfig`), issued and renewed by the daemon
(`KeepTailscaleCert`) and by `provision` (`tailnet-cert` row); the phone
drawer lists the tailnet name as a trusted target that skips the trust
page. Left for later: reconnect tolerating one missed ping on mobile.

## Track C — shared tailnet server

`picode provision --user` creates a per-user daemon (`127.0.0.1` or a
unix socket, `auth.mode=all`); `picode gateway` (system unit, tailnet
:443) identifies the person through Tailscale (`tailscale serve`
identity headers, else the local `whois` API) and reverse-proxies HTTP,
WebSocket and SSE to that person's daemon; `login → linux user` map in
`/etc/picode/gateway.json`; first visit auto-pairs the device through a
gateway-signed hop; the gateway never forwards a client's Authorization.

## Track D — public access

OIDC Authorization Code + PKCE in the gateway (`internal/oidc`, stdlib
JWKS verification), allow-listed logins; rate limits on `/pair` and the
callback; audit events through the feed; HSTS / CSP / Referrer-Policy /
Permissions-Policy; `:8470` disabled; `/api/share` hides LAN details;
`X-Forwarded-*` trusted only from the configured proxy CIDR; Caddy and
Cloudflare Tunnel recipes; PiCode never speaks ACME.
