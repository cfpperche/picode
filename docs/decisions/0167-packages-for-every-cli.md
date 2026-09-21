# ADR-0167: Native packages for every agent CLI

- **Status**: accepted (owner approved the study and its four recommendations, 2026-09-20)
- **Date**: 2026-09-20
- **Boundary**: persistence — PiCode writes other CLIs' own plugin stores (OpenCode's `plugin` array; nothing else needs a file write, but the surface owns the same files the vendor does). process — PiCode runs vendor plugin commands, which fetch from git, npm and vendor marketplaces. security model — who consents to a plugin's code, and how far PiCode goes on the user's behalf.
- **Amends**: ADR-0069 ("PiCode does not implement other CLIs' … package managers") for a bounded exception: PiCode runs the vendor's own plugin commands and keeps no package manager of its own. **Extends**: ADR-0102 (Packages under Agent CLIs), ADR-0150 (one driver per CLI behind one interface). **Sibling**: ADR-0163 (the generic-file-driver family).

## Context

`#/clis/<cli>/packages` is Pi only: `supportsCliPackages` is `["pi"]`
(`web/shared/domain/cliPackages.js`), and every other CLI renders *"Packages
for &lt;CLI&gt; are in development — coming soon."* The other eight CLIs PiCode
launches each ship a plugin system, verified on this machine 2026-09-20:

| CLI | Verbs (vendor's own) | Roster | Scopes |
|---|---|---|---|
| Claude Code | `claude plugin {list,install,uninstall,enable,disable,update,details,marketplace}` | `list --json`, `--available` | user, project, local |
| Codex | `codex plugin {add,remove,list,marketplace}` | `list --json`, `--available` | machine |
| Grok | `grok plugin {list,install,uninstall,update,enable,disable,details,validate,marketplace}` | `list --json`, `--available` | machine |
| Hermes | `hermes plugins {list,install,search,browse,update,remove,enable,disable,capabilities,doctor,pack}` | `list --json`; curated catalog | machine |
| OpenCode | `opencode plugin <npm module> [-g]` writes config; **no list, no remove, no marketplace** | its config `plugin` array + two plugin directories | global, project |
| Muse Code | `muse plugins {install,list,inspect,approve,reject,enable,disable,update,remove,validate,marketplace}` | `list --json`, `--available` (the id lives in each row's `record`; the CLI gates the whole surface per machine through its own feature config) | user, project |
| Antigravity | `agy plugin {list,import,install,uninstall,enable,disable,validate,link}` | `list` prints a JSON envelope once plugins exist, one sentence while none do (no flag) | machine |
| Omp | `omp plugin {install,uninstall,list,link,doctor,features,config,enable,disable,marketplace,discover,upgrade}` | `list --json`; the catalog is `discover`, which names no source and so is shown as information | user, project |

Two accepted decisions stand in the way and are amended by name: ADR-0069
(no other CLI's package manager) and ADR-0102's alternatives ("one generic
package backend … would invent common package semantics").

The depth is uneven in the direction that matters to a UI: six CLIs expose
their whole surface with `--json`; OpenCode has no roster command at all;
Antigravity prints a sentence. A pane that flattens the three invents controls
the vendor does not have.

## Decision

Every agent CLI's Packages pane at `#/clis/<cli>/packages` manages that CLI's
native plugins through its own binary, behind one interface in a new
`internal/clipkgs` and a per-CLI declaration — the shape ADR-0150 gave
Connectors and ADR-0163 gave Settings. Pi keeps `internal/pipkg` and its
existing pane unchanged (machine / workspace / agent scope, PiCode's gallery,
config descriptors); guest CLIs get what they actually expose and nothing
else. Reads prefer the vendor's `--json`; a CLI without a roster command is
read from its own config and plugin directories (OpenCode), and a CLI whose
roster has no machine-readable flag is read from its own output either way —
Antigravity's `list` prints JSON once anything is imported and a sentence while
nothing is. Install, remove, enable/disable and marketplace
actions run the vendor's own command through the existing durable job lane
(`internal/clijob`, ADR-0087/0093) — it refuses while that CLI's terminals are
live unless confirmed, which here is the point: a plugin lands in the CLI's
next start. PiCode never passes a vendor's auto-consent flag (`-y`, `--trust`,
`--accept-hooks`, an accepted command hash); where a vendor demands a human
confirmation, the pane shows the vendor's refusal verbatim with a copyable
command for a terminal. No PiCode package database, no cross-CLI transfer, no
per-agent scope for a guest, no generic config editor for a plugin PiCode has
no schema for, and no update badge until a real availability signal exists.

## Consequences

One surface for the nine CLIs PiCode launches, and a pane whose controls are
the vendor's own verbs — the asymmetry is stated in words where a control
cannot exist, instead of a disabled button. Costs, accepted: a guest pane is
never richer than its vendor (no toggle for Codex, no marketplace for
OpenCode, no roster JSON for Antigravity, no update badge for anyone in v1);
PiCode becomes a participant in stores other programs write, so
re-read-before-write, 409 on a stale revision and atomic writes are
load-bearing, and the OpenCode splice is the one place PiCode edits a plugin
list by hand; every vendor's CLI output is a moving target — a shape change
must fail loudly (never an empty roster that reads as "nothing installed"),
with one live test per driver that breaks when the vendor's output moves.
If we are wrong about a vendor's shape, that CLI's pane is blocked with the
vendor's own error text and nothing else regresses; if we are wrong about
consent, PiCode has passed a flag the user should have answered, which is why
no auto-consent flag is ever sent.

## Alternatives considered

- **One PiCode package backend shared by the CLIs**: where ADR-0102 already
  refused it. It would have to invent common install semantics across npm,
  git, marketplace refs and vendor caches, and would own state each vendor
  owns.
- **A PiCode-curated cross-CLI plugin catalog**: would make PiCode vouch for
  other vendors' plugin code — a trust decision it does not own. The vendor's
  available list carries the facts the pane must show (Codex's `installPolicy`
  and `authPolicy`, Claude's marketplace-declared command hash).
- **Read-only guest panes (roster without verbs)**: a fabricated surface one
  paragraph from the connectors plan already refused; install is the reason a
  user opens this pane.
- **Per-CLI bespoke panes**: eight near-identical surfaces to keep in sync for
  no capability gain (ADR-0150's reasoning, unchanged).
- **Writing every guest's roster ourselves instead of asking the CLI**: the
  vendor's cache, snapshots and trust state are not a file format PiCode can
  promise; the binary stays the authority.
- **Shipping `update` with a blind button**: an action that runs without any
  availability signal is the "fabricated control" failure; the row names the
  vendor command and availability waits for a real comparison.

## Amendment (2026-09-21)

The Consequences above left an update badge out "until a real availability
signal exists". That signal now exists and is the CLI's own: `GET
/api/cli-packages/updates` compares the installed roster with the CLI's catalog
(`clipkgs.CheckUpdates`), `pipkg.Newer` decides, and the pane offers **Update**
only on the rows the catalog says are behind — before a check it offers the
check itself, and a catalog that cannot be read says why instead of reporting
everything current. A version pair the comparator cannot read is not a claim.
Five CLIs gain the action (Claude Code, Grok, Hermes, Muse Code, Omp); Codex,
OpenCode and Antigravity keep none, because no vendor update verb exists.
Omp's version-only catalog supports the badge even though it cannot drive an
Install (amendment above).
