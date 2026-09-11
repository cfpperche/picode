// Composer floats over ~200px of the conversation. "At the bottom"
// means inside that pad, not a 48px sliver under the overlay.
export const PAD = 320;
// A Matrix chat panel has no composer over it, and is a few hundred pixels
// tall: 320 px there would call every position "the bottom" and yank a
// reader who scrolled up back down on the next delta.
export const PANEL_PAD = 64;

export function stuckToBottom(el, pad = PAD) {
  if (!el) return true;
  return el.scrollHeight - el.scrollTop - el.clientHeight < pad;
}

export function pinToBottom(el) {
  if (el) el.scrollTop = el.scrollHeight;
}
