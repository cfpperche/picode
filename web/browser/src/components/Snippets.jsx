import { useEffect, useRef, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import * as Dialog from "./ResponsiveDialog.jsx";
import { IconArchive, IconArchiveRestore, IconChevronLeft, IconCopy, IconPaste, IconPlus, IconSearch, IconStar, IconTrash, IconX } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { applySnips, touches } from "@picode/shared/domain/feedReducers.js";
import { parseForm, snipSchema } from "@picode/shared/contracts/schemas.js";
import {
  SNIP_RESERVED, applyConversions, bodyLimit, clearDraft, detectConversions, draftToRestore, expandSnip, formFromSnip,
  insertLiteralBraces, parseSnip, readDraft, sameDraft, setDefaultInBody, snipSlug, tagsFromInput, titleFromText, writeDraft,
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
  const [importing, setImporting] = useState(false);
  const [templates, setTemplates] = useState([]);
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
    if (hidden || templates.length) return;
    api("/api/snips/templates").then((d) => setTemplates(d.items || [])).catch(() => {});
  }, [hidden, templates.length]);

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
    } catch (err) { toastError(err); }
    load();
  }

  async function archive(p, e) {
    if (e) e.preventDefault();
    try {
      await api("/api/snips/" + encodeURIComponent(p.id) + "/archived", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ archived: !p.archivedAt }),
      });
    } catch (err) { toastError(err); }
    load();
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

  // One handoff for every "start me from this text" door: the editor already
  // restores a draft, so a starter and a duplicate are the same mechanism as
  // an import, with the origin recorded (ADR-0131's sibling rule: a handoff
  // is not a crash).
  // Duplicate (F6): a near-miss is cloned, not retyped. The copy is a new
  // snippet — the original is never touched — and the slug says so.
  async function duplicate(p) {
    // The list carries no body (it never does), so a duplicate from a row
    // reads the snippet first; the detail already has it.
    let src = p;
    if (src && src.id && src.body == null) {
      try { src = await api("/api/snips/" + encodeURIComponent(src.id)); } catch (e) { toastError(e); return; }
    }
    const title = ((src.title || "").trim() + " copy").trim();
    const slug = (src.slug ? src.slug + "-copy" : "").slice(0, 64);
    if (typeof sessionStorage !== "undefined") {
      writeDraft(sessionStorage, "new", {
        ...formFromSnip(null), title, slug, slugLocked: !!slug, body: src.body || "", tags: (src.tags || []).join(", "), kind: src.kind || "prompt",
      }, "", "duplicate");
    }
    location.hash = snippetsHash("new");
  }

  function startFrom(body, title, origin, extra) {
    if (typeof sessionStorage !== "undefined") {
      writeDraft(sessionStorage, "new", { ...formFromSnip(null), body, title, ...(extra || {}) }, "", origin);
    }
    location.hash = snippetsHash("new");
  }

  let body;
  if (sub === "new") {
    body = <Editor key="new" onCancel={() => { location.hash = snippetsHash(""); }} onSaved={(id) => { location.hash = snippetsHash(id); load(); }} />;
  } else if (sub) {
    if (items === null && !loadErr) body = <Skeleton />;
    else if (!current) body = <div className="mcp-empty"><p>That snippet is gone.</p><a className="btn btn-ghost" href={snippetsHash("")}>All snippets</a></div>;
    else body = <Editor key={current.id} initial={current} onCancel={() => { location.hash = snippetsHash(""); }} onSaved={() => { load(); }} onDelete={() => remove(current)} onDuplicate={duplicate} />;
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
        onImport={() => setImporting(true)}
        templates={templates}
        onStarter={(tpl) => startFrom(tpl.body, tpl.title, "starter", {
          // The slug is derived here, not at save: the reader sees the
          // address the snippet will get, and can change it before saving.
          slug: snipSlug(tpl.title), slugLocked: true, tags: (tpl.tags || []).join(", "), kind: tpl.kind,
        })}
        onDuplicate={duplicate}
      />
    );
  }

  return (
    <PageFrame id="snippets-view" title="Snippets" hidden={hidden}>
      {body}
      <ImportDialog
        open={importing}
        onClose={() => setImporting(false)}
        onOpen={(body, title) => {
          // Hand the import to the editor through the draft it already
          // restores, so the studio keeps exactly one create path.
          setImporting(false);
          startFrom(body, title, "import");
        }}
      />
    </PageFrame>
  );
}

