### Fixed

- **Providers**: when a CLI renews its tokens by itself, PiCode **harvests the renewal back into the saved row** on the next look — Verify, Usage and the in-use match keep working without a re-import. Matched by the stable refresh token, never guessed; the CLI's own file is only ever read.
