// Shell frame mode (ADR-0123): when the desktop shell loads this UI, it
// sets `window.__PICODE_SHELL__` and the UI claims the window frame — it
// draws its own window controls and reserves the top-right slot for them.
// The shell's injected fallback frame watches `data-picode-frame` on the
// root element and retires itself when the UI has claimed the frame, so
// the two never draw twice and neither side depends on the other's deploy
// cadence. In a browser none of this activates.

export function isShell() {
  return typeof window !== "undefined" && window.__PICODE_SHELL__ === true;
}

export function claimFrame() {
  if (!isShell()) return;
  document.documentElement.dataset.picodeFrame = "1";
}

export function releaseFrame() {
  if (typeof document === "undefined") return;
  delete document.documentElement.dataset.picodeFrame;
}
