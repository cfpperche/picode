### Fixed
- **The Add/Edit endpoint buttons stay in reach.** A tall custom endpoint form
  scrolled its **Back** / **Add endpoint** buttons off the bottom of the dialog,
  so saving meant scrolling to find them. The action row is now pinned to the
  dialog's (or the phone sheet's) bottom edge and the form scrolls under it.
- **Verify says what it covers instead of a status it never had.** An endpoint
  whose API type PiCode cannot speak (a hand-edited `models.json`) answered
  "rejected the request (0)"; it now names the type and the four API shapes
  verification can address, and reports that nothing was spent.
