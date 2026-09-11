import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Background, BackgroundVariant, ConnectionLineType, ConnectionMode, Handle, MiniMap, NodeResizer, Position, ReactFlow, ReactFlowProvider, applyNodeChanges, useReactFlow } from "@xyflow/react";
import { CANVAS_LIMITS, CANVAS_ZOOM, EDGE_KINDS, UNIT_PX, legacyViewportKey, normalizeViewport, pointerAtZoom, pxToUnits, unitsToPx, viewportKey } from "@picode/shared/domain/canvas.js";
import { CANVAS_PATTERN_EVENT, readCanvasPattern } from "@picode/shared/domain/canvasPattern.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import Panel from "./Panel.jsx";
import Link from "./Link.jsx";
import { IconLink } from "../Icons.jsx";
import { hasPane } from "./PanelBody.jsx";
import { CANVAS_MARGIN } from "./chunkLoader.js";
import { captureStill, captureStills, readStill } from "./stills.js";
import "@xyflow/react/dist/style.css";

// Plane — the surface a canvas's panels sit on (plan
// docs/plans/matrix-canvas.md §4; ADR-0113 is the model, C0 the
// measurements: docs/benchmarks/2026-09-10-node-canvas.md). Since ADR-0118 it
// is the only host there is, and it is a host and nothing more: the surface
// owns the store, the saving, the focus and the keyboard and hands this
// component the panel models. Every panel is the same `Panel` wrapper
// rendered as a React Flow node type — one header, one status chip, one set
// of actions, one keyboard, on the plane and in the maximize layer alike.
//
// The host is lazy-imported by the surface, and the surface is lazy-imported
// by App.jsx (ADR-0118): eager, React Flow alone cost the desktop's main
// chunk +63.3 KB gzip; split, it costs 205 B and lands in a chunk only a
// viewer who opens a canvas fetches.
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
//     on a dark app, so it is themed from our tokens (canvas.css) and placed
//     where it does not cover a panel.
//   * A live pane takes a pointer **only at zoom 1.0** — the one rule C0
//     changed. Everywhere else the body is pointer-inert and a click on it
//     snaps the plane to 1 before the pointer reaches xterm.
//
// The viewport is per viewer: localStorage `picode-canvas-view:<id>`,
// restored on open, fit to the panels the first time. A camera is not an
// edit (ADR-0113), so it never reaches the store.
//
// Edges (ADR-0116) are the second thing on the plane. `connectionMode` is
// **loose**, which is React Flow's word for "a handle may meet any handle":
// an edge is undirected (§6), so one connector per panel is the honest
// model, and `source`/`target` in `onConnect` are only the order the pointer
// happened to travel — the store sorts the pair before it writes. A drop is
// refused live for the two cases the store would refuse anyway (a panel to
// itself, a pair already linked) rather than answered with a 409; every
// other refusal is the surface's, because it is the surface that asks the
// owner first.

