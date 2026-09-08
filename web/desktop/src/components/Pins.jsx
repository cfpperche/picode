import { useEffect, useState } from "react";
import { IconPlus, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { go, pinRoute } from "../lib/routes.js";
import { toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

// The list follows the change feed (ADR-0048): every pin.created /
// pin.updated / pin.deleted carries the summary, so another browser's edit
// shows here without anyone navigating. `picode-pins` stays as the studio's
// same-tab nudge for when the feed is down. Nothing reloads on hashchange:
// the route only decides which card is active.
export default function Pins() {
  const [pins, setPins] = useState([]);
  const [openId, setOpenId] = useState(() => pinRoute().id);

  async function load() {
    try {
      const d = await api("/api/pins");
      setPins(d.pins || []);
    } catch (e) { toastError(e); }
  }

  useEffect(() => {
    load();
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

  return (
    <div className="side-section pins-pane">
      <div className="pins-head">
        <span className="pins-title">Pins</span>
        <button type="button" className="ws-icon-btn" title="New pin" onClick={() => go("pins-new")}><IconPlus /></button>
      </div>
      {pins.length === 0 ? (
        <p className="side-empty pins-empty">No pins</p>
      ) : (
        <ul className="pin-list">
          {pins.map((p) => (
            <li key={p.id} className={"pin-card" + (openId === p.id ? " active" : "")} onClick={() => go("pin:" + p.id)}>
              <div className="pin-card-row">
                <span className="pin-card-title">{p.title}</span>
                <button type="button" className="ws-icon-btn danger" title="Delete pin" onClick={(e) => remove(p, e)}><IconX size={12} /></button>
              </div>
              {p.tags && p.tags.length ? <div className="pin-card-tags">{p.tags.map((t) => "#" + t).join(" ")}</div> : null}
              {p.fileCount ? <div className="pin-card-files">{p.fileCount} {p.fileCount === 1 ? "file" : "files"}</div> : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
