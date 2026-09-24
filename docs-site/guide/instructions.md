---
description: See which instruction files (AGENTS.md, CLAUDE.md and their kin) each agent CLI reads in a workspace, and why it skips the others.
---

# Agent instructions

Every agent CLI PiCode runs reads `AGENTS.md`, but each one decides differently
which file wins. Claude Code reads `CLAUDE.md` instead of `AGENTS.md` when both
exist; Codex, Pi, OpenCode, Hermes and Muse Code read `AGENTS.md`; Grok reads
both. The **Instructions** page shows, for one workspace, what each installed
CLI will read and what it leaves out.

## Open it

Open the **…** menu of a workspace in the sidebar and choose **Instructions**.
The page opens over your tabs, like Agent CLIs, and **Back** returns to them. It lists the instruction files it finds: in the workspace and the folders
inside it, in the folders above it, and your personal files (such as
`~/.claude/CLAUDE.md`). Each column is one CLI; each cell says what that CLI
does with the file:

| Cell | Meaning |
|---|---|
| **Reads** | The file is in the agent's instructions from the start of a session |
| **On demand** | Read once the agent works in that folder |
| **Skipped** | Another file wins; pick the cell to see which one and why |
| **Needs trust** | The CLI ignores the folder's instructions until you trust the folder in that CLI |
| **—** | The CLI never reads this file here |
| **Unknown** | PiCode cannot tell from here |

Pick a cell to read the reason below the table. **Start in** answers for a
session that starts in a folder inside the workspace, which matters in a repository with
nested `AGENTS.md` files. A file name opens it in the editor.

## Findings

Above the table, one line per thing you may want to change, each with the
file to open:

- a `CLAUDE.md` that sends Claude Code to `AGENTS.md` in words, which keeps
  Claude Code from loading `AGENTS.md` (a line `@AGENTS.md` makes it load);
- a `CLAUDE.local.md` that turns `AGENTS.md` off for Claude Code;
- folders where Claude Code and the other CLIs read different files;
- a file over a CLI's size limit (Hermes cuts long files on models with a
  small window; Codex and Antigravity keep a fixed number of bytes);
- import lines (`@path`) that most CLIs read as plain text;
- a folder Grok does not trust yet.

Some fixes are a setting rather than a file. A CLI's **Settings** page has an
**Instructions** group: Claude Code's **Project instructions** (read CLAUDE.md,
AGENTS.md or both), the extra file names and the size limit Codex uses,
Hermes's size limit, and extra instruction files for OpenCode. A finding
whose fix is one of them opens that page.

Under the table, **What agents here read** lists what the latest session of
each agent in this workspace actually loaded, for Claude Code, Codex and Grok,
which record it. The other CLIs do not record it, so they are not listed.

Two findings offer **Review change**: a `CLAUDE.md` that only points to
`AGENTS.md` in words (the change adds an `@AGENTS.md` line), and a personal
file that is not in `.gitignore`. **Add personal file** creates
`CLAUDE.local.md` or `AGENTS.override.md` and adds it to `.gitignore`. Each
shows the exact change first and writes only when you choose **Write
change**. Nothing is committed: the change waits in the Git tab for you to
review. If a file changed after the change was shown, PiCode shows the new
change instead of writing.

When you create an agent from the sidebar, the **New agent** dialog shows the
same answer in one line for the CLI you pick.

## Draft a first AGENTS.md

When a workspace has no instruction file, the page offers **Draft with an
agent**. Pick the agent that should write it: the dialog shows the exact
request it starts with. The agent reads the repository and writes
`AGENTS.md` at its root like any other change it makes, so you review it in
the Git tab before you commit. Hermes Agent can't start with a request yet,
so the dialog leaves it out.

## What it is not

The page writes only the changes listed above, and only after you confirm
them; anything else is fixed in the editor. It shows what each CLI's rules say for the version PiCode measured;
a CLI that changes its rules later can differ until PiCode catches up.
Hermes also refuses a file its prompt-injection scan flags, which PiCode
cannot predict.
