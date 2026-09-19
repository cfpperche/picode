# 2026-09-19 — Send posted to the wrong door (feat/annot-prompt)

Owner: chip saved, toast "Annotations saved; the terminal is not running an
agent CLI, so nothing was sent" — and nothing arrived. Both halves of the
toast were wrong: the terminal was fine, the URL was not. `sendAll` posted
`{message, paths}` to `/drop` (one FILE upload: `{name, mime, data}`), which
400'd "file data is required". The prompt door (`/prompt`) owns that shape
(ADR-0089); the single-send before it made the same call, so delivery never
worked — only staging did.

## What landed
- `sendAll` posts to `/prompt`; a refused paste toasts the server's reason
  instead of a canned wrong one. No test change (no unit covers the URL;
  JSX builds).
- Debt recorded: `/prompt` takes ≤4 files per paste — 3+ notes save but do
  not deliver in one call (chunk vs. combined note vs. raising the cap).

## Next up
- Owner re-tests Send 1 into a RUNNING agent terminal (first `running`
  terminal in list order receives it).
