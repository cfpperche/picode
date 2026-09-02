# Browser extension — implementation roadmap

- **Date:** 2026-09-01
- **Status:** Track A + B shipped. Track C (actuator) in flight (ADR-0054).
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

## Track B — Devices + Windows host install

- `make desktop` builds `picode-desktop.exe` (tray, GUI) **and** `picode-nmh.exe` (console native host). Chrome on Windows cannot use the tray binary (`windowsgui` + `--parent-window`).
- `picode-desktop extension-install` copies `picode-nmh.exe` next to the tray and registers that path.
- Side panel pings `/api/devices/ping` with `kind=extension`. `#/devices` shows **Chrome extension** on this machine. Preferences → Server: one line + Open guide when disconnected.

## Track C — Actuator (shipped on this branch)

ADR-0054: the agent's reply may end in one ```picode-act fenced block;
the server validates it into a batch (`act_batches`, migration 021), the
panel polls for it via the native host, asks once per origin, and executes
it action by action with visible highlights. Outcomes go back as one more
watched turn; 3 rounds hard cap; Stop ends the loop. The panel must stay
open — a claimed-but-unexecuted batch is resumable, a pending one expires
after 10 minutes.

| # | grant | tab | batch | action |
|---|---|---|---|---|
| 1 | absent | on origin | any | ask Allow / Not now; Not now = stopped result |
| 2 | granted | other tab | pending | blocked=origin; not claimed; polls resume |
| 3 | granted | on origin | pending | claim, execute with highlights |
| 4 | * | navigates mid-run | claimed | remaining steps = "page changed"; result posted |
| 5 | * | * | settled, no block | watching flips false; "Done." |
| 6 | * | * | round ≥ 3 | last round; no follow-up |
| 7 | * | * | Stop | stopped result; loop ends |
| 8 | * | observed by another run | send with act | 409 busy; the reply still arrives |

## Later tracks

- **D:** `packages/pi-tab` molde `pi-inbox`.
- **E:** Edge registry copy; Firefox; store.
