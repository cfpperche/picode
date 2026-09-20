// xterm.js #3184/#802: alt-screen has no scrollback, so the viewport
// scrollbar has nothing to move. #1310 then turns the wheel into Up/Down
// (Pi's composer). #426: when mouse tracking is on, xterm sends SGR to the
// app — that is what should scroll a TUI. We only inject SGR if tracking is
// off. Do not capture/preventDefault on the pane (that blocked #426).

const acc = new WeakMap();

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
  const mouse = term.modes && term.modes.mouseTrackingMode;
  if (mouse && mouse !== "none") return "skip";
  const buf = term.buffer && term.buffer.active;
  const rows = term.rows || 0;
  const canXterm = buf && buf.type !== "alternate" && ((buf.baseY || 0) > 0 || (buf.length || 0) > rows);
  if (canXterm) return "skip";
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

export function wireTermWheel(term, send) {
  if (!term || typeof term.attachCustomWheelEventHandler !== "function") return;
  term.attachCustomWheelEventHandler((ev) => {
    const action = applyTermWheel(term, ev, send);
    if (action === "skip") return true;
    if (ev && typeof ev.preventDefault === "function") ev.preventDefault();
    return false;
  });
}

// The Agent CLIs touch gesture belongs to the terminal pane, not its route.
// Keep native xterm mouse reporting/scrollback arbitration in wireTermWheel.
export function wireTermTouch(host) {
  let lastY = null;
  let acc = 0;
  const reset = () => { lastY = null; acc = 0; };
  const start = e => { lastY = e.touches.length === 1 ? e.touches[0].clientY : null; acc = 0; };
  const move = e => {
    if (e.touches.length !== 1) { reset(); return; }
    if (lastY === null) return;
    const y = e.touches[0].clientY;
    acc += lastY - y;
    lastY = y;
    const target = host.querySelector(".xterm-screen") || host.querySelector(".xterm");
    if (!target) return;
    while (Math.abs(acc) >= 16) {
      const direction = acc > 0 ? 1 : -1;
      acc -= direction * 16;
      target.dispatchEvent(new WheelEvent("wheel", {
        deltaY: direction * 40, deltaMode: 0,
        clientX: e.touches[0].clientX, clientY: y,
        bubbles: true, cancelable: true,
      }));
    }
    e.preventDefault();
  };
  host.addEventListener("touchstart", start, { passive: true });
  host.addEventListener("touchmove", move, { passive: false });
  host.addEventListener("touchend", reset);
  host.addEventListener("touchcancel", reset);
  return () => {
    host.removeEventListener("touchstart", start);
    host.removeEventListener("touchmove", move);
    host.removeEventListener("touchend", reset);
    host.removeEventListener("touchcancel", reset);
  };
}
