# Native CLI packages (ADR-0102)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Packages is a pane of the selected CLI (`#/clis/<cli>/packages`) in each app.
The shared `cliPackages` module parses canonical/legacy URLs, declares native
package capabilities (Pi initially), and validates workspace/agent identities
from the fleet APIs. Canonical URLs carry `workspaceId`, `agentId` and install
`scope`; an unscoped URL means machine packages. Opening the pane from Agent
CLIs while a sidebar agent is selected writes that identity onto the hash so
the workspace/agent radios appear; an unscoped URL with nobody selected stays
machine-only. Legacy `#/packages*`,
`#/clis/packages*` and mobile `#/more/packages*` rewrite onto the pane. Missing or mismatched targets block editing instead of falling back.
Context refresh failures retain the mounted package view and its draft, with
writes blocked until retry succeeds. Each mutation revalidates its URL target,
including after confirmation. A changed workspace path or agent work path
retains the draft but requires a confirmed reload before editing. Roles saves
retain both layers and preserve an unsaved draft in the other layer.
Package reads show failures with retry rather
than reporting an empty installation. This view does not load terminal inventory
or CLI lifecycle jobs. Existing Pi package APIs, commands and persistence stay
unchanged; the desktop roles editor remains native-file-backed. Mobile config
links offer the desktop layout with the same URL until a mobile editor exists.

Configuration (ADR-0099, ADR-0119): a package is configurable in this view
when a config descriptor resolves for it. Descriptors resolve in order — the
user's own description (`DataDir/package-configs/<id>.json`), then
`picode.config` in the extension's `package.json`, then PiCode's catalog
(`internal/pipkg/configdescriptor.go`). The generic engine writes the declared
file (agent or workspace scope, per descriptor) with the descriptor's typed
fields — enum, boolean (tri-state: untouched = unset), number with min/max,
string, secret — preserving unknown keys and refusing to replace an
unparseable file without explicit force. The config page names the source of
the descriptor (user, catalog, manifest). A user description is created with
**Describe config…** on an undescribed card and edited or deleted from the
config page; deleting it never touches the described file. pi-roles keeps its
bespoke two-layer editor. No descriptor resolves, no Configure button.

## The guest CLIs (ADR-0167)

Pi keeps everything above. The other eight CLIs get the same pane, driven by
`internal/clipkgs`: **one declaration per CLI** says what it really exposes,
and `Caps` is derived from that declaration — a nil argv builder *is* a verb
the CLI does not have, so the pane cannot offer a control the vendor lacks.
The vendor binary stays the authority; PiCode keeps no package database, and
never passes a vendor's auto-consent flag (`-y`, `--trust`, `--confirm`,
`--allow-tool-override`, an accepted command hash). Where a vendor demands a
human answer, the refusal comes back **with the exact command** — rendered
server-side by the same builder the request executed, with credentials in a
URL redacted — and the pane offers it with **Copy command** beside the vendor's
own words, for synchronous refusals and for a failed job alike.

`GET /api/cli-packages` answers with the CLI's scopes, capabilities, notes and
rows. Scopes are the CLI's own: Claude Code adds `local` (uncommitted) beside
`user` and `project`; OpenCode, Muse and Omp have machine and project; Codex,
Grok, Hermes and Antigravity are machine-only. The unified read
(`GET /api/packages/report`, ADR-0176) adds PiCode's own `agent` layer where a
launch can pass it on: Omp's entries live on the agent row like Pi's, are
answered from the store with no vendor call, and reach the CLI at its next start
as `-e` — with `--no-extensions --no-skills` when the agent is isolated, the two
flags that CLI has (ADR-0176 slice 4). The legacy guest route still refuses
`agent` by name — no vendor has a per-terminal plugin layer — and a project
scope without a workspace folder is refused before anything runs.

The roster comes from the CLI itself:

| CLI | Roster | Notes |
|---|---|---|
| Claude Code | `claude plugin list --json` (+ `--available`) | `id@marketplace`, scopes `user\|project\|local`, plus vendor-owned `synced` rows |
| Codex | `codex plugin list --json` (+ `--available`) | remote marketplaces need the network and a sign-in |
| Grok | `grok plugin list --json` (+ `--available`) | |
| Hermes | `hermes plugins list --json`, catalog via `plugins search --json` | `picode-native` is PiCode's own integration |
| OpenCode | its own configs and plugin directories | no list, remove, disable or marketplace command exists |
| Muse Code | `muse plugins list --json` (+ `--available`) | the id lives in the row's `record`, not on the row; capability trust stays with the user. The CLI gates its whole plugin surface per machine through its own cached feature config — where that gate is off, every verb answers *"plugins are not available in this build"* and the pane shows that sentence. The catalog does **not** mark an installed plugin (`status: "available"` survives an install), so `catalogNeedsRoster` joins it with the CLI's own roster; and the CLI's list omits the app's built-in first-party plugins, which only its TUI shows (all measured 2026-09-21) |
| Antigravity | `agy plugin list` (a JSON envelope once plugins exist, one sentence while none do) | its subcommands take no flags: a leading `--help` is read as the plugin name (measured 2026-09-20) |
| Omp | `omp plugin list --json`; catalog via `omp plugin discover` (prose — `--json` is accepted and ignored) | the catalog is information only: discover prints a name and a version and never names the source an install needs, while `omp plugin install <name>` resolves through npm (measured 2026-09-21: it 404s on the registry), so `catalogInstall` is false and the list note states the `name@marketplace` form |

