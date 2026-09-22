# Packages and manifests

## Next

- Package config descriptors (ADR-0119; C0–C5 shipped). Backlog (owner):
  upstream `picode.config` proposal. Plan: `docs/plans/package-config-manifest.md`.
- Guest packages: nothing queued — what is left is under `## Debts`.

## Debts

- [x] **The update action shipped with its signal.** Paid 2026-09-21
  (`feat/pkg-update-badge`, ADR-0167 amendment): `GET
  /api/cli-packages/updates` compares the CLI's roster with its own catalog
  (`clipkgs.CheckUpdates`, `pipkg.Newer`), the pane runs it once per mount where
  the CLI has the verb, and Update appears only on the rows the catalog says
  are behind (`→ 0.3.0`). Measured end to end in the live harness: Muse
  installed at 0.2.0, the marketplace snapshot refreshed with the vendor's own
  command to 0.3.0, the row came back `BEHIND latest=0.3.0`. A catalog that
  cannot be read leaves rows unmarked with the reason; a version pair the
  comparator cannot parse is not a claim. Omp gets the badge without an Install
  because `discover` carries versions.
- [x] **A consent refusal carried no command.** Paid 2026-09-21
  (`feat/pkgs-polish`): `clipkgs.Command`/`MarketCommand` render the exact line
  from the same builder the request executes (with URL credentials redacted),
  every synchronous refusal returns it, a failed package job carries it too
  (`GET /api/cli-jobs` and the 202 answer), and both panes show **Copy command**
  beside the vendor's words with "Run it in a terminal." — Grok's `--trust` and
  Claude's marketplace-declared command are the cases it exists for.
- [x] **Reads and the OpenCode splice disagreed about a `plugin` string.**
  Paid 2026-09-21 by measuring the vendor instead of guessing: OpenCode refuses
  that document itself — running its own `opencode plugin <module>` against
  `"plugin": "is-odd"` prints *"Configuration is invalid … Expected array |
  undefined"* and writes nothing. The read now fails loudly naming the file and
  the form the vendor expects (`feat/pkgs-polish`), instead of listing a module
  the CLI never loads and then refusing to remove it.
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
- [x] **`GET /api/packages/report` still drops the isolation switch.** The badge
  half of this debt was paid 2026-09-21 (`feat/packages-guests`, slice 2): the
  four guest reads answer from `pkgs.DriverFor(cli)` — a pure mapper
  (`internal/pkgs/guest_view.go`) reproduces the pane's bytes, proven over
  fixtures and over the real routes — and every guest driver now maps the
  vendor's whole capability set, `Caps.Update` included. So
  `/api/packages/updates?cli=<guest>` answers the vendor's own check where the
  CLI has the update verb instead of 400. Paid 2026-09-22 (`feat/packages-pane`,
  slice 3 — the close this debt named): `handlePackageReport` hands the agent
  row's flag to the engine (`Query.AgentIsolated`), Pi's driver puts it on the
  report (`Report.Isolated`, `json:"isolated,omitempty"` — a switched-off agent
  is the absent field, which is why an unswitched report has no key), and the
  merged pane both reads it (`checked={!!report.isolated}`) and writes it back
  through `PATCH /api/agents/<id> {packagesIsolated}`.
  `internal/server/package_report_test.go` pins an isolated agent's report at
  `"isolated":true`, and a scratch instance measured the round trip end to end:
  PATCH true → report `"isolated":true` → the box drawn ticked.
  Plan: `docs/plans/packages-unification.md`.

- [x] **Omp's own `extensions` list is invisible in the guest packages pane.**
  Paid 2026-09-21 (`feat/packages-omp`, slice 5 of
  `docs/plans/packages-unification.md`), which is also where the debt was
  measured. The omp roster read now merges the two layers the CLI loads from:
  the workspace's `<ws>/.omp/settings.json`, parsed with the standard library,
  and the user level through the CLI's own `omp config get <key> --json` (never
  a YAML parser for `~/.omp/agent/config.yml`). One row per entry —
  `SourceKind: extension`, the entry as written, the CLI's own name for it
  (base without the extension, `index.ts` → its directory), the resolved path
  when something is there, `Scope` machine or workspace, and `Enabled` from
  `disabledExtensions` (`extension-module:<name>`, the id the vendor's own
  dashboard writes). A row the plugin roster already carries is not duplicated,
  and an entry both layers name is one row — the workspace's. Removal writes
  what the row came from: the workspace entry is spliced out of that file by
  PiCode (the entry, and its id out of `disabledExtensions`, every other byte
  preserved), the user layer goes through `omp config set extensions '<json
  array>'`. Measured 2026-09-21 on 18.2.8: `omp plugin list --json` is the
  plugin store and never names a configured extension; `omp config get
  extensions --json` answers `{key,value,type,description}` for the layer the
  working directory resolves to — a project that declares `extensions` replaces
  the user's, it does not merge; `omp config set` writes the user file wherever
  it runs and refuses `--scope`; and `omp install <path>` links a *package* (it
  fails on a single `.ts` file with ENOTDIR on `<file>/package.json`), so no
  vendor verb adds or removes a configured extension. `omp plugin link <path>`
  stays *not* the answer — for a local package its scope is the machine
  (`--scope=project` wrote nothing into the project; the flag is marketplace
  installs only). Live: `GET /api/cli-packages?cli=omp&scope=project&workspace=
  <picode>` answers the `browser` row with its resolved path, and the removal
  answers the CLI's fresh list instead of reserving a job, because a command
  with no argv is the engine's own write.

