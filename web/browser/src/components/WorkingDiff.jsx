import { useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { hunksFromDiff, countOf } from "@picode/shared/domain/diff.js";
import { treeApiBase } from "../lib/fileTree.js";
import { ownerFileURL } from "../lib/fileIO.js";
import DiffLine from "./DiffLine.jsx";
import GitAssetPreview from "./GitAssetPreview.jsx";

export default function WorkingDiff({ owner, path, root = "", nonce, onClose, onOpenFile }) {
  const [result, setResult] = useState(null);
  const [failure, setFailure] = useState(null);
  const [retry, setRetry] = useState(0);
  const base = treeApiBase(owner?.kind);
  const ownerId = owner?.id || "";
  const key = JSON.stringify([base, ownerId, root, path]);
  const diff = result?.key === key ? result.diff : null;
  const error = failure?.key === key ? failure.message : "";
  const clean = error === "no difference for this path";

  useEffect(() => {
    if (!ownerId || !path) return;
    const request = new AbortController();
    setFailure(null);
    api(ownerFileURL(owner, "gitdiff", path, root), { signal: request.signal })
      .then((data) => { if (!request.signal.aborted) setResult({ key, diff: data }); })
      .catch((e) => {
        if (!request.signal.aborted) {
          setFailure({ key, message: e.message || "Could not read this diff." });
          if (e.message === "no difference for this path") setResult(null);
        }
      });
    return () => request.abort();
  }, [key, nonce, retry]); // key includes owner, root and path

  if (!path) return null;
  const { add, del } = countOf(diff?.patch || "");
  return (
    <section className="gg-detail" aria-label={`Changes to ${path}`} aria-busy={!diff && !error}>
      <header className="gg-detail-head">
        <h3 className="gg-detail-subject" title={diff?.oldPath ? `${diff.oldPath} → ${path}` : path}>{path}</h3>
        <span className="gg-spacer" />
        <div className="ft-diff-actions" data-align-row>
          {diff ? <span className="gg-detail-meta">{diff.binary ? "binary" : <><span className="gg-add">+{add}</span> <span className="gg-del">−{del}</span></>}</span> : null}
          {onOpenFile && diff?.status !== "deleted" ? <button type="button" className="btn btn-sm btn-ghost" onClick={() => onOpenFile(path)}>Open file</button> : null}
          {onClose ? <button type="button" className="btn btn-sm btn-ghost" onClick={onClose} aria-label="Close diff panel">Close</button> : null}
        </div>
      </header>
      {error ? <p className="file-pane-notice" role="status"><span>{clean ? "No changes in this file." : error}</span>{!clean ? <button type="button" className="btn btn-sm" onClick={() => setRetry((n) => n + 1)}>Try again</button> : null}</p> : null}
      <div className="gg-detail-body">
        {!diff && !error ? <div className="file-skel"><span className="skel-line w-80" /><span className="skel-line w-50" /></div> : null}
        {diff?.truncated ? <p className="gg-warn">This diff is too large to show in full — the rest is cut off.</p> : null}
        {diff?.binary ? (
          <GitAssetPreview base={base} ownerId={ownerId} path={path} root={root} oldPath={diff.oldPath} status={diff.status} fallback={<p className="diff-empty">Binary file — no text diff.</p>} />
        ) : diff ? <div className="diff">{hunksFromDiff(diff.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}</div> : null}
      </div>
    </section>
  );
}
