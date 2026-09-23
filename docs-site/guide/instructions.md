---
description: See which instruction files (AGENTS.md, CLAUDE.md and their kin) each agent CLI reads in a workspace, and why it skips the others.
---

# Agent instructions

Every agent CLI PiCode runs reads `AGENTS.md`, but each one decides differently
which file wins. Claude Code reads `CLAUDE.md` instead of `AGENTS.md` when both
exist; Codex, Pi, OpenCode, Hermes and Muse Code read `AGENTS.md`; Grok reads
both. The **Instructions** tab shows, for one workspace, what each installed
CLI will read and what it leaves out.

## Open it

Open a workspace's **…** menu in the sidebar and choose **Instructions**. The
tab lists the instruction files it finds: in the workspace and its
subfolders, in the folders above it, and your personal files (such as
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
session that starts in a subfolder, which matters in a repository with
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

When you create an agent from the sidebar, the **New agent** dialog shows the
same answer in one line for the CLI you pick.

## What it is not

The tab reads files and never writes them: fixing a finding is done in the
editor. It shows what each CLI's rules say for the version PiCode measured;
a CLI that changes its rules later can differ until PiCode catches up.
Hermes also refuses a file its prompt-injection scan flags, which PiCode
cannot predict.
