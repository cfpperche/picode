import { useEffect, useRef, useState } from "react";
import { IconChevronRight } from "../components/Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { reminderLine } from "@picode/shared/domain/pinReminder.js";
import { toastError } from "../lib/toast.js";

// Pins on the phone (More → Pins): the same list the desktop sidebar
// shows — starred first, a search over title, tags and note, archived
// found by search — and the one action the phone owns here: New.
function listURL(q) {
  return q.trim() ? "/api/pins?q=" + encodeURIComponent(q.trim()) : "/api/pins";
}

export default function PinsList({ onOpen, onNew }) {
  const [pins, setPins] = useState(null);
  const [q, setQ] = useState("");
  const latest = useRef("");
  latest.current = q;

  async function load() {
    const qq = latest.current;
    try {
      const d = await api(listURL(qq));
      if (latest.current !== qq) return;
      setPins(d.pins || []);
    } catch (e) { toastError(e); setPins([]); }
  }

  useEffect(() => {
    const t = setTimeout(load, q ? 150 : 0);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("pin."))) load();
  }), []);

  return (
    <div className="m-v2-lists m-pins" aria-label="Pins">
      <div className="m-screen-head m-list-head">
        <input type="search" className="dlg-input m-list-search" aria-label="Search pins" placeholder="Search pins" value={q} onChange={(e) => setQ(e.target.value)} />
      </div>
      {pins === null ? <p className="m-pin-msg">Loading…</p> : pins.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>{q.trim() ? "No pins match." : "No pins yet."}</p>
          {q.trim() ? <button type="button" className="btn btn-sm" onClick={() => setQ("")}>Clear search</button> : <button type="button" className="btn btn-sm btn-primary" onClick={onNew}>New pin</button>}
        </div>
      ) : (
        <section className="m-section">
          <ul className="m-list m-menu m-group-list">
            {pins.map((p) => (
              <li key={p.id} className={"m-row" + (p.starred ? " is-starred" : "")}>
                <button type="button" className="m-row-main" onClick={() => onOpen(p.id)}>
                  <span className="m-row-text">
                    <span className="m-row-title">{p.starred ? "★ " : ""}{p.title}</span>
                    <span className="m-row-sub">
                      {[p.tags && p.tags.length ? p.tags.map((t) => "#" + t).join(" ") : "", p.archivedAt ? "archived" : "", p.reminder && !p.archivedAt ? reminderLine(p.reminder) : ""].filter(Boolean).join(" · ") || (p.fileCount ? p.fileCount + " files" : "")}
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
