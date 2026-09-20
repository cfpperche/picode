### Added
- **`*` in a browser grant means any site.** The domains field now takes the
  one token for "this agent may open anything" (still http and https only),
  and everything else keeps working as before: exact hosts, `*.example.com`
  for a domain and its subdomains, ports ignored.

### Fixed
- **The domains field says what it means.** Each entry is described as you
  type it — including the shapes that open nothing (`*example.com`, `*.`, a
  lone `.`), which used to be accepted and saved in silence.
