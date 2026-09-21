# Packages and manifests

## Next

- Package config descriptors (ADR-0119; C0–C5 shipped). Backlog (owner):
  upstream `picode.config` proposal. Plan: `docs/plans/package-config-manifest.md`.
- **Omp's marketplace availability list is unverified**: `omp plugin marketplace
  list --json` printed prose for the empty case (2026-09-20), so the pane
  manages marketplace *sources* for Omp and offers no catalog tab. Measure a
  configured marketplace before promoting `available` in its declaration. Muse
  is the precedent for how this goes wrong: its catalog *was* measurable, and
  only the TUI showed it.

## Debts

- [ ] **No Update action for a guest CLI's plugins.** ADR-0167 ships install,
  remove and enable/disable, and leaves `update` to the vendor's own command:
  an honest "update available" badge needs the vendor's catalog compared per
  CLI (Hermes' curated catalog, Muse's marketplace snapshot, Codex's remote
  marketplace), and PiCode has that machinery only for Pi's packages. The row
  menu names the command (`hermes plugins update`, `muse plugins update`,
  `claude plugin update`, `omp plugin upgrade`, `grok plugin update`).
  Whoever builds the availability signal promotes the action in
  `internal/clipkgs/specs.go` — the argv builders already exist for Claude
  Code, Grok, Hermes, Muse and Omp.
- [ ] **A consent refusal shows the vendor's words but no copy button.** Plan
  §6 said "refusal verbatim + copyable command"; v1 ships the verbatim half.
  Grok is where it bites: `grok plugin install <git URL|local dir>` refuses
  without `--trust`, which PiCode never passes (ADR-0167), so the row carries
  the vendor's refusal and the user retypes the command — an Install button
  that cannot succeed from the pane. The command is already known server-side
  (`clipkgs.Argv`); returning it beside the refusal, and saying the answer has
  to be given in a terminal, is the work. Claude's marketplace-declared
  command (`--accept-command`, a sha256 only a person should confirm) is the
  second case.
- [ ] **Reads and the OpenCode splice disagree about a `plugin` string.**
  `opencodeModules` reads `"plugin": "pkg"` as one module (the vendor's schema
  says array), while `removeArrayElement` refuses to splice it (409). Honest,
  but a user who hand-wrote the string form cannot remove it from the pane.
- [ ] **Muse Code's built-in first-party plugins are outside every surface
  PiCode may read.** Its TUI `/plugins` panel lists one (`TBH Reminders`,
  `built-in`/`product`) that no CLI command reports — not `plugins list
  --json`, not `--available`, not the store's own `installed.json`; the
  bundles live under `~/.local/share/muse/plugins/cache/builtin/`. PiCode
  therefore manages what the vendor's store manages and shows nothing it
  cannot change. Promoting these would mean reading the app's internal bundle
  and offering rows that cannot be removed — a vendor-side change, not a
  PiCode one.

- [x] **Four vendors' non-empty plugin-list JSON was read tolerantly.** Paid
  2026-09-21: every guest is now measured non-empty against the real binary —
  Grok, Omp, Hermes, Claude Code, Codex, Antigravity, OpenCode and Muse — with
  fixtures in `internal/clipkgs/testdata/` and the gated live harness
  (`PICODE_PKGS_LIVE=1`). The tolerant reader stays only as a fallback that
  refuses a shape it does not recognize.
- [x] **Muse Code's plugin roster was read tolerantly.** Paid 2026-09-21
  (`parseMuse`/`museRecordRow`, fixture `testdata/muse.list.json`, live harness
  installs a bundle and reads the row back). The earlier "its build refuses
  every plugin verb" reading was the vendor's per-machine feature gate measured
  with a fresh HOME — the harness now seeds that config from the machine's own
  cache.
