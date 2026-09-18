# pi-browser

Read the web page the human has open in **PiCode's work browser** (the desktop
app) — the `browser` tool (ADR-0132, ADR-0134).

| Verb | What it returns | Tier |
|---|---|---|
| `snapshot` | the page as an accessibility tree: `role "name"` lines | read |
| `screenshot` | the page on screen as an image the model sees | read |
| `events` | what the tab recorded: navigation, console, network | read |
| `evaluate` | the result of a JavaScript expression | act |
| `navigate` | the page after a navigation, inside the grant's domains | act |
| `cdp` | one Chrome DevTools Protocol method by name — needs Developer mode **and** the Full tier | full |

Read-only unless granted: the tool names a verb, the daemon maps it to a CDP
method the shell's catalog allows at that tier, and the default policy is
**read on the tab on screen** — with no grant it cannot click, type or
navigate. Acting needs a grant (Settings ▸ Browser). Each verb answers in its
own shape: `snapshot` as `role "name"` lines, `events` as the recorded ring,
`evaluate` as the value (or `the page threw: …`), `navigate` as where it went,
`cdp` as the method's JSON.

`cdp` is the one verb that names a protocol method itself, and it is the one
the owner opens deliberately: with **Developer mode** off (the default) the
daemon refuses it; with it on and the agent at the Full tier, every call is
listed in Settings ▸ Browser ▸ Developer mode, allowed or refused (ADR-0144).
Parameters go in as JSON: `{verb: "cdp", method: "Network.getAllCookies",
params: '{"urls":true}'}`.

Nothing here runs without the desktop app: with no shell connected the daemon
answers that the desktop app is not connected, and the tool says so.

Install it like any pi package (it finds PiCode through `server.json` and the
install token, exactly as `pi-checklist` and `pi-inbox` do).