- [x] **An extension row draws Enable/Disable, and Omp's disable verb does not
  know extensions.** Paid 2026-09-22 (`feat/packages-toggle`, the debt ADR-0176
  slice 5 recorded for `feat/packages-omp`). `clipkgs.OmpExtensionToggle` asks
  the same two layers the removal asks, by the identity the pane sent: the
  workspace's `<ws>/.omp/settings.json` gets the CLI's own id
  (`extension-module:<name>`) spliced into `disabledExtensions` — out of it when
  the row is being enabled, and the key itself created when the file has none —
  every other byte preserved and the result re-parsed and compared; the user
  layer runs the CLI's own `omp config set disabledExtensions '<json array>'` in
  the user's directory, which is where that command writes (measured 2026-09-21:
  it writes the user file wherever it runs and refuses `--scope`). The driver's
  `Toggle` asks the engine for an extension before it asks for the vendor's verb,
  exactly as `Remove` does, and answers the CLI's fresh list — the line it ran
  rides a refusal, and the workspace write has none, the same empty line
  OpenCode's in-process toggle answers. A plugin row keeps today's behaviour, and
  the pane is unchanged (the control already rendered). Tests:
  `TestOmpExtensionToggleWritesTheWorkspaceFile`,
  `TestOmpExtensionToggleCreatesTheDisabledList`,
  `TestOmpExtensionToggleRunsTheVendorCommandForTheUserLayer`,
  `TestGuestToggleOmpExtensionWritesTheCLIsOwnList` and
  `TestCLIPackagesOmpExtensionToggle`.

- [x] **OpenCode's removal is unreachable from the pane.** Paid 2026-09-22
  (`feat/packages-opencode`): the transport declaration became one fact per verb
  (`Caps.Lane Transport` in `internal/pkgs/pkgs.go`) instead of the single
  `Async` bool, which could only have described one half of a mixed CLI. A verb
  the CLI reaches with its own command is the lane's (`clipkgs.Commands`); a
  verb whose only path is PiCode's edit of the CLI's config file is a write, and
  the driver performs it (`clipkgs.Writes`, asked of the engine before the
  vendor's verbs). OpenCode declares `lane {install:true, remove:false,
  update:false, marketplace:false}`: its install is `opencode plugin <module>`,
  its removal the splice of its own `opencode.json` that `opencode.go` already
  wrote (comments and every other byte survive, the result re-parsed and
  compared, the atomic write keeping the file's own mode). The driver performs
  that write in process and answers the empty line it ran, so
  `POST /api/cli-packages/remove` answers the CLI's fresh list with 200 where it
  answered 400 *"this CLI does not expose that operation: opencode remove"*
  while `Caps.Remove` was true; a module the CLI's own configs do not name is
  still refused by name (`ErrStale`, 409) with nothing written. The pane reads
  the per-verb declaration (`laneMutation(caps, "remove")`,
  `web/shared/domain/cliPackages.js`) and takes the flow Pi's direct mutations
  take — the transcript while the write runs, then the CLI's fresh list read
  back — so the state a removal draws is a transcript where the 400 was drawn as
  a row error. Measured 2026-09-22 on the branch: `GET
  /api/packages/report?cli=opencode&vendor=user` answers that `lane` object;
  removing `opencode-wakatime` from a two-module config left
  `{"plugin": ["opencode-notify"], "model": …, // the module list}` intact and
  answered the one-row fresh list; and a scratch instance drew the removal
  transcript (card read). Tests: `TestOpenCodeRemoveThroughTheEngine` (three
  cases; the expected file is built from the original's bytes around the removed
  element), `TestOpenCodeRemoveThroughTheEngineRefusesAModuleNoConfigNames`,
  `TestTheTransportDeclarationIsPerVerb`,
  `TestGuestRemoveOpenCodeWritesTheFileItReads`,
  `TestCLIPackagesOpenCodeRemoveIsPiCodeOwnWrite`; re-tabled rows:
  `TestGuestMutationRefusalsKeepTheEnginesWords` (the OpenCode removal row) and
  the `directMutation` rows in `cliPackages.test.js`, plus the new
  `the transport declaration is read one verb at a time`.

- [ ] **The `/api/cli-packages*` family is an alias for one release
  (ADR-0176).** The guest routes answer from `pkgs.DriverFor(cli)` and are
  mapped back to the bytes the pane has always parsed
  (`internal/pkgs/guest_view.go`), so a caller can be moved over deliberately;
  `docs/architecture/packages.md` and `docs/architecture/routes.md` carry the
  deadline. The one in-repo caller left is `web/shared/domain/cliPackages.js`
  (`available`, `marketplaces`, install/remove/update/toggle/marketplace/
  inspect) — the unified reads already go to `/api/packages*`. Closes when that
  module reads the unified paths for every CLI and the handlers,
  `guest_view.go`'s route mappers and their tests are removed together.
