# Multi-CLI ADE — taking Pi out of the dependency role

> **Status: approved by the owner (2026-09-22).** Decision: [ADR-0179](../decisions/0179-agent-clis-optional.md).
> Progress and debts: [`docs/handoff/open/multi-cli-ade.md`](../handoff/open/multi-cli-ade.md).
> **Executed 2026-09-22.** All five branches landed; the pi-update route and
> `errAgentCmdMissing` named below as current are gone. An adversarial review
> the same day found and fixed three defects in what landed (guest automations,
> the installer's npm prefix, CLI panes for guest tabs): see the session notes
> `2026-09-22-guest-automations`, `-npm-user-prefix` and `-ade-review-tail`.

## Why

The 2026-09-22 inventory found the multi-CLI migration complete in the
decisions (ADR-0160 made any launchable CLI an agent; ADRs 0097, 0150, 0154,
0163, 0165, 0167, 0169, 0171, 0174 all say "every agent CLI") and almost
complete in the runtime, but missing from three layers:

1. **Behaviour that still treats Pi as infrastructure** (ADR-0003, never
   superseded): the System page's fixed `Pi` block and npm warning; the
   `LookPath("pi")` gate that fails every automation; `piStep` blocking
   `picode provision`; the Windows installer that cannot finish without Pi and
   reads the Node major from Pi's package; the free-agent path that injects Pi
   as installed and only creates Pi.
2. **Front-door copy** defining the product as "for Pi": landing page, meta
   description, `llms.txt`, Getting started, README, `--help` banner, the
   systemd unit's `Description=`, the GitHub description.
3. **`AGENTS.md` itself** ("managed agents remain Pi"), loaded by every agent
   session and false since ADR-0160 (2026-09-19) — the root cause of the bias
   reproducing.

Owner decisions (2026-09-22): Pi leaves the installer runtime (CLIs are
installed from Agent CLIs, ADR-0093's lane); free agents get a CLI picker;
all five branches, in the order below.

## What stays Pi-only

See ADR-0179's decision table: managed mode (`pi --mode rpc`, structured
chat, composer, `/api/catalog`, Pi's Packages and Settings panes, `/trust`),
automation **start** runs (they create a Pi agent), and automation
**message** runs whose target is a stopped Pi agent. Everything else stops
checking for Pi.

Each branch is one session: `make worktree NAME=<x>`, iterate with
`make ci-scoped`, finish with `make close`, note + changelog fragment written
in a subagent, `git merge --ff-only` on the root and `make ci`. Merge `main`
first: other sessions edit `docs-site/` on the root checkout.

## 1. `feat/ade-boundary` — the decision and the front doors (landed with ADR-0179)

ADR-0179; `AGENTS.md` scope paragraph; README strapline, "Why PiCode", the
"What stays yours" table, quick start, from-source, diagram; docs-site
`config.mjs` description, landing hero and cards, Getting started (tmux →
PiCode → the CLIs you use → create an agent, with the Agent vs Terminal
sentence), `from-source.md`, `remote-server.md`, `shared-server.md`,
`public-access.md`; `scripts/docs-llms.mjs` intro; `--help` banner
(`cmd/picode/main.go`); unit `Description=` (`internal/install/unit.go`);
`docs/architecture.md` one-paragraph definition; this plan and the topic file.

## 2. `feat/system-clis` — System page and the automations gate

**Server.** `internal/server/system.go`: remove the `Pi` struct and the npm
warning from `systemReport`; `/api/system` probes no CLI. tmux warning becomes
"tmux is not installed — agents and terminals need tmux 3.5+ …". Remove
`PiUpdateCard` and `POST /api/system/pi-update` (`server.go:259`) once the
scratch instance confirms ADR-0087's lifecycle lane updates Pi
(`internal/clilifecycle/lifecycle.go:94`, the Update pill in
`AgentClis.jsx:178`); otherwise keep the route and move the button to
`#/clis/pi`.

`internal/server/automations_run.go:157`: `fireInput` gains `TargetIsPi`;
`decideFire` applies `PiMissing` only when `Action == start` or `TargetIsPi`.
`skipBody(reasonPiMissing)` becomes "Pi is not installed or not on PATH, so
this Pi agent could not start."

| Conditions | Action |
|---|---|
| start, pi absent | failed · pi missing (exists) |
| message, Pi target stopped, pi absent | failed · pi missing (exists) |
| message, Pi target running (managed), pi absent | deliver (exists) |
| message, guest target, pi absent, launch terminal exists | deliver through the door (`doorRun`) — **new** |
| message, guest target, pi absent, no launch terminal | failed · "this agent has no launch terminal" (`doorRun:260`) — **new** |

Rows live in `automations_test.go:70-76`; add the last two.

**UI** (`web/browser` and `web/mobile` twins; `System.jsx` is byte-identical,
keep it so). `System.jsx:83-95`: Dependencies = tmux (required), mkcert and
tailscale (optional); new "Agent CLIs" section fed by `GET /api/clis`
(`describeCLI` over `clilaunch.Catalog()`, `cli_launch.go:209`): one row per
CLI with `diagnostic.version`, "not installed" or "update available" linking
to `#/clis/<id>`; empty state "No agent CLI installed yet." + "Open Agent
CLIs". Reuse `.sys-rows`. `Automations.jsx:84,200-202` (+ mobile
`:108,241-243`): `piMissing` reads the `pi` row of `/api/clis` instead of a
regex over `warnings`; banner only when some automation needs Pi (start, or
message to a `cli=pi` agent); text "Pi is not installed, so automations that
start or message a Pi agent cannot run."; action "Install Pi" → `#/clis/pi`.
Fixture `scripts/qa-mobile-workflows-v2.mjs:84,284`.

Docs: `docs/architecture/automations.md` (fire table),
`docs-site/guide/automations.md:143`, `docs/architecture.md` (the ADR-0003
"helpful dependency UX" paragraph). OpenAPI regenerates in `make close`.
Visual review: System and blocked Automations, desktop and 390px,
`window.__picodeOverlayAudit()`.

## 3. `feat/runtime-without-pi` — provision and the Windows installer

**Provision** (`internal/provision/steps.go:36,103-115`): `piStep()` becomes
`clisStep()` (ID `clis`, "agent CLIs on PATH"): walks `clilaunch.Catalog()`
with `lookPath`, always `ok(...)` ("found: pi, claude, codex" or "none yet —
install one from Agent CLIs after first login"), no `Fix`, never blocks
`Converged()`. Tests: `steps_test.go:214` (order) and `TestPiStep` →
`TestCLIsStep`. `internal/provision/container.go:87-90`: the member container
image installs tmux and git; CLIs come from Agent CLIs. Docs:
`docs/architecture.md:50-54`, `docs-site/guide/remote-server.md` (drop the
ADR-0179 caveat), `public-access.md:76`.

**Installer** (`internal/desktop/stages.go`, `cmd/picode-desktop/stages.go`,
`internal/desktop/bootstrap.go`): `RuntimeTools` = tmux, git, curl, node,
npm; probe without `missing:pi`; `RuntimeNodeMajor = "22"` (CI parity)
replaces `piNodeMajor` / `PiRegistryURL` / `ParsePiNodeMajor`; remove
`needPi`, `PiInstallArgs`, `UserPiInstallScript` and the branches at
`:182-190,:213-216,:257-276`; `--user` (`main.go:47`) aims the binary only;
sentences `bootstrap.go:296` and `stages.go:168` without "pi". Tests:
`internal/desktop/stages_test.go` (probe fixture, `TestParsePiNodeMajor`
removed), `bootstrap_test.go:81`, `cmd/picode-desktop/stages_test.go:244,282,293`.

| Conditions | Action |
|---|---|
| tmux/git/curl/node/npm missing, Ubuntu | install-runtime (apt + NodeSource 22) |
| all present, Pi absent | provision (Pi is not checked) |
| node present, major < 22 | upgrade through NodeSource |
| non-Ubuntu distro, something missing | refuse with instructions (as today) |

Docs: `docs/architecture.md:70-78`, `docs-site/guide/windows-desktop.md`,
the bullet in `docs/handoff/open/windows-wsl.md` ("aims both the binary and
pi" → the binary). Acceptance evidence = a run on the `picode-test` VM
without Pi (owner; a debt in the topic file until it happens).
`make desktop-restart` is the owner's.

## 4. `feat/free-agent-cli` — free agents of any CLI

**Server** (`internal/server/agents.go:496-520`): the free-agent create takes
`cli` (empty = pi) and calls `AddAgentWithCLI(store.FreeWorkspaceID, …)` (the
store already maps an empty workspace to free, `store/agents.go:504-506`);
for non-Pi, `attachAgentTerminal` generalized to `(workspaceID, path)` and
called with `FreeWorkspaceID` + `agent.WorkPath` (a free terminal, as
`handleCreateCLITerminal` already makes); `patchNewAgent` only for Pi.
Mirrors the workspace handler (`agents.go:541-590`).

| Conditions | Action |
|---|---|
| `cli` empty or `pi` | row `cli=pi`, provider/model/thinking applied |
| launchable CLI, installed | row with that `cli`, free terminal with launch set |
| launchable CLI, not installed | refuse "X is not installed." (the handoff's sentence) |
| unknown or detect-only CLI | refuse (already in `normalizeAgentCLI`) |

Tests: free-agent handler per CLI, `TestAddAgentWithCLI`;
`TestEveryMutationAppendsAnEvent` is unchanged (same mutator).

**UI** (`web/browser` + `web/mobile`): `Sidebar.jsx:228-232`,
`App.jsx:3690,3785,4283-4284` and the empty card stop opening `CreateForm`
kind=free and open `NewCliPrincipal` in free mode (Folder field, POST
`/api/agents`); `NewCliPrincipal.jsx:52` stops requiring a workspace;
`managedPrincipal.js:17` stops forcing `installed: true` on Pi (Pi stays
first when installed; empty falls into "No agents to create. Install a CLI
first."). `CreateForm` remains for `kind === "workspace"`. Deliberate
simplification: a free Pi agent's provider/model/thinking are set afterwards
in Settings, as the workspace path already does.

Docs: `docs/architecture/managed-principals.md`,
`docs-site/guide/getting-started.md` (pick the CLI), `agent-clis.md:206-213`.
Visual review of the dialog and the empty states in both apps.

## 5. `feat/ade-copy-tail` — configuration cluster, labels, internal docs

No new behaviour. Pattern: where the path is "Agent CLIs → Pi → X" and X
exists for other CLIs (ADRs 0150/0163/0167/0169/0174), say "Agent CLIs → the
CLI → X" and name Pi only where it is the only one.

- docs-site: `configure.md:10-26`, `settings.md:2,9` (title "CLI settings",
  Pi body: split into the Pi editor and the other CLIs' native settings),
  `providers.md:2`, `mcp.md:2,9-15`, `guide/index.md:16`,
  `agent-clis.md:31,39`, `integrations.md:15-16,44`, `backup.md:23,39` (say
  only `~/.pi` is covered), `windows-desktop.md:184`, `commands.md:3` (it is
  the managed Pi agent's composer).
- internal docs: `docs/guidelines.md:56-71` + `docs/benchmarks.md:54-55`
  ("pi correlation" → "vendor correlation" for the nine CLIs),
  `docs/philosophy.md:19-51`, `docs/README.md:8,20`, `CONTRIBUTING.md:3,20,52`,
  `docs/architecture.md:168,281-309` (diagram with the Agent CLIs stack).
- UI (twins): aliases `userMenuModel.js:50-52` / `moreMenuModel.js:44-46`;
  `Palette.jsx:52` (neutral "CLI settings" opening the hub);
  `routes.js:230-232` (generic commands go to the hub, not to Pi);
  `AgentClis.jsx:278` "Open Pi" → "Open <CLI>"; `Mcps.jsx:416-418,433`
  fallback to the current CLI; `BrowserPage.jsx:946`, `ComputerPage.jsx:237`
  ("A CLI started outside PiCode…"); `Automations.jsx:410,679` "pi's default"
  → "the CLI's default"; `Settings.jsx:513` "Copy Pi sessions (~/.pi)";
  `PiSettings.jsx:144,193`; `termMenu.js:95`; `App.jsx:685,2202`;
  `notice.js:253,285` (the agent's real `cli`); `web/desktop/management.html:160`.
- Go: comments `server.go:60`, `rpc/runtime.go:185`; dead sentinel
  `errAgentCmdMissing` (`agents.go:659-677`); `mcp.go:37` empty cli;
  `packages_watch.go` skips the scan when `~/.pi/agent` does not exist.
- `PiKeys.jsx` (serves six CLIs) and `PiSpinner.jsx`: rename only if the
  churn is worth it; otherwise record as a debt.

## Verification

1. Per branch: `make ci-scoped` green, `make close`, note + fragment, ff on
   `main`, `make ci`.
2. Scratch instance (`scripts/qa-scratch.sh`) with `pi` off the PATH:
   `/api/system` without a `pi` key or npm warning; System shows tmux as the
   only requirement and the Agent CLIs section with "not installed" for Pi;
   no Automations banner when only guest-agent automations exist; a message
   automation to a Claude Code or Omp agent with its terminal open delivers through the door;
   `picode provision --dry-run` converges with an informational `clis` step;
   a free Omp agent from the sidebar opens its free terminal.
3. Instance with `pi` on the PATH: nothing regresses (System shows Pi's
   version, start automations run, managed chat intact).
4. `go test ./internal/desktop/... ./cmd/picode-desktop/... ./internal/provision/...`
   green; the real installer run on the `picode-test` VM without Pi stays a
   debt until the owner runs it.
5. Docs: `make docs` (regenerates `openapi.json` and `llms.txt`; the new
   sentence appears in `docs-site/public/llms.txt`), Vale clean, `docs-check`
   with no pending capture (no committed System/Automations capture exists).
6. Visual review (branches 2 and 4): screenshots read in a subagent,
   `__picodeOverlayAudit()` ok, the five-question card answered.

## Owner steps outside the repo

- `gh repo edit cfpperche/picode --description "PiCode — browser-based Agent Development Environment for coding-agent CLIs (Pi, Claude Code, Codex, Grok, Hermes, OpenCode, Muse Code, Antigravity, Omp)."` and review the topics.
- `make deploy` when the branches should go live (in a batch);
  `make desktop-restart` after branch 3; the VM run.

## Risks

- `/api/system` changes shape: an external client reading `system.pi` breaks
  (only the UI and one QA fixture do today).
- Removing `POST /api/system/pi-update` depends on the lifecycle lane really
  updating Pi; otherwise keep the route and move the button to `#/clis/pi`.
- Branch 4 changes how a free Pi agent is created (configure after, not
  during); the owner may veto that simplification in review.
- Parallel sessions on the root: never `git add -A`; merge `main` before
  every `make close`.
