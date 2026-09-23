### Fixed
- The desktop Management window measures the pnpm store, the Go caches, npm and uv where each tool keeps them. Before, a pnpm store outside `~/.cache/pnpm` showed as a few KB and cleaning it never changed the number.
- Saving WSL settings in the Config tab no longer leaves two copies of a setting that appeared under both `[wsl2]` and `[experimental]`. It also no longer shows a value from a section WSL does not read.
- **Give back held space** no longer lets Windows restart the distro during the conversion.
