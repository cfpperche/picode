### Changed
- **Surface split (routes)**: the responsive web app previously served at
  `/desktop/` now lives at `/browser/`, and `/desktop/` serves the new
  bundle composed for the Windows shell only (its entry wraps the app with
  the shell chrome; the browser never loads it). The Management page moved
  into that bundle — its design tokens are imported from the shared
  package, ending the hand-copied token block. The launcher and the docs
  screenshot machinery follow the new routes. ADR-0122.
