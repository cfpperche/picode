import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { IconArchive, IconArchiveRestore, IconClip, IconSketch, IconStar, IconX } from "./Icons.jsx";
import PageFrame from "./PageFrame.jsx";
import PinEditor from "./PinEditor.jsx";
import PinReminderPicker from "./PinReminderPicker.jsx";

const PinSketch = lazy(() => import("./PinSketch.jsx"));
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import {
  PIN_LIMITS, autoTitle, bodyLimit, clearDraft, draftToRestore, normalizeTag, pinFileSrc,
  readDraft, sameDraft, stripBackgroundFiles, writeDraft,
} from "@picode/shared/domain/pinDraft.js";
import { go, pinRoute } from "../lib/routes.js";
import { pinFileFromDrop, pinFileURL } from "../lib/pinFileDrop.js";
import { notify, toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

// The studio keeps three things apart (pins v2 review):
//   base   — the server copy it loaded (title, tags, body, updatedAt)
//   draft  — what is on screen; retained in sessionStorage while it
//            differs from base, so leaving and coming back loses nothing
//   files  — attachments, each drawn through a versioned URL so an edited
//            sketch shows its new picture instead of the cached one
// Save sends base.updatedAt as the precondition: a 409 means someone else
// wrote first, and the studio offers Reload instead of overwriting them.

function blank() {
  return { title: "", tags: [], body: "", tagDraft: "" };
}

function pingList() {
  try { window.dispatchEvent(new Event("picode-pins")); } catch { /* ignore */ }
}

function storage() {
  try { return typeof window !== "undefined" ? window.sessionStorage : null; } catch { return null; }
}

// A pin the studio created on the way to an attachment: Cancel offers to
// delete it while it still holds nothing but that attachment.
function autoKey(id) { return "picode-pin-auto:" + id; }
function markAuto(id) { try { storage().setItem(autoKey(id), "1"); } catch { /* ignore */ } }
function isAuto(id) { try { return !!id && storage().getItem(autoKey(id)) === "1"; } catch { return false; } }
function unmarkAuto(id) { try { storage().removeItem(autoKey(id)); } catch { /* ignore */ } }

function fileExt(name) {
  const m = /\.([a-z0-9]{1,8})$/i.exec(String(name || ""));
  return (m ? m[1] : "file").toUpperCase();
}

function prettySize(n) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  return (n / (1024 * 1024)).toFixed(1) + " MB";
}

function prettyKB(bytes) {
  return (bytes / 1000).toFixed(bytes >= 10_000 ? 0 : 1) + " KB";
}

async function postForm(url, fd) {
  const res = await fetch(url, { method: "POST", body: fd });
  if (!res.ok) {
    let msg = res.statusText;
    try { msg = (await res.json()).error || msg; } catch { /* keep */ }
    const err = new Error(msg);
    err.status = res.status;
    throw err;
  }
  return res.json();
}

function postFile(pinId, file) {
  const fd = new FormData();
  fd.append("file", file);
  return postForm("/api/pins/" + encodeURIComponent(pinId) + "/files", fd);
}

// Swap every reference to a file's URL in the markdown for its current
// versioned one, so an edited sketch changes picture inside the note too.
function reversion(md, pinId, f) {
  const plain = pinFileURL({ pinId, fileId: f.id });
  const re = new RegExp(plain.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + "(\\?v=[^)\\s]*)?", "g");
  return String(md || "").replace(re, pinFileSrc(pinId, f));
}

// What the server will refuse, said before the round trip.
function limitError(draft, tags) {
  if ([...draft.title.trim()].length > PIN_LIMITS.title) return "Title is too long (max " + PIN_LIMITS.title + " characters).";
  if (tags.length > PIN_LIMITS.tags) return "A pin can have at most " + PIN_LIMITS.tags + " tags.";
  const long = tags.find((t) => [...t].length > PIN_LIMITS.tag);
  if (long) return "Tag \"" + long + "\" is too long (max " + PIN_LIMITS.tag + " characters).";
  if (bodyLimit(draft.body).over) return "The note is too long (max " + prettyKB(PIN_LIMITS.bodyBytes) + ").";
  return "";
}

