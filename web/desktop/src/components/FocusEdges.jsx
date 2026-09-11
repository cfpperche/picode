import { IconCollapse } from "./Icons.jsx";
import { formatChord, primaryChord } from "../lib/appKeys.js";

// The three edge strips of fullscreen (focus) mode. They are real
// elements, not a bare pointermove listener, because an embedded frame
// swallows the pointer — today that is the PDF preview inside a file tab,
// which reaches the window edge once the chrome is gone. Only something
// painted above it still sees the pointer there. The right strip stays
// pointer-transparent so a surface's scrollbar keeps its grab area; the
// window listener catches the pointer over it instead.
const ZONE_HINT = {
  left: "Sidebar — rest the pointer here",
  top: "Tabs — rest the pointer here",
  right: "Inspector — rest the pointer here",
};

export default function FocusEdges({ zones }) {
  if (!zones || !zones.length) return null;
  return (
    <>
      {zones.map((zone) => (
        <div
          key={zone}
          className={`focus-edge focus-edge-${zone}`}
          data-zone={zone}
          aria-hidden="true"
          title={ZONE_HINT[zone]}
        />
      ))}
    </>
  );
}

// The way out that does not need a keyboard: it rides at the right end of
// the tab strip, so the top reveal always carries it — even with no tabs
// open, and after a reload, when the browser is not in fullscreen at all.
// The glyph is the whole control: the label and the chord are the hint
// (title for the pointer, aria-label for the name), so the strip carries an
// exit without carrying a sentence. 16px like the icon buttons beside it: at
// 14px the glyph read a size smaller than its neighbours in the same row.
export function FocusLeave({ onLeave }) {
  const chord = formatChord(primaryChord("app.fullscreen.toggle"));
  return (
    <button
      type="button"
      className="focus-leave"
      onClick={onLeave}
      aria-label="Leave fullscreen"
      title={chord ? `Leave fullscreen (${chord})` : "Leave fullscreen"}
    >
      <IconCollapse size={16} />
    </button>
  );
}
