// Terminal socket lifecycle: one WebSocket per xterm attach, with
// automatic reattach. Locking a phone suspends the page and the browser
// drops the /ws/term WebSocket; the tmux session survives server-side
// (only the attach dies), so the client can simply attach again — same
// xterm, scrollback and scroll position intact — instead of leaving a
// dead "— detached —" pane that only an exit/re-enter revives.
//
// Contract with the entry object (see mobile/desktop lib/terms.js):
//   entry.sock          the current WebSocket (replaced on every connect)
//   entry.closedByUser  set by closeTerm()/dropTermSocket(): stops
//                       reattach and unwires everything. One-way.
// Control state lives on entry.__sockCtl; ctl.suspended is the reversible
// stop (suspendTermSocket/kickTermSocket): the socket is closed and no
// retry runs, but the xterm, its scrollback and the control block stay so
// the same instance resumes later — a hidden matrix panel, not a closed
// terminal (ADR-0109, docs/plans/matrix-app.md §4.5).
// Handlers:
//   onOpen()    a connection opened (first or reattach): fit + resize
//               so the fresh `tmux attach` gets the real size.
//   onMessage(ev)  every WebSocket message.
//   onState(false) an established connection was lost (once per drop).
//   onGiveUp()  RECONNECT_BURST consecutive attempts failed — the tmux
//               session is probably gone; say so, wait for a kick.

export const RECONNECT_BASE_MS = 1000;
export const RECONNECT_MAX_MS = 10000;
export const RECONNECT_BURST = 12;

const OPEN = 1;
const CONNECTING = 0;

function defaultDeps() {
  return {
    WebSocketImpl: typeof WebSocket !== "undefined" ? WebSocket : null,
    setTimer: (fn, ms) => setTimeout(fn, ms),
    clearTimer: (t) => clearTimeout(t),
    doc: typeof document !== "undefined" ? document : null,
    win: typeof window !== "undefined" ? window : null,
  };
}

export function connectTermSocket(entry, url, handlers = {}, deps = defaultDeps()) {
  if (!deps.WebSocketImpl) return null;
  const ctl = {
    url,
    handlers,
    deps,
    attempts: 0, // consecutive connects that did not open
    timer: 0, // pending reconnect timer
    live: false, // true between onopen and the next close
    gaveUp: false, // burst exhausted; a kick restarts it
    suspended: false, // parked on purpose; a kick lifts it
    unwireWindow: null,
  };
  entry.__sockCtl = ctl;

  const dead = () => !entry.sock || (entry.sock.readyState !== CONNECTING && entry.sock.readyState !== OPEN);

  function open() {
    if (ctl.timer) {
      ctl.deps.clearTimer(ctl.timer);
      ctl.timer = 0;
    }
    if (entry.closedByUser || ctl.suspended || !dead()) return;
    let ws;
    try {
      ws = new deps.WebSocketImpl(ctl.url);
    } catch {
      schedule();
      return;
    }
    ws.binaryType = "arraybuffer";
    entry.sock = ws;
    ws.onopen = () => {
      ctl.attempts = 0;
      ctl.live = true;
      if (ctl.handlers.onOpen) ctl.handlers.onOpen();
    };
    ws.onmessage = (ev) => {
      if (ctl.handlers.onMessage) ctl.handlers.onMessage(ev);
    };
    ws.onclose = () => {
      const wasLive = ctl.live;
      ctl.live = false;
      // A suspended close is silent: no "— detached —" line, no retry.
      if (entry.closedByUser || ctl.suspended) return;
      if (wasLive && ctl.handlers.onState) ctl.handlers.onState(false);
      schedule();
    };
  }

  function schedule() {
    if (ctl.timer || ctl.gaveUp || entry.closedByUser || ctl.suspended) return;
    ctl.attempts += 1;
    if (ctl.attempts > RECONNECT_BURST) {
      ctl.gaveUp = true;
      if (ctl.handlers.onGiveUp) ctl.handlers.onGiveUp();
      return;
    }
    const delay = Math.min(RECONNECT_BASE_MS * 2 ** (ctl.attempts - 1), RECONNECT_MAX_MS);
    ctl.timer = ctl.deps.setTimer(() => {
      ctl.timer = 0;
      open();
    }, delay);
  }

  // Conditions changed — the page became visible, the network returned,
  // the pane was remounted, or a suspended panel came back into view.
  // Retry immediately; a burst that gave up starts over, so a recreated
  // tmux session still heals.
  function kick() {
    if (entry.closedByUser) return;
    ctl.suspended = false;
    if (ctl.gaveUp) {
      ctl.gaveUp = false;
      ctl.attempts = 0;
    }
    open();
  }

  function wireWindow() {
    if (!ctl.deps.doc || !ctl.deps.win || ctl.unwireWindow) return;
    const onVis = () => {
      if (ctl.deps.doc.visibilityState === "visible") kick();
    };
    const onOnline = () => kick();
    ctl.deps.doc.addEventListener("visibilitychange", onVis);
    ctl.deps.win.addEventListener("online", onOnline);
    ctl.unwireWindow = () => {
      ctl.deps.doc.removeEventListener("visibilitychange", onVis);
      ctl.deps.win.removeEventListener("online", onOnline);
      ctl.unwireWindow = null;
    };
  }
  wireWindow();
  open();
  ctl.kick = kick;
  return ctl;
}

// User-driven close (closeTerm): stop reattaching, unwire everything.
export function dropTermSocket(entry) {
  const ctl = entry && entry.__sockCtl;
  if (!ctl) return;
  entry.closedByUser = true;
  if (ctl.unwireWindow) ctl.unwireWindow();
  if (ctl.timer) {
    ctl.deps.clearTimer(ctl.timer);
    ctl.timer = 0;
  }
  const st = entry.sock ? entry.sock.readyState : 3;
  if (st === CONNECTING || st === OPEN) {
    try {
      entry.sock.close();
    } catch {
      /* ignore */
    }
  }
  delete entry.__sockCtl;
}

// Reattach now if the socket is dead (visibility, online, remount), and
// lift a suspension.
export function kickTermSocket(entry) {
  const ctl = entry && entry.__sockCtl;
  if (ctl && ctl.kick) ctl.kick();
}

// Reversible stop: close the socket and stop retrying, keep the xterm and
// the control block. The window listeners stay wired — a visibility or
// online event kicks, which is exactly what a panel scrolled back into
// view wants — but only kickTermSocket lifts the suspension itself.
export function suspendTermSocket(entry) {
  const ctl = entry && entry.__sockCtl;
  if (!ctl) return;
  ctl.suspended = true;
  if (ctl.timer) {
    ctl.deps.clearTimer(ctl.timer);
    ctl.timer = 0;
  }
  const st = entry.sock ? entry.sock.readyState : 3;
  if (st === CONNECTING || st === OPEN) {
    try {
      entry.sock.close();
    } catch {
      /* ignore */
    }
  }
}

export function isTermSocketSuspended(entry) {
  const ctl = entry && entry.__sockCtl;
  return !!(ctl && ctl.suspended);
}
