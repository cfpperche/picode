// fileDocs — the open documents of a matrix's `file` panels, kept outside
// React so a body that moves host keeps its text (docs/plans/matrix-canvas.md
// §4.2). It is the matrix's `paneOwnership.js` for an editor: maximizing a
// panel, and switching layout mode, unmount the body in one host and mount
// it in the other inside the same commit, and a `createFileDocument` made in a
// component's `useMemo` dies with the component — which would throw away an
// unsaved draft on a button the viewer pressed to *read more of it*.
//
// The key is the panel's ref (`<owner>:<id>:<path>`), so two panels on the
// same file share one document and the same unsaved text, exactly as two
// hosts of one terminal share one xterm.
const held = new Map(); // key → { doc, mounted, drop }

// holdDocument(key, make): the document for this key, made once. Every hold
// is one mount and must be matched by a release.
export function holdDocument(key, make) {
  let e = held.get(key);
  if (!e) {
    e = { doc: make(), mounted: 0, drop: 0 };
    held.set(key, e);
  }
  if (e.drop) {
    clearTimeout(e.drop);
    e.drop = 0;
  }
  e.mounted += 1;
  return e.doc;
}

// releaseDocument(key): a mount left. Whether it is *gone* is decided a tick
// later, because a body moving host mounts before the old one unmounts —
// the same rule the chunk loader and the pane registry follow. A document
// with unsaved text is never dropped: the panel leaving the matrix
// (forgetDocument) is what frees it.
export function releaseDocument(key) {
  const e = held.get(key);
  if (!e) return;
  e.mounted = Math.max(0, e.mounted - 1);
  if (e.mounted || e.drop) return;
  e.drop = setTimeout(() => {
    e.drop = 0;
    if (e.mounted || held.get(key) !== e) return;
    if (e.doc.getSnapshot().dirty && !e.forgotten) return;
    e.doc.dispose();
    held.delete(key);
  }, 0);
}

// forgetDocument(key): the panel is gone from the matrix. Its draft goes
// with it — the remove is reversible through the toast, the text is not,
// which is why removing a dirty panel asks first.
export function forgetDocument(key) {
  const e = held.get(key);
  if (!e) return;
  // The body is still mounted when the panel is removed (React unmounts it
  // on the next commit), so this marks the document rather than disposing
  // under it: the release that follows drops it, unsaved text and all.
  e.forgotten = true;
  if (e.mounted) return;
  if (e.drop) clearTimeout(e.drop);
  held.delete(key);
  e.doc.dispose();
}

// heldDocumentDirty(key): does this key hold unsaved text? The surface asks
// before it does something that would throw it away.
export function heldDocumentDirty(key) {
  const e = held.get(key);
  return !!e && !!e.doc.getSnapshot().dirty;
}
