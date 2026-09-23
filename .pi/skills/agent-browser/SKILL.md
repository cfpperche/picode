---
name: agent-browser
description: PiCode-only browser QA. Drive Chromium via the native agent_browser tool (npm:pi-agent-browser-native) against a scratch instance started by scripts/qa-scratch.sh. Use when opening PiCode itself, clicking the UI, screenshots, overlay audit, or end-to-end visual checks. Read the traps list before the first call.
---

# PiCode browser QA

Use the native `agent_browser` tool (not `bash agent-browser`). Load CLI
patterns only if needed: `agent_browser  args: ["skills", "get", "core"]`.

## Start from a scratch instance, not production (ADR-0086)

```bash
scripts/qa-scratch.sh start <name>     # builds this worktree with the UI embedded,
                                       # isolated HOME/PICODE_DATA, free port, prints the URL
scripts/qa-scratch.sh seed <name>      # a workspace, an agent and a terminal via the API
scripts/qa-scratch.sh stop <name>
```

`start` reports up only when **this** scratch's daemon answers on its port —
it wrote `server.json` for that port and is the listener — and `seed` refuses
any other instance. A healthy answer is not proof: on 2026-09-22 another
session's scratch took the port during the build, this daemon never bound,
and the old `seed` created a workspace, an agent and a terminal inside the
other session's instance. If `start` says the port is answered by someone
else, pick another port (`start <name> <port>`).

Production (`~/.picode/server.json`, usually :8445) is other agents' working
instance: never restart it, never `pkill` around it, and read it only when
the owner asks about a live report. It is HTTPS with a self-signed
certificate: `AGENT_BROWSER_IGNORE_HTTPS_ERRORS=1`; its `/api/version` and
`/api/health` answer empty without a browser session.

## Before you say a UI task is done

1. Console sweep: `eval` `JSON.stringify(window.__picodeOverlayAudit())`.
   `ok: false` is FAIL.
2. Screenshot + `read` the PNG (keep it in `var/screenshots/`). Eval alone
   is not a visual verdict.
3. Clickability: a real click on every primary control you touched. After
   close, assert `getComputedStyle(el).display === "none"`.
4. Open/close cycles stay closed. State survives reload.

## Rebuild or you are testing the old binary

UI is `go:embed` of `internal/web/public/`. `qa-scratch.sh start` builds;
after an edit, `stop` + `start` (or `make web && go build -tags embedui`
into the same path, then `eval location.reload()`).

## Traps (each one cost a session an hour; they look like product bugs)

- `eval` shares one page scope: a second `const x` throws — wrap every
  eval in `(() => { … })()`.
- Radix menus/popovers open on pointerdown: `el.click()` inside eval does
  nothing — use a real `click` on the trigger.
- No `:has-text()` selectors; match text inside an eval. Sidebar rows are
  not buttons: navigate with `location.hash = '#/agent/<id>'`.
- The first eval right after `open` often runs before render — `wait` for
  the surface's root selector, then eval.
- Hidden views stay mounted: scope selectors to the view
  (`#inspector button[type=submit]`, not `button[type=submit]`).
- Time-sensitive checks (throttle, debounce) fit in ONE eval with
  `setTimeout` and a `window.__log` array; each CLI call costs ~1 s.
- Mobile shell: `close --all` → `set viewport 390 844` → `open …/?mobile=1#/`
  → `wait "#m-app"`. A desktop session that once landed on mobile stays
  stuck until `close --all`.
- Over plain HTTP by IP the page is not a secure context: `crypto.randomUUID`
  is undefined and a throw in an effect blanks `#root` with no console
  error. Use `localhost`.
- Never `pkill -f <pattern>` — the pattern matches your own shell. Stop
  daemons by port (`fuser -k <port>/tcp`) or with `qa-scratch.sh stop`.
- Never kill tmux sessions by prefix: every real session is `picode-…` too.
  A `grep '^picode-' | xargs kill-session` sweep killed 29 sessions on
  2026-09-06. Kill only exact names your own scratch instance's API returned.
- Other sessions restart production and kill processes by name; a scratch
  daemon that dies mid-run was probably hit — check `journalctl --user -u
  picode` before blaming the code.
- Terminals a scratch launches live in that instance's own tmux server
  (`-S <worktree>/var/qa/<name>/data/tmux.sock`, ADR-0139), not the owner's:
  pass `-S <that socket>` to every tmux command that should see them;
  without it `tmux ls` shows the owner's sessions and none of the scratch's.
- The docs fixture (`make fixture`, :18740) seeds a dirty git repo on
  purpose; never `git checkout -- .` inside it.

Pair with `/skill:visual-review` for the visual-card. This skill acts;
visual-review judges pixels.
