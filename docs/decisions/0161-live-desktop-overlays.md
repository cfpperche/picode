# ADR-0161: Live desktop overlays

- **Status**: accepted
- **Date**: 2026-09-20
- **Boundary**: protocol + security model — the trusted desktop document reports native paint/input regions through `chrome_layers`; external page webviews cannot change those regions.

## Context

The owner rejected hiding a work page behind a captured still when any HTML
layer crosses it. Focusing the address field could replace the entire page
with gray, even for an empty history list. The requested behavior applies
to all desktop overlays and keeps the underlying page alive.

An isolated Windows/WebView2 experiment demonstrated translucent HTML above
a live sibling page, typing in both documents, and containment after resize.
Transparency alone does not remove a native window's input region.

## Decision

Keep the existing React document and its components, state, handlers, Radix
portals and focus behavior. Host it as the transparent `main-content` child
of the existing main window, above native page siblings. `chrome_layers`
receives the visible page rectangles, floating layer rectangles and theme
background. The chrome's native region is the full window minus the pages,
plus the floating layers. Windows routes pointer input through the holes.
Ancestors of live page placeholders clear their CSS background; other
surfaces keep their existing paint. The underlying pages are not hidden or
captured to open an overlay.

The bridge accepts only the `main-content` webview, in addition to Tauri's
existing trusted-origin ACL. It validates finite, bounded rectangles and
caps each list at 128. The frontend serializes updates and coalesces DOM
changes into animation frames. It transfers geometry, not DOM, arbitrary
scripts, component state, action names or site content. A page remains
untrusted even when it navigates to a trusted server URL.

## Consequences

All host overlays use the same composition without cloning React state
into another browser context. Modal backdrops cover native pages while
allowing their pixels to remain live. Page visibility still follows tab and
route selection. Native layer allocation, clipping, DPI conversion and
lifecycle now require Windows tests as well as DOM geometry tests.

The shell advertises the new protocol before loading the document. During
paired shell/UI upgrades, older shells keep their existing behavior;
shipping the UI alone does not falsely claim native layering support.

The QA build feature is excluded from the resident build. Its example runs
against an explicit scratch loopback HTTP port with a separate temporary
profile and without single-instance, tray or keepalive duties. Its temporary
capability configuration is not part of the production build.

## Alternatives considered

A screenshot backdrop fails the owner's live-page requirement. Injecting
host controls into arbitrary sites crosses the wrong trust boundary.
Rendering each React overlay in a separate WebView duplicates state and
requires an action bridge for every component; one native chrome surface
preserves the existing UI instead. A runtime migration is unnecessary for
the composition demonstrated by the prototype.

## Amendment 2026-09-25 — the legacy hide path retired (owner)

The web no longer hides and freezes the native page for shells without `__PICODE_LIVE_LAYERS__`: `coverDecision`, `verifyPreviewUrl`, `subscribeFloatingLayers`, the still backdrop and `data-covered` are gone, and the overlay audit no longer accepts a parked page. `btab_preview` stays for annotation captures. Measured before removal: the installed `picode-shell.exe` (2026-09-25) injects the marker, and PiCode runs only on the owner's machine.
