import { useEffect, useState } from "react";
import { IconX } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import { api } from "../lib/api.js";
import { go, pinRoute } from "../lib/routes.js";
import { toast, toastError } from "../lib/toast.js";

function blank() {
  return { title: "", tags: [], body: "", tagDraft: "" };
}

function pingList() {
  try { window.dispatchEvent(new Event("picode-pins")); } catch { /* ignore */ }
}

export default function PinStudio({ hidden }) {
  const info = hidden ? { mode: "", id: "" } : pinRoute();
  const [draft, setDraft] = useState(blank);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    if (hidden) return;
    if (info.mode === "new") {
      setDraft(blank());
      setLoaded(true);
      return;
    }
    if (info.mode !== "edit" || !info.id) return;
    let stop = false;
    setLoaded(false);
    api("/api/pins/" + encodeURIComponent(info.id)).then((p) => {
      if (stop) return;
      setDraft({ title: p.title || "", tags: p.tags || [], body: p.body || "", tagDraft: "" });
      setLoaded(true);
    }).catch((e) => {
      toastError(e);
      go();
    });
    return () => { stop = true; };
  }, [hidden, info.mode, info.id]);

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
      if (info.mode === "edit" && info.id) {
        await api("/api/pins/" + encodeURIComponent(info.id), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
        pingList();
        toast.ok("Pin saved.");
      } else {
        const p = await api("/api/pins", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
        pingList();
        go("pin:" + p.id);
      }
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  async function remove() {
    if (!info.id) return;
    setBusy(true);
    try {
      await api("/api/pins/" + encodeURIComponent(info.id), { method: "DELETE" });
      pingList();
      go();
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  return (
    <PageFrame id="pin-studio" title={info.mode === "edit" ? "Edit pin" : "New pin"} hidden={hidden}>
      {!hidden && loaded ? (
        <form className="pin-form pin-studio-form" onSubmit={(e) => { e.preventDefault(); save(); }}>
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
            {info.mode === "edit" ? <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={remove}>Delete</button> : null}
            <span className="pin-form-spacer" />
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => go()}>Cancel</button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
          </div>
        </form>
      ) : null}
    </PageFrame>
  );
}
