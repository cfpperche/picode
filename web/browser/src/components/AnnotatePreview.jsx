import { useEffect, useRef, useState } from "react";
import AnnotateStrip from "./AnnotateStrip.jsx";
import { toast } from "../lib/toast.js";

// AnnotatePreview — the fallback face of annotate mode (no desktop shell, so
// no live page to inject into). A clearly-labeled sample page with the same
// interaction shape as the in-page script: click to pin (numbered), an
// anchored card (icon, input, trash, mic, Cancel, Save), saved notes as chips
// (text, … menu, ×). Nothing here sends anywhere: Send says where sending
// happens. Built for the owner's screenshot acceptance and as the visible
// shape of the desktop mode.
let seq = 0;

const BLOCKS = [
  { id: "hero", kicker: "Acme — annotate this page", title: "Click anything to pin a note", body: "Pins, the card and the chips behave like the desktop mode on a live page." },
  { id: "start", title: "Getting started", body: "Pin a note on this block — the card opens beside it, Save collapses it to a chip." },
  { id: "price", title: "Pricing", body: "Every plan ships the same annotate flow. Click here to question a number." },
  { id: "news", title: "Changelog", body: "Shipped Tuesday: numbered pins, the anchored card, one Send for the whole set." },
];

export default function AnnotatePreview({ onExit }) {
  const [items, setItems] = useState([]);
  const [editing, setEditing] = useState(null); // id with the open card
  const [menuFor, setMenuFor] = useState(null);
  const [draft, setDraft] = useState("");
  const [shotsOn, setShotsOn] = useState(true);
  const inputRef = useRef(null);

  const saved = items.filter((it) => it.saved);
  useEffect(() => {
    if (editing != null) inputRef.current?.focus();
  }, [editing]);
  useEffect(() => {
    const onKey = (e) => {
      if (e.key === "Escape") {
        e.preventDefault();
        if (menuFor != null) setMenuFor(null);
        else if (editing != null) cancelCard();
        else onExit?.();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const renumbered = (list) => list.map((it, i) => ({ ...it, n: i + 1 }));

  const pick = (block, e) => {
    // Every event field is read synchronously here: the updater below can run
    // after React released the synthetic event, when currentTarget is null
    // (a blank root from a render-phase throw — found via agent-browser).
    const node = e.currentTarget;
    // Locked while a card is open, like the page script: a stray click must
    // not steal the card and drop the unsaved draft (owner 2026-09-18).
    if (editing != null) return;
    if (!node || e.target.closest?.("[data-annot-ui]")) return;
    const box = node.getBoundingClientRect();
    const x = e.clientX - box.left + node.offsetLeft;
    const y = e.clientY - box.top + node.offsetTop;
    const id = ++seq;
    setItems((prev) => renumbered([...prev, {
      id,
      n: prev.length + 1,
      block,
      x,
      y,
      comment: "",
      saved: false,
    }]));
    setMenuFor(null);
    setDraft("");
    setEditing(id);
  };

  const removeItem = (id) => {
    setItems((prev) => renumbered(prev.filter((it) => it.id !== id)));
    if (editing === id) setEditing(null);
    if (menuFor === id) setMenuFor(null);
  };

  const clearAll = () => {
    if (items.length === 0) return;
    setItems([]);
    setEditing(null);
    setMenuFor(null);
    toast.ok("All annotations discarded.");
  };

  const cancelCard = () => {
    const it = items.find((x) => x.id === editing);
    if (it && !it.saved) {
      removeItem(editing);
      return;
    }
    setEditing(null);
  };

  const saveCard = () => {
    const text = draft.trim();
    setItems((prev) => prev.map((it) => (it.id === editing ? { ...it, comment: text, saved: true } : it)));
    setEditing(null);
  };

  const openCard = (id) => {
    const it = items.find((x) => x.id === id);
    setDraft(it ? it.comment : "");
    setMenuFor(null);
    setEditing(id);
  };

  const copyText = (id) => {
    const it = items.find((x) => x.id === id);
    const text = (it?.comment || "").trim();
    setMenuFor(null);
    if (!text) {
      toast("Nothing to copy yet.");
      return;
    }
    if (navigator.clipboard) {
      navigator.clipboard.writeText(text).then(
        () => toast.ok("Annotation text copied."),
        () => toast("Copy failed in this browser."),
      );
    } else {
      toast("Copy is not available in this browser.");
    }
  };

  // Pin/chip/dots reopen only with no card open — same lock as the page.
  const guardedOpen = (id) => {
    if (editing != null) return;
    openCard(id);
  };
  const guardedMenu = (id) => {
    if (editing != null) return;
    setMenuFor(menuFor === id ? null : id);
  };
  const editingItem = items.find((x) => x.id === editing) || null;
  const menuItem = items.find((x) => x.id === menuFor) || null;

  const overlay = (it) => (
    <span key={it.id}>
      <button
        type="button"
        data-annot-ui
        className={"annot-pin" + (editing === it.id ? " on" : "")}
        style={{ left: it.x - 9, top: it.y - 9 }}
        title={`Edit annotation ${it.n}`}
        aria-label={`Edit annotation ${it.n}`}
        onClick={(e) => { e.stopPropagation(); guardedOpen(it.id); }}
      >{it.n}</button>
      {it.saved ? (
        <span
          className="annot-chip"
          data-annot-ui
          style={{ left: Math.min(it.x + 12, 400), top: Math.max(it.y - 16, 0) }}
          onClick={(e) => { e.stopPropagation(); guardedOpen(it.id); }}
        >
          <span className="annot-chip-txt" title={it.comment || `Annotation ${it.n}`}>{it.comment || `Annotation ${it.n}`}</span>
          <button type="button" data-annot-ui title="Annotation options" aria-label="Annotation options" onClick={(e) => { e.stopPropagation(); guardedMenu(it.id); }}>…</button>
          <button type="button" data-annot-ui title="Remove this annotation" aria-label="Remove this annotation" onClick={(e) => { e.stopPropagation(); removeItem(it.id); }}>×</button>
        </span>
      ) : null}
    </span>
  );

  return (
    <div className="annot-preview">
      <div className="annot-preview-bar">
        <AnnotateStrip
          count={saved.length}
          total={items.length}
          shotsOn={shotsOn}
          onExit={onExit}
          onClear={clearAll}
          onUndo={() => items.length && removeItem(items[items.length - 1].id)}
          onToggleShots={() => setShotsOn((v) => !v)}
          onHint={() => toast.ok("Click an element to pin it · Enter saves · Esc exits.")}
          onSend={() => toast("Sending needs the desktop app — annotations ride the agent terminal there.")}
        />
      </div>
      <div className="annot-mock">
        <span className="annot-badge">Preview — the desktop app annotates live pages</span>
        {BLOCKS.map((b, i) => (
          <div
            key={b.id}
            className={"annot-mock-block" + (i === 0 ? " hero" : "")}
            onClick={(e) => pick(b.id, e)}
          >
            {b.kicker ? <span className="annot-mock-kicker">{b.kicker}</span> : null}
            <h3>{b.title}</h3>
            <p>{b.body}</p>
            {i === 0 ? <span className="annot-mock-cta">Get started</span> : null}
            {items.filter((it) => it.block === b.id).map(overlay)}
          </div>
        ))}
        {editingItem ? (
          <div
            className="annot-card"
            data-annot-ui
            style={{ left: Math.min(editingItem.x, 380), top: editingItem.y + 22 }}
            role="dialog"
            aria-label="Annotation"
          >
            <div className="annot-card-row">
              <span className="annot-card-mark" aria-hidden="true">✎</span>
              <input
                ref={inputRef}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    saveCard();
                  }
                }}
                placeholder="add a comment..."
                aria-label="Annotation comment"
              />
              <button type="button" className="annot-card-ic" title="Discard this annotation" aria-label="Discard this annotation" onClick={() => removeItem(editingItem.id)}>🗑</button>
              <button type="button" className="annot-card-ic" title="Dictate the note" aria-label="Dictate the note" onClick={() => toast("Voice input needs the desktop app — type the note.")}>🎙</button>
            </div>
            <div className="annot-card-actions">
              <button type="button" className="annot-card-cancel" onClick={cancelCard}>Cancel</button>
              <button type="button" className="annot-card-save" onClick={saveCard}>Save</button>
            </div>
          </div>
        ) : null}
        {menuItem ? (
          <div className="annot-menu" data-annot-ui style={{ left: Math.min(menuItem.x + 12, 420), top: menuItem.y + 18 }} role="menu">
            <button type="button" role="menuitem" onClick={() => openCard(menuItem.id)}>Edit</button>
            <button type="button" role="menuitem" onClick={() => copyText(menuItem.id)}>Copy text</button>
            <button type="button" role="menuitem" className="danger" onClick={() => removeItem(menuItem.id)}>Remove</button>
          </div>
        ) : null}
      </div>
    </div>
  );
}
