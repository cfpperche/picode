// Alt-screen apps (Pi TUI, vim, less) have no xterm scrollback. xterm.js
// turns the wheel into Up/Down or SGR mouse, both of which hit Pi's composer.
// PageUp/PageDown scroll the TUI viewport (and page in vim/less). Shift+wheel
// stays with xterm.

const PAGE_UP = new TextEncoder().encode("\x1b[5~");
const PAGE_DOWN = new TextEncoder().encode("\x1b[6~");
const NONE = new Uint8Array(0);
const acc = new WeakMap();

const THRESHOLD = 60;

export function altScreenWheelBytes(term, ev) {
  if (!term || !ev || ev.shiftKey) return null;
  if (!term.buffer || !term.buffer.active || term.buffer.active.type !== "alternate") return null;
  const dy = Number(ev.deltaY) || 0;
  if (dy === 0) return NONE;
  const next = (acc.get(term) || 0) + dy;
  if (Math.abs(next) < THRESHOLD) {
    acc.set(term, next);
    return NONE;
  }
  acc.set(term, 0);
  return next < 0 ? PAGE_UP : PAGE_DOWN;
}

export function wireTermWheel(term, send) {
  if (!term || typeof term.attachCustomWheelEventHandler !== "function") return;
  term.attachCustomWheelEventHandler((ev) => {
    const bytes = altScreenWheelBytes(term, ev);
    if (bytes === null) return true;
    if (ev && typeof ev.preventDefault === "function") ev.preventDefault();
    if (bytes.length && send) send(bytes);
    return false;
  });
}
