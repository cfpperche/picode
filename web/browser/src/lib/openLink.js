import { isLoopbackUrl } from "@picode/shared/client/devservers.js";

// Where a link printed in a terminal opens. The decision table (one row per
// combination, and `openLink.test.js` walks it):
//
//   kind   host        browser.localOpenDest   action
//   file   —           —                       file    (the file pane, unchanged)
//   http   elsewhere   —                       external (the system browser, unchanged)
//   http   loopback    "app" (default)         app     (PiCode's own browser surface)
//   http   loopback    "external"              external
//   http   loopback    unreadable (fetch down) app
//
// "Local development sites — where localhost and dev servers open by default"
// is a user preference (Browser settings, slice 3) and it means exactly this
// case: a dev server URL printed in a terminal. A viewer who set it to
// external keeps the system browser; the default gets PiCode's own surface.
// Not covered here (deliberately): the Servers panel's own **Open** button —
// an explicit control inside PiCode asks for PiCode, and the preference
// governs the default, not a click.
export function linkOpenTarget(link, localOpenDest) {
  if (!link) return null;
  if (link.kind !== "http") return { action: "file", path: link.path };
  if (!isLoopbackUrl(link.href)) return { action: "external", url: link.href };
  return { action: localOpenDest === "external" ? "external" : "app", url: link.href };
}
