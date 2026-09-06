// Phone IME geometry (ADR-0044 amendment): the software keyboard overlays
// the layout viewport on iOS, so a bottom composer or extra-keys row at
// 100dvh sits under it. visualViewport is the source of truth; Chrome's
// VirtualKeyboard API is an optional extra. Pure: no DOM.

export const KEYBOARD_INSET_THRESHOLD = 80;
export const HARD_KEYBOARD_WAIT_MS = 300;

export function keyboardInset({ innerHeight, vvHeight, vvOffsetTop }) {
  const inner = Number(innerHeight) || 0;
  const height = Number(vvHeight) || 0;
  const top = Number(vvOffsetTop) || 0;
  return Math.max(0, Math.round(inner - height - top));
}

export function extraKeysVisible({ termFocused, hardKeyboard }) {
  return !!termFocused && !hardKeyboard;
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

// CSS variables for the mobile shell. Pinch-zoom (scale !== 1) must not
// rewrite them: shrinking the app to the pinch rectangle is worse than
// leaving the last keyboard-aware size.
export function shellVars({ innerHeight, vvHeight, vvOffsetTop, scale }) {
  if (scale != null && scale !== 1) return null;
  const height = Math.round(Number(vvHeight) || Number(innerHeight) || 0);
  const offsetTop = Math.round(Number(vvOffsetTop) || 0);
  return {
    height,
    offsetTop,
    inset: keyboardInset({ innerHeight, vvHeight, vvOffsetTop }),
  };
}

// A bar key: apply sticky modifiers, write the socket, and only refocus
// xterm when it already held focus (so a tap never summons the IME).
export function sendTermSeq(entry, seq, hostHadFocus) {
  if (!entry) return { sent: false, refocus: false, bytes: "" };
  const bytes = entry.sticky ? entry.sticky.applyKey(seq) : seq;
  if (entry.sock && entry.sock.readyState === 1) {
    entry.sock.send(typeof bytes === "string" ? new TextEncoder().encode(bytes) : bytes);
  }
  return { sent: true, refocus: !!hostHadFocus, bytes };
}
