# 2026-09-18 — fix/computer-type-pacing

Found in the first live run of the computer tool from Claude Code (ADR-0154,
`picode mcp computer`): `type` into Windows 11 Notepad produced the text's
last character repeated (" 0123456789" landed as " 9999999999"; two
characters in one call survived, eleven did not). Cause: `KEYEVENTF_UNICODE`
packets sent with no gap; the app translates each message with the newest
packet's character. Fix in `desktop-shell/src/input.rs`: `TYPE_PACE` 5 ms
between characters and `MAX_TYPE` 2 000 per call (10 s, inside the daemon's
20 s window; the refusal tells the model to split).

Proof: cross `cargo check` clean; `make desktop-shell` builds; live proof
after `make desktop-restart` (owner's call): type a sentence into Notepad
and read it back with `snapshot`.

## Next up
- Owner: `make desktop-restart`, then ask the Claude Code terminal to type into Notepad again.

## Debts
- 5 ms is a measured floor on an idle machine; under load the same race can return. If it does, batch per word or post `WM_CHAR` to the focused control instead of `SendInput`.
