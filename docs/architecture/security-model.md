# Security model (ADR-0007)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

- **HTTPS always** (bind 0.0.0.0): mkcert-issued cert via
  `scripts/setup-cert.sh` (SANs: localhost + LAN + tailscale; CA exported to
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
  `internal/auth` gates every `/api` and `/ws` request: principal from
  the `picode_session` cookie or `Authorization: Bearer` (install token
  at `<data>/token`, or a token session); `Host` and `Origin` checked in
  every mode; modes `off | remote (default: loopback auto-pairs) | all`.
  A browser-like loopback visit with no cookie reuses the newest live
  session with its user-agent label (secret rotated in place, presence
  asked first so an active browser keeps its cookie — ADR-0049
  amendment 2026-09-03) instead of minting a duplicate row per launch.
  Pairing codes (`/pair?code=`, ten minutes, one use, lockout after five
  failures) mint browser sessions; Preferences → Server lists and revokes
  them. Expired sessions stop listing; a daily sweep prunes
  revoked/expired rows after 7 days. PiCode executes with the user's
  permissions, like Pi itself.
  Roadmap for tailnet, shared and public servers:
  `docs/design/remote-modes-roadmap.md`.
- **HTML previews are a capability route, not the session** (ADR-0136):
  `POST /api/previews` mints an hour-long, session-bound ticket for one
  `.html`/`.htm`; `GET /preview/<ticket>/<path>` serves only allowlisted web
  assets under the owner's folder (dotfiles and symlinks out, `no-referrer`,
  `no-store`) to a frame sandboxed by response header (`sandbox`, never
  `allow-same-origin`). A sandboxed page sends no cookie, reads no `/api`
  answer (CORS) and its writes are refused as `Origin: null`; revoking the
  session kills its tickets; the daemon restart drops them all.
