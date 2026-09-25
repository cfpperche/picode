### Changed

- **The HTTPS certificate no longer lists Docker's internal addresses.**
  Issuing or renewing it now skips Docker's virtual network bridges and
  WSL's internal loopback address. It keeps your LAN, Tailscale and
  loopback addresses. A renewal keeps every name the old certificate had,
  but recomputes its addresses from the network as it is now.
