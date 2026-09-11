import { memo } from "react";
import { BaseEdge, EdgeLabelRenderer, Position, getSimpleBezierPath } from "@xyflow/react";
import { IconLink, IconUnlink } from "../Icons.jsx";

// Link — how an edge draws on the plane (ADR-0116). One edge type, and
// it says two things and only two: these two panels are linked, and whether
// that link **currently grants** the mailbox contact.
//
// A curve, and with no arrowhead, because an edge is undirected (ADR-0116
// §6): the mailbox is symmetric, so a marker would promise a direction the
// store cannot keep — but a *symmetric* curve promises nothing, and the
// straight diagonal this used to draw read as a stray rule across the
// plane rather than as a tie between two cards. The line runs between the
// two connectors in the panels' headers, which is where the gesture starts
// and ends.
//
// **Which curve**, decided by drawing all three of React Flow's on a real
// plane with the four arrangements that actually occur (side by side, one
// above the other, diagonal, overlapping columns):
//
//   * `getBezierPath` derives its control points from each handle's
//     declared side, and both of ours are `Position.Top`. With the target
//     *below* the source that makes the source control 6.25·√Δy **above**
//     the source (`calculateControlOffset` takes the sqrt branch for a
//     negative distance) while the target control sits at the midpoint: the
//     line leaves the top card going up, hooks over itself and comes back
//     down. A loop, exactly as feared.
//   * `getSmoothStepPath` draws orthogonal dog-legs with rounded corners.
//     It is honest and never loops, but it reads as *wiring* — a flowchart
//     of directed steps — which is the one thing an undirected mailbox
//     contact is not.
//   * `getSimpleBezierPath` ignores Top-vs-Bottom entirely (its `getControl`
//     only distinguishes the horizontal sides) and puts both control points
//     on the midline: `[x1, midY]` and `[x2, midY]`. That is a symmetric S
//     — the same shape whichever end you call source — and it cannot loop,
//     because neither control ever leaves the band between the two ends.
//
// So: simple bezier, and `connectionLineType` on the plane is
// `SimpleBezier`, so the line you drag is the line you get. One honest
// consequence to know before "fixing" it: an S-curve between two points
// that are **aligned** is a straight line, and that is geometry, not a bug
// — two panels tidied into the same row still join with a straight
// segment. The diagonal, which is the case that was reported, bows.
//
// §7 is the reason this is a component at all: "a broken end does not
// grant", and it must **read broken** rather than be a dotted line a viewer
// learns to ignore. So a link that grants nothing is drawn in the warn
// colour, with an unlink mark, the word "Broken", and — pointed at or
// selected — the one line that says which end and why, as text rather than
// as a `title` nobody can reach without a mouse hover they did not know to
// try. Nothing here decides that: `data.grants` is canvasGrants.js's answer,
// the same one the Messages audit list reads.
//
// Hovering the **chip** is what shows the reason, not only hovering the
// line: a link between two panels that sit close together can run entirely
// behind them, and then the line itself is nowhere to click. The chip is
// drawn above the nodes and is always there for a broken link, so the state
// and its reason are reachable whatever the layout does.
//
// **Two links between the same pair do not exist**, so there is no
// parallel-edge separation to keep: `validateEdge`
// (web/shared/domain/canvas.js) and the store's unique ordered pair both
// refuse a duplicate, and the plane's `isValidConnection` refuses the drop
// while the pointer is still down. The normal offset below is the *chip's*,
// pushing one label off its own line — not a fan of arcs. If a second edge
// per pair is ever allowed, this is where an index-varied curvature goes.
//
// The chip is a real button and the click target that removes; selecting the
// line and pressing Delete does the same thing (Plane owns that key,
// because React Flow's own delete key is off — Delete on a focused panel
// removes the panel).
//
// **When the chip is drawn** is a decision the first browser pass forced. A
// straight line between two headers has its midpoint on a panel more often
// than not, so a chip on every edge covered the name of whatever card it
// crossed. A link that grants keeps its chip for the pointer that asks —
// hover or selection — and a link that grants nothing keeps it always,
// because that is the state the owner must not be able to miss. The grace
// period is what stops the chip vanishing as the pointer travels the gap
// from the line to it.
function Link({ id, sourceX, sourceY, sourcePosition, targetX, targetY, targetPosition, selected, data }) {
  // labelX / labelY are the **curve's** centre, from the same function that
  // drew it — never a midpoint computed here, which would put the chip off
  // the line the moment the line stopped being straight.
  const [path, labelX, labelY] = getSimpleBezierPath({
    sourceX,
    sourceY,
    sourcePosition: sourcePosition || Position.Top,
    targetX,
    targetY,
    targetPosition: targetPosition || Position.Top,
  });
  const grants = !!(data && data.grants);
  const reason = (data && data.reason) || "";
  const label = (data && data.label) || "link";
  const state = grants ? " is-live" : " is-broken";
  const shown = !grants || selected || !!(data && data.hovered);
  // The chip does not sit *on* the line's midpoint. Two connectors in two
  // headers put that midpoint on a panel's name more often than not, so the
  // whole group is pushed one chip-height along the line's normal: for the
  // common side-by-side pair that is straight up, into the empty plane above
  // the headers. The offset is written as a direction and scaled in CSS by
  // 1 / zoom, so it is the same number of screen pixels at every zoom.
  // The normal is the **curve's**, not the chord's. For this cubic —
  // P0 (x1,y1), P1 (x1,m), P2 (x2,m), P3 (x2,y2) with m the midline — the
  // derivative at t = 0.5 works out to (2·Δx, Δy), so pushing along the
  // chord's normal would slide the chip along the curve instead of off it.
  const dx = targetX - sourceX;
  const dy = targetY - sourceY;
  const tx = 2 * dx;
  const ty = dy;
  const len = Math.hypot(tx, ty) || 1;
  // Both normals are perpendicular; the stored pair order decides which one
  // the arithmetic lands on, and that order is the panel ids sorted — not
  // anything the viewer can see. Pick the same side every time: up, and for
  // a vertical line, right.
  let nx = ty / len;
  let ny = -tx / len;
  if (ny > 0 || (ny === 0 && nx < 0)) { nx = -nx; ny = -ny; }
  return (
    <>
      <BaseEdge
        id={id}
        path={path}
        className={"cv-edge" + state + (selected ? " is-selected" : "")}
        // 22, and canvas.css gives the interaction path `non-scaling-stroke`
        // so those are 22 **screen** pixels at every zoom — as a plain SVG
        // width it was 22 plane units, which is 8.8 screen px at 0.4 and
        // 4.4 at the floor. The band a pointer has to find must not shrink
        // with the camera; the painted line still thins with everything
        // else, because that is what tells you how far out you are.
        interactionWidth={22}
      />
      {shown ? (
        <EdgeLabelRenderer>
          <div
            className={"cv-edge-label" + state + (selected ? " is-selected" : "")}
            style={{ transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`, "--cv-edge-nx": nx, "--cv-edge-ny": ny }}
            onPointerEnter={() => data.onHover(id, true)}
            onPointerLeave={() => data.onHover(id, false)}
          >
            <div className="cv-edge-chip-wrap">
              <button
                type="button"
                className="cv-edge-chip nodrag nopan"
                title={reason || `${label} — remove this link`}
                aria-label={(reason ? reason + " " : "") + "Remove the link between " + label}
                onClick={(e) => { e.stopPropagation(); data.onRemove(id); }}
              >
                {grants ? <IconLink size={12} /> : <IconUnlink size={12} />}
                {grants ? null : <span className="cv-edge-word">Broken</span>}
              </button>
              {reason && (selected || (data && data.hovered)) ? <span className="cv-edge-reason" role="status">{reason}</span> : null}
            </div>
          </div>
        </EdgeLabelRenderer>
      ) : null}
    </>
  );
}

export default memo(Link);
