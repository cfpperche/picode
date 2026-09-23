---
description: How each removed agent ended — the answer you give when you remove it, beside what PiCode saw.
---

# Outcomes

When you remove an agent, PiCode keeps a short record of how it went: the
answer you give, how the agent was set up, and what PiCode saw while it ran.
**Outcomes** lists those records, and the dashboard counts them.

- **Where:** the user menu ▸ **Outcomes**, or `#/outcomes`. The home
  dashboard has an **Agent outcomes** section for its date range.
- **Not this:** not telemetry. Records stay in PiCode's data folder on this
  machine and are never sent anywhere.

## The question when you remove an agent

The remove dialog asks **How did it go?** It is optional: **Remove** works
whether you answer or not.

| Answer | Then |
|---|---|
| Resolved | Nothing else |
| Partly, Didn't resolve | **What got in the way?** — pick any that apply |
| Just trying | Nothing else |

The reasons are: wrong setup (model, CLI, permissions, connectors or
folder), misunderstood the task, got stuck or looped, said done when it
wasn't, too slow or too costly, and switched to another agent. A short note
is optional.

PiCode does not ask when there is nothing to answer about:

- the agent never worked (it never started a turn), or
- it lived less than a minute, or
- you turned the question off (**Stop asking** in the dialog, or the
  **Ask when removing agents** switch on the Outcomes page).

Removing a workspace ends its agents too. Their records say "With its
workspace" and carry no answer; you can answer later.

If you also choose to delete the agent's sessions, the record keeps the
setup and the numbers, but nothing can read the conversation afterwards.

## What a record holds

| Part | What |
|---|---|
| Answer | The outcome, the reasons and your note |
| Setup | CLI, provider and model, thinking, tools, checklist, packages, extra prompt, launch arguments. Environment variables are kept by **name only**, never their values |
| What PiCode saw | How long the agent lived, how many turns it took, how often it asked for you, checklist progress |
| Sessions | Where the sessions were, and whether the removal deleted them |
| Cost | What those sessions cost, read before any deletion. A `~` marks a list-price estimate for turns the CLI left unpriced. For Pi it covers every session of the agent; for other agents, their last session. Grok, Hermes Agent, OpenCode and Antigravity show **—**: PiCode cannot read their sessions as files |

Turns count each time the agent went from waiting to working. Agents created
before PiCode counted turns show **—** instead of a number.

## Using the page

- The numbers at the top follow the date range, workspace and CLI filters.
  **Resolved** is the share answered *Resolved* among agents that were set a
  task: *Just trying* answers are left out. **Answered when asked** counts
  answers given in the remove dialog.
- Filter by date range, workspace, CLI and outcome.
- Open a row to see the whole record. **Answer** (or **Change answer**) sets
  the outcome later; **Delete record** removes it from the page and from
  every count.
- **Export** downloads every record as JSON lines, including removals you
  undid.

Undoing a removal keeps its record out of the counts.
