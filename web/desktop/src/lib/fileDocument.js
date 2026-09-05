// One open document. Reads and writes are injected so the same lifecycle is
// exercised by the browser and node:test (ADR-0074).
export function createFileDocument({ read, write, release = () => {} }) {
  let state = { kind: "load", text: "", dirty: false, saving: false, refreshing: false, error: "", revision: 0 };
  let savedText = "";
  let mtime = 0;
  let lifetime = 0;
  let readID = 0;
  let edits = 0;
  let reader = null;
  let pendingSave = null;
  const listeners = new Set();
  const publish = (patch) => {
    state = { ...state, ...patch };
    listeners.forEach((fn) => fn());
  };

  const doc = {
    getSnapshot: () => state,
    subscribe: (fn) => { listeners.add(fn); return () => listeners.delete(fn); },
    edit(text) {
      if (state.kind !== "text") return;
      edits++;
      publish({ text, dirty: text !== savedText });
    },
    async refresh({ discard = false } = {}) {
      if (state.saving || (state.dirty && !discard)) return false;
      reader?.abort();
      reader = new AbortController();
      const id = ++readID;
      const editAtStart = edits;
      publish({ refreshing: true });
      try {
        const page = await read(reader.signal);
        if (id !== readID || editAtStart !== edits) {
          release(page);
          return false;
        }
        const changed = state.kind !== page.kind || (page.kind === "text" ? state.text !== page.text : state.src !== page.src);
        release(state);
        savedText = page.text || "";
        mtime = Number(page.mtime) || 0;
        publish({ ...page, text: savedText, dirty: false, error: "", revision: state.revision + (changed ? 1 : 0) });
        return true;
      } catch (e) {
        if (id === readID && editAtStart === edits) {
          publish({ kind: state.kind === "load" ? "msg" : state.kind, error: e.message || "Could not read this file." });
        }
        return false;
      } finally {
        if (id === readID) publish({ refreshing: false });
      }
    },
    save() {
      if (pendingSave) return pendingSave;
      if (state.kind !== "text" || !state.dirty) return Promise.resolve(!state.dirty);
      // A read started before this write must never replace its buffer.
      reader?.abort();
      readID++;
      const generation = lifetime;
      const text = state.text;
      publish({ saving: true, refreshing: false, error: "" });
      pendingSave = Promise.resolve().then(() => write(text, mtime)).then((page) => {
        if (generation !== lifetime) return false;
        savedText = text;
        mtime = Number(page.mtime) || 0;
        publish({ dirty: state.text !== savedText });
        return !state.dirty;
      }).catch((e) => {
        if (generation === lifetime) publish({ error: e.message || "Could not save this file." });
        return false;
      }).finally(() => {
        pendingSave = null;
        if (generation === lifetime) publish({ saving: false });
      });
      return pendingSave;
    },
    dispose() {
      lifetime++;
      readID++;
      reader?.abort();
      release(state);
    },
  };
  return doc;
}

// A second click while deciding shares the first decision; callers still
// check their own navigation generation before applying it.
export function createDocumentGuard(doc, ask) {
  let pending = null;
  return () => {
    if (pending) return pending;
    pending = (async () => {
      if (doc.getSnapshot().saving) await doc.save();
      if (!doc.getSnapshot().dirty) return true;
      const choice = await ask();
      if (choice === "discard") return true;
      if (choice === "save") return doc.save();
      return false;
    })().finally(() => { pending = null; });
    return pending;
  };
}
