### Fixed
- **Resetting a site's permission now really forgets it.** The shell kept
  every decision in the engine's own per-origin memory (`SetPermissionState`)
  as well as in its map, so "Reset" in Site settings left the site working
  until the app restarted. A policy write that names a site now tells the
  engine too, and "Always allow" from the Ask bar writes the site's standing
  on both sides.
