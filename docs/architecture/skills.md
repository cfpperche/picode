# Skills (ADR-0196)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per
> subsystem). Edit here; the index only links.

The Skills tab (`#/clis/<cli>/skills`) reports the Agent Skills
(agentskills.io) a CLI loads: every `SKILL.md` folder in the places that CLI
reads, in its own precedence order, which copy wins, who installed it, and what
it costs at start. Slices of [the plan](../plans/skills.md):

- **Slice 1** reads.
- **Slice 2** installs, removes, checks and updates.
- **Slice 3** switches one skill off in the CLI's own settings.
- **Slice 4** gives an agent its own skills.
- **Next:** the marketplace.

## The declaration

`internal/skills/specs.go` holds one `Spec` per CLI — the shape of the
`internal/clipkgs` specs and `internal/climemory`. A spec lists `Roots` in
precedence order (the first root that has a name wins), each with its scope
(`machine`, a `~/` path, or `workspace`, relative to the workspace),
`WalkUp` (read in every parent up to the repository root, nearest first) and
`Recursive` (a `SKILL.md` at any depth, for Hermes' categories and Grok's
synced folders). `ProjectTrust`, `TrustCommand` and `TrustNote` state the CLI's
gate on workspace skills; `Toggle` and `Launch` state, in words, the vendor
mechanism the later slices write through.

| CLI | Precedence (measured 2026-09-23) |
|---|---|
| pi | `~/.pi/agent/skills`, `~/.agents/skills`, `.pi/skills`, `.agents/skills` (walk-up); workspace skills need Pi's trust |
| omp | `.omp`, `.agents`, `.agent` (walk-up), `.claude`, `.github`, then `~/.omp/agent/skills`, `~/.agents/skills`, `~/.agent/skills` |
| claude-code | `~/.claude/skills` over `.claude/skills` (personal outranks project); never `.agents/skills` |
| codex | `.agents/skills` (walk-up), `~/.agents/skills`, `/etc/codex/skills` |
| grok | `.grok`, `.agents`, `.claude` (walk-up, trusted projects only), `~/.grok`, `~/.agents`, `~/.claude` (recursive) |
| hermes | `.hermes`, `.agents` (after `hermes skills trust`), `~/.hermes/skills` (recursive) |
| opencode | `.opencode`, `.claude`, `.agents` (walk-up), `~/.config/opencode/skills`, `~/.claude/skills`, `~/.agents/skills` |
| muse | `.agents`, `.claude` (trusted workspaces), `~/.agents/skills`, `~/.claude/skills` |
| agy | `.agents/skills`, `.agent/skills`, `~/.gemini/antigravity-cli/skills` |

`TestSkillsJSListMatchesTheDeclarations` keeps the list of CLIs equal in Go
and in `web/shared/domain/cliSkills.js`.

## The reader

`skills.Read(Query)` resolves the roots, lists `<root>/<name>/SKILL.md` (or any
depth for a recursive root; a linked skill folder is followed and marked), and
builds one `Row` per folder:

- **Frontmatter** is YAML (`gopkg.in/yaml.v3`). A header that is not YAML — an
  unquoted description with `: ` inside — is read line by line, as the CLIs
  do, and the row says so. Spec problems (name rules, a name that does not
  match its folder, a missing or over-long description) are listed; a folder
  with no header or no description is `invalid`, since the CLIs skip it.
- **Status**: `loaded`, `shadowed` (with the root that wins), `needs-trust`
  (the CLI's own record says the workspace is not trusted — only Pi's
  `trust.json` is readable), `if-trusted` (a CLI whose trust PiCode cannot
  read), `invalid`.
- **Digest**: `Digest` is Vercel's `computedHash` — sha256 over each regular
  file's slash-separated relative path then its bytes, `.git` and
  `node_modules` skipped, files ordered as JavaScript's `localeCompare` orders
  them. `localeLess` reproduces that order for ASCII (ICU root collation,
  punctuation not ignorable, lowercase before uppercase as a tertiary
  difference); `TestLocaleOrderMatchesNode` and `TestDigestMatchesTheSkillsCLI`
  pin both against Node where it is installed. Folders over 1,000 files or
  25 MiB get no digest.
- **Provenance** joins three locks PiCode reads but does not own: the
  workspace's `skills-lock.json` (v1; `Modified` when the folder no longer
  matches its `computedHash`), `~/.agents/.skill-lock.json` (v3) and the Hermes
  hub's `~/.hermes/skills/.hub/lock.json` (keyed by install path). A lock that
  cannot be parsed is a note on the report, never an empty list.
- **Cost**: `(len(name) + len(description)) / 4` tokens, the part every
  session carries before the skill is used.
