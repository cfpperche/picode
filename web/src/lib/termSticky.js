// Sticky modifiers for the phone's terminal keys (ADR-0044 amendment):
// tap Ctrl or Alt once, and the next key — from the phone keyboard or
// from the key bar — is sent with that modifier, then the modifier
// drops. This is what Termux, Blink and mtmux do; a soft keyboard has
// nothing to hold down. An armed modifier that nobody uses expires, so
// a forgotten Ctrl cannot turn a later `c` into an interrupt.
//
// Pure state machine: no DOM, no xterm. ShellTerm passes term.onData
// through apply(); the key bar calls arm()/applyKey().

export const STICKY_TIMEOUT_MS = 5000;

const ARROWS = { "\x1b[A": "A", "\x1b[B": "B", "\x1b[C": "C", "\x1b[D": "D", "\x1b[H": "H", "\x1b[F": "F" };

export function createSticky(opts = {}) {
  const now = opts.now || (() => Date.now());
  const timeout = opts.timeout ?? STICKY_TIMEOUT_MS;
  const listeners = new Set();
  let ctrl = 0; // 0 = off, else the time it was armed
  let alt = 0;

  function expire() {
    const t = now();
    if (ctrl && t - ctrl > timeout) ctrl = 0;
    if (alt && t - alt > timeout) alt = 0;
  }
  function emit() {
    for (const fn of listeners) fn(state());
  }
  function state() {
    expire();
    return { ctrl: !!ctrl, alt: !!alt };
  }
  function arm(mod) {
    expire();
    if (mod === "ctrl") ctrl = ctrl ? 0 : now();
    else if (mod === "alt") alt = alt ? 0 : now();
    emit();
    return state();
  }
  function disarm() {
    ctrl = 0;
    alt = 0;
    emit();
  }
  // apply: a single character typed on the phone keyboard. Ctrl turns a
  // letter (or [ \ ] ^ _ @ ?) into its control byte; Alt prefixes ESC.
  // Anything longer than one character (a paste, an escape sequence
  // xterm produced itself) passes untouched and leaves the modifiers
  // armed — the person has not "used" them yet.
  function apply(data) {
    expire();
    if (!ctrl && !alt) return data;
    if (typeof data !== "string" || data.length !== 1) return data;
    let out = data;
    if (ctrl) {
      const c = controlByte(data);
      if (c === null) return data; // Ctrl+digit etc.: nothing to send, keep waiting
      out = c;
    }
    if (alt) out = "\x1b" + out;
    disarm();
    return out;
  }
  // applyKey: a sequence from the key bar. Arrows/Home/End take the
  // xterm modified form (\x1b[1;5A for Ctrl, ;3 for Alt, ;7 for both);
  // a single character goes through apply(); other sequences (Esc, Tab,
  // PgUp) are sent as they are and clear an armed Alt/Ctrl only when it
  // applied — they never consume a modifier they cannot use.
  function applyKey(seq) {
    expire();
    if (!ctrl && !alt) return seq;
    const final = ARROWS[seq];
    if (final) {
      const mod = 1 + (alt ? 2 : 0) + (ctrl ? 4 : 0);
      disarm();
      return "\x1b[1;" + mod + final;
    }
    if (seq.length === 1) return apply(seq);
    return seq;
  }
  function subscribe(fn) {
    listeners.add(fn);
    return () => listeners.delete(fn);
  }
  return { arm, disarm, apply, applyKey, state, subscribe };
}

// controlByte maps a key to its Ctrl form, or null when Ctrl+key has no
// terminal meaning.
export function controlByte(ch) {
  if (/^[a-zA-Z]$/.test(ch)) return String.fromCharCode(ch.toUpperCase().charCodeAt(0) & 0x1f);
  const map = { "@": "\x00", "[": "\x1b", "\\": "\x1c", "]": "\x1d", "^": "\x1e", "_": "\x1f", "/": "\x1f", "?": "\x7f", " ": "\x00" };
  return Object.prototype.hasOwnProperty.call(map, ch) ? map[ch] : null;
}
