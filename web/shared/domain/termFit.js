// Fit xterm to its pane whenever the pane's box changes — sidebar drag,
// tab switch, file pane, window resize. window.resize alone misses all of those.
// Dragging the sidebar fires many observations; debounce so tmux gets one SIGWINCH.

export const FIT_DEBOUNCE_MS = 150;

export function sendTermResize(entry) {
  const term = entry && entry.term;
  const sock = entry && entry.sock;
  if (!term || !sock || sock.readyState !== 1) return false;
  if (term.cols < 2 || term.rows < 2) return false;
  sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
  return true;
}

// After FitAddon floors the column count, unused pixels sit on the right
// of .xterm-screen (it is left-aligned). Split them so --term-pad reads
// the same on both sides. Helpers live inside the screen, so IME and
// mouse mapping stay lined up with the grid.
export function termSlackPad(paneWidth, screenWidth) {
  const pane = Math.round(Number(paneWidth) || 0);
  const screen = Math.round(Number(screenWidth) || 0);
  if (pane < 2 || screen < 2 || screen > pane) return { slack: 0, left: 0, right: 0 };
  const slack = pane - screen;
  const left = Math.floor(slack / 2);
  return { slack, left, right: slack - left };
}

export function applyTermSlack(term) {
  const el = term && term.element;
  const screen = el && el.querySelector(".xterm-screen");
  if (!el || !screen) return { slack: 0, left: 0, right: 0 };
  const pad = termSlackPad(el.clientWidth, screen.offsetWidth);
  screen.style.marginLeft = pad.left + "px";
  screen.style.marginRight = pad.right + "px";
  return pad;
}

function runFit(entry) {
  const el = entry.paneEl;
  if (!el || !el.isConnected) return;
  if (el.clientWidth < 2 || el.clientHeight < 2) return;
  if (entry.fit) entry.fit.fit();
  applyTermSlack(entry.term);
  sendTermResize(entry);
}

export function scheduleTermFit(entry, immediate) {
  if (!entry || !entry.fit || !entry.paneEl) return;
  if (entry._fitTimer) {
    clearTimeout(entry._fitTimer);
    entry._fitTimer = 0;
  }
  if (entry._fitFrame) {
    cancelAnimationFrame(entry._fitFrame);
    entry._fitFrame = 0;
  }
  const kick = () => {
    entry._fitTimer = 0;
    entry._fitFrame = requestAnimationFrame(() => {
      entry._fitFrame = 0;
      runFit(entry);
    });
  };
  if (immediate) kick();
  else entry._fitTimer = setTimeout(kick, FIT_DEBOUNCE_MS);
}

export function wireTermFit(entry) {
  if (!entry || !entry.paneEl) return;
  const stop = () => {
    if (entry._fitTimer) clearTimeout(entry._fitTimer);
    if (entry._fitFrame) cancelAnimationFrame(entry._fitFrame);
    entry._fitTimer = 0;
    entry._fitFrame = 0;
  };
  if (typeof ResizeObserver === "undefined") {
    const onWin = () => scheduleTermFit(entry);
    window.addEventListener("resize", onWin);
    entry.unwireFit = () => { stop(); window.removeEventListener("resize", onWin); };
    return;
  }
  const ro = new ResizeObserver(() => scheduleTermFit(entry));
  ro.observe(entry.paneEl);
  entry.unwireFit = () => { stop(); ro.disconnect(); };
}
