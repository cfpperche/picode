// One gesture, two decision points: the terminal's Ctrl+V keydown pastes
// clipboard *text* (termKeys.js, async readText), while the paste *event*
// carries files (TermSurface onPasteCapture, sync). When the event claims a
// files-paste for the attach bar, the keydown's late text must not land in
// the composer's draft as well — so the claim suppresses exactly that path.
// Element-scoped (the focused textarea): a paste claimed in one pane never
// eats a paste in another. Time-boxed: suppressing only gates the redundant
// keydown path — text still lands through the native paste — so even a
// generous window cannot lose a paste. `now` is injectable for tests.
const WINDOW_MS = 1500;

let claim = null; // { el, at } | null

export function suppressKeyPasteFor(el, now = Date.now()) {
  claim = el ? { el, at: now } : null;
}

export function keyPasteSuppressed(el, now = Date.now()) {
  return !!claim && !!el && claim.el === el && now - claim.at < WINDOW_MS;
}

export function resetPasteClaim() {
  claim = null;
}
