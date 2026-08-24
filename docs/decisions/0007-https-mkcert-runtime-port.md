# ADR-0007: HTTPS by default with mkcert trust — port configurable at runtime

- **Status**: accepted (supersedes the localhost-only clause of the
  architecture.md security model)
- **Date**: 2026-08-24

## Context

PiCode is a browser tool; users access it from the machine's browser and,
increasingly, from other devices (phone over tailnet, LAN). The M0 contract
(localhost-only bind) blocked the product's own mobile/tailnet use case, and
plain HTTP on a LAN is not acceptable. Meanwhile the owner's agentdeck
project proved a TLS pattern in production: self-signed bootstrap +
optional mkcert-issued local CA with system/WINDOWS/iOS trust import.

Port conflicts also surfaced: fixed ports collide with sibling tools
(agentdeck runs on 8444) and users must be able to change the port without
touching env vars — the audience includes terminal-averse users (the port
setting must live in the Settings UI).

## Decision

1. **HTTPS always by default** (bind 0.0.0.0). `PICODE_INSECURE=1` disables
   TLS for dev/behind-proxy only. Certificates: `scripts/setup-cert.sh`
   issues via **mkcert** (installs mkcert if missing, discovers SANs =
   localhost + LAN IPs + tailscale, exports CA to Windows trust store on
   WSL with one UAC prompt, optional iOS import via `--ios`). Without the
   script, the server generates a **self-signed cert** (10y, same SANs) —
   zero-config bootstrap; browsers warn until the mkcert upgrade runs.
2. **Port from the Settings UI (source of truth)**: `settings.port` in the
   DB, editable at `#/settings`; precedence **DB (UI) > PICODE_PORT env >
   default range 8445-8455**. UI changes are single ports (validated and
   probe-bound before accept — busy port = 409, current server untouched);
   ranges via env are the headless/ops affordance.
3. **Graceful rebind**: on port change the server binds the NEW listener
   before shutting the old one down; on bind failure the setting reverts
   and the current server keeps serving. The settings response carries the
   new port; the UI reconnects automatically.
4. **Discovery**: `<data>/server.json` (`url`, `port`, `pid`) written on
   every bind — scripts and future CLI tooling find the live server without
   guessing ports.

## Consequences

- **Easier**: green-padlock HTTPS from the Windows browser over WSL; phone
  access via tailnet works with the same cert; no port fights between
  local tools; port changes need zero terminal knowledge.
- **Accepted risk**: binding 0.0.0.0 without app-level auth — same stance
  as agentdeck: personal-machine/tailnet tool, trust boundary is the
  local network. Documented; token auth remains the recorded debt if
  PiCode is ever exposed beyond the tailnet.
- **Harder**: rebind loop in main.go (bind-new-first, revert-on-failure);
  mkcert CA lifecycle (weekly systemd timer renews; WarnIfExpiring logs).

## Alternatives considered

- **Stay localhost-only + token**: rejected — kills phone/tailnet usage,
  the pattern's main point; agentdeck proved the risk acceptable.
- **Let's Encrypt / public DNS**: rejected — needs a public domain for a
  local tool; mkcert gives the same UX offline.
- **Fixed single port**: rejected — collisions with sibling tools; the
  owner runs multiple agent web tools concurrently.
