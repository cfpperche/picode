import { loadPolicy } from "@picode/shared/domain/matrix.js";

// ChunkLoader — the glue between the browser's intersection test and the
// pure loadPolicy (docs/plans/matrix-app.md §4.5). One per surface. Panel
// wrappers register their DOM node; an IntersectionObserver rooted on the
// surface's scroll container (grid mode) or on the React Flow pane (canvas
// mode) feeds `near`; drag, resize, focus and maximize feed `pinned`; the
// surface feeds `hidden` (a hidden matrix reads as every panel far away).
// Every change and every deadline runs the policy once; onChange receives
// the new loaded set and what each body should render. Nothing here touches
// sockets: the components mount and unmount bodies from the set, and
// TerminalPanel's unmount is what suspends an attach (paneOwnership.js).
//
// Canvas mode adds the zoom (matrix-canvas.md §4.3). The observer accounts
// for the plane's transform — C0 verified it — but the band is a
// *screen-space* rule, so zooming out multiplies the plane it covers (500
// of 500 panels at 0.2). What bounds the attaches down there is the still
// rule: the loader carries the zoom and the hysteretic band into
// loadPolicy, and hands the per-panel body kind to the surface.
const NONE = new Set();
export const GRID_MARGIN = "100% 0px"; // the grid only scrolls vertically
export const CANVAS_MARGIN = "100%"; // a plane pans both ways (C0: 49 panels in band against 28)

function sameBodies(a, b) {
  const ka = Object.keys(a);
  if (ka.length !== Object.keys(b).length) return false;
  return ka.every((k) => a[k] === b[k]);
}

export class ChunkLoader {
  constructor(onChange) {
    this.onChange = onChange;
    this.els = new Map(); // id → wrapper element
    this.near = new Map(); // id → the observer's last word
    this.loaded = new Set();
    this.bodies = {}; // id → off | live | still | plate
    this.pinned = new Set();
    // Bodies that must not be unmounted even by a hidden matrix, because
    // unmounting them would throw work away: an editor with unsaved changes
    // (matrix-canvas.md §4.2). `pinned` is the gesture's and the focus's,
    // and a hidden matrix drops it; `kept` survives that.
    this.kept = new Set();
    this.panes = new Map(); // id → does this body hold an xterm?
    this.timers = {};
    this.hidden = false;
    this.wake = 0;
    this.observer = null;
    this.margin = GRID_MARGIN;
    this.dropping = new Map(); // id → the timer of a wrapper that just unmounted
    // The canvas's camera. `band` is the hysteresis's only memory: the
    // policy answers with it and it comes back in on the next tick.
    this.view = { zoom: 1, band: "live" };
    // beforeChange(prev, next): the canvas sets this to capture a still of
    // every pane that is about to stop being live — before the swap, never
    // in the unmount (C0: a capture in the cleanup is one crossing late).
    this.beforeChange = null;
  }

  // attach(root, rootMargin): (re)create the observer on the scroll
  // container or the canvas pane. Called through a callback ref, so a null
  // root (the empty state replaced the grid) just disconnects.
  attach(root, rootMargin) {
    if (rootMargin) this.margin = rootMargin;
    if (this.observer) this.observer.disconnect();
    this.observer = null;
    if (!root || typeof IntersectionObserver === "undefined") return;
    this.observer = new IntersectionObserver((entries) => {
      for (const en of entries) {
        const id = en.target.dataset.mxPanel;
        if (id) this.near.set(id, en.isIntersecting);
      }
      this.tick();
    }, { root, rootMargin: this.margin, threshold: 0 });
    for (const el of this.els.values()) this.observer.observe(el);
  }

  // setZoom(zoom): the canvas's viewport moved. Grid mode never calls it
  // and stays at 1, where every loaded body is live.
  setZoom(zoom) {
    if (!(zoom > 0) || zoom === this.view.zoom) return;
    this.view = { ...this.view, zoom };
    this.tick();
  }

  observe(id, el) {
    if (!id || !el) return;
    const drop = this.dropping.get(id);
    if (drop) {
      clearTimeout(drop);
      this.dropping.delete(id);
    }
    el.dataset.mxPanel = id;
    this.els.set(id, el);
    if (this.observer) this.observer.observe(el);
  }

