# 2026-09-09 — feat/apps-native-surface: Matrix phase 1 — native app surfaces (ADR-0109)

Shipped: ADR-0109 (amends 0036) — `apps.Manifest.Surface` (`""` | `"native"`, `omitempty`, apiVersion stays 1), `apps.SurfaceNative`,
`internal/apps/native.go` (one detail view, 400 on action) and the hidden `demo-native` app beside `demo` (`PICODE_DEMO_APP=1`).
Contract: `supportedApp(manifest, nativeSurfaces)`, `nativeApp()`, `surface` kept verbatim by `normalizeManifests`. Desktop: registry
`web/desktop/src/lib/nativeApps.js` (`nativeApps`, `appTile`, `nativeSurfaceFor`; the instance is assembled in `App.jsx`),
`NativeDemoSurface.jsx` (AppSurface chrome + the fleet's first terminal through `TermSurface`), `host` = `{fleet, openTabs, openTab,
openInteractive, revealAgent, openFileTab, feed}`; an unregistered native tile stays dimmed with a title. Mobile: tile "Desktop only"
(disabled, no navigation); `#/app/<id>` → `NativeAppNotice` (one line + Back). Engine: `ShellTerm`'s active effect re-appends the pane
and a host parks only a pane it still holds; `suspendTermSocket` / `isTermSocketSuspended`. Docs: routes.md Apps sentence,
agent-manager.md manifest fields, fragment `docs/changelog.d/apps-native-surface.md`.

Verified: `make ci-scoped` PASS (fmt, vet, hooks, go[5], test-js 1 214 tests, build). Decision table: Go `TestNativeDemoApp` /
`TestNativeAppOnTheWire`; contract `supportedApp gates on the surface a shell can draw` (one assertion per row); desktop `appTile:*` /
`nativeSurfaceFor:*` (7 tests); termSocket suspend / kick / drop (3). Scratch QA (`qa-scratch apps-native-surface`, `agent-browser
--session apps-native-surface`): pane hand-off tab↔app both ways with the same xterm (32 vs 29 rows per host); suspend on the live entry
wrote no "— detached —" line and kick reconnected the same instance; mobile tile disabled, deep link + Back; overlay audit ok; 12 captures
in `var/screenshots/ans-*.png` (dark and light), read in subagents. Fixture terminals deleted through the scratch API before stop.

visual-review: PASS (ans-1…12, card 5/5 each; note: dark `btn-primary` label contrast is 3.02:1 — theme token, pre-existing)

Not done / debts: feed rows carry no `session`, so the demo POSTs `/api/terminals/{id}/open` itself (`host.openTerminal` is a phase-3
candidate); closing a terminal's tab while a native app shows its pane disposes the xterm — phase 3's ownership rule (plan §4.5)
decides; the desktop's unsupported tile hints through `title` only.

Merge: main moved during the session (phase 2 landed) — merged into the branch before `make close`; fast-forward ready.
