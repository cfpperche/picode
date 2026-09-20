# 2026-09-19 — the page's hotkeys were eating the note (feat/annot-shield)

Owner: typed "te", pressed "s", GitHub's search took over — reproduced with a
real key: the keydown crossed the shadow boundary `composed`, and outside the
tree its target is the HOST (a div) — not a form field — so @github/hotkey
(the library GitHub ships) fired, preventDefault ate the "s", focus moved.
Same root cause hid a second bug: Enter-to-save compared `e.target` to the
input, which can never be true at a document listener; Enter had never saved.

## What landed
- Shield at the shadow root: keydown/keypress/keyup + pointer/mouse stop
  propagating to the page (insertion and focus are default actions:
  unchanged). Enter/Escape ask the card flag, not the retargeted target.
- Guards: Rust 11/11 (`the_script_shields_…`, `enter_saves_through_…`) and
  the JS source scan (12/12 annotate file). test-js + build green.
- Harness committed: `scripts/fixtures/annotate-hotkey.html` (real
  @github/hotkey from unpkg, steps in its header). Run against it, real keys:
  MAIN → value "te", panel open, focus stolen (the owner's screenshot);
  FIXED → value "tes", Saved via Enter, chip, panel closed, focus kept.
  Real semantic click on Save also saves (pointer shield proven harmless).

## Next up
- Owner: desktop-restart + deploy, then retype the note and press Send.

## Debts
- The shield covers bubble-phase page listeners (what the libraries use);
  a capture-phase page listener is out of reach without owning the DOM.
