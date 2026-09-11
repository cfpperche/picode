import { useMemo, useRef } from "react";
import GridLayout, { useContainerWidth } from "react-grid-layout";
import { fastVerticalCompactor } from "react-grid-layout/extras";
import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import { MATRIX_LIMITS } from "@picode/shared/domain/matrix.js";
import Panel from "./Panel.jsx";

// MatrixGrid — the react-grid-layout 2.2.4 wrapper, the v2 API the phase-0
// study measured (docs/benchmarks/2026-09-09-matrix-live-grid.md): width
// from useContainerWidth on the canvas, 12 columns of 24 px rows with an
// 8 px gutter, drag by the panel header, the header buttons as the cancel
// zone, corner and edge resize, the O(n log n) vertical compactor from
// `extras`, children keyed by panel id. Gesture edges reach the surface
// (pin the panel, hold feed events); onLayoutChange carries the measured
// width so the surface can ignore a change it cannot trust (hidden, 0).
//
// Edges belong to the matrix, not to a mode (ADR-0116), but a grid has no
// plane to draw one on: `links` is the per-panel count its header wears
// instead, and the Messages audit list is where a grid-mode owner reads and
// revokes them.
const GRID = { cols: MATRIX_LIMITS.cols, rowHeight: 24, margin: [8, 8], containerPadding: [0, 0] };
const DRAG = { enabled: true, handle: ".mx-head", cancel: ".mx-actions", threshold: 3 };
const RESIZE = { enabled: true, handles: ["se", "s", "e"] };

// compactPanels(panels) -> [{id, x, y, w, h}]: the rows as the grid will
// draw them. react-grid-layout compacts a controlled `layout` silently —
// when only the prop moved and the compacted result did not, it fires no
// onLayoutChange — so the surface runs the same compactor over the store
// and saves the difference itself (plan §4.7's "a compaction that shifted
// 200 panels sends 200 rows once"). The compactor clones its input.
export function compactPanels(panels) {
  const layout = panels.map((p) => ({ i: p.id, x: p.x, y: p.y, w: p.w, h: p.h, minW: MATRIX_LIMITS.minW, minH: MATRIX_LIMITS.minH }));
  return fastVerticalCompactor.compact(layout, MATRIX_LIMITS.cols).map((l) => ({ id: l.i, x: l.x, y: l.y, w: l.w, h: l.h }));
}

export default function MatrixGrid({ models, loaded, hidden, focusedId, engaged, maximizedId, tabStopId, loader, handlers, links, onLayoutChange, onGestureStart, onGestureStop }) {
  const { width, containerRef } = useContainerWidth({ initialWidth: 0 });
  // A hidden tab measures 0: keep drawing at the last real width so the
  // wrappers keep their places (and their scroll offset) until the reveal.
  const lastWidth = useRef(0);
  if (width > 0) lastWidth.current = width;
  const drawWidth = width > 0 ? width : lastWidth.current || 1200;
  const layout = useMemo(
    () => models.map((m) => ({ i: m.id, x: m.x, y: m.y, w: m.w, h: m.h, minW: MATRIX_LIMITS.minW, minH: MATRIX_LIMITS.minH })),
    [models],
  );
  return (
    <div className="mx-canvas" ref={containerRef}>
      <GridLayout
        layout={layout}
        width={drawWidth}
        gridConfig={GRID}
        dragConfig={DRAG}
        resizeConfig={RESIZE}
        compactor={fastVerticalCompactor}
        onLayoutChange={(next) => onLayoutChange(next, width)}
        onDragStart={(l, item) => onGestureStart(item ? item.i : "")}
        onDragStop={(l, item) => onGestureStop(item ? item.i : "")}
        onResizeStart={(l, item) => onGestureStart(item ? item.i : "")}
        onResizeStop={(l, item) => onGestureStop(item ? item.i : "")}
      >
        {models.map((m) => (
          <Panel
            key={m.id}
            model={m}
            loaded={loaded.has(m.id)}
            hidden={hidden}
            focused={focusedId === m.id}
            engaged={engaged}
            maximized={maximizedId === m.id}
            tabStop={tabStopId === m.id}
            loader={loader}
            handlers={handlers}
            links={links[m.id]}
          />
        ))}
      </GridLayout>
    </div>
  );
}
