import { useEffect, useRef, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { IconArchive, IconArchiveRestore, IconChevronLeft, IconPlus, IconSearch, IconStar, IconTrash, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { applySnips, touches } from "@picode/shared/domain/feedReducers.js";
import { parseForm, snipSchema } from "@picode/shared/contracts/schemas.js";
import {
  SNIP_RESERVED, bodyLimit, clearDraft, draftToRestore, expandSnip, formFromSnip,
  insertLiteralBraces, parseSnip, readDraft, sameDraft, snipSlug, tagsFromInput, writeDraft,
} from "@picode/shared/domain/snipDraft.js";
import { snippetRoute, snippetsHash } from "../lib/routes.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";

function listURL(q, view) {
  if (q.trim()) return "/api/snips?q=" + encodeURIComponent(q.trim());
  return view === "archived" ? "/api/snips?archived=1" : "/api/snips";
}

export default function Snippets({ hidden }) {
  const [sub, setSub] = useState(() => snippetRoute());
  const [items, setItems] = useState(null);
  const [archivedCount, setArchivedCount] = useState(0);
  const [q, setQ] = useState("");
  const [view, setView] = useState("live");
  const [loadErr, setLoadErr] = useState("");
  const latest = useRef({ q: "", view: "live" });
  latest.current = { q, view };

  async function load() {
    const { q: qq, view: vv } = latest.current;
    try {
      const d = await api(listURL(qq, vv));
      if (latest.current.q !== qq || latest.current.view !== vv) return;
      setItems(d.snips || []);
      setArchivedCount(d.archived || 0);
      setLoadErr("");
    } catch (e) {
      if (latest.current.q !== qq || latest.current.view !== vv) return;
      setLoadErr(e.message || "Could not load snippets.");
    }
  }

  useEffect(() => {
    const on = () => setSub(snippetRoute());
    window.addEventListener("hashchange", on);
    return () => window.removeEventListener("hashchange", on);
  }, []);

  useEffect(() => {
    if (hidden) return;
    const t = setTimeout(load, q ? 150 : 0);
    return () => clearTimeout(t);
  }, [hidden, q, view, sub]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset") { load(); return; }
    if (!touches(ev, ["snip"])) return;
    setItems((cur) => {
      if (!cur) { load(); return cur; }
      const next = applySnips(cur, ev);
      if (next === null) { load(); return cur; }
      return next;
    });
  }), []);

  const current = sub && sub !== "new" && items ? items.find((s) => s.id === sub) : null;

  async function star(p, e) {
    if (e) e.preventDefault();
    setItems((list) => (list || []).map((x) => (x.id === p.id ? { ...x, starred: !p.starred } : x)));
    try {
      await api("/api/snips/" + encodeURIComponent(p.id) + "/starred", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ starred: !p.starred }),
      });
    } catch (err) { toastError(err); load(); }
  }

  async function archive(p, e) {
    if (e) e.preventDefault();
    try {
      await api("/api/snips/" + encodeURIComponent(p.id) + "/archived", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ archived: !p.archivedAt }),
      });
    } catch (err) { toastError(err); }
  }

  async function remove(p) {
    const ok = await askConfirm({
      title: "Delete snippet",
      message: "Delete \"" + (p.title || "this snippet") + "\"? This cannot be undone.",
      confirmLabel: "Delete",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/snips/" + encodeURIComponent(p.id), { method: "DELETE" });
      if (snippetRoute() === p.id) location.hash = snippetsHash("");
      load();
    } catch (err) { toastError(err); }
  }

  let body;
  if (sub === "new") {
    body = <Editor key="new" onCancel={() => { location.hash = snippetsHash(""); }} onSaved={(id) => { location.hash = snippetsHash(id); load(); }} />;
  } else if (sub) {
    if (items === null && !loadErr) body = <Skeleton />;
    else if (!current) body = <div className="mcp-empty"><p>That snippet is gone.</p><a className="btn btn-ghost" href={snippetsHash("")}>All snippets</a></div>;
    else body = <Editor key={current.id} initial={current} onCancel={() => { location.hash = snippetsHash(""); }} onSaved={() => { load(); }} onDelete={() => remove(current)} />;
  } else {
    body = (
      <List
        items={items}
        loadErr={loadErr}
        q={q}
        setQ={setQ}
        view={view}
        setView={setView}
        archivedCount={archivedCount}
        onStar={star}
        onArchive={archive}
        onDelete={remove}
      />
    );
  }

  return (
    <PageFrame id="snippets-view" title="Snippets" hidden={hidden}>
      {body}
    </PageFrame>
  );
}

function Skeleton() {
  return (
    <div className="mcp-skel" aria-hidden="true">
      <span className="skel-line w-70" />
      <span className="skel-line w-50" />
      <span className="skel-line w-40" />
    </div>
  );
}

