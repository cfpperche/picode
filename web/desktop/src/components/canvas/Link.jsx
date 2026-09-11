import { memo } from "react";
import { BaseEdge, EdgeLabelRenderer, getStraightPath } from "@xyflow/react";
import { IconLink, IconUnlink } from "../Icons.jsx";

// Link — how an edge draws on the plane (ADR-0116). One edge type, and
// it says two things and only two: these two panels are linked, and whether
// that link **currently grants** the mailbox contact.
//
// Straight, and with no arrowhead, because an edge is undirected (ADR-0116
// §6): the mailbox is symmetric, so a curve or a marker would promise a
// direction the store cannot keep. The line runs between the two connectors
// in the panels' headers, which is where the gesture starts and ends.
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
function Link({ id, sourceX, sourceY, targetX, targetY, selected, data }) {
  const [path, labelX, labelY] = getStraightPath({ sourceX, sourceY, targetX, targetY });
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
  const dx = targetX - sourceX;
  const dy = targetY - sourceY;
  const len = Math.hypot(dx, dy) || 1;
  // Both normals are perpendicular; the stored pair order decides which one
  // the arithmetic lands on, and that order is the panel ids sorted — not
  // anything the viewer can see. Pick the same side every time: up, and for
  // a vertical line, right.
  let nx = dy / len;
  let ny = -dx / len;
  if (ny > 0 || (ny === 0 && nx < 0)) { nx = -nx; ny = -ny; }
  return (
    <>
      <BaseEdge
        id={id}
        path={path}
        className={"cv-edge" + state + (selected ? " is-selected" : "")}
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
