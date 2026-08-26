import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { IconX, IconClip, IconSketch } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import PinEditor from "./PinEditor.jsx";

const PinSketch = lazy(() => import("./PinSketch.jsx"));
import { api } from "../lib/api.js";
import { go, pinRoute } from "../lib/routes.js";
import { pinFileFromDrop, pinFileURL } from "../lib/pinFileDrop.js";
import { toast, toastError } from "../lib/toast.js";

function blank() {
  return { title: "", tags: [], body: "", tagDraft: "" };
}

function pingList() {
  try { window.dispatchEvent(new Event("picode-pins")); } catch { /* ignore */ }
}

function fileURL(pinId, f) {
  return "/api/pins/" + encodeURIComponent(pinId) + "/files/" + encodeURIComponent(f.id);
}

function fileExt(name) {
  const m = /\.([a-z0-9]{1,8})$/i.exec(String(name || ""));
  return (m ? m[1] : "file").toUpperCase();
}

function prettySize(n) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  return (n / (1024 * 1024)).toFixed(1) + " MB";
}

async function postFile(pinId, file) {
  const fd = new FormData();
  fd.append("file", file);
  const res = await fetch("/api/pins/" + encodeURIComponent(pinId) + "/files", { method: "POST", body: fd });
  if (!res.ok) {
    let msg = res.statusText;
    try { msg = (await res.json()).error || msg; } catch { /* keep */ }
    throw new Error(msg);
  }
  return res.json();
}