- **Also loaded by**: the other CLIs whose resolved roots include the same
  folder.

## The route and the pane

`GET /api/skills/report?cli&workspace` (`internal/server/skills.go`) answers the
report; an unknown CLI is a 400 naming the declared ones. Pi's trust comes from
`pisettings.Trusted`.

The pane is `CliSkills.jsx` in each app, mounted by `AgentClis.jsx` for
`pane === "skills"`; the address, the words and the filters live in
`web/shared/domain/cliSkills.js` (`cliSkillsHash`, `skillStatus`,
`skillOrigin`, `visibleSkills`, `trustLine`), tested in `cliSkills.test.js`.
The desktop draws a table (skill, status, source, folder, cost at start) whose
rows open to the folder, the reason and the other CLIs that read it; the phone
draws the same rows as a list. Scope chips in the setup panes' order —
**Global**, the workspace by name, and, where the spec declares `AgentScope`
(Pi, Omp: their launch already carries an agent's packages), the agent by name,
which shows what that agent loads or says it runs isolated; the report names
the agent only when it is this CLI's and in that workspace — a filter, a one-line summary and, when workspace skills wait on
trust, the vendor's command with **Copy**. An empty pane is one line and
**Check again**; the report reloads when the window regains focus, because
skill folders change outside PiCode.

## Installing (slice 2)

`skills.Manager` (`internal/skills/install.go`) is PiCode's own writer; no
vendor command runs and the job lane is not involved (it only executes
programs, one job per machine).

- **Sources** (`source.go`): `owner/repo[/path][#ref]` or a GitHub URL is the
  codeload tarball (no git needed; the top folder is stripped); an `https://`
  site is its `/.well-known/agent-skills/index.json` (discovery schema 0.2.0:
  `skill-md` or `archive` entries, each checked against its `sha256:` digest,
  any mismatch refusing the whole source); an absolute or `~/` path is a local
  folder. Downloads go through a client that refuses private addresses at dial
  time (redirects included) and stay https. Extraction keeps regular files and
  folders only — links and devices are skipped and named in the notes — and
  refuses a path that leaves its folder or is absolute; limits are 50 MiB
  downloaded, 200 MiB unpacked, 20,000 files per source.
- **Preview** stages the source under `<data>/skills/stage/<id>` (30 minutes)
  and lists every `SKILL.md` folder with its spec problems, digest, files and
  the advisory scan (`scan.go`: pipe-to-shell, credential paths, paste and
  tunnel hosts, destructive deletes, compiled programs, invisible characters as
  critical; long encoded blobs and instruction overrides as warnings). A
  finding is shown, never a verdict.
- **Install** writes the canonical `<base>/.agents/skills/<name>` by copying to
  a sibling temp folder and renaming, then links the folders of CLIs that do
  not read it, derived from the declarations (`linkDirs`): `.claude/skills` in
  a workspace; `~/.claude/skills`, `~/.hermes/skills` and
  `~/.gemini/antigravity-cli/skills` on the machine, only for a CLI whose home
  folder exists. A relative symlink, or a copy where links fail. The lock is
  `skills-lock.json` v1 in the workspace or `~/.agents/.skill-lock.json` v3,
  read so that every other entry and field survives, re-read before the atomic
  write (409 `stale` if it moved), and never migrated (409 `lock-version`).
  Global entries PiCode writes carry `computedHash` beside the CLI's fields.
- **Refusals** are `skills.Conflict` codes the pane answers: `exists` (a folder
  no lock names: **Adopt** records it, **Replace** overwrites), `update` (in a
  lock with other content), `critical` (needs `acceptCritical`), `invalid`,
  `gone` (preview expired, 410), `modified` and `unlocked` (remove or update
  needs confirmation).
- **Remove** deletes the canonical folder, the links that point at it and the
  copies with its content, and the lock entry. **Update** refetches the entry's
  source, refuses to overwrite local edits without `force`, and rewrites the
  hash. **Check** fetches each source once and reports `current`, `behind`,
  `modified`, `unreachable` (with the reason) or `missing`.

Routes: `POST /api/skills/preview`, `POST /api/skills`, `DELETE /api/skills`,
`POST /api/skills/update`, `GET /api/skills/updates`; every write publishes the
ephemeral `skills.changed`. The pane's **Add skill** dialog
(`AddSkillDialog.jsx`, both apps; Zod `skillSourceSchema`) previews, shows the
findings and files, asks for the second confirmation on a critical finding and
turns a 409 into its choices; **Check for updates** marks rows, and a row in
`.agents/skills` offers **Update** and **Remove** with an inline confirmation.

## The agent scope (slice 4)

An agent's own skills are a list on its row (`agents.skills`, migration 073:
`{name, digest, source, dir}`), written by `Store.SetAgentSkills`, which
announces `agent.updated`. The content is a copy in PiCode's cache,
`<data>/skills/cache/<digest>/<name>`, written by `Manager.Install` with
`scope=agent` (`cache` in `install.go`): no lock, no link, nothing in the
workspace or the home folder. One copy serves every agent with that content.
The cache is never swept, so a restored agent (ADR-0205) finds its folders;
`ExitConfig.Skills` carries the list across the exit.

