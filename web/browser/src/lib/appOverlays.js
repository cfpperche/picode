// Which application overlays are on screen right now. A work browser's page
// is a native WebView2 sibling of the window: it paints over every HTML
// element, so an open dialog ends up behind the page (owner report
// 2026-09-15, "New workspace" cut in half by x.com). The work tab hides the
// native view while one is open, and the dialog's own overlay dims the whole
// window again — the modal reads as a modal.
//
// The DOM is the source of truth on purpose: every modal in the app renders
// its content as `.dlg` with data-state through ResponsiveDialog (ADR-0046),
// and a closed one unmounts. Watching the DOM covers call sites no list
// would; the work tab's own menu is not a `.dlg` and handles itself.

export const OPEN_DIALOG_SELECTOR = '.dlg[data-state="open"]';

const listeners = new Set();
let open = false;
let observer = null;

function scan() {
  const next =
    typeof document !== "undefined" &&
    document.querySelector(OPEN_DIALOG_SELECTOR) !== null;
  if (next === open) return;
  open = next;
  for (const fn of listeners) fn(open);
}

function ensureObserver() {
  if (observer || typeof MutationObserver === "undefined" || typeof document === "undefined") {
    return;
  }
  observer = new MutationObserver(scan);
  observer.observe(document.body, {
    subtree: true,
    childList: true,
    attributes: true,
    attributeFilter: ["data-state"],
  });
  scan();
}

// subscribeAppOverlays(fn) -> unsubscribe. fn is called immediately with the
// current state and again on every change.
export function subscribeAppOverlays(fn) {
  ensureObserver();
  listeners.add(fn);
  fn(open);
  return () => listeners.delete(fn);
}
