import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Background, BackgroundVariant, MiniMap, NodeResizer, ReactFlow, ReactFlowProvider, applyNodeChanges, useReactFlow } from "@xyflow/react";
import { CANVAS_ZOOM, MATRIX_LIMITS, UNIT_PX, normalizeViewport, pointerAtZoom, pxToUnits, unitsToPx, viewportKey } from "@picode/shared/domain/matrix.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import Panel from "./Panel.jsx";
import { hasPane } from "./PanelBody.jsx";
import { CANVAS_MARGIN } from "./chunkLoader.js";
import { captureStill, captureStills, readStill } from "./stills.js";
import "@xyflow/react/dist/style.css";

// MatrixCanvas — canvas mode's host (plan docs/plans/matrix-canvas.md §4;
// ADR-0113 is the model, C0 the measurements:
// docs/benchmarks/2026-09-10-node-canvas.md). A peer of MatrixGrid: the
// surface owns the store, the saving, the focus and the keyboard and hands
// this component the same panel models. Every panel is the same `Panel`
// wrapper rendered as a React Flow node type, so a panel behaves identically
// in both modes — one header, one status chip, one set of actions, one
// keyboard.
//
// The host is lazy-imported by the surface: eager, React Flow costs the
// desktop's main chunk +63.3 KB gzip (2.6× react-grid-layout); split, it
// costs 205 B and lands in a chunk only a viewer who opens a canvas fetches.
//
// What C0 measured, and why each line is the way it is:
//
//   * `onlyRenderVisibleElements` is **off**. It is the cheaper renderer,
//     but it unmounts a node with no dwell and no hysteresis, and a node's
//     unmount is a live terminal's unmount: three fast pan round trips
//     churned 54 socket suspends and kicks with it on, zero with it off.
//     Our own band (chunkLoader.js) already culls the thing that costs.
//   * `snapToGrid` with an 8 px `snapGrid` — the canvas unit (ADR-0113), so
//     a drag lands on whole units and the store never sees a fraction. The
//     resizer does not honour it, so a resize is rounded by `pxToUnits`.
//   * `nodrag` / `nowheel` on the body and the header buttons, the header as
//     the drag handle: a wheel over a pane scrolls the terminal instead of
//     zooming the plane, and a drag-select inside a pane selects text
//     instead of moving the panel.
//   * The minimap costs nothing at 500 nodes but ships as a near-white panel
//     on a dark app, so it is themed from our tokens (matrix.css) and placed
//     where it does not cover a panel.
//   * A live pane takes a pointer **only at zoom 1.0** — the one rule C0
//     changed. Everywhere else the body is pointer-inert and a click on it
//     snaps the plane to 1 before the pointer reaches xterm.
//
// The viewport is per viewer: localStorage `picode-matrix-view:<id>`,
// restored on open, fit to the panels the first time. A camera is not an
// edit (ADR-0113), so it never reaches the store.

const NODE_TYPE = "panel";
// Which panels get the snap-to-1 layer: the ones whose body is a terminal
// the pointer could miss, plus every name-plate — down there the plate *is*
// the panel, there is no header to reach, and a click has nothing else to
// mean than "bring me closer". A row that answers with one line and one
// action (a managed agent, a gone terminal, a stopped agent) keeps its own
// button at every zoom: it has no cell to land on.
const snapTarget = (data) => data.bodyKind === "plate" || (data.bodyKind !== "off" && !data.maximized && hasPane(data.model));
const SNAP = [UNIT_PX, UNIT_PX];
const MOTION_MS = 180;
const VIEW_SAVE_MS = 400;
const FIT = { padding: 0.12, maxZoom: CANVAS_ZOOM.exact };
const MIN_W_PX = MATRIX_LIMITS.canvasMinW * UNIT_PX;
const MIN_H_PX = MATRIX_LIMITS.canvasMinH * UNIT_PX;
const MAX_PX = MATRIX_LIMITS.canvasMax * UNIT_PX;
const PLANE_PX = MATRIX_LIMITS.canvasCoord * UNIT_PX;
const NODE_EXTENT = [[-PLANE_PX, -PLANE_PX], [PLANE_PX, PLANE_PX]];

