# ADR-0043: Browser extension is a native-messaging client of existing agents

- **Status**: accepted
- **Date**: 2026-09-01

## Context

PiCode agents already drive an isolated Chromium (`agent_browser`). That
reaches public pages and headed logins in a parallel browser. It does not
see the tab the human is looking at, the text they selected, or the SSO
session in their daily Chrome.

A Chrome extension is the product shape that closes that gap: a side panel
and a context menu that send **this** tab to an agent that already exists.
Three extension stories already live in the repo and must not be mixed:

| Surface | Owns |
|---|---|
| Pi packages (ADR-0010) | tools inside the `pi` process |
| Apps host (ADR-0036) | GUI inside PiCode |
| This ADR | reach into the human's everyday Chrome |

Benchmarks that set the shape: Cursor's adaptation rule (agent *control*,
not a worse IDE/chatbot); t3code's busy-send = follow-up; paseo's many
clients (the extension is one more, like a phone on `#/devices`); Claude
in Chrome / Edge Copilot (side panel + current tab, not a browser fork);
`packages/pi-inbox` (thin client reads `server.json` and POSTs loopback).
Comet/Dia-as-browser and MV3-as-Playwright were refused.

Constraints that forced the transport:

- Chrome native-messaging max message is 1 MB.
- PiCode's port is dynamic (`server.json`, ADR-0007) and TLS is often
  mkcert/self-signed.
- The owner's Chrome is **Windows**; the server is **WSL**. A Linux
  `picode` binary cannot be Windows Chrome's native host.

## Decision

PiCode ships a Chromium MV3 extension in `ext/` (sideload in v1) that
talks to the local server **only** through a native-messaging host in the
Go binary.

1. **The agent stays `pi`.** The extension is a sensor plus a short
   composer. It does not embed the ADE, does not call a model, and does
   not replace `agent_browser`.
2. **Host is the same product binary.** Chrome launches it with
   `chrome-extension://<id>/` as argv[1] (also `picode browser-host` for
   tests). On Linux/macOS that binary is `picode`. On Windows it is
   `picode-desktop`, which reads `server.json` through WSL and proxies
   HTTPS loopback, same as the tray health poll.
3. **v1 is sensor-only.** Current tab URL, title, selection; screenshot
   opt-in (JPEG, under the 1 MB native-messaging cap). Actuating the page
   and an agent tool (`read_current_tab`) are later tracks, not this ADR.
4. **Stopped agents start.** `POST /api/extension/send` starts managed
   mode then delivers the prompt. Interactive (tmux) agents are refused
   with "This agent is in the terminal." — a context-menu click must not
   kill the TUI (ADR-0006 exclusivity still holds; we just do not trip it
   from the extension). Busy managed agents reuse `SendTurn`'s prompt →
   follow-up mapping (conversation-control C3).
5. **Pinned extension id.** `ext/manifest.json` carries a public `key` so
   sideload keeps a stable origin for the native-host allow-list
   (`beoccbnjejkjjjcmcfhnnklbjaaddolp`). Chrome-only in v1.
6. **Not an App, not a pi package.** `#/devices` may later show the
   extension as a client (Track B). Agent-facing tools wait for a
   `packages/pi-tab` carve-out, the `pi-inbox` precedent.

## Consequences

- **Easier:** page context reaches the agent the user already configured;
  pairing survives port rebinds because the host re-reads `server.json`;
  WSL is a first-class path rather than an afterthought.
- **Harder:** Windows Chrome cannot be served by `picode extension-install`
  inside the distro — the native host must be a Windows exe
  (`picode-desktop extension-install`). Sideload is the v1 distribution
  (no Web Store). `windowsgui` on `picode-desktop` may fight Chrome's
  stdio pipes; if dogfood proves that, a console sibling is the fix, not
  a protocol change.
- **Accepted cost:** the extension inherits ADR-0007's trust boundary
  (personal machine / tailnet, no app-level token). Screenshot is opt-in
  and JPEG-capped so native messaging does not silently drop the send.
- **If wrong:** delete `ext/` and the `/api/extension/*` routes; the host
  subcommand becomes a no-op. Isolated Chromium is unchanged.

## Alternatives considered

- **`fetch('https://localhost:8445')` from the extension** — port and
  cert churn; WSL Chrome cannot read `~/.picode/server.json`. Lost.
- **Iframe the ADE in the side panel** — reuses UI but cannot capture the
  tab and is a worse PiCode at ~400px. Lost (Raycast/ADR-0036 primitives).
- **CDP attach to the user's Chrome** — cheaper spike, fragile profile
  lock, not a product. Deferred, not v1.
- **Replace `agent_browser` with MV3** — service workers sleep; content
  scripts lose to CDP on waits/snapshots/record. Both stay.
- **Put the extension in `packages/` or `web/`** — `packages/` is the MIT
  pi-package carve-out; `web/` is the ADE SPA. A fourth client belongs at
  `ext/`.
