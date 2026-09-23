# Skills for every agent CLI — inventory, install, toggles, agent sets, marketplace, experiments

Plan approved by the owner on 2026-09-23. Decision:
[ADR-0196](../decisions/0196-skills-for-every-agent-cli.md). Study:
[2026-09-23 — a skills marketplace](../benchmarks/2026-09-23-skills-marketplace.md).
Open items: [`docs/handoff/open/skills.md`](../handoff/open/skills.md).

## Why

PiCode reaches a skill only inside one CLI's package or plugin (ADR-0176).
Standalone skills are invisible. A standalone skill is a folder with a
`SKILL.md` in the agentskills.io format, and all nine CLIs read that format.
Measured on 2026-09-23: 13 skills installed by `npx skills` show up in no pane,
and the user menu promises "Packages — Skills, extensions, updates".

## Done means

1. **Report.** Every CLI has a **Skills** tab at `#/clis/<cli>/skills` in both
   apps. It lists what that CLI loads, per scope, with:
   - where each skill comes from and which copy wins;
   - provenance read from the locks;
   - an estimated context cost;
   - whether trust is needed, with the vendor's command to copy.
2. **Install.** Install, remove, update and check work for the **This machine**
   and **This workspace** scopes. They write the canonical `.agents/skills`
   folder, a link for the CLIs that do not read it, and the lock the `skills` CLI
   reads. A skill folder that no lock names is never overwritten.
3. **Toggles.** Each CLI that has a per-skill toggle gets a switch that writes
   the vendor's own key. A CLI without one (Pi, agy) shows one sentence instead.
4. **This agent.** An agent-scope list travels at launch through the CLI's own
   flag, and the launch preview shows it. When a guest's list changes while its
   terminal runs, the terminal is marked as needing a restart.
5. **Marketplace.** One catalog for every CLI, drawn from seed repositories,
   sources the user adds and `.well-known` domains. skills.sh can be switched on
   and is off by default.
6. **Outcomes.** Each launch records its skill set, and Outcomes can compare
   runs with and without a skill.

## Non-goals

- A PiCode "verified" mark, auto-consent, or trusting a workspace for the user.
- A PiCode package database. The skill folders, the two locks and each CLI's own
  files are the truth.
- Enterprise registries (JFrog, Google Cloud), administrator allow-lists and
  publishing skills.
- Managing plugins. Packages keeps them, under ADR-0167.
- Agent-authored skills. When they come, they need a review gate, like Hermes'
  `skills.write_approval`.

## The model

```go
// internal/skills
type Scope string // machine | workspace | agent

type Spec struct { // one per CLI, declarative (clipkgs/climemory shape)
    CLI             string
    ProjectDirs     []string // read order, relative to the workspace (walk rule per CLI)
    UserDirs        []string // read order, "~"-relative
    Trust           *TrustRule // nil, or the vendor's trust command and how to read its state
    Toggle          ToggleKind // none | claudeOverrides | codexConfig | opencodePermission | ompIgnored | grokDisabled | hermesDisabled | museVerb
    Launch          LaunchKind // none | piSkill | ompGlobs | claudePluginDir | hermesSkills | opencodeConfig
    Roster          []string   // vendor command that prints its own view, if any (live parity)
}

type Row struct {
    CLI, Name, Description, Scope, Path, Dir string
    Digest            string // computedHash-compatible sha256
    Loaded            bool   // this CLI loads it (false: shadowed, disabled, untrusted)
    ShadowedBy        string
    Enabled           *bool  // nil when the CLI has no toggle
    NeedsTrust        bool
    TrustCommand      string // the vendor's line, never run by PiCode
    Provenance        *Provenance // from skills-lock.json, ~/.agents/.skill-lock.json, Hermes hub lock
    LinkedFrom        []string    // other CLI folders that point here
    AlsoLoadedBy      []string    // other CLIs that read the same folder
    EstimatedTokens   int         // (len(name)+len(description))/4, vendor's number when printed
    Findings          []Finding   // advisory scan, never a verdict
}
```

Sources live in `internal/skills/source` (GitHub tarball via codeload, git via
`internal/gitclone` when git exists, `.tar.gz`/`.zip`, local folder,
`.well-known` 0.2.0, skills.sh). The catalog lives in `internal/skillcatalog`,
built on the `internal/mcpcatalog` pattern. Staging is `<data>/skills/stage/`,
the content cache is `<data>/skills/cache/<digest>/`, and the catalog cache is
`<data>/skills-catalog.json`, versioned. Sources the user adds are stored with
`SetSetting("skills.sources")`, and the skills.sh switch with
`SetSetting("skills.skillssh")`; both already emit `setting.updated`.

## Routes

