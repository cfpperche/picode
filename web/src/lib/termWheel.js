// Wheel on a PiCode terminal must move the view. xterm.js otherwise
// forwards it as Up/Down or SGR mouse, which Pi's composer eats.
// Order: scroll xterm's own buffer; if that did not move (TUI / alt-screen
// with no scrollback), PageUp/PageDown to the process. Shift+wheel is
// left to xterm.

const PAGE_UP = new TextEncoder().encode("\x1b[5~");
const PAGE_DOWN = new TextEncoder().encode("\x1b[6~");
const acc = new WeakMap();
const THRESHOLD = 40;

export function wheelLineCount(ev) {
  const dy = Number(ev && ev.deltaY) || 0;
  if (dy === 0) return 0;
  if (ev.deltaMode === 1) return Math.trunc(dy) || (dy < 0 ? -1 : 1);
  const n = Math.min(12, Math.max(1, Math.round(Math.abs(dy) / 40)));
  return dy < 0 ? -n : n;
}

export function pageBytesFor(term, dy) {
  if (!term || !dy) return null;
  const next = (acc.get(term) || 0) + dy;
  if (Math.abs(next) < THRESHOLD) {
    acc.set(term, next);
    return new Uint8Array(0);
  }
  acc.set(term, 0);
  return next < 0 ? PAGE_UP : PAGE_DOWN;
}

export function applyTermWheel(term, ev, send) {
  if (!term || !ev || ev.shiftKey) return "skip";
  const dy = Number(ev.deltaY) || 0;
  if (dy === 0) return "skip";
  const before = term.buffer && term.buffer.active ? term.buffer.active.viewportY : 0;
  if (typeof term.scrollLines === "function") term.scrollLines(wheelLineCount(ev));
  const after = term.buffer && term.buffer.active ? term.buffer.active.viewportY : before;
  if (after !== before) return "xterm";
  const bytes = pageBytesFor(term, dy);
  if (bytes && bytes.length && send) send(bytes);
  return bytes && bytes.length ? "page" : "hold";
}

export function wireTermWheel(term, send, el) {
  if (!term) return;
  const target = el || term.element;
  if (!target || typeof target.addEventListener !== "function") return;
  const onWheel = (ev) => {
    const action = applyTermWheel(term, ev, send);
    if (action === "skip") return;
    if (typeof ev.preventDefault === "function") ev.preventDefault();
    if (typeof ev.stopPropagation === "function") ev.stopPropagation();
  };
  target.addEventListener("wheel", onWheel, { capture: true, passive: false });
  if (typeof term.attachCustomWheelEventHandler === "function") {
    term.attachCustomWheelEventHandler((ev) => !!(ev && ev.shiftKey));
  }
}
