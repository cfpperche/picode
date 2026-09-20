# 2026-09-20 — term-keybar-ime: IME stays open across key-bar taps
Follow-up to term-keybar-ios (246bf2cc). Owner confirmed the TUI moves
now, but iOS still closed the keyboard on every tap. WebKit ignores
pointerdown cancelation for the blur; what it honors is a NON-PASSIVE
touchstart with preventDefault (CodeMirror/Monaco toolbar pattern).
Touch now owns the action via a delegated listener on the toolbar
(data-act/data-arg), mouse keeps pointerdown, Enter/Space keep click —
one path per input, so no dedupe window can eat rapid repeated arrows.
.m-key gains touch-action:none/user-select:none.
Known tradeoff: a drag starting on a key can no longer scroll the bar;
the gaps and the pinned hide button still can (Fable-first row is
designed to fit 390px anyway).
Blind spot: headless tests cannot reproduce iOS blur — owner device is
the verdict.

## Next up

- Owner device confirmation; if the IME still closes on any specific
  key, instrument with a focus/blur log on the xterm textarea.
