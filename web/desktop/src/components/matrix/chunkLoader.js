import { loadPolicy } from "@picode/shared/domain/matrix.js";

// ChunkLoader — the glue between the browser's intersection test and the
// pure loadPolicy (docs/plans/matrix-app.md §4.5). One per surface. Panel
// wrappers register their DOM node; an IntersectionObserver rooted on the
// surface's scroll container, one viewport height of margin each way,
// feeds `near`; drag, resize, focus and maximize feed `pinned`; the
// surface feeds `hidden` (a hidden matrix reads as every panel far away).
// Every change and every deadline runs the policy once; onChange receives
// the new loaded set. Nothing here touches sockets: the components mount
// and unmount bodies from the set, and TerminalPanel's unmount is what
// suspends an attach (paneOwnership.js).
const NONE = new Set();

export class ChunkLoader {
  constructor(onChange) {
    this.onChange = onChange;
    this.els = new Map(); // id → wrapper element
    this.near = new Map(); // id → the observer's last word
    this.loaded = new Set();
    this.pinned = new Set();
    this.timers = {};
    this.hidden = false;
    this.wake = 0;
    this.observer = null;
  }

  // attach(root): (re)create the observer on the scroll container. Called
  // through a callback ref, so a null root (the empty state replaced the
  // grid) just disconnects.
  attach(root) {
    if (this.observer) this.observer.disconnect();
    this.observer = null;
    if (!root || typeof IntersectionObserver === "undefined") return;
    this.observer = new IntersectionObserver((entries) => {
      for (const en of entries) {
        const id = en.target.dataset.mxPanel;
        if (id) this.near.set(id, en.isIntersecting);
      }
      this.tick();
    }, { root, rootMargin: "100% 0px", threshold: 0 });
    for (const el of this.els.values()) this.observer.observe(el);
  }

  observe(id, el) {
    if (!id || !el) return;
    el.dataset.mxPanel = id;
    this.els.set(id, el);
    if (this.observer) this.observer.observe(el);
  }

  unobserve(id) {
    const el = this.els.get(id);
    if (el && this.observer) this.observer.unobserve(el);
    this.els.delete(id);
    this.near.delete(id);
    delete this.timers[id];
    if (this.loaded.delete(id)) this.emit();
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

  tick() {
    if (this.wake) {
      clearTimeout(this.wake);
      this.wake = 0;
    }
    const now = Date.now();
    const entries = [];
    for (const id of this.els.keys()) {
      entries.push({ id, near: !this.hidden && !!this.near.get(id), loaded: this.loaded.has(id) });
    }
    // A hidden matrix holds no attaches of its own (plan §4.5): far and
    // unpinned, whatever was focused or maximized when it was last shown.
    const res = loadPolicy(entries, now, this.hidden ? NONE : this.pinned, this.timers);
    this.timers = res.timers;
    let changed = false;
    for (const id of res.load) { this.loaded.add(id); changed = true; }
    for (const id of res.unload) { this.loaded.delete(id); changed = true; }
    if (changed) this.emit();
    if (res.wakeAt) {
      this.wake = setTimeout(() => { this.wake = 0; this.tick(); }, Math.max(0, res.wakeAt - now));
    }
  }

  emit() {
    this.onChange(new Set(this.loaded));
  }

  dispose() {
    if (this.wake) clearTimeout(this.wake);
    this.wake = 0;
    if (this.observer) this.observer.disconnect();
    this.observer = null;
    this.els.clear();
    this.near.clear();
    this.loaded.clear();
    this.pinned.clear();
    this.timers = {};
  }
}
