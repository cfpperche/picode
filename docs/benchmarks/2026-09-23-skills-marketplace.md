# Study: a skills marketplace for nine agent CLIs — what Packages covers, what the CLIs read, and what the market built

**Date**: 2026-09-23 · **Asked by**: the owner. The request was an easy way for
users to control the lifecycle and experimentation of the skills the agent CLIs
use, starting with a marketplace for installing skills into projects, and first
to find out whether Packages already covers it.
**Decision**: [ADR-0196](../decisions/0196-skills-for-every-agent-cli.md); plan
in [`docs/plans/skills.md`](../plans/skills.md).
**Method**: all nine CLIs are installed here. Their skill commands and flags were
read from `--help` at the versions in §3, and rosters were read where a CLI
prints one (`muse skills list --json`, `claude plugin list --json`). Vendor
documentation and third-party reports are marked as such.

## 1. The answer

**Packages does not cover it.** Packages reaches a skill only when the skill
ships inside one CLI's package or plugin:
- Pi packages can carry skills; the gallery labels a hit `kind: skill`
  (`internal/pipkg/gallery.go:193`), and an entry can filter them
  (`internal/pipkg/pipkg.go:177`).
- Claude, Codex, Grok, Muse, agy and Omp plugins can bundle a `skills/` folder.

A **standalone** skill is invisible. That is a folder with a `SKILL.md`, which
is how most skills are distributed. It is not listed, not installable, and has
no enable/disable, versions, updates, audits, per-agent choice or measurement.

The receipt:
- 13 skills installed by Vercel's `npx skills` live in `~/.agents/skills`
  (lock `~/.agents/.skill-lock.json`, version 3, source
  `heygen-com/hyperframes`), with copies in `~/.claude/skills`.
- `muse skills list --json` reports all 13, and marks the `.claude` copies
  `skill-shadowed`.
- `claude plugin list --json` reports none of them, so neither does the Claude
  Code Packages pane.
- The user menu still says "Packages — Skills, extensions, updates"
  (`web/browser/src/lib/userMenuModel.js:18`,
  `web/mobile/src/lib/moreMenuModel.js:19`).

What exists for skills today:
- the managed-Pi composer's `/skill:` picker, which lists and nothing else
  (`internal/slashres`);
- the all-or-nothing `--no-skills` for isolated Pi and Omp agents
  (`internal/store/agents.go:353`, `internal/server/cli_launch.go:1113`).

## 2. The format

Agent Skills has been an open standard since 2025-12-18 (agentskills.io,
stewarded through the Agentic AI Foundation). About 40 products support it
(June 2026 ecosystem report).

A skill is a directory whose `SKILL.md` has YAML frontmatter and a Markdown
body:
- `name`: 1–64 characters, lowercase letters, digits and single hyphens; it
  must match the folder name.
- `description`: 1–1024 characters.
- optional fields: `license`, `compatibility` (≤ 500 characters), `metadata`
  (a string map) and the experimental `allowed-tools`.
- optional folders: `scripts/`, `references/` and `assets/`.

Loading is progressive: only name and description sit in context until the skill
is used.

ADR-0167 refused a common package backend because it "would have to invent
common install semantics". For skills those semantics already exist.

## 3. What the nine CLIs read (measured 2026-09-23)

