---
description: A saved prompt or shell command, with placeholders filled at send time.
---

# Snippets

A snippet is a prompt or a shell command you save once and reuse. It can
carry **placeholders** (<code v-pre>{{name}}</code>) that you fill at send time, and it
works everywhere you work: the managed agent's composer, the command
palette, a running Agent CLI terminal (Claude Code, Codex, Grok, Hermes,
OpenCode, Pi), or a plain shell pane.

- **Where:** **Tools → Snippets**, `Ctrl+K` → Snippets, or **More → Snippets** on a phone.
- **Not this:** not an [automation](/guide/automations) that runs on its own.

## Create one

1. Open **Tools → Snippets** (user menu) or the command palette (`Ctrl+K`
   → "Snippets").
2. **New snippet**, then fill:
   - **Kind** — *Prompt* is sent to an agent or a CLI terminal; *Command*
     is pasted into a shell and run.
   - **Title** and an optional **description** and **tags**.
   - **Body** — the text with placeholders.
3. **Save**. The slug under the title is the snippet's handle
   (`/snip:review-pr`); it fills from the title and stays editable.

The editor answers while you type instead of after the click:

| You see | What it means |
|---|---|
| A green line under **Slug** (`/snip:review-pr · sent to an agent`) | The address is free and this is where the snippet goes when you send it. |
| A red line under **Slug** (`Another snippet already uses /snip:…`) | Pick another address; **Save** and **Create snippet** stay off until you do. Editing a snippet never reports its own address as a clash. |
| A red line under the **body** (broken or unclosed placeholder) | The table stays on your last good version, dimmed, so you keep your placeholders in sight. |

Your work is kept as a draft while you type — **closing the tab, reloading or
coming back later does not lose the text**. Saving (or discarding) clears it.

## Placeholders

| You write | What happens |
|---|---|
| <code v-pre>{{name}}</code> | The sheet asks for a value every time. |
| <code v-pre>{{name=default}}</code> | Used as-is unless you type a value. |
| <code v-pre>{{cwd}}</code> <code v-pre>{{workspace}}</code> <code v-pre>{{branch}}</code> <code v-pre>{{date}}</code> <code v-pre>{{agent}}</code> <code v-pre>{{cli}}</code> | Filled from the target — folder, repository, date, agent or CLI name. Not asked. |
| <code v-pre>{{{{</code> | A literal <code v-pre>{{</code> (the editor's **Insert** button writes it). |

Inserted values are never re-scanned, and nothing on the body is ever
executed by PiCode — expansion is text substitution only.

### The placeholder table

Below the body, each placeholder gets a row you can fill in without editing
the body by hand. The body stays the source of truth: what you set here is
written back into it.

| Column | What it does |
|---|---|
| **Default** | The value used when the field is left empty. Typing one turns the placeholder into <code v-pre>{{name=value}}</code>. |
| **Optional** | The field can be skipped — written as <code v-pre>{{name=}}</code> when it has no default yet. |
| **Enum** | A comma-separated list of allowed values (`dev, prod`). At send time the field becomes a dropdown, and a value outside the list is refused. |

Reserved names (<code v-pre>{{cwd}}</code>, <code v-pre>{{workspace}}</code>, <code v-pre>{{branch}}</code>, <code v-pre>{{date}}</code>, <code v-pre>{{agent}}</code>, <code v-pre>{{cli}}</code>) show **from the target** and no controls — PiCode
fills them.

On the phone the same table is a card per placeholder, with the same three
settings.

## Start from something

| Door | What it saves you |
|---|---|
| **Starters** | Six ready snippets (review a pull request, explain an error, write a commit message, summarize changes, run the tests and fix failures, standup update). Open one, adjust the text, save. The grid starts open on an empty page and folds away once you have snippets. |
| **Save selection as snippet** (right-click a selection) | Turns text you already wrote into a snippet, with the first words as its title. The composer's toolbar has the same button when a selection exists. |
| **Import** (in the editor) | Paste a raw prompt; PiCode suggests turning `[BRACKETS]` or `UPPER_CASE` into <code v-pre>{{lower_snake}}</code> placeholders — each suggestion can be switched off. |
| **Duplicate** (on a snippet) | Copies the body, kind and tags into a new snippet: name it, save it. |
| **Save as snippet** (phone, Message options) | Saves the draft you are writing in the composer. |

## Use one

| Where | How |
|---|---|
| Agent composer | Type `/snip:` and pick. **Insert snippet** splices the text into your draft; **Send snippet** fills, sends, and keeps any pictures you attached. |
| Command palette | `Ctrl+K` → **Send snippet: …** sends to the focused agent. |
| CLI terminal (desktop) | Right-click the pane → **Send to terminal…** |
| Shell pane (desktop) | Right-click the pane → **Run command…** |
| Phone | More → Snippets to manage; terminal **actions → Send to terminal… / Run command…** |

Filling a snippet shows one field per placeholder. A placeholder with a
**choices** list (the table's Enum column) becomes a dropdown of those values;
<code v-pre>{{cwd}}</code>-style names are filled from the target and not asked at all.

A **Command** always stops at one confirm that names the terminal and
shows the exact command after expansion. PiCode does not quote values:
write <code v-pre>echo "{{msg}}"</code> in the snippet if the shell should see one word.

## What it is not

- **Not a Pi template.** Pi's own `~/.pi/agent/prompts/*.md` keep working
  as before; snippets are PiCode's library and expand inside PiCode, so
  they also reach CLIs that have never seen your files.
- **Not a script runner.** No loops, no conditionals, no `!` shell
  interpolation, and nothing runs at expand time.

## Reference

- `GET/POST /api/snips`, `GET /api/snips/slug/{slug}` (the editor's address check),
  `POST /api/snips/{id}/expand`, `POST /api/snips/{id}/run` — see the [API reference](/api/).
- Decision: [ADR-0130](https://github.com/cfpperche/picode/blob/main/docs/decisions/0130-picode-snippets.md).
