# Packages and manifests

- [ ] **Muse Code's plugin roster is still read tolerantly.** The installed
  build (Muse Code 1.3.0) exits 2 with "plugins are not available in this
  build" for every plugin verb, so its non-empty JSON could not be observed;
  `parseVendorRows` maps field names by candidate and refuses an unknown shape.
  Grok, Omp, Hermes, Claude Code, Codex, Antigravity and OpenCode were all
  measured non-empty offline (fixtures in `internal/clipkgs/testdata/`, harness
  in `live_test.go`, `PICODE_PKGS_LIVE=1`). Re-run the harness on a Muse build
  that ships plugins and pin the row shape.
- [ ] **A consent refusal shows the vendor's words but no copy button.**
  Plan §6 said "refusal verbatim + copyable command"; v1 ships the verbatim
  half. Grok is where it bites: `grok plugin install <local dir>` refuses
  without `--trust`, which PiCode never passes (ADR-0167), so the row carries
  the vendor's refusal and the user retypes the command. The command is already
  known server-side (`clipkgs.Argv`); returning it with a refusal is the work.
- [ ] **Reads and the OpenCode splice disagree about a `plugin` string.**
  `opencodeModules` reads `"plugin": "pkg"` as one module (the vendor's schema
  says array), while `removeArrayElement` refuses to splice it (409). Honest,
  but a user who hand-wrote the string form cannot remove it from the pane.

## Next

- Package config descriptors (ADR-0119; C0–C5 shipped). Backlog (owner): upstream `picode.config` proposal. Plan: `docs/plans/package-config-manifest.md`.

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
- [ ] **Four vendors' non-empty plugin-list JSON is parsed tolerantly, not from
  a measured sample.** Grok, Muse and Omp ship no plugin on this machine, and
  Antigravity prints prose with no `--json` (its subcommands read a leading
  flag as the plugin name — `agy plugin uninstall --help` really tried to
  uninstall `--help`, measured 2026-09-20). `internal/clipkgs/live_test.go`
  (`PICODE_PKGS_LIVE=1`) is where each shape is pinned; until a real install
  runs there, `parseVendorRows` maps field names by candidate and refuses a
  shape it cannot read instead of reporting an empty list.

- [ ] **Muse Code's plugin roster is still read tolerantly.** The installed
  build (Muse Code 1.3.0) exits 2 with "plugins are not available in this
  build" for every plugin verb, so its non-empty JSON could not be observed;
  `parseVendorRows` maps field names by candidate and refuses an unknown shape.
  Grok, Omp, Hermes, Claude Code, Codex, Antigravity and OpenCode were all
  measured non-empty offline (fixtures in `internal/clipkgs/testdata/`, harness
  in `live_test.go`, `PICODE_PKGS_LIVE=1`). Re-run the harness on a Muse build
  that ships plugins and pin the row shape.
- [ ] **A consent refusal shows the vendor's words but no copy button.**
  Plan §6 said "refusal verbatim + copyable command"; v1 ships the verbatim
  half. Grok is where it bites: `grok plugin install <local dir>` refuses
  without `--trust`, which PiCode never passes (ADR-0167), so the row carries
  the vendor's refusal and the user retypes the command. The command is already
  known server-side (`clipkgs.Argv`); returning it with a refusal is the work.
- [ ] **Reads and the OpenCode splice disagree about a `plugin` string.**
  `opencodeModules` reads `"plugin": "pkg"` as one module (the vendor's schema
  says array), while `removeArrayElement` refuses to splice it (409). Honest,
  but a user who hand-wrote the string form cannot remove it from the pane.

## Next

- Omp's marketplace availability list is unverified: `omp plugin marketplace
  list --json` printed prose for the empty case on 2026-09-20, so the pane
  manages marketplace *sources* for Omp and offers no catalog tab. Measure a
  configured marketplace before promoting `available` in its declaration.
