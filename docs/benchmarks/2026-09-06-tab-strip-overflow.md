# Study: the editor tab strip when tabs overflow (VS Code, Zed, JetBrains, Firefox, Chrome, UI kits)

- **Date:** 2026-09-06
- **Sources:** public source code and docs, cited per fact below. Nothing
  was cloned; inferences are marked *inf.* The in-house counterpart is
  `web/desktop/src/components/AgentTabs.jsx` with `.tab-strip` in
  `web/desktop/src/styles/app.css`.
- **Scope:** what a horizontally overflowing tab strip does about the
  scrollbar, off-screen tabs, wheel, keyboard and the active tab; what
  PiCode adopts; what it refuses.

## Why now

The owner's screenshot (Windows Chrome, 2026-09-06) shows the strip with a
classic thin scrollbar and arrow buttons eating the bottom of the 40 px
tab bar, and the active tab clipped at the right edge. Measured in headless
Chromium 152 at 1280 px with seven terminal tabs:

| Fact | Value |
|---|---|
| `.tab-strip` offsetHeight / clientHeight | 39 / 29 px — the scrollbar takes 10 px |
| `scrollbar-width` computed on the strip | `thin` (inherited from `* { scrollbar-width: thin }`) |
| Active tab left edge vs strip clientWidth | 1151 px vs 995 px — off screen, `scrollLeft` 0 |
| Vertical wheel over the strip | `scrollLeft` stays 0 |
| Gutter with `scrollbar-width: auto` (the `::-webkit-scrollbar` 8 px path) | 15 px in headless; a classic (layout) bar, not an overlay |
| Gutter with `scrollbar-width: none` | 0 px |

Root cause of the visual: Chromium ≥ 121 ignores `::-webkit-scrollbar*`
on any element whose `scrollbar-width` is not `auto`, so the app's "8 px
overlay" rules never apply to the strip and Windows paints its standard
thin bar with arrows (`agent-clis.css` and `.dlg-sheet` already carry
per-view workarounds for the same collision). Neither path is an overlay;
both steal height from the 40 px chrome.

## What each one decided

| | Scrollbar | Off-screen cue | Wheel | Active tab | Keys / list |
|---|---|---|---|---|---|
| VS Code | 3 px overlay, shown on hover, fades after 500 ms (`ScrollableElement`, `useShadows:false`) | none besides the bar; `…` "Show Opened Editors" | vertical → horizontal (`scrollYToX`); Shift too | scrolled into view on every layout | Ctrl+PgUp/PgDn; Ctrl+Tab MRU |
| Zed | none rendered (`overflow_x_scroll`, no Scrollbar element) | none | trackpad only (*inf.*) | `scroll_to_active_item`, suppressed after a manual scroll | multi-row refused by maintainers |
| JetBrains | native thin bar on hover | `▾` dropdown of hidden tabs at the right end | scrolls | kept in view | Alt+←/→; tab limit 10 |
| Sublime 4 | none | `‹ ›` scroll buttons + `▾` list | wheel *switches* tabs | kept in view | — |
| Firefox | none (`arrowscrollbox`) | `‹ ›` arrows only when overflowing, disabled at the ends; 7 px radial shadow at the un-scrolled edge; `▾` "List all tabs" | dominant axis, `deltaMode`-scaled, `preventDefault` | `scrollIntoView({block:"nearest"})` on select | pinned tabs excluded from scrolling |
| Chrome | shrink-only, no scroll (flag removed in 144 for maintenance reasons, *inf.* not a UX verdict) | Tab Search, vertical tabs | switches tabs on Linux only | n/a | Ctrl+Shift+A |
| Material UI | native bar hidden (`scrollbar-width:none` + negative margin) | `‹ ›` buttons when not all visible (`scrollButtons:'auto'`) | none | `scrollSelectedIntoView`, animated | roving tabindex, Arrow/Home/End |
| Ant Design | transform-based, no bar | `-ping-left/right` edge classes + "more" dropdown | dominant axis, `preventDefault` only if moved | `scrollToTab` | Arrow keys |
| Radix / Mantine ScrollArea | overlay that "takes up no space", type `hover`, hide delay 600 / 1000 ms | Mantine `Scroller`: chevrons on overflow | Shift+wheel | — | — |
| NN/g | "If people see a scrollbar, they assume there's additional content" | arrows "frequently remain unnoticed" when hover-only; keep them visible | — | — | — |