| CLI (version) | Project | User | Toggle | Per agent at launch | Own manager |
|---|---|---|---|---|---|
| Pi 0.87.1 | `.agents/skills` (ancestors), `.pi/skills` | `~/.agents/skills`, `~/.pi/agent/skills` | none (a package entry can filter its skills) | `--skill <path>` (adds, repeatable); `--no-skills` | `pi install` packages |
| Omp 18.2.11 | `.omp/skills` (ancestors), `.agents/skills`, `.claude/skills`, `.github/skills` | `~/.omp/agent/skills` | `skills.ignoredSkills`, `skills.includeSkills`, `disabledExtensions: [skill:<n>]` | `--skills=<globs>` (filters); `--no-skills` | plugins |
| Claude Code 2.1.280 | **only** `.claude/skills` (+ nested, `--add-dir`) | `~/.claude/skills` | `skillOverrides` (`on`, `name-only`, `user-invocable-only`, `off`); `permissions` `Skill(name)`; `disableSkillShellExecution` | `--plugin-dir <dir>` (session plugin; skills get a namespace); `--disable-slash-commands` turns all off | plugins + marketplaces; `plugin eval`; `/skill-doctor` |
| Codex 0.156.1 | `.agents/skills`, cwd up to the repo root | `~/.agents/skills`; `/etc/codex/skills` | `[[skills.config]] path = … enabled = false` in `~/.codex/config.toml` (docs) | none measured; `-c key=value` and `-p <profile>` exist | `$skill-installer` (from `openai/skills`); plugins |
| Grok 1.0.41 | `.grok/skills`, `.agents/skills`, `.claude/skills`, cwd up to the repo root | `~/.grok/skills`, `~/.claude/skills` | `[skills] disabled = [names]`, `ignore`, `paths` in `~/.grok/config.toml` (bundled guide) | none in `--help` (the docs mention `--plugin-dir`) | plugins + marketplace; `grok inspect --json` |
| Hermes 0.21.4 | `.hermes/skills`, `.agents/skills`, only after `hermes skills trust` | `~/.hermes/skills`, `skills.external_dirs` | `skills.disabled` (and `skills.platform_disabled.<p>`) in `config.yaml`; `hermes skills config` is interactive only | `--skills` / `-s <name>` (preloads) | full hub (§4) |
| OpenCode | `.opencode/skills`, `.agents/skills`, `.claude/skills` up to the worktree | `~/.config/opencode/skills`, `~/.agents/skills`, `~/.claude/skills` | `permission.skill` (`allow` / `deny` / `ask` by pattern; per agent) (docs) | `OPENCODE_CONFIG_CONTENT` can carry `permission.skill` | none |
| Muse Code 1.3.0 | trusted workspaces only; `--trust-workspace` trusts one run and saves nothing | `~/.agents/skills` (wins over `~/.claude/skills`) | `muse skills enable/disable/user-only --scope` | none | `muse skills list/inspect/validate/install <path>/import --from claude\|codex/update/uninstall`, `--json` with a per-skill `context_cost` |
| Antigravity (`agy` 1.2.9) | `.agents/skills` (legacy `.agent/skills`) | `~/.gemini/antigravity-cli/skills` | none documented | none | plugins |

Four things follow from the table:
1. **One project folder serves most CLIs.** `.agents/skills` is read by seven
   of the nine. Claude Code needs `.claude/skills`, and Muse's project folder
   was not measured because reading it needs trust. A workspace install is
   therefore one canonical folder, one link and a lock — what `npx skills`
   already does.
2. **Per-agent skill sets work without touching the repository** for Pi, Omp,
   Claude Code and Hermes, and for OpenCode by filtering. No tool we found does
   this across CLIs.
3. **Trust is the user's act.** Hermes and Muse will not read project skills
   until the user trusts the workspace.
4. **Two CLIs cannot exclude a skill.** Pi and agy have no per-skill toggle;
   the surface must say so instead of drawing a switch.

## 4. Marketplaces, installers, registries

