# Study: Chrome's tab strip height and the connected active tab

- **Date:** 2026-09-15 (owner request: our header must match Chrome's height
  and our selected tab must read connected to the page, like Chrome's).
- **Sources:** Chromium `chrome/browser/ui/layout_constants.cc` (current
  main, read from raw.githubusercontent.com — every number below is a
  `GetLayoutConstant` case), the in-repo tab-strip study
  (`2026-09-06-tab-strip-overflow.md`), and a CSS inverted-corners survey
  (cssscript.com/inverted-corner-tabs — technique reference only).
- **Scope:** the header row height and the active tab's connection to the
  surface below on `/browser` and on `/desktop` (shell merged row,
  ADR-0122). Overflow, keyboard and the "All tabs" list are unchanged.

## What Chrome decided (desktop, non-touch, dip)

| Constant | Value | Meaning |
|---|---|---|
| `kTabHeight` | 34 + `kTabstripToolbarOverlap` = **35** | tab glyph height |
| `kTabStripPadding` | **6** | top inset of tabs inside the strip |
| `kTabStripHeight` | 35 + 6 = **41** | whole header height |
| `kTabstripToolbarOverlap` | **1** | active tab overlaps the toolbar's top edge by 1px |
| `kTabSeparatorHeight` | **20** | hairline between two idle tabs |
| `kToolbarCornerRadius` | **8** | top corner radius language |

Mechanism: the active tab shares its background with the toolbar and
covers the strip's bottom edge (the 1px overlap), so no line cuts it off
from the page. Inactive tabs sit on the frame color, divided by 20px
separators. There is no full-width rule under the strip — the seam is the
frame/toolbar color step itself.

## What PiCode adopts

1. **Header height 41px** (`--chrome-h`), tabs inset 6px from the top for
   35px tabs — the exact `kTabStripHeight`/`kTabHeight` pair.
2. **Connected active tab, pure CSS.** The strip's `border-bottom`
   becomes an `::after` hairline layer; the active tab (`z-index: 3`)
   paints over it while inactive tabs (`z-index: auto`, earlier in DOM
   order) stay behind it. No JavaScript mask, no resize/scroll tracking —
   the connection survives smooth-scroll animation for free.
3. **Toolbar plane = `--bg-panel`.** The active tab matches the file,
   tree and git headers and the web tab toolbar, which sit directly
   below the strip — the way Chrome's tab matches its toolbar. An
   agent, terminal, app or dashboard tab meets a darker page (chat and
   terminal are `--bg-base`; the session bar lives inline in the
   composer, not under the strip) — the honest equivalent of Chrome's
   toolbar→page step, with no hairline cutting the tab either way.
4. **One header bar.** Sidebar head, tab strip and inspector head share
   the `--bg-base` frame plane at the same height, so their hairlines
   read as one line across the app, broken only under the active tab.
   The `/desktop` shell row owns the same `::after` line full-width.
5. **Separators 20px**, corner radius stays 8px (already Chrome's).
6. **No through-header verticals.** The sidebar/inspector edges are
   `::before` verticals starting below the 41px bar instead of
   full-height borders (a cover painted from inside the head lands 1px
   off — `overflow: hidden` clips at the padding edge — so the border
   itself moved). Shell mode (`/desktop`) keeps full-height verticals:
   its bar is the shell row itself, and the inspector head below the
   row stays on the panel plane, blending down into its rail (ADR-0122).

## Where PiCode differs

| Chrome | PiCode |
|---|---|
| 1px geometric overlap of tab over toolbar | stacking cover of the hairline layer — same pixel, no layout overlap |
| uniform toolbar under every tab | per-surface honesty: panel toolbars connect, terminal keeps its ground |
| full-width toolbar color step, no rule | hairline under inactive tabs kept, so the strip stays defined over a dark terminal |

## What PiCode refuses (v1)

Concave "ears" on the active tab (they only read over a uniform
toolbar; ours meets terminal/chat/file), tab shrinking to favicons
(2026-09-06 refusal stands), a synthetic uniform toolbar row under
terminal tabs.

## Ritual

The tab CSS cites this study; the implementation ships behind no flag.
