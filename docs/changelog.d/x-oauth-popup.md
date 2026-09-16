### Fixed
- **Sign-in popups open as real windows.** “Continue with Google” (and the
  other OAuth providers that open a sized popup) now completes instead of
  dead-ending: `window.open` with a size keeps its link to the page that
  opened it, which is how the credential comes back. Plain `target=_blank`
  and unsized `window.open` still open as editor tabs.
