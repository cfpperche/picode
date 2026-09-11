import { Suspense, lazy, useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { applyTui, touches } from "@picode/shared/domain/feedReducers.js";
import { displayAgentName, locate } from "@picode/shared/domain/tree.js";
import { terminalActivityStamp, terminalCli, terminalCliLabel, terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { agentRowStatus, agentStatusLabel } from "@picode/shared/domain/agentStatus.js";
import { EDGE_KINDS, PANEL_DEFAULT_CANVAS, applyCanvasEvent, bindingState, buildRef, layoutDiff, neighborPanel, nextSlot, normalizeCanvasDetail, normalizeCanvasList, panelOrder, parseRef, refOwner, tidyCanvas, validateEdge } from "@picode/shared/domain/canvas.js";
import { crossFolderConfirm, edgeGrant, enrolOffer, linkChipTitle, linkCounts, peerIndex, removeConfirm } from "@picode/shared/domain/canvasGrants.js";
import { basename } from "@picode/shared/domain/diff.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import { paneLeaveKey } from "@picode/shared/domain/termKeys.js";
import AppIcon from "../AppIcon.jsx";
import { IconEllipsis, IconGrid, IconPencil, IconPlus, IconTrash } from "../Icons.jsx";
import { go, isFileTab, parseFileTab, pinHash } from "../../lib/routes.js";
import { notify, toast, toastError } from "../../lib/toast.js";
import { askConfirm } from "../../lib/confirm.js";
import { methodLabel } from "../../lib/needsYou.js";
import { FREE_WS } from "../../lib/termGroups.js";
import { PanelHead } from "./Panel.jsx";
import PanelBody from "./PanelBody.jsx";
import PanelPicker from "./PanelPicker.jsx";
import NameDialog from "./NameDialog.jsx";
import { ChunkLoader } from "./chunkLoader.js";
import { forgetPane, ownedByTab } from "./paneOwnership.js";
import { forgetDocument, heldDocumentDirty } from "../../lib/fileDocs.js";
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
// panel wrappers. State is { list, byId } reduced by applyCanvasEvent from
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
const NO_LINKS = {};
const NO_EDGES = [];
const OWNER_WORD = { term: "Terminal", workspace: "Folder", agent: "Agent" };
// `GET /api/communication` is what says whether an edge grants anything
// right now: its `connections[].active` is the store's own `peerCurrent`
// (ADR-0104). Read when a matrix holds an edge, refreshed on peer.* feed
// rows and on a stale reveal — never on a timer.
const NO_PEERS = { owners: [], connections: [] };
// The feed rows that can change what a link grants: the connection itself,
// and the owner whose recorded session `peerCurrent` compares against.
const PEER_EVENTS = /^(peer\.|agent\.(updated|deleted)|terminal\.(updated|deleted))/;
const PEERS_DEBOUNCE_MS = 400;

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
  if (ev.type !== "canvas.panel.added" || !d || !d.panel) return state;
  return patchPanels(state, d.id, (ps) => (ps.some((p) => p.pending) ? ps.filter((p) => !(p.pending && p.kind === d.panel.kind && p.ref === d.panel.ref)) : ps));
}

// ownerFolder(kind, row, fleet): the folder an owner works in — what a
// `git.updated` names when the fleet's watcher inspects it.
function ownerFolder(kind, row, fleet) {
  if (!row) return "";
  if (kind === "term") return row.cwd || "";
  if (kind === "workspace") return row.path || "";
  const loc = locate(fleet.workspaces, fleet.freeAgents, row.id);
  return row.workPath || (loc && loc.workspace && loc.workspace.path) || "";
}

const MODEL_KEYS = ["id", "kind", "ref", "x", "y", "w", "h", "pending", "state", "target", "name", "hint", "cwd", "status", "label", "stamp", "owned", "dirty", "wsId", "ask"];
function sameModel(a, b) {
  return !!a && MODEL_KEYS.every((k) => a[k] === b[k]);
}

// buildModel: everything a Panel renders, derived once per change from the
// panel row, the fleet, the tmux working list and the open tabs; the
// previous object is returned unchanged when nothing differs, so the
// memoized wrappers stay put.
function buildModel(panel, fleet, workingIds, openTabs, dirtyIds, prev) {
  const state = bindingState(panel, fleet);
  let target = null;
  let name = panel.ref;
  let hint = "";
  let cwd = "";
  let status = "stopped";
  let label = "Gone";
  let stamp = "";
  let wsId = "";
  let ask = "";
  if (panel.kind === "note") {
    // A pin's summary is its own header: the title names it, the tags are
    // the subdued line, and the chip says what the panel is rather than
    // inventing a status a note does not have.
    target = (Array.isArray(fleet.pins) ? fleet.pins : []).find((x) => x && x.id === panel.ref) || null;
    if (target) {
      name = target.title || "Untitled note";
      hint = (target.tags || []).join(" · ");
      status = "ready";
      label = "Note";
      stamp = target.updatedAt || "";
    }
  } else if (panel.kind === "file" || panel.kind === "diff") {
    // The file's own name is the panel's; the folder is the subdued line,
    // the way a file tab and the tree name one. The owner (terminal, agent
    // or folder) is what the read goes through and what can be gone, and
    // `cwd` is that owner's folder — what a `git.updated` is matched against.
    const at = parseRef(panel.kind, panel.ref);
    target = at ? refOwner(at, fleet) : null;
    if (at) {
      // The path is in the ref, so a panel whose owner is gone still names
      // its file; only the chip falls back to the gone word every kind uses.
      name = basename(at.path) || at.path;
      hint = at.path.slice(0, Math.max(0, at.path.length - name.length - 1));
      if (target) {
        cwd = ownerFolder(at.owner.kind, target, fleet);
        status = "ready";
        label = panel.kind === "diff" ? "Diff" : "File";
      }
    }
  } else if (panel.kind === "terminal") {
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
      // The transcript window is workspace-scoped (the route refuses an
      // agent that is not in the workspace it names), and a free agent's
      // workspace is the reserved one.
      wsId = (loc.workspace && loc.workspace.id) || FREE_WS;
      // What the agent is blocked on, read from the **fleet** row and never
      // from a panel's socket (ADR-0062's vocabulary, lib/needsYou.js's
      // words). That is what lets a panel that is unloaded, quiet or a
      // name-plate still say Needs you, and what the chat body's bar shows.
      if (status === "needs-you" && target.dialog) ask = target.dialog.title || methodLabel(target.dialog.method);
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
  // An editor with unsaved text says so in the chip, and that is also what
  // pins it against the viewport (chunkLoader `keep`).
  const dirty = panel.kind === "file" && !!dirtyIds && dirtyIds.has(panel.id);
  if (dirty) {
    status = "needs-you";
    label = "Unsaved";
  }
  const next = {
    id: panel.id, kind: panel.kind, ref: panel.ref, x: panel.x, y: panel.y, w: panel.w, h: panel.h, pending: !!panel.pending,
    state, target, name, hint, cwd, status, label, stamp, dirty, wsId, ask, owned: ownedByTab(panel.kind, panel.ref, openTabs),
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
  // The file panels whose editor holds unsaved text. It is chip state and
  // pin state at once: a dirty body is `keep`-pinned, so the band never
  // unmounts work in progress, not even when the matrix is hidden.
  const [dirtyIds, setDirtyIds] = useState(() => new Set());
  const dirtyRef = useRef(dirtyIds);
  dirtyRef.current = dirtyIds;
  // The pins a note panel binds (docs/plans/matrix-canvas.md §4.2). `null`
  // means "not read yet", the same distinction the fleet's `loaded` makes:
  // before the read, a note is not gone. Summaries only — the body reads its
  // own pin, because a 100 KB note has no business in this list.
  const [pins, setPins] = useState(null);
  // The connection list (ADR-0104), joined with the edges to say which link
  // currently grants (ADR-0116 §7). `null` until the first read, the same
  // "not read yet" the pins and the fleet make, so a link never reads broken
  // before anything has been read.
  const [peers, setPeers] = useState(null);
  const peersAt = useRef(0);
  const peersNeededRef = useRef(false);
  const pinsAt = useRef(0);
  // The pins list is read when something needs it — a note on this matrix,
  // or the picker offering one — and never otherwise: a matrix of terminals
  // costs no pin request at all.
  const pinsNeededRef = useRef(false);
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
      const list = normalizeCanvasList(await api("/api/canvases"));
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
      const det = normalizeCanvasDetail(await api("/api/canvases/" + enc(id)));
      if (!det) throw new Error("That matrix could not be read.");
      detailAt.current[id] = Date.now();
      if (pendingRef.current.matrixId === id) pendingRef.current = { matrixId: "", rows: new Map() };
      setStore((s) => {
        const next = applyCanvasEvent(s, { type: "canvas.updated", data: det.canvas });
        return { ...next, byId: { ...next.byId, [id]: det } };
      });
      setDetailError("");
    } catch (e) {
      if (e && e.status === 404) {
        setStore((s) => applyCanvasEvent(s, { type: "canvas.deleted", data: { id } }));
        return;
      }
      setDetailError(humanizeError(msgOf(e)));
    }
  }, []);
  // A grant can stop granting without the connection row moving: ADR-0104
  // invalidates a connection whose *owner* changed session, so a resume or a
  // fork silently breaks every link that end carries. The connection list is
  // therefore refetched for `peer.*` **and** for the two owner events, which
  // is why it is debounced — `agent.updated` is a busy row.
  const peersTimer = useRef(0);
  const peersDirty = useCallback(() => {
    if (!peersNeededRef.current || peersTimer.current) return;
    peersTimer.current = setTimeout(() => { peersTimer.current = 0; loadPeersRef.current(); }, PEERS_DEBOUNCE_MS);
  }, []);
  useEffect(() => () => { if (peersTimer.current) clearTimeout(peersTimer.current); }, []);
  const loadPeers = useCallback(async () => {
    try {
      const d = await api("/api/communication");
      peersAt.current = Date.now();
      setPeers({ owners: Array.isArray(d && d.owners) ? d.owners : [], connections: Array.isArray(d && d.connections) ? d.connections : [] });
    } catch {
      // Keep the last answer: a failed read must not flip every live link to
      // broken, which is the one direction a wrong answer is dangerous in.
    }
  }, []);
  const loadPeersRef = useRef(() => {});
  loadPeersRef.current = loadPeers;
  const loadPins = useCallback(async () => {
    try {
      const d = await api("/api/pins");
      pinsAt.current = Date.now();
      setPins(Array.isArray(d && d.pins) ? d.pins.filter((p) => p && p.id) : []);
    } catch {
      // The feed reconnect and the next reveal try again; a note panel keeps
      // the last list it had rather than flashing every pin as gone.
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
        if (pinsNeededRef.current) loadPins();
        if (peersNeededRef.current) loadPeers();
        return;
      }
      if (ev.type === "agent.tui") { setWorkingIds((cur) => applyTui(cur, ev)); return; }
      // What a drawn line grants changes with the connection list, never with
      // the matrix: refetch that, not the matrix (ADR-0116 §7 — on the feed,
      // never on a timer).
      if (PEER_EVENTS.test(ev.type)) peersDirty();
      if (ev.type.startsWith("peer.")) return;
      // A deleted target holds no pane: dispose what the Matrix kept
      // suspended now, not when the LRU gets to it (paneOwnership.js).
      if (ev.type === "terminal.deleted" || ev.type === "agent.deleted") {
        const gone = ev.data && ev.data.id;
        const kind = ev.type === "terminal.deleted" ? "terminal" : "agent";
        if (gone) forgetPane(gone, ownedByTab(kind, gone, (hostRef.current && hostRef.current.openTabs) || NO_PANELS));
        return;
      }
      // A note follows its pin: the feed carries the summary, so the list
      // above is patched by refetching it and the body refetches its own
      // markdown (NotePanel). No timer either side.
      if (touches(ev, ["pin"])) { if (pinsNeededRef.current) loadPins(); return; }
      if (!touches(ev, ["canvas"])) return;
      // Another client's rows arrive after the gesture ends (plan §4.7).
      if (gestureRef.current) { queuedRef.current.push(ev); return; }
      setStore((s) => settle(applyCanvasEvent(s, ev), ev));
    });
  }, [feed, loadList, loadDetail, loadPins, peersDirty]);
  useEffect(() => {
    if (hidden) return;
    const now = Date.now();
    if (listAt.current && now - listAt.current > REVEAL_STALE_MS) loadList();
    const id = currentRef.current;
    if (id && detailAt.current[id] && now - detailAt.current[id] > REVEAL_STALE_MS) loadDetail(id);
    if (pinsNeededRef.current && pinsAt.current && now - pinsAt.current > REVEAL_STALE_MS) loadPins();
    if (peersNeededRef.current && peersAt.current && now - peersAt.current > REVEAL_STALE_MS) loadPeers();
  }, [hidden, loadList, loadDetail, loadPins, loadPeers]);

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
  const edges = detail && Array.isArray(detail.edges) ? detail.edges : NO_EDGES;
  const edgesRef = useRef(NO_EDGES);
  edgesRef.current = edges;
  // The connection list is read only when this matrix actually holds a link.
  // A matrix with no edge asks nothing of ADR-0104, and drawing one reads it
  // fresh anyway (drawEdge below), because enrolment is exactly the thing
  // that may have changed a second ago.
  peersNeededRef.current = edges.length > 0;
  useEffect(() => {
    if (edges.length && !peersAt.current) loadPeers();
  }, [edges.length, loadPeers]);

  // Pins: read once when the first note panel appears or the picker opens,
  // then only the feed (and a stale reveal). `pins` stays null until then,
  // which is what keeps a note from reading as gone before the read.
  const pinsNeeded = pickerOpen || panels.some((p) => p.kind === "note");
  pinsNeededRef.current = pinsNeeded;
  useEffect(() => {
    if (pinsNeeded && !pinsAt.current) loadPins();
  }, [pinsNeeded, loadPins]);

  // The tmux-reported Working state of the agents on this matrix: one read
  // on open and reveal, then agent.tui from the feed (the sidebar's rule).
  const agentRefs = useMemo(() => panels.filter((p) => p.kind === "agent").map((p) => p.ref).sort().join(","), [panels]);
  useEffect(() => {
    if (hidden || !agentRefs) return undefined;
    let stop = false;
    api("/api/tui-working?ids=" + enc(agentRefs)).then((d) => { if (!stop) setWorkingIds((d && d.working) || []); }).catch(() => { /* the feed catches up */ });
    return () => { stop = true; };
  }, [agentRefs, hidden]);

  // Files and diffs come from where a path already exists — the file tabs
  // open in the desktop right now — so the picker lists and never browses
  // (the entry points are docs/architecture/matrix.md, Surface). One list
  // feeds both groups: the same path as a file and as a diff is two panels,
  // which is exactly what the unique index allows.
  const fileOptions = useMemo(() => {
    const out = [];
    for (const tab of openTabs) {
      if (!isFileTab(tab)) continue;
      const t = parseFileTab(tab);
      if (!t) continue;
      const owner = { kind: t.kind, id: t.id };
      const ref = buildRef("file", { owner, path: t.path });
      if (!ref) continue;
      const row = refOwner({ owner }, fleet);
      const where = (row && row.name) || OWNER_WORD[t.kind] || "Agent";
      const name = basename(t.path) || t.path;
      const dir = t.path.slice(0, Math.max(0, t.path.length - name.length - 1));
      out.push({ ref, name, hint: dir ? dir + " · " + where : where, owner, path: t.path });
    }
    return out;
  }, [openTabs, fleet.workspaces, fleet.freeAgents, fleet.terminals]);

  // ---- panel models -------------------------------------------------------
  const modelsRef = useRef(new Map());
  // What every binding is judged against: the desktop's fleet plus the lists
  // the other kinds bind (pins). One object so bindingState has one argument.
  const bindings = useMemo(() => ({ ...fleet, pins }), [fleet, pins]);
  const models = useMemo(() => {
    const prev = modelsRef.current;
    const next = new Map();
    const out = panels.map((p) => {
      const m = buildModel(p, bindings, workingIds, openTabs, dirtyIds, prev.get(p.id));
      next.set(p.id, m);
      return m;
    });
    modelsRef.current = next;
    return out;
  }, [panels, bindings, workingIds, openTabs, dirtyIds]);
  useEffect(() => {
    if (focusedId && !models.some((m) => m.id === focusedId)) setFocusedId("");
    if (maximizedId && !models.some((m) => m.id === maximizedId)) setMaximizedId("");
  }, [models, focusedId, maximizedId]);

  // ---- edges (ADR-0116) ---------------------------------------------------
  // What every panel is called, and which folder it works in: the reason a
  // broken link shows, the sentence the enrolment offers and the one that
  // names both folders are all built from these, so a dialog never says an
  // id at an owner.
  const names = useMemo(() => Object.fromEntries(models.map((m) => [m.id, m.name])), [models]);
  const folders = useMemo(
    () => Object.fromEntries(models.map((m) => [m.id, shortPath(m.cwd) !== "—" ? shortPath(m.cwd) : m.hint || ""])),
    [models],
  );
  const namesRef = useRef(names);
  namesRef.current = names;
  const foldersRef = useRef(folders);
  foldersRef.current = folders;
  // peers is null until the first read; an edge on an unread list says
  // nothing rather than "broken", which would be a lie in the one direction
  // that matters.
  const peerIx = useMemo(() => peerIndex(peers || NO_PEERS), [peers]);
  const grants = useMemo(() => {
    const out = new Map();
    for (const e of edges) {
      const g = edgeGrant(e, panels, peerIx, names);
      if (g) out.set(e.id, g);
    }
    return out;
  }, [edges, panels, peerIx, names]);
  // What the canvas draws: one row per edge whose two ends are on the board.
  // An edge whose endpoint the client has not got is the one case the plane
  // must not draw (edgeEndpoints' null), and it is simply absent here.
  const edgeRows = useMemo(() => edges.map((e) => {
    const g = grants.get(e.id);
    if (!g) return null;
    return {
      id: e.id,
      source: e.aPanel,
      target: e.bPanel,
      grants: peers ? g.grants : true,
      reason: peers ? g.reason : "",
      label: g.ends[0].name + " and " + g.ends[1].name,
    };
  }).filter(Boolean), [edges, grants, peers]);
  // What a grid-mode header wears instead: the count, and whether any of
  // them grants nothing.
  const links = useMemo(() => {
    if (!edges.length) return NO_LINKS;
    const counts = linkCounts(edges, panels, peerIx, names);
    const out = {};
    for (const [id, at] of Object.entries(counts)) {
      out[id] = { count: at.count, broken: peers ? at.broken : 0, title: linkChipTitle(peers ? at : { count: at.count, broken: 0 }) };
    }
    return out;
  }, [edges, panels, peerIx, names, peers]);

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
      const res = await api("/api/canvases/" + enc(matrixId) + "/layout", json("PATCH", { ifUpdatedAt: det.canvas.updatedAt, panels: [...rows.values()] }));
      setStore((s) => applyCanvasEvent(s, { type: "canvas.layout", data: res }));
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
    const diff = layoutDiff(det.panels.filter((p) => !p.pending), next);
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
  const onGestureStart = useCallback((id) => {
    gestureRef.current = true;
    loader.pin(id, true);
  }, [loader]);
  const onGestureStop = useCallback((id) => {
    gestureRef.current = false;
    loader.pin(id, false);
    const q = queuedRef.current;
    queuedRef.current = [];
    if (q.length) setStore((s) => q.reduce((acc, ev) => settle(applyCanvasEvent(acc, ev), ev), s));
  }, [loader]);

  // ---- mutations ----------------------------------------------------------
  async function createMatrix(name) {
    const res = await api("/api/canvases", json("POST", { name }));
    setStore((s) => applyCanvasEvent(s, { type: "canvas.created", data: res }));
    select(res.id);
  }
  async function renameMatrix(name) {
    const id = currentRef.current;
    const m = storeRef.current.list.find((x) => x.id === id);
    if (!m) return;
    try {
      const res = await api("/api/canvases/" + enc(id), json("PATCH", { name, ifUpdatedAt: m.updatedAt }));
      setStore((s) => applyCanvasEvent(s, { type: "canvas.updated", data: res }));
    } catch (e) {
      if (e && e.status === 409) { await loadDetail(id); throw new Error("Matrix changed elsewhere — reloaded. Try again."); }
      throw e;
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
      await api("/api/canvases/" + enc(id), { method: "DELETE" });
      setStore((s) => applyCanvasEvent(s, { type: "canvas.deleted", data: { id } }));
    } catch (e) {
      if (e && e.status === 404) setStore((s) => applyCanvasEvent(s, { type: "canvas.deleted", data: { id } }));
      else toastError(e);
    }
  }
  async function addPanel({ kind, ref }) {
    setPickerOpen(false);
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    if (!det) return;
    const size = PANEL_DEFAULT_CANVAS;
    const { x, y } = nextSlot(det.panels, size.w, size.h);
    const tmp = "tmp-" + Date.now().toString(36) + Math.random().toString(36).slice(2, 6);
    const body = { kind, ref, x, y, w: size.w, h: size.h };
    // The wrapper shows at once and settles on the server's answer.
    setStore((s) => patchPanels(s, id, (ps) => [...ps, { id: tmp, ...body, createdAt: "", pending: true }]));
    try {
      const res = await api("/api/canvases/" + enc(id) + "/panels", json("POST", body));
      setStore((s) => applyCanvasEvent(patchPanels(s, id, (ps) => ps.filter((p) => p.id !== tmp)), { type: "canvas.panel.added", data: res }));
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
    // Undo puts the panel back; it cannot put unsaved text back, so this is
    // the one remove that asks.
    if (model.kind === "file" && heldDocumentDirty(model.ref)) {
      const ok = await askConfirm({
        title: "Remove " + model.name + "?",
        message: "It has unsaved changes. Removing the panel loses them.",
        confirmLabel: "Remove panel",
        danger: true,
      });
      if (!ok) return;
    }
    const order = panelOrder(panelsRef.current);
    const at = order.indexOf(model.id);
    const next = order[at + 1] || order[at - 1] || "";
    const removed = { type: "canvas.panel.removed", data: { id, panelId: model.id } };
    try {
      await api("/api/canvases/" + enc(id) + "/panels/" + enc(model.id), { method: "DELETE" });
      setStore((s) => applyCanvasEvent(s, removed));
      // The panel is gone, so its document, its pin and its chip go with it.
      if (model.kind === "file") {
        forgetDocument(model.ref);
        loader.keep(model.id, false);
        setDirtyIds((cur) => {
          if (!cur.has(model.id)) return cur;
          const next = new Set(cur);
          next.delete(model.id);
          return next;
        });
      }
      notify({
        level: "info",
        title: "Removed " + model.name + " from the matrix.",
        actions: [{ label: "Undo", primary: true, run: () => restorePanel(id, model) }],
        key: "mx-removed:" + model.id,
      });
    } catch (e) {
      if (e && e.status === 404) setStore((s) => applyCanvasEvent(s, removed));
      else toastError(e);
    }
    if (focusedRef.current === model.id) focusPanel(next);
  }
  // ---- drawing and removing an edge (ADR-0116 §2, §4, §5) ----------------
  // Only the owner draws one, in the browser, through the authenticated
  // owner API; the gesture is the canvas's connector, and everything that
  // decides is here. Two rules the ADR sets and this function keeps:
  //
  //   * **An edge never enrols silently.** The connection list is read
  //     *fresh* — enrolment is exactly the thing that may have changed a
  //     second ago — and every question is asked before anything is written.
  //     Cancel at either question leaves no edge and no connection.
  //   * **The two questions never merge.** Pairing across folders is the one
  //     power a workspace could not give, so it is confirmed on its own and
  //     names both folders; the enrolment offer is the existing owner action
  //     ADR-0104 already has, and says what it grants.
  async function drawEdge(connection) {
    const a = connection && connection.source;
    const b = connection && connection.target;
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    if (!a || !b || !det) return;
    // The server's refusals, in the server's words, before the request: a
    // pair already linked, a kind with no mailbox, the cap.
    const refusal = validateEdge({ aPanel: a, bPanel: b }, det.panels, det.edges || NO_EDGES);
    if (refusal) { notify({ level: "warn", title: humanizeError(refusal), key: "mx-edge-refused" }); return; }
    let payload;
    try {
      payload = await api("/api/communication");
    } catch (e) { toastError(e); return; }
    setPeers({ owners: payload.owners || [], connections: payload.connections || [] });
    const ix = peerIndex(payload);
    const grant = edgeGrant({ id: "draft", aPanel: a, bPanel: b }, det.panels, ix, namesRef.current);
    if (!grant) return;
    // Across two folders, first and on its own (§5). Cancel writes nothing.
    if (grant.cross && !(await askConfirm(crossFolderConfirm(grant.ends, foldersRef.current)))) return;
    const offer = enrolOffer(grant.ends);
    if (offer && offer.blocked) {
      notify({ level: "warn", title: offer.title, body: offer.message, key: "mx-edge-blocked:" + a + "|" + b });
      return;
    }
    if (offer) {
      if (!(await askConfirm(offer))) return;
      const launch = Array.isArray(payload.launchCLIs) ? payload.launchCLIs : [];
      for (const end of grant.ends) {
        if (end.state === "on") continue;
        const owner = (payload.owners || []).find((o) => o && o.kind === end.kind && o.ownerId === end.ref);
        if (!owner) continue;
        try {
          await api("/api/communication", json("POST", { kind: owner.kind, ownerId: owner.ownerId, sessionKey: owner.sessionKey, automatic: launch.includes(owner.cli) }));
        } catch (e) {
          // The enrolment is the Messages view's action; when it needs the
          // connection guide, that is where the guide lives.
          notify({
            level: "error",
            title: "Couldn’t connect " + end.name + ".",
            body: msgOf(e),
            actions: [{ label: "Open Messages", primary: true, run: () => { location.hash = "#/clis/messages"; } }],
            key: "mx-edge-enrol:" + end.key,
          });
          loadPeers();
          return;
        }
      }
      loadPeers();
    }
    try {
      const res = await api("/api/canvases/" + enc(id) + "/edges", json("POST", { aPanel: a, bPanel: b }));
      setStore((st) => applyCanvasEvent(st, { type: "canvas.edge.added", data: res }));
    } catch (e) {
      toastError(e);
      loadDetail(id);
    }
  }
  // Removing the line revokes the grant (§4): there is no second write,
  // because there was never a first — the contact is derived from the live
  // edge on every read. It asks only when the link currently grants; taking
  // away a link that grants nothing takes nothing away.
  async function removeEdge(edgeId) {
    const id = currentRef.current;
    const det = storeRef.current.byId[id];
    const row = det && (det.edges || NO_EDGES).find((e) => e.id === edgeId);
    if (!row) return;
    const grant = edgeGrant(row, det.panels, peerIndex(peers || NO_PEERS), namesRef.current);
    const ask = peers ? removeConfirm(grant) : null;
    if (ask && !(await askConfirm(ask))) return;
    const removed = { type: "canvas.edge.removed", data: { id, edgeId } };
    try {
      await api("/api/canvases/" + enc(id) + "/edges/" + enc(edgeId), { method: "DELETE" });
      setStore((st) => applyCanvasEvent(st, removed));
    } catch (e) {
      if (e && e.status === 404) setStore((st) => applyCanvasEvent(st, removed));
      else toastError(e);
    }
  }
  // Stable for the life of the surface, so the canvas's edge objects (and
  // React Flow's diff of them) do not churn on every render.
  const edgeActions = useRef({});
  edgeActions.current = { drawEdge, removeEdge };
  const onConnect = useCallback((c) => { edgeActions.current.drawEdge(c); }, []);
  const onRemoveEdge = useCallback((edgeId) => { edgeActions.current.removeEdge(edgeId); }, []);

  async function restorePanel(matrixId, model) {
    const body = { kind: model.kind, ref: model.ref, x: model.x, y: model.y, w: model.w, h: model.h };
    try {
      const res = await api("/api/canvases/" + enc(matrixId) + "/panels", json("POST", body));
      setStore((s) => applyCanvasEvent(s, { type: "canvas.panel.added", data: res }));
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
    const canvas = canvasRef.current;
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
    if (canvasRef.current) canvasRef.current.snapToOne(id);
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
      const canvas = canvasRef.current;
      if (!canvas || e.ctrlKey || e.metaKey || e.altKey) return;
      if (e.key === "+" || e.key === "=") { e.preventDefault(); canvas.zoomIn(); return; }
      if (e.key === "-" || e.key === "_") { e.preventDefault(); canvas.zoomOut(); return; }
      if (e.key === "0") { e.preventDefault(); canvas.fit(); }
    },
    onOpen: (model) => {
      const h = hostRef.current || {};
      // A note is written in Pin Studio, which is a route and not a tab.
      if (model.kind === "note") { location.hash = pinHash(model.ref); return; }
      // A file opens the tab it would have had: #/file/<owner>/<id>/<path>.
      // A diff opens that same tab on its Diff view — the pair ADR-0074 uses.
      if (model.kind === "file" || model.kind === "diff") {
        const at = parseRef(model.kind, model.ref);
        if (at && h.openFileTab) h.openFileTab(at.owner.kind, at.owner.id, at.path, model.kind === "diff" ? "diff" : "file");
        return;
      }
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
    // The grid's link chip: the count is spatial nowhere, so it goes to the
    // one place every live edge is listed and can be revoked (ADR-0116's
    // "the canvas is not the only place a grant may be audited").
    onLinks: () => { location.hash = "#/clis/messages"; },
    // Maximize takes the keyboard into the pane (the layer mounts its body
    // with autoFocus); Restore hands it back to the wrapper's chrome.
    onMaximize: (model) => {
      if (maximizedRef.current === model.id) { restoreMaximized(); return; }
      setMaximizedId(model.id);
      engagePanel(model.id);
    },
    // onOpenFile(model, path): a path the body found — a terminal's OSC 8
    // link, or the diff body's own **Open file**. The owner is the panel's
    // binding, which for a file or a diff is inside the ref.
    onOpenFile: (model, path) => {
      const h = hostRef.current || {};
      if (!h.openFileTab) return;
      if (model.kind === "file" || model.kind === "diff") {
        const at = parseRef(model.kind, model.ref);
        if (at) h.openFileTab(at.owner.kind, at.owner.id, path);
        return;
      }
      h.openFileTab(model.kind === "agent" ? "agent" : "term", model.ref, path);
    },
    // onDirty(model, dirty): an editor gained or lost unsaved text. It pins
    // the panel — the same `pinned` set a drag and the focus use, except
    // that this one survives a hidden matrix, because a hidden tab losing a
    // draft is exactly the silent unmount this rule exists to stop.
    onDirty: (model, dirty) => {
      loader.keep(model.id, !!dirty);
      setDirtyIds((cur) => {
        if (cur.has(model.id) === !!dirty) return cur;
        const next = new Set(cur);
        if (dirty) next.add(model.id);
        else next.delete(model.id);
        return next;
      });
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
        <p className="app-blank-sub">A matrix shows many agents, terminals and notes side by side, live.</p>
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
        <p className="app-blank-sub">A panel is one agent, terminal or note on this matrix.</p>
        <button type="button" className="btn btn-sm btn-primary" onClick={() => setPickerOpen(true)}><IconPlus size={13} /> Add panel</button>
      </div>
    );
  } else {
    body = (
      <div className="mx-stage">
        <div className="mx-body is-canvas" ref={setBody} inert={!!maximizedId}>
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
                edgeRows={edgeRows}
                onConnect={onConnect}
                onRemoveEdge={onRemoveEdge}
                onLayout={applyLayout}
                onGestureStart={onGestureStart}
                onGestureStop={onGestureStop}
                onReady={onCanvasReady}
              />
          </Suspense>
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
                body={bodyKinds[maxModel.id] === "off" ? "off" : "live"}
                hidden={!!hidden}
                focused={engaged && focusedId === maxModel.id}
                onOpen={() => handlers.onOpen(maxModel)}
                onRemove={() => handlers.onRemove(maxModel)}
                onRun={() => handlers.onRun(maxModel)}
                onOpenFile={(path) => handlers.onOpenFile(maxModel, path)}
                onDirty={(dirty) => handlers.onDirty(maxModel, dirty)}
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
            <button type="button" className="btn btn-sm" onClick={() => setPickerOpen(true)}><IconPlus size={13} /> Add panel</button>
          ) : null}
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => setNameDialog("new")} title="Create another matrix">New matrix</button>
          {current ? (
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild>
                <button type="button" className="btn btn-sm btn-ghost mx-menu-btn" aria-label={"Actions for " + current.name} title="Tidy, rename or delete this matrix"><IconEllipsis size={15} /></button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Portal>
                <DropdownMenu.Content className="ws-row-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
                  {panels.length ? (
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
        pins={pins}
        files={fileOptions}
        onMatrix={onMatrix}
        workingIds={workingIds}
        onPick={addPanel}
        onNewPin={() => { setPickerOpen(false); go("pins-new"); }}
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
