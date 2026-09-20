# 2026-09-20 — term-keybar-scroll: bar scrolls again, IME still stays open
Follow-up to term-keybar-ime (ea6f7147): the IME fix worked but
touch-action:none + touchstart preventDefault killed the row's native
pan (owner: "o rolar ficou uma merda"). Researched the platform
contract (MDN: canceling touchstart cancels pan with the blur) and the
SO consensus (8692678, 20915251): gesture discrimination. Implemented:
touchmove past 8px slop scrolls the row by hand (velocity tracked),
release with velocity glides with friction 0.94/frame, stationary
release = tap = dispatch on touchend. Mouse/keyboard paths unchanged.
Tradeoff: key dispatch happens on touchend (one tap's ~80ms later than
before) — imperceptible for selector navigation; drag starting on a key
no longer sends a key (it scrolls instead — that is the point).

## Next up

- Owner device confirmation (scroll feel + keyboard persistence).
