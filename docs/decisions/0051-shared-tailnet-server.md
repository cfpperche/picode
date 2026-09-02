# ADR-0051: Several people on one tailnet box — a daemon per person, a gateway in front

- **Status:** accepted (2026-09-02)
- **Amends:** ADR-0049 (mode `all` from the environment), ADR-0050 (public URL from the environment; diagnosis behind a proxy), ADR-0020 (a second provisioning list, for members)
- **Roadmap:** `docs/design/remote-modes-roadmap.md`, Track C

## Context

A PiCode gives its person a shell: `pi` runs commands, edits files,
holds provider credentials. Two people sharing one daemon would share
all of that, whatever the UI pretended. So "shared" cannot mean a
multi-user application; it has to mean **one daemon per Linux user**,
and the operating system keeps them apart. What is left to build is the
front door: something on the tailnet that knows who is knocking and
sends them to their own daemon.

The tailnet already knows. Every connection arrives through Tailscale
from a node that belongs to a login; `tailscale whois <ip>` returns that
login. No password, no second identity provider, nothing the client can
forge — the peer address is a property of the tunnel, not a header.

## Decision

1. **Topology.** `picode gateway` — a system unit, root, the only thing
   listening on the tailnet interface (`:443`, the Tailscale-issued
   certificate for the box's name, ADR-0050 B.2). Behind it, one
   `picode` per member as their own user unit: `PICODE_HOST=127.0.0.1`,
   `PICODE_INSECURE=1` (TLS ends at the gateway — the documented use of
   that switch), `PICODE_AUTH_MODE=all` (everyone pairs, loopback
   included: the gateway *is* loopback to the daemon), and
   `PICODE_PUBLIC_URL=https://<box name>` so the daemon's Host/Origin
   checks, pairing links and drawer all speak the gateway's name.
2. **Identity.** `internal/gateway.Identity` asks `tailscale whois
   --json` for the peer IP (cached 60 s). Non-tailnet addresses are
   refused before asking. Nothing the request carries is consulted:
   incoming `Authorization`, `Forwarded`, `X-Forwarded-*`, `X-Real-IP`
   and `X-PiCode-*` are dropped before proxying, and the gateway sets
   `X-Forwarded-For` (peer), `-Proto` (https) and `-Host` itself.
3. **Mapping.** `/etc/picode/gateway.json`: `users` = Tailscale login →
   Linux user, edited with `picode users add|remove|list` (root). A
   login absent from the map gets a 403 page naming the command; the
   gateway never falls back to another daemon.
4. **Finding the daemon.** Per request, the gateway reads the member's
   own `~/.picode/server.json` (the URL — ports can rebind) and
   `~/.picode/token`. Root reading two files in a home is the whole
   privilege the gateway exercises; the token is used for exactly one
   thing (below) and never forwarded.
5. **First visit.** A browser navigation with no `picode_session`
   cookie at all: the gateway asks that member's daemon for a pairing
   code with the daemon's own token and redirects to `/pair?code=…`;
   the daemon's confirm page sets the cookie on the gateway's origin.
   One tap per device, ever. A cookie that exists but is stale is the
   SPA's business (its pairing screen), so the gateway cannot loop.
   One code per peer per 10 s.
6. **Streams.** `httputil.ReverseProxy` with `FlushInterval: -1` for
   `/api/events` (SSE) and the standard library's WebSocket pass-through
   for `/ws/*`. Tested with a live SSE frame and an upgrade echo.
7. **Members.** `picode provision --user <u> --shared` (root): account
   (`useradd -m`), linger, the shared binary at `/usr/local/bin/picode`,
   the environment drop-in above (written by root, handed back with
   `chown`), then the member's own provision pass as that user through
   `runuser` with their `XDG_RUNTIME_DIR` — which installs and starts
   their unit — and a loopback health check that also refuses a member
   daemon facing the network.
8. **Isolation invariants**, each with a test: a request reaches only
   the daemon of the identified login; remapping the same peer to
   another login reaches the other daemon; unknown login → 403; whois
   failure → 503 (never a default user); client `Authorization` never
   reaches a daemon; a member daemon on a non-loopback bind fails
   provisioning.

## Consequences

- One product, three modes, one binary: a member's PiCode is the
  ordinary PiCode told four things by its environment.
- The gateway is stdlib only; the box needs Tailscale with MagicDNS and
  HTTPS certificates. Members share the machine's CPU, disk and network
  like any Linux users do.
- Members' devices see one origin (`https://box.tailxxxx.ts.net`), so
  Web Push, `/api/events` and the phone shell work unchanged.
- The gateway's own pages (403, 503, 502) wear the same look as `/pair`;
  the template is duplicated on purpose — the gateway must not import
  the server.

## Alternatives considered

- **One daemon with in-app users** — rejected: `pi` is a shell; only
  the OS separates people.
- **`tailscale serve` and its identity headers** — rejected: the gateway
  terminates TLS itself and must not trust headers it did not set;
  `whois` on the peer address needs no trust in the request.
- **Unix sockets per member** — deferred: loopback plus the member's own
  `server.json` needs no port table and keeps the local clients
  (`pi-inbox`, the native host) working as they are.
- **Static port table in the config** — rejected: `server.json` is
  already the truth and survives a rebind.