// Import (snippets v2, F7): paste a prompt written for another tool and
// accept the conversions we recognise. Detection is a suggestion, never a
// rewrite: every kind starts on, each one can be switched off, and the
// editor opens with the result so the body is readable before anything is
// saved. Nothing here touches the API — it opens the editor.
const SUGGESTED_KEY = "picode-snippets-suggested";

// Suggested starters (Automations pattern): always open while the reader has
// nothing, a remembered <details> once they do — so an old list does not push
// the reader's own snippets below a wall of suggestions.
function Suggested({ templates, onStarter, open }) {
  const [expanded, setExpanded] = useState(() => {
    if (open) return true;
    try { return localStorage.getItem(SUGGESTED_KEY) === "1"; } catch { return false; }
  });
  const grid = (
    <ul className="auto-tpl-grid">
      {templates.map((tpl) => (
        <li key={tpl.id}>
          <button type="button" className="auto-tpl" onClick={() => onStarter(tpl)}>
            <span className="auto-tpl-name">{tpl.title}</span>
            <span className="auto-tpl-desc">{tpl.description}</span>
            <span className="auto-tpl-when">{tpl.kind === "shell" ? "Command" : "Prompt"}{(tpl.tags || []).length ? " · " + tpl.tags.join(", ") : ""}</span>
          </button>
        </li>
      ))}
    </ul>
  );
  if (open) return <section className="settings-section auto-suggested"><h3>Start from a starter</h3>{grid}</section>;
  return (
    <details className="auto-details auto-suggested" open={expanded} onToggle={(e) => {
      setExpanded(e.currentTarget.open);
      try { localStorage.setItem(SUGGESTED_KEY, e.currentTarget.open ? "1" : "0"); } catch { /* ignore */ }
    }}>
      <summary>Start from a starter</summary>
      {grid}
    </details>
  );
}