Mutating a plugin runs the vendor's own command, in the workspace folder for a
project scope, through the durable job lane (`internal/clijob`, ADR-0087):
202 with the job, request-key idempotency, at most one active job, and the
existing refusal while that CLI's terminals are running (a plugin lands in the
next start). A job carries its arguments in its payload, so `Resolve` can build
the argv after a restart; every mutation publishes the ephemeral `cli.packages`
event and drops that CLI's roster cache.

**OpenCode is the one file write.** It exposes `opencode plugin <module> [-g]`
to add and nothing to remove, so removal deletes one element from the `plugin`
array of the config file that names it, using the precedence
`internal/connectors` already measured (the `.jsonc` first). The edit is
surgical: comments and every other byte survive,
the result is re-parsed and compared to the document before it (only that
element may differ), the write is atomic with the file's own mode, and a shape
the scanner does not recognize is refused (409) rather than saved. A
`"plugin": "name"` **string** is not a module PiCode lists: OpenCode itself
refuses that document ("Expected array | undefined"), so the read names the file
and the form the vendor expects instead of showing a plugin the CLI never loads.

Rows show the vendor's own words. A verb the CLI lacks carries the declaration's
one-line note instead of a disabled button: Codex cannot disable a plugin, OpenCode
has no marketplace, Antigravity has no update command. `update` is deliberately
absent for guests in v1 — an availability badge needs the vendor's catalog
compared, and a button without that signal is a blind action; the row names the
vendor's command instead (`docs/handoff/open/packages.md`).

**Availability, and why Update is conditional.** `GET /api/cli-packages/updates`
(`clipkgs.CheckUpdates`) compares the CLI's own roster with its own catalog and
marks the rows the catalog publishes a newer version of; the pane runs it once
per mount where the CLI has an update verb, and a row offers **Update** only
there — before the check the pane offers **Check for updates** instead of one
button per row. Versions are compared with `pipkg.Newer`, so a vendor that
versions outside semver gets no badge rather than a wrong one, and a catalog
that cannot be read leaves the roster unmarked with the reason in the note
(never "everything is up to date"). Omp's version-only catalog is enough for
the badge even though it cannot drive an Install; Codex, OpenCode and
Antigravity have no update verb and so no Update action at all.

A read that fails is never an empty list: unparsable vendor output is a 502
carrying the CLI's own text, a missing binary is a 400, a changed file is a 409.
`internal/clipkgs/live_test.go` exercises every driver against the real
binaries in a sandbox HOME (`PICODE_PKGS_LIVE=1`), which is where a vendor's
shape change is meant to be caught.

### How a guest list reads (2026-09-21)

The defaults are the machine's scope: the web clients **omit** `scope` for it,
so `clipkgs.List`, `Available` and `CheckUpdates` resolve the scope once at the
top (`normalizedScope`) and use the resolved id for the vendor's argv, the rows'
filter and the cache key. Passing the empty string through was a defect, not a
shorthand: Claude Code's parser compares a row's scope with the requested one,
its rows say `user`, so the pane showed an empty list for a CLI that has plugins
(regression test `TestTheDefaultScopeReadsTheMachineScope`).

`web/shared/domain/cliPackages.js` decides what the pane shows without touching
the rows: `groupInstalledRows` groups a list that mixes origins by the vendor's
own provenance word (`sourceKind` — the specific kinds before the marketplace
fallback, since Claude names a synced plugin's marketplace `synced` too) and
returns nothing for a single-origin list, so a header only ever explains a
mixture; `matchParts` splits text around the filter's needle for the highlight.
Both are pure and tested in `cliPackages.test.js`.

The list is the same card grid Pi's Packages pane uses. The filter row and the
availability check sit in one `position: sticky` block, because a vendor catalog
runs to hundreds of rows; actions live on a card footer the grid aligns across a
row, and the two app files differ only on their dialog line. A card carries no
preview frame: Pi's has one because a package may ship a capture, a vendor plugin
has none. On an empty installed list the pane reads the catalog once (where the
CLI has one and can install) and offers up to three real entries with Install —
the vendor's own data, never a suggestion PiCode made up.
The availability check is not part of the filter: it renders whenever the CLI has an update verb, including a list of one or of none. A group header's count is plain text, not the toolbar's count chip.
