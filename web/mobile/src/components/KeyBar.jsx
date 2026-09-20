// Phone terminal accessory (ADR-0044): one row that scrolls sideways,
// Fable-first, sitting on top of the software keyboard. Ctrl/Alt are
// sticky. A key never summons the IME — pointerdown preventDefault keeps
// focus where it is. Hide (pinned, does not scroll away) blurs xterm.
//
// iOS (owner report 2026-09-20): with the action on click, tapping a key
// blurred the terminal — the IME closed and the click itself was
// suppressed, so the TUI never moved. The key now acts on pointerdown
// (default prevented, so focus stays) with a short guard so the fallback
// click of a mouse cannot double-send.

import { useRef } from "react";
import { IconKeyboard } from "./Icons.jsx";
import { KEYS } from "../lib/termKeyBar.js";

export { KEYS };

export default function KeyBar({ armed, onArm, onKey, onHide }) {
  const lastAct = useRef(0);
  const once = (fn, arg) => (e) => {
    e.preventDefault();
    const now = Date.now();
    if (now - lastAct.current < 350) return;
    lastAct.current = now;
    fn(arg);
  };
  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys">
      <div className="m-keybar-scroller">
        {KEYS.map((k) => k.mod ? (
          <button
            key={k.id}
            type="button"
            className={"m-key m-key-mod" + (armed && armed[k.mod] ? " on" : "")}
            aria-pressed={!!(armed && armed[k.mod])}
            title={k.label + " — arms the next key"}
            onPointerDown={once(onArm, k.mod)}
            onClick={once(onArm, k.mod)}
          >{k.label}</button>
        ) : (
          <button
            key={k.id}
            type="button"
            className="m-key"
            title={k.title || k.label}
            onPointerDown={once(onKey, k.seq)}
            onClick={once(onKey, k.seq)}
          >{k.label}</button>
        ))}
      </div>
      <button
        type="button"
        className="m-key m-key-hide"
        title="Hide keyboard"
        aria-label="Hide keyboard"
        onPointerDown={(e) => e.preventDefault()}
        onClick={onHide}
      ><IconKeyboard size={16} /></button>
    </div>
  );
}
