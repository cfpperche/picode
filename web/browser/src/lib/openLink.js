import { isLoopbackUrl } from "@picode/shared/client/devservers.js";

// Where a link printed in a terminal opens. The decision table (one row per
// combination, and `openLink.test.js` walks it):
//
//   kind   host        preference          action
//   file   —           —                   file    (the file pane, unchanged)
//   http   loopback    localOpenDest       app unless "external"
//   http   elsewhere   webOpenDest         app unless "external"
//
// A URL printed in a terminal is a browsing action inside PiCode, and where it
// lands is the human's preference — the same two rows Browser settings already
// offers ("Local development sites" and "Web URL and link open destination",
// both defaulting to PiCode's own surface). Before 2026-09-17 only the loopback
// case reached PiCode and every other host went straight to the system browser,
// which made Ctrl+click on a terminal link the one door that ignored where the
// human had said web URLs open (owner's call, 2026-09-17).
//
// The preference is read fresh per link, like the shell's popup door, so it is
// whatever it is now. Not covered here (deliberately): the Servers panel's own
// **Open** button — an explicit control inside PiCode asks for PiCode, and the
// preference governs the default, not a click.
export function linkOpenTarget(link, localOpenDest, webOpenDest) {
  if (!link) return null;
  if (link.kind !== "http") return { action: "file", path: link.path };
  const pref = isLoopbackUrl(link.href) ? localOpenDest : webOpenDest;
  return { action: pref === "external" ? "external" : "app", url: link.href };
}
