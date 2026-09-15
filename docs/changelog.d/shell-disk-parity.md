### Added
- **The shell tray reaches disk parity (ADR-0142, slice 2).** The tray now
  shows the disk line (`WSL … · ≈… held · C: … free`, with the `low`
  warning), a Give-back item with the readiness interlock, an explicit
  stop-the-distro confirmation and a measured result, plus Restart PiCode
  and View logs — every duty the Go tray menu owned. One board composes
  status, disk and tooltip so no timer erases another's write; the
  keepalive re-arms itself after a compact.
