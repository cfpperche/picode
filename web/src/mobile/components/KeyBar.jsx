// The keys a phone keyboard does not have, the way Termux, Blink and
// terminal-web lay them out (ADR-0044 amendment): one row that scrolls
// sideways — nothing shrinks, nothing clips — with the sticky Ctrl and
// Alt first, in the thumb zone. A key never summons the phone keyboard;
// ⌨ does. Ctrl/Alt arm for the next key (bar or phone) and light up.
import { IconKeyboard, IconX } from "../../components/Icons.jsx";

export const KEYS = [
  { id: "ctrl", label: "Ctrl", mod: "ctrl" },
  { id: "alt", label: "Alt", mod: "alt" },
  { id: "esc", label: "Esc", seq: "\x1b" },
  { id: "tab", label: "Tab", seq: "\t" },
  { id: "up", label: "↑", seq: "\x1b[A", gap: true },
  { id: "down", label: "↓", seq: "\x1b[B" },
  { id: "left", label: "←", seq: "\x1b[D" },
  { id: "right", label: "→", seq: "\x1b[C" },
  { id: "home", label: "Home", seq: "\x1b[H", gap: true },
  { id: "end", label: "End", seq: "\x1b[F" },
  { id: "pgup", label: "PgUp", seq: "\x1b[5~" },
  { id: "pgdn", label: "PgDn", seq: "\x1b[6~" },
  { id: "intr", label: "^C", seq: "\x03", gap: true, title: "Interrupt (Ctrl+C)" },
  { id: "pipe", label: "|", seq: "|", gap: true },
  { id: "tilde", label: "~", seq: "~" },
  { id: "slash", label: "/", seq: "/" },
  { id: "dash", label: "-", seq: "-" },
];

export default function KeyBar({ armed, onArm, onKey, onType, onClose }) {
  const still = (e) => e.preventDefault(); // keep focus where it is
  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys">
      <div className="m-keybar-row">
        {KEYS.map((k) => k.mod ? (
          <button key={k.id} type="button" className={"m-key m-key-mod" + (armed && armed[k.mod] ? " on" : "") + (k.gap ? " m-key-gap" : "")}
            aria-pressed={!!(armed && armed[k.mod])} title={k.label + " — arms the next key"} onPointerDown={still} onClick={() => onArm(k.mod)}>{k.label}</button>
        ) : (
          <button key={k.id} type="button" className={"m-key" + (k.gap ? " m-key-gap" : "")} title={k.title || k.label} onPointerDown={still} onClick={() => onKey(k.seq)}>{k.label}</button>
        ))}
        <button type="button" className="m-key m-key-icon m-key-gap" title="Type — open the keyboard" aria-label="Open the keyboard to type" onPointerDown={still} onClick={onType}><IconKeyboard size={16} /></button>
        <button type="button" className="m-key m-key-icon" title="Hide keys" aria-label="Hide terminal keys" onPointerDown={still} onClick={onClose}><IconX size={14} /></button>
      </div>
    </div>
  );
}
