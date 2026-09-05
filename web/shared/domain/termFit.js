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

function runFit(entry) {
  const el = entry.paneEl;
  if (!el || !el.isConnected) return;
  if (el.clientWidth < 2 || el.clientHeight < 2) return;
  if (entry.fit) entry.fit.fit();
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