Each CLI receives the list its own way, measured on 2026-09-24 with the CLI's
own command list (`get_commands` over RPC, `claude plugin details`):

| CLI | At launch | Isolated agent | Same name as a folder skill |
|---|---|---|---|
| Pi | `--skill <folder>` per skill (`store.Agent.CLIFlags`) | `--no-skills` stays; `--skill` still loads | the folder's copy wins |
| Omp | `--config <data>/skills/agents/<id>/omp.yml` with `skills.customDirectories` (`agentOmpSkillFlags`) | the overlay turns every `skills.enable*` source off instead of `--no-skills`, which also drops custom folders | the agent's copy wins |
| Claude Code | `--plugin-dir <run>/picode-agent`, a copy of each skill under `skills/` (`writeClaudeAgentPlugin`); no manifest needed | — | none: plugin skills are `/picode-agent:<name>` |

Codex, Grok, Hermes, OpenCode, Muse and Antigravity have no per-launch way to
add a folder (Hermes `--skills` preloads an installed skill; OpenCode's
`permission.skill` filters), so their report has no agent and the pane says
so in one line. A skill whose cached folder is gone is left out of the launch
(`Agent.SkillDirs`) and reported as `missing`.

The reader (`agentRows`) lists the agent's skills with scope `agent` and root
`This agent only`, in the precedence above (`Spec.AgentOrder`). Routes:
`POST /api/skills` and `DELETE /api/skills` take `scope=agent` with `agent`;
`GET /api/skills/report?agent=` fills them in. The launch fingerprint of an
agent terminal now folds in the agent's own scope for every CLI
(`agentLaunchFingerprint`): Pi's flags as before, and for Omp and Claude Code a
marker of packages, isolation and skill digests, computed before any
injection, so the terminal view's "Launch changes pending" follows a skill
added or removed while it runs.

## Per-skill switches (slice 3)

`internal/skills/toggle.go` writes and reads the switch each vendor keeps.
Measured 2026-09-24 in a throwaway HOME, read back with each CLI's own
command:

| CLI | Key | Machine file | Workspace file | Read back with |
|---|---|---|---|---|
| Claude Code | `skillOverrides.<name> = "off"` (absent = on) | `~/.claude/settings.json` | `.claude/settings.local.json` (what `/skills` writes); local, then project, then user decides | the `init` line of `claude -p --output-format stream-json` |
| Codex | `[[skills.config]] path = "<abs>/SKILL.md" enabled = false` | `~/.codex/config.toml` | the same user file: a project layer's `skills.config` is ignored, and the path targets a workspace skill | `codex debug prompt-input` |
| OpenCode | `permission.skill.<name> = "deny"` (globs; the last matching rule wins) | `~/.config/opencode/opencode.json(c)` | `opencode.json(c)` or `.opencode/opencode.json(c)` | `opencode debug config` |
| Omp | `skills.ignoredSkills` (globs) | `~/.omp/agent/config.yml` | `.omp/config.yml`, which **replaces** the machine list, so the first workspace write copies it | `omp config get skills.ignoredSkills` |
| Grok | `[skills] disabled = [names]` | `~/.grok/config.toml` | machine only | `grok inspect --json` (`disabled`) |
| Hermes | `skills.disabled` | `~/.hermes/config.yaml` | machine only | `hermes skills list` |
| Muse | `muse skills disable\|enable <SKILL.md> --scope user\|project` | Muse writes `~/.config/muse/settings.json` itself | the same file, under `projects.<workspace>` | `muse skills list --json` (`activation`) |

Pi and Antigravity have no per-skill switch, and their pane says so in one
line.

- **Reading:** the reader sets `Row.Enabled` on every folder skill the CLI
  would load. A loaded skill whose switch is off gets the status `disabled`.
- **Writing:** every write goes through `clisettings.Doc`, so comments, key
  order and other keys keep their bytes. `SetScalar` handles a map entry and
  `SetStrings` a list. `AppendArrayTable` and `RemoveArrayTables` handle
  Codex's elements: PiCode appends and removes an element whole and never
  edits one in place.
