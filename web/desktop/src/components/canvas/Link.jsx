import { memo } from "react";
import { BaseEdge, EdgeLabelRenderer } from "@xyflow/react";
import { borderPoint, linkPath, pointAnchor } from "@picode/shared/domain/canvasAnchors.js";
import { IconLink, IconUnlink } from "../Icons.jsx";

// Link — how an edge draws on the plane (ADR-0116). One edge type, and
// it says two things and only two: these two panels are linked, and whether
// that link **currently grants** the mailbox contact.
//
// A curve, and with no arrowhead, because an edge is undirected (ADR-0116
// §6): the mailbox is symmetric, so a marker would promise a direction the
// store cannot keep.
//
// **Where it touches** is not this component's decision and not a handle's
// either. The plane hands each edge an `anchor` — two points and the two
// borders they sit on — computed by `canvasAnchors.js` from the two
// rectangles and from every other link the same panels carry: the border
// facing the other panel, then a spread along that border so several links
// out of one panel do not stack. This is React Flow's *floating edge*, and
// it replaces the version that read `sourceX`/`sourceY` off a single
// `Position.Top` handle and drew with `getSimpleBezierPath`, which
// distinguishes only the horizontal sides: "top" was never honoured, the
// line touched wherever the curve happened to land (a capture of the shipped
// build has it leaving one panel near its bottom-right corner), and a panel's
// second link left from the same point as its first.
//
// The curve itself is `linkPath`'s, for one reason: the chip needs the
// curve's own centre and its normal *at that centre*, and only whoever placed
// the control points can answer that. It puts each control point along its
// border's outward normal, so the line leaves and enters at a right angle —
// which is what makes the tie read as attached to the two cards rather than
// ruled across them — and it is symmetric, so swapping the ends (which the
// store's sorted pair does arbitrarily) draws the same shape.
//
// One honest consequence, so nobody files it twice: two panels tidied into
// the same row are joined by a straight horizontal segment, because the
// shortest tie between two facing borders at the same height is a straight
// line. What changed is that it is now a short segment in the gap between
// them instead of a diagonal rule from one header to the other, drawn across
// whatever sat in between.
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
// while the pointer is still down. The spread is between links to *different*
// panels; the normal offset below is the *chip's*, pushing one label off its
// own line.
//
// The chip is a real button and the click target that removes; selecting the
// line and pressing Delete does the same thing (Plane owns that key,
// because React Flow's own delete key is off — Delete on a focused panel
// removes the panel).
//
// **When the chip is drawn** is a decision the first browser pass forced. A
// line whose midpoint lands on a panel covered the name of whatever card it
// crossed. A link that grants keeps its chip for the pointer that asks —
// hover or selection — and a link that grants nothing keeps it always,
// because that is the state the owner must not be able to miss. The grace
// period is what stops the chip vanishing as the pointer travels the gap
// from the line to it.
function Link({ id, sourceX, sourceY, targetX, targetY, selected, data }) {
  // The anchor is the plane's (`canvasAnchors.anchorLinks`), because the
  // spread needs every link both panels carry and an edge can only see its
  // own. The fallback is the handle geometry React Flow hands every edge, so
  // a frame rendered before the plane's first anchor pass still draws a line
  // between the two connectors rather than nothing.
  const anchor = (data && data.anchor) || {
    sourceX, sourceY, sourceSide: sourceX <= targetX ? "right" : "left",
    targetX, targetY, targetSide: sourceX <= targetX ? "left" : "right",
  };
  // labelX / labelY are the **curve's** centre and nx / ny its normal there,
  // from the same function that drew it — never a midpoint computed here,
  // which would put the chip off the line the moment the line bows.
  const { path, labelX, labelY, nx, ny } = linkPath(anchor);
  const grants = !!(data && data.grants);
  const reason = (data && data.reason) || "";
  const label = (data && data.label) || "link";
  const state = grants ? " is-live" : " is-broken";
  const shown = !grants || selected || !!(data && data.hovered);
  // The chip does not sit *on* the line's centre. That point lands on a
  // panel often enough — and, now that a link runs border to border, in the
  // gap between two close panels where there is no room — so the whole group
  // is pushed one chip-height along the line's normal: for the common
  // side-by-side pair that is straight up, into the empty plane. The offset
  // is written as a direction and scaled in CSS by 1 / zoom, so it is the
  // same number of screen pixels at every zoom. Which of the two normals
  // `linkPath` returns is settled there (always up, and right for a vertical
  // line), because the stored pair order is nothing a viewer can see.
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

// ConnectionPreview — the line under the pointer during a drag, and the
// reason it is a component at all: a preview that anchors differently from
// the edge it becomes is a preview that lies. React Flow's built-in
// connection line reads the handle's own position and one of its three path
// functions; this one runs exactly the two functions the finished edge runs.
//
// The moving end has no rectangle, so the pointer stands in as a zero-sized
// one (`pointAnchor`) and the fixed end still picks the border facing it. Let
// go over a panel the drop can land on and the far end jumps to *that*
// panel's facing border — which is where the edge will be a moment later.
//
// The one thing it cannot show is the spread: the link being drawn is not in
// the set yet, so it takes the point facing its target and joins the fan when
// the store answers. Dropping onto a side that already carries links is
// therefore the one gesture where the line moves slightly on release.
export function ConnectionPreview({ fromNode, toNode, toX, toY, connectionStatus }) {
  const from = rectOf(fromNode);
  if (!from) return null;
  const landing = connectionStatus !== "invalid" && toNode && toNode.id !== fromNode.id ? rectOf(toNode) : null;
  const to = landing || pointAnchor(toX, toY);
  const a = borderPoint(from, to);
  const b = landing ? borderPoint(landing, from) : { x: toX, y: toY, side: opposite(a.side) };
  const { path } = linkPath({
    sourceX: a.x, sourceY: a.y, sourceSide: a.side,
    targetX: b.x, targetY: b.y, targetSide: b.side,
  });
  return <path d={path} fill="none" className="react-flow__connection-path" />;
}

const OPPOSITE = { top: "bottom", bottom: "top", left: "right", right: "left" };
const opposite = (side) => OPPOSITE[side] || "left";

// React Flow's internal node: the absolute position it keeps for the plane,
// and the size it measured from the DOM.
function rectOf(node) {
  const pos = node && node.internals && node.internals.positionAbsolute;
  const size = (node && node.measured) || {};
  if (!pos || !size.width || !size.height) return null;
  return { x: pos.x, y: pos.y, width: size.width, height: size.height };
}