  // unobserve(id): the wrapper unmounted. Whether it is *gone* is decided a
  // tick later, because switching layout mode unmounts every panel in one
  // host and mounts it in the other inside the same commit — the same rule
  // paneOwnership.js follows for the pane itself. Dropping the loaded flag
  // at once would make the switch a socket suspend and kick per panel.
  unobserve(id) {
    const el = this.els.get(id);
    if (el && this.observer) this.observer.unobserve(el);
    this.els.delete(id);
    this.near.delete(id);
    this.panes.delete(id);
    delete this.timers[id];
    if (!this.loaded.has(id) && !Object.hasOwn(this.bodies, id)) return;
    const drop = this.dropping.get(id);
    if (drop) clearTimeout(drop);
    this.dropping.set(id, setTimeout(() => {
      this.dropping.delete(id);
      if (this.els.has(id)) return; // it came back in another host
      this.loaded.delete(id);
      if (Object.hasOwn(this.bodies, id)) {
        const next = { ...this.bodies };
        delete next[id];
        this.bodies = next;
      }
      this.emit();
    }, 0));
  }

  setHidden(hidden) {
    if (this.hidden === !!hidden) return;
    this.hidden = !!hidden;
    this.tick();
  }

  pin(id, on) {
    if (!id) return;
    const had = this.pinned.has(id);
    if (on) this.pinned.add(id);
    else this.pinned.delete(id);
    if (had !== !!on) this.tick();
  }

  // keep(id, on): this body holds unsaved work. It never unloads — not on
  // the band's 5 s, not when the matrix is hidden — until it says so itself.
  keep(id, on) {
    if (!id) return;
    const had = this.kept.has(id);
    if (on) this.kept.add(id);
    else this.kept.delete(id);
    if (had !== !!on) this.tick();
  }

  // setPane(id, on): whether this panel's body is a terminal. The zoom's
  // still rule is about a cell, so a body without one is decided by the
  // other half of the table (loadPolicy). Unknown reads as a pane, which is
  // what every panel was before C3.
  setPane(id, on) {
    if (!id) return;
    const had = this.panes.get(id);
    if (had === !!on) return;
    this.panes.set(id, !!on);
    this.tick();
  }

  tick() {
    if (this.wake) {
      clearTimeout(this.wake);
      this.wake = 0;
    }
    const now = Date.now();
    const entries = [];
    for (const id of this.els.keys()) {
      entries.push({ id, near: !this.hidden && !!this.near.get(id), loaded: this.loaded.has(id), pane: this.panes.get(id) !== false });
    }
    // A hidden matrix holds no attaches of its own (plan §4.5): far and
    // unpinned, whatever was focused or maximized when it was last shown.
    const res = loadPolicy(entries, now, this.pinnedNow(), this.timers, this.view);
    this.timers = res.timers;
    this.view = { ...this.view, band: res.band };
    let changed = false;
    for (const id of res.load) { this.loaded.add(id); changed = true; }
    for (const id of res.unload) { this.loaded.delete(id); changed = true; }
    if (!sameBodies(this.bodies, res.bodies)) {
      // Before the swap, not after: the canvas takes its stills here.
      if (this.beforeChange) this.beforeChange(this.bodies, res.bodies);
      this.bodies = res.bodies;
      changed = true;
    }
    if (changed) this.emit();
    if (res.wakeAt) {
      this.wake = setTimeout(() => { this.wake = 0; this.tick(); }, Math.max(0, res.wakeAt - now));
    }
  }

  // A hidden matrix holds no attaches of its own (plan §4.5), so its
  // gestures and focus stop pinning — but a body with unsaved work is kept
  // whatever the matrix is doing.
  pinnedNow() {
    if (!this.hidden) return this.kept.size ? new Set([...this.pinned, ...this.kept]) : this.pinned;
    return this.kept.size ? this.kept : NONE;
  }

  emit() {
    this.onChange(new Set(this.loaded), this.bodies);
  }

  dispose() {
    if (this.wake) clearTimeout(this.wake);
    this.wake = 0;
    for (const t of this.dropping.values()) clearTimeout(t);
    this.dropping.clear();
    if (this.observer) this.observer.disconnect();
    this.observer = null;
    this.els.clear();
    this.near.clear();
    this.loaded.clear();
    this.pinned.clear();
    this.kept.clear();
    this.panes.clear();
    this.timers = {};
    this.bodies = {};
    this.beforeChange = null;
  }
}
