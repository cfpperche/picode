import { memo } from "react";
import { BaseEdge, EdgeLabelRenderer, getStraightPath } from "@xyflow/react";
import { IconLink, IconUnlink } from "../Icons.jsx";

// MatrixLink — how an edge draws on the plane (ADR-0116). One edge type, and
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
// colour, with an unlink mark, the word "Broken", and — selected or hovered
// — the one line that says which end and why. Nothing here decides that:
// `data.grant` is matrixGrants.js's answer, the same one the Messages audit
// list reads.
//
// The chip is a real button and the only click target that removes: it is in
// the tab order, so an owner who never touches the plane can still revoke a
// grant from here. Selecting the line and pressing Delete does the same
// thing (MatrixCanvas owns that key, because React Flow's own delete key is
// off — Delete on a focused panel removes the panel).
function MatrixLink({ id, sourceX, sourceY, targetX, targetY, selected, data }) {
  const [path, labelX, labelY] = getStraightPath({ sourceX, sourceY, targetX, targetY });
  const grants = !!(data && data.grants);
  const reason = (data && data.reason) || "";
  const label = (data && data.label) || "link";
  const state = grants ? " is-live" : " is-broken";
  return (
    <>
      <BaseEdge
        id={id}
        path={path}
        className={"mx-edge" + state + (selected ? " is-selected" : "")}
        interactionWidth={22}
      />
      <EdgeLabelRenderer>
        <div
          className={"mx-edge-label" + state + (selected ? " is-selected" : "")}
          style={{ transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)` }}
        >
          <div className="mx-edge-chip-wrap">
            <button
              type="button"
              className="mx-edge-chip nodrag nopan"
              title={reason || `${label} — remove this link`}
              aria-label={(reason ? reason + " " : "") + "Remove the link between " + label}
              onClick={(e) => { e.stopPropagation(); data.onRemove(id); }}
            >
              {grants ? <IconLink size={12} /> : <IconUnlink size={12} />}
              {grants ? null : <span className="mx-edge-word">Broken</span>}
            </button>
            {reason && selected ? <span className="mx-edge-reason" role="status">{reason}</span> : null}
          </div>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}

export default memo(MatrixLink);