function readView(id) {
  try {
    return normalizeViewport(JSON.parse(localStorage.getItem(viewportKey(id)) || "null"));
  } catch {
    return null;
  }
}
function writeView(id, v) {
  try {
    localStorage.setItem(viewportKey(id), JSON.stringify({ x: v.x, y: v.y, zoom: v.zoom }));
  } catch { /* storage off */ }
}

// PanelNode — the node type. Memoized: a pan or a zoom never reaches it
// (React Flow transforms the plane instead of re-rendering its nodes), and a
// data object is rebuilt only when that panel's own state changed, so a
// viewport change does not re-render 500 bodies.
const PanelNode = memo(function PanelNode({ id, data }) {
  return (
    <Panel
      model={data.model}
      loaded={data.loaded}
      hidden={data.hidden}
      focused={data.focused}
      engaged={data.engaged}
      maximized={data.maximized}
      tabStop={data.tabStop}
      loader={data.loader}
      handlers={data.handlers}
      bodyKind={data.bodyKind}
      still={data.still}
      note={data.note}
      pointer={data.pointer}
    >
      <NodeResizer
        nodeId={id}
        minWidth={MIN_W_PX}
        minHeight={MIN_H_PX}
        maxWidth={MAX_PX}
        maxHeight={MAX_PX}
        onResizeStart={data.onResizeStart}
        onResizeEnd={data.onResizeEnd}
      />
      {!data.pointer && snapTarget(data) ? (
        // The pointer layer. xterm maps a click as `cell × zoom` under the
        // plane's transform (C0), and PiCode ships tmux `mouse on`, so a
        // click at 0.8 reaches copy mode and every TUI at the wrong cell.
        // Nothing on screen would say so — hence this: the click snaps the
        // plane to 1 and then the pointer reaches the pane. Below the band
        // it is also what turns a still back into a terminal. The keyboard
        // path (Enter to engage) snaps the same way, so the layer is a
        // pointer affordance and stays out of the tab order.
        // Below 0.4 there is no header to drag by, so the plate is the
        // handle and the layer lets a drag through — under React Flow's
        // 3 px threshold it is still a click, and a click still snaps.
        <button
          type="button"
          className={"mx-snap" + (data.bodyKind === "plate" ? "" : " nodrag")}
          tabIndex={-1}
          aria-hidden="true"
          onClick={data.onSnap}
        >
          <span className="mx-snap-label">Zoom to 100 %</span>
        </button>
      ) : null}
    </Panel>
  );
});

const NODE_TYPES = { [NODE_TYPE]: PanelNode };

// The node's data object is kept when nothing in it changed, so React Flow's
// own memo holds and a neighbour's drag never re-renders a body.
const DATA_KEYS = ["model", "loaded", "hidden", "focused", "engaged", "maximized", "tabStop", "bodyKind", "still", "note", "pointer"];
function sameData(a, b) {
  return !!a && !!b && DATA_KEYS.every((k) => a[k] === b[k]);
}

