import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { hunksFromDiff, countOf } from "../lib/diff.js";
import { treeApiBase } from "../lib/fileTree.js";
import DiffLine from "./DiffLine.jsx";
import GitAssetPreview from "./GitAssetPreview.jsx";

// One file's working-tree diff — what a change dot in the file tree expands
// into (ADR-0032). Rendered with the commit pane's classes: same lines, same
// counts, a different question (vs HEAD, not vs a parent commit). `nonce`
// re-fetches when the surface refreshes.
export default function WorkingDiff({ owner, path, nonce, onClose, onOpenFile }) {
  const [diff, setDiff] = useState(null);
  const [error, setError] = useState("");

  const base = treeApiBase(owner ? owner.kind : "agent");
  const ownerId = owner ? owner.id : "";

  useEffect(() => {
    if (!ownerId || !path) return;
    let stop = false;
    setDiff(null);
    setError("");
    api(`${base}${encodeURIComponent(ownerId)}/gitdiff?path=${encodeURIComponent(path)}`)
      .then((d) => { if (!stop) setDiff(d); })
      .catch((e) => { if (!stop) setError(e.message || "Could not read this diff."); });
    return () => { stop = true; };
  }, [base, ownerId, path, nonce]);

  if (!path) return null;

  if (error) {
    return (
      <section className="gg-detail" aria-label="Working diff">
        <p className="gg-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={onClose}>Close</button>
        </p>
      </section>
    );
  }

  if (!diff) {
    return (
      <section className="gg-detail" aria-label="Working diff" aria-busy="true">
        <header className="gg-detail-head">
          <span className="gg-skel gg-skel-title" />
        </header>
        <div className="gg-detail-body">
          <span className="gg-skel gg-skel-line" style={{ width: "55%" }} />
          <span className="gg-skel gg-skel-line" style={{ width: "35%" }} />
        </div>
      </section>
    );
  }

  const { add, del } = countOf(diff.patch);

  return (
    <section className="gg-detail" aria-label={`Changes to ${path}`}>
      <header className="gg-detail-head">
        <h3 className="gg-detail-subject" title={diff.oldPath ? `${diff.oldPath} → ${path}` : path}>{path}</h3>
        <span className="gg-spacer" />
        <span className="gg-detail-meta">
          {diff.binary ? "binary" : <><span className="gg-add">+{add}</span> <span className="gg-del">−{del}</span></>}
        </span>
        {onOpenFile ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={() => onOpenFile(path)}>
            Open file
          </button>
        ) : null}
        <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>Close</button>
      </header>

      <div className="gg-detail-body">
        {diff.truncated ? (
          <p className="gg-warn">This diff is too large to show in full — the rest is cut off.</p>
        ) : null}
        {diff.binary ? (
          <GitAssetPreview
            base={base}
            ownerId={ownerId}
            path={path}
            oldPath={diff.oldPath}
            status={diff.status}
            fallback={<p className="diff-empty">Binary file — no text diff.</p>}
          />
        ) : (
          <div className="diff">
            {hunksFromDiff(diff.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}
          </div>
        )}
      </div>
    </section>
  );
}
