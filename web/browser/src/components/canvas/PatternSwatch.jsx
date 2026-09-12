// A real sample of a plane pattern, drawn beside its row in the canvas's own
// Background submenu (CanvasSurface.jsx). The geometry is React Flow's own —
// a circle per tile for dots, the tile-crossing path for lines, the same path
// kept short for cross — and the colours are the tokens Plane.jsx feeds the
// plane (canvas.css), so the sample cannot drift from what the plane draws.
//
// It lives inside the app (ADR-0109's 2026-09-11 amendment): the ground is
// the Canvas's own setting, so nothing outside components/canvas/ draws it.
//
// The one deliberate difference is density: the plane tiles every 32px, the
// swatch every 8. The swatch is a 22px square, and at the plane's own spacing
// a Cross sample would hold no mark at all.
const TILE = 8;

export default function PatternSwatch({ kind }) {
  const id = `cv-bg-swatch-${kind}`;
  return (
    <span className="cv-bg-swatch" data-pattern={kind} aria-hidden="true">
      {kind === "plain" ? null : (
        <svg width="100%" height="100%" focusable="false">
          <defs>
            <pattern id={id} x="0" y="0" width={TILE} height={TILE} patternUnits="userSpaceOnUse">
              {kind === "dots" ? <circle cx={TILE / 2} cy={TILE / 2} r="1" className="cv-bg-swatch-dot" /> : null}
              {kind === "lines" ? <path d={`M${TILE / 2} 0 V${TILE} M0 ${TILE / 2} H${TILE}`} className="cv-bg-swatch-line" /> : null}
              {kind === "cross" ? <path d={`M${TILE / 2} ${TILE / 2 - 2} v4 M${TILE / 2 - 2} ${TILE / 2} h4`} className="cv-bg-swatch-line" /> : null}
            </pattern>
          </defs>
          <rect width="100%" height="100%" fill={`url(#${id})`} />
        </svg>
      )}
    </span>
  );
}
