# 2026-09-20 — term-keybar-ios: key-bar taps closed the IME and never moved the TUI
Owner report (iOS, pi model selector): tapping ▲ in the key bar closed
the keyboard and the selection did not move. Root cause: the key action
lived on click — on iOS the tap blurred xterm first (IME closed, bar hid
itself) and the click was suppressed, so no escape sequence was ever
written. Fix: keys and sticky modifiers act on pointerdown with
preventDefault (focus stays, IME stays open) under a 350ms dedupe guard;
the accessory refocuses xterm unconditionally after sending (user gesture
reopens the IME); sendTermSeq reports the socket truth (`open`) and the
screens toast "Terminal reconnecting" instead of eating keys on a dead
socket. Tests updated to the new contract (7 pass).
Blind spot: iOS blur behaviour is not reproducible in headless tests —
needs the owner's device to confirm the keyboard stays open.

## Next up

- Owner device confirmation; then Fatia ∞ stays deferred until
  ADR-0091 is re-measured.