Receipts: VS Code [multiEditorTabsControl.ts](https://github.com/microsoft/vscode/blob/main/src/vs/workbench/browser/parts/editor/multiEditorTabsControl.ts), [scrollableElementOptions.ts](https://github.com/microsoft/vscode/blob/main/src/vs/base/browser/ui/scrollbar/scrollableElementOptions.ts), [scrollbars.css](https://github.com/microsoft/vscode/blob/main/src/vs/base/browser/ui/scrollbar/media/scrollbars.css), spurious-click complaints [#111817](https://github.com/microsoft/vscode/issues/111817) / [#233864](https://github.com/microsoft/vscode/issues/233864); Zed [tab_bar.rs](https://github.com/zed-industries/zed/blob/main/crates/ui/src/components/tab_bar.rs), [PR #36827](https://github.com/zed-industries/zed/pull/36827); JetBrains [Editor Tabs settings](https://www.jetbrains.com/help/idea/settings-editor-tabs.html); Sublime [#5487](https://github.com/sublimehq/sublime_text/issues/5487); Firefox [arrowscrollbox.js](https://searchfox.org/mozilla-central/source/toolkit/content/widgets/arrowscrollbox.js), [tabs.css](https://hg-edge.mozilla.org/mozilla-central/raw-file/tip/browser/themes/shared/tabbrowser/tabs.css); Chrome [tab_style.cc](https://chromium.googlesource.com/chromium/src/+/refs/heads/main/chrome/browser/ui/tabs/tab_style.cc), [issue 478527840](https://issues.chromium.org/issues/478527840); MUI [Tabs.js](https://github.com/mui/material-ui/blob/master/packages/mui-material/src/Tabs/Tabs.js); Ant [useTouchMove.ts](https://github.com/react-component/tabs/blob/master/src/hooks/useTouchMove.ts); Radix [scroll-area.tsx](https://github.com/radix-ui/primitives/blob/main/packages/react/scroll-area/src/scroll-area.tsx); Mantine [Scroller](https://mantine.dev/core/scroller/); NN/g [horizontal scrolling](https://www.nngroup.com/articles/horizontal-scrolling/), [scrollbars](https://www.nngroup.com/articles/scrolling-and-scrollbars/); MDN [scrollbar-width](https://developer.mozilla.org/en-US/docs/Web/CSS/scrollbar-width) (warning: `none` needs another scrolling mechanism), [wheel event](https://developer.mozilla.org/en-US/docs/Web/API/Element/wheel_event).

## What PiCode adopts

1. **No layout scrollbar on the strip.** `scrollbar-width: none` (plus the
   webkit fallback), so tabs get the full 40 px back (Zed, Firefox, MUI).
2. **Active tab always revealed** on selection and on open, moving the
   strip as little as possible (Firefox `nearest`, VS Code, MUI):
   `lib/tabStrip.js` computes the offset, `scroll-behavior: smooth` unless
   reduced motion, an instant jump on the first paint. Shipped in phase 1
   together with item 1.
3. **Vertical wheel scrolls the strip** with the dominant-axis rule, only
   when the strip overflows, `preventDefault` only when it moved, no
   remap when `deltaX` is already present or `ctrlKey` (pinch) is set;
   `overscroll-behavior-x: contain` (Firefox, VS Code, Ant). Successive
   ticks accumulate on a pending target so a smooth scroll in flight
   does not swallow the distance asked for; the target is forgotten once
   `scrollLeft` has been still for three frames — exact in every engine,
   where `scrollend` is Chromium/Firefox only. Shipped in phase 2.
4. **Edge cue + arrows while overflowing.** A fade mask at whichever edge
   still has content and `‹ ›` buttons at the strip ends, always visible
   while overflowing and disabled at the ends (Firefox, MUI, NN/g).
   One `ResizeObserver` (strip + tabs) and a scroll listener feed
   `data-at-start` / `data-at-end` on the strip; CSS does the rest
   (`lib/useTabStrip.js`). Shipped in phase 2 with item 3.
5. **Thin overlay position indicator** (3 px, `pointer-events: none`,
   appears on hover/scroll, fades after 500 ms) so the NN/g "scrollbar
   means more content" signal survives without VS Code's spurious-click
   problem. Shipped in phase 2 at the bottom edge; moved to the top edge in
   the debts pass because the bottom belongs to the active tab's accent
   underline (VS Code lives with the overlap; its active tab is marked at
   the top).
6. **"All tabs" list** (a list button after the right arrow, shown only
   while overflowing) listing every tab with its face, name and status
   dot, the hidden ones first under "Out of view" (JetBrains, Sublime,
   Firefox, VS Code). Radix DropdownMenu, `hiddenTabs` in `lib/tabStrip.js`.
   Shipped in phase 3.
7. **Keyboard:** `Alt+[` / `Alt+]` for previous / next tab (owner decision;
   Ctrl+Tab, Ctrl+PgUp/PgDn and Ctrl+W are browser-reserved), rebindable
   in the app-keys catalog, plus `role=tablist` / `role=tab` with
   **manual activation** — arrows / Home / End move focus, Enter or Space
   selects (MUI default, Radix `manual`). Automatic activation was tried
   and dropped: selecting a terminal tab hands focus to its xterm textarea,
   so the next arrow would reach the shell. Terminals return every Global
   app chord to the app (`wireTermKeys` passthrough) — before, `Ctrl+K`
   opened the palette *and* sent `\x0b`. Shipped in phase 3. The debts
   pass added closing from the keyboard: `Alt+W` (catalog, `app.tab.close`)
   and Delete / Backspace on a focused tab, focus moving to the neighbour
   (ARIA tabs-with-removal).
8. **Label cap:** `.mtab-label { max-width: 200px }` with ellipsis so one
   long name cannot swallow the strip (Chrome 232 dip, VS Code fixed max
   160); the tooltip carries "name — CLI or path". `.mtab { flex: none }`
   keeps tabs from shrinking once the label may overflow — the strip
   scrolls, tabs never collapse toward their favicon. Shipped in phase 4.

## Where PiCode improves on them

| Benchmark decision | PiCode |
|---|---|
| VS Code: 3 px hover-only bar is the only cue and is draggable | Indicator is non-interactive; the arrows and the list carry the clicks |
| Firefox: arrows appear with a jump when overflow starts | Arrows and fades reserve no width when hidden (`.main-tabs-end` already exists at the right; a matching left slot only mounts on overflow) |
| Zed: no cue at all | Fade + arrows + list |
| Everyone: tab status is invisible once scrolled away | The "All tabs" list keeps the needs-you / working dots, and a needs-you tab that is off screen surfaces the dot on the arrow at its side (verified on the docs fixture with a real `needs-you` terminal state) |

## What PiCode refuses (v1)

Tab shrinking to favicons (Chrome/Safari), multi-row wrap (Zed refused it;
VS Code default off), wheel-switches-tabs (Sublime; VS Code default off),
a tab limit (VS Code default off), pinned sticky tabs, hold-to-scroll
arrows, touch inertia beyond native scrolling, mobile (no strip there).

## Ritual

This study backs the tab-strip change; the implementation cites it in the
changelog entry. The global `* { scrollbar-width: thin }` collision with
`::-webkit-scrollbar` that the first measurement exposed was fixed in the
debts pass: the standard properties now live under
`@supports not selector(::-webkit-scrollbar)` in both apps.

## In-page tab bars (2026-09-24)

The same chrome now covers the tab bars inside pages, which used to show a
native horizontal scrollbar when they outgrew a narrow pane:
`web/browser/src/components/OverflowTabs.jsx` wraps the CLI sections
(`CliPaneTabs`), the CLIs / Messages / Settings bar (`CliTabs`) and an
app's page tabs (`AppSurface` `PageTabs`). It reuses `useTabStrip` and
`revealLeft`: no scrollbar, fades on the side that still has tabs, arrows,
and a list (out-of-view tabs first) that follows a tab's `href` or calls
the page's own navigation. Two differences from the editor strip: the
vertical wheel stays with the page (`useTabStrip(…, { wheel: false })`),
because these bars sit inside a page that scrolls vertically, and there
is no position indicator — on a card's top edge it read as a stray
scrollbar thumb. Bars that wrap instead (Preferences, llama) and the phone app
(swipe, ADR-0072 keeps it separate) are unchanged.
