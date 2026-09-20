# Work browser pending items

This plan closes the five open work-browser items identified in
`docs/handoff/open/work-browser-tabs.md`.

## Order and gates

1. **HTTP destination coverage** — cover terminal missing, plain shell refusal,
   Agent CLI delivery, and agent/terminal identity routing. Gate: the handler
   tests assert status, response reason, and delivery effect.
2. **Declaration-order check** — add a static check for component references
   that execute before lexical declarations. Gate: a regression fixture for the
   blank-window class fails, while valid deferred callbacks pass.
3. **Windows Ask acceptance** — run the allow-once, always-allow, block,
   timeout, close-tab, inactive-tab, and site-over-global cases on a scratch
   Windows host. Gate: screenshots and logs show each final state and no held
   deferral remains.
4. **Legacy COM capture** — add Windows coverage for valid, empty, failed,
   not-painted, timeout, and hide/restore capture paths. Gate: the test proves
   non-empty pixels and restores the native view.
5. **Responsive width** — add the native device-toolbar half: width/height
   presets, custom bounds, zoom, reset, per-tab state, and overlay geometry.
   Gate: media queries and menu/capture overlays follow the selected bounds.
   Full CDP device emulation needs a separate ADR.

## External acceptance

The Windows Ask and COM items cannot be declared complete from Linux or from
`cargo xwin build` alone. They require a real Windows run by the owner or a
controlled Windows host with captured evidence.
