# ADR-0196: Skills for every agent CLI — one reader over each CLI's own skill folders, installs that write those folders and Vercel-compatible locks, native toggles, per-agent sets at launch, and a federated catalog that never vouches

- **Status**: accepted (owner approved the plan, 2026-09-23)
- **Date**: 2026-09-23
- **Boundary**: persistence — PiCode writes skill folders into the user's home
  and workspace (`.agents/skills`, plus links in `.claude/skills` and the other
  CLIs' own folders), the project `skills-lock.json`, the global
  `~/.agents/.skill-lock.json`, each CLI's own toggle key, and a new
  `agents.skills` column. process — PiCode fetches third-party content (GitHub
  tarballs, git, archives, `.well-known` indexes, and skills.sh search when the
  user turns it on) and stages it in the data dir. security model — who consents
  to a skill's instructions and scripts: PiCode shows provenance, digests,
  third-party audits and an advisory scan, never vouches, and never trusts a
  workspace on the user's behalf.
- **Amends**: ADR-0167, for skills only: its refusal of a PiCode cross-CLI
  catalog and of cross-CLI transfer. Plugins keep ADR-0167 unchanged.
  **Builds on**: ADR-0157 (curated connector catalog), ADR-0176 (package
  model), ADR-0194 (exit records).
- **Study**: [2026-09-23 — a skills marketplace for nine agent CLIs](../benchmarks/2026-09-23-skills-marketplace.md)
  · **Plan**: [docs/plans/skills.md](../plans/skills.md)

## Context

Agent Skills has been an open standard (agentskills.io) since 2025-12-18. A skill
is a folder with a `SKILL.md`: YAML frontmatter (`name`, `description`) and
Markdown instructions, optionally with `scripts/`, `references/` and `assets/`.
All nine CLIs PiCode launches read this format (measured on the installed
versions, 2026-09-23).

PiCode reaches a skill only when it ships inside one CLI's package or plugin
(ADR-0176). Standalone skills are invisible. On this machine, 13 skills installed
by Vercel's `npx skills` live in `~/.agents/skills`, with copies in
`~/.claude/skills`, and no pane shows them; `claude plugin list --json` does not
list them either. The user menu still promises "Packages — Skills, extensions,
updates".

What the nine CLIs do is uneven, and the unevenness is what a surface has to
state:

| Fact (measured) | Consequence |
|---|---|
| `.agents/skills` in a project is read by seven of the nine; Claude Code reads only `.claude/skills`; Muse's project folder needs workspace trust to be read | one workspace install can serve every CLI, if Claude Code gets a link |
| user folders differ: `~/.agents/skills` (Pi, Codex, OpenCode, Muse), `~/.claude/skills`, `~/.grok/skills`, `~/.hermes/skills`, `~/.gemini/antigravity-cli/skills`, `~/.omp/agent/skills` | a machine install is one canonical folder plus links |
| toggles are each CLI's own: Claude `skillOverrides`, Codex `[[skills.config]]`, OpenCode `permission.skill`, Omp `skills.ignoredSkills`, Grok `[skills] disabled`, Hermes `skills.disabled`, Muse `muse skills disable`; Pi and agy have none | excluding a CLI is a vendor write, and two CLIs cannot exclude |
| launch flags: Pi `--skill <path>`, Omp `--skills=<globs>`, Claude Code `--plugin-dir`, Hermes `--skills`; OpenCode takes `permission.skill` through `OPENCODE_CONFIG_CONTENT` | a per-agent skill set needs no repository change for five CLIs |
| Hermes loads project skills only after `hermes skills trust`; Muse only in a trusted workspace | trust is the user's act; PiCode can only show the command |

The market already has the pieces:
- **Vercel's `skills` CLI** keeps a canonical copy with links per agent,
  `skills-lock.json` v1 in the project and a v3 lock in `~/.agents`, and shows
  Gen, Socket and Snyk audits.
- **Hermes' hub** has trust tiers, a scan on install and a lock that records the
  content hash.
- **Claude's marketplaces** pin by SHA.
- **The `.well-known/agent-skills` discovery RFC** makes a sha256 digest
  mandatory.

The security record is why consent must be explicit: ClawHavoc planted about
1,184 malicious skills on ClawHub (January 2026), and Snyk's ToxicSkills found a
flaw in 36.8% of 3,984 skills. SkillsBench found that curated skills raise pass
rates on average but made 16 of 84 tasks worse. Skills therefore need to be
tried before they are trusted.

ADR-0167 refused a PiCode package backend and a PiCode-curated cross-CLI catalog
for two reasons: plugins share no semantics ("it would have to invent common
install semantics"), and a curated catalog "would make PiCode vouch for other
vendors' plugin code". The first reason does not hold for skills, because the
Agent Skills spec *is* the common semantics. The second still holds; a federated
catalog that shows provenance and other parties' audits, with no PiCode verdict,
answers it.

## Decision

Skills are one subsystem, `internal/skills`. Each CLI has a declaration: its
project and user folders in its own read order, its trust requirement and the
vendor's trust command, its toggle mechanism, its launch injection and its roster
command. This is the shape of the `internal/clipkgs` specs and
`internal/climemory`.

The **Skills** tab at `#/clis/<cli>/skills` reports what that CLI loads, where
each skill comes from, what shadows what, the skill's provenance and its
estimated context cost.

**Installing** is PiCode's own write, in this order:
1. stage the content in the data dir;
2. validate it against the spec;
3. run the advisory scan;
4. show the preview and wait for explicit consent;
5. rename it into place atomically.

**Where installs go:**
- **Workspace:** the canonical `<ws>/.agents/skills/<name>`, a relative link (or
  a recorded copy) in `<ws>/.claude/skills/<name>`, and an entry in
  `skills-lock.json` v1.
- **Machine:** `~/.agents/skills/<name>`, links for the CLIs whose user folder
  differs, and an entry in `~/.agents/.skill-lock.json` v3.

These are the formats Vercel's CLI reads, so either tool can update what the
other installed.

**Toggles:** excluding a CLI uses that CLI's own toggle. Where there is none, the
row says so.

**The agent scope** is a PiCode list on the agent row (`agents.skills`). Each
launch passes it through the CLI's own flag where one exists. Each launch also
records its skill set in the launch snapshot, so Outcomes can compare runs with
and without a skill.

**The Marketplace** is one federated catalog for every CLI. It draws on seed git
repositories, sources the user adds and `.well-known` domains. skills.sh search
runs only when the user turns it on; it is off by default, and the switch says
that the query goes to Vercel. A card shows the source, the commit or digest, the
files, the license, the scan's findings and the other parties' audits. It never
shows a PiCode verdict.

**Consent:** PiCode never passes an auto-consent or auto-trust flag. Where a CLI
needs trust, the row shows the vendor's command for the user to copy.

**No PiCode package database:** the skill folders, the two locks and each CLI's
own files are the truth.

## Consequences

Easier:
- Every skill a machine or workspace carries is visible in each CLI that loads
  it, with its origin.
- One install serves every CLI in a project.
- An agent can try a skill without the repository changing.
- Outcomes can show whether a skill helped.
- `npx skills` users and PiCode users can share a repository's lock.

Harder, and accepted as cost:
- **Writes into the user's files.** PiCode writes into the user's repository and
  home, so these rules are load-bearing: re-read before write, 409 on a stale
  file, atomic renames, and a refusal to overwrite a folder that no lock names.
- **Vendor drift.** Nine vendors' folders and toggle keys move. The declaration
  is pinned by a live test per CLI, and a shape PiCode does not recognise fails
  loudly — it never reads as "no skills".
- **Lock format drift.** Vercel's lock formats move too: its CLI discards a
  global lock whose version it does not know. PiCode writes only the version it
  reads and never migrates one.
- **Digest interop.** The digest has to reproduce Vercel's `computedHash`, which
  orders paths with JavaScript's `localeCompare`. Vectors from the real CLI pin
  it; if they diverge, the install slice stops for an owner call.
- **Codex toggle.** It needs a TOML array-of-tables writer, which `clisettings`
  refuses today.
- **Network content.** Content fetched from the network is untrusted:
  - download and extract limits apply;
  - extraction refuses traversal, absolute paths, links and devices;
  - sources a user adds go through the SSRF guard PiCode already uses for web
    apps.

If we are wrong about a vendor's folder, that CLI's rows are wrong or missing,
its live test fails, and nothing else regresses. If we are wrong about consent,
PiCode has installed instructions the user did not read. That is why every
install shows its files first, a critical finding asks a second time, and nothing
installs in bulk.

## Alternatives considered

- **Keep skills inside Packages.** The format most skills ship in stays
  invisible, and one skill has to be wrapped differently for each CLI.
- **Run `npx skills` as the vendor binary**, as ADR-0167 does for plugins.
  - It needs Node, which PiCode does not require.
  - It covers seven of the nine CLIs (not Muse, not Omp).
  - Telemetry is on by default, and a known lock bug (#542) affects project
    installs.
  - It has no toggles and no per-agent sets, which PiCode would build anyway.
- **A PiCode-owned skills store injected into every launch.** It duplicates each
  CLI's own loader and breaks every launch made outside PiCode. ADR-0150 refused
  the same shape for connectors for the same reason.
- **A curated catalog with a PiCode mark.** This is the trust decision ADR-0167
  refused. Audits and scanners exist outside PiCode and are shown as theirs.
- **skills.sh on by default.** Every query would go to Vercel through an
  undocumented endpoint. The owner chose opt-in.
- **A CLI-neutral workspace view first.** Two studies propose one: the omp
  study's Project pane and the 2026-09-23 AGENTS.md study's workspace tab. Neither
  is decided. The per-CLI tab follows the house pattern (Packages, Connectors,
  Memory), and the same report can feed a workspace view if one is approved.