function ImportDialog({ open, onClose, onOpen }) {
  const [text, setText] = useState("");
  const [off, setOff] = useState({});
  useEffect(() => { if (!open) { setText(""); setOff({}); } }, [open]);
  const det = detectConversions(text);
  const accepted = det.kinds.filter((k) => !off[k.kind]).map((k) => k.kind);
  const out = applyConversions(text, accepted);
  const ready = !!out.trim();
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg snip-import" aria-describedby={undefined}>
          <Dialog.Title className="dlg-title">Import a prompt</Dialog.Title>
          <Dialog.Description className="sr-only">Paste a prompt, review the suggested placeholder conversions, then open it in the editor.</Dialog.Description>
          <textarea
            className="auto-textarea snip-import-body"
            value={text}
            autoFocus
            rows={8}
            aria-label="Prompt to import"
            placeholder={"Paste a prompt you already use — [BRACKETS] and UPPER_CASE become {{placeholders}}."}
            onChange={(e) => setText(e.target.value)}
          />
          {!text.trim() ? null : det.kinds.length ? (
            <div className="snip-conv">
              <div className="snip-conv-head">Placeholders found</div>
              {det.kinds.map((k) => (
                <label key={k.kind} className="snip-conv-row">
                  <input type="checkbox" checked={!off[k.kind]} aria-label={"Convert " + k.label} onChange={(e) => setOff((cur) => ({ ...cur, [k.kind]: !e.target.checked }))} />
                  <span>
                    Convert {k.count} {k.label} → <code className="snip-cell-name">{k.names.slice(0, 3).map((n) => "{{" + n + "}}").join(", ")}</code>{k.names.length > 3 ? " +" + (k.names.length - 3) : ""}
                  </span>
                </label>
              ))}
            </div>
          ) : (
            <p className="auto-hint">No placeholders found — it will open as plain text.</p>
          )}
          <div className="dlg-actions" data-align-row>
            {ready ? null : <span className="auto-hint">Paste a prompt to continue.</span>}
            <button type="button" className="btn" onClick={onClose}>Cancel</button>
            <button type="button" className="btn btn-primary" disabled={!ready} onClick={() => onOpen(out, titleFromText(out))}>Open in editor</button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
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

function List({ items, loadErr, q, setQ, view, setView, archivedCount, onStar, onArchive, onDelete, onImport, templates, onStarter, onDuplicate }) {
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
          : (
            <>
              <button type="button" className="btn" onClick={onImport}><IconPaste /> Import</button>
              {(items.length > 0 || searching) ? <a className="btn btn-primary" href={snippetsHash("new")}><IconPlus /> New snippet</a> : null}
            </>
          )}
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
                <button type="button" className="btn btn-ghost" title="Duplicate" aria-label={"Duplicate " + p.title} onClick={(e) => { e.preventDefault(); onDuplicate(p); }}><IconCopy /></button>
                <button type="button" className="btn btn-ghost" title={p.archivedAt ? "Unarchive" : "Archive"} onClick={(e) => onArchive(p, e)}>{p.archivedAt ? <IconArchiveRestore /> : <IconArchive />}</button>
                <button type="button" className="btn btn-ghost btn-danger" title="Delete snippet" onClick={() => onDelete(p)}><IconX /></button>
              </div>
            </li>
          ))}
        </ul>
      )}
      {!searching && view === "live" && templates && templates.length
        ? <Suggested templates={templates} onStarter={onStarter} open={items.length === 0} />
        : null}
      {!searching && view === "live" && archivedCount > 0 ? (
        <button type="button" className="pins-archived-link" onClick={() => setView("archived")}>
          {archivedCount} archived
        </button>
      ) : null}
    </>
  );
}

