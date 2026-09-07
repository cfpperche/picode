import { useEffect, useRef, useState } from "react";
import {
  extraKeysVisible,
  hardKeyboardLikely,
  keyboardInset,
  sendTermSeq,
} from "@picode/shared/domain/keyboardInset.js";

// Extra keys travel with the software keyboard: visible while the
// terminal host holds focus, gone on blur. A bar tap never steals
// focus (the caller preventDefault's pointerdown).

export function useTermAccessory(hostRef, entryOf, attachKey) {
  const entryRef = useRef(entryOf);
  entryRef.current = entryOf;
  const [armed, setArmed] = useState({ ctrl: false, alt: false });
  const [focused, setFocused] = useState(false);
  const [hardKeyboard, setHardKeyboard] = useState(false);

  useEffect(() => {
    let off = null;
    const tick = setInterval(() => {
      const entry = entryRef.current && entryRef.current();
      if (!entry || !entry.sticky) return;
      const st = entry.sticky.state();
      setArmed((cur) => (cur.ctrl === st.ctrl && cur.alt === st.alt ? cur : st));
    }, 250);
    const entry = entryRef.current && entryRef.current();
    if (entry && entry.sticky) off = entry.sticky.subscribe((st) => setArmed(st));
    return () => { clearInterval(tick); if (off) off(); };
  }, [attachKey]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return undefined;
    let touchedAt = 0;
    let focusedAt = 0;
    const inHost = (el) => !!(el && host.contains(el));
    const onPointer = () => { touchedAt = Date.now(); };
    const onFocusIn = (ev) => {
      if (!inHost(ev.target)) return;
      setFocused(true);
      if (Date.now() - touchedAt > 500) return;
      focusedAt = Date.now();
      const vv = window.visualViewport;
      setTimeout(() => {
        if (Date.now() - focusedAt < 280) return;
        const inset = keyboardInset({
          innerHeight: window.innerHeight,
          vvHeight: vv ? vv.height : window.innerHeight,
          vvOffsetTop: vv ? vv.offsetTop : 0,
        });
        const finePointer = window.matchMedia("(hover: hover) and (pointer: fine)").matches;
        if (hardKeyboardLikely({ termFocused: true, inset, elapsedMs: 300, finePointer })) {
          setHardKeyboard(true);
        }
      }, 300);
    };
    const onFocusOut = (ev) => {
      if (!inHost(ev.target)) return;
      const next = ev.relatedTarget;
      if (inHost(next)) return;
      setFocused(false);
    };
    document.addEventListener("pointerdown", onPointer, true);
    document.addEventListener("focusin", onFocusIn);
    document.addEventListener("focusout", onFocusOut);
    if (inHost(document.activeElement)) setFocused(true);
    return () => {
      document.removeEventListener("pointerdown", onPointer, true);
      document.removeEventListener("focusin", onFocusIn);
      document.removeEventListener("focusout", onFocusOut);
    };
  }, [hostRef, attachKey]);

  function sendKey(seq) {
    const entry = entryRef.current && entryRef.current();
    const host = hostRef.current;
    const hadFocus = !!(host && document.activeElement && host.contains(document.activeElement));
    const out = sendTermSeq(entry, seq, hadFocus);
    if (out.refocus && entry && entry.term) entry.term.focus();
  }

  function armKey(mod) {
    const entry = entryRef.current && entryRef.current();
    if (entry && entry.sticky) setArmed(entry.sticky.arm(mod));
  }

  function show() {
    setHardKeyboard(false);
    const entry = entryRef.current && entryRef.current();
    if (entry && entry.term) entry.term.focus();
  }

  function hide() {
    const entry = entryRef.current && entryRef.current();
    if (entry && entry.term && typeof entry.term.blur === "function") entry.term.blur();
    const host = hostRef.current;
    const ta = host && host.querySelector("textarea");
    if (ta) ta.blur();
    setFocused(false);
  }

  return {
    armed,
    visible: extraKeysVisible({ termFocused: focused, hardKeyboard }),
    sendKey,
    armKey,
    show,
    hide,
  };
}
