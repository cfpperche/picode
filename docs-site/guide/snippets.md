# Snippets

A snippet is a prompt or a shell command you save once and reuse. It can
carry **placeholders** (`&#123;&#123;name&#125;&#125;`) that you fill at send time, and it
works everywhere you work: the managed agent's composer, the command
palette, a running Agent CLI terminal (Claude Code, Codex, Grok, Hermes,
OpenCode, Pi), or a plain shell pane. On the phone it lives under
**More → Snippets**.

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

## Placeholders

| You write | What happens |
|---|---|
| `&#123;&#123;name&#125;&#125;` | The sheet asks for a value every time. |
| `&#123;&#123;name=default&#125;&#125;` | Used as-is unless you type a value. |
| `&#123;&#123;cwd&#125;&#125;` `&#123;&#123;workspace&#125;&#125;` `&#123;&#123;branch&#125;&#125;` `&#123;&#123;date&#125;&#125;` `&#123;&#123;agent&#125;&#125;` `&#123;&#123;cli&#125;&#125;` | Filled from the target — folder, repository, date, agent or CLI name. Not asked. |
| `&#123;&#123;&#123;&#123;` | A literal `&#123;&#123;` (the editor's **Insert** button writes it). |

Inserted values are never re-scanned, and nothing on the body is ever
executed by PiCode — expansion is text substitution only.

## Use one

| Where | How |
|---|---|
| Agent composer | Type `/snip:` and pick. **Insert snippet** splices the text into your draft; **Send snippet** fills, sends, and keeps any pictures you attached. |
| Command palette | `Ctrl+K` → **Send snippet: …** sends to the focused agent. |
| CLI terminal (desktop) | Right-click the pane → **Send to terminal…** |
| Shell pane (desktop) | Right-click the pane → **Run command…** |
| Phone | More → Snippets to manage; terminal **actions → Send to terminal… / Run command…** |

A **Command** always stops at one confirm that names the terminal and
shows the exact command after expansion. PiCode does not quote values:
write `echo "&#123;&#123;msg&#125;&#125;"` in the snippet if the shell should see one word.

## What it is not

- **Not a Pi template.** Pi's own `~/.pi/agent/prompts/*.md` keep working
  as before; snippets are PiCode's library and expand inside PiCode, so
  they also reach CLIs that have never seen your files.
- **Not a script runner.** No loops, no conditionals, no `!` shell
  interpolation, and nothing runs at expand time.

## Reference

- `GET/POST /api/snips`, `POST /api/snips/&#123;id&#125;/expand`, `POST /api/snips/&#123;id&#125;/run` — see the [API reference](/api/).
- Decision: [ADR-0130](https://github.com/cfpperche/picode/blob/main/docs/decisions/0130-picode-snippets.md).
