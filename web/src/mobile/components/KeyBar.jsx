// The keys a phone keyboard does not have. Each sends its byte sequence to
// the terminal's socket and hands focus back to xterm, so the soft
// keyboard stays up. Two rows, 44px targets, thumb zone.
const KEYS = [
  [
    ["Esc", "\x1b"], ["Tab", "\t"], ["Ctrl+C", "\x03"], ["Ctrl+D", "\x04"], ["Ctrl+Z", "\x1a"], ["Ctrl+L", "\x0c"],
  ],
  [
    ["↑", "\x1b[A"], ["↓", "\x1b[B"], ["←", "\x1b[D"], ["→", "\x1b[C"], ["/", "/"], ["|", "|"], ["-", "-"], ["~", "~"],
  ],
];

import { IconX } from "../../components/Icons.jsx";

export default function KeyBar({ onKey, onClose }) {
  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys">
      {KEYS.map((row, i) => (
        <div key={i} className="m-keybar-row">
          {row.map(([label, seq]) => (
            <button key={label} type="button" className="m-key" onPointerDown={(e) => e.preventDefault()} onClick={() => onKey(seq)}>{label}</button>
          ))}
          {i === 0 && onClose ? (
            <button type="button" className="m-key m-key-close" title="Hide keys" aria-label="Hide terminal keys" onPointerDown={(e) => e.preventDefault()} onClick={onClose}><IconX size={14} /></button>
          ) : null}
        </div>
      ))}
    </div>
  );
}
