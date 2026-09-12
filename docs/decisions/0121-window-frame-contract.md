# ADR-0121: window-frame-contract

- **Status**: superseded by ADR-0122
- **Date**: 2026-09-12
- **Boundary**: protocol — the contract between the desktop shell and the UI the daemon serves over HTTP: who draws the window's frame (drag regions and minimize/maximize/close), and how the two sides agree without coordinating releases.

## Context

The desktop shell (ADR-0120) runs Tauri windows undecorated — the native
title bar read as a second, empty bar above the app's own chrome, and the
owner asked for it gone. With `decorations: false` someone must draw the
frame: drag handling and window controls. Three facts constrain the
choice. (1) The main window's UI is served by the daemon over HTTP, and
the daemon serves the **deployed** bundle, not the repo's working tree —
a frame drawn only by the served UI stays invisible until the owner runs
`make deploy` (ADR-0105). (2) Tauri's ACL refuses `window.*` IPC from
remote origins unless the capability lists them, so any frame drawn by
the served UI needs the daemon's origin trusted (it is: `localhost:8445`
in the shell's capability file). (3) The same UI also runs in ordinary
browsers, where no window exists and nothing frame-related may render.

Options considered: a shell-drawn native bar above the webview (rejected:
recreates the two-bar problem); a frame drawn only by the served UI
(rejected: dead window until deploy); a frame injected only by the shell
(shipped first — worked with any bundle, but its fixed top-right cluster
collided with view-specific widgets, and the shell cannot know where each
view parks its own controls).

## Decision

The frame is claimed by whoever can serve it best, detected at runtime,
with no release coordination. The shell's initialization script sets
`window.__PICODE_SHELL__` on every page and injects a fallback frame:
mousedown/double-click drag wiring on the top chrome rows, plus a
minimize/maximize/close cluster fixed to the top-right corner. When the
served UI runs in shell mode it **claims the frame** by setting
`data-picode-frame="1"` on the root element and drawing its own controls
inside its chrome, padding its top bars clear of the reserved slot; the
injected script watches for the claim and removes the fallback cluster.
The handshake is monotonic per page load, self-healing across
navigations, and needs no version negotiation: before the UI's deploy the
fallback serves; after it, the UI's integrated frame wins.

## Consequences

The shell never shows a dead window regardless of which bundle the daemon
serves; the UI's frame is properly composed with its own layout instead of
floating over it; and the two sides can ship in either order. Views in the
served UI must honour the reserved top-right slot when
`data-picode-frame` is set — this is a permanent layout convention
(documented in `docs/benchmarks.md`), and a new view that parks controls
at the window's top-right without checking the flag will collide. The
daemon origin must stay trusted in the shell's capability for the
UI-drawn controls to work; revoking it silently kills every window action
from the served UI, which is why the capability file carries the reason.
Drag wiring stays shell-owned (it works on any bundle, old or new). If we
are wrong about the handshake — say the UI claims the frame and then fails
to draw controls — the window loses its buttons until the next deploy; the
mitigation is that claiming and drawing land in the same commit.

## Alternatives considered

**Shell-drawn native title bar strip** (a slim bar above the webview):
loses — it is the second bar the owner asked to remove, only thinner.
**Served UI only, no fallback**: loses — the window is unusable (no drag,
no controls) until the owner deploys, and old bundles stay broken after.
**Injected fallback only, no frame mode**: loses — the fixed cluster
collides with view-specific top-right widgets on every view it cannot
know about, which is the defect that prompted this ADR.
