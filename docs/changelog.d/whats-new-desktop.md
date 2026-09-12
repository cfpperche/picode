### Changed
- **What's new is a two-column board on a desktop.** The release dialog takes
  880 px instead of 400 and lays the highlights out in two columns, so a whole
  release fits one screen: the nine highlights of v0.2.0 are read without
  scrolling, and a summary stops wrapping at ~40 characters. The dialog also
  centres itself at any window height — a short window scrolls inside the
  dialog instead of pushing it against the top of the screen.

### Fixed
- The release dialog was 400 px wide, not the 560 px it asked for: `.dlg` is
  declared later in the stylesheet at the same specificity, so its `width` —
  and its `padding`, which kept the header and footer hairlines inset from the
  dialog's edge — won silently.
- The matrix highlight of v0.2.0 wore a generic sparkle: the note's icon name
  had no glyph in either shell. Desktop draws it with the Canvas glyph, the
  phone with the panel grid, and `web/tools/release-note-icons.test.mjs` now
  fails when a published icon name is missing.
- A build with no release notes showed the changelog link twice — once as the
  empty state's action and again in the footer.
