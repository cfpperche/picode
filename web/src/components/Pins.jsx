import { useEffect, useState } from "react";
import { IconPlus, IconX } from "./Icons.jsx";
import { api } from "../lib/api.js";
import { go, pinRoute } from "../lib/routes.js";
import { toastError } from "../lib/toast.js";

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
    const on = () => { load(); setOpenId(pinRoute().id); };
    window.addEventListener("hashchange", on);
    window.addEventListener("picode-pins", on);
    return () => {
      window.removeEventListener("hashchange", on);
      window.removeEventListener("picode-pins", on);
    };
  }, []);

  async function remove(id, e) {
    e.stopPropagation();
    try {
      await api("/api/pins/" + encodeURIComponent(id), { method: "DELETE" });
      if (pinRoute().id === id) go();
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
                <button type="button" className="ws-icon-btn danger" title="Delete pin" onClick={(e) => remove(p.id, e)}><IconX size={12} /></button>
              </div>
              {p.tags && p.tags.length ? <div className="pin-card-tags">{p.tags.map((t) => "#" + t).join(" ")}</div> : null}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