function Flow({ matrixId, models, loaded, bodies, hidden, focusedId, engaged, maximizedId, tabStopId, loader, handlers, onLayout, onGestureStart, onGestureStop, onReady }) {
  const rf = useReactFlow();
  const rootRef = useRef(null);
  const [nodes, setNodes] = useState([]);
  // The zoom lives in a ref — a pan or a zoom must not re-render a body —
  // and only what it *decides* is state: whether a pointer may reach a pane,
  // and the readout that tells the viewer why.
  const zoomRef = useRef(1);
  const [pointer, setPointer] = useState(true);
  const [zoomPct, setZoomPct] = useState(100);
  // A still is memory-only (stills.js), so this counter is how a fresh
  // capture reaches the nodes that show one.
  const [stillTick, setStillTick] = useState(0);
  const modelsRef = useRef(models);
  modelsRef.current = models;
  const bodiesRef = useRef(bodies);
  bodiesRef.current = bodies;
  const saveTimer = useRef(0);
  const fitted = useRef(false);
  const stored = useMemo(() => readView(matrixId), [matrixId]);

  // ---- the camera --------------------------------------------------------
  const applyZoom = useCallback((zoom) => {
    zoomRef.current = zoom;
    // One CSS variable carries the zoom to the name-plates, which size
    // themselves by 1 / zoom so they stay readable however far out you are.
    if (rootRef.current) rootRef.current.style.setProperty("--mx-zoom", String(zoom));
    const next = pointerAtZoom(zoom);
    setPointer((cur) => (cur === next ? cur : next));
    const pct = Math.round(zoom * 100);
    setZoomPct((cur) => (cur === pct ? cur : pct));
  }, []);

  const centerOn = useCallback((id, zoom) => {
    const node = rf.getNode(id);
    if (!node) return false;
    const w = node.measured?.width || node.width || 0;
    const h = node.measured?.height || node.height || 0;
    rf.setCenter(node.position.x + w / 2, node.position.y + h / 2, { zoom, duration: MOTION_MS });
    return true;
  }, [rf]);

  // snapToOne: "snap to 1 on engage" (plan §4.3). Nothing to do when the
  // plane is already there — the common case, and an animation for nothing
  // is worse than none.
  const snapToOne = useCallback((id) => {
    if (pointerAtZoom(zoomRef.current)) return;
    if (!id || !centerOn(id, CANVAS_ZOOM.exact)) rf.zoomTo(CANVAS_ZOOM.exact, { duration: MOTION_MS });
  }, [centerOn, rf]);
  const snapRef = useRef(snapToOne);
  snapRef.current = snapToOne;

  // reveal(id): pan an off-screen panel into view — which is what loads it.
  // A panel already on screen is left alone: an arrow key must not shove the
  // plane about.
  const reveal = useCallback((id) => {
    const node = rf.getNode(id);
    const el = rootRef.current;
    if (!node || !el) return;
    const vp = rf.getViewport();
    const box = el.getBoundingClientRect();
    const x0 = -vp.x / vp.zoom;
    const y0 = -vp.y / vp.zoom;
    const w = node.measured?.width || node.width || 0;
    const h = node.measured?.height || node.height || 0;
    const inside = node.position.x >= x0 && node.position.y >= y0
      && node.position.x + w <= x0 + box.width / vp.zoom
      && node.position.y + h <= y0 + box.height / vp.zoom;
    if (!inside) centerOn(id, vp.zoom);
  }, [centerOn, rf]);

  const fit = useCallback(() => { rf.fitView({ ...FIT, duration: MOTION_MS }); }, [rf]);
  const zoomIn = useCallback(() => { rf.zoomIn({ duration: MOTION_MS }); }, [rf]);
  const zoomOut = useCallback(() => { rf.zoomOut({ duration: MOTION_MS }); }, [rf]);

  // ---- gestures ----------------------------------------------------------
  // A drag or a resize moves the local nodes at once; the store — and the
  // PATCH of exactly what moved — takes the answer when the gesture ends, so
  // a marquee drag of fifty panels is one request of fifty rows.
  const saveNodes = useCallback((list) => {
    const rows = (list || []).map((n) => ({
      id: n.id,
      // `width`/`height` are ours (we set them from the store and the
      // resizer writes them back); `measured` is the DOM's and lags a
      // rounded resize by a frame.
      ...pxToUnits({ x: n.position.x, y: n.position.y, width: n.width ?? n.measured?.width, height: n.height ?? n.measured?.height }),
    }));
    if (rows.length) onLayout(rows);
  }, [onLayout]);

  const gestures = useRef({});
  gestures.current = {
    resizeStart: (id) => onGestureStart(id),
    resizeEnd: (id, params) => {
      onGestureStop(id);
      if (params) onLayout([{ id, ...pxToUnits(params) }]);
    },
  };
  const onNodesChange = useCallback((changes) => { setNodes((cur) => applyNodeChanges(changes, cur)); }, []);
  const onNodeDragStart = useCallback((_e, node) => { onGestureStart(node ? node.id : ""); }, [onGestureStart]);
  const onNodeDragStop = useCallback((_e, node, dragged) => {
    onGestureStop(node ? node.id : "");
    saveNodes(dragged && dragged.length ? dragged : node ? [node] : []);
  }, [onGestureStop, saveNodes]);
  const onSelectionDragStart = useCallback(() => { onGestureStart(""); }, [onGestureStart]);
  const onSelectionDragStop = useCallback((_e, dragged) => { onGestureStop(""); saveNodes(dragged); }, [onGestureStop, saveNodes]);

  // Per-node callbacks, stable for the life of the host, so they never take
  // part in the data comparison above.
  const nodeCallbacks = useCallback((id) => ({
    loader,
    handlers,
    onSnap: () => snapRef.current(id),
    onResizeStart: () => gestures.current.resizeStart(id),
    onResizeEnd: (_evt, params) => gestures.current.resizeEnd(id, params),
  }), [loader, handlers]);

  // ---- the still, captured before the flip --------------------------------
  // C0: React renders the still body before it runs the live body's cleanup,
  // so a capture in the cleanup is always one crossing late — the first
  // crossing on a fresh page painted an empty still. The loader calls this
  // with the bodies that are about to stop being live, before it emits.
  useEffect(() => {
    loader.beforeChange = (prev, next) => {
      const refs = modelsRef.current.filter((m) => prev[m.id] === "live" && next[m.id] !== "live").map((m) => m.ref);
      if (!refs.length) return;
      captureStills(refs);
      setStillTick((n) => n + 1);
    };
    return () => { loader.beforeChange = null; };
  }, [loader]);

  // ---- the observer root is the plane, and its margin has both axes -------
  const setRoot = useCallback((el) => {
    rootRef.current = el;
    if (el) el.style.setProperty("--mx-zoom", String(zoomRef.current));
    loader.attach(el, CANVAS_MARGIN);
  }, [loader]);

  // ---- nodes --------------------------------------------------------------
  const nodeData = useMemo(() => {
    const out = new Map();
    for (const m of models) {
      const kind = bodies[m.id] || (loaded.has(m.id) ? "live" : "off");
      const still = kind === "still" ? readStill(m.ref) : null;
      // A still says how old it is only when the feed has seen the panel do
      // something since: an age on a screen nothing has touched is noise.
      const note = still && m.stamp && Date.parse(m.stamp) > still.at
        ? "Still · " + relTime(new Date(still.at).toISOString())
        : "";
      out.set(m.id, {
        model: m,
        loaded: loaded.has(m.id),
        hidden,
        focused: focusedId === m.id,
        engaged,
        maximized: maximizedId === m.id,
        tabStop: tabStopId === m.id,
        bodyKind: kind,
        still,
        note,
        pointer,
      });
    }
    return out;
  }, [models, loaded, bodies, hidden, focusedId, engaged, maximizedId, tabStopId, pointer, stillTick]);

  useEffect(() => {
    setNodes((cur) => {
      const by = new Map(cur.map((n) => [n.id, n]));
      let changed = cur.length !== nodeData.size;
      const next = [];
      for (const [id, data] of nodeData) {
        const prev = by.get(id);
        const box = unitsToPx(data.model);
        // A node under a gesture owns its own geometry until the gesture
        // ends; the store is the truth every other moment.
        const busy = prev && (prev.dragging || prev.resizing);
        const position = busy || (prev && prev.position.x === box.x && prev.position.y === box.y) ? prev.position : { x: box.x, y: box.y };
        const width = busy ? prev.width : box.width;
        const height = busy ? prev.height : box.height;
        if (prev && prev.position === position && prev.width === width && prev.height === height && sameData(prev.data, data)) {
          next.push(prev);
          continue;
        }
        changed = true;
        next.push({
          ...(prev || {}),
          id,
          type: NODE_TYPE,
          dragHandle: data.bodyKind === "plate" ? undefined : ".mx-head",
          position,
          width,
          height,
          data: prev && sameData(prev.data, data) ? prev.data : { ...data, ...nodeCallbacks(id) },
        });
      }
      return changed ? next : cur;
    });
  }, [nodeData, nodeCallbacks]);

  // ---- the first camera ---------------------------------------------------
  // A still can also be *missing* rather than stale: the camera can arrive
  // below the band without ever crossing it here — a stored viewport, or the
  // switch from grid mode, where the panes went live in the other host and
  // were suspended before this one mounted. Their xterm is still in the
  // registry (a suspend keeps the instance), so their screens are still
  // capturable, and filling the gaps is the same 0.02 ms.
  const fillStills = useCallback(() => {
    let got = 0;
    for (const m of modelsRef.current) if (!readStill(m.ref) && captureStill(m.ref)) got += 1;
    if (got) setStillTick((n) => n + 1);
  }, []);
  // enterZoom: the camera arrived somewhere — restored, fitted or switched
  // into.
  const enterZoom = useCallback((zoom) => {
    applyZoom(zoom);
    loader.setZoom(zoom);
    if (zoom < CANVAS_ZOOM.live) fillStills();
  }, [applyZoom, loader, fillStills]);
  useEffect(() => { enterZoom(stored ? stored.zoom : CANVAS_ZOOM.exact); }, [enterZoom, stored]);
  // Fit the plane the first time this viewer opens this matrix. The `fitView`
  // prop cannot do it: the first render has no nodes yet.
  useEffect(() => {
    if (fitted.current || stored || !nodes.length) return;
    fitted.current = true;
    rf.fitView(FIT);
    enterZoom(rf.getZoom());
  }, [nodes.length, stored, rf, enterZoom]);

  const onMove = useCallback((_evt, viewport) => { applyZoom(viewport.zoom); }, [applyZoom]);
  // The band flips when the gesture ends, not once per wheel frame: one
  // crossing per gesture instead of sixty, which is the same reason the
  // threshold is hysteretic at all.
  const onMoveEnd = useCallback((_evt, viewport) => {
    applyZoom(viewport.zoom);
    loader.setZoom(viewport.zoom);
    // While the panes are live, refresh their stills — a capture is 0.02 ms
    // (C0) — so the picture the viewer meets on the way out is current. Those
    // bodies are not showing a still, so nothing has to re-render for it.
    if (viewport.zoom >= CANVAS_ZOOM.live) {
      captureStills(modelsRef.current.filter((m) => bodiesRef.current[m.id] === "live").map((m) => m.ref));
    } else {
      fillStills();
    }
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => { saveTimer.current = 0; writeView(matrixId, viewport); }, VIEW_SAVE_MS);
  }, [applyZoom, loader, matrixId, fillStills]);
  useEffect(() => () => { if (saveTimer.current) clearTimeout(saveTimer.current); }, []);

  // ---- what the surface's keyboard drives ---------------------------------
  useEffect(() => {
    if (!onReady) return undefined;
    onReady({ snapToOne, reveal, fit, zoomIn, zoomOut, zoom: () => zoomRef.current });
    return () => onReady(null);
  }, [onReady, snapToOne, reveal, fit, zoomIn, zoomOut]);

  return (
    <div className="mx-canvas-flow" ref={setRoot}>
      <ReactFlow
        nodes={nodes}
        nodeTypes={NODE_TYPES}
        onNodesChange={onNodesChange}
        onNodeDragStart={onNodeDragStart}
        onNodeDragStop={onNodeDragStop}
        onSelectionDragStart={onSelectionDragStart}
        onSelectionDragStop={onSelectionDragStop}
        onMove={onMove}
        onMoveEnd={onMoveEnd}
        minZoom={CANVAS_ZOOM.min}
        maxZoom={CANVAS_ZOOM.max}
        snapToGrid
        snapGrid={SNAP}
        onlyRenderVisibleElements={false}
        nodeExtent={NODE_EXTENT}
        defaultViewport={stored || undefined}
        selectionOnDrag
        panOnDrag={[1, 2]}
        selectNodesOnDrag={false}
        nodesFocusable={false}
        disableKeyboardA11y
        deleteKeyCode={null}
        nodeDragThreshold={3}
        attributionPosition="bottom-left"
      >
        <Background variant={BackgroundVariant.Dots} gap={UNIT_PX * 4} size={1} />
        <MiniMap pannable zoomable ariaLabel="Panels on the plane" />
      </ReactFlow>
      <div className="mx-zoom" role="group" aria-label="Zoom" data-align-row>
        <button type="button" className="mx-zoom-btn" aria-label="Zoom out" title="Zoom out (−)" onClick={zoomOut}>−</button>
        <button
          type="button"
          className="mx-zoom-pct"
          title="Back to 100 % — the only zoom where a click lands on the cell it points at"
          onClick={() => snapToOne(focusedId)}
        >
          {zoomPct} %
        </button>
        <button type="button" className="mx-zoom-btn" aria-label="Zoom in" title="Zoom in (+)" onClick={zoomIn}>+</button>
        <button type="button" className="mx-zoom-fit" title="Fit every panel (0)" onClick={fit}>Fit</button>
      </div>
    </div>
  );
}

export default function MatrixCanvas(props) {
  return (
    <ReactFlowProvider>
      <Flow {...props} />
    </ReactFlowProvider>
  );
}
