// Termux's extra keys, cell for cell (ADR-0044 amendment): two rows of
// seven equal cells on the terminal's own background, no boxes, labels
// in small capitals. Row one: ESC / — HOME ↑ END PGUP. Row two: TAB CTRL
// ALT ← ↓ → PGDN. Ctrl and Alt are sticky (lib/termSticky.js): tap, and
// the next key — from the phone keyboard or the grid — carries the
// modifier. A key never summons the phone keyboard; tapping the terminal
// does, as in Termux. The header's keyboard icon shows or hides the grid.
export const ROWS = [
  [
    { id: "esc", label: "ESC", seq: "\x1b" },
    { id: "slash", label: "/", seq: "/" },
    { id: "dash", label: "—", seq: "-", title: "Dash" },
    { id: "home", label: "HOME", seq: "\x1b[H" },
    { id: "up", label: "↑", seq: "\x1b[A" },
    { id: "end", label: "END", seq: "\x1b[F" },
    { id: "pgup", label: "PGUP", seq: "\x1b[5~" },
  ],
  [
    { id: "tab", label: "⇆", seq: "\t", title: "Tab" },
    { id: "ctrl", label: "CTRL", mod: "ctrl" },
    { id: "alt", label: "ALT", mod: "alt" },
    { id: "left", label: "←", seq: "\x1b[D" },
    { id: "down", label: "↓", seq: "\x1b[B" },
    { id: "right", label: "→", seq: "\x1b[C" },
    { id: "pgdn", label: "PGDN", seq: "\x1b[6~" },
  ],
];

export default function KeyBar({ armed, onArm, onKey }) {
  const still = (e) => e.preventDefault(); // keep focus where it is
  return (
    <div className="m-keybar" role="toolbar" aria-label="Terminal keys">
      {ROWS.map((row, i) => (
        <div key={i} className="m-keybar-row">
          {row.map((k) => k.mod ? (
            <button key={k.id} type="button" className={"m-key m-key-mod" + (armed && armed[k.mod] ? " on" : "")}
              aria-pressed={!!(armed && armed[k.mod])} title={k.label + " — arms the next key"} onPointerDown={still} onClick={() => onArm(k.mod)}>{k.label}</button>
          ) : (
            <button key={k.id} type="button" className="m-key" title={k.title || k.label} onPointerDown={still} onClick={() => onKey(k.seq)}>{k.label}</button>
          ))}
        </div>
      ))}
    </div>
  );
}
