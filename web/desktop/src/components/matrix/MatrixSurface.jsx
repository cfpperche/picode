import { Suspense, lazy, useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { applyTui, touches } from "@picode/shared/domain/feedReducers.js";
import { displayAgentName, locate } from "@picode/shared/domain/tree.js";
import { terminalActivityStamp, terminalCli, terminalCliLabel, terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { agentRowStatus, agentStatusLabel } from "@picode/shared/domain/agentStatus.js";
import { MATRIX_MODES, applyMatrixEvent, bindingState, canvasToGrid, gridToCanvas, layoutDiff, neighborPanel, nextSlot, normalizeMatrixDetail, normalizeMatrixList, panelDefault, panelOrder, tidyCanvas } from "@picode/shared/domain/matrix.js";
import { paneLeaveKey } from "@picode/shared/domain/termKeys.js";
import AppIcon from "../AppIcon.jsx";
import { IconEllipsis, IconGrid, IconPencil, IconPlus, IconTrash } from "../Icons.jsx";
import { notify, toast, toastError } from "../../lib/toast.js";
import { askConfirm } from "../../lib/confirm.js";
import MatrixGrid, { compactPanels } from "./MatrixGrid.jsx";
import { PanelHead } from "./Panel.jsx";
import PanelBody from "./PanelBody.jsx";
import PanelPicker from "./PanelPicker.jsx";
import NameDialog from "./NameDialog.jsx";
import { ChunkLoader, GRID_MARGIN } from "./chunkLoader.js";
import { forgetPane, ownedByTab } from "./paneOwnership.js";
import "../../styles/matrix.css";

// Canvas mode is lazy-imported: React Flow costs the desktop's main chunk
// +63.3 KB gzip eager against 205 B split (C0), and only a viewer who opens
// a canvas fetches it.
const MatrixCanvas = lazy(() => import("./MatrixCanvas.jsx"));

// MatrixSurface — the Matrix app's native surface (ADR-0109; plan
// docs/plans/matrix-app.md; API docs/architecture/matrix.md). One tab,
// `x:matrix` / #/app/matrix/<matrixId>: a header with the matrix switcher,
// New matrix, Add panel and a menu (Rename / Delete); under it the scroll
// container chunk loading observes, holding one react-grid-layout of
// panel wrappers. State is { list, byId } reduced by applyMatrixEvent from
// the change feed; the surface reads /api/matrices once on open and on a
// reveal older than 10 s, never on a timer. Layout edits are optimistic
// and saved as the changed subset, debounced 500 ms under ifUpdatedAt; a
// 409 refetches and says so. "Last matrix opened" is the one thing kept
// per browser (localStorage). The app reads the desktop only through
// `host` (fleet, openTabs, openTab, openInteractive, revealAgent,
// openFileTab, feed) — never App state.
//
// Keyboard (plan §4.6 "Focus"): one focused panel per matrix (focusedId)
// and one bit saying whether the keyboard is inside its terminal
// (engaged). Only the focused, engaged panel passes autoFocus down, so
// exactly one xterm ever calls term.focus(). Arrows move focus between
// wrappers (neighborPanel) and scroll the target into view, which loads
// it; Enter engages; Shift+Esc (paneLeaveKey) comes back to the chrome;
// Delete removes with an Undo toast; Home/End jump. Maximize is
// host-level state: the maximized panel's body mounts in a layer over the
// grid (the pane follows the visible host; the wrapper shows a stand-in
// line), the layout is untouched, Esc on any chrome restores.

const LAST_KEY = "picode-matrix-last";
const MODE_LABEL = { grid: "Grid", canvas: "Canvas" };
const REVEAL_STALE_MS = 10000;
const SAVE_DEBOUNCE_MS = 500;
const EMPTY = { list: [], byId: {} };
const EMPTY_FLEET = { workspaces: [], freeAgents: [], terminals: [] };
const NO_PANELS = [];
const enc = encodeURIComponent;
const json = (method, body) => ({ method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
const msgOf = (e) => (e && e.message ? e.message : String(e));
const readLast = () => { try { return localStorage.getItem(LAST_KEY) || ""; } catch { return ""; } };
const writeLast = (id) => { try { if (id) localStorage.setItem(LAST_KEY, id); else localStorage.removeItem(LAST_KEY); } catch { /* storage off */ } };
const rectOf = (p) => ({ x: p.x, y: p.y, w: p.w, h: p.h });
const rowOf = (p) => ({ id: p.id, ...rectOf(p) });
const NO_BODIES = {};

function patchPanels(state, id, fn) {
  const det = state.byId[id];
  if (!det) return state;
  return { ...state, byId: { ...state.byId, [id]: { ...det, panels: fn(det.panels) } } };
}

// settle: the feed's matrix.panel.added often beats the POST's own answer;
// the optimistic wrapper of that binding leaves at once, so the two never
// share a slot for a frame (the compactor would push the real one down).
function settle(state, ev) {
  const d = ev && ev.data;
  if (ev.type !== "matrix.panel.added" || !d || !d.panel) return state;
  return patchPanels(state, d.id, (ps) => (ps.some((p) => p.pending) ? ps.filter((p) => !(p.pending && p.kind === d.panel.kind && p.ref === d.panel.ref)) : ps));
}

const MODEL_KEYS = ["id", "kind", "ref", "x", "y", "w", "h", "pending", "state", "target", "name", "hint", "cwd", "status", "label", "stamp", "owned"];
function sameModel(a, b) {
  return !!a && MODEL_KEYS.every((k) => a[k] === b[k]);
}

// buildModel: everything a Panel renders, derived once per change from the
// panel row, the fleet, the tmux working list and the open tabs; the
// previous object is returned unchanged when nothing differs, so the
// memoized wrappers stay put.
function buildModel(panel, fleet, workingIds, openTabs, prev) {
  const state = bindingState(panel, fleet);
  let target = null;
  let name = panel.ref;
  let hint = "";
  let cwd = "";
  let status = "stopped";
  let label = "Gone";
  let stamp = "";
  if (panel.kind === "terminal") {
    target = (fleet.terminals || []).find((t) => t && t.id === panel.ref) || null;
    if (target) {
      const cli = terminalCli(target);
      name = target.name || "Terminal";
      hint = cli ? terminalCliLabel(cli) : "Shell";
      cwd = target.cwd || "";
      status = terminalStatus(target);
      label = terminalStatusLabel(target);
      stamp = terminalActivityStamp(target);
    }
  } else {
    const loc = locate(fleet.workspaces, fleet.freeAgents, panel.ref);
    target = loc && loc.agent && loc.agent.id === panel.ref ? loc.agent : null;
    if (target) {
      name = displayAgentName(target, loc.workspace);
      hint = loc.workspace ? loc.workspace.name : "Free agent";
      cwd = target.workPath || (loc.workspace && loc.workspace.path) || "";
      status = agentRowStatus(target, { workingIds });
      label = agentStatusLabel(status);
      stamp = target.lastStatusAt || target.lastStartedAt || "";
    }
  }
  // A deleted target has no row left to read: keep the words the panel
  // showed a moment ago, so the gone row reads "grid10 · Shell — That
  // terminal is gone." and not its id. A page opened after the deletion
  // has nothing to remember and falls back to the ref.
  if (!target && prev && prev.name && prev.name !== panel.ref) {
    name = prev.name;
    hint = prev.hint;
  }
  const next = {
    id: panel.id, kind: panel.kind, ref: panel.ref, x: panel.x, y: panel.y, w: panel.w, h: panel.h, pending: !!panel.pending,
    state, target, name, hint, cwd, status, label, stamp, owned: ownedByTab(panel.kind, panel.ref, openTabs),
  };
  return sameModel(prev, next) ? prev : next;
}

export default function MatrixSurface({ manifest, hidden, onClose, host, initialPath, onPathChange }) {
  const title = (manifest && manifest.name) || "Matrix";
  const fleet = (host && host.fleet) || EMPTY_FLEET;
  // A host that does not say is taken at its word (the fleet it gave is the
  // fleet there is); the desktop says `loaded` and it is false until boot ends.
  const fleetLoaded = fleet.loaded !== false;
  const openTabs = (host && host.openTabs) || NO_PANELS;
  const hostRef = useRef(host);
  hostRef.current = host;
  const pathRef = useRef(onPathChange);
  pathRef.current = onPathChange;

  const [store, setStore] = useState(EMPTY);
  const storeRef = useRef(store);
  storeRef.current = store;
  const [listLoaded, setListLoaded] = useState(false);
  const [listError, setListError] = useState("");
  const [currentId, setCurrentId] = useState(() => initialPath || readLast());
  const currentRef = useRef(currentId);
  currentRef.current = currentId;
  const [detailError, setDetailError] = useState("");
  const [loadedIds, setLoadedIds] = useState(() => new Set());
  // Canvas mode: what the zoom says each loaded body renders (live · still ·
  // plate · off — matrix-canvas.md §4.3). Grid mode gets "live" for every
  // loaded panel and never reads this.
  const [bodyKinds, setBodyKinds] = useState(NO_BODIES);
  // The canvas host's handle: snap to 1, pan a panel into view, zoom, fit.
  // Null in grid mode, and while the lazy chunk is still on the wire.
  const canvasRef = useRef(null);
  const modeRef = useRef("grid");
  const [modeBusy, setModeBusy] = useState(false);
  const [focusedId, setFocusedId] = useState("");
  const [engaged, setEngaged] = useState(false);
  const [maximizedId, setMaximizedId] = useState("");
  const focusedRef = useRef("");
  focusedRef.current = focusedId;
  const maximizedRef = useRef("");
  maximizedRef.current = maximizedId;
  const panelsRef = useRef(NO_PANELS);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [nameDialog, setNameDialog] = useState(""); // "" | "new" | "rename"
  const [workingIds, setWorkingIds] = useState([]);
  const hiddenRef = useRef(hidden);
  hiddenRef.current = hidden;
  const gestureRef = useRef(false);
  const queuedRef = useRef([]);
  const pendingRef = useRef({ matrixId: "", rows: new Map() });
  const saveTimer = useRef(0);
  const listAt = useRef(0);
  const detailAt = useRef({});
  const bodyRef = useRef(null);
  const rootRef = useRef(null);
  const maxRef = useRef(null);

  // ---- chunk loading -----------------------------------------------------
  const onChunk = useCallback((ids, bodies) => {
    setLoadedIds(ids);
    setBodyKinds(bodies);
  }, []);
  const loader = useMemo(() => new ChunkLoader(onChunk), [onChunk]);
  useEffect(() => () => loader.dispose(), [loader]);
  // The observer's root is the grid's scroll container. Canvas mode roots it
  // on the plane instead, with a margin on both axes (MatrixCanvas), so the
  // surface only remembers the element here.
  const setBody = useCallback((el) => { bodyRef.current = el; }, []);
  const setGridBody = useCallback((el) => { bodyRef.current = el; loader.attach(el, GRID_MARGIN); }, [loader]);
  const onCanvasReady = useCallback((handle) => { canvasRef.current = handle; }, []);
  useEffect(() => { loader.setHidden(!!hidden); }, [hidden, loader]);
  useEffect(() => {
    if (!focusedId) return undefined;
    loader.pin(focusedId, true);
    return () => loader.pin(focusedId, false);
  }, [focusedId, loader]);
  useEffect(() => {
    if (!maximizedId) return undefined;
    loader.pin(maximizedId, true);
    return () => loader.pin(maximizedId, false);
  }, [maximizedId, loader]);

  // ---- reads: once on open, once on a stale reveal, then the feed ---------
  const loadList = useCallback(async () => {
    try {
      const list = normalizeMatrixList(await api("/api/matrices"));
      listAt.current = Date.now();
      setStore((s) => ({ ...s, list }));
      setListError("");
    } catch (e) {
      setListError(humanizeError(msgOf(e)));
    } finally {
      setListLoaded(true);
    }
  }, []);
  const loadDetail = useCallback(async (id) => {
    if (!id) return;
    try {
      const det = normalizeMatrixDetail(await api("/api/matrices/" + enc(id)));
      if (!det) throw new Error("That matrix could not be read.");
      detailAt.current[id] = Date.now();
      if (pendingRef.current.matrixId === id) pendingRef.current = { matrixId: "", rows: new Map() };
      setStore((s) => {
        const next = applyMatrixEvent(s, { type: "matrix.updated", data: det.matrix });
        return { ...next, byId: { ...next.byId, [id]: det } };
      });
      setDetailError("");
    } catch (e) {
      if (e && e.status === 404) {
        setStore((s) => applyMatrixEvent(s, { type: "matrix.deleted", data: { id } }));
        return;
      }
      setDetailError(humanizeError(msgOf(e)));
    }
  }, []);
  useEffect(() => { loadList(); }, [loadList]);
  const feed = host && host.feed;
  useEffect(() => {
    if (typeof feed !== "function") return undefined;
    return feed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") {
        if (ev.data && ev.data.first) return;
        loadList();
        if (currentRef.current) loadDetail(currentRef.current);
        return;
      }
      if (ev.type === "agent.tui") { setWorkingIds((cur) => applyTui(cur, ev)); return; }
      // A deleted target holds no pane: dispose what the Matrix kept
      // suspended now, not when the LRU gets to it (paneOwnership.js).
      if (ev.type === "terminal.deleted" || ev.type === "agent.deleted") {
        const gone = ev.data && ev.data.id;
        const kind = ev.type === "terminal.deleted" ? "terminal" : "agent";
        if (gone) forgetPane(gone, ownedByTab(kind, gone, (hostRef.current && hostRef.current.openTabs) || NO_PANELS));
        return;
      }
      if (!touches(ev, ["matrix"])) return;
      // Another client's rows arrive after the gesture ends (plan §4.7).
      if (gestureRef.current) { queuedRef.current.push(ev); return; }
      setStore((s) => settle(applyMatrixEvent(s, ev), ev));
    });
  }, [feed, loadList, loadDetail]);
  useEffect(() => {
    if (hidden) return;
    const now = Date.now();
    if (listAt.current && now - listAt.current > REVEAL_STALE_MS) loadList();
    const id = currentRef.current;
    if (id && detailAt.current[id] && now - detailAt.current[id] > REVEAL_STALE_MS) loadDetail(id);
  }, [hidden, loadList, loadDetail]);

  // ---- which matrix -------------------------------------------------------
  const flushRef = useRef(() => {});
  const select = useCallback((id) => {
    flushRef.current();
    currentRef.current = id;
    setCurrentId(id);
    setFocusedId("");
    setEngaged(false);
    setMaximizedId("");
    setDetailError("");
    writeLast(id);
    if (pathRef.current) pathRef.current(id);
  }, []);
  useEffect(() => {
    if (initialPath && initialPath !== currentRef.current) select(initialPath);
  }, [initialPath, select]);
  // Revealed, the tab's hash is #/app/matrix again: say which matrix.
  useEffect(() => {
    if (!hidden && currentRef.current && pathRef.current) pathRef.current(currentRef.current);
  }, [hidden]);
  useEffect(() => {
    if (!listLoaded) return;
    const list = store.list;
    if (currentId && list.some((m) => m.id === currentId)) return;
    const last = readLast();
    const pick = list.some((m) => m.id === last) ? last : list.length ? list[0].id : "";
    if (pick !== currentId) select(pick);
  }, [listLoaded, store.list, currentId, select]);
  useEffect(() => {
    if (currentId && !storeRef.current.byId[currentId]) loadDetail(currentId);
  }, [currentId, loadDetail]);
  const current = store.list.find((m) => m.id === currentId) || null;
  const detail = store.byId[currentId] || null;
  const panels = detail ? detail.panels : NO_PANELS;
  panelsRef.current = panels;
  // The layout mode decides what x/y/w/h mean, which host draws them and
  // which rules every save is judged by (ADR-0113).
  const mode = current ? current.mode : "grid";
  modeRef.current = mode;

  // The tmux-reported Working state of the agents on this matrix: one read
  // on open and reveal, then agent.tui from the feed (the sidebar's rule).
  const agentRefs = useMemo(() => panels.filter((p) => p.kind === "agent").map((p) => p.ref).sort().join(","), [panels]);
  useEffect(() => {
    if (hidden || !agentRefs) return undefined;
    let stop = false;
    api("/api/tui-working?ids=" + enc(agentRefs)).then((d) => { if (!stop) setWorkingIds((d && d.working) || []); }).catch(() => { /* the feed catches up */ });
    return () => { stop = true; };
  }, [agentRefs, hidden]);

  // ---- panel models -------------------------------------------------------
  const modelsRef = useRef(new Map());
  const models = useMemo(() => {
    const prev = modelsRef.current;
    const next = new Map();
    const out = panels.map((p) => {
      const m = buildModel(p, fleet, workingIds, openTabs, prev.get(p.id));
      next.set(p.id, m);
      return m;
    });
    modelsRef.current = next;
    return out;
  }, [panels, fleet.workspaces, fleet.freeAgents, fleet.terminals, workingIds, openTabs]);
  useEffect(() => {
    if (focusedId && !models.some((m) => m.id === focusedId)) setFocusedId("");
    if (maximizedId && !models.some((m) => m.id === maximizedId)) setMaximizedId("");
  }, [models, focusedId, maximizedId]);

  // ---- saving the layout (plan §4.7) --------------------------------------
  // Answers the `updatedAt` it left behind ("" when there was nothing to
  // send, false when it refused and reloaded), so a mutation that has to
  // follow it does not send a precondition the save has just invalidated.
  const flushLayout = useCallback(async () => {
    if (saveTimer.current) { clearTimeout(saveTimer.current); saveTimer.current = 0; }
    const { matrixId, rows } = pendingRef.current;
    pendingRef.current = { matrixId: "", rows: new Map() };
    const det = storeRef.current.byId[matrixId];
    if (!det || !rows.size) return "";
    try {
      const res = await api("/api/matrices/" + enc(matrixId) + "/layout", json("PATCH", { ifUpdatedAt: det.matrix.updatedAt, panels: [...rows.values()] }));
      setStore((s) => applyMatrixEvent(s, { type: "matrix.layout", data: res }));
      return (res && res.updatedAt) || "";
    } catch (e) {
      if (e && e.status === 409) toast.info("Matrix changed elsewhere — reloaded.");
      else toastError(e);
      await loadDetail(matrixId);
      return false;
    }
  }, [loadDetail]);
  flushRef.current = flushLayout;
  useEffect(() => { if (hidden) flushRef.current(); }, [hidden]);
  useEffect(() => () => { flushRef.current(); }, []);
  // applyLayout(next): the rows as the grid draws them — from a drag or
  // resize (onLayoutChange) or from compacting the store (below). The
  // local rows move at once; the changed subset waits for the debounce.
  const applyLayout = useCallback((next) => {
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    if (!det) return;
    const diff = layoutDiff(det.panels.filter((p) => !p.pending), next, modeRef.current);
    const to = new Map(next.map((p) => [p.id, p]));
    const moved = det.panels.some((p) => to.has(p.id) && (to.get(p.id).x !== p.x || to.get(p.id).y !== p.y || to.get(p.id).w !== p.w || to.get(p.id).h !== p.h));
    if (moved) setStore((s) => patchPanels(s, id, (ps) => ps.map((p) => (to.has(p.id) ? { ...p, ...rectOf(to.get(p.id)) } : p))));
    if (!diff.length) return;
    if (pendingRef.current.matrixId && pendingRef.current.matrixId !== id) flushRef.current();
    if (pendingRef.current.matrixId !== id) pendingRef.current = { matrixId: id, rows: new Map() };
    for (const p of diff) pendingRef.current.rows.set(p.id, p);
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => { saveTimer.current = 0; flushRef.current(); }, SAVE_DEBOUNCE_MS);
  }, []);
  const onLayoutChange = useCallback((layout, width) => {
    if (hiddenRef.current || !(width > 0) || gestureRef.current) return;
    applyLayout(layout.map((l) => ({ id: l.i, x: l.x, y: l.y, w: l.w, h: l.h })));
  }, [applyLayout]);
  // The store stays compacted: what the grid draws is what gets saved,
  // including the compaction after a remove and a matrix stored with gaps.
  // Canvas mode has no automatic compaction — free placement is the point —
  // so the pack is the explicit **Tidy** action instead (plan §3, §4.4).
  useEffect(() => {
    if (hidden || gestureRef.current || modeBusy || !panels.length || mode !== "grid") return;
    applyLayout(compactPanels(panels));
  }, [panels, hidden, mode, modeBusy, applyLayout]);
  const onGestureStart = useCallback((id) => {
    gestureRef.current = true;
    loader.pin(id, true);
  }, [loader]);
  const onGestureStop = useCallback((id) => {
    gestureRef.current = false;
    loader.pin(id, false);
    const q = queuedRef.current;
    queuedRef.current = [];
    if (q.length) setStore((s) => q.reduce((acc, ev) => settle(applyMatrixEvent(acc, ev), ev), s));
  }, [loader]);

  // ---- mutations ----------------------------------------------------------
  async function createMatrix(name) {
    const res = await api("/api/matrices", json("POST", { name }));
    setStore((s) => applyMatrixEvent(s, { type: "matrix.created", data: res }));
    select(res.id);
  }
  async function renameMatrix(name) {
    const id = currentRef.current;
    const m = storeRef.current.list.find((x) => x.id === id);
    if (!m) return;
    try {
      const res = await api("/api/matrices/" + enc(id), json("PATCH", { name, ifUpdatedAt: m.updatedAt }));
      setStore((s) => applyMatrixEvent(s, { type: "matrix.updated", data: res }));
    } catch (e) {
      if (e && e.status === 409) { await loadDetail(id); throw new Error("Matrix changed elsewhere — reloaded. Try again."); }
      throw e;
    }
  }
  // setMode(next): the Grid | Canvas switch (ADR-0113). The transform lives
  // twice on purpose — Go writes it, `matrix.js` previews it, with the same
  // fixtures — so the preview *is* what the answer will say: the panels move
  // at once and the 200 reconciles them through the one reducer path every
  // other mutation uses. Going back to the grid loses where a panel sat on
  // the plane, so that direction asks first; grid → canvas loses nothing and
  // does not.
  async function setMode(next) {
    const id = currentRef.current;
    const m = storeRef.current.list.find((x) => x.id === id);
    const det = storeRef.current.byId[id];
    if (!m || !det || !MATRIX_MODES.includes(next) || m.mode === next || modeBusy) return;
    if (next === "grid" && det.panels.length) {
      const ok = await askConfirm({
        title: "Switch to grid?",
        message: "Grid mode packs the " + det.panels.length + " panels into 12 columns; where they sit on the plane is not kept.",
        confirmLabel: "Switch to grid",
      });
      if (!ok) return;
    }
    setModeBusy(true);
    // The moves made under the old mode go first, and they carry the
    // precondition this switch has to use.
    const flushed = await flushRef.current();
    if (flushed === false) { setModeBusy(false); return; }
    const moved = (next === "canvas" ? gridToCanvas(det.panels) : canvasToGrid(det.panels)).map(rowOf);
    setStore((s) => applyMatrixEvent(s, { type: "matrix.mode", data: { ...m, mode: next, panels: moved } }));
    setMaximizedId("");
    try {
      const res = await api("/api/matrices/" + enc(id), json("PATCH", { mode: next, ifUpdatedAt: flushed || m.updatedAt }));
      setStore((s) => applyMatrixEvent(s, { type: "matrix.mode", data: res }));
    } catch (e) {
      if (e && e.status === 409) toast.info("Matrix changed elsewhere — reloaded.");
      else toastError(e);
      await loadDetail(id);
    } finally {
      setModeBusy(false);
    }
  }
  // Tidy: canvas mode's answer to react-grid-layout's automatic compaction —
  // explicit instead of silent. Reading order, sizes kept, saved once.
  function tidy() {
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    if (!det) return;
    const rows = det.panels.filter((p) => !p.pending);
    if (!rows.length) return;
    applyLayout(tidyCanvas(rows));
    flushRef.current();
  }
  async function deleteMatrix() {
    const id = currentRef.current;
    const m = storeRef.current.list.find((x) => x.id === id);
    if (!m) return;
    const ok = await askConfirm({
      title: "Delete " + m.name + "?",
      message: "The matrix and its panels go away. The agents and terminals on it are not touched.",
      confirmLabel: "Delete matrix",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/matrices/" + enc(id), { method: "DELETE" });
      setStore((s) => applyMatrixEvent(s, { type: "matrix.deleted", data: { id } }));
    } catch (e) {
      if (e && e.status === 404) setStore((s) => applyMatrixEvent(s, { type: "matrix.deleted", data: { id } }));
      else toastError(e);
    }
  }
  async function addPanel({ kind, ref }) {
    setPickerOpen(false);
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    if (!det) return;
    const size = panelDefault(modeRef.current);
    const { x, y } = nextSlot(det.panels, size.w, size.h, modeRef.current);
    const tmp = "tmp-" + Date.now().toString(36) + Math.random().toString(36).slice(2, 6);
    const body = { kind, ref, x, y, w: size.w, h: size.h };
    // The wrapper shows at once and settles on the server's answer.
    setStore((s) => patchPanels(s, id, (ps) => [...ps, { id: tmp, ...body, createdAt: "", pending: true }]));
    try {
      const res = await api("/api/matrices/" + enc(id) + "/panels", json("POST", body));
      setStore((s) => applyMatrixEvent(patchPanels(s, id, (ps) => ps.filter((p) => p.id !== tmp)), { type: "matrix.panel.added", data: res }));
      const added = res && res.panel && res.panel.id;
      if (added) {
        setFocusedId(added);
        requestAnimationFrame(() => {
          const el = bodyRef.current && bodyRef.current.querySelector('[data-mx-panel="' + added + '"]');
          if (el && el.scrollIntoView) el.scrollIntoView({ block: "nearest" });
        });
      }
    } catch (e) {
      setStore((s) => patchPanels(s, id, (ps) => ps.filter((p) => p.id !== tmp)));
      toastError(e);
    }
  }
  // Remove is reversible: the panel leaves at once and the toast's Undo
  // adds the same binding back at its old rectangle (the compactor settles
  // it if the hole closed meanwhile). Focus moves to the next panel in
  // reading order, else the previous, so a keyboard user never lands on
  // nothing.
  async function removePanel(model) {
    const id = currentRef.current;
    if (model.pending) return;
    const order = panelOrder(panelsRef.current);
    const at = order.indexOf(model.id);
    const next = order[at + 1] || order[at - 1] || "";
    const removed = { type: "matrix.panel.removed", data: { id, panelId: model.id } };
    try {
      await api("/api/matrices/" + enc(id) + "/panels/" + enc(model.id), { method: "DELETE" });
      setStore((s) => applyMatrixEvent(s, removed));
      notify({
        level: "info",
        title: "Removed " + model.name + " from the matrix.",
        actions: [{ label: "Undo", primary: true, run: () => restorePanel(id, model) }],
        key: "mx-removed:" + model.id,
      });
    } catch (e) {
      if (e && e.status === 404) setStore((s) => applyMatrixEvent(s, removed));
      else toastError(e);
    }
    if (focusedRef.current === model.id) focusPanel(next);
  }
  async function restorePanel(matrixId, model) {
    const body = { kind: model.kind, ref: model.ref, x: model.x, y: model.y, w: model.w, h: model.h };
    try {
      const res = await api("/api/matrices/" + enc(matrixId) + "/panels", json("POST", body));
      setStore((s) => applyMatrixEvent(s, { type: "matrix.panel.added", data: res }));
      if (currentRef.current === matrixId && res && res.panel) focusPanel(res.panel.id);
    } catch (e) {
      toastError(e);
    }
  }

  // ---- focus (plan §4.6) --------------------------------------------------
  // focusPanel(id): the wrapper takes DOM focus and scrolls into view (which
  // loads it); the keyboard stays on the chrome until Enter.
  function focusPanel(id) {
    setFocusedId(id || "");
    setEngaged(false);
    if (!id) return;
    // A canvas has nothing to scroll: an off-screen panel is panned into
    // view instead, which is what loads it.
    const canvas = modeRef.current === "canvas" ? canvasRef.current : null;
    if (canvas) canvas.reveal(id);
    requestAnimationFrame(() => {
      const el = bodyRef.current && bodyRef.current.querySelector('[data-mx-panel="' + id + '"]');
      if (!el) return;
      if (!canvas && el.scrollIntoView) el.scrollIntoView({ block: "nearest" });
      el.focus({ preventScroll: true });
    });
  }
  // engagePanel: the pane takes the keyboard. On a canvas it takes the
  // pointer too, and a pointer is only honest at zoom 1.0 (C0), so the plane
  // snaps there first — "snap to 1 on engage" (plan §4.3).
  function engagePanel(id) {
    if (modeRef.current === "canvas" && canvasRef.current) canvasRef.current.snapToOne(id);
    setFocusedId(id);
    setEngaged(true);
  }
  function restoreMaximized() {
    const id = maximizedRef.current;
    if (!id) return;
    setMaximizedId("");
    focusPanel(id);
  }

  // ---- panel actions (stable for the memoized wrappers) -------------------
  const handlers = useMemo(() => ({
    // onFocus(model, body): a pointer on the wrapper — inside the body the
    // xterm takes the click, so the keyboard is in the pane; on the chrome
    // the wrapper itself takes the focus.
    onFocus: (model, body) => {
      setFocusedId((cur) => (cur === model.id ? cur : model.id));
      setEngaged(!!body);
    },
    // onKey(e, model): the wrapper's keydown. From inside the pane only the
    // leave chord is ours (xterm passes it through); every other key there
    // is the shell's. On the chrome itself: the model above. Buttons in the
    // header or the body keep their own keys.
    onKey: (e, model) => {
      const el = e.currentTarget;
      const t = e.target;
      if (t !== el && t.closest && t.closest(".xterm")) {
        if (!paneLeaveKey(e)) return;
        e.preventDefault();
        setEngaged(false);
        el.focus();
        return;
      }
      if (t !== el) return;
      const dir = { ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down" }[e.key];
      if (dir) {
        e.preventDefault();
        const next = neighborPanel(panelsRef.current, model.id, dir);
        if (next) focusPanel(next);
        return;
      }
      if (e.key === "Home" || e.key === "End") {
        e.preventDefault();
        const order = panelOrder(panelsRef.current);
        const next = e.key === "Home" ? order[0] : order[order.length - 1];
        if (next && next !== model.id) focusPanel(next);
        return;
      }
      if (e.key === "Enter") {
        e.preventDefault();
        // A live pane takes the keyboard; a row with one action runs it.
        const action = el.querySelector(".mx-panel-body .mx-placeholder .btn");
        if (action && !el.querySelector(".mx-panel-body .xterm")) action.click();
        else engagePanel(model.id);
        return;
      }
      if (e.key === "Delete" || e.key === "Backspace") {
        e.preventDefault();
        removePanel(model);
        return;
      }
      if (e.key === "Escape" && maximizedRef.current) {
        e.preventDefault();
        restoreMaximized();
        return;
      }
      // The canvas's own keys, on the chrome only (inside a pane every key
      // but the leave chord is the shell's).
      const canvas = modeRef.current === "canvas" ? canvasRef.current : null;
      if (!canvas || e.ctrlKey || e.metaKey || e.altKey) return;
      if (e.key === "+" || e.key === "=") { e.preventDefault(); canvas.zoomIn(); return; }
      if (e.key === "-" || e.key === "_") { e.preventDefault(); canvas.zoomOut(); return; }
      if (e.key === "0") { e.preventDefault(); canvas.fit(); }
    },
    onOpen: (model) => {
      const h = hostRef.current || {};
      if (model.kind === "terminal") { if (h.openTab) h.openTab("t:" + model.ref); return; }
      if (model.state === "agent-interactive") { if (h.openInteractive) h.openInteractive(model.ref); return; }
      if (h.revealAgent) h.revealAgent(model.ref);
    },
    onRun: (model) => {
      // Run starts the TUI in place: the feed flips the mode and the body
      // swaps to the live pane without leaving the matrix.
      api("/api/agents/" + enc(model.ref) + "/open", { method: "POST" }).catch(toastError);
    },
    onRemove: (model) => { removePanel(model); },
    // Maximize takes the keyboard into the pane (the layer mounts its body
    // with autoFocus); Restore hands it back to the wrapper's chrome.
    onMaximize: (model) => {
      if (maximizedRef.current === model.id) { restoreMaximized(); return; }
      setMaximizedId(model.id);
      engagePanel(model.id);
    },
    onOpenFile: (model, path) => {
      const h = hostRef.current || {};
      if (h.openFileTab) h.openFileTab(model.kind === "agent" ? "agent" : "term", model.ref, path);
    },
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }), []);

  // ---- maximize (plan §4.6: the panel takes the surface, layout untouched)
  // Esc on any chrome restores — the layer's head, the surface header, the
  // wrapper — never from inside the pane (the TUI may need it; Shift+Esc
  // leaves the pane first) and never out of a dialog or menu, which own
  // their own Esc.
  useEffect(() => {
    if (!maximizedId || hidden) return undefined;
    const onKey = (e) => {
      if (e.key !== "Escape" || e.defaultPrevented) return;
      const t = e.target;
      if (t && t.closest && t.closest(".xterm, [role=\"dialog\"], [role=\"menu\"], [role=\"listbox\"]")) return;
      e.preventDefault();
      restoreMaximized();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [maximizedId, hidden]);
  // The layer's own keys: Shift+Esc from inside the pane focuses the
  // layer's chrome; Enter on it goes back in.
  function onLayerKey(e) {
    const el = e.currentTarget;
    const t = e.target;
    if (t !== el && t.closest && t.closest(".xterm")) {
      if (!paneLeaveKey(e)) return;
      e.preventDefault();
      setEngaged(false);
      el.focus();
      return;
    }
    if (t === el && e.key === "Enter") {
      e.preventDefault();
      engagePanel(maximizedRef.current);
    }
  }

  // ---- render -------------------------------------------------------------
  const onMatrix = useMemo(() => new Set(panels.map((p) => p.kind + ":" + p.ref)), [panels]);
  const rootClass = "app-surface native-surface mx-surface" + (maximizedId ? " is-max" : "");
  // The roving tab stop: the focused panel, else the first in reading order.
  const tabStopId = useMemo(() => (focusedId && panels.some((p) => p.id === focusedId) ? focusedId : panelOrder(panels)[0] || ""), [panels, focusedId]);
  const maxModel = maximizedId ? models.find((m) => m.id === maximizedId) || null : null;
  let body;
  if (listError && !store.list.length) {
    body = (
      <p className="ft-msg">
        {listError}{" "}
        <button type="button" className="btn btn-sm" onClick={loadList}>Try again</button>
      </p>
    );
  } else if (!listLoaded) {
    body = <div className="mx-skel" aria-busy="true"><span className="skel-line" /><span className="skel-line" /><span className="skel-line" /></div>;
  } else if (!store.list.length) {
    body = (
      <div className="app-blank">
        <AppIcon name="matrix" label={title} size={24} />
        <p className="app-blank-title">No matrix yet.</p>
        <p className="app-blank-sub">A matrix shows many agents and terminals side by side, live.</p>
        <button type="button" className="btn btn-sm btn-primary" onClick={() => setNameDialog("new")}><IconPlus size={13} /> New matrix</button>
      </div>
    );
  } else if (detailError && !detail) {
    body = (
      <p className="ft-msg">
        {detailError}{" "}
        <button type="button" className="btn btn-sm" onClick={() => loadDetail(currentId)}>Try again</button>
      </p>
    );
  } else if (!detail) {
    body = <div className="mx-skel" aria-busy="true"><span className="skel-line" /><span className="skel-line" /><span className="skel-line" /></div>;
  } else if (!fleetLoaded) {
    // The fleet decides what every panel is bound to. Read before it lands,
    // an empty fleet reads as "all gone" — the skeleton says "not read yet"
    // instead of flashing an error row on every panel.
    body = <div className="mx-skel" aria-busy="true"><span className="skel-line" /><span className="skel-line" /><span className="skel-line" /></div>;
  } else if (!panels.length) {
    body = (
      <div className="app-blank">
        <AppIcon name="matrix" label={title} size={24} />
        <p className="app-blank-title">Add your first panel.</p>
        <p className="app-blank-sub">A panel is one agent or terminal, live on this matrix.</p>
        <button type="button" className="btn btn-sm btn-primary" onClick={() => setPickerOpen(true)}><IconPlus size={13} /> Add panel</button>
      </div>
    );
  } else {
    body = (
      <div className="mx-stage">
        <div className={"mx-body" + (mode === "canvas" ? " is-canvas" : "")} ref={mode === "canvas" ? setBody : setGridBody} inert={!!maximizedId}>
          {mode === "canvas" ? (
            <Suspense fallback={<div className="mx-skel" aria-busy="true"><span className="skel-line" /><span className="skel-line" /><span className="skel-line" /></div>}>
              <MatrixCanvas
                key={currentId}
                matrixId={currentId}
                models={models}
                loaded={loadedIds}
                bodies={bodyKinds}
                hidden={!!hidden}
                focusedId={focusedId}
                engaged={engaged}
                maximizedId={maximizedId}
                tabStopId={tabStopId}
                loader={loader}
                handlers={handlers}
                onLayout={applyLayout}
                onGestureStart={onGestureStart}
                onGestureStop={onGestureStop}
                onReady={onCanvasReady}
              />
            </Suspense>
          ) : (
            <MatrixGrid
              models={models}
              loaded={loadedIds}
              hidden={!!hidden}
              focusedId={focusedId}
              engaged={engaged}
              maximizedId={maximizedId}
              tabStopId={tabStopId}
              loader={loader}
              handlers={handlers}
              onLayoutChange={onLayoutChange}
              onGestureStart={onGestureStart}
              onGestureStop={onGestureStop}
            />
          )}
        </div>
        {maxModel ? (
          // The maximized panel's body, in a layer over the grid: the same
          // xterm (ShellTerm re-claims the pane), fitted to the layer.
          <div className="mx-max" role="group" aria-label={maxModel.name + " — maximized"} tabIndex={-1} ref={maxRef} onKeyDown={onLayerKey}>
            <PanelHead model={maxModel} loaded={loadedIds.has(maxModel.id)} maximized handlers={handlers} fixed />
            <div className="mx-panel-body">
              <PanelBody
                model={maxModel}
                loaded={loadedIds.has(maxModel.id)}
                hidden={!!hidden}
                focused={engaged && focusedId === maxModel.id}
                onOpen={() => handlers.onOpen(maxModel)}
                onRemove={() => handlers.onRemove(maxModel)}
                onRun={() => handlers.onRun(maxModel)}
                onOpenFile={(path) => handlers.onOpenFile(maxModel, path)}
              />
            </div>
          </div>
        ) : null}
      </div>
    );
  }
  return (
    <section className={rootClass} aria-label={title} hidden={!!hidden} ref={rootRef}>
      <header className="ft-head">
        <span className="app-head-icon"><AppIcon name={manifest ? manifest.icon : "matrix"} label={title} size={14} /></span>
        <h2 className="ft-title" title={title}>{title}</h2>
        <div className="app-head-left mx-head-left" data-align-row>
          {store.list.length ? (
            <select className="mx-select" aria-label="Matrix" value={currentId} onChange={(e) => select(e.target.value)}>
              {store.list.map((m) => <option key={m.id} value={m.id}>{m.name}</option>)}
            </select>
          ) : null}
          {current ? (
            <div className="termset-seg mx-seg" role="radiogroup" aria-label="Layout mode">
              {MATRIX_MODES.map((m) => (
                <label className="termset-seg-opt" key={m} htmlFor={"mx-mode-" + m}>
                  <input id={"mx-mode-" + m} type="radio" name="mx-mode" checked={mode === m} onChange={() => setMode(m)} />
                  <span className="termset-seg-face">{MODE_LABEL[m]}</span>
                </label>
              ))}
            </div>
          ) : null}
          {current ? (
            <button type="button" className="btn btn-sm" onClick={() => setPickerOpen(true)}><IconPlus size={13} /> Add panel</button>
          ) : null}
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => setNameDialog("new")} title="Create another matrix">New matrix</button>
          {current ? (
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button type="button" className="btn btn-sm btn-ghost mx-menu-btn" aria-label={"Actions for " + current.name} title="Rename or delete this matrix"><IconEllipsis size={15} /></button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content className="ws-row-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
                  {mode === "canvas" && panels.length ? (
                    <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => { tidy(); }}><IconGrid size={13} /> Tidy panels</DropdownMenu.Item>
                  ) : null}
                  <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => setNameDialog("rename")}><IconPencil size={13} /> Rename</DropdownMenu.Item>
                  <DropdownMenu.Item className="ws-row-menu-item danger" onSelect={() => { deleteMatrix(); }}><IconTrash size={13} /> Delete matrix</DropdownMenu.Item>
                </DropdownMenu.Content>
              </DropdownMenu.Portal>
            </DropdownMenu.Root>
          ) : null}
        </div>
        <span className="ft-spacer" />
        <div className="app-head-right" data-align-row>
          {onClose ? (
            <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>Close</button>
          ) : null}
        </div>
      </header>
      {body}
      <PanelPicker
        open={pickerOpen}
        fleet={fleet}
        onMatrix={onMatrix}
        workingIds={workingIds}
        onPick={addPanel}
        onClose={() => setPickerOpen(false)}
      />
      <NameDialog
        open={nameDialog === "new"}
        title="New matrix"
        action="Create"
        initial=""
        onSubmit={createMatrix}
        onClose={() => setNameDialog("")}
      />
      <NameDialog
        open={nameDialog === "rename"}
        title="Rename matrix"
        action="Rename"
        initial={current ? current.name : ""}
        onSubmit={renameMatrix}
        onClose={() => setNameDialog("")}
      />
    </section>
  );
}
