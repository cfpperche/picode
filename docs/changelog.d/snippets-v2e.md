### Added

- **The snippet editor tells you when an address is already taken** — while you
  type, not after the save: the line under the **Slug** field says which
  snippet holds `/snip:…`, and Save stays off until you pick another one. On an
  existing snippet its own address is never reported as a clash.

### Changed

- **Half-written snippets no longer die with the tab.** Drafts moved from the
  per-tab shelf to the browser's durable one, so closing the tab or reloading
  brings the text back (a capture or an import is still handed over as a new
  snippet rather than announced as "unsaved changes").