export default function PinStudio() {
  const info = pinRoute();
  const [draft, setDraft] = useState(blank);
  const [files, setFiles] = useState([]);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [drag, setDrag] = useState(false);
  const pick = useRef(null);
  const edRef = useRef(null);
  const [sketch, setSketch] = useState(null);

  useEffect(() => {
    if (info.mode === "new") {
      setDraft(blank());
      setFiles([]);
      setLoaded(true);
      return;
    }
    if (info.mode !== "edit" || !info.id) return;
    let stop = false;
    setLoaded(false);
    api("/api/pins/" + encodeURIComponent(info.id)).then((p) => {
      if (stop) return;
      setDraft({ title: p.title || "", tags: p.tags || [], body: p.body || "", tagDraft: "" });
      setFiles(p.files || []);
      setLoaded(true);
      const pending = sessionStorage.getItem("picode-sketch");
      if (pending) {
        sessionStorage.removeItem("picode-sketch");
        try {
          const opt = JSON.parse(pending);
          if (opt && opt.baseFileId) setSketch({ source: "annotate", baseFileId: opt.baseFileId, backgroundURL: fileURL(p.id, { id: opt.baseFileId }) });
          else setSketch({ source: "blank" });
        } catch { /* ignore */ }
      }
    }).catch((e) => {
      toastError(e);
      go();
    });
    return () => { stop = true; };
  }, [info.mode, info.id]);

  function addTag() {
    const t = draft.tagDraft.trim().replace(/^#/, "").toLowerCase().replace(/\s+/g, "-");
    if (!t || draft.tags.includes(t) || draft.tags.length >= 16) {
      setDraft({ ...draft, tagDraft: "" });
      return;
    }
    setDraft({ ...draft, tags: [...draft.tags, t], tagDraft: "" });
  }

  async function ensurePin() {
    if (info.id) return info.id;
    const title = draft.title.trim() || "Untitled";
    const p = await api("/api/pins", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title, tags: draft.tags, body: draft.body }),
    });
    return p.id;
  }

  async function startSketch(opt) {
    const intent = opt || { source: "blank" };
    if (!info.id) {
      setBusy(true);
      try {
        const id = await ensurePin();
        sessionStorage.setItem("picode-sketch", JSON.stringify(intent));
        pingList();
        go("pin:" + id);
      } catch (e) { toastError(e); }
      finally { setBusy(false); }
      return;
    }
    if (intent.id) {
      try {
        const scene = await fetch("/api/pins/" + encodeURIComponent(info.id) + "/files/" + encodeURIComponent(intent.id) + "/scene").then((r) => {
          if (!r.ok) throw new Error("Could not open sketch");
          return r.json();
        });
        setSketch({ id: intent.id, source: "blank", scene });
      } catch (e) { toastError(e); }
      return;
    }
    if (intent.baseFileId) {
      setSketch({ source: "annotate", baseFileId: intent.baseFileId, backgroundURL: fileURL(info.id, { id: intent.baseFileId }) });
      return;
    }
    setSketch({ source: "blank" });
  }

  async function saveSketch({ scene, preview }) {
    if (!info.id) return;
    const fd = new FormData();
    fd.append("scene", new Blob([JSON.stringify(scene)], { type: "application/json" }));
    fd.append("preview", preview, "preview.png");
    fd.append("source", sketch && sketch.source === "annotate" ? "annotate" : "blank");
    fd.append("name", "Sketch");
    if (sketch && sketch.id) fd.append("id", sketch.id);
    if (sketch && sketch.baseFileId) fd.append("baseFileId", sketch.baseFileId);
    const res = await fetch("/api/pins/" + encodeURIComponent(info.id) + "/sketches", { method: "POST", body: fd });
    if (!res.ok) {
      let msg = res.statusText;
      try { msg = (await res.json()).error || msg; } catch { /* keep */ }
      throw new Error(msg);
    }
    const meta = await res.json();
    setFiles((cur) => {
      const rest = cur.filter((x) => x.id !== meta.id);
      return rest.concat([meta]);
    });
    const ed = edRef.current;
    if (ed) ed.chain().focus().setImage({ src: fileURL(info.id, meta), alt: meta.name }).run();
    pingList();
    setSketch(null);
  }

  async function addFiles(list) {
    const incoming = [...(list || [])].filter(Boolean);
    if (!incoming.length) return;
    setBusy(true);
    try {
      const id = await ensurePin();
      const added = [];
      for (const file of incoming) {
        added.push(await postFile(id, file));
      }
      if (!info.id) {
        pingList();
        go("pin:" + id);
        return;
      }
      setFiles((cur) => cur.concat(added));
      const ed = edRef.current;
      if (ed) {
        for (const f of added) {
          if (f.kind !== "image") continue;
          ed.chain().focus().setImage({ src: fileURL(id, f), alt: f.name }).run();
        }
      }
      pingList();
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  function insertRef(f) {
    if (!info.id) return;
    const url = fileURL(info.id, f);
    const ed = edRef.current;
    if (ed && (f.kind === "image" || f.kind === "sketch")) { ed.chain().focus().setImage({ src: url, alt: f.name }).run(); return; }
    if (ed) { ed.chain().focus().insertContent('<p><a href="' + url + '">' + f.name + "</a></p>").run(); return; }
  }

  async function dropFile(f) {
    if (!info.id) return;
    try {
      await api("/api/pins/" + encodeURIComponent(info.id) + "/files/" + encodeURIComponent(f.id), { method: "DELETE" });
      setFiles((cur) => cur.filter((x) => x.id !== f.id));
      pingList();
    } catch (e) { toastError(e); }
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
    <PageFrame id="pin-studio" title={info.mode === "edit" ? "Edit pin" : "New pin"}>
      {loaded ? (
        <form
          className={"pin-form pin-studio-form" + (drag ? " pin-drop" : "")}
          onSubmit={(e) => { e.preventDefault(); save(); }}
          onPaste={(e) => {
            const items = [...(e.clipboardData && e.clipboardData.files ? e.clipboardData.files : [])];
            if (!items.length) return;
            e.preventDefault();
            addFiles(items);
          }}
          onDragOver={(e) => { e.preventDefault(); setDrag(true); }}
          onDragLeave={() => setDrag(false)}
          onDrop={(e) => {
            e.preventDefault();
            setDrag(false);
            const existing = pinFileFromDrop(e);
            if (existing) {
              const url = pinFileURL(existing);
              const ed = edRef.current;
              if (ed && url) ed.chain().focus().setImage({ src: url, alt: existing.name || "" }).run();
              return;
            }
            addFiles(e.dataTransfer && e.dataTransfer.files);
          }}
        >
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

          <div className="pin-attach-bar">
            <input ref={pick} type="file" multiple hidden onChange={(e) => { addFiles(e.target.files); e.target.value = ""; }} />
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => pick.current && pick.current.click()}>
              <IconClip /> Attach
            </button>
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => startSketch({ source: "blank" })}>
              <IconSketch /> Sketch
            </button>
            <span className="pin-attach-hint">Paste, drop, or draw</span>
          </div>

          {files.length ? (
            <ul className="pin-gallery">
              {files.map((f) => (
                <li key={f.id} className={"pin-att pin-att-" + f.kind}>
                  <button type="button" className="pin-att-x" title="Remove file" onClick={() => dropFile(f)}><IconX size={12} /></button>
                  {f.kind === "image"
                    ? <button type="button" className="pin-att-draw" title="Annotate" onClick={() => startSketch({ baseFileId: f.id })}><IconSketch size={12} /></button>
                    : null}
                  <button
                    type="button"
                    className="pin-att-face"
                    title={f.kind === "sketch" ? "Edit sketch" : "Insert in text"}
                    draggable
                    onDragStart={(e) => {
                      const ref = { pinId: info.id, fileId: f.id, name: f.name, kind: f.kind };
                      e.dataTransfer.setData("application/x-picode-pin-file", JSON.stringify(ref));
                      e.dataTransfer.setData("text/uri-list", fileURL(info.id, f));
                      e.dataTransfer.effectAllowed = "copy";
                    }}
                    onClick={() => f.kind === "sketch" ? startSketch({ id: f.id }) : insertRef(f)}
                  >
                    {f.kind === "image" || f.kind === "sketch"
                      ? <img src={info.id ? fileURL(info.id, f) : ""} alt="" />
                      : <span className="pin-att-ext">{fileExt(f.name)}</span>}
                    {f.kind === "sketch" ? <span className="pin-att-badge">SKETCH</span> : null}
                  </button>
                  <div className="pin-att-meta">
                    <span className="pin-att-name" title={f.name}>{f.name}</span>
                    <span className="pin-att-size">{prettySize(f.size)}</span>
                  </div>
                </li>
              ))}
            </ul>
          ) : null}

          <PinEditor
            pinId={info.id || "new"}
            markdown={draft.body}
            onMarkdown={(md) => setDraft((d) => ({ ...d, body: md }))}
            onFiles={addFiles}
            onReady={(ed) => { edRef.current = ed; }}
          />
          <div className="pin-form-actions">
            {info.mode === "edit" ? <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={remove}>Delete</button> : null}
            <span className="pin-form-spacer" />
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => go()}>Cancel</button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={busy}>Save</button>
          </div>
        </form>
      ) : null}
      {sketch ? (
        <Suspense fallback={null}>
          <PinSketch
            open
            title={sketch.id ? "Edit sketch" : (sketch.source === "annotate" ? "Annotate" : "New sketch")}
            initial={sketch.scene || null}
            backgroundURL={sketch.backgroundURL || ""}
            onSave={saveSketch}
            onClose={() => setSketch(null)}
          />
        </Suspense>
      ) : null}
    </PageFrame>
  );
}
