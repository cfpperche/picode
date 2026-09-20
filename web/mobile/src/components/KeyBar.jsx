// Phone terminal accessory (ADR-0044): one row that scrolls sideways,
// Fable-first, sitting on top of the software keyboard. Ctrl/Alt are
// sticky. A key never summons the IME. Hide (pinned, does not scroll
// away) blurs xterm.
//
// The keyboard belongs to the user (owner, iOS, 2026-09-20): a key-bar
// tap must move the TUI and leave the IME open; only Hide or a real tap
// in the pane closes it. Canceling pointerdown is not enough on iOS —
// WebKit still blurs the terminal. What it honors is a NON-PASSIVE
// touchstart with its default prevented (the CodeMirror/Monaco-toolbar
// trick), so touch owns the action there. Mouse acts on pointerdown and
// Enter/Space on click; each input fires exactly one path, so rapid
// repeated arrows are never swallowed by a dedupe window. The tradeoff:
// a drag starting on a key can no longer scroll the bar (the gaps and
// the hide button still can; the Fable-first row is designed to fit).

import { useEffect, useRef } from "react";
import { IconKeyboard } from "./Icons.jsx";
import { KEYS } from "../lib/termKeyBar.js";

export { KEYS };

export default function KeyBar({ armed, onArm, onKey, onHide }) {
  const barRef = useRef(null);
  // Fresh every render, so the effect (rebound each render too) and the
  // React handlers always call with the current props.
  const dispatch = (act, arg) => {
    if (act === "key") onKey(arg);
    else if (act === "mod") onArm(arg);
    else onHide();
  };

  useEffect(() => {
    const bar = barRef.current;
    if (!bar) return undefined;
    const onTouchStart = (e) => {
      const btn = e.target.closest("button[data-act]");
      if (!btn) return;
      e.preventDefault(); // no blur, no IME dance, no synthetic click
      dispatch(btn.dataset.act, btn.dataset.arg);
    };
    bar.addEventListener("touchstart", onTouchStart, { passive: false });
    return () => bar.removeEventListener("touchstart", onTouchStart);
  });

  const props = (act, arg) => ({
    "data-act": act,
    "data-arg": arg,
    onPointerDown: (e) => {
      if (e.pointerType === "touch") return; // touchstart owns it
      e.preventDefault();
      dispatch(act, arg);
    },
    onClick: () => dispatch(act, arg),
  });

  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys" ref={barRef}>
      <div className="m-keybar-scroller">
        {KEYS.map((k) => k.mod ? (
          <button
            key={k.id}
            type="button"
            className={"m-key m-key-mod" + (armed && armed[k.mod] ? " on" : "")}
            aria-pressed={!!(armed && armed[k.mod])}
            title={k.label + " — arms the next key"}
            {...props("mod", k.mod)}
          >{k.label}</button>
        ) : (
          <button
            key={k.id}
            type="button"
            className="m-key"
            title={k.title || k.label}
            {...props("key", k.seq)}
          >{k.label}</button>
        ))}
      </div>
      <button
        type="button"
        className="m-key m-key-hide"
        title="Hide keyboard"
        aria-label="Hide keyboard"
        {...props("hide")}
      ><IconKeyboard size={16} /></button>
    </div>
  );
}
