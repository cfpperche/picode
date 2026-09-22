# Native CLIs: packages, providers, settings

## Next

- Memory table slice 4 — the cross-project view for Claude Code's 37 stores,
  and with it TanStack Table v9 plus virtualization. The only slice that adds
  a dependency; the owner's call (`docs/plans/memory-table.md`).
- Omp Fatia 2 (sessions): the gate is one real omp session on disk (needs auth on this machine); `ResumeArgs: ["--resume", id]` per its `--help`. Plan: `docs/plans/omp-cli.md`.
- Physical iPhone/PWA/IME acceptance for the native panes is external; mobile package configuration is desktop-only by design (`docs/plans/cli-native-packages.md`).
- The Keyboard pane: redesign it for Pi, then let it edit the guest keymaps that exist (`claude-code`, `codex`, `opencode`, `agy`, `omp`; `hermes` partial; `grok` and `muse` have none). Plan: `docs/plans/keyboard-pane.md`; the four owner questions in its §9 come first.

## Debts

- **Pi persists about forty keys; the pane shows eleven rows over thirteen
  keys.** The five added on 2026-09-20 were the ones that change visible
  behaviour. `npmCommand` is an argv array and needs a list kind before it can
  be a row; the rest are TUI ergonomics nobody has asked for. Adding one is
  now a line in `piRows.js` plus its field in `internal/pisettings` — the
  resolver follows the table on its own.

- **Codex has no memory-clear command PiCode can name.** The spec carried
  `codex`, which the read-only notice offered as "the vendor's own command";
  `codex` on its own does nothing to a memory, so the field is now empty and
  the pane shows the note alone (visual review 2026-09-20). Whoever confirms
  how Codex clears its memories puts it back.
- **The bulk-delete bar does not follow the scroll.** On a 55-row table a
  selection made near the bottom leaves the Delete button thousands of pixels
  up. Sticky needs a top offset for the app's fixed header, which no token
  names today; nobody has measured whether it is worth one.
- **Deleting a memory leaves its row in `MEMORY.md`.** The confirm says so and
  the dangling row is reported on the next load, which is honest but not
  finished: whether PiCode should edit the index is ADR-0163's line about not
  writing what it does not own, and it is the owner's call
  (`docs/plans/memory-table.md`).
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

- **`scripts/qa-cli-settings.mjs` went stale in the native-settings landing, and
  two of its rows still fail on a fresh fixture.** Repaired where the Keyboard
  pane needed it (2026-09-21): `layerOf()` reads `[data-layer]` on the body
  wrapper (the marker moved off the section), `#g-compact` is
  `#g-compactionEnabled`, the "unsupported CLI" row waits on the pane instead of
  the "in development" notice no managed CLI shows any more, the audited
  captures settle bottom-anchored overlays first, and a failing run now records
  the rows that passed. Still red, both **reproduced on `main` without the
  Keyboard pane's changes** and both outside this pane: (1) the post-loop
  untrusted/trust/free-agent matrix stops at a `ready()` that waits for a
  `.settings-section` a blocked project layer does not render; (2) the mobile
  **"initial failure retries"** row leaves the pane on `Try again` — the 503
  stub is restored and the button clicked, and `#pi-settings-view .settings-section`
  never comes back (screenshot `failure.png` in the run's output dir, a11y tree
  in `failure.json`). Their owner should re-point them
  (`docs/handoff/2026-09-21-keyboard-row.md`).

  **2026-09-21 (P2a, `feat/keyboard-guests`), re-measured:** three more stale
  assertions, not two. The `settings-desc code` row still demanded
  `/keybindings\.json$/` after P0 moved the period *inside* the code box — fixed
  in the branch (accepts the trailing period again). The app's toast stack sits
  on the phone's bottom edge at capture time and covered whatever pane was drawn
  next; the guest captures now wait for it to leave (`settleToasts()`), and with
  that the mobile trust-matrix rows that used to die on "Element covered by
  `span.notice-text`" pass in the same run. Row (1) is still red and still
  reproduced: the run stopped at the same blocked-project-layer `ready()`. Row
  (2) lives in `qa-cli-settings-recovery.mjs`, not re-measured here.

- **Antigravity's key map is researched and not declared** (P3, 2026-09-21).
  Measured: `~/.gemini/antigravity-cli/keybindings.json`, 36 ids in 10
  namespaces, `id -> [chord]`, one *override* layer — the vendor documents
  per-action fallback and "delete the file to restore defaults", so removing a
  row returns to the built-in default rather than leaving a hole, and `[]`
  disables a default. No checksum or signature: a byte-preserving splice is safe,
  and the CLI never rewrites the file itself (mtime old while sessions ran).
  Two open points, both cheap: (a) **the pickup is unmeasured** — there is no
  `/reload` and the docs only say settings load at startup, but the binary has a
  `file_watcher.go`, so the honest sentence ("restart it") needs the experiment:
  remap `cli.cycle_mode` from `shift+tab` to `ctrl+n` in a live `agy` session
  (tmux + a trust prompt) and press both keys — the footer's mode chip says which
  one the session is still honouring. **Restore the file from a backup
  afterwards** (the first attempt at this left the user's file rewritten and had
  to be restored byte-for-byte); (b) **the labels**: the vendor publishes two
  documentation generations — `/docs/cli/using` and `/docs/cli/vim-editor-mode`
  match the installed build row-for-row, `/docs/cli/reference` has drifted to
  renamed ids (`prompt.*`) — so the catalog comes from the first two, or the rows
  are labelled from their ids.
