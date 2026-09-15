# 2026-09-15 — browser-newtab-recheck

The "quick win" the owner picked first: fix the "new browser tab" crash.

## Done

- Reproduced? **No.** In a scratch web instance the click produced no console
  error, the React root stayed alive and the route moved to `#/` — the crash
  the topic note recorded ("verified pre-existing on a main scratch") does not
  reproduce on this path today.
- Also read the defect class that produced the blank window: `showFullUrl` is
  already hoisted above the effect that reads it (`WebTab.jsx:23` vs `:32`), so
  it is not that either.
- The topic note now carries the measurement instead of the claim, and names
  the shell path as the likely home (`window.__TAURI__` only exists there).

## Next up

- Reproduce in the deployed desktop shell (a real click on the tab-strip
  button) before changing any code; then fix with evidence.
- Then steps 1–5 of ADR-0143 (terminal agents as principals).
