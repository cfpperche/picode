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