function List({ items, loadErr, q, setQ, view, setView, archivedCount, onStar, onArchive, onDelete }) {
  if (items === null && !loadErr) return <Skeleton />;
  if (loadErr && items === null) {
    return <div className="mcp-empty"><p>{loadErr}</p></div>;
  }
  const searching = !!q.trim();
  const empty = searching ? "No snippets match." : view === "archived" ? "Nothing archived." : "Save a prompt you reuse.";
  return (
    <>
      <div className="auto-toolbar" data-align-row>
        <label className="snip-search">
          <IconSearch />
          <input type="search" className="dlg-input" value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search snippets" aria-label="Search snippets" onKeyDown={(e) => { if (e.key === "Escape") setQ(""); }} />
        </label>
        {view === "archived" && !searching
          ? <button type="button" className="btn btn-ghost" onClick={() => setView("live")}>Live</button>
          : (items.length > 0 || searching)
            ? <a className="btn btn-primary" href={snippetsHash("new")}><IconPlus /> New snippet</a>
            : null}
      </div>
      {items.length === 0 ? (
        <div className="mcp-empty">
          <p>{empty}</p>
          {!searching && view === "live" ? <a className="btn btn-primary" href={snippetsHash("new")}><IconPlus /> New snippet</a> : null}
        </div>
      ) : (
        <ul className="auto-list">
          {items.map((p) => (
            <li key={p.id} className="auto-row">
              <a className="auto-row-main" href={snippetsHash(p.id)}>
                <span className="auto-name">{p.title}</span>
                <span className="auto-when">{p.kind === "shell" ? "Command" : "Prompt"}{p.description ? " · " + p.description : " · /snip:" + p.slug}{p.archivedAt && searching ? " · archived" : ""}</span>
              </a>
              <div className="auto-row-actions" data-align-row>
                <button type="button" className={"btn btn-ghost" + (p.starred ? " on" : "")} title={p.starred ? "Unstar" : "Keep on top"} aria-pressed={p.starred} onClick={(e) => onStar(p, e)}><IconStar /></button>
                <button type="button" className="btn btn-ghost" title={p.archivedAt ? "Unarchive" : "Archive"} onClick={(e) => onArchive(p, e)}>{p.archivedAt ? <IconArchiveRestore /> : <IconArchive />}</button>
                <button type="button" className="btn btn-ghost btn-danger" title="Delete snippet" onClick={() => onDelete(p)}><IconX /></button>
              </div>
            </li>
          ))}
        </ul>
      )}
      {!searching && view === "live" && archivedCount > 0 ? (
        <button type="button" className="pins-archived-link" onClick={() => setView("archived")}>
          {archivedCount} archived
        </button>
      ) : null}
    </>
  );
}

