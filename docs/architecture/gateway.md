# Shared box: the gateway (ADR-0051)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`picode gateway` is a root system unit on the tailnet interface
(`:443`, Tailscale-issued certificate). Per request it asks `tailscale
whois` who owns the peer address, maps that login to a Linux user
(`/etc/picode/gateway.json`, `picode users`), reads that user's
`~/.picode/server.json` and proxies to their daemon — one `picode` per
member, as their own user unit, on loopback, plain HTTP, mode `all`,
public URL = the box's name (`picode provision --user U --shared`).
A first visit with no session cookie gets a pairing code minted with
the member's install token and lands on `/pair`. Client `Authorization`
and `X-Forwarded-*` never pass; SSE and WebSockets do. A second, plain
listener behind a TLS proxy (ADR-0052) admits people off the tailnet by
a Google or GitHub login (stdlib OIDC/OAuth, signed session cookie,
routes under `/-/`); members may run in a systemd-nspawn container
each (`provision --shared --container`).
