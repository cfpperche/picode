# Packages unification — one subsystem, per-CLI drivers

Plan for review (2026-09-21). Supersedes the *engine* split of ADR-0167; the
ADR itself travels with slice 0. Owner's constraint: **nothing Pi does today
may change** in observable behaviour.

## Why

Pi and the eight guest CLIs are served by two engines chosen by *which CLI a
row is*, not by what that CLI's own mechanism can do. Pi's surface predates
the multi-CLI direction (ADR-0010) and ADR-0167 kept it verbatim ("Pi keeps
`internal/pipkg`… unchanged") while generalizing the pane and the routes for
the guests. Since ADR-0160 (CLI runtimes are agents) that axis is legacy.

Measured symptom (2026-09-21): an extension installed for the `picode`
workspace by a project setting appears in Pi's pane — `Packages.jsx` reads
`.pi/settings.json` — and is invisible in Omp's, because `GuestPackages.jsx`
reads `omp plugin list --json` and the entry is not a plugin. Omp loads it
(`omp config get extensions --json` returns it); only the reader is missing.

## Done means

1. One engine — `List / Available / Install / Remove / Update / Toggle /
   Inspect` with `cli` as a parameter; scope is a *capability* the driver
   declares (`machine | workspace | agent`), not a Pi-only branch.
2. One HTTP family `/api/packages*`; `/api/cli-packages*` answers as an alias
   for one release.
3. One pane, in both apps, whose controls are gated by declared capabilities.
   Every Pi control exists with the same words and the same effect.
4. The agent scope is a capability: Pi and Omp load extensions at launch
   (`-e`, both measured), Claude Code / Codex / OpenCode inject MCP servers
   through the lane ADR-0154 already built, and a CLI with neither declares
   none.
5. Omp's own `extensions` / `disabledExtensions` render as rows — closing the
   debt in `docs/handoff/open/packages.md`.

## Non-negotiable: Pi does not regress

Every row below keeps its observable behaviour; where the user sees words,
the words stay.

| Pi today | Where | After |
|---|---|---|
| Radios **This machine / This workspace (name) / This agent (name, "only X, every session")** | `Packages.jsx` | same labels, produced from the driver's declared scopes |
| Install runs `pi install [-l] <src> --no-approve` with a step transcript | `pipkg.MutateArgs`, `Packages.jsx` | same argv and transcript, through the shared job lane |
| Agent scope: `agents.packages`, `pi -e` on every start, isolation `--no-extensions --no-skills --no-prompt-templates --no-themes` | store + `CLIFlags` | same store field and flags; named by the launch-injection capability |
| Marketplace tab: npm gallery (`keywords:pi-package`) | `pipkg.Gallery` | driver capability `Catalog = gallery` |
| Update badge ("behind") from `/api/packages/updates` | update check | driver capability `Update`, same badge |
| **Configure** (roles, compact, web-search…) and **Describe config…** | descriptors (`/api/packages/config`, `/describe`, ADR-0119) | driver capability `Config`, same editors and links |
| Derived `capabilities.webSearch` | `DetectWebSearch` | row on the unified `Report` |
| Source validation (shell metacharacters), `Kind` (npm/git/path), relative-path resolution to the settings dir | `ValidSource` / `KindOf` / `AbsPathSource` | unchanged, under the Pi driver |
| Empty state **Open the Marketplace** | pane | same |
| Mobile parity | both apps carry the pane | same component, both apps |

A **parity suite** pins this table; it is the acceptance for slices 1–3.

## Non-goals

- No PiCode package database and no cross-CLI transfer (ADR-0167 stands).
- No generic config editor for a guest plugin.
- No new vendor semantics: parsers, argv builders and fixtures stay exactly as
  measured.
- No change to how a CLI loads its own files.

## The model

```go
type Scope string // machine | workspace | agent

type Row struct {
    CLI, Name, Source, Kind string
    Scope                   Scope
    Enabled                 bool
    Version, InstalledPath  string
    ConfigKind, Provenance  string
}

type Caps struct {
    List, Install, Remove, Update, Toggle, Inspect, Catalog, Config bool
}

type Report struct {
    CLI          string
    Scopes       []Scope
    Rows         []Row
    Catalog      Catalog  // none | gallery | vendor | marketplace
    Caps         Caps
    Capabilities Capabilities // webSearch, …
}

type Driver interface {
    ID() string
    Scopes() []Scope
    Caps() Caps
    List(ctx context.Context, q Query) (Report, error)
    Available(ctx context.Context, q Query) ([]Row, error)
    Install(ctx context.Context, r Request) (Command, error)
    Remove(ctx context.Context, r Request) (Command, error)
    // Update / Toggle / Inspect are reached only where Caps declares them.
}
```

`Command` is the pair ADR-0167 already ships: the argv (or the file write)
plus the exact line shown to the user. Mutations run through
`internal/clijob` for every driver.

`internal/pkgs` holds the interface and the registry. `pipkg` becomes the Pi
driver (its readers, validators, gallery and descriptors intact); `clipkgs`
becomes the eight drivers (same `spec` table, same argv builders, same parsers,
same `testdata/` fixtures — one file per CLI behind the interface).

## Routes

| Today | After |
|---|---|
| `/api/packages*` | same path, `cli` optional (default `pi`) |
| `/api/cli-packages*` | alias onto the same handler for one release |
| `cli=pi` refused with 400 (`cli_packages_test.go`) | answered; that row is replaced by the parity suite |

Hashes keep their redirects (`#/clis/packages/pi*` → `#/clis/pi/packages`,
already shipped by ADR-0167).

## Slice plan

| # | Target | Change | Gate |
|---|---|---|---|
| 0 | Design record | ADR `packages-unification` (supersedes 0167's engine clause; amends 0010) + this plan | owner's review |
| 1 | Interface + Pi driver | `internal/pkgs` with `Driver`/`Caps`/`Row`; Pi behind it; `/api/packages*` answers `cli=pi|""`; `/api/cli-packages?cli=pi` starts answering; **no UI change** | parity suite green; every existing `pipkg` and `/api/packages*` test green unchanged; zero diff under `web/` |
| 2 | Guests behind the interface | the eight drivers; parsers, argv builders and fixtures untouched | `internal/clipkgs` tests + fixtures green; live harness `PICODE_PKGS_LIVE=1` |
| 3 | One pane, both apps | merge the two components; controls gated by `Caps`; vendored refresh paths unchanged | visual pass on machine / workspace / agent / empty / marketplace / refusal, desktop + mobile; `__picodeOverlayAudit()` ok |
| 4 | Agent scope as a capability | driver declares launch injection: Pi `-e` (unchanged), Omp `-e` wired into its launch plan, MCP CLIs name the ADR-0154 lane | Omp agent scope writes the entry and the next launch carries `-e`; Pi unchanged |
| 5 | Omp's own extensions source | driver reads `extensions` / `disabledExtensions` (project `<ws>/.omp/settings.json`, user `~/.omp/agent/config.yml`); rows + remove writes the project file | the debt in `docs/handoff/open/packages.md` closes; seen in the pane |
| 6 | Docs | one `docs/architecture/packages.md` replacing the split description; guides (packages, picode-mcp, browser-tool); routes doc; ADR index | `make docs-check` + vale |

Order matters: 1 and 2 are behaviour-neutral, 3 is the only visible arc, 4–5
close the case that exposed the split.

## Decision table (the conditions that change the outcome)

| CLI | Scope asked | Driver declares it | Action |
|---|---|---|---|
| any | machine | yes | list / install through the driver's own verb |
| any | workspace | yes | project file or vendor project verb; with no workspace selected, one line of context — never a control that cannot work |
| any | agent | yes (launch injection) | write the agent's list; the note says it takes effect at the next start |
| any | agent | no | the radio is absent; a link naming it is refused with the reason |
| any | any | CLI not installed | one line + **Check setup**, never an empty roster |
| guest | any | vendor output unparseable | `ErrRosterShape` in the vendor's words, never "none installed" |
| pi | any | n/a | identical to today |

## Risks

| Risk | Mitigation |
|---|---|
| Pi regresses behind the interface | slice 1 is behaviour-neutral by construction: both route families answer, the existing tests are untouched, and `web/` is unchanged |
| The single pane drops a control | the non-regression table is the checklist; the visual pass captures every state on both apps |
| The job lane changes Pi's UX | owner call in slice 1; refused → Pi keeps its synchronous call and the difference becomes a capability (`Async`) |
| Guests' tolerant readers drift | fixtures and the live harness stay the contract; parsers do not move |
| Route churn breaks callers | `/api/cli-packages*` alias for one release; hash redirects already exist |

## Owner calls

1. Pi's mutation on the shared job lane (202 + job row) or keep it synchronous
   behind an `Async` capability?
2. Does the agent scope reach every CLI that can take it in this arc (Omp
   first), or stay Pi-only until each CLI is measured?
3. Does `/api/packages*` without `cli` keep answering `pi` for longer than one
   release?

## What would make us stop

- If slice 1 cannot keep `web/` untouched, the model is wrong, not the pane.
- If a guest driver needs a capability Pi's model cannot express, the model
  gains a capability — it does not gain a special case.
