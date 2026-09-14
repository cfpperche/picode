# pi-browser

Read the web page the human has open in **PiCode's work browser** (the desktop
app) — the `browser` tool (ADR-0132, ADR-0134).

| Verb | What it returns | Tier |
|---|---|---|
| `snapshot` | the page as an accessibility tree: `role "name"` lines | read |
| `screenshot` | a PNG of the page on screen, written to a temp path | read |
| `events` | what the tab recorded: navigation, console, network | read |

Read-only by construction: the tool names a verb, the daemon maps it to a CDP
method the shell's catalog allows at that tier, and the default policy is
**read on the tab on screen** — it cannot click, type or navigate. Acting needs
a per-agent grant (Settings ▸ Browser, slice 4).

Nothing here runs without the desktop app: with no shell connected the daemon
answers that the desktop app is not connected, and the tool says so.

Install it like any pi package (it finds PiCode through `server.json` and the
install token, exactly as `pi-checklist` and `pi-inbox` do).
