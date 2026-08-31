import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { hunksFromDiff, countOf } from "../lib/diff.js";
import DiffLine from "./DiffLine.jsx";
import { IconChevronRight } from "./Icons.jsx";

// One commit, read through the owner that opened the graph (ADR-0022). The
// patch arrives already split per file; hunksFromDiff turns each one into the
// same lines the chat's edit cards use.

function when(at) {
  if (!at) return "";
  return new Date(at * 1000).toLocaleString(undefined, {
    year: "numeric", month: "short", day: "2-digit", hour: "2-digit", minute: "2-digit",
  });
}

export default function CommitDetail({ owner, hash, onClose, onSelectCommit }) {
  const [detail, setDetail] = useState(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState({});

  const base = owner && owner.kind === "term" ? "/api/terminals/" : "/api/agents/";
  const ownerId = owner ? owner.id : "";

  useEffect(() => {
    if (!ownerId || !hash) return;
    let stop = false;
    setDetail(null);
    setError("");
    setOpen({});
    api(`${base}${encodeURIComponent(ownerId)}/git/commit?hash=${encodeURIComponent(hash)}`)
      .then((d) => { if (!stop) setDetail(d); })
      .catch((e) => { if (!stop) setError(e.message || "Could not read this commit."); });
    return () => { stop = true; };
  }, [base, ownerId, hash]);

  if (!hash) return null;

  if (error) {
    return (
      <section className="gg-detail" aria-label="Commit">
        <p className="gg-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={onClose}>Close</button>
        </p>
      </section>
    );
  }

  if (!detail) {
    return (
      <section className="gg-detail" aria-label="Commit" aria-busy="true">
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

  const files = detail.files || [];

  return (
    <section className="gg-detail" aria-label={`Commit ${detail.hash.slice(0, 7)}`}>
      <header className="gg-detail-head">
        <h3 className="gg-detail-subject" title={detail.subject}>{detail.subject}</h3>
        <span className="gg-spacer" />
        <span className="gg-detail-meta">
          {detail.author} · {when(detail.at)} · <span className="gg-hash">{detail.hash.slice(0, 7)}</span>
        </span>
        <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>Close</button>
      </header>

      <div className="gg-detail-body">
        {(detail.parents || []).length > 0 ? (
          <p className="gg-parents">
            {detail.parents.length === 1 ? "Parent" : "Parents"}
            {detail.parents.map((p) =>
              onSelectCommit ? (
                <button key={p} type="button" className="gg-parent" onClick={() => onSelectCommit(p)}>
                  {p.slice(0, 7)}
                </button>
              ) : (
                <span key={p} className="gg-hash">{p.slice(0, 7)}</span>
              ),
            )}
          </p>
        ) : null}

        {detail.body ? <pre className="gg-body">{detail.body}</pre> : null}

        {detail.truncated ? (
          <p className="gg-warn">
            This commit's diff is too large to show in full — the rest is cut off.
          </p>
        ) : null}

        {files.length === 0 ? (
          <p className="gg-msg">This commit changes no files.</p>
        ) : (
          files.map((f) => {
            const shown = open[f.path] !== false;
            // Prefer git's own numstat: it counts the whole diff even when the
            // patch was capped. Counting patch lines is the fallback only.
            const { add, del } = f.add != null || f.del != null
              ? { add: f.add || 0, del: f.del || 0 }
              : countOf(f.patch);
            return (
              <div key={f.path} className="gg-file">
                <button
                  type="button"
                  className="gg-file-head"
                  aria-expanded={shown}
                  onClick={() => setOpen((o) => ({ ...o, [f.path]: !shown }))}
                >
                  <span className={"gg-file-chev" + (shown ? " open" : "")}>
                    <IconChevronRight size={12} />
                  </span>
                  <span className="gg-file-path" title={f.oldPath ? `${f.oldPath} → ${f.path}` : f.path}>
                    {f.oldPath ? <span className="gg-file-old">{f.oldPath} → </span> : null}
                    {f.path}
                  </span>
                  <span className="gg-file-stat">
                    {f.binary ? "binary" : <><span className="gg-add">+{add}</span> <span className="gg-del">−{del}</span></>}
                  </span>
                </button>
                {shown ? (
                  f.binary ? (
                    <p className="diff-empty">Binary file — no text diff.</p>
                  ) : (
                    <div className="diff">
                      {hunksFromDiff(f.patch).hunks.map((h, i) => <DiffLine key={i} h={h} />)}
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
