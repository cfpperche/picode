import { sendLabel } from "../lib/annotate.js";

// AnnotateStrip — the chrome strip that replaces the URL bar while annotate
// mode is on (owner reference 2026-09-18): a close ✕ (exit), a trash
// (discard every annotation), three small icons (undo last, screenshot crops,
// shortcut hints), a hint line, and "Send N" carrying the pending count.
// One component for both shells: the desktop toolbar and the fallback
// preview render it with the same props.
const ICONS = {
  undo: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
      <path d="M6.5 3.5L3 7l3.5 3.5" />
      <path d="M3.5 7H10a3 3 0 010 6H8" />
    </svg>
  ),
  camera: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
      <rect x="1.5" y="4.5" width="13" height="9" rx="1.5" />
      <circle cx="8" cy="9" r="2.4" />
      <path d="M5.5 4.5l1-2h3l1 2" />
    </svg>
  ),
  kbd: (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
      <rect x="1.5" y="4" width="13" height="8" rx="1.5" />
      <path d="M4.5 7h1M7 7h1M9.5 7h1M11.5 7h.5M4.5 9.5h7" />
    </svg>
  ),
};

export default function AnnotateStrip({
  count = 0,
  total = 0,
  shotsOn = true,
  sending = false,
  onExit,
  onClear,
  onUndo,
  onToggleShots,
  onHint,
  onSend,
}) {
  const saved = Number(count) || 0;
  const all = Number(total) || 0;
  const draftOpen = all > saved;
  const hint = sending
    ? "Sending…"
    : all === 0
      ? "Click any element to pin a note"
      : draftOpen
        ? "Save the open note, then Send"
        : saved === 1
          ? "1 note ready"
          : `${saved} notes ready`;
  return (
    <div className="annot-strip" role="toolbar" aria-label="Annotating">
      <button type="button" className="annot-ic" title="Stop annotating (Esc)" aria-label="Stop annotating" onClick={onExit}>✕</button>
      <span className="annot-sep" aria-hidden="true" />
      <button type="button" className="annot-ic" title="Discard every annotation" aria-label="Discard every annotation" onClick={onClear}>🗑</button>
      <button type="button" className="annot-ic" title="Remove the last annotation" aria-label="Remove the last annotation" onClick={onUndo} disabled={all === 0}>{ICONS.undo}</button>
      <button
        type="button"
        className={"annot-ic" + (shotsOn ? " on" : "")}
        title={shotsOn ? "Screenshot crops: on (click to skip them)" : "Screenshot crops: off (click to include them)"}
        aria-label="Toggle screenshot crops"
        aria-pressed={shotsOn}
        onClick={onToggleShots}
      >{ICONS.camera}</button>
      <button type="button" className="annot-ic" title="Annotate shortcuts" aria-label="Annotate shortcuts" onClick={onHint}>{ICONS.kbd}</button>
      <span className="annot-hint">{hint}</span>
      <button
        type="button"
        className="annot-send"
        title={saved === 0 ? "Pin a note first — click any element" : `Send ${saved} annotation${saved === 1 ? "" : "s"} to the agent`}
        aria-label={sendLabel(saved)}
        disabled={saved === 0 || sending}
        onClick={onSend}
      >{sending ? "Sending…" : sendLabel(saved)}</button>
    </div>
  );
}
