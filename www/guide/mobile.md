# PiCode on your phone

Open the same address you use on the desktop (`https://<host>:8445`) on your
phone. Below 768px PiCode switches to the **mobile shell** — a console for
watching your agents and answering them when you are away from a desk, not
the workstation squeezed onto a small screen.

## Install it

- **Android / Chrome**: More → **Install app**.
- **iPhone / Safari**: Share → **Add to Home Screen**. Installed, it opens
  full screen and remembers you.

The desktop's **QR** button (sidebar header, or More → *Open on another
phone*) shows the address to scan. Your phone must trust the same mkcert
certificate the desktop uses — see [Getting started](/guide/getting-started).

## What is where

- **Now** — what needs you first: an agent waiting on a permission or a
  question (answer it right there), then blocking Inbox items, then who is
  running, today's spend, and the last finished runs.
- **Inbox** — the same Inbox as the desktop: approvals, questions, results.
- **Work** — the same three views as the desktop sidebar: **Workspaces**
  (each folder with its agents and terminals; **+ Agent** / **+ Terminal**
  on the card), **Agents** (free agents, outside any workspace) and
  **Terminals** (free terminals, outside any workspace). **Start** / **Stop** on an agent row,
  **Remove** on a terminal row; **New** creates whatever the view shows.
- **More** — Providers, Settings, Preferences, MCP, Packages, Devices, System,
  and **Desktop layout** if you want the full shell on this screen.

Tap an agent to open it: the conversation (with any question the agent is
asking), the composer with **prompt / steer / follow-up** and dictation, and
**Stop** to abort the current turn. An agent living in a terminal shows a
**Chat | Terminal** switch.

Tap a terminal to attach to it. Under the terminal a **key bar** gives you
what the phone keyboard lacks: Esc, Tab, Ctrl+C, Ctrl+D, Ctrl+Z, Ctrl+L,
arrows, and `/ | - ~`.

## Push notifications

More → **Notifications** → **Enable push on this device**. PiCode then
wakes the phone when an agent is blocked on a question or a permission
that nobody is watching, when a blocking Inbox item arrives, or when a run
finishes while nobody was looking — each has its own switch. Tapping the
notification opens that agent or item.

PiCode stays quiet while a browser on the host machine is open: the phone
is for when you are away.

- Needs the `https://` address (the same mkcert certificate as the desktop).
- **iPhone**: add PiCode to the Home Screen first, open it from there, then
  enable. Safari will not show the permission prompt otherwise.
- **Send test** proves the path end to end.