function Editor({ initial, onCancel, onSaved, onDelete }) {
  const id = initial ? initial.id : "new";
  const [full, setFull] = useState(initial && initial.body != null ? initial : null);
  const [f, setF] = useState(() => formFromSnip(initial && initial.body != null ? initial : null));
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [slugLocked, setSlugLocked] = useState(!!(initial && initial.slug));
  const bodyRef = useRef(null);
  const loaded = useRef(false);
  const hydrated = useRef(false);

  useEffect(() => {
    if (initial && initial.id) {
      api("/api/snips/" + encodeURIComponent(initial.id)).then((p) => {
        setFull(p);
        const form = formFromSnip(p);
        const stored = typeof sessionStorage !== "undefined" ? readDraft(sessionStorage, p.id) : null;
        const restore = draftToRestore(stored, form);
        setF(restore ? { ...form, ...restore } : form);
        setSlugLocked(true);
        hydrated.current = true;
      }).catch((e) => setErr(e.message || "Could not load snippet."));
      return;
    }
    if (loaded.current) return;
    loaded.current = true;
    const stored = typeof sessionStorage !== "undefined" ? readDraft(sessionStorage, "new") : null;
    const restore = draftToRestore(stored, null);
    if (restore) setF(restore);
    hydrated.current = true;
  }, [initial?.id]);

  useEffect(() => {
    if (!hydrated.current || typeof sessionStorage === "undefined") return;
    writeDraft(sessionStorage, id, { ...f, slugLocked }, full ? full.updatedAt : "");
  }, [f, id, full, slugLocked]);

  function set(patch) {
    setF((cur) => {
      const next = { ...cur, ...patch };
      if (!slugLocked && patch.title != null) next.slug = snipSlug(patch.title);
      return next;
    });
  }

  const parsed = parseSnip(f.body);
  const preview = parsed.ok ? expandSnip(f.body, {}, {}) : { text: "", missing: [] };
  const cap = bodyLimit(f.body);

  async function save(e) {
    e.preventDefault();
    const checked = parseForm(snipSchema, f);
    if (!checked.ok) { setErr(checked.error); return; }
    setBusy(true);
    setErr("");
    const payload = {
      title: checked.value.title,
      slug: checked.value.slug || snipSlug(checked.value.title),
      description: checked.value.description,
      body: checked.value.body,
      tags: tagsFromInput(checked.value.tags),
      kind: full && full.kind === "shell" ? "shell" : "prompt",
    };
    if (full && full.updatedAt) payload.ifUpdatedAt = full.updatedAt;
    try {
      const p = full
        ? await api("/api/snips/" + encodeURIComponent(full.id), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) })
        : await api("/api/snips", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (typeof sessionStorage !== "undefined") clearDraft(sessionStorage, id);
      toast.ok("Saved.");
      onSaved(p.id);
    } catch (ex) {
      if (/changed elsewhere/i.test(ex.message || "") || /409/.test(String(ex.status))) {
        setErr("This snippet changed elsewhere. Reload to see the latest version.");
      } else {
        setErr(ex.message || "Could not save.");
      }
    } finally {
      setBusy(false);
    }
  }

  function insertBraces() {
    const el = bodyRef.current;
    const start = el ? el.selectionStart : f.body.length;
    const end = el ? el.selectionEnd : start;
    const next = insertLiteralBraces(f.body, start, end);
    set({ body: next });
    requestAnimationFrame(() => {
      if (!bodyRef.current) return;
      const pos = start + 4;
      bodyRef.current.focus();
      bodyRef.current.setSelectionRange(pos, pos);
    });
  }

  const dirty = full ? !sameDraft(f, formFromSnip(full)) : !sameDraft(f, formFromSnip(null));

  if (initial && initial.id && !full && !err) return <Skeleton />;

  return (
    <form className="auto-form" onSubmit={save} noValidate>
      <div className="auto-detail-head" data-align-row>
        <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}><IconChevronLeft /> All snippets</button>
        <div className="auto-detail-actions" data-align-row>
          {onDelete ? <button type="button" className="btn btn-ghost btn-danger" onClick={onDelete}><IconTrash /> Delete</button> : null}
          <button type="submit" className="btn btn-primary" disabled={busy || !dirty}>{busy ? "Saving…" : "Save"}</button>
        </div>
      </div>
      <h3 className="auto-detail-title">{full ? "Edit snippet" : "New snippet"}</h3>
      {err ? <p className="form-error" role="alert">{err}</p> : null}

      <label className="auto-field">
        <span>Title</span>
        <input className="dlg-input" value={f.title} onChange={(e) => set({ title: e.target.value })} maxLength={200} required />
      </label>
      <label className="auto-field">
        <span>Slug</span>
        <input className="dlg-input" value={f.slug} onChange={(e) => { setSlugLocked(true); set({ slug: e.target.value }); }} placeholder={snipSlug(f.title) || "review-pr"} />
        <span className="auto-hint">Used as /snip:{f.slug || snipSlug(f.title) || "name"}</span>
      </label>
      <label className="auto-field">
        <span>Description</span>
        <input className="dlg-input" value={f.description} onChange={(e) => set({ description: e.target.value })} maxLength={500} />
      </label>
      <label className="auto-field">
        <span>Tags</span>
        <input className="dlg-input" value={f.tags} onChange={(e) => set({ tags: e.target.value })} placeholder="review, git" />
      </label>
      <label className="auto-field">
        <span>Body</span>
        <textarea ref={bodyRef} className="auto-textarea snip-body" value={f.body} onChange={(e) => set({ body: e.target.value })} rows={10} />
        <div className="snip-body-bar" data-align-row>
          <button type="button" className="btn btn-ghost btn-sm" onClick={insertBraces}>Insert {"{{"}</button>
          {cap.near || cap.over ? <span className={"auto-hint" + (cap.over ? " bad" : "")}>{Math.round(cap.bytes / 1024)} / 100 KB</span> : null}
        </div>
      </label>
      {parsed.ok && parsed.placeholders.length ? (
        <div className="snip-chips">
          {parsed.placeholders.map((p) => (
            <span key={p.name} className={"snip-chip" + (SNIP_RESERVED.includes(p.name) ? " reserved" : "")}>
              {p.name}{p.optional ? "=" + (p.default || "…") : ""}{SNIP_RESERVED.includes(p.name) ? " · from the target" : ""}
            </span>
          ))}
        </div>
      ) : null}
      {parsed.ok && preview.text ? (
        <label className="auto-field">
          <span>Preview</span>
          <pre className="auto-prompt snip-preview">{preview.text}</pre>
        </label>
      ) : null}
    </form>
  );
}
