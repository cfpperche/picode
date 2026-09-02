// The keys a phone keyboard does not have. Each sends its byte sequence
// to the terminal's socket. A key never summons the phone keyboard: if
// it was up it stays up, if it was down it stays down — the ⌨ key is
// the one that opens it. Two rows, 44px targets, thumb zone, groups
// separated by a little air: escapes and control chords; arrows and
// the symbols a mobile keyboard buries.
import { IconKeyboard, IconX } from "../../components/Icons.jsx";

const ROWS = [
  [
    { g: [["Esc", "\x1b"], ["Tab", "\t"]] },
    { g: [["Ctrl+C", "\x03"], ["Ctrl+D", "\x04"], ["Ctrl+Z", "\x1a"], ["Ctrl+L", "\x0c"]] },
  ],
  [
    { g: [["↑", "\x1b[A"], ["↓", "\x1b[B"], ["←", "\x1b[D"], ["→", "\x1b[C"]] },
    { g: [["/", "/"], ["|", "|"], ["-", "-"], ["~", "~"]] },
  ],
];

export default function KeyBar({ onKey, onType, onClose }) {
  const still = (e) => e.preventDefault(); // keep focus where it is
  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys">
      {ROWS.map((groups, i) => (
        <div key={i} className="m-keybar-row">
          {groups.map((grp, j) => (
            <div key={j} className="m-keygroup">
              {grp.g.map(([label, seq]) => (
                <button key={label} type="button" className="m-key" onPointerDown={still} onClick={() => onKey(seq)}>{label}</button>
              ))}
            </div>
          ))}
          <div className="m-keygroup m-keygroup-end">
            {i === 0 ? (
              <button type="button" className="m-key m-key-icon" title="Type — open the keyboard" aria-label="Open the keyboard to type" onPointerDown={still} onClick={onType}><IconKeyboard size={16} /></button>
            ) : (
              <button type="button" className="m-key m-key-icon m-key-close" title="Hide keys" aria-label="Hide terminal keys" onPointerDown={still} onClick={onClose}><IconX size={14} /></button>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