const NODE_TYPE = "panel";
const EDGE_TYPE = "link";
// One handle per panel, so a client that sends the two ids in either order
// still names the same connector on both ends.
const LINK_HANDLE = "link";
// React Flow snaps a dropped connection to the nearest handle inside this
// radius (plane pixels). The connector is a 16 px knob in a 28 px header:
// at 20 it took three tries to land on a neighbour, at 40 a drop in open
// space still found the panel the pointer was over.
const CONNECT_RADIUS = 40;
const HOVER_GRACE_MS = 180;
// Which panels get the snap-to-1 layer: the ones whose body is a terminal
// the pointer could miss, plus every name-plate — down there the plate *is*
// the panel, there is no header to reach, and a click has nothing else to
// mean than "bring me closer". A row that answers with one line and one
// action (a managed agent, a gone terminal, a stopped agent) keeps its own
// button at every zoom: it has no cell to land on.
const snapTarget = (data) => data.bodyKind === "plate" || (data.bodyKind !== "off" && !data.maximized && hasPane(data.model));
const SNAP = [UNIT_PX, UNIT_PX];
// The reader's ground (web/shared/domain/canvasPattern.js). "plain" is the
// absence of a <Background>, so it is a null here rather than a fourth
// variant React Flow does not have. The gap is four cells either way, so
// switching texture never moves a panel or changes what snapping means.
const BG_VARIANT = {
  dots: BackgroundVariant.Dots,
  lines: BackgroundVariant.Lines,
  cross: BackgroundVariant.Cross,
  plain: null,
};
const BG_GAP = UNIT_PX * 4;
const MOTION_MS = 180;
const VIEW_SAVE_MS = 400;
// Fit leaves the clusters their corners. The chrome floats on the plane
// now, so a symmetric 12 % padding is not enough: fitView centres the
// content inside the padded rect, and the spare height went to both ends
// equally — which put the first panel's header under the top-left cluster.
// These two bands are the cluster geometry in screen pixels: 12 px inset +
// 36 px control + a gap at the top, and the zoom cluster above the 150 px
// minimap at the bottom. A viewer can of course still drag a panel under a
// cluster; what this fixes is the one camera the surface chooses itself,
// and the one `Fit` goes back to.
const FIT_TOP = 60;
const FIT_BOTTOM = 204;
const FIT_SIDE = 24;
// On a pane too short to spare 264 px the bands shrink together rather than
// eating the whole viewport: chrome room is never worth more than a third
// of the height a reader came for.
function fitOpts(el) {
  const h = (el && el.clientHeight) || 0;
  const k = h && FIT_TOP + FIT_BOTTOM > h / 3 ? (h / 3) / (FIT_TOP + FIT_BOTTOM) : 1;
  return {
    padding: {
      top: Math.round(FIT_TOP * k) + "px",
      right: FIT_SIDE + "px",
      bottom: Math.round(FIT_BOTTOM * k) + "px",
      left: FIT_SIDE + "px",
    },
    maxZoom: CANVAS_ZOOM.exact,
  };
}
const MIN_W_PX = CANVAS_LIMITS.canvasMinW * UNIT_PX;
const MIN_H_PX = CANVAS_LIMITS.canvasMinH * UNIT_PX;
const MAX_PX = CANVAS_LIMITS.canvasMax * UNIT_PX;
const PLANE_PX = CANVAS_LIMITS.canvasCoord * UNIT_PX;
const NODE_EXTENT = [[-PLANE_PX, -PLANE_PX], [PLANE_PX, PLANE_PX]];

// readView also carries a camera stored under ADR-0118's old key across,
// once: the key moves, the old one is removed, and a reader who had this
// plane parked somewhere finds it there instead of fitted.
function readView(id) {
  try {
    const key = viewportKey(id);
    let raw = localStorage.getItem(key);
    if (raw === null) {
      const was = legacyViewportKey(id);
      raw = localStorage.getItem(was);
      if (raw !== null) {
        localStorage.setItem(key, raw);
        localStorage.removeItem(was);
      }
    }
    return normalizeViewport(JSON.parse(raw || "null"));
  } catch {
    return null;
  }
}
function writeView(id, v) {
  try {
    localStorage.setItem(viewportKey(id), JSON.stringify({ x: v.x, y: v.y, zoom: v.zoom }));
  } catch { /* storage off */ }
}

// LinkHandle — the connector (ADR-0116 §2: only the owner draws an edge, in
// the browser). It lives in the header's action row, which already carries
// `nodrag`, so a drag from it starts a connection and never moves the panel;
// React Flow adds its own `nodrag`/`nopan` on top. Hidden until the panel is
// hovered, focused or selected (canvas.css) — a plane of 200 panels must not
// wear 200 knobs — and only on the two kinds that have a mailbox: a note has
// none, the store refuses such an edge, and offering what the store refuses
// is how a UI teaches a lie.
const LinkHandle = memo(function LinkHandle({ name }) {
  return (
    <Handle
      type="source"
      position={Position.Top}
      id={LINK_HANDLE}
      className="cv-connect"
      title={"Link " + name + " to another panel — they will be able to message each other"}
    >
      <IconLink size={11} />
    </Handle>
  );
});

