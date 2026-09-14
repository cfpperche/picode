---
description: Tools so a pi in a terminal can file questions and finished work into PiCode's Inbox.
---

# Inbox tools for pi

An agent working in a terminal has no way to reach you — it dumps text
into the transcript and hopes you are watching. Questions stall the run;
finished work scrolls away.

- **Where:** the **Inbox** app is PiCode core. The bridge is the `pi-inbox` package — install from [Packages](/guide/packages).
- **Not this:** not [session messages](/guide/communication) between agents. Without the package, a `pi` in a terminal cannot file into the Inbox; the Inbox itself still exists.

`pi-inbox` gives raw pi sessions two tools that file into the Inbox. It is an
**optional pi package — not part of PiCode core**: a `pi` outside PiCode is unaffected.

Install `packages/pi-inbox` from the PiCode repository. Guide for install
targets: [Packages](/guide/packages).

```sh
pi install -l /path/to/picode/packages/pi-inbox
```

## The two tools

| Tool | Blocking? | What happens |
|---|---|---|
| `notify_human` | no | A non-blocking FYI lands in the Inbox feed; the agent keeps working |
| `ask_human` | **yes** | The question lands in the Inbox and the agent's turn **ends**; your reply arrives later as a follow-up message — the agent parks, and a stopped agent picks the reply up on its next start |

Use `ask_human` when only you can decide (approvals, ambiguous specs);
`notify_human` for everything the agent merely must not keep silent
about.

## Where it runs

| | What you get |
|---|---|
| **Pi TUI** (terminal, inside PiCode's machine) | Both tools file into the Inbox; items carry the agent's name — or, for pi launched as a PiCode Agent CLI terminal, the terminal's identity so the Inbox reply reaches that terminal (see below) |
| **PiCode chat agents** | The same tools on every managed agent |
| **A `pi` elsewhere** (no reachable PiCode) | The tools return a soft explanatory result; nothing breaks |
| **PiCode core** | The Inbox app itself — the package only POSTs into it |

## Replying to a question from a terminal pi (since 0.2.0)

A `pi` launched as an Agent CLI terminal stamps its questions with the
terminal's identity (`sourceKind: "terminal"`). Replying in the Inbox
delivers the answer through that terminal's receiver straight into the
session that asked — the same door PiCode's own "Ask" uses, reversed.
The item only closes when the terminal takes the answer; if the terminal
moved on, restarted, or never processed it, the item reopens with your
reply kept for a retry.

Items filed by pi-inbox **0.1.x** from a terminal carry no terminal
identity (`pi (unmanaged)`); PiCode refuses to answer those silently and
tells you to reply in the terminal. Update the package to get the
delivery:

```sh
pi install -l /path/to/picode/packages/pi-inbox
```

## How it reaches PiCode

On every call the extension re-reads `<data dir>/server.json` (PiCode
rewrites it on each start, so port changes are picked up) and POSTs the
item to `POST /api/inbox` over loopback. Identity comes from
`PICODE_AGENT_ID`, which PiCode sets on every managed agent; a plain
terminal `pi` files as *pi (unmanaged)*. The self-signed certificate is
accepted for that single loopback request only.

## How you know it worked

Open the **Inbox** tab: a `notify_human` shows up instantly as an item
from your agent's name; an `ask_human` appears as a question you can
answer right there — the answer walks back into the agent's session as
its next message. No item and a soft "no reachable PiCode" result means
the package cannot see a running PiCode on this machine.
