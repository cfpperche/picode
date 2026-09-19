# Native CLIs: packages, providers, settings

## Next

- Omp Fatia 2 (sessions): the gate is one real omp session on disk (needs auth on this machine); `ResumeArgs: ["--resume", id]` per its `--help`. Plan: `docs/plans/omp-cli.md`.
- Physical iPhone/PWA/IME acceptance for the native panes is external; mobile package configuration is desktop-only by design (`docs/plans/cli-native-packages.md`).

## Debts

- **A flake in the gate, seen once (2026-09-19).** `TestCLIAdapterPreviewMatchesExecution`
  failed in a parallel `make ci` shard with `TempDir RemoveAll cleanup: unlinkat
  …/data/native-observations: directory not empty` — a writer still touching the
  temp data dir while the test's cleanup removes it. It passes 3/3 alone and the
  next full run was green, so the cost is a red gate once in a while, not a wrong
  answer. Whoever owns the CLI inspection test owns the fix (wait for the writer,
  or give the observation dir its own lifetime).
- Native packages/providers/settings: real downloads, vendor OAuth, credential changes, device acceptance and a real process restart remain external.
- Agent CLIs is not in `SURFACE_PROFILES`, so no docs-shots capture covers it.
- A launch that failed inside tmux (session start refused after the row exists) keeps a covering test since Fatia 3a: the pre-flight still rejects the reachable failures first, and the dead-socket attempt persists without leaking a session.
- Muse Code and Antigravity are full session citizens since Fatias 1–4 (Reader via `muse export` / brain `transcript.jsonl`, native Writer + Prompter both ways, handoff source and target). Remaining asymmetry: Antigravity reports Working/Ready through its title reporter (needs-you never — no approval signal); Muse Code stays honestly Open (R3233 hooks exist but scrub the hook env to PATH, so no per-terminal attribution; Fatia 6).