| Route | Answers |
|---|---|
| `GET /api/skills/report?cli&scope&workspace&agent` | the rows above for one CLI |
| `POST /api/skills/preview` | fetch and stage a source: files, frontmatter, findings, audits |
| `POST /api/skills`, `DELETE /api/skills`, `POST /api/skills/update` | install, remove, update (`scope` machine, workspace or agent) |
| `GET /api/skills/updates` | locks compared with their sources |
| `POST /api/skills/toggle` | the CLI's own toggle |
| `GET /api/skills/catalog?q&skillssh=1`, `GET/POST/DELETE /api/skills/sources` | the federated catalog and the user's sources |

Each write publishes an ephemeral `skills.changed` event. The agent list uses
`agent.updated`.

## Per-CLI declaration (measured 2026-09-23; "measure" is a slice gate)

| CLI | Project | User | Toggle | Launch |
|---|---|---|---|---|
| pi | `.agents/skills`, `.pi/skills` | `~/.agents/skills`, `~/.pi/agent/skills` | none | `--skill <path>`; isolated `--no-skills` |
| omp | `.omp`, `.agents`, `.agent` (walk-up), `.claude`, `.github` | `~/.omp/agent/skills`, `~/.agents/skills`, `~/.agent/skills` (measured) | `skills.ignoredSkills` (project splice; user `omp config set`) | `--skills=<globs>`; `--no-skills` |
| claude-code | `.claude/skills` only → link | `~/.claude/skills` → link | `skillOverrides` | `--plugin-dir` synthetic plugin in the run dir (measure: manifest needed?) |
| codex | `.agents/skills` | `~/.agents/skills` | `[[skills.config]]` (new TOML array-of-tables primitive) | `-c` or `-p` (measure) |
| grok | `.grok`, `.agents`, `.claude` (trusted projects only, measured) | `~/.grok/skills`, `~/.agents/skills`, `~/.claude/skills` | `[skills] disabled` (TOML list) | measure |
| hermes | `.hermes`, `.agents` after `hermes skills trust` | `~/.hermes/skills` → link | `skills.disabled` (YAML list) | `--skills=<name>` (the `=` form: `term_intercept.go:620` does not know `-s`) |
| opencode | `.opencode`, `.agents`, `.claude` | `~/.config/opencode/skills`, `~/.agents/skills` | `permission.skill` (JSONC splice) | `OPENCODE_CONFIG_CONTENT` with `permission.skill` |
| muse | `.agents`, `.claude` in trusted workspaces, above the user folders (measured) | `~/.agents/skills`, `~/.claude/skills` | `muse skills enable/disable --scope` | none |
| agy | `.agents/skills` | `~/.gemini/antigravity-cli/skills` → link | none | none |

## Slices

Each slice is one branch and one session. Iterate with `make ci-scoped`, finish
with `make close`, write the closing docs from `make close-summary` in a
subagent, then fast-forward and run `make ci` on `main`.

| # | Target | Change | Gate |
|---|---|---|---|
| 0 | `feat/skills-adr` | this plan, ADR-0196, the study, the open topic | `make docs-check`, `make vale` |
| 1 | `feat/skills-inventory` | read-only: `internal/skills` (declarations, reader, digest, shadowing, estimate, provenance, trust state); `GET /api/skills/report`; `skills` pane registered (`cliLaunch.js`, `CliPaneTabs.jsx`, `AgentClis.jsx`, both apps); Installed read-only with empty and error states; Packages menu copy corrected; `slashres` reads through the new reader; measure the "measure" folders | table tests with a temp HOME and an empty PATH; JS↔Go list parity; `cliLaunch.test.js`, `application-routes.test.mjs`; live `PICODE_SKILLS_LIVE=1` against `muse skills list --json`, `hermes skills list`, `grok inspect --json`; visual-review; `docs/architecture/skills.md`, `docs-site/guide/skills.md`, routes.md, `make openapi` |
| 2 | `feat/skills-install` | sources, stage, spec validation, advisory scan, preview dialog (`ResponsiveDialog`/`MobileSheet`, Zod, `noValidate`), install/remove/update/check for machine and workspace, both locks, links with copy fallback | every install/remove/update row below; hash vectors from the real `npx skills`; extraction refuses `..`, absolute paths, links and devices; SSRF guard; 10 MiB / 25 MiB / 1000-file limits; 409 on a stale lock |
| 3 | `feat/skills-toggles` | Claude `skillOverrides`, Codex `[[skills.config]]`, OpenCode `permission.skill`, Omp `skills.ignoredSkills`, Grok `[skills] disabled`, Hermes `skills.disabled`, Muse `muse skills disable/enable`; Pi and agy get the declaration's sentence | golden files per format (comments and bytes preserved); `TestEveryToggleIsARealSettingsKey` extended; live read-back through each vendor roster |
| 4 | `feat/skills-agent` | migration `070_agent_skills.sql`, `SetAgentSkills` + event, `scope=agent`; digest-addressed cache; launch injection per the table; launch-plan preview; the restart fingerprint also covers guests (`applyTerminalLaunch`); "Try in this agent" | `TestEveryMutationAppendsAnEvent`, `mutation_coverage_test.go`; the launch rows below; argv per CLI; each flag checked against `--help` |
| 5 | `feat/skills-marketplace` | `internal/skillcatalog`: seed repositories (confirmed in the slice), the user's sources, `.well-known` domains, opt-in skills.sh with its audits; the Marketplace tab; source management | httptest suite on the `mcpcatalog` pattern; a skills.sh failure is one line |
| 6 | `feat/skills-outcomes` | `Snapshot.Skills` (name, digest, scope, via) on each launch → `ExitConfig` → Outcomes groups with/without; lifecycle (trying → in project → disabled → removed) with **Promote to project**; optional `claude plugin eval --ablation` job with `--max-cost-usd` | aggregation tests; QA with two agents on one task |
| 7 | `feat/skills-doctor` (if debts remain) | locally modified skills (diff, restore), broken links, diverged copies, per-account homes | — |

