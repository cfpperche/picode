---
description: What the dashboard's spend, token and limit numbers mean, and where they come from.
---

# Dashboard

The dashboard adds up what every agent CLI on this machine did: Pi, Claude
Code, Codex, Grok and the rest. PiCode reads each CLI's own session files and
changes nothing in them.

- **Where:** the home screen.
- **Not this:** not a bill. A plan subscription is not charged per token;
  PiCode shows what the tokens would cost on the API.

## One workspace

Open **Workspaces → workspace menu (⋯) → Overview** for one project's
questions, agents, Git state, missions and recorded activity. This opens a
page; your editor tabs stay as they were. Pick Today, 7 days, 30 days or All
for the activity chart. Each row opens the existing Inbox, agent, Git or
Missions view where you can act.

Activity is assigned from the folder each CLI recorded for a session. A
session under a nested workspace belongs to that more specific workspace.
Sessions without a recorded folder are not included; changing the registered
workspaces can change which older sessions appear. The home dashboard above
still counts the whole machine.

## Spend: reported and estimated

Some CLIs write down what each turn cost, and PiCode uses that figure. Codex
never does, and Claude Code does not for a session still running. PiCode
**estimates** the turns without a price at list price from
[LiteLLM's public price table](https://github.com/BerriAI/litellm), the same
table the `ccusage` tool uses. Estimates are always marked:

| Where | Mark |
|---|---|
| Spend card | "$X estimated at list price" under the total |
| By CLI | "estimated", or "$X est." when only part of it is |
| Spend by model | a `~` before the amount |

A model the table does not list stays **not priced**. PiCode shows "not
priced" there, not $0.00.

### Turning the estimate off

PiCode downloads the price table once a day and keeps a copy in its data
folder (`var/litellm-prices.json`). To stop the download, start PiCode with:

```bash
PICODE_PRICE_TABLE_URL=off
```

With the download off, estimates come from the copy PiCode already has.
Without a copy, nothing is estimated. To use another copy of the table, set
the variable to its URL.

## Limits

The **Limits** card shows how much of each plan is used and when it resets.
It combines two sources: what a CLI records itself (Codex), and the plans
PiCode reads for your signed-in accounts on the Providers screen. If a plan
needs a new sign-in, the card says so and links to the right screen.

## What each CLI reports

Not every CLI records everything. The last card lists, for each CLI, which
numbers it writes. When a number covers only some CLIs, the card showing it
says which.