// PanelNode — the node type. Memoized: a pan or a zoom never reaches it
// (React Flow transforms the plane instead of re-rendering its nodes), and a
// data object is rebuilt only when that panel's own state changed, so a
// viewport change does not re-render 500 bodies.
const PanelNode = memo(function PanelNode({ id, data }) {
  return (
    <Panel
      connector={data.connectable ? <LinkHandle name={data.model.name} /> : null}
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
      links={data.links}
      pointer={data.pointer}
    >
      {/* The resize targets. The component stays — it is what drives the
          resize and what fires `onResizeEnd` once per gesture, which is the
          save path's debounce. What it ships is the target: handles sized
          in *plane* units, so they shrink with the camera, and line
          controls one plane pixel wide.
          `autoScale` off on purpose: the library's own answer is an inline
          `scale: max(1 / zoom, 1)` on the four handles only, which leaves
          the four edges untouched and does nothing at all above zoom 1.
          canvas.css counter-scales every control off --cv-zoom instead, so
          there is one rule and one place to read it; leaving both on would
          divide by the zoom twice. */}
      <NodeResizer
        nodeId={id}
        autoScale={false}
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
          className={"cv-snap" + (data.bodyKind === "plate" ? "" : " nodrag")}
          tabIndex={-1}
          aria-hidden="true"
          onClick={data.onSnap}
        >
          <span className="cv-snap-label">Zoom to 100 %</span>
        </button>
      ) : null}
    </Panel>
  );
});

const NODE_TYPES = { [NODE_TYPE]: PanelNode };
const EDGE_TYPES = { [EDGE_TYPE]: Link };

// The node's data object is kept when nothing in it changed, so React Flow's
// own memo holds and a neighbour's drag never re-renders a body.
const DATA_KEYS = ["model", "loaded", "hidden", "focused", "engaged", "maximized", "tabStop", "bodyKind", "still", "note", "pointer", "connectable", "links"];
function sameData(a, b) {
  return !!a && !!b && DATA_KEYS.every((k) => a[k] === b[k]);
}

