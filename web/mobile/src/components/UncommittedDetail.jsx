import { useEffect, useState } from "react";
import { api } from "@picode/shared/client/api.js";
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

  // Owner kinds: agent (default), term, and — for the phone's Changes
  // screen — a workspace folder itself.
  const base = owner && owner.kind === "term" ? "/api/terminals/" : owner && owner.kind === "workspace" ? "/api/workspaces/" : "/api/agents/";
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
        <h3 className="gg-detail-subject">Uncommitted changes{changes.length ? ` (${changes.length})` : ""}</h3>
      </header>

      <div className="gg-detail-body">
        {changes.length === 0 ? (
          <p className="gg-msg">No uncommitted changes.</p>
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
                    <GitAssetPreview
                      base={base}
                      ownerId={ownerId}
                      path={c.path}
                      oldPath={diff.oldPath}
                      status={c.kind}
                      fallback={<p className="diff-empty">Binary file — no text diff.</p>}
                    />
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
