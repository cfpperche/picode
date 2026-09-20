# PiCode on your phone

Open `https://<host>:8445/mobile/` for the phone app. Opening the server's
root address chooses the phone app below 768px unless you saved a layout
preference. `/desktop/` always opens the responsive desktop app. Resizing
or rotating keeps your current app and connection.

The phone app has its own interface and download, with Now, Inbox, Work
and More for managing work while away from your desk.
Existing links and home-screen installations continue to work.

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
  beside each workspace), **Agents** (free agents, outside any workspace) and
  **Terminals** (free terminals, outside any workspace). **Start** / **Stop** on an agent row.
  A terminal's **⋯** is Rename, Continue in… when a conversation is pinned,
  Start or Restart/Stop, and Remove. **New** creates whatever the view shows.
- **More** — search grouped tools: **Tools** (Pins, Agent CLIs including
  Providers, Settings and Packages, Automations, Apps, llama.cpp,
  Integrations) and **PiCode** (Preferences, Notifications, Devices,
  System). **Desktop layout** opens the desktop shell on this screen.

Work keeps its view selector and **New** action at the top. Tap **Search**
to filter by name or folder. Failed refreshes keep the last loaded work
visible and offer a retry.

Tap an agent to open its conversation, pending question and compact message
box. **Enter** adds a new line; tap **Send** (or Ctrl/Cmd+Enter on a hardware
keyboard) to send. **Message options** contains prompt / steer / follow-up,
Photos, workspace files, sketch, dictation, voice and an expanded editor.
**Settings** opens without leaving the conversation. Unsent text and images
stay with each agent while you move between screens in the open app; a full
reload clears these temporary drafts. A failed send keeps the draft for
**Retry**. **Stop** remains available while the agent is working or waiting.

An Agent CLI terminal
adds **Attach** in the header — Photos, a file, or a file from the
folder — and Send types it into the TUI. An agent living in a terminal
shows a **Chat | Terminal** switch.

Replying to that terminal agent from the Inbox keeps you on its Terminal
screen. A small card moves through **Receiving → Processing → Returning** and
shows the answer as it arrives; **Cancel and return** stops the temporary turn
and restores the TUI. If automatic return fails, **Return to terminal** restarts
that TUI and remains available if you need to retry. PiCode does not open a
chat or change the agent's saved run mode.

![The Now tab: what needs you first — here a seeded demo question with Accept / Ignore](../img/app-mobile.png)

![The Inbox tab: the same approvals, questions and results as the desktop](../img/app-mobile-inbox.png)

<video controls muted preload="metadata" poster="/picode/video/take-it-anywhere-poster.jpg" src="/picode/video/take-it-anywhere.mp4" style="width:100%;border-radius:12px"></video>

*Video: the phone views — same server, same agents.*

Pull down on Now, Work or the Inbox to refresh. Swipe an Inbox row to the
left for Done, Snooze and Delete, or open the row’s action menu. The change count on a workspace
still opens its uncommitted changes.

## Files and Git

In **Work**, each workspace has **Files** and **Git**. In an agent, open
**Project tools** in the header; in a terminal, open **Terminal actions**.
Each tool takes the whole screen and returns to its project context.

**Files** browses folders and searches filenames within the current folder.
Tap a text file to edit; **Save** writes it back. Images and other supported
media open as previews. Leaving unsaved edits offers **Save**, **Discard**
or **Cancel**. If another process changed the file, your draft stays available
while you choose recovery. If a terminal changed folders, **Follow** explicitly
opens its current folder.

**Git** includes **Changes**, **History** and **PR**. Filter history by branch
or remote refs, open commits and compare working trees.
**Git actions** offers Fetch, Pull, Push, Commit, Commit and push, and pull
request creation. **Prepare** puts the command in a terminal for you to read
and submit. **Run when no agent is working here** starts it only when the
server's activity check allows it; otherwise the command is prepared with
the reason. **Ask an agent** sends the request through that agent's existing
channel. The terminal or agent shows the actual result.

Tap a terminal to attach to it. Opening it does not raise the phone
keyboard. Tapping the pane (or the header keyboard icon) opens the
phone keyboard; a one-row **key bar** sits immediately above it with Esc, Tab,
CTRL, Alt, arrows, `^C`, and — after a sideways scroll — Home, End,
Page Up, Page Down, and `| ~ / -`.

## Terminal keys

A terminal on the phone has extra keys the software keyboard lacks. They
are an accessory: they appear with the phone keyboard and go away with
it. Tap the pane, or the header's keyboard icon, to show both. Hide on
the row (or the header icon again) dismisses both.

```
esc  tab  ctrl  alt  ◀  ▲  ▼  ▶  ^C  …  hide
```

Swipe the row sideways for Home, End, Page Up, Page Down, and `| ~ / -`.
The hide button stays pinned on the right.

- **CTRL** and **ALT** are sticky: tap once, then the next key — from
  the phone keyboard or the row — is sent with that modifier. CTRL then
  `c` interrupts; ALT then `b` moves back a word; CTRL then ↑ is the
  modified arrow. An armed modifier lights up and expires after five
  seconds if nothing uses it.
- A key on the row never opens the phone keyboard; tapping the terminal
  does. The row rises above the keyboard so the prompt stays visible.
- The small undo/Done bar just above the phone keyboard is Safari's,
  not PiCode's — a web app cannot hide it.
- With a hardware keyboard attached, the row steps aside (the header
  icon brings it back).

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
