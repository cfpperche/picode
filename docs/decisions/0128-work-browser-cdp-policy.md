# ADR-0128: work-browser-cdp-policy

- **Status**: accepted (owner, 2026-09-13 — direction agreed in session)
- **Date**: 2026-09-13
- **Boundary**: security model — how agents are granted access to live web
  pages (transport, policy gates, access tiers) and who can bypass them.

## Context

The Desktop v2 Phase 3 work browser is an embedded browser (ADR-0120 shell,
child WebView2s) where human and agents act on the same tabs — the ChatGPT
desktop Work browser is the benchmark. The engine spike (2026-09-12/13,
`feat/browser-lab`, documented in `docs/plans/desktop-v2.md`) proved on the
real shell: GitHub login, restart persistence, the full Google OAuth flow
with WebView2's own UA, and end-to-end CDP from the daemon side
(`Runtime.evaluate` + `Page.captureScreenshot` on the logged-in view) over a
loopback `--remote-debugging-port`.

Open forces for the product form:

- The owner's security requirement is a **domain allowlist per agent** plus
  **access tiers** (`read`, `act`, `full`) — the allowlist is intended as
  *the* control, so it must be structurally unbypassable, not advisory.
- The spike's loopback port has **no auth**: any local process can attach and
  drive the browser with the user's sessions, ignoring any policy the daemon
  enforces. Chromium's direction is also to restrict remote-debugging flags.
- WebView2 exposes per-controller CDP through the host API
  (`CallDevToolsProtocolMethod`, `GetDevToolsProtocolEventReceiver`); wry
  surfaces the `ICoreWebView2Controller` on Windows, so the shell can bridge
  without forking wry.
- The pipe (`--remote-debugging-pipe`) is not wireable: the WebView2 runtime
  owns the browser process launch, so the host cannot own the pipe fds.
- Layout (owner, same round): the work browser lives as **tabs in the
  editor** with a Chrome-inspired tab strip — not a fixed side panel.

## Decision

1. **Default transport: host-API bridge.** The shell bridges CDP over
   `CallDevToolsProtocolMethod` / event receivers per WebView2 controller,
   exposed to the daemon over the existing authenticated shell↔daemon
   channel. No debug port exists in default operation. Targeting is per
   browser tab (one controller each).
2. **Opt-in loopback port stays available.** Settings ▸ Browser ▸ advanced
   toggle (default **off**) enables `--remote-debugging-port` on loopback for
   special situations the owner wants (external tooling, Playwright-style
   debugging). The toggle's own copy states the cost paid when it is on: any
   local process can attach, and the agent policy binds only commands that
   flow through the daemon. The lab's `PICODE_LAB_NO_CDP` env gate evolves
   into this setting.
3. **Policy model, per agent, deny by default**: `{domains: [wildcard
   patterns incl. `localhost:*`], tier}` with tiers mapping to a CDP command
   catalog — **`read`**: screenshot, DOM/CSS/Network read, Emulation (no
   `Runtime.evaluate`, no `Input.*`); **`act`**: `read` + `Runtime.evaluate`
   + `Input.dispatch*` + `Page.navigate`; **`full`**: `act` + downloads,
   clipboard, `Page.printToPDF`, file chooser.
4. **Two gates, both enforced**: navigation (WebView2 `NavigationStarting`
   cancels any origin outside the agent's allowlist — the tab physically
   cannot load it) and command (the host-API bridge checks every CDP method
   against the tier before delivery). Documented nuance: `Runtime.evaluate`
   (tier `act`+) runs with the page's own trust; `read` has no evaluate, which
   is what makes read-only real.
5. **Profile**: one shared WebView2 user-data folder,
   `%LOCALAPPDATA%\PiCode\WebView2` (shipped in the spike) — work sessions
   are isolated from the user's personal browser by construction.

## Consequences

- The allowlist is structural by default: no unauthenticated local path to
  the browser exists unless the owner flips the opt-in, knowing the cost.
- Per-tab targeting is native (one controller per tab) instead of policing
  browser-level targets.
- Cost accepted: the shell marshals CDP JSON both ways (~1–2 days of Rust +
  a policy catalog table); we own that bridge code.
- Cost accepted (opt-in only): with the loopback port on, the policy is
  advisory for third-party local processes; the toggle copy says so.
- Cost accepted: Google's embedded-browser block is risk-based and
  machine-variable — the Chrome-UA override remains a per-domain fallback
  lever in the product, not a requirement.
- If we are wrong about tiers being enough (an agent escapes via page-side
  fetch, for instance), the remedy is tier `read` for that agent — the model
  degrades gracefully because `read` has no execute path.
- The daemon remains the policy decision point (it knows agent identities);
  the shell re-verifies navigation at the webview level. Neither alone can
  grant what the policy denies.

## Alternatives rejected

- **Loopback port as the default** (spike form): policy becomes advisory —
  any local process bypasses the allowlist; also a moving target as Chromium
  restricts debug flags. Kept, but only as the opt-in.
- **Pipe transport**: Chromium's CDP pipe belongs to the browser process
  launcher; the WebView2 runtime owns that launch, so the host cannot wire
  the fds.
- **Launching external Chrome/Edge** (original Phase 3 sketch): owner
  corrected 2026-09-13 — the product is an embedded browser (ChatGPT Work
  benchmark), not a launched one.