export default function PinStudio() {
  const info = pinRoute();
  const [draft, setDraft] = useState(blank);
  const [base, setBase] = useState(null);
  const [restored, setRestored] = useState(false);
  const [files, setFiles] = useState([]);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [drag, setDrag] = useState(false);
  const pick = useRef(null);
  const edRef = useRef(null);
  const [sketch, setSketch] = useState(null);
  const [reminder, setReminder] = useState(null);
  const [flags, setFlags] = useState({ starred: false, archivedAt: null });
  const [tick, setTick] = useState(0);
  const draftId = info.mode === "edit" ? info.id : "";

  useEffect(() => {
    if (info.mode === "new") {
      const kept = draftToRestore(readDraft(storage(), ""), null);
      setBase(null);
      setDraft(kept ? { ...kept, tagDraft: "" } : blank());
      setRestored(!!kept);
      setFiles([]);
      setReminder(null);
      setFlags({ starred: false, archivedAt: null });
      setLoaded(true);
      return;
    }
    if (info.mode !== "edit" || !info.id) return;
    let stop = false;
    setLoaded(false);
    api("/api/pins/" + encodeURIComponent(info.id)).then((p) => {
      if (stop) return;
      const server = { title: p.title || "", tags: p.tags || [], body: p.body || "", updatedAt: p.updatedAt || "" };
      const kept = draftToRestore(readDraft(storage(), p.id), server);
      setBase(server);
      setDraft({ ...(kept || server), tagDraft: "" });
      setRestored(!!kept);
      setFiles(p.files || []);
      setReminder(p.reminder || null);
      setFlags({ starred: !!p.starred, archivedAt: p.archivedAt || null });
      setLoaded(true);
      const pending = sessionStorage.getItem("picode-sketch");
      if (pending) {
        sessionStorage.removeItem("picode-sketch");
        try {
          const opt = JSON.parse(pending);
          const baseFile = opt && opt.baseFileId ? (p.files || []).find((f) => f.id === opt.baseFileId) : null;
          if (baseFile) setSketch({ source: "annotate", baseFileId: baseFile.id, backgroundURL: pinFileSrc(p.id, baseFile) });
          else setSketch({ source: "blank" });
        } catch { /* ignore */ }
      }
    }).catch((e) => {
      toastError(e);
      go();
    });
    return () => { stop = true; };
  }, [info.mode, info.id, tick]);

  // The reminder is the server's promise, not part of the draft: a change
  // made elsewhere (the phone, the engine firing a once) shows here at once.
  useEffect(() => {
    if (!info.id) return undefined;
    return subscribeFeed((ev) => {
      if (ev.type === "pin.updated" && ev.data && ev.data.id === info.id) {
        setReminder(ev.data.reminder || null);
        setFlags({ starred: !!ev.data.starred, archivedAt: ev.data.archivedAt || null });
      }
    });
  }, [info.id]);

  // Retain the draft while it differs from the server copy; drop it the
  // moment they agree again. A reload or a detour through another tab
  // comes back to the same text.
  useEffect(() => {
    if (!loaded) return;
    const cur = { title: draft.title, tags: draft.tags, body: draft.body };
    const same = base ? sameDraft(cur, base) : sameDraft(cur, { title: "", tags: [], body: "" });
    if (same) clearDraft(storage(), draftId);
    else writeDraft(storage(), draftId, cur, base ? base.updatedAt : "");
  }, [loaded, draft.title, draft.tags, draft.body, base, draftId]);

  // Closing the tab is the one exit sessionStorage does not survive.
  useEffect(() => {
    function onLeave(e) {
      if (!loaded) return;
      const cur = { title: draft.title, tags: draft.tags, body: draft.body };
      const same = base ? sameDraft(cur, base) : sameDraft(cur, { title: "", tags: [], body: "" });
      if (same) return;
      e.preventDefault();
      e.returnValue = "";
    }
    window.addEventListener("beforeunload", onLeave);
    return () => window.removeEventListener("beforeunload", onLeave);
  }, [loaded, draft.title, draft.tags, draft.body, base]);

  const fileURL = (pinId, f) => pinFileSrc(pinId, f);

  function addTag() {
    const t = normalizeTag(draft.tagDraft);
    if (!t || draft.tags.includes(t) || draft.tags.length >= PIN_LIMITS.tags) {
      setDraft({ ...draft, tagDraft: "" });
      return;
    }
    if ([...t].length > PIN_LIMITS.tag) {
      toast("Tags have at most " + PIN_LIMITS.tag + " characters.", "info");
      return;
    }
    setDraft({ ...draft, tags: [...draft.tags, t], tagDraft: "" });
  }

  // A pin created on the way to an attachment: named after the file (or
  // "Sketch"), never "Untitled", and remembered as auto-made so Cancel can
  // offer to delete it.
  async function ensurePin(incoming, fallback) {
    if (info.id) return info.id;
    const typed = draft.title.trim();
    const title = autoTitle(typed, incoming, fallback);
    const p = await api("/api/pins", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title, tags: draft.tags, body: draft.body }),
    });
    clearDraft(storage(), "");
    if (!typed) markAuto(p.id);
    return p.id;
  }

  async function startSketch(opt) {
    const intent = opt || { source: "blank" };
    if (!info.id) {
      setBusy(true);
      try {
        const id = await ensurePin([], "Sketch");
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
        const meta = files.find((f) => f.id === intent.id);
        const baseFile = meta && meta.baseFileId ? files.find((f) => f.id === meta.baseFileId) : null;
        setSketch({ id: intent.id, source: meta && meta.source === "annotate" ? "annotate" : "blank", baseFileId: baseFile ? baseFile.id : "", backgroundURL: baseFile ? fileURL(info.id, baseFile) : "", scene });
      } catch (e) { toastError(e); }
      return;
    }
    if (intent.baseFileId) {
      const baseFile = files.find((f) => f.id === intent.baseFileId);
      setSketch({ source: "annotate", baseFileId: intent.baseFileId, backgroundURL: fileURL(info.id, baseFile || { id: intent.baseFileId }) });
      return;
    }
    setSketch({ source: "blank" });
  }

  async function saveSketch({ scene, preview }) {
    if (!info.id) return;
    const fd = new FormData();
    // The annotated picture is kept by reference (baseFileId), so its
    // bytes never ride the scene and the 2 MB cap is about the drawing.
    fd.append("scene", new Blob([JSON.stringify(stripBackgroundFiles(scene))], { type: "application/json" }));
    fd.append("preview", preview, "preview.png");
    fd.append("source", sketch && sketch.source === "annotate" ? "annotate" : "blank");
    fd.append("name", "Sketch");
    if (sketch && sketch.id) fd.append("id", sketch.id);
    if (sketch && sketch.baseFileId) fd.append("baseFileId", sketch.baseFileId);
    const meta = await postForm("/api/pins/" + encodeURIComponent(info.id) + "/sketches", fd);
    const editing = !!(sketch && sketch.id);
    setFiles((cur) => cur.filter((x) => x.id !== meta.id).concat([meta]));
    const ed = edRef.current;
    if (editing) {
      // The note keeps one picture of the sketch, now at its new version.
      const md = reversion(draft.body, info.id, meta);
      if (md !== draft.body) {
        setDraft((d) => ({ ...d, body: md }));
        if (ed) ed.commands.setContent(md);
      }
    } else if (ed) {
      ed.chain().focus().setImage({ src: fileURL(info.id, meta), alt: meta.name }).run();
    }
    pingList();
    setSketch(null);
  }

  async function addFiles(list) {
    const incoming = [...(list || [])].filter(Boolean);
    if (!incoming.length) return;
    setBusy(true);
    try {
      const id = await ensurePin(incoming);
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

  // Drop the local text and fetch the server copy again (after a 409).
  function reload() {
    clearDraft(storage(), draftId);
    setRestored(false);
    setTick((t) => t + 1);
  }

  async function save() {
    const title = draft.title.trim();
    if (!title) { toast("Give the pin a title.", "info"); return; }
    let tags = draft.tags;
    const pending = normalizeTag(draft.tagDraft);
    if (pending && !tags.includes(pending)) tags = [...tags, pending];
    const refused = limitError({ ...draft, title }, tags);
    if (refused) { toast(refused, "warn"); return; }
    setBusy(true);
    try {
      const body = { title, tags, body: draft.body };
      if (info.mode === "edit" && info.id) {
        const p = await api("/api/pins/" + encodeURIComponent(info.id), {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ ...body, ifUpdatedAt: base ? base.updatedAt : "" }),
        });
        const server = { title: p.title || "", tags: p.tags || [], body: p.body || "", updatedAt: p.updatedAt || "" };
        setBase(server);
        setDraft({ ...server, tagDraft: "" });
        setRestored(false);
        clearDraft(storage(), info.id);
        unmarkAuto(info.id);
        pingList();
        toast.ok("Pin saved.");
      } else {
        const p = await api("/api/pins", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
        clearDraft(storage(), "");
        pingList();
        go("pin:" + p.id);
      }
    } catch (e) {
      if (e && e.status === 409) {
        notify({
          level: "warn",
          title: e.message || "This pin changed elsewhere.",
          body: "Your text is kept here until you save it or reload.",
          actions: [{ label: "Reload", run: reload, primary: true }],
          key: "pin-conflict:" + info.id,
        });
      } else {
        toastError(e);
      }
    } finally { setBusy(false); }
  }

  async function cancel() {
    if (info.id && isAuto(info.id) && base && sameDraft({ title: draft.title, tags: draft.tags, body: draft.body }, base)) {
      const ok = await askConfirm({
        title: "Keep this pin?",
        message: "\"" + base.title + "\" was created to hold what you attached. Delete it, or keep it as it is?",
        confirmLabel: "Delete pin",
        danger: true,
      });
      if (ok) {
        try {
          await api("/api/pins/" + encodeURIComponent(info.id), { method: "DELETE" });
          unmarkAuto(info.id);
          clearDraft(storage(), info.id);
          pingList();
        } catch (e) { toastError(e); return; }
      } else {
        unmarkAuto(info.id);
      }
    }
    go();
  }

  // Keep on top and Archive are not edits of the draft: they write at
  // once, like the reminder, and the feed echoes them back.
  async function setFlag(route, body) {
    if (!info.id) return;
    try {
      const p = await api("/api/pins/" + encodeURIComponent(info.id) + "/" + route, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
      setFlags({ starred: !!p.starred, archivedAt: p.archivedAt || null });
      pingList();
    } catch (e) { toastError(e); }
  }

  function discardRestored() {
    clearDraft(storage(), draftId);
    setDraft(base ? { ...base, tagDraft: "" } : blank());
    setRestored(false);
  }

  async function remove() {
    if (!info.id) return;
    const ok = await askConfirm({
      title: "Delete pin",
      message: "Delete \"" + (draft.title || "this pin") + "\"? This cannot be undone.",
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api("/api/pins/" + encodeURIComponent(info.id), { method: "DELETE" });
      clearDraft(storage(), info.id);
      unmarkAuto(info.id);
      pingList();
      go();
    } catch (e) { toastError(e); }
    finally { setBusy(false); }
  }

  const bodyStand = bodyLimit(draft.body);

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
          {restored ? (
            <div className="pin-restored" role="status">
              <span>Unsaved changes restored.</span>
              <button type="button" className="btn btn-ghost btn-sm" onClick={discardRestored}>Discard</button>
            </div>
          ) : null}
          <input
            className="pin-input"
            value={draft.title}
            onChange={(e) => setDraft({ ...draft, title: e.target.value })}
            placeholder="Pin title"
            aria-label="Pin title"
            maxLength={PIN_LIMITS.title}
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
              maxLength={PIN_LIMITS.tag + 1}
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
            {info.id ? <span className="pin-attach-spacer" /> : null}
            {info.id ? <PinReminderPicker pinId={info.id} reminder={reminder} onChange={setReminder} /> : null}
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
            {info.mode === "edit" ? (
              <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={() => setFlag("archived", { archived: !flags.archivedAt })}>
                {flags.archivedAt ? <IconArchiveRestore /> : <IconArchive />} {flags.archivedAt ? "Unarchive" : "Archive"}
              </button>
            ) : null}
            {info.mode === "edit" ? (
              <button type="button" className={"btn btn-ghost btn-sm pin-star-btn" + (flags.starred ? " on" : "")} aria-pressed={flags.starred} disabled={busy} onClick={() => setFlag("starred", { starred: !flags.starred })}>
                <IconStar /> {flags.starred ? "On top" : "Keep on top"}
              </button>
            ) : null}
            {flags.archivedAt ? <span className="pin-archived-note">Archived · reminders paused</span> : null}
            <span className="pin-form-spacer" />
            {bodyStand.near ? (
              <span className={"pin-limit" + (bodyStand.over ? " over" : "")} aria-live="polite">
                {prettyKB(bodyStand.bytes)} / {prettyKB(bodyStand.max)}
              </span>
            ) : null}
            <button type="button" className="btn btn-ghost btn-sm" disabled={busy} onClick={cancel}>Cancel</button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={busy || bodyStand.over}>Save</button>
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