- **Turning back on:** this removes PiCode's key. A file left holding only
  what PiCode's first write created is deleted, so the folder is as it was.
- **Read-back:** after each write the switch is read back. If a stronger layer
  or a user's glob still decides, the result names that file instead of
  claiming the switch moved.
- **Route:** `POST /api/skills/toggle {cli, scope, workspace, dir, enabled}`
  finds the row again in a fresh report, so it only ever writes a path the
  reader listed. It answers 409 on a stale file and announces
  `skills.changed`.
- **Pane:** a Radix switch on each row that has one. The switch flips at
  once, and the toast carries the read-back and a machine-only note.

## The Marketplace (slice 5)

`internal/skillcatalog` is one catalog for every CLI, on the
`internal/mcpcatalog` pattern: the cache on disk (`<data>/skills-catalog.json`,
versioned, one entry per source) answers first; a search starts a background
read for a source that has none or is a day old (a failure is retried after
30 minutes, keeps the last good list and records why); each read's end is the
ephemeral `skills.catalog` event, so an open pane refreshes from the feed.

- **Sources.** The seeds are `anthropics/skills`, `openai/skills` and
  `vercel-labs/agent-skills` (owner, 2026-09-24). The person's own live in the
  `skills.sources` setting (GitHub `owner/repo[/folder][#ref]` or an https site
  with `/.well-known/agent-skills/index.json`; a local folder is refused, it
  goes through Add skill). A GitHub source is one tree call to the API
  (unauthenticated, 60 an hour; the catalog makes one a day per source) and one
  raw `SKILL.md` per skill, header only (64 KiB), eight at a time, at most 300
  per source; hidden folders count (`openai/skills` keeps its set in
  `skills/.curated`). Measured 2026-09-24: 20 + 44 + 9 skills in 3.8 s.
- **skills.sh** is asked only when the `skills.skillssh` setting is on and the
  query has two characters or more (`/api/search`, undocumented, no token,
  measured 2026-09-24). Its answer has no description and no audits; the card
  names skills.sh with its install count and links to the page where skills.sh
  shows its audits. A failure is one line beside the cards.
- **Install** from a card opens `AddSkillDialog` on the card's `install`
  (`owner/repo/folder`, the repository for a skills.sh card, the site for an
  index) with the skill's name preselected: the preview, the scan and the
  consent are the ones a typed source gets. A card reads **Installed** only
  when a row of this CLI has that name *and* its lock names that source.
- **A name that differs from its folder** at the source installs under the
  name (`Candidate.Folder`, owner's call 2026-09-24: 5 of the 73 seed skills);
  the preview says so. An installed folder that does not match its name is
  still reported as a problem.

Routes: `GET /api/skills/catalog?q=` (cards, per-source state, the switch,
`skillsshError`), `POST`/`DELETE /api/skills/sources` (409 for a duplicate, 400
for a seed or a folder), `PUT /api/skills/skillssh`. All of it goes through
`skills.PublicClient` (https, private addresses refused).

## Outcomes and the lifecycle (slice 6)

Every agent terminal's launch records the skills it loaded
(`Snapshot.Skills`), and the exit keeps them (`docs/architecture/agent-exits.md`).
Outcomes shows, per skill, resolved among answered attempts with it and without
it — the headline's own measure — and marks a side with fewer than five
answered runs as *few runs*: a hint, not a finding.

The lifecycle is trying (an agent's own list) → in the project → off (the CLI's
switch) → removed. **Promote** (`POST /api/skills/promote {agent, name}`) is
the step from trying to the project: `Manager.Promote` stages the agent's
cached copy — the content it ran with, never a fresh download — installs it
into the agent's workspace like any workspace install (links, lock with the
original source so Check and Update reach it, the same 409s), then takes it
off the agent's own list. A free agent has no project to promote into. Not
built: `claude plugin eval --ablation` runs (they spend real tokens).

## Live parity

`internal/skills/live_test.go` runs the real `muse` against a fixture in a
sandbox HOME (`PICODE_SKILLS_LIVE=1`, `PICODE_LIVE_SANDBOX` equal to HOME) and
requires Muse's loaded skills and the reader's to be the same names from the
same folders. `TestLiveSkillsCLIHash` installs a fixture with the real
`npx skills` (telemetry off) and requires its `computedHash` to equal
`Digest` — measured equal on 2026-09-23, mixed-case file names included. `internal/skillcatalog/live_test.go` (`PICODE_SKILLS_LIVE=1`) reads the three seeds and one skills.sh search. Grok (`grok inspect --json`) and Hermes (`hermes skills list`)
were compared by hand on 2026-09-23; their differences are vendor rules the
reader does not model yet (`docs/handoff/open/skills.md`).
