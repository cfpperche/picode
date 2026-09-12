# ADR-0122: shell-app-bar

- **Status**: accepted
- **Date**: 2026-09-12
- **Boundary**: protocol — supersedes ADR-0121's frame handshake between the desktop shell and the served UI: the window frame is now drawn entirely by the shell and never by the served UI.

## Context

ADR-0121 split the frame across two sides with a runtime handshake: the
shell injected a fallback frame into every served page, and the UI —
detecting the shell — claimed the frame and drew controls in a reserved
slot of its own chrome. The owner reviewed the result and rejected it:
the controls floated over view-specific content and read as an overlay
stuck onto the app. The reference is OpenAI's ChatGPT desktop app: a
dedicated, full-width app bar owned by the shell — brand on the left,
flat Windows caption buttons on the right, the whole strip is the drag
region — and the browser renders no bar at all.

Facts that shape the decision: Tauri 2's multiwebview mode (the
`unstable` feature) lets one window host several webviews with explicit
bounds; a local page's IPC is trusted by default, while a remote origin
needs a capability grant; and the daemon serves the deployed bundle, so
anything frame-related drawn by the served UI ships only at deploy time.

## Decision

Every shell window is undecorated and hosts **two webviews**: a 40px
local app bar (`titlebar.html` — brand, drag region, double-click
maximize, flat caption buttons with the Windows close-red hover) across
the top edge, and the page itself — the daemon's UI for the main window,
the Management page for the second — filling the rest. A resize handler
restretches both webviews in logical pixels. The bar is local on
purpose: the ACL trusts local pages, so the frame works with any bundle
the daemon serves and no initialization script touches the served UI
anymore. ADR-0121's frame mode, handshake, and reserved-slot convention
are reverted; the capability keeps the daemon origin trusted for the
core window permissions the content webview's commands may still need.

## Consequences

The frame is one implementation, owned by the shell, identical on every
window, and immune to what the daemon serves — no handshake, no
reserved-slot convention in the web app, no deploy coupling. The costs:
the shell now depends on Tauri's `unstable` multiwebview feature (API
surface may move between Tauri releases; pinned by the lockfile), the
two-webview layout is repositioned manually on every resize (a few
rect assignments per event), and the served UI gains ~40px less height
inside the same window size. Menus in the bar (File/Edit/View like the
reference app) are deliberately deferred — the bar ships with brand and
controls only; native menus would be a separate decision. If the
multiwebview feature breaks in a Tauri upgrade, the fallback is the
pre-0121 single webview plus injected frame, which this ADR supersedes
but does not delete.

## Alternatives considered

**Frame mode in the served UI (ADR-0121)**: lost — owner review found
the controls colliding with each view's own top-right widgets; the shell
cannot know where a view parks its content. **Native (Win32) custom
non-client area**: lost — deep owner-drawn title bar work in tao/Win32
that Tauri does not expose, for the same visual result the two-webview
bar already gives. **Two stacked windows glued together**: lost — move
and resize sync between windows flickers on Windows; child webviews in
one window resize atomically.
