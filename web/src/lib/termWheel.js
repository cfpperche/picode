// Wheel on a PiCode terminal must move the view. xterm.js otherwise
// forwards it as Up/Down, which Pi's composer eats. Order: scroll xterm's
// own buffer; if that did not move (TUI / no scrollback), send SGR wheel
// at the top of the screen so Pi's transcript ScrollView moves — not PageUp
// (tmux often swallows it) and not Up/Down (composer). Shift+wheel is xterm.

const acc = new WeakMap();

export function wheelLineCount(ev) {
  const dy = Number(ev && ev.deltaY) || 0;
  if (dy === 0) return 0;
  if (ev.deltaMode === 1) return Math.trunc(dy) || (dy < 0 ? -1 : 1);
  const n = Math.min(12, Math.max(1, Math.round(Math.abs(dy) / 40)));
  return dy < 0 ? -n : n;
}

export function sgrWheelBytes(dy) {
  const n = Number(dy) || 0;
  if (!n) return new Uint8Array(0);
  const btn = n < 0 ? 64 : 65;
  const reps = Math.min(6, Math.max(1, Math.round(Math.abs(n) / 40)));
  const one = "\x1b[<" + btn + ";2;2M";
  return new TextEncoder().encode(one.repeat(reps));
}

export function applyTermWheel(term, ev, send) {
  if (!term || !ev || ev.shiftKey) return "skip";
  const dy = Number(ev.deltaY) || 0;
  if (dy === 0) return "skip";
  const before = term.buffer && term.buffer.active ? term.buffer.active.viewportY : 0;
  if (typeof term.scrollLines === "function") term.scrollLines(wheelLineCount(ev));
  const after = term.buffer && term.buffer.active ? term.buffer.active.viewportY : before;
  if (after !== before) return "xterm";
  const next = (acc.get(term) || 0) + dy;
  if (Math.abs(next) < 24) {
    acc.set(term, next);
    return "hold";
  }
  acc.set(term, 0);
  const bytes = sgrWheelBytes(next);
  if (bytes.length && send) send(bytes);
  return bytes.length ? "sgr" : "hold";
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
