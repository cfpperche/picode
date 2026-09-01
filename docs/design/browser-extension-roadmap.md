# Browser extension — implementation roadmap

- **Date:** 2026-09-01
- **Status:** Track A in flight (ADR-0043).
- **Why:** agents already browse in an isolated Chromium; they cannot see
  the tab the human is on. A Chrome side panel + context menu sends that
  tab to an existing PiCode agent.

Canonical: [ADR-0043](../decisions/0043-browser-extension-native-host.md) ·
user guide [www/guide/browser-extension.md](../../www/guide/browser-extension.md).

Owner calls (2026-09-01): no CDP spike; stopped = Start and send;
screenshot opt-in; agent tool (Track D) out of v1; Chrome-only.

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **A — Sensor + native host** | ADR-0043 accepted |
| 2 | **B — Devices + install UX in Preferences** | A sends a prompt from Chrome Windows/WSL |
| 3 | **C — Actuator on the current tab** | A+B in daily dogfood |
| 4 | **D — `read_current_tab` pi package** | C exists, or the owner wants the agent to pull the tab |
| 5 | **E — Edge host copy, Firefox, store** | A stable |

## Refuse (all tracks)

| Temptation | Why not |
|---|---|
| Iframe the ADE in the side panel | Worse PiCode at 400px; still cannot capture the tab |
| React/Vite in `ext/` | Second bundle. Panel is a form |
| Replace `agent_browser` | MV3 loses waits/snapshots/record |
| `fetch` to localhost as the only channel | Port, cert, WSL |
| `<all_urls>` silent scrape / `chrome.debugger` | Not the product |
| Autostart an interactive (tmux) agent | Would kill the TUI from a context menu |
| Web Store / Firefox in v1 | Sideload + Chrome |

## Track A — Sensor (this slice)

Side panel + context menu + native host + `GET/POST /api/extension/*`.

| # | picode | host | agent | tab | action |
|---|---|---|---|---|---|
| 1 | down | ok | * | * | empty: "PiCode is not running." + Retry |
| 2 | * | missing | * | * | empty: "Install the PiCode host." + Open guide |
| 3 | up | ok | none | * | empty: "No agents yet." + Open PiCode |
| 4 | up | ok | managed idle | http(s) | `prompt` + `[browser-tab]` block |
| 5 | up | ok | managed busy/waiting | http(s) | same POST; server maps to `follow_up` |
| 6 | up | ok | stopped | http(s) | start managed, then prompt ("Start and send") |
| 7 | up | ok | interactive | * | refuse: "This agent is in the terminal." + Open PiCode |
| 8 | up | ok | * | `chrome://` / store / `file:` | refuse: "This page can't be sent." |
| 9 | up | ok | * | no selection | URL+title only |
| 10 | WSL + Chrome Windows | picode-desktop | managed idle | http(s) | same as 4 — the owner's machine |

Screenshot is a checkbox, off by default, JPEG under the 1 MB native-messaging cap.

## Later tracks (not this slice)

- **B:** `#/devices` kind=extension; Preferences one-liner; `picode-desktop extension-install` dogfood.
- **C:** content script on the active tab, per-origin grant, visible clicks.
- **D:** `packages/pi-tab` molde `pi-inbox`.
- **E:** Edge registry copy; Firefox; store.