| Who | What it does | What to take |
|---|---|---|
| **skills.sh + `npx skills`** (Vercel, January 2026) | directory of hundreds of thousands of skills; installs to 75+ agents (seven of ours: antigravity, claude-code, codex, grok, hermes-agent, opencode, pi); canonical copy + a link per agent, `--copy` fallback; `add`/`list`/`find`/`remove`/`update`/`init`; Gen, Socket and Snyk audits on every skill page since 2026-02-17, shown before install since `skills@1.4.0` | the canonical-plus-links layout; both locks (§5); audits shown as theirs. Telemetry is on by default (`DISABLE_TELEMETRY=1`). The v1 API needs a Vercel OIDC token; `/api/search` is undocumented |
| **Hermes Skills Hub** (0.21.4, measured) | `browse`/`search`/`install`/`inspect`/`list`/`check`/`update`/`audit`/`uninstall`/`reset`/`diff`/`tap`/`publish`/`snapshot`/`trust`; sources: official, skills.sh, GitHub, `.well-known`, ClawHub, LobeHub, browse.sh, URL; trust tiers `builtin` / `official` / `trusted` / `community`; a scan on install that `--force` cannot override for `dangerous`; a write-approval gate for skills the agent authors | the lifecycle verbs, the tiers, the hash in the lock, `check` separate from `update`, and agent-authored skills staged for approval |
| **Claude Code plugin marketplaces** | `marketplace.json` in git; sources: relative, GitHub, git URL, git-subdir, npm, archive; pin by `ref`/`sha`; `extraKnownMarketplaces` for teams, `strictKnownMarketplaces` / `blockedMarketplaces` for administrators; `claude plugin details` reports token cost | pinning by SHA; admin allow-lists as a later need |
| **Codex** | `$skill-installer` from `openai/skills` (system, curated and experimental tiers) | a seed source |
| **xAI** | `xai-org/plugin-marketplace`; the TUI Marketplace tab | — |
| **Discovery RFC** (Cloudflare, schema 0.2.0) | `/.well-known/agent-skills/index.json`; entries `name`, `description`, `type` (`skill-md` or `archive`), `url`, `digest` (`sha256:<hex>`); the client **must** verify the digest | a decentralised source with integrity built in |
| **Enterprise registries** | JFrog Agent Skills Registry; Google Cloud Skill Registry | out of scope; the lock and sources model leaves room |

## 5. Lock files (read from source and from this machine)

- **Project `skills-lock.json`, version 1** (`vercel-labs/skills`
  `src/local-lock.ts`).
  - Shape: `{version, skills: {<name>: {source, sourceUrl?, ref?, sourceType,
    skillPath?, computedHash, wellKnownDigest?}}}`.
  - Sorted by name and timestamp-free, so two branches that add different skills
    merge cleanly. It is meant to be committed.
  - `computedHash` is the sha256 of every file's relative path followed by its
    content, with files sorted by JavaScript `localeCompare`, skipping `.git` and
    `node_modules`.
- **Global `~/.agents/.skill-lock.json`, version 3** (a real one on this
  machine).
  - Shape: `{version, skills: {<name>: {source, sourceType, sourceUrl,
    skillPath, skillFolderHash, pluginName, installedAt, updatedAt}}}`.
  - `skillFolderHash` is a GitHub tree SHA, not a content hash.
  - The CLI discards a lock whose version is older than its own.
- **Hermes `~/.hermes/skills/.hub/lock.json`**: source URL, content hash,
  scanner version, findings and timestamp (vendor docs).

## 6. The security record

- **ClawHavoc** (Koi Security and Antiy CERT, January 2026): about 1,184
  malicious skills on ClawHub, roughly one in five at the time. They used
  typosquatted names and bot-inflated downloads, and delivered infostealers
  (AMOS on macOS).
- **Snyk ToxicSkills** (2026-02-05): of 3,984 skills from ClawHub and skills.sh,
  36.82% had at least one flaw and 13.4% a critical issue. 76 payloads were
  confirmed malicious by human review.
- **Mitigations the market converged on:**
  - third-party audits on the card (skills.sh);
  - a scan on install that cannot be forced past `dangerous` (Hermes);
  - trust before project skills load (Hermes, Muse);
  - a mandatory digest (discovery RFC);
  - SHA pinning (Claude marketplaces);
  - administrator allow-lists (Claude managed settings).

The barrier to publishing a malicious skill is a `SKILL.md` and an account.

## 7. Experimentation

- **SkillsBench** (arXiv 2602.12670): curated skills raised pass rates by about
  16 points on average, but 16 of 84 tasks got worse. Skills the model wrote for
  itself averaged −1.3 points. A skill has to be tried, and an agent-authored
  skill needs review.
- **Claude Code `claude plugin eval --ablation with-without`**: runs a plugin's
  eval cases with and without it and reports the score delta. Skills-dir plugins
  resolve as `<name>@skills-dir`, and `--max-cost-usd` caps the spend.
  `/skill-doctor` lists context cost and invocation counts per skill.
