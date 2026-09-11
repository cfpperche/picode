// A real sample of a Canvas plane pattern, for the preference card that
// offers it. The geometry is React Flow's own — a circle per tile for dots,
// the tile-crossing path for lines, the same path kept short for cross — and
// the colours are the tokens Plane.jsx feeds the plane (canvas.css), so the
// card cannot drift from what the plane draws.
//
// The one deliberate difference is density: the plane tiles every 32px, the
// swatch every 16. A card is a tenth of the plane's width, and at the plane's
// own spacing a Cross sample would hold two marks and read as Plain.
const TILE = 16;

export default function PatternSwatch({ kind }) {
  const id = `bg-swatch-${kind}`;
  return (
    <span className="bg-swatch" data-pattern={kind} aria-hidden="true">
      {kind === "plain" ? null : (
        <svg width="100%" height="100%" focusable="false">
          <defs>
            <pattern id={id} x="0" y="0" width={TILE} height={TILE} patternUnits="userSpaceOnUse">
              {kind === "dots" ? <circle cx="1" cy="1" r="1" className="bg-swatch-dot" /> : null}
              {kind === "lines" ? <path d={`M${TILE / 2} 0 V${TILE} M0 ${TILE / 2} H${TILE}`} className="bg-swatch-line" /> : null}
              {kind === "cross" ? <path d={`M${TILE / 2} ${TILE / 2 - 3} v6 M${TILE / 2 - 3} ${TILE / 2} h6`} className="bg-swatch-line" /> : null}
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill={`url(#${id})`} />
        </svg>
      )}
    </span>
  );
}
