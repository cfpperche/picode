### Fixed
- **Menus and other layers no longer come up hidden under the work browser.**
  The editor's tab menus, the command palette, dialogs, dropdowns and toasts
  now decide by geometry: any floating layer that reaches the page parks the
  native view and freezes the page behind it. Previously only dialogs did
  (and only the ones wearing the app's own dialog class), so the palette and
  every chrome menu could appear behind the page.
