# Native CLIs: packages, providers, settings

## Next

- Physical iPhone/PWA/IME acceptance for the native panes is external; mobile package configuration is desktop-only by design (`docs/plans/cli-native-packages.md`).

## Debts

- Native packages/providers/settings: real downloads, vendor OAuth, credential changes, device acceptance and a real process restart remain external.
- Agent CLIs is not in `SURFACE_PROFILES`, so no docs-shots capture covers it.
- A launch that fails inside tmux (session start refused after the row exists) is the one branch of createCLITerminal that publishes terminal.changed without a covering test: the pre-flight keeps rejecting the reachable failures (missing executable, missing folder) before the row is created, and there is no seam to force a tmux failure.
- Muse Code and Antigravity are list-only session sources: no `Reader` (their transcripts — Muse's event JSONL, Antigravity's `steps` table plus `brain/<id>/**/transcript/*.jsonl` — are not mapped to the portable timeline), no `Writer`/`Prompter`, so neither can be a handoff source or target yet.
