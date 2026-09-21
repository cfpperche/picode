---
description: The Inspector's Servers tab — what is listening on this machine, what it is, and what PiCode may do about it.
---

# Dev servers

Your agent runs `npm run dev` and a server comes up on some port. The
**Servers** tab in the Inspector rail — right-hand side, beside Changes,
Files and PR — tells you which port, what is on it, and who started it.

A dev server is the one thing PiCode's file preview cannot show: it exists
only while its process does, and its own machinery (hot reload, WebSockets,
absolute URLs) breaks under a proxy. So **PiCode does not proxy it.** The
panel finds it; the page opens talking to your server directly.

Servers is the one Inspector tab that does not follow what you have
selected — what is listening on the machine is true whether or not an agent
is open — so the tab is there even with nothing selected.

## What a row tells you

| | |
|---|---|
| **The name** | What the page calls itself, when the port serves a page |
| **The address** | `http://localhost:5173` and so on |
| **The owner** | `terminal "sidebar" · PiCode · 2 h`, or *started outside PiCode* |
| **A pill** | *HTTPS*, *not a page*, *starting*, *not answering* |

*starting* means the process is under a minute old and has not answered
yet. *not answering* means it is listening but said nothing — usually a
server still booting, sometimes one that is wedged.

*not a page* matters more than it looks: a port that answers JSON, a 404 or
a control channel is listed with its address but **no Open button**, because
there is nothing a browser tab could usefully show. Only pages get a button.

## What is listed, and what is not

A port is a row when **a PiCode terminal or agent owns it**, or when it is
one of the usual dev ports *and* something is listening on it right now.

Everything else on your machine — your database, a private API, PiCode's own
daemon — is not this panel's business and is not listed. Ports PiCode can
see but cannot attribute wait behind **Show N more**.

To find out what a port is, PiCode makes one plain HTTP request to it, and
one TLS attempt if that got nothing. This is the only time the daemon talks
to your own servers. The answer is cached for two minutes (ten seconds for a
port that said nothing, so a booting server updates quickly), and
**Refresh** asks again — a panel left open does not fill your dev server's
log.

## What you can do with a row

| Action | Notes |
|---|---|
| **Open in PiCode** | The page in PiCode's own browser tab |
| **Open in browser** | Your normal browser |
| **Copy address** | |
| **Show terminal** | Jumps to the terminal or agent that started it |
| **Stop server…** | Only for a server PiCode started. Confirms first, naming the process and its terminal |
| **Hide from the list** | For noise you do not want to see |

**Stop** sends a polite termination first; a process that survives offers
**Force stop**. PiCode re-checks at the moment you press it that the process
is still the one holding that port, so a stale row can never signal
something that reused the process id.

**Hide** remembers *that listener*, not that port — start a new server on
the same port and it appears as a new row. Hidden rows are counted below the
list and come back with **Show again**.

Stop and Hide both need PiCode to know which process is behind the port, so
a server started outside PiCode offers neither: it gets Open and Copy
address, and says so.

## Things it deliberately does not do

- **No restart.** PiCode does not know the command that started your server.
  Stop is offered; starting again is your shell.
- **No proxy.** Hot reload, browser developer tools and service workers keep working
  because nothing sits in between.
- **Nothing outside Linux.** Attribution reads `/proc`. On macOS, and on a
  Windows-native daemon, the panel says so in one line, lists only the usual
  dev ports that answer, and refuses Stop.
- **Not on the phone.** The Inspector rail is a desktop surface.

A server on an unusual port that PiCode cannot attribute will not appear at
all — type its address into the browser tab instead.

## Opening a link from a terminal

Ctrl+click an `http://` link printed in a terminal (or use the pane menu)
and it opens where you said web links should open. Two rows in **Browser**
settings decide it — one for `localhost` addresses, one for everything else
— and both default to PiCode's own tab. A path under the terminal's folder
still opens as a file.

The Servers panel's own **Open** button ignores that preference on purpose:
you clicked a control inside PiCode, so you get PiCode.
