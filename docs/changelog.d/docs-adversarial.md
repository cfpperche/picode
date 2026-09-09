### Added
- **`picode version` (`--version`, `-v`).** Prints the build identity and exits. The install verifier already ran `picode --version`; until now the flag silently started a server instead.

### Fixed
- **The HTTP docs pair shells the way the server actually pairs.** `docs/api` now shows `picode pair` (the old example called an endpoint that never existed) and no longer claims `PICODE_INSECURE=1` skips pairing — that is the **Who must pair: Off** setting (`PICODE_AUTH_MODE=off`).
- **The architecture index renders as one table again** on GitHub and the docs site (a stray blank line had broken it at ADR-0090).
- **Stale file path in the routes doc** (`web/src/lib/fileDocument.js` → `web/desktop/…`, mirrored in `web/mobile`).
