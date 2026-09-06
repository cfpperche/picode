// Phone terminal accessory (ADR-0044): one row that scrolls sideways,
// Fable-first, sitting on top of the software keyboard. Ctrl/Alt are
// sticky. A key never summons the IME — pointerdown preventDefault keeps
// focus where it is. Hide (pinned, does not scroll away) blurs xterm.

import { IconKeyboard } from "./Icons.jsx";
import { KEYS } from "../lib/termKeyBar.js";

export { KEYS };

export default function KeyBar({ armed, onArm, onKey, onHide }) {
  const still = (e) => e.preventDefault();
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
            onPointerDown={still}
            onClick={() => onArm(k.mod)}
          >{k.label}</button>
        ) : (
          <button
            key={k.id}
            type="button"
            className="m-key"
            title={k.title || k.label}
            onPointerDown={still}
            onClick={() => onKey(k.seq)}
          >{k.label}</button>
        ))}
      </div>
      <button
        type="button"
        className="m-key m-key-hide"
        title="Hide keyboard"
        aria-label="Hide keyboard"
        onPointerDown={still}
        onClick={onHide}
      ><IconKeyboard size={16} /></button>
    </div>
  );
}