## Decision table

Every row is a test in the slice that ships it.

| Verb | Condition | Action |
|---|---|---|
| install | free name, valid spec, digest matches | stage → preview → consent → atomic rename → links → lock → `skills.changed` |
| install | same name, same digest | no write; "already installed" |
| install | same name, other digest, in a lock | becomes an update (shows the diff) |
| install | same name, in no lock | refuse; offer **Adopt** (lock only) or replace with explicit confirmation |
| install | `name` ≠ folder, no `description`, or over the limits | refuse, naming the rule |
| install | `.well-known` digest mismatch | refuse: corrupted or tampered content |
| install | critical scan finding or critical audit | explicit second confirmation; never in bulk |
| install | the CLI needs trust (Hermes, Muse) | install; show the vendor's command with **Copy command**; never trust for the user |
| install | the CLI's terminals are running | install; "applies at the next start" (Claude Code reloads live) |
| install | the link cannot be created | copy, recorded as a copy in the lock |
| any | the lock or file changed since the read | 409; read again |
| remove | in a lock, digest unchanged | remove folder, links and entry |
| remove | locally modified | confirmation showing the diff |
| remove | in no lock | only with confirmation: "not installed by PiCode or by skills" |
| remove | used by agents | name the agents that lose it at their next start |
| update | new digest at the source, no local change | update and write the new hash |
| update | local change | do not overwrite; offer the diff or force |
| update | source unreachable | the reason on the row; never "all up to date" |
| toggle | the CLI has a toggle | write the vendor's key at the row's scope, read the roster back |
| toggle | Pi or agy | no control; the declaration's sentence |
| launch | the CLI has an injection and the skill is cached | flag in argv, shown in the preview, recorded in the snapshot |
| launch | the CLI has no injection | no agent scope offered; the row says why |
| launch | the skill is gone from the cache | start without it; the agent card says so |
| launch | isolated agent (Pi, Omp) | `--no-skills` plus the agent's own |
| launch | the list changed while a terminal runs | "restart needed", guests included |

## Risks

- **Digest interop.** `computedHash` orders paths with JavaScript's
  `localeCompare`, and Go must reproduce that. Vectors from the real CLI are
  slice 2's gate.
- **Lock versions.** Vercel's CLI discards a global lock whose version is older
  than its own. PiCode writes only the version it read and never migrates one.
- **Links on Windows** or with `core.symlinks=false`: fall back to a copy that
  is recorded in the lock; slice 7 flags drift.
- **Vendor drift.** Folders and keys move every week. The declaration and a live
  test per CLI catch it, and an unknown shape fails loudly.
- **Trust.** Never automated. Muse's `--trust-workspace` is used for reading
  only.
- **Shadowing.** Each CLI has its own order, and the 13 skills on this machine
  are the first fixture.
- **Per-account homes.** `clicreds` links `skills` back to the real folder;
  slice 1 confirms that the reader follows the effective home.

## Owner calls

- **Slice 2's digest gate.** If Go cannot match `computedHash`, choose between
  a PiCode digest that is marked as such and not writing Vercel's locks.
- **Slice 5's seed sources.** The list is proposed in the slice and approved by
  the owner.
- **A later workspace view.** If the omp study's Project pane or the AGENTS.md
  study's workspace tab is approved, decide whether Skills moves there.

## What would make us stop

- Slice 1's live parity cannot reproduce Muse's, Hermes' or Grok's own roster.
- The digest does not interoperate and the owner prefers not to write Vercel's
  locks.
- A vendor requires interactive trust even to read.
