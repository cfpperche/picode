# pi-browser

Drive the browser tab beside the agent's session in the **PiCode desktop app**
— the `browser` tool (ADR-0132, ADR-0172).

| Verb | What it returns | Who |
|---|---|---|
| `open` | opens the split beside this session, optional url | any identified caller |
| `snapshot` | the page as an accessibility tree: `role "name"` lines | the session tab |
| `screenshot` | the page on screen as an image the model sees | the session tab |
| `events` | what the tab recorded: navigation, console, network | the session tab |
| `click` | clicks a selector, or a point | the session tab |
| `type` | inserts text, optionally into a selector | the session tab |
| `press` | a key: Enter, Tab, Escape, an arrow, or one character | the session tab |
| `evaluate` | the result of a JavaScript expression | the session tab |
| `navigate` | the page after a navigation, any http(s) URL | the session tab |
| `cdp` | one Chrome DevTools Protocol method by name — needs Developer mode **and** the Full tier | grant |
| `history` | where the human has been — off until Settings ▸ Browser allows it | setting |

This is not a headless browser. Unattended browsing stays on the runtime's
own tool. An identified caller does not need a stored grant to drive the
tab beside its session; closing the split stops it. A caller with no
identity cannot drive a tab. Raw CDP stays behind Developer mode and the
Full tier.

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
