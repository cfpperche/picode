// Imperative xterm instances — not React state. One attach per agent.
export const terms = new Map();

// Keep the xterm node when React unmounts the tab host (switching tabs).
export function parkTerm(el) {
  if (!el || !el.isConnected || typeof document === "undefined") return;
  let box = document.getElementById("term-park");
  if (!box) {
    box = document.createElement("div");
    box.id = "term-park";
    box.hidden = true;
    box.setAttribute("aria-hidden", "true");
    document.body.appendChild(box);
  }
  if (el.parentElement !== box) box.appendChild(el);
}

// Dispose a live attach: close the socket, drop the xterm instance.
export function closeTerm(id) {
  const t = terms.get(id);
  if (!t) return;
  t.closedByUser = true;
  if (t.unwireLinks) try { t.unwireLinks(); } catch { /* ignore */ }
  try { t.sock.close(); } catch { /* ignore */ }
  t.term.dispose();
  t.paneEl.remove();
  terms.delete(id);
}
