# Native CLIs: packages, providers, settings

## Next

- Physical iPhone/PWA/IME acceptance for the native panes is external; mobile package configuration is desktop-only by design (`docs/plans/cli-native-packages.md`).

## Debts

- Native packages/providers/settings: real downloads, vendor OAuth, credential changes, device acceptance and a real process restart remain external.
- Agent CLIs is not in `SURFACE_PROFILES`, so no docs-shots capture covers it.
- A launch that failed inside tmux (session start refused after the row exists) keeps a covering test since Fatia 3a: the pre-flight still rejects the reachable failures first, and the dead-socket attempt persists without leaking a session.
- Muse Code and Antigravity are full session citizens since Fatias 1–4 (Reader via `muse export` / brain `transcript.jsonl`, native Writer + Prompter both ways, handoff source and target). Remaining asymmetry: Antigravity reports Working/Ready through its title reporter (needs-you never — no approval signal); Muse Code stays honestly Open (R3233 hooks exist but scrub the hook env to PATH, so no per-terminal attribution; Fatia 6).
