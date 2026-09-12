import { useCallback, useEffect, useRef, useState } from "react";
import { CANVAS_LIMITS } from "@picode/shared/domain/canvas.js";

// TextPanel — words written on the plane. The first panel kind that holds
// its own content instead of naming something else in PiCode (migration
// 046): a label beside a terminal, a note to the next reader, the sentence
// that says why three agents are sitting together.
//
// It is a plain <textarea>, not an editor. A pin is what holds prose, and a
// pin panel already shows one with its markdown, its title and its tags; a
// second rich editor here would be two ways to write the same thing, and the
// heavier one would be the one with no title to find it by.
//
// **Saving is on blur and on a pause, never on a keystroke.** The store
// keeps the whole string and announces it (`canvas.panel.content`), so every
// key would be a transaction and an event to every other browser. The draft
// lives in this component until then, which is also what lets the panel be
// dragged, zoomed and unloaded mid-sentence without losing it.
const SAVE_AFTER_MS = 800;

export default function TextPanel({ panelId, content, onSave }) {
  const [draft, setDraft] = useState(content || "");
  const timer = useRef(0);
  const draftRef = useRef(draft);
  draftRef.current = draft;
  const savedRef = useRef(content || "");
  const saveRef = useRef(onSave);
  saveRef.current = onSave;

  // A change from somewhere else — another browser, this reader's other tab
  // — overwrites the box only when the box is not being typed in. What the
  // reader has in their hands wins over what the feed carries; the next save
  // reconciles it, and the last writer wins, which is what a shared label on
  // a plane should do.
  useEffect(() => {
    const next = content || "";
    if (next === savedRef.current) return;
    savedRef.current = next;
    if (!timer.current) setDraft(next);
  }, [content]);

  const flush = useCallback(() => {
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = 0;
    }
    const text = draftRef.current;
    if (text === savedRef.current) return;
    savedRef.current = text;
    if (saveRef.current) saveRef.current(text);
  }, []);

  // The pause timer is cleared on unmount, and what was typed is saved: a
  // panel unloaded by the chunk loader mid-sentence must not lose it.
  useEffect(() => () => flush(), [flush]);

  const onChange = useCallback((e) => {
    setDraft(e.target.value);
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => { timer.current = 0; flush(); }, SAVE_AFTER_MS);
  }, [flush]);

  return (
    <textarea
      className="cv-text nodrag nowheel"
      value={draft}
      onChange={onChange}
      onBlur={flush}
      maxLength={CANVAS_LIMITS.text}
      spellCheck={false}
      placeholder="Write here."
      aria-label="Panel text"
      data-panel={panelId}
    />
  );
}
