# Inbox tools for pi

An agent working in a terminal has no way to reach you — it dumps text
into the transcript and hopes you are watching. Questions stall the run;
finished work scrolls away.

`pi-inbox` gives raw pi sessions two tools that file into PiCode's
**Inbox** app. The Inbox itself **is** PiCode core (approvals, questions
and results in one feed, on desktop and phone); this package is the
bridge that lets a `pi` you installed use it. It is an **optional pi
package — not part of PiCode core**: without it, nothing changes
anywhere, and a `pi` outside PiCode is unaffected.

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
| **Pi TUI** (terminal, inside PiCode's machine) | Both tools file into the Inbox; items carry the agent's name |
| **PiCode chat agents** | The same tools on every managed agent |
| **A `pi` elsewhere** (no reachable PiCode) | The tools return a soft explanatory result; nothing breaks |
| **PiCode core** | The Inbox app itself — the package only POSTs into it |

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
