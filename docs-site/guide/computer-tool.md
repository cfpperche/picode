---
description: Let an agent use this computer through PiCode's desktop app.
---

# Computer use for pi

Let an agent use the Windows desktop through PiCode's own desktop app: see a
monitor or a window, read a window's controls, click, type, scroll, use the
clipboard, open a program. The agent asks for one of twenty-three actions;
PiCode checks that you switched that agent on, and the desktop app does the
work on your desktop.

- **Where:** install `pi-computer` ([Packages](/guide/packages)), then switch the agent on in **Settings ▸ Computer** (user menu, desktop app only).
- **Not this:** not the [work browser](/guide/browser-tool). That reads and drives web pages inside PiCode's own browser. This page is the whole desktop.

Claude Code, Codex and the other agent CLIs get the same tool over MCP:
[PiCode tools for other agent CLIs](/guide/picode-mcp).

`pi-computer` is an **optional pi package — an extension, not part of PiCode
core**. It reaches the daemon over the same authenticated API every other
call uses (the install token); it opens no port and adds no credential.

## What it does, plainly

With the switch on, the agent acts **with your own permissions, on your
desktop, without a sandbox** — the same way you would at the keyboard. Any
window it can see, it can click; any program you can open, it can open. Off
by default; off is one click away, and it holds from the next call. Text on
the screen can try to instruct an agent; this version does not filter it.
Keep the switch off for agents you are not watching.

## What it can do

| Action | Answer |
|---|---|
| `screenshot`, `zoom`, `wait` | an image the agent sees — one monitor, or one window when it names one |
| `snapshot` | a window's controls as lines: role, name, position in the last image, the text of a field |
| `windows`, `focus` | what is open, and bringing one window to the front |
| clicks, `left_click_drag`, `mouse_move`, `scroll`, `type`, `key`, `hold_key` | the action, then a fresh image |
| `type` with `mode: "paste"` | the text through the clipboard and one Ctrl+V — whole and fast, for long text or an app that garbles keystrokes; your clipboard text is put back afterwards (an image or files on it are not) |
| `clipboard_read`, `clipboard_write` | the clipboard's text |
| `open` | a program, a file or a URL, opened as Windows would |

Every coordinate the agent sends is a pixel of the last image it received.
The guidelines it carries say so, and say when to stop and ask you: before
paying, sending a message, deleting files or typing a password.

## Two things it needs

1. **The desktop app, running and connected.** The actions run inside the
   PiCode desktop shell on Windows. Without it the agent is told plainly:
   *the desktop app is not connected*.
2. **A visible package and a switched-on principal.** Install scope decides
   who sees the tool; the switch decides who may use it. A `pi` you started
   outside PiCode has no identity here and is always refused.

## Managed agent, or a CLI in a terminal?

| Where it runs | Identity | What it gets |
|---|---|---|
| An agent PiCode manages | its agent id | its own switch in Settings ▸ Computer |
| A CLI in a PiCode terminal | the terminal's id | its own switch, listed by the terminal's name |
| A `pi` you started yourself | none | refused, always |

## Where a click or a keystroke lands

Input goes to the window in front, and you move the front with every click
of your own. So the agent may click, type or press keys only while the
window in front is still the one it last looked at (its last screenshot)
or chose (`focus`). If you clicked elsewhere in between, the action is
refused and the agent is told to look again — nothing is typed into what
you are using. The refusal is one of the rows on Settings ▸ Computer.

## What you see

Every call, allowed or refused, is listed on **Settings ▸ Computer** and
counted on the dashboard ("Desktop"). In the chat, each step shows the
capture the agent got back as its last capture.