export function Editor({ initial, prefill, onCancel, onSaved, onDelete, onDuplicate, keepDraft = true, cancelLabel = "All snippets" }) {
  const id = initial ? initial.id : "new";
  const [full, setFull] = useState(initial && initial.body != null ? initial : null);
  const [f, setF] = useState(() => {
    const base = formFromSnip(initial && initial.body != null ? initial : null);
    return prefill ? { ...base, ...prefill } : base;
  });
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [slugLocked, setSlugLocked] = useState(!!(initial && initial.slug));
  const [enums, setEnums] = useState({}); // name -> enum[]; rides the draft, sent on save
  const [tryValues, setTryValues] = useState({}); // name -> sample value for Try it (session only)
  const bodyRef = useRef(null);
  const lastGood = useRef("");
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
        if (stored && stored.enums) setEnums(stored.enums);
        setSlugLocked(true);
        hydrated.current = true;
      }).catch((e) => setErr(e.message || "Could not load snippet."));
      return;
    }
    if (loaded.current) return;
    loaded.current = true;
    // keepDraft=false is a capture: the sheet owns the text and must not
    // read (or clobber) the draft of a snippet the reader is still writing
    // in the studio.
    if (!keepDraft) { hydrated.current = true; return; }
    const stored = typeof sessionStorage !== "undefined" ? readDraft(sessionStorage, "new") : null;
    const restore = draftToRestore(stored, null);
    if (restore) setF(restore);
    if (restore && restore.slugLocked) setSlugLocked(true);
    if (stored && stored.enums) setEnums(stored.enums);
    hydrated.current = true;
  }, [initial?.id]);

  useEffect(() => {
    if (!keepDraft || !hydrated.current || typeof sessionStorage === "undefined") return;
    writeDraft(sessionStorage, id, { ...f, enums, slugLocked }, full ? full.updatedAt : "");
  }, [f, id, full, slugLocked, enums, keepDraft]);

  function set(patch) {
    setF((cur) => {
      const next = { ...cur, ...patch };
      if (!slugLocked && patch.title != null) next.slug = snipSlug(patch.title);
      return next;
    });
  }

  const parsed = parseSnip(f.body);
  // While the body is invalid the table and preview fall back to the last
  // good parse, dimmed — the user never loses sight of what they had.
  const live = parsed.ok ? parsed : parseSnip(lastGood.current || "");
  const invalid = !parsed.ok;
  if (parsed.ok) lastGood.current = f.body;
  const preview = parsed.ok ? expandSnip(f.body, {}, {}) : { text: "", missing: [] };
  const tryOut = expandSnip(parsed.ok ? f.body : (lastGood.current || ""), tryValues, {});
  useEffect(() => { setTryValues({}); }, [id]);
  const cap = bodyLimit(f.body);
  const validationErr = invalid ? (
    parsed.error === "unclosed placeholder" ? "Close every placeholder — check around “" + (parsed.excerpt || f.body.slice(-24)) + "”."
      : parsed.error === "invalid placeholder name" ? "Invalid placeholder name near “" + (parsed.excerpt || "") + "”."
      : parsed.error
  ) : "";

  async function save(e) {
    e.preventDefault();
    const checked = parseForm(snipSchema, f);
    if (!checked.ok) { setErr(checked.error); return; }
    if (!parsed.ok) { setErr(validationErr); return; }
    setBusy(true);
    setErr("");
    const payload = {
      title: checked.value.title,
      slug: checked.value.slug || snipSlug(checked.value.title),
      description: checked.value.description,
      body: checked.value.body,
      tags: tagsFromInput(checked.value.tags),
      kind: f.kind === "shell" ? "shell" : "prompt",
      placeholders: (parsed.placeholders || []).map((ph) => ({ ...ph, enum: enums[ph.name] })),
    };
    if (full && full.updatedAt) payload.ifUpdatedAt = full.updatedAt;
    try {
      const p = full
        ? await api("/api/snips/" + encodeURIComponent(full.id), { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) })
        : await api("/api/snips", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (typeof sessionStorage !== "undefined" && keepDraft) clearDraft(sessionStorage, id);
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
        <button type="button" className="btn btn-ghost btn-sm" onClick={onCancel}><IconChevronLeft /> {cancelLabel}</button>
        <div className="auto-detail-actions" data-align-row>
          {onDelete ? <button type="button" className="btn btn-ghost btn-danger" onClick={onDelete}><IconTrash /> Delete</button> : null}
          {onDuplicate ? <button type="button" className="btn btn-ghost" onClick={() => onDuplicate(full || initial)}><IconCopy /> Duplicate</button> : null}
          <button type="submit" className="btn btn-primary" disabled={busy || !dirty || invalid}>{busy ? "Saving…" : "Save"}</button>
          {invalid && !busy ? <span className="auto-hint bad">Fix the placeholders first.</span> : null}
        </div>
      </div>
      <h3 className="auto-detail-title">{full ? "Edit snippet" : "New snippet"}</h3>
      {err ? <p className="form-error" role="alert">{err}</p> : null}

      <label className="auto-field">
        <span>Kind</span>
        <select className="auto-select" value={f.kind || "prompt"} onChange={(e) => set({ kind: e.target.value })} aria-label="Kind">
          <option value="prompt">Prompt</option>
          <option value="shell">Command</option>
        </select>
        <span className="auto-hint">{f.kind === "shell" ? "Pasted into a terminal shell after you confirm." : "Sent to an agent or a CLI terminal."}</span>
      </label>
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
        <textarea ref={bodyRef} className={"auto-textarea snip-body" + (invalid ? " is-invalid" : "")} value={f.body} onChange={(e) => set({ body: e.target.value })} rows={10} />
        <div className="snip-body-bar">
          <button type="button" className="btn btn-ghost btn-sm" onClick={insertBraces}>Insert {"{{"}</button>
          {cap.near || cap.over ? <span className={"auto-hint" + (cap.over ? " bad" : "")}>{Math.round(cap.bytes / 1024)} / 100 KB</span> : null}
        </div>
      </label>
      {invalid ? <p className="form-error" role="alert">{validationErr}</p> : null}
      {live && live.placeholders.length ? (
        <div className={"snip-table-wrap" + (invalid ? " is-stale" : "")}>
          <table className="snip-table">
            <thead>
              <tr><th>Placeholder</th><th>Default</th><th>Optional</th><th>Enum (comma)</th></tr>
            </thead>
            <tbody>
              {live.placeholders.map((ph) => {
                const reserved = SNIP_RESERVED.includes(ph.name);
                return (
                  <tr key={ph.name} className={reserved ? " reserved" : ""}>
                    <td>
                      <code className="snip-cell-name">{"{{"}{ph.name}{"}}"}</code>
                      {reserved ? <span className="auto-hint"> from the target</span> : null}
                    </td>
                    <td>
                      {reserved ? <span className="auto-hint">—</span> : (
                        <input
                          className="dlg-input snip-cell-input"
                          value={ph.optional ? ph.default : ""}
                          placeholder={ph.optional ? "" : "required"}
                          disabled={reserved}
                          aria-label={"Default for " + ph.name}
                          onChange={(e) => set({ body: setDefaultInBody(f.body, ph.name, e.target.value, true) })}
                        />
                      )}
                    </td>
                    <td>
                      {reserved ? <span className="auto-hint">—</span> : (
                        <input
                          type="checkbox"
                          checked={ph.optional}
                          aria-label={"Optional for " + ph.name}
                          onChange={(e) => set({ body: e.target.checked
                            ? setDefaultInBody(f.body, ph.name, ph.default, true)
                            : setDefaultInBody(f.body, ph.name, "", false) })}
                        />
                      )}
                    </td>
                    <td>
                      {reserved ? <span className="auto-hint">—</span> : (
                        <input
                          className="dlg-input snip-cell-input"
                          value={(enums[ph.name] || []).join(", ")}
                          placeholder="dev, prod"
                          aria-label={"Enum values for " + ph.name}
                          onChange={(e) => setEnums((cur) => ({ ...cur, [ph.name]: e.target.value.split(",").map((x) => x.trim()).filter(Boolean) }))}
                        />
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      ) : null}
      {!invalid && parsed.ok && !live.placeholders.length && preview.text ? (
        <label className="auto-field">
          <span>Preview</span>
          <pre className="auto-prompt snip-preview">{preview.text}</pre>
        </label>
      ) : null}
      {live && live.placeholders.length ? (
        <div className={"snip-try" + (invalid ? " is-stale" : "")} aria-label="Try it">
          <div className="snip-try-head">Try it</div>
          <div className="snip-try-grid">
            {live.placeholders.map((ph) => {
              const reserved = SNIP_RESERVED.includes(ph.name);
              const choices = enums[ph.name] || [];
              const cur = tryValues[ph.name] != null ? tryValues[ph.name] : (ph.optional ? ph.default : "");
              return (
                <label key={ph.name} className="snip-try-field">
                  <span className="snip-cell-name">{"{{"}{ph.name}{"}}"}</span>
                  {reserved ? <span className="auto-hint">filled where it runs</span> : choices.length ? (
                    <select className="dlg-input" value={cur} aria-label={"Sample value for " + ph.name} onChange={(e) => setTryValues((v) => ({ ...v, [ph.name]: e.target.value }))}>
                      <option value="">{ph.optional && ph.default ? "Default (" + ph.default + ")" : "Default"}</option>
                      {choices.map((c) => <option key={c} value={c}>{c}</option>)}
                    </select>
                  ) : (
                    <input className="dlg-input" value={cur} placeholder={ph.optional ? ph.default || "optional" : "required"} aria-label={"Sample value for " + ph.name} onChange={(e) => setTryValues((v) => ({ ...v, [ph.name]: e.target.value }))} />
                  )}
                </label>
              );
            })}
          </div>
          {tryOut.text ? <pre className="auto-prompt snip-preview">{tryOut.text}</pre> : null}
          {tryOut.missing.length ? <p className="auto-hint">Will prompt when it runs: {tryOut.missing.join(", ")}</p> : null}
        </div>
      ) : null}
    </form>
  );
}
