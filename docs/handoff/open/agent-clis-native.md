# Native CLIs: packages, providers, settings

## Next

- Omp Fatia 2 (sessions): the gate is one real omp session on disk (needs auth on this machine); `ResumeArgs: ["--resume", id]` per its `--help`. Plan: `docs/plans/omp-cli.md`.
- Physical iPhone/PWA/IME acceptance for the native panes is external; mobile package configuration is desktop-only by design (`docs/plans/cli-native-packages.md`).

## Debts

- **A boolean still looks different in Pi's pane and the guests'.** Pi uses
  the Radix switch its Auto-compact row always used; the guest pane uses a
  checkbox because it needs a third state — an unset key must not read as
  "Off" when the CLI's own default is on, and a switch cannot show that. One
  of the two has to give: either the guests gain a tri-state control, or the
  unset state moves entirely into the source line and both become switches.
- **`resolveLayer` is a named list.** A key added to `/api/pi-settings` is
  invisible in the pane until it is added there too — that is how the Theme
  row first rendered empty while the file said `dark` (2026-09-20). Pi's pane
  is declarative in its rows now but not in its resolver.
- **Pi persists about forty keys; the pane shows thirteen.** The five added on
  2026-09-20 were the ones that change visible behaviour. `npmCommand` is an
  argv array and needs list support before it can be a row; the rest are TUI
  ergonomics nobody has asked for.

- **The Memory pane has no toggle and the 409 has no Replace control.** The
  docs now describe both honestly — the pane links to Settings, and `force`
  exists in the API with nothing sending it. Whoever wants the controls ships
  them; until then the sentences stay as they are (ADR-0163, adversarial
  review 2026-09-20).
- **Two of the six memory stores have no config key at all.** Grok's switch is
  `--experimental-memory` / `GROK_MEMORY=1` and Muse's is a runtime capability,
  so their panes cannot link anywhere. Wiring a launch flag into the pane is a
  different mechanism and needs its own decision.

- **Omp's memory folder is probed, not documented.** The local backend's path
  is not in the vendor docs, so `climemory` looks in `<ws>/.omp/memories` and
  `~/.omp/agent/memories` and otherwise reports memory as off. If Omp writes
  somewhere else, the pane says off while memory is on. Fix by reading the
  path out of a real `memory.backend: local` run (ADR-0163).
- **Antigravity's memory tier is `unknown`.** No vendor documentation confirms
  a memory the CLI manages; the transcripts under `~/.gemini/…/brain/` are
  session history. Whoever gets a confirmation promotes the tier.
- **Guest CLIs have machine and workspace layers only** — no per-agent
  settings layer, the same phase-1 limit connectors carry (ADR-0150).
  The Keyboard pane stays Pi-only: no guest CLI exposes a key map PiCode can
  write.
- **The mobile panes have no physical-device acceptance.** Both mount and
  render at a real 390x844 viewport with no horizontal overflow, and the
  editable memory box, its actions and the Codex Memory group were captured
  there. A real iPhone, the PWA and IME behaviour remain external.

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
