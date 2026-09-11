### Changed
- **Providers: one row per account, columns that line up.** The roster is a
  dense table now — provider, account, identity, usage, 7-day spend and the
  actions sit in fixed columns instead of being pushed to the two edges of
  the card. The quota reading moved into its own column, so two accounts'
  bars can be compared without reading a label, and a long vendor reason
  ("Rate limited.") stays on one line instead of wrapping under the buttons.
  Ten providers plus the llama.cpp pointer and Recently used now fit one
  1000 px-tall window without scrolling.
- The search field and Add provider are one cluster instead of two distant
  corners, and a second account of the same provider repeats the provider's
  name, dimmed and indented, so a group reads on its own.
- Recently used is a line of chips (provider, Sign in, remove) instead of a
  full-height list of rows.

### Fixed
- Column headings are only drawn when there are rows, so an empty search no
  longer leaves a header over nothing, and the empty roster shows one action
  instead of two identical Add provider buttons.
- With no providers connected the page no longer renders an empty toolbar
  row.
