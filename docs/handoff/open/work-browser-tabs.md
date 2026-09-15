# Work browser (Phase 3)

Slices 1 and 2.1 (tabs, toolbar, browser settings, screenshot) are accepted
and deployed; slice 2 landed in three parts — the bridge and tier gate
(`feat/browser-cdp`), the command channel (`feat/browser-agent`, ADR-0132)
and the `browser` Pi tool with its policy default (`feat/browser-policy`,
ADR-0134: read on the tab on screen). Plan: `docs/plans/desktop-v2.md`
(Phase 3 + the slice 2 section), policy: ADR-0128 as amended by 0134.

The split view with per-pane bindings (ADR-0135, `feat/agent-browser-split`)
landed 2026-09-14, accepted live on Windows the same day (snapshot +
screenshot round-tripped through the bound pane).

## Next

- Slice 4 (`feat/browser-grants`) left: `act` verbs + origin rule, grants editor, shell navigation gate mirroring `browser.AllowsOrigin`.
- New browser tab button (tab-strip end) crashes the app (`ReferenceError`, React root unmounts; verified pre-existing on a `main` scratch 2026-09-15).

## Notes

- **Split survives relaunch: DONE (2026-09-14)** — layout + last url persist; boot prunes dead hosts; webview recreated on host-tab show. Post-deploy: re-check the recreate in the shell.

## Slice 3, state (owner reviewing the Settings ▸ Browser parity list one
item at a time, 2026-09-15)

Landed: the history store and its dropdown, the Clear browsing data dialog
(per-kind WebView2 masks + time range), the Browsing history dialog (search,
day groups, favicons, per-row menu, bulk remove), the reference-shaped
settings page with the master Browser switch, open destinations, Show full
URL, password/contact autofill.

Still unbuilt, in the owner's order: **Downloads** (Location, "Ask where to
save downloads", Download history — recipe in
`docs/handoff/2026-09-14-browser-downloads.md`), **Browser permissions**
(Site settings camera/mic — the `PermissionRequested` handler — the History
select, Enable site tools), **Developer mode** (the elevated-risk toggle over
full CDP access), then find in page and the device toolbar.

## Annotations (backlog, owner-registered 2026-09-15)

The reference has "Annotation screenshots" (Always include / Only when needed
/ Never). It is deliberately absent from our page: PiCode has no annotation
feature, and a switch that controls nothing is a dead control. Build the
feature, then the row.

1. **Annotate mode in the work browser tab** — pick an element or draw a
   rectangle, then a comment box.
2. **Capture** — the region through CDP `Page.captureScreenshot` with `clip`,
   plus the selector and the URL.
3. **Store + endpoints** — migration, feed event (ADR-0048).
4. **Delivery to the agent** — the annotation (comment + image) enters the
   session the agent reads.
5. **The settings row** — "Annotation screenshots", honoured by the capture
   step, once 1–4 exist.

Step 4 opens a new user→agent input path: it needs an ADR before the
protocol is fixed.

## Traps (paid for, keep them paid)

- Edit in the worktree. UI edits that land in the root checkout leave scratch
  and deploy testing stale bundles (this happened three times).
- `toast(msg)` defaults to `err`; success needs `toast.ok`.
- `.web-tab-toolbar button` beats bare class selectors — prefix toolbar button
  overrides with `.web-tab-toolbar`.
- `um-popover` is styled for the user menu; do not reuse it elsewhere.
- Data-URL downloads are blocked inside WebView2: write files from Rust and
  toast the path.
- Tauri commands that create or drive webviews must be async (a sync command
  runs on the main thread — the lab's v1.0 freeze).
- `webview2-com-sys 0.38` pins windows/windows-core 0.61; the handlers
  (`CapturePreviewCompletedHandler`,
  `CallDevToolsProtocolMethodCompletedHandler`) come ready-made.
- COM interfaces are not `Send`: keep event receivers on the UI thread (the
  `RECEIVERS` thread-local in `btab.rs`), never in Tauri state.
- An HTML popover can never paint over a WebView2 sibling: the options menu
  slides the page down (`MENU_H` in `WebTab.jsx`) instead of flipping z-order.

## Scope (v1): Pi only, and read for a TUI

The tool is a pi package, so reach follows install scope: **This agent** is
private to a managed agent, **This machine** (`~/.pi/agent`) and **this
project** are visible to a plain `pi` too. A managed agent reports its id and
can hold a grant; a pi TUI has no id, falls back to the default, and reads the
tab on screen. Identity is an assertion, not proof (ADR-0134) — a gate on a
missing id would be cosmetic, so v1 documents the behavior instead of faking a
boundary. Owner: validate in use, revisit scope after.

**Name collision (found 2026-09-13):** npm already has a `pi-browser`
("Playwright-backed pi extension that registers the pi-browser tool", keyword
`pi-package`). Ours is local-only today, so nothing breaks — but the Packages
gallery searches npm, so a search there offers *that* package, whose tool is
also called `browser`. Publishing ours needs a name decision (scope or
rename), and the gallery hit is worth a second look before anyone installs it
expecting this one.

## Debts

- `btab_layer.toml` (autogenerated) is a dead permission.

## Notes

- The options menu's page slide is not animated.
