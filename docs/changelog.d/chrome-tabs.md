### Changed

- Web: the editor tab strip (browser and desktop) now follows Chrome's CR23
  tab language — raised active tab, hover pill, idle separators — instead of
  the rectangular accent-underline tabs. Overflow, keys and the all-tabs list
  are unchanged.
- Header matches Chrome's tab strip: 41px tall with 35px tabs inset 6px
  from the top, 20px idle separators, 8px corner radius
  (`docs/benchmarks/2026-09-15-chrome-tab-header.md`).
- The selected tab now reads connected to the page on `/browser` and
  `/desktop`: it shares the toolbar background and no hairline cuts it
  off, while the sidebar head, tab strip and inspector head read as one
  header bar with a single hairline between them and the content and no
  pane verticals crossing it.
