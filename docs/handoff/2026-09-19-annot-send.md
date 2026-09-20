# 2026-09-19 — saved chip, Send 0 (feat/annot-send)

Owner: note "home" saved as a chip, strip still bare Send. The page's sync
never survived the trip: it posted pre-stringified text, which
WebMessageAsJson can hand back JSON-encoded a second time — a string with
no kind, dropped silently. (Slice 1's relay was likely never proven: pins
render in-page either way.)

## What landed
- Page posts the message OBJECT; host serializes once (deterministic).
- Chrome unwraps via `parseAnnotMessage` (single, double, or objects).
- Guards: `parseAnnotMessage` unit tests (10/10 file), Rust wire-contract
  test (8/8). Chromium harness: enter/pick/state all objects, state
  carries count + items.
- Rust relay untouched (cannot compile-check it here).

## Next up
- Owner rebuilds + restarts the shell, saves a note, Send shows Send 1.
- Sending needs a RUNNING agent terminal; otherwise "Nothing to send to".

## Debts
- End-to-end relay still Windows-only; unit + harness are the evidence.
