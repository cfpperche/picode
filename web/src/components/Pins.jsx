import { useEffect, useState } from "react";
import { IconPlus, IconX } from "./Icons.jsx";
import { api } from "../lib/api.js";
import { toastError, toast } from "../lib/toast.js";

function blank() {
  return { id: null, title: "", tags: [], body: "", tagDraft: "" };
}

export default function Pins() {
  const [pins, setPins] = useState([]);
  const [draft, setDraft] = useState(null);
  const [busy, setBusy] = useState(false);

  async function load() {
    try {
      const d = await api("/api/pins");
      setPins(d.pins || []);
    } catch (e) { toastError(e); }
  }
  useEffect(() => { load(); }, []);

  function dirty() {
    if (!draft) return false;
    return !!(draft.title.trim() || draft.tags.length || draft.body.trim() || draft.tagDraft.trim());
  }

  function startNew() {
    if (draft && dirty()) { toast("Save or cancel first.", "info"); return; }
    setDraft(blank());
  }

  function open(p) {
    if (draft && dirty() && draft.id !== p.id) { toast("Save or cancel first.", "info"); return; }
    setDraft({ id: p.id, title: p.title, tags: p.tags || [], body: p.body || "", tagDraft: "" });
  }

  function addTag() {
    const t = draft.tagDraft.trim().replace(/^#/, "").toLowerCase().replace(/\s+/g, "-");
    if (!t || draft.tags.includes(t) || draft.tags.length >= 16) {
      setDraft({ ...draft, tagDraft: "" });
      return;
    }
    setDraft({ ...draft, tags: [...draft.tags, t], tagDraft: "" });
  }

  async function save() {
    const title = draft.title.trim();
    if (!title) { toast("Give the pin a title.", "info"); return; }
    let tags = draft.tags;
    const pending = draft.tagDraft.trim().replace(/^#/, "").toLowerCase().replace(/\s+/g, "-");
    if (pending && !tags.includes(pending)) tags = [...tags, pending];
    setBusy(true);
    try {
      const body = { title, tags, body: draft.body };
      if (draft.id) await api("/api/pins/" + draft.id, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      else await api("/api/pins", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      setDraft(null);
      await load();
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  async function remove(id, e) {
    if (e) e.stopPropagation();
    setBusy(true);
    try {
      await api("/api/pins/" + id, { method: "DELETE" });
      if (draft && draft.id === id) setDraft(null);
      await load();
    } catch (err) { toastError(err); }
    finally { setBusy(false); }
  }

  return (
    <div className="side-section pins-pane">
      <div className="pins-head">
        <span className="pins-title">Pins</span>
        <button type="button" className="ws-icon-btn" title="New pin" onClick={startNew}><IconPlus /></button>
      </div>

      {draft ? (
        <form className="pin-form" onSubmit={(e) => { e.preventDefault(); save(); }}>
          <input
            className="pin-input"
            value={draft.title}
            onChange={(e) => setDraft({ ...draft, title: e.target.value })}
            placeholder="Pin title"
            aria-label="Pin title"
            autoFocus
          />
          <div className="pin-tags" aria-label="Pin tags">
            {draft.tags.map((t) => (
              <button type="button" key={t} className="pin-tag" title={"Remove " + t} onClick={() => setDraft({ ...draft, tags: draft.tags.filter((x) => x !== t) })}>
                #{t}<IconX size={10} />
              </button>
            ))}
            <input
              className="pin-tag-input"
              value={draft.tagDraft}
              onChange={(e) => setDraft({ ...draft, tagDraft: e.target.value })}
              onBlur={addTag}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === ",") { e.preventDefault(); addTag(); }
                if (e.key === "Backspace" && !draft.tagDraft) setDraft({ ...draft, tags: draft.tags.slice(0, -1) });
              }}
              placeholder={draft.tags.length ? "Add tag" : "Add tags"}
              aria-label="Add pin tag"
            />
          </div>
          <textarea
            className="pin-body"
            value={draft.body}
            onChange={(e) => setDraft({ ...draft, body: e.target.value })}
            placeholder="Write in markdown…"
            aria-label="Pin body"
          />
          <div className="pin-form-actions">
            {draft.id ? <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => remove(draft.id)}>Delete</button> : null}
            <span className="pin-form-spacer" />
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => setDraft(null)}>Cancel</button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
          </div>
        </form>
      ) : pins.length === 0 ? (
        <p className="side-empty pins-empty">No pins</p>
      ) : (
        <ul className="pin-list">
          {pins.map((p) => (
            <li key={p.id} className="pin-card" onClick={() => open(p)}>
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
