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
| Status | **Loaded**; **Shadowed** (another copy with the same name wins); **If trusted** or **Needs trust** (the CLI loads a project's skills only in a folder you trusted in that CLI); **Not loaded** (the `SKILL.md` has no header or no description) |
| Source | who installed it — the repository named in the `skills` tool's lock or the Hermes hub's — or **Added by hand**; **edited** means the folder changed since it was installed |
| Folder | where the CLI found it; **link** means the folder points somewhere else |
| At start | an estimate of what the skill's name and description cost in every session |

Click a row to see its full folder, the reason for its status and the other
CLIs that read the same folder. **All**, the workspace name and **This
machine** narrow the list; the filter matches names, descriptions and sources.

## Trust

Pi, Grok, Hermes and Muse read the skills in a workspace only after you trust the
folder in that CLI. When skills in a workspace wait on trust, the tab says
so and offers the CLI's own command to copy (for Hermes, `hermes skills
trust`). PiCode never trusts a folder for you.

## Where each CLI looks

Most CLIs read `.agents/skills` in the project. Claude Code reads only
`.claude/skills`. The first folder that has a name wins, so the same skill in
two places shows one **Loaded** row and one **Shadowed** row.

Installing, switching a skill off and choosing skills per agent come next; see
the [plan](https://github.com/cfpperche/picode/blob/main/docs/plans/skills.md).
