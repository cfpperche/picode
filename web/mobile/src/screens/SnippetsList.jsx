import { useEffect, useRef, useState } from "react";
import { IconChevronRight } from "../components/Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { toastError } from "../lib/toast.js";

// Snippets on the phone (More → Snippets): the same list the desktop
// studio shows — starred first, a search over title, tags and body —
// and the one action the phone owns here: New.
function listURL(q) {
  return q.trim() ? "/api/snips?q=" + encodeURIComponent(q.trim()) : "/api/snips";
}

export default function SnippetsList({ onOpen, onNew }) {
  const [snips, setSnips] = useState(null);
  const [q, setQ] = useState("");
  const latest = useRef("");
  latest.current = q;

  async function load() {
    const qq = latest.current;
    try {
      const d = await api(listURL(qq));
      if (latest.current !== qq) return;
      setSnips(d.snips || []);
    } catch (e) { toastError(e); setSnips([]); }
  }

  useEffect(() => {
    const t = setTimeout(load, q ? 150 : 0);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("snip."))) load();
  }), []);

  return (
    <div className="m-v2-lists m-pins" aria-label="Snippets">
      <div className="m-screen-head m-list-head">
        <input type="search" className="dlg-input m-list-search" aria-label="Search snippets" placeholder="Search snippets" value={q} onChange={(e) => setQ(e.target.value)} />
      </div>
      {snips === null ? <p className="m-pin-msg">Loading…</p> : snips.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>{q.trim() ? "No snippets match." : "Save a prompt you reuse."}</p>
          {q.trim() ? <button type="button" className="btn btn-sm" onClick={() => setQ("")}>Clear search</button> : <button type="button" className="btn btn-sm btn-primary" onClick={onNew}>New snippet</button>}
        </div>
      ) : (
        <section className="m-section">
          <ul className="m-list m-menu m-group-list">
            {snips.map((s) => (
              <li key={s.id} className={"m-row" + (s.starred ? " is-starred" : "")}>
                <button type="button" className="m-row-main" onClick={() => onOpen(s.id)}>
                  <span className="m-row-text">
                    <span className="m-row-title">{s.starred ? "★ " : ""}{s.title}</span>
                    <span className="m-row-sub">
                      {[s.kind === "shell" ? "Command" : "Prompt", "/snip:" + s.slug, s.archivedAt ? "archived" : ""].filter(Boolean).join(" · ")}
                    </span>
                  </span>
                  <IconChevronRight size={16} className="m-row-chev" />
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
