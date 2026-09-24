---
description: See which Agent Skills each agent CLI loads, from which folder, who installed them and what they cost at start.
---

# Skills

A skill is a folder with a `SKILL.md` file: a name, a description, and
instructions an agent reads when a task calls for them. Every agent CLI in
PiCode reads skills, but each one looks in its own folders, in its own order.
The **Skills** tab shows what one CLI will actually load.

- **Where:** **Agent CLIs →** a CLI **→ Skills**. Select an agent first to
  include the skills in its workspace.
- **Not this:** plugins and packages a CLI installs with its own commands are
  in [Packages](/guide/packages); MCP servers are in [Connectors](/guide/mcp).

## Reading the list

| Column | Meaning |
|---|---|
| Skill | the skill's name and description |
| Status | **Loaded**; **Off** (switched off in the CLI's own settings); **Shadowed** (another copy with the same name wins); **If trusted** or **Needs trust** (the CLI loads a project's skills only in a folder you trusted in that CLI); **Not loaded** (the `SKILL.md` has no header or no description) |
| Source | who installed it — the repository named in the `skills` tool's lock or the Hermes hub's — or **Added by hand**; **edited** means the folder changed since it was installed |
| Folder | where the CLI found it; **link** means the folder points somewhere else |
| At start | an estimate of what the skill's name and description cost in every session |

Click a row to see its full folder, the reason for its status and the other
CLIs that read the same folder. The chips run **Global** (this computer), then
the workspace by its name, then — for Pi, Omp and Claude Code — the agent by
its name, showing everything that agent loads (or, for an agent that runs
isolated, only its own skills). The filter matches names, descriptions and
sources.

## Turning a skill off

The switch at the start of a row turns the skill off without removing it.
PiCode writes the CLI's own setting, so the CLI and its own menus agree:

| CLI | Where the switch is saved |
|---|---|
| Claude Code | `skillOverrides` in its settings: Global writes `~/.claude/settings.json`, a workspace writes `.claude/settings.local.json` |
| Codex | `[[skills.config]]` in `~/.codex/config.toml`, for Global and workspace skills alike |
| OpenCode | `permission.skill` in `opencode.json`, Global or in the workspace |
| Omp | `skills.ignoredSkills`, Global or in `.omp/config.yml` inside the workspace |
| Grok, Hermes | their own `disabled` list, for the whole computer |
| Muse | Muse's own `muse skills disable` |

Pi and Antigravity have no switch for one skill; remove the skill to stop it
loading. If another settings file still decides, for example a project
setting in Claude Code, the message after the switch names that file.

## Trust

Pi, Grok, Hermes and Muse read the skills in a workspace only after you trust the
folder in that CLI. When skills in a workspace wait on trust, the tab says
so and offers the CLI's own command to copy (for Hermes, `hermes skills
trust`). PiCode never trusts a folder for you.

## Where each CLI looks

Most CLIs read `.agents/skills` in the project. Claude Code reads only
`.claude/skills`. The first folder that has a name wins, so the same skill in
two places shows one **Loaded** row and one **Shadowed** row.

## Adding a skill

**Add skill** asks where the skill comes from:

| Source | Example |
|---|---|
| A GitHub repository, optionally a folder and a branch or tag | `anthropics/skills`, `anthropics/skills/skills/pdf#main` |
| A GitHub address | `https://github.com/owner/repo/tree/main/skills/pdf` |
| A site that publishes a skills index | `https://example.com` (read from `/.well-known/agent-skills/index.json`) |
| A folder on this computer | `~/my-skills/release-notes` |

**Look inside** downloads it without installing anything and shows every skill
it carries: its files, format problems and what the safety scan noticed. The
scan looks for known risky patterns, such as a script piped into a shell or a
read of your SSH keys; it is not a review, and PiCode never marks a skill as
safe. A critical finding asks you to confirm that you read the files.

Pick **Global** or the workspace. A workspace install goes into
`.agents/skills`, which seven of the nine CLIs read, with a link in
`.claude/skills` for Claude Code; a machine install goes into `~/.agents/skills`
with links for Claude Code, Hermes and Antigravity where they are installed.
Both are recorded in the same lock file the `skills` command-line tool uses
(`skills-lock.json` in the project, `~/.agents/.skill-lock.json` on the
computer), so either tool can update what the other installed.

If a folder with that name is already there, PiCode asks: keep it and only
record where it came from, or replace it.

## Updating and removing

**Check for updates** compares every recorded skill with its source. A skill
with a newer version shows **update**; open the row and choose **Update**. If
you edited the skill since it was installed, PiCode asks before overwriting
your changes.

**Remove** is offered for skills in `.agents/skills`, and asks first: every CLI
that reads that folder loses the skill. Skills in a CLI's own folder are that
CLI's to manage.

## Trying a skill in one agent

Open the Skills tab with an agent selected, choose its chip, then **Add skill**:
the dialog adds the skill to that agent alone. PiCode keeps its own copy and
hands it to the agent the next time it starts, so the workspace and your home
folder stay as they were. The agent's terminal shows **Launch changes pending**
until you restart it.

| CLI | How the agent gets it |
|---|---|
| Pi | loaded like any other skill; if a folder has a skill with the same name, the folder's wins |
| Omp | loaded ahead of the folders' copies |
| Claude Code | as `/picode-agent:<name>` |

The other CLIs cannot take a skill for one agent when they start, and their tab
says so. **Remove** on an agent's row takes it off that agent at its next start.
A row marked **Missing** lost PiCode's copy; add it again.

Switching a skill off for one CLI comes next; see the
[plan](https://github.com/cfpperche/picode/blob/main/docs/plans/skills.md).
