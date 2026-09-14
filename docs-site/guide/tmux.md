---
description: See every tmux session on the machine, including ones PiCode no longer knows.
---

# tmux sessions

PiCode runs every terminal and every interactive agent inside a tmux session,
so closing the browser or restarting the daemon does not kill your work. The
**tmux** app shows the whole server those sessions live on — including the ones
PiCode no longer knows anything about.

- **Where:** **Apps → tmux** on the desktop, or **More → Apps → tmux** on a phone.
- **Not this:** not the terminal tab itself. This is the list of sessions on the machine, including orphans.

> **What is tmux?** A program that keeps terminals running after you close the
> window you were watching them in. PiCode uses it so an agent survives a
> browser close, a phone lock or a daemon restart.

## Sessions

Every session on the machine's tmux server, in four kinds:

| Kind | What it is |
|---|---|
| **PiCode** | A terminal or an agent PiCode knows about. **Open** takes you to it. |
| **Not in PiCode's records** | A PiCode-shaped session with no terminal or agent behind it: the leftover of a run whose records are gone, or a session belonging to another PiCode on this machine. PiCode cannot tell those apart, and says so. |
| **Not PiCode's** | Your own tmux session, from your own shell. Read-only here: PiCode never touches it. |
| **exited** | The session's process ended. With PiCode's settings a dead process usually takes its session with it, so this is rare and worth looking at. |

Click a row to open its own page: the full session name, the folder it runs
in, the command, how long it has been up, and how many browsers or clients are
attached right now. A session PiCode knows opens its terminal or agent from
there.

## Removing a leftover

A session under **Not in PiCode's records** offers **Remove this session** on
its own page. It stops
whatever is still running inside that session — the work in it is lost — so it
asks first, and it re-reads the session before acting: if the session changed
between the page being drawn and your click, nothing happens and the page
refreshes.

Removal is never automatic and never a bulk action. There is no "clean up
everything" button on purpose: a session PiCode cannot attribute may be another
PiCode's live work, and the only thing that can settle that is you looking at
it. Before acting, PiCode re-reads the session — if it changed between the
page being drawn and your click, nothing happens and the page refreshes.

## Server

The **Server** tab shows the facts about the tmux server itself: its version
whether it is running, the socket path, how many sessions and clients are
attached, and the keyboard mode tmux reports (PiCode sets `xterm` on attach so
Shift+Enter reaches your agents; if the server value differs, the terminal
settings are where to change it — Preferences → Terminal).

Below that, **In PiCode's records, not on this server** lists terminals whose
session is gone — including, marked **lost at restart**, ones that were running
when the daemon last shut down and did not survive it.

## Related

- [Terminals](./terminal-status) — what a terminal is, and the CLI activity it
  reports.
- [Settings](./settings) — tmux options live in Preferences → Terminal, which
  reads the live catalog from your own server.