- **Muse** reports `context_cost` (startup bytes and estimated tokens) for every
  skill in `--json`.
- **OpenAI**, "Testing Agent Skills Systematically with Evals": checks for
  outcome, process (did the skill fire) and style.

PiCode's own lever is that it launches the agents and already records how they
end (ADR-0194). The skill set a launch carried is the missing column.

## 8. What PiCode adapts

| Pattern | From | PiCode adaptation |
|---|---|---|
| One catalog for every CLI, written through per-CLI drivers | ADR-0157 (Connectors marketplace) | the Skills Marketplace; Installed \| Marketplace like Packages and Connectors |
| Canonical folder + links + a committed lock | `npx skills` | same folders, same lock formats, so either tool can update what the other installed |
| Lifecycle verbs, trust tiers, hash in the lock, scan on install | Hermes hub | check / update / audit, advisory scan, provenance per row; tiers are the source's, never a PiCode mark |
| Audits on the card | skills.sh | shown with the auditor's name, only when the user opted into skills.sh |
| Digest-verified decentralised sources | discovery RFC | `.well-known` as a source kind; mismatch refuses |
| Per-skill context cost | Muse, Claude `/skill-doctor` | an estimate on every row; the vendor's number where it prints one |
| With-and-without comparison | Claude `plugin eval`, SkillsBench | the launch snapshot records the skill set, and Outcomes groups by it; `plugin eval` as an optional job with a cost cap |
| Everything a CLI reads from a workspace, per CLI | VS Code Agent Customizations, the omp study's Project pane, the 2026-09-23 AGENTS.md study | a per-CLI tab now; the same report can feed a workspace view if one is approved |

## 9. Decisions (owner, 2026-09-23)

1. Amend ADR-0167 for skills: cross-CLI install and a federated catalog are
   allowed; PiCode never vouches.
2. Engine: native Go, with locks compatible with the `skills` CLI; Node is not
   required.
3. Placement: a Skills tab per CLI (`#/clis/<cli>/skills`).
4. skills.sh: opt-in and off by default; the switch says the query goes to
   Vercel.

Recorded in ADR-0196. The slices are in [`docs/plans/skills.md`](../plans/skills.md).

## Sources

- Agent Skills specification — https://agentskills.io/specification
- Agent Skills ecosystem report 2026 — https://agentman.ai/blog/agent-skills-ecosystem-report-2026
- vercel-labs/skills (README, `src/local-lock.ts`, `src/skill-lock.ts`) — https://github.com/vercel-labs/skills ; lock bug — https://github.com/vercel-labs/skills/issues/542
- skills.sh — https://www.skills.sh/ ; audits — https://vercel.com/changelog/automated-security-audits-now-available-for-skills-sh ; API — https://www.skills.sh/docs/api
- Claude Code skills — https://code.claude.com/docs/en/skills ; marketplaces — https://code.claude.com/docs/en/plugin-marketplaces
- Codex skills — https://learn.chatgpt.com/docs/build-skills
- OpenCode skills — https://opencode.ai/docs/skills
- Grok skills, plugins and marketplaces — https://docs.x.ai/build/features/skills-plugins-marketplaces (and the guide Grok ships, `docs/user-guide/08-skills.md`)
- Antigravity skills — https://antigravity.google/docs/skills/
- Hermes skills — https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/features/skills.md
- Pi skills — https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/skills.md
- Agent Skills discovery RFC — https://github.com/cloudflare/agent-skills-discovery-rfc
- Snyk ToxicSkills — https://snyk.io/blog/toxicskills-malicious-ai-agent-skills-clawhub/ ; HKCERT on ClawHub — https://www.hkcert.org/blog/openclaw-s-rapid-adoption-exposes-skills-supply-chain-and-fake-installer-risks-in-a-high-privilege-ai-agent-platform ; threat taxonomy — https://arxiv.org/pdf/2604.02837
- SkillsBench — https://arxiv.org/abs/2602.12670
- OpenAI, Testing Agent Skills Systematically with Evals — https://developers.openai.com/blog/eval-skills
