# ADR-0050: A PiCode you own on a tailnet server

- **Status:** accepted (B.1, 2026-09-02); B.2 pending — see "Deferred"
- **Amends:** ADR-0007 (host and public URL become settings; the "no public DNS" alternative is re-opened as B.2), ADR-0018/0020 (the unit gains an environment drop-in), ADR-0043 (the native host can target another machine), ADR-0049 (the install token is a token session)
- **Roadmap:** `docs/design/remote-modes-roadmap.md`, Track B

## Context

Track A (ADR-0049) made every non-loopback request authenticate. That is
necessary for a server but not sufficient: nothing else in PiCode knew it
could live anywhere but the machine in front of the owner.

- The bind host was `PICODE_HOST` only; the public address did not exist
  as a concept, so `server.json` always said `https://localhost:8445` and
  pairing links were built from whatever host the browser happened to use.
- The systemd unit was fixed text: an operator who needed
  `PICODE_DATA` or a CA for Node had to hand-edit it, and `picode deploy`
  rewrote it.
- `picode update` installed whatever the release URL returned. The
  release workflow publishes `SHA256SUMS`; nothing read it.
- The Chrome native host and `pi-inbox` found the daemon through the
  local `server.json` and token — files that do not exist on a PC that
  only runs the extension.
- Reading the code also showed a Track A defect: the install token was a
  hash in memory with a synthetic principal, so the extension's presence
  ping never joined a row on Devices.

## Decision

1. **Host and public URL are settings.** `server.host` follows the port's
   precedence (DB > `PICODE_HOST` > `0.0.0.0`); `server.public_url` has no
   env (advisory; a server that needs it has the UI). `PUT
   /api/server/host` validates an IP on this machine, probe-binds and asks
   the main loop to rebind — the reply leaves on the old listener, the UI
   reconnects on the new address, and a failed bind reverts both host and
   port. `PUT /api/server/public-url` never moves the listener: pairing
   links (`pairLinks`), `server.json` (`url`, `bind`, `publicUrl`) and the
   phone drawer (`share.Diagnose`, a `public` target listed first) read
   it; `auth.HostAllowed`/`originAllowed` already accepted it. `GET
   /api/server` lists the machine's interfaces and suggests the tailnet
   name and IP so Preferences → Server can fill the field with a click.
2. **The unit's environment lives in a drop-in.**
   `~/.config/systemd/user/picode.service.d/env.conf`, written by `picode
   install --env KEY=VALUE` (repeatable, merged, `""` removes), parsed and
   quoted the way systemd does. `picode deploy` never touches it. This is
   how `PICODE_DATA`, `PICODE_HOST` for a first start, and
   `NODE_EXTRA_CA_CERTS` for `pi-inbox` reach the service.
3. **`picode update` verifies `SHA256SUMS`** before `Deploy`, and refuses
   a release that has none. The check is a pure function
   (`install.VerifySHA256`) so the three outcomes are unit-tested.
4. **No `doctor` verb.** `picode provision --dry-run` already is a
   read-only, readable table over a converging Step engine. Three rows
   join it — `pi` on PATH, Tailscale up (with the tailnet name), and
   "reachable from other machines" read from `server.json` — instead of a
   second command that would print the same engine's output.
5. **Off-box clients read `remote.json`.** `<data>/remote.json` (`url`,
   `token`, optional `caFile`) wins over `server.json` and the local token
   in the native host; `picode extension-install --server URL --token T
   [--ca FILE]` writes it (0600). TLS is verified for names — system roots
   plus `caFile` — and skipped only for loopback and IP literals, the
   addresses mkcert covers. `pi-inbox` honours `PICODE_URL` and
   `PICODE_TOKEN` the same way.
6. **The install token is a token session** (label "Install token"):
   `<data>/token` holds its secret, bearers resolve through
   `LookupSession` alone, rotation revokes the old row. It lists on
   Devices and presence joins onto it, so a connected extension shows
   there; the duplicate status line in Preferences is gone.

## Deferred: B.2 — `tailscale cert` served by SNI

The roadmap's headline for this track was a certificate the phone
already trusts on the tailnet name. That means serving a Let's Encrypt
leaf for `box.tailxxxx.ts.net` *next to* the mkcert leaf for IPs, chosen
per handshake by `ClientHelloInfo.ServerName` in `tlsutil.LiveConfig`,
issued by `tailscale cert` and renewed in-process. It is the one item
that touches the live TLS path of the owner's daily instance, and the
mkcert leaf already covers the tailnet IP end to end (pairing over
`100.x` verified 2026-09-02). It ships as its own amendment to this ADR
once the pieces above have been dogfooded on a real box. Until then the
phone trusts the mkcert CA once, through the QR's trust page.

## Known limits (documented in the guide, not hidden)

- `reveal`, the folder picker, llama.cpp and `gh` act **on the server**.
- Provider OAuth callbacks bind loopback ports on the machine running
  `pi` (Anthropic 53692, Codex 1455, MCP auto-auth): from a remote
  browser they need `ssh -L`/`tailscale` port forwarding or a device-code
  provider. The guide says which.
- `pi-inbox` is Node: for a name it needs the CA Node trusts —
  `NODE_EXTRA_CA_CERTS` in the drop-in until B.2 gives a public chain.

## Consequences

- One daemon, one settings surface, for M0 and M1: nothing changes on a
  laptop; a server is the same binary told where it lives.
- `server.json` gained two fields; every reader keeps working (`url` is
  unchanged for local clients).
- Verified updates close a supply-chain hole that predates this track.

## Alternatives considered

- **`picode doctor` as a new command** — rejected: same engine, second
  output; the Step list is where server checks belong.
- **Names as bind hosts** — rejected: a bind is an interface, and a name
  that resolves elsewhere tomorrow would silently take the server off the
  machine.
- **ACME inside PiCode** — still rejected (ADR-0007); B.2 uses Tailscale's
  issuance, M3 terminates TLS at a proxy (ADR-0052).
- **`NODE_TLS_REJECT_UNAUTHORIZED=0` for `pi-inbox` remote** — rejected:
  it disables verification for every request the agent makes.

## Amendments

- **2026-09-02 — a bind to an outside address keeps loopback.** The
  owner's first click on *Tailnet only* took `localhost` away: the new
  bind on the same port could not bind-new-first (0.0.0.0 overlaps every
  address), the loop reverted, and the tab had already moved to the
  tailnet address — unreachable from the Windows browser and without a
  cookie there. Now a bind to a specific outside address also listens on
  `127.0.0.1` on the same port ("Tailnet and this machine", never
  "instead of"); a host change drops the old listener, binds the new one
  and restores the old one if that fails; the tab keeps a local origin.
