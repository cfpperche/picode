// Fit xterm to its pane whenever the pane's box changes — sidebar drag,
// tab switch, file pane, window resize. window.resize alone misses all of those.

export function sendTermResize(entry) {
  const term = entry && entry.term;
  const sock = entry && entry.sock;
  if (!term || !sock || sock.readyState !== 1) return false;
  if (term.cols < 2 || term.rows < 2) return false;
  sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
  return true;
}

export function scheduleTermFit(entry) {
  if (!entry || !entry.fit || !entry.paneEl) return;
  if (entry._fitFrame) cancelAnimationFrame(entry._fitFrame);
  entry._fitFrame = requestAnimationFrame(() => {
    entry._fitFrame = 0;
    const el = entry.paneEl;
    if (!el || !el.isConnected) return;
    if (el.clientWidth < 2 || el.clientHeight < 2) return;
    entry.fit.fit();
    sendTermResize(entry);
  });
}

export function wireTermFit(entry) {
  if (!entry || !entry.paneEl) return;
  if (typeof ResizeObserver === "undefined") {
    const onWin = () => scheduleTermFit(entry);
    window.addEventListener("resize", onWin);
    entry.unwireFit = () => window.removeEventListener("resize", onWin);
    return;
  }
  const ro = new ResizeObserver(() => scheduleTermFit(entry));
  ro.observe(entry.paneEl);
  entry.unwireFit = () => {
    if (entry._fitFrame) cancelAnimationFrame(entry._fitFrame);
    ro.disconnect();
  };
}
