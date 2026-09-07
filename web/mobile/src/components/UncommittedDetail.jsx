import { useEffect, useRef, useState } from "react";
import { api, humanizeError } from "@picode/shared/client/api.js";
import { hunksFromDiff, countOf } from "@picode/shared/domain/diff.js";
import DiffLine from "./DiffLine.jsx";
import GitAssetPreview from "./GitAssetPreview.jsx";
import { IconChevronRight } from "./Icons.jsx";

// Mobile-owned read-only changes. Fetch patches on expansion through the
// existing owner-scoped endpoints; the screen header owns navigation.

export default function UncommittedDetail({ owner, onClose }) {
  const [status, setStatus] = useState(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState({});
  const [diffs, setDiffs] = useState({});
  const [loading, setLoading] = useState(true);
  const [refresh, setRefresh] = useState(0);
  const generation = useRef(0);

  // Owner kinds: agent (default), term, and — for the phone's Changes
  // screen — a workspace folder itself.
  const base = owner && owner.kind === "term" ? "/api/terminals/" : owner && owner.kind === "workspace" ? "/api/workspaces/" : "/api/agents/";
  const ownerId = owner ? owner.id : "";

  useEffect(() => () => { generation.current += 1; }, []);

  useEffect(() => {
    setLoading(true);
    setError("");
    if (!ownerId) { setError("This workspace is unavailable."); setLoading(false); return; }
    let stop = false;
    api(`${base}${encodeURIComponent(ownerId)}/gitstatus`)
      .then((s) => { if (!stop) { generation.current += 1; setStatus(s); setDiffs({}); setOpen({}); } })
      .catch((e) => { if (!stop) setError(humanizeError(e.message || "Could not read the working tree.")); })
      .finally(() => { if (!stop) setLoading(false); });
    return () => { stop = true; };
  }, [base, ownerId, refresh]);

  const loadDiff = (path) => {
    const current = generation.current;
    setDiffs((m) => ({ ...m, [path]: { loading: true } }));
    api(`${base}${encodeURIComponent(ownerId)}/gitdiff?path=${encodeURIComponent(path)}`)
      .then((d) => { if (generation.current === current) setDiffs((m) => ({ ...m, [path]: d })); })
      .catch((e) => { if (generation.current === current) setDiffs((m) => ({ ...m, [path]: { error: humanizeError(e.message || "Could not read this diff.") } })); });
  };

  const toggle = (path) => {
    const opening = !open[path];
    setOpen((o) => ({ ...o, [path]: opening }));
    if (opening && (!diffs[path] || diffs[path].error)) loadDiff(path);
  };

  if (error && !status) {
    return (
      <section className="gg-detail" aria-label="Uncommitted changes">
        <div className="m-tool-state" role="alert"><p>{error}</p><button type="button" className="btn btn-sm" onClick={() => setRefresh((n) => n + 1)}>Retry</button></div>
      </section>
    );
  }

  if (!status) {
    return (
      <section className="gg-detail" aria-label="Uncommitted changes" aria-busy="true" role="status">
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

  const changes = status.changes || [];

  return (
    <section className="gg-detail" aria-label="Uncommitted changes" aria-busy={loading}>
      <header className="gg-detail-head">
        <h3 className="gg-detail-subject">{changes.length ? `${changes.length} changed ${changes.length === 1 ? "file" : "files"}` : "Working tree"}</h3>
        <button type="button" className="btn btn-sm" disabled={loading} onClick={() => setRefresh((n) => n + 1)}>{loading ? "Refreshing…" : "Refresh"}</button>
      </header>

      <div className="gg-detail-body">
        {error ? <div className="m-tool-state" role="alert"><p>{error}</p><button type="button" className="btn btn-sm" onClick={() => setRefresh((n) => n + 1)}>Retry</button></div> : null}
        {changes.length === 0 ? (
          <div className="m-tool-state"><p>{status.git === false ? "This folder is not a Git repository." : "No uncommitted changes."}</p><button type="button" className="btn btn-sm" onClick={onClose}>Back</button></div>
        ) : (
          changes.map((c) => {
            const shown = Boolean(open[c.path]);
            const diff = diffs[c.path];
            return (
              <div key={c.path} className="gg-file">
                <button
                  type="button"
                  className="gg-file-head"
                  aria-expanded={shown}
                  onClick={() => toggle(c.path)}
                >
                  <span className={"gg-file-chev" + (shown ? " open" : "")}>
                    <IconChevronRight size={12} />
                  </span>
                  <span className="gg-file-path"><span className="m-change-filename">{c.path.split("/").pop()}</span>{c.path.includes("/") ? <span className="m-change-directory">{c.path.slice(0, c.path.lastIndexOf("/"))}</span> : null}</span>
                  <span className={"gg-file-kind gg-file-kind-" + c.kind}>{fileKind(c.kind)}</span>
                  {diff && !diff.loading && !diff.error && !diff.binary ? (
                    <FileStat patch={diff.patch} />
                  ) : null}
                </button>
                {shown ? (
                  !diff || diff.loading ? (
                    <div className="m-tool-state m-tool-loading" role="status" aria-label="Loading patch" aria-busy="true"><span className="gg-skel" /><span className="gg-skel" /><span className="gg-skel" /></div>
                  ) : diff.error ? (
                    <div className="m-tool-state" role="alert"><p>{diff.error}</p><button type="button" className="btn btn-sm" onClick={() => loadDiff(c.path)}>Retry</button></div>
                  ) : diff.binary ? (
                    <GitAssetPreview
                      base={base}
                      ownerId={ownerId}
                      path={c.path}
                      oldPath={diff.oldPath}
                      status={c.kind}
                      fallback={<p className="diff-empty">Binary file — no text diff.</p>}
                    />
                  ) : (
                    diff.patch ? <div className="diff" tabIndex={0} role="region" aria-label={`Changes in ${c.path}`}>
                      {hunksFromDiff(diff.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}
                    </div> : <p className="diff-empty">No text changes.</p>
                  )
                ) : null}
              </div>
            );
          })
        )}
      </div>
    </section>
  );
}

function fileKind(kind) {
  return ({ M: "Modified", A: "Added", D: "Deleted", R: "Renamed", C: "Copied", "?": "New", "??": "New", U: "Conflict", modified: "Modified", added: "Added", deleted: "Deleted", renamed: "Renamed", copied: "Copied", untracked: "New", conflict: "Conflict" })[kind] || kind;
}

function FileStat({ patch }) {
  const { add, del } = countOf(patch);
  return (
    <span className="gg-file-stat">
      <span className="gg-add">+{add}</span> <span className="gg-del">−{del}</span>
    </span>
  );
}
