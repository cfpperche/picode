# Automations

Run an agent on a schedule, or whenever another tool calls a URL. Every run
is an ordinary PiCode agent session: it shows in the sidebar, the dashboard
counts its cost, and its result lands in the Inbox.

Open it from the user menu → **Automations**, or `Ctrl+K` → Automations.

## Create one

1. **Create automation**.
2. **Name** it. The agent that runs it takes the same name.
3. **What it does** — *Start a new run* (a fresh session each time on the
   automation's own agent, in the workspace you pick) or *Message an agent*
   (the prompt is queued to an agent you already have).
4. **Prompt** — what the agent should do each time.
5. **Schedule** — Hourly, Daily, Weekdays or Weekly at a time, or *Custom*
   for a cron line (`minute hour day month weekday`, your local time zone).
6. **Webhook** — turn on to get a URL other tools can POST to.
7. **Limits** — max cost per run, max runs per hour/day/week.

**Run now** on the list or the detail page tries it immediately, even when
the automation is switched off.

## Describe it instead

With an agent open, type `/automate` followed by what you want, for
example `/automate every weekday at 9, summarize what changed since
yesterday`. The agent reads the repository and drafts the name, prompt,
schedule and limits; the editor opens pre-filled with a "Drafted by …"
line. Review, adjust, **Create**. If the agent answers without a config,
the editor still opens with your description as the prompt. Bare
`/automate` asks for the description first.

## Templates

**Suggested** on the Automations page lists ready-made jobs any agent can
run on a local repository: morning brief, nightly tests, docs drift,
outdated dependencies, stale branches, changelog draft, vulnerability
audit. Filter by category, click one, and the editor opens filled in. In
the editor, **Start from template…** does the same for an automation you
are already writing. Templates never push, commit or delete anything;
they report.

## Webhook

After saving, the detail page shows the URL and the secret **once**. Send
the secret as a header:

```bash
curl -X POST https://localhost:8445/api/automations/<id>/fire \
  -H "Authorization: Bearer <secret>" \
  -d '{"anything": "the agent should know"}'
```

The body (up to 64 KB, any format) is appended to the prompt as text. A
wrong secret answers `401`; **Regenerate secret** replaces it. The URL is
your PiCode server's, so a caller outside your machine or tailnet needs to
reach it the same way your browser does.

## What a run can end as

| Result | Meaning |
|---|---|
| Done | The agent finished. Its final message is the Inbox item. |
| Skipped · busy | The previous run was still going. Runs never overlap. |
| Skipped · rate cap | The runs-per-window limit was reached. |
| Skipped · agent in terminal | The agent is open in a terminal, where messages are not delivered automatically. |
| Failed · cost cap | Spending passed the limit; the run was stopped. |
| Failed · pi missing | `pi` is not installed or not on PATH. |
| Failed · daemon restarted | PiCode restarted mid-run. The next scheduled run starts fresh. |

## Timing

Schedules run only while PiCode is running (the machine is on). Each
automation fires a few minutes after its slot — a fixed offset per
automation, never more than half the interval — so many "daily at 09:00"
automations do not hit your provider at the same second. If PiCode was
down when a slot passed, the missed work runs **once** when it comes back.
