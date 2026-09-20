// Phone IME geometry (ADR-0044 amendment): the software keyboard overlays
// the layout viewport on iOS, so a bottom composer or extra-keys row at
// 100dvh sits under it. visualViewport is the source of truth; Chrome's
// VirtualKeyboard API is an optional extra. Pure: no DOM.
//
// Pin the shell to the visual viewport only while the keyboard is actually
// covering pixels. Pinning at rest (always writing --vv-height) left a
// black strip under the terminal on iOS 26.

export const KEYBOARD_INSET_THRESHOLD = 80;
export const HARD_KEYBOARD_WAIT_MS = 300;

export function keyboardInset({ innerHeight, vvHeight, vvOffsetTop }) {
  const inner = Number(innerHeight) || 0;
  const height = Number(vvHeight) || 0;
  const top = Number(vvOffsetTop) || 0;
  return Math.max(0, Math.round(inner - height - top));
}

// Extra keys travel with the phone keyboard. Programmatic xterm focus on
// attach is not a user tap — iOS will not open the IME, so the row must
// stay hidden until the user arms it (tap the pane or the header icon).
export function extraKeysVisible({ termFocused, hardKeyboard, userArmed }) {
  return !!termFocused && !hardKeyboard && !!userArmed;
}

// Hardware keyboards do not shrink the visual viewport. A coarse pointer
// (a phone) with no inset is still a software keyboard we failed to
// measure — show the row. A fine pointer with no inset after a short
// wait is a real keyboard; hide the accessory.
export function hardKeyboardLikely({ termFocused, inset, elapsedMs, finePointer }) {
  if (!termFocused) return false;
  if ((elapsedMs || 0) < HARD_KEYBOARD_WAIT_MS) return false;
  if ((inset || 0) >= KEYBOARD_INSET_THRESHOLD) return false;
  return !!finePointer;
}

export function inputIsFocused(el) {
  if (!el || el === el.ownerDocument?.body) return false;
  const tag = el.tagName;
  if (tag === "TEXTAREA" || tag === "INPUT" || tag === "SELECT") return true;
  return !!el.isContentEditable;
}

// Pin the shell only when the IME is covering the page. `restHeight` is
// the visual viewport while no field is focused. A shrink against that
// baseline catches iOS overlaying the keyboard without changing
// innerHeight. Focus alone is not enough — attach-time focus used to
// leave a black strip without opening the IME.
export function shellLayout({
  innerHeight, vvHeight, vvOffsetTop, scale, restHeight, inputFocused,
}) {
  if (scale != null && scale !== 1) return null;
  const inset = keyboardInset({ innerHeight, vvHeight, vvOffsetTop });
  const vsRest = restHeight != null
    ? Math.max(0, Math.round(Number(restHeight) - (Number(vvHeight) || 0)))
    : 0;
  // innerHeight − vvHeight can exceed the threshold at rest on iOS
  // (100vh vs the home indicator). That is not the IME. Pin only when
  // a focused field actually shrank the visual viewport against the
  // unfocused baseline — otherwise first-open focus left a black strip.
  const keyboardOpen = !!inputFocused && vsRest >= KEYBOARD_INSET_THRESHOLD;
  if (!keyboardOpen) {
    return { keyboardOpen: false, height: 0, offsetTop: 0, inset: 0 };
  }
  return {
    keyboardOpen: true,
    height: Math.round(Number(vvHeight) || Number(innerHeight) || 0),
    offsetTop: Math.round(Number(vvOffsetTop) || 0),
    inset: Math.max(inset, vsRest),
  };
}

export function sendTermSeq(entry, seq, hostHadFocus) {
  if (!entry) return { sent: false, open: false, refocus: false, bytes: "" };
  const bytes = entry.sticky ? entry.sticky.applyKey(seq) : seq;
  const open = !!(entry.sock && entry.sock.readyState === 1);
  if (open) {
    entry.sock.send(typeof bytes === "string" ? new TextEncoder().encode(bytes) : bytes);
  }
  // `open` is the honest one — a closed socket silently ate keys, and the
  // caller must know instead of reporting a key that went nowhere.
  return { sent: open, open, refocus: !!hostHadFocus, bytes };
}
