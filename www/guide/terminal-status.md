# Terminal status for coding CLIs

A coding CLI running inside a PiCode terminal can report its own
lifecycle, so the sidebar shows when it is **working** and when it
**needs you** — the same states Pi agents already show (ADR-0056).

PiCode does not read the terminal's pixels. The CLI tells us: you wire
a small hook (Claude Code) or a notify command (Codex), and it POSTs
one line to the PiCode server whenever the state changes. PiCode
prepares everything else: every terminal you open carries
`PICODE_TERM_ID` and `PICODE_TERM_URL` in its environment, so the hook
knows which terminal it belongs to and where to report.

## States

| State | Meaning | Where you see it |
|---|---|---|
| `working` | the CLI started a turn and has not finished | spinner on the terminal row and tab |
| `needs-you` | the CLI is waiting on you (permission prompt, question) | accent "Needs you" chip |
| `idle` | the CLI finished its turn | nothing — quiet means idle |

A `working` report expires after 30 minutes of silence. No chip means
"no signal" — never a guess.

## Claude Code

Add to `~/.claude/settings.json` (merge into the existing `hooks`
key if you have one):

```json
{
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "picode-hook working" }] }],
    "Stop":             [{ "hooks": [{ "type": "command", "command": "picode-hook idle" }] }],
    "Notification":     [{ "hooks": [{ "type": "command", "command": "picode-hook needs-you" }] }]
  }
}
```

`Notification` fires for permission prompts and idle prompts — exactly
the "needs you" moments. The hook input also arrives on stdin if you
want more detail later; the state word is all PiCode needs.

## Codex

Codex reports the end of a turn. In `~/.codex/config.toml`:

```toml
notify = ["picode-hook", "idle"]
```

Codex has no open hook for "waiting on approval", so a Codex terminal
shows working/idle only. That is Codex's surface today, not a PiCode
limitation we chose.

## The `picode-hook` helper

Save as `~/.local/bin/picode-hook`, `chmod +x`:

```sh
#!/bin/sh
# Report this terminal's guest-CLI state to PiCode (ADR-0056).
# Needs PICODE_TERM_ID + PICODE_TERM_URL (set by PiCode on every
# terminal it opens) and the install token for the API.
[ -n "$PICODE_TERM_ID" ] || exit 0
TOKEN=$(cat "${PICODE_DATA:-$HOME/.picode}/token" 2>/dev/null)
curl -fsS -o /dev/null --max-time 3 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"state\":\"$1\",\"cli\":\"${PICODE_HOOK_CLI:-}\"}" \
  "$PICODE_TERM_URL/api/terminals/$PICODE_TERM_ID/state" 2>/dev/null || true
```

Tell the hook which CLI it serves so the tooltip can name it, once per
shell (or hardcode it in the snippet):

```sh
export PICODE_HOOK_CLI=claude-code   # or codex, grok, opencode, …
```

Notes:

- Outside a PiCode terminal (`$PICODE_TERM_ID` empty) the hook is a
  no-op — safe to install globally.
- The endpoint is the ordinary authenticated API (ADR-0049): the
  helper sends the install token, same as every non-browser client.
- A failed report is silently dropped: status is a courtesy signal,
  never something worth alerting about.
