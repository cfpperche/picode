// Composer floats over ~200px of the conversation. "At the bottom"
// means inside that pad, not a 48px sliver under the overlay.
const PAD = 320;

export function stuckToBottom(el) {
  if (!el) return true;
  return el.scrollHeight - el.scrollTop - el.clientHeight < PAD;
}

export function pinToBottom(el) {
  if (el) el.scrollTop = el.scrollHeight;
}