function Flow({ canvasId, models, loaded, bodies, hidden, focusedId, engaged, maximizedId, tabStopId, loader, handlers, links, edgeRows, onConnect, onRemoveEdge, onLayout, onGestureStart, onGestureStop, onReady }) {
  const rf = useReactFlow();
  const rootRef = useRef(null);
  const [nodes, setNodes] = useState([]);
  // The zoom lives in a ref — a pan or a zoom must not re-render a body —
  // and only what it *decides* is state: whether a pointer may reach a pane,
  // and the readout that tells the viewer why.
  const zoomRef = useRef(1);
  const [pointer, setPointer] = useState(true);
  const [zoomPct, setZoomPct] = useState(100);
  // The plane's texture is a preference, not plane state: Preferences
  // writes it and announces, every open plane re-reads. No reload, and no
  // prop to thread through the surface for something the surface does not
  // own. The *colour* is CSS (canvas.css) keyed off data-bg, so a theme
  // switch re-tints without passing through React at all.
  const [bgPattern, setBgPattern] = useState(readCanvasPattern);
  useEffect(() => {
    function onPattern() { setBgPattern(readCanvasPattern()); }
    window.addEventListener(CANVAS_PATTERN_EVENT, onPattern);
    return () => window.removeEventListener(CANVAS_PATTERN_EVENT, onPattern);
  }, []);
  // A still is memory-only (stills.js), so this counter is how a fresh
  // capture reaches the nodes that show one.
  const [stillTick, setStillTick] = useState(0);
  // Which edge the viewer has selected: the line's own reason shows there,
  // and Delete removes it. Only one, because a link is removed one decision
  // at a time — a marquee that revokes six grants at once is not a thing
  // this surface offers.
  const [selectedEdge, setSelectedEdge] = useState("");
  // Which edge the pointer is on. A live link shows its chip only while it is
  // asked for, and the chip sits off the line, so the hover has to survive
  // the pointer crossing the gap to reach it.
  const [hoverEdge, setHoverEdge] = useState("");
  const hoverTimer = useRef(0);
  const modelsRef = useRef(models);
  modelsRef.current = models;
  const bodiesRef = useRef(bodies);
  bodiesRef.current = bodies;
  const saveTimer = useRef(0);
  const fitted = useRef(false);
  const stored = useMemo(() => readView(canvasId), [canvasId]);

  // ---- the camera --------------------------------------------------------
  const applyZoom = useCallback((zoom) => {
    zoomRef.current = zoom;
    // One CSS variable carries the zoom to the name-plates, which size
    // themselves by 1 / zoom so they stay readable however far out you are.
    if (rootRef.current) rootRef.current.style.setProperty("--cv-zoom", String(zoom));
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

  const fit = useCallback(() => { rf.fitView({ ...fitOpts(rootRef.current), duration: MOTION_MS }); }, [rf]);
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
    if (el) el.style.setProperty("--cv-zoom", String(zoomRef.current));
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
        connectable: EDGE_KINDS.includes(m.kind),
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
        // The header's link chip. The plane draws the lines, but a line can
        // be off the camera, run entirely behind the two panels it joins, or
        // be a few pixels long when zoomed out — and the maximize layer has
        // no line at all. The chip is what a panel carrying links says for
        // itself, wherever the camera is (ADR-0116).
        links: links[m.id] || null,
      });
    }
    return out;
  }, [models, loaded, bodies, hidden, focusedId, engaged, maximizedId, tabStopId, pointer, stillTick, links]);

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
          dragHandle: data.bodyKind === "plate" ? undefined : ".cv-head",
          position,
          width,
          height,
          data: prev && sameData(prev.data, data) ? prev.data : { ...data, ...nodeCallbacks(id) },
        });
      }
      return changed ? next : cur;
    });
  }, [nodeData, nodeCallbacks]);

  // ---- edges (ADR-0116) ---------------------------------------------------
  // The surface hands rows that already know whether they grant: this host
  // draws, it does not judge. `canvasGrants.js` is the one place that joins
  // a canvas's edges with the connection list, so the plane and the Messages
  // audit list can never disagree about a live grant.
  const removeEdge = useCallback((id) => { onRemoveEdge(id); }, [onRemoveEdge]);
  const hoverEdgeAt = useCallback((id, on) => {
    if (hoverTimer.current) { clearTimeout(hoverTimer.current); hoverTimer.current = 0; }
    if (on) { setHoverEdge(id); return; }
    // A grace period, not a debounce: the pointer leaving the line on its way
    // to the chip must not take the chip with it.
    hoverTimer.current = setTimeout(() => { hoverTimer.current = 0; setHoverEdge((cur) => (cur === id ? "" : cur)); }, HOVER_GRACE_MS);
  }, []);
  useEffect(() => () => { if (hoverTimer.current) clearTimeout(hoverTimer.current); }, []);
  const onEdgeMouseEnter = useCallback((_e, edge) => { hoverEdgeAt(edge.id, true); }, [hoverEdgeAt]);
  const onEdgeMouseLeave = useCallback((_e, edge) => { hoverEdgeAt(edge.id, false); }, [hoverEdgeAt]);
  const rfEdges = useMemo(() => (edgeRows || []).map((r) => ({
    id: r.id,
    source: r.source,
    target: r.target,
    sourceHandle: LINK_HANDLE,
    targetHandle: LINK_HANDLE,
    type: EDGE_TYPE,
    selected: r.id === selectedEdge,
    data: { grants: r.grants, reason: r.reason, label: r.label, hovered: r.id === hoverEdge, onRemove: removeEdge, onHover: hoverEdgeAt },
  })), [edgeRows, selectedEdge, hoverEdge, removeEdge, hoverEdgeAt]);
  // Selection is the library's (a click on the line, a click on the pane to
  // drop it); only that change is kept, because every other edge change here
  // would be a write the store has not agreed to.
  const onEdgesChange = useCallback((changes) => {
    for (const c of changes) {
      if (c.type !== "select") continue;
      setSelectedEdge((cur) => (c.selected ? c.id : cur === c.id ? "" : cur));
    }
  }, []);
  // Drawing the same pair twice, or a panel to itself, is refused while the
  // pointer is still down — the drop reads invalid instead of arriving as a
  // 409 the viewer has to read.
  const linked = useMemo(() => new Set((edgeRows || []).map((r) => [r.source, r.target].sort().join("|"))), [edgeRows]);
  const isValidConnection = useCallback(
    (c) => !!c && !!c.source && !!c.target && c.source !== c.target && !linked.has([c.source, c.target].sort().join("|")),
    [linked],
  );
  // Delete on a selected edge. React Flow's own delete key is off (Delete on
  // a focused panel removes the panel), and the line is not a focusable DOM
  // node, so the key is read from the document — but only while an edge is
  // selected, never from inside a pane, a field or a dialog.
  useEffect(() => {
    if (!selectedEdge || hidden) return undefined;
    const onKey = (e) => {
      if (e.key !== "Delete" && e.key !== "Backspace") return;
      if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return;
      const t = e.target;
      if (t && t.closest && t.closest("input, textarea, select, [contenteditable=\"true\"], .xterm, [data-cv-panel], [role=\"dialog\"]")) return;
      e.preventDefault();
      onRemoveEdge(selectedEdge);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [selectedEdge, hidden, onRemoveEdge]);

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
  // The zoom belongs to this host, and the loader outlives it: unmounting
  // the plane hands the loader back zoom 1, where every loaded body is live.
  // Without it a surface that loses its plane while zoomed out keeps drawing
  // name-plates — and since phase 4 keeps its chat sockets closed with them.
  useEffect(() => () => loader.setZoom(CANVAS_ZOOM.exact), [loader]);
  // Fit the plane the first time this viewer opens this canvas. The `fitView`
  // prop cannot do it: the first render has no nodes yet.
  useEffect(() => {
    if (fitted.current || stored || !nodes.length) return;
    fitted.current = true;
    rf.fitView(fitOpts(rootRef.current));
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
    saveTimer.current = setTimeout(() => { saveTimer.current = 0; writeView(canvasId, viewport); }, VIEW_SAVE_MS);
  }, [applyZoom, loader, canvasId, fillStills]);
  useEffect(() => () => { if (saveTimer.current) clearTimeout(saveTimer.current); }, []);

  // ---- what the surface's keyboard drives ---------------------------------
  useEffect(() => {
    if (!onReady) return undefined;
    onReady({ snapToOne, reveal, fit, zoomIn, zoomOut, zoom: () => zoomRef.current });
    return () => onReady(null);
  }, [onReady, snapToOne, reveal, fit, zoomIn, zoomOut]);

  return (
    <div className="cv-canvas-flow" ref={setRoot} data-bg={bgPattern}>
      {/* The zoom cluster floats bottom-right beside the minimap, but it is
          declared before the plane so Tab runs through the chrome and only
          then reaches the roving panel — the same order the surface's
          top-left cluster keeps. Both clusters are positioned, so the DOM
          order costs the layout nothing. */}
      <div className="cv-cluster cv-zoom" role="group" aria-label="Zoom" data-align-row>
        <button type="button" className="cv-zoom-btn" aria-label="Zoom out" title="Zoom out (−)" onClick={zoomOut}>−</button>
        <button
          type="button"
          className="cv-zoom-pct"
          title="Back to 100 % — the only zoom where a click lands on the cell it points at"
          onClick={() => snapToOne(focusedId)}
        >
          {zoomPct} %
        </button>
        <button type="button" className="cv-zoom-btn" aria-label="Zoom in" title="Zoom in (+)" onClick={zoomIn}>+</button>
        <button type="button" className="cv-zoom-fit" title="Fit every panel (0)" onClick={fit}>Fit</button>
      </div>
      <ReactFlow
        nodes={nodes}
        nodeTypes={NODE_TYPES}
        edges={rfEdges}
        edgeTypes={EDGE_TYPES}
        onEdgesChange={onEdgesChange}
        onEdgeMouseEnter={onEdgeMouseEnter}
        onEdgeMouseLeave={onEdgeMouseLeave}
        onConnect={onConnect}
        isValidConnection={isValidConnection}
        connectionMode={ConnectionMode.Loose}
        connectionLineType={ConnectionLineType.SimpleBezier}
        connectionRadius={CONNECT_RADIUS}
        edgesFocusable={false}
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
        {BG_VARIANT[bgPattern] ? (
          // Dots is 2, not 1: a single-pixel dot every four cells is the
          // sparsest of the three, and at --canvas-pattern's ~2:1 it read
          // as nothing on the light plane while the grid and the cross
          // read fine. The **gap** is the thing that must not move — it is
          // four 8 px cells and panels snap to it — so the mark grows
          // instead. Cross keeps 6: its tick is already 6 px of line.
          <Background variant={BG_VARIANT[bgPattern]} gap={BG_GAP} size={bgPattern === "dots" ? 2 : 6} lineWidth={1} />
        ) : null}
        <MiniMap pannable zoomable ariaLabel="Panels on the plane" />
      </ReactFlow>
    </div>
  );
}

export default function Plane(props) {
  return (
    <ReactFlowProvider>
      <Flow {...props} />
    </ReactFlowProvider>
  );
}
