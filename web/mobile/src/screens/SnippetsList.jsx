import { useEffect, useRef, useState } from "react";
import * as Sheet from "../components/MobileSheet.jsx";
import { IconChevronRight, IconPaste } from "../components/Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { toastError } from "../lib/toast.js";
import { applyConversions, detectConversions, formFromSnip, titleFromText, writeDraft } from "@picode/shared/domain/snipDraft.js";

// Snippets on the phone (More → Snippets): the same list the desktop
// studio shows — starred first, a search over title, tags and body —
// and the one action the phone owns here: New.
function listURL(q) {
  return q.trim() ? "/api/snips?q=" + encodeURIComponent(q.trim()) : "/api/snips";
}

function storage() {
  try { return typeof window !== "undefined" ? window.sessionStorage : null; } catch { return null; }
}

// Import (snippets v2, F7) on the phone: same detection, same promise —
// suggestions you can switch off, and the editor opens with the result.
// The handoff is the draft the editor already restores.
function ImportSheet({ open, onClose, onOpen }) {
  const [text, setText] = useState("");
  const [off, setOff] = useState({});
  useEffect(() => { if (!open) { setText(""); setOff({}); } }, [open]);
  const det = detectConversions(text);
  const accepted = det.kinds.filter((k) => !off[k.kind]).map((k) => k.kind);
  const out = applyConversions(text, accepted);
  return (
    <Sheet.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Sheet.Portal>
        <Sheet.Overlay className="dlg-overlay" />
        <Sheet.Content className="dlg m-snip-import" aria-describedby={undefined}>
          <Sheet.Title className="dlg-title">Import a prompt</Sheet.Title>
          <textarea
            className="auto-textarea snip-import-body"
            value={text}
            rows={7}
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
            {out.trim() ? null : <span className="auto-hint">Paste a prompt to continue.</span>}
            <Sheet.Close asChild><button type="button" className="btn">Cancel</button></Sheet.Close>
            <button type="button" className="btn btn-primary" disabled={!out.trim()} onClick={() => onOpen(out)}>Open in editor</button>
          </div>
        </Sheet.Content>
      </Sheet.Portal>
    </Sheet.Root>
  );
}

export default function SnippetsList({ onOpen, onNew }) {
  const [snips, setSnips] = useState(null);
  const [q, setQ] = useState("");
  const [importing, setImporting] = useState(false);
  const latest = useRef("");
  latest.current = q;

  async function load() {
    const qq = latest.current;
    try {
      const d = await api(listURL(qq));
      if (latest.current !== qq) return;
      setSnips(d.snips || []);
    } catch (e) { toastError(e); setSnips([]); }
  }

  useEffect(() => {
    const t = setTimeout(load, q ? 150 : 0);
    return () => clearTimeout(t);
  }, [q]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("snip."))) load();
  }), []);

  return (
    <div className="m-v2-lists m-pins" aria-label="Snippets">
      <div className="m-screen-head m-list-head">
        <input type="search" className="dlg-input m-list-search" aria-label="Search snippets" placeholder="Search snippets" value={q} onChange={(e) => setQ(e.target.value)} />
        <button type="button" className="btn btn-sm" onClick={() => setImporting(true)}><IconPaste /> Import</button>
      </div>
      {snips === null ? <p className="m-pin-msg">Loading…</p> : snips.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>{q.trim() ? "No snippets match." : "Save a prompt you reuse."}</p>
          {q.trim() ? <button type="button" className="btn btn-sm" onClick={() => setQ("")}>Clear search</button> : <button type="button" className="btn btn-sm btn-primary" onClick={onNew}>New snippet</button>}
        </div>
      ) : (
        <section className="m-section">
          <ul className="m-list m-menu m-group-list">
            {snips.map((s) => (
              <li key={s.id} className={"m-row" + (s.starred ? " is-starred" : "")}>
                <button type="button" className="m-row-main" onClick={() => onOpen(s.id)}>
                  <span className="m-row-text">
                    <span className="m-row-title">{s.starred ? "★ " : ""}{s.title}</span>
                    <span className="m-row-sub">
                      {[s.kind === "shell" ? "Command" : "Prompt", "/snip:" + s.slug, s.archivedAt ? "archived" : ""].filter(Boolean).join(" · ")}
                    </span>
                  </span>
                  <IconChevronRight size={16} className="m-row-chev" />
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}
      <ImportSheet
        open={importing}
        onClose={() => setImporting(false)}
        onOpen={(body) => {
          writeDraft(storage(), "", { ...formFromSnip(null), body, title: titleFromText(body) }, "", "import");
          setImporting(false);
          onNew();
        }}
      />
    </div>
  );
}
