# Live desktop overlays

Owner direction (2026-09-20): all PiCode desktop overlays must leave the work
page visible and live. Frozen screenshots and hiding the page to reveal HTML
are not an acceptable final interaction. This is a proposed replacement of
the overlay mechanism, not a claim that the shipped mechanism changed.

## Gate 1: native composition

`desktop-shell/examples/live_overlay_lab.rs` is an isolated Windows/Tauri
experiment. It creates a live page, a toolbar, and a transparent native HTML
overlay sibling. It has its own temporary WebView2 profile, no resident,
tray, daemon connection or single-instance plugin. A fixed lab-only CDP
port (19473) permits automation; close the lab after testing. Do not open
real sites or credentials in the lab.

The lab title signal is fixture plumbing, not a proposed production IPC.
The toolbar alone handles those signals; arbitrary site content must never
be granted overlay actions. The fixture uses native controls to test input
and composition; it is not the final product design. Product components
will retain existing React primitives and shared tokens (Cursor's deference
and keyboard-first bars in `docs/benchmarks.md`).

Build from `desktop-shell/`:

```sh
./examples/overlay-lab/build.sh
```

The example needs its own Common Controls v6 manifest: tauri-build embeds
that dependency for the production binary, not for an additional example.
The build script compiles the manifest with `llvm-rc` and links the resource
directly, avoiding a dependency on Windows `mt.exe` in WSL. A lab without it failed to start with Windows status `0xc0000139`
(`comctl32.dll!TaskDialogIndirect`).

Run only the example executable, in a separate Windows process. Do not
restart or replace the installed shell. Native screenshots must capture the
composed window: a CDP screenshot of one WebView does not prove stacking.

| Conditions | Required action / evidence |
|---|---|
| Suggestions open, toolbar input focused | Page animation remains visible; typing stays in the address field |
| Suggestion selected / Escape | Close overlay; navigate / restore focus respectively |
| Menu and submenu open | Both are visible and usable above the page |
| Modal open | Page remains visible through dimming; focus and clicks stay in modal |
| Nonmodal, click outside | Dismiss without losing the intended page interaction |
| Resize / maximize / DPI change | Overlay follows anchor and stays inside window |
| Tab switch / close | Close owned overlays; no orphan native surface |
| Overlay creation or update fails | Keep live page, restore focus, show actionable error in host |

Do not advance to product migration unless the native composition gate
passes. If sibling WebViews fail composition, record the evidence and
revisit hosting; do not silently switch to captures or another runtime.

## After the gate

1. Record the approved composition and trusted overlay bridge in an ADR.
2. Build per-window lifecycle, stacking, focus and geometry management.
   Exchange typed component data and action IDs, with owner and revision
   checks; give no overlay capabilities to external pages.
3. Migrate suggestions/menu first, then all host overlays crossing native
   pages. Keep the normal browser HTML path. Remove overlay-only capture
   and parking, retaining explicit screenshot and annotation features.
4. Exercise the decision table and error/empty/loading states, audit native
   geometry, and review screenshots in a subagent. Validate light/dark,
   keyboard/IME and Windows scaling 100/125/150/200 percent.
5. Scoped gates, Windows build, closing docs, fast-forward and full CI.
   Deployment remains owner-controlled.

## Product implementation refinement

The native composition proof passed for transparency, typing and resize at
150 percent Windows scaling. The implementation keeps the existing React
document as the native overlay surface instead of cloning each component
into another WebView. A geometry-only bridge is smaller and preserves the
actual handlers, forms and Radix state. ADR-0161 records this refinement.

The `overlay_product_qa` Cargo example (feature `overlay-qa`) runs the real
product shell code against the scratch daemon, without resident duties.
The ordinary `picode-shell` build has no QA entry. Run
`desktop-shell/examples/overlay-lab/build-product.sh <scratch-port>` to build
with a temporary capability limited to that origin and restore production
ACL schemas afterward. Set `PICODE_CDP_PORT` only for that QA process if
automation is needed.

| Product conditions | Action | Evidence required |
|---|---|---|
| No native pages | Full chrome region, ordinary backgrounds | JS collection test |
| Active page, no overlay | Native page hole; page gets input | JS + native screenshot/input |
| Nonmodal intersects page | Add only overlay region; page stays live | JS + native menu/input |
| Modal backdrop | Add full backdrop region; page remains visible below | JS + native modal |
| Hidden or offscreen tab/layer | No region or background changes from it | JS collection test |
| Window/DPI changes | Recompute physical region with outward rounding/clamping | Rust table + native resize |
| Invalid/oversized regions or wrong caller | Reject without changing native region | Rust validation + IPC rejection |
| Native IPC failure | Visible retry action; never start a capture | Browser fault injection |
| Existing page shown again | Raise chrome above it | Native tab cycle |
| Old shell/no shell | No new bridge registration | JS compatibility test |

The prototype's native input helpers were corrected to verify the exact
foreground HWND before every click/text operation. Earlier unguarded
attempts could reach a different foreground window; do not reuse those
helpers or infer acceptance from those attempts.

## Product evidence (2026-09-20)

The release QA example must use the normal release/LTO configuration.
Overriding only the example with `-C lto=off` produced a Windows access
violation in Tokio runtime startup; the normal release build opened and
passed the recorded composition checks. This is a QA build constraint,
not a change to the product compiler profile.

Native PrintWindow captures of the scratch product passed visual review
for empty, suggestions, menu, dialog and reconnecting states. Shadow
regions include CSS blur extents. The live fixture advanced 168 frames in
700 ms under suggestions and a modal, and 169 under the menu. Targeted CDP
typing reached the dialog field. Native geometry rejected reversed bounds
and excessive regions; the latter displayed Retry and recovered. An
untrusted page could not invoke the bridge. Tab switch/return preserved
live composition. Physical product input and the remaining OS matrix are
explicitly tracked in `docs/handoff/open/live-desktop-overlays.md`.
