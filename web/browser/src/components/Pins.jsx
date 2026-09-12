import { useEffect, useRef, useState } from "react";
import { IconArchive, IconArchiveRestore, IconPlus, IconSearch, IconStar, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { reminderLine } from "@picode/shared/domain/pinReminder.js";
import { go, pinRoute } from "../lib/routes.js";
import { toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

// The Pins tab (docs/plans/pins-v2.md slice 4): a search over title, tags
// and body (live and archived alike), starred pins on top, and an archive
// out of the way but one click from view. The list follows the change
// feed (ADR-0048): every pin.* carries the summary, so another browser's
// edit shows here without anyone navigating; `picode-pins` stays as the
// studio's same-tab nudge for when the feed is down.
function listURL(q, view) {
  if (q.trim()) return "/api/pins?q=" + encodeURIComponent(q.trim());
  return view === "archived" ? "/api/pins?archived=1" : "/api/pins";
}

export default function Pins() {
  const [pins, setPins] = useState([]);
  const [archivedCount, setArchivedCount] = useState(0);
  const [q, setQ] = useState("");
  const [view, setView] = useState("live");
  const [openId, setOpenId] = useState(() => pinRoute().id);
  const latest = useRef({ q: "", view: "live" });
  latest.current = { q, view };

  async function load() {
    const { q: qq, view: vv } = latest.current;
    try {
      const d = await api(listURL(qq, vv));
      // A slower answer for an older query must not overwrite the newer list.
      if (latest.current.q !== qq || latest.current.view !== vv) return;
      setPins(d.pins || []);
      setArchivedCount(d.archived || 0);
    } catch (e) { toastError(e); }
  }

  useEffect(() => {
    const t = setTimeout(load, q ? 150 : 0);
    return () => clearTimeout(t);
  }, [q, view]);

  useEffect(() => {
    const onHash = () => setOpenId(pinRoute().id);
    const onPing = () => load();
    window.addEventListener("hashchange", onHash);
    window.addEventListener("picode-pins", onPing);
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("pin."))) load();
    });
    return () => {
      window.removeEventListener("hashchange", onHash);
      window.removeEventListener("picode-pins", onPing);
      unsub();
    };
  }, []);

  async function remove(p, e) {
    e.stopPropagation();
    const ok = await askConfirm({
      title: "Delete pin",
      message: "Delete \"" + (p.title || "this pin") + "\"? This cannot be undone.",
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/pins/" + encodeURIComponent(p.id), { method: "DELETE" });
      if (pinRoute().id === p.id) go();
      await load();
    } catch (err) { toastError(err); }
  }

  async function star(p, e) {
    e.stopPropagation();
    try {
      await api("/api/pins/" + encodeURIComponent(p.id) + "/starred", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ starred: !p.starred }) });
    } catch (err) { toastError(err); }
  }

  async function archive(p, e) {
    e.stopPropagation();
    try {
      await api("/api/pins/" + encodeURIComponent(p.id) + "/archived", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ archived: !p.archivedAt }) });
    } catch (err) { toastError(err); }
  }

  const searching = !!q.trim();
  const empty = searching ? "No pins match." : view === "archived" ? "Nothing archived." : "No pins";

  return (
    <div className="side-section pins-pane">
      <div className="pins-head">
        <span className="pins-title">{view === "archived" && !searching ? "Archived pins" : "Pins"}</span>
        {view === "archived" && !searching
          ? <button type="button" className="ws-icon-btn" title="Back to pins" onClick={() => setView("live")}><IconX size={12} /></button>
          : <button type="button" className="ws-icon-btn" title="New pin" onClick={() => go("pins-new")}><IconPlus /></button>}
      </div>
      <label className="pins-search">
        <IconSearch />
        <input
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search pins"
          aria-label="Search pins"
          onKeyDown={(e) => { if (e.key === "Escape") setQ(""); }}
        />
      </label>
      {pins.length === 0 ? (
        <p className="side-empty pins-empty">{empty}</p>
      ) : (
        <ul className="pin-list">
          {pins.map((p) => (
            <li key={p.id} className={"pin-card" + (openId === p.id ? " active" : "") + (p.starred ? " starred" : "") + (p.archivedAt ? " archived" : "")} onClick={() => go("pin:" + p.id)}>
              <div className="pin-card-row">
                <span className="pin-card-title">{p.title}</span>
                <span className="pin-card-acts">
                  <button type="button" className={"ws-icon-btn pin-star" + (p.starred ? " on" : "")} title={p.starred ? "Unstar" : "Keep on top"} aria-pressed={p.starred} onClick={(e) => star(p, e)}><IconStar size={12} /></button>
                  <button type="button" className="ws-icon-btn" title={p.archivedAt ? "Unarchive" : "Archive"} onClick={(e) => archive(p, e)}>{p.archivedAt ? <IconArchiveRestore size={12} /> : <IconArchive size={12} />}</button>
                  <button type="button" className="ws-icon-btn danger" title="Delete pin" onClick={(e) => remove(p, e)}><IconX size={12} /></button>
                </span>
              </div>
              {p.tags && p.tags.length ? <div className="pin-card-tags">{p.tags.map((t) => "#" + t).join(" ")}</div> : null}
              {p.fileCount ? <div className="pin-card-files">{p.fileCount} {p.fileCount === 1 ? "file" : "files"}</div> : null}
              {p.reminder && !p.archivedAt ? <div className="pin-card-remind">{reminderLine(p.reminder)}</div> : null}
              {p.archivedAt && searching ? <div className="pin-card-archived">archived</div> : null}
            </li>
          ))}
        </ul>
      )}
      {!searching && view === "live" && archivedCount > 0 ? (
        <button type="button" className="pins-archived-link" onClick={() => setView("archived")}>
          {archivedCount} archived
        </button>
      ) : null}
    </div>
  );
}
