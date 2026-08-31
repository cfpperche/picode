import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { hunksFromDiff, countOf } from "../lib/diff.js";
import DiffLine from "./DiffLine.jsx";
import { IconChevronRight } from "./Icons.jsx";

// The dirty working tree behind the graph's Uncommitted Changes row
// (ADR-0038). The file list comes from gitstatus and each patch is fetched
// lazily from gitdiff on expand — the same owner-scoped endpoints the file
// tree's diff reads. Read-only, like everything else in the graph.

export default function UncommittedDetail({ owner, onClose }) {
  const [status, setStatus] = useState(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState({});
  const [diffs, setDiffs] = useState({});

  const base = owner && owner.kind === "term" ? "/api/terminals/" : "/api/agents/";
  const ownerId = owner ? owner.id : "";

  useEffect(() => {
    if (!ownerId) return;
    let stop = false;
    api(`${base}${encodeURIComponent(ownerId)}/gitstatus`)
      .then((s) => { if (!stop) setStatus(s); })
      .catch((e) => { if (!stop) setError(e.message || "Could not read the working tree."); });
    return () => { stop = true; };
  }, [base, ownerId]);

  const toggle = (path) => {
    const opening = !open[path];
    setOpen((o) => ({ ...o, [path]: opening }));
    if (opening && !diffs[path]) {
      api(`${base}${encodeURIComponent(ownerId)}/gitdiff?path=${encodeURIComponent(path)}`)
        .then((d) => setDiffs((m) => ({ ...m, [path]: d })))
        .catch((e) => setDiffs((m) => ({ ...m, [path]: { error: e.message || "Could not read this diff." } })));
    }
  };

  if (error) {
    return (
      <section className="gg-detail" aria-label="Uncommitted changes">
        <p className="gg-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={onClose}>Close</button>
        </p>
      </section>
    );
  }

  if (!status) {
    return (
      <section className="gg-detail" aria-label="Uncommitted changes" aria-busy="true">
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
    <section className="gg-detail" aria-label="Uncommitted changes">
      <header className="gg-detail-head">
        <h3 className="gg-detail-subject">Uncommitted Changes ({changes.length})</h3>
        <span className="gg-spacer" />
        <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>Close</button>
      </header>

      <div className="gg-detail-body">
        {changes.length === 0 ? (
          <p className="gg-msg">The working tree is clean now — Refresh the graph.</p>
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
                  <span className="gg-file-path" title={c.path}>{c.path}</span>
                  <span className={"gg-file-kind gg-file-kind-" + c.kind}>{c.kind}</span>
                  {diff && !diff.error && !diff.binary ? (
                    <FileStat patch={diff.patch} />
                  ) : null}
                </button>
                {shown ? (
                  !diff ? (
                    <p className="diff-empty">Loading…</p>
                  ) : diff.error ? (
                    <p className="diff-empty">{diff.error}</p>
                  ) : diff.binary ? (
                    <p className="diff-empty">Binary file — no text diff.</p>
                  ) : (
                    <div className="diff">
                      {hunksFromDiff(diff.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}
                    </div>
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

function FileStat({ patch }) {
  const { add, del } = countOf(patch);
  return (
    <span className="gg-file-stat">
      <span className="gg-add">+{add}</span> <span className="gg-del">−{del}</span>
    </span>
  );
}
