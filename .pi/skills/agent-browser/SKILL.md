---
name: agent-browser
description: Browser automation with a real Chromium via the agent-browser CLI (Vercel). Use when the task requires opening PiCode itself, clicking through its UI, filling forms, reading page content, live screenshots, console/network inspection, or testing the web UI end-to-end. Adapted from the agentdeck-proven skill.
---

# agent-browser

`agent-browser` is a stateful CLI that drives a Chromium session. Every bash
invocation continues the same browser session until you run `close`.

## Load the full guide first

The CLI ships its own version-matched skill docs. Before automating, run:

```bash
agent-browser skills get core
```

For specialized scenarios: `agent-browser skills list` (exploratory testing,
Electron, cloud browsers, ...).

## Quick reference

```bash
agent-browser open <url>          # Navigate (starts the session)
agent-browser snapshot            # Accessibility tree with @refs (e.g. @e3)
agent-browser click @e3           # Click element by ref or CSS selector
agent-browser type @e5 "text"     # Type into element
agent-browser fill @e5 "text"     # Clear field and fill
agent-browser press Enter         # Press key (Tab, Control+a, ...)
agent-browser select <sel> <val>  # Select dropdown option
agent-browser get text <sel>      # text | html | title | url | value | attr <name>
agent-browser read <url>          # Fetch agent-readable page text
agent-browser eval "<js>"         # Run JavaScript in the page
agent-browser screenshot out.png  # Save screenshot (view it with the read tool)
agent-browser pdf out.pdf         # Save page as PDF
agent-browser wait <sel|ms>       # Wait for element or time
agent-browser close               # Close the browser session
```

## Standard workflow

1. `open <url>` → 2. `snapshot` (discover @refs) → 3. act (`click`, `fill`...)
→ 4. verify (`snapshot` again / `get text` / `screenshot` + `read`) →
5. `close`.

## Tips (the ones that bite)

- **Refs go stale after any action that changes the page** — snapshot again
  before every action.
- Prefer `snapshot` for acting; screenshots verify *visuals*, not structure.
- Session persists across commands — no need to re-open between steps.
- **Zombie discipline**: sessions survive aborted tasks. Start tricky work
  with `close` (a failed close is harmless — but see the session-reset note
  below) and ALWAYS `close` before finishing.
- **Session-reset gotcha (observed here)**: running `close` immediately
  followed by `open` can leave the browser on `about:blank`. If
  `get url` shows `about:blank` after open, just re-run `open` once.

## Setup

```bash
npm install -g agent-browser
```

## Testing PiCode itself

### Exploratory QA recipe (learned the hard way)

Functional click-throughs are NOT enough. Before declaring UI work done, also:

1. **Clickability sweep**: attempt a REAL click on every primary control
   (`click` fails loudly when an element is covered — CSS `display` beating
   the `hidden` attribute creates invisible full-surface overlays).
2. **Hidden means pixels gone**: after every close, assert
   `getComputedStyle(el).display === "none"` — NEVER trust `.hidden` or
   the `hidden` attribute alone. Author `display:flex/grid` beats the UA
   `[hidden]` rule. This is how the dock closer shipped as a lie.
3. **Open/close cycles**: dock/panel/tab/modal — open, close, verify it STAYS
   closed after unrelated interactions (sidebar clicks, reloads). Auto-reopen
   after close is a bug users hit immediately.
4. **Persistence across reload**: state should survive; selection/layout
   should be sane with zero running agents (no dead zones).
5. **IDE conventions**: agents open as TABS in the editor area; docks/panels
   open only by explicit user action; closers always visible on hover.
6. **Console + network sweep** at the end: zero errors expected.

PiCode serves HTTPS (self-signed or mkcert) with a **runtime-configurable
port** — don't assume 8445:

```bash
# Where is the server right now?
cat ~/.picode/server.json     # {"url":"https://localhost:8445", ...}

# HTTPS route — ignore the local cert once per session:
export AGENT_BROWSER_IGNORE_HTTPS_ERRORS=1
agent-browser open https://localhost:8445

# Plain-HTTP route (skip certs entirely): restart the server with
# PICODE_INSECURE=1 ./bin/picode  → http://localhost:8445
```

### The go:embed rule (critical for UI work)

The UI is baked into the binary (`internal/web/public/` via `go:embed`).
Editing app.js/style.css does NOTHING until you rebuild:

```bash
go build -o bin/picode ./cmd/picode && \
  tmux kill-session -t picode-dev 2>/dev/null; \
  tmux new-session -d -s picode-dev -c "$PWD" './bin/picode 2>&1 | tee -a /tmp/picode-dev.log'
```

Then in the browser session: `agent-browser eval "location.reload()"`.

### Known UI map (verify with snapshot; refs change)

- User menu: sidebar footer button (hostname + "this machine") — theme
  Light/System/Dark, Settings link.
- Settings route: `#/settings` — Appearance (theme cards), Server (port —
  triggers a real rebind; the page reconnects to the new port), System, About.
- Workspace card: `Run` (managed mode) / `Stop`; conversation panel with
  tool pills (click to expand) and composer (Prompt/Steer/Follow-up).

### Verification combos

```bash
agent-browser snapshot | grep -i "Run agent"      # empty state present
agent-browser eval "document.documentElement.dataset.theme"   # theme applied
agent-browser screenshot var/screenshots/x.png    # then `read` it (pixels!)
agent-browser eval "localStorage.getItem('picode-theme')"
```

Interactive checks complement `.pi/skills/visual-review`: use this skill to
ACT on the UI, and visual-review's capture→read loop to JUDGE pixels.
