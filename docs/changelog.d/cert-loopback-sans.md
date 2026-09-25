### Fixed

- **The HTTPS certificate now covers `127.0.0.1` and `::1`, not only
  `localhost`.** A tool or script that reached PiCode by the loopback address
  got a certificate error. New certificates include both. Running
  `picode provision` reissues an existing one that lacks them, from the same
  local certificate authority, so browsers keep trusting it. Every name the
  old certificate had, such as a Tailscale name or a LAN address, is kept.
