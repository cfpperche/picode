# Chrome extension (ADR-0043)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

`ext/` is a sideload MV3 extension: side panel + context menu send the
current tab (URL, title, selection, optional JPEG) to an existing agent.
It is not an App (ADR-0036) and not a pi package (ADR-0010). Transport is
Chrome native messaging to the same product binary (`picode` on
Linux/macOS; `picode-desktop` on Windows/WSL), which re-reads
`server.json` — or `remote.json` when the PiCode is on another machine
(ADR-0050) — and calls `GET/POST /api/extension/*` with the install
token. Isolated Chromium
(`agent_browser`) stays the automation engine. v1 is Chrome-only and
sensor-only; actuating the page is a later track.
