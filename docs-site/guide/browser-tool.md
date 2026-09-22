---
description: Let an agent drive the browser beside its session.
---

# Browser tools for pi

Let an agent open a browser beside its session and drive that tab: navigate, click, type, and read the page. You see the same tab and can click and sign in. Closing the split stops it.

- **Where:** install `pi-browser` ([Packages](/guide/packages)). The agent calls `open`, then `navigate`, `click` or `type` on the tab beside its session.
- **Not this:** not a headless browser. Unattended browsing stays on the headless tool the runtime already provides (`agent-browser` in this project). Not the [Chrome extension](/guide/browser-extension) either — that sends your Chrome tab to an agent.

Claude Code, Codex and the other agent CLIs get the same tool over MCP:
[PiCode tools for other agent CLIs](/guide/picode-mcp).

`pi-browser` is an **optional pi package — an extension, not part of PiCode
core**. It reaches the daemon over the same authenticated API every other
call uses (the install token); it opens no port and adds no credential.

Install `packages/pi-browser` from the PiCode repository. Guide for install
targets: [Packages](/guide/packages). Pick **This agent** to reach one agent
only, or **Global** so a plain `pi` in a terminal sees it too.

## What it can do

| Verb | Answer |
|---|---|
| `snapshot` | The page as roles and names (the accessibility tree, no script runs) |
| `screenshot` | The page as an image the agent sees, for when the look matters |
| `events` | What the tab recorded since the sequence number you last saw |

These three are **read**, and read is the default: nobody has to grant
anything, and no grant is needed to keep them.

## Reading is the ceiling until you grant more

An agent with no grant reads **the tab you have on screen** — the one you are
looking at, not any tab, and not one it chooses. Anything beyond that needs a
grant, per agent:

- `act` adds `evaluate` (run an expression in the page) and `navigate` (go to
  a URL);
- `navigate` also needs the destination's origin in the grant's domain list;
  `file:`, `data:` and `javascript:` are never allowed, listed or not;
- `full` is the tier that reaches outside the page (downloads, uploads).

The grants editor is **Settings ▸ Browser**: one row per agent — pick the
tier, and for Act or Full list the hosts it may reach (paste-friendly:
scheme, path and port are stripped). It saves per row and the change is
live for the agent's next command.

## Two things it needs

1. **The desktop app, running and connected.** The work browser is a native
   WebView2 view hosted by the PiCode desktop shell — the enforcement lives
   there. Without it the agent is told plainly: *the desktop app is not
   connected*.
2. **A visible package.** Install scope decides who sees it: **This agent**
   is private to that agent, and the pane offers it for the CLIs whose launch
   can carry the entries (Pi and Omp); **Global** (`~/.pi/agent`) and the
   workspace layer are visible to a plain `pi` too.

## Managed agent, or a plain `pi` TUI?

Both work, and the difference is identity, not connectivity:

| Where pi runs | Identity | What it gets |
|---|---|---|
| An agent PiCode manages | it reports its agent id | its own grant, read by default |
| A `pi` TUI in a PiCode terminal | none | read, on the tab on screen |
| A `pi` TUI you started yourself | none | the same, if the package is visible to it |

A grant belongs to an **agent id**, so only a managed agent can be granted
`act`. Identity is an assertion, not a proof (ADR-0134): the trust boundary
is the install token, and this policy shapes agents that cooperate rather
than containing one that does not.
