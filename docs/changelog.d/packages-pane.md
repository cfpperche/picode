### Changed

- **One packages pane, every CLI.** Which pane opened used to depend on the
  CLI's name: Pi got the rich one, the other eight a simpler one. Now a single
  pane asks the CLI's own engine what it can do and draws that — the gallery
  where the CLI's catalog is PiCode's, the vendor's roster elsewhere, and each
  control (install, remove, toggle, update, inspect, the config editor, the
  agent's "only this agent's packages" switch) only where the CLI has the
  mechanism behind it. The scope radios appear when the layer is one the read
  can honour, and the copy each surface spoke before is kept word for word
  ("Install to" for the gallery, "Plugins go to" for a vendor). Links that
  pointed at either old pane keep working.
