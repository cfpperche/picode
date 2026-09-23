# Skills (ADR-0196)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per
> subsystem). Edit here; the index only links.

The Skills tab (`#/clis/<cli>/skills`) reports the Agent Skills
(agentskills.io) a CLI loads: every `SKILL.md` folder in the places that CLI
reads, in its own precedence order, which copy wins, who installed it, and what
it costs at start. Slice 1 of [the plan](../plans/skills.md) is read-only;
installing, toggles, per-agent sets and the marketplace are the next slices.

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
draws the same rows as a list. Scope radios (All / the workspace / This
machine), a filter, a one-line summary and, when workspace skills wait on
trust, the vendor's command with **Copy**. An empty pane is one line and
**Check again**; the report reloads when the window regains focus, because
skill folders change outside PiCode.

## Live parity

`internal/skills/live_test.go` runs the real `muse` against a fixture in a
sandbox HOME (`PICODE_SKILLS_LIVE=1`, `PICODE_LIVE_SANDBOX` equal to HOME) and
requires Muse's loaded skills and the reader's to be the same names from the
same folders. Grok (`grok inspect --json`) and Hermes (`hermes skills list`)
were compared by hand on 2026-09-23; their differences are vendor rules the
reader does not model yet (`docs/handoff/open/skills.md`).
