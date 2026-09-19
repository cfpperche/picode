# 2026-09-19 — annotate hover was inverted (feat/annot-hover)

Owner: no blue hover before the first pick, hover stuck on after it, plus
Esc needing two presses. Root cause: the page script read open-state from
`node.style.display`, which starts as `""` — so `!== "none"` was true
before anything opened and false only after. Inherited from slice 1.

## What landed
- `cardOpen`/`menuOpen` flags own the state; all 8 inline-display reads
  gone. Guards: JS source-scan test + Rust absence test (old had 8 hits).
- Proven in Chromium against the real script: hover→block initially, none
  with the card open, block after Save, first-press Esc exits. Old script
  reproduced the owner's report in the same harness.
- Rust 7/7, annotate JS file 9/9.

## Next up
- Owner rebuilds + restarts the desktop shell (the script embeds in the
  binary) and re-tests hover on a live page.

## Debts
- None new; style inspector + settings row still open (v2c).
