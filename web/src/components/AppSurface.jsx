import { useCallback, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, humanizeError } from "../lib/api.js";
import { normalizeView, supportedApp, SUPPORTED_API } from "../lib/appPrimitives.js";
import { useKeptScroll } from "../lib/keepScroll.js";
import { relTime, absTime } from "../lib/relTime.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import AppIcon from "./AppIcon.jsx";
import { IconChevronLeft, IconCheck, IconClock, IconInbox } from "./Icons.jsx";

const SKELETON_ROWS = 5;
// A hidden tab keeps its view; revealing it refetches only when the last read
// is old enough to have missed something. Flipping between two tabs must not
// re-ask the app on every switch.
const REVEAL_STALE_MS = 10_000;

// One open app (ADR-0036). The app answers with a primitive tree; this
// surface renders it with host components — chrome (header, split,
// selection, timestamps) stays host-owned, and a tree this build can't
// speak is refused, never guessed at.
export default function AppSurface({ appId, hidden, manifest, onClose }) {
  const [path, setPath] = useState("");
  const [view, setView] = useState(null); // normalized tree
  const [unsupported, setUnsupported] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  // Latest-wins, never skip: a click can navigate while a focus-triggered
  // refresh is in flight — dropping that load would eat the navigation.
  const seqRef = useRef(0);
  const detailRef = useRef(null);
  // Leaving the tab must not close the item the reader opened: the surface
  // stays mounted, so `path` and the loaded view survive the switch.
  const rootRef = useKeptScroll(hidden, [".app-body", ".app-pane-list"]);
  const loadRef = useRef(() => {});
  const pathRef = useRef("");
  // Mounting counts as a read: the effect below loads immediately, and the
  // reveal check must not fire a second load on top of it.
  const lastLoadRef = useRef(Date.now());

  const load = useCallback(async (p) => {
    const seq = ++seqRef.current;
    setBusy(true);
    try {
      const raw = await api("/api/apps/" + encodeURIComponent(appId) + "/view" + (p ? "?path=" + encodeURIComponent(p) : ""));
      if (seq !== seqRef.current) return; // a newer load superseded this one
      const v = normalizeView(raw);
      if (v) { setView(v); setUnsupported(false); setError(""); }
      else { setView(null); setUnsupported(true); setError(""); }
    } catch (e) {
      if (seq === seqRef.current) setError(humanizeError(e.message || String(e)));
    } finally {
      if (seq === seqRef.current) {
        setBusy(false);
        lastLoadRef.current = Date.now();
      }
    }
  }, [appId]);

  useEffect(() => { load(path); }, [path, load]);
  loadRef.current = load;
  pathRef.current = path;
  useEffect(() => {
    // Stacked (narrow) split: the detail sits below the list, so a pick
    // has to bring it into view — otherwise selecting looks like nothing.
    if (!path || !detailRef.current) return;
    if (window.matchMedia("(min-width: 881px)").matches) return;
    detailRef.current.scrollIntoView({ block: "start", behavior: "smooth" });
  }, [path]);
  useEffect(() => {
    // Like the file tree: refresh when the page comes back, no polling. Apps on
    // hidden tabs sit this out — every open tab would re-ask. Their reveal is
    // the refresh.
    const onVisible = () => { if (!document.hidden && !hidden) load(path); };
    window.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    return () => {
      window.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
    };
  }, [load, path, hidden]);
  // Reappearing is the other "back to look at it" moment — the window never
  // lost focus, so nothing above fires.
  useEffect(() => {
    if (hidden) return;
    if (Date.now() - lastLoadRef.current > REVEAL_STALE_MS) loadRef.current(pathRef.current);
  }, [hidden]);

  async function fire(action, args) {
    if (action.confirm) {
      const ok = await askConfirm({ title: action.label, message: action.confirm, confirmLabel: action.label, danger: !!action.danger });
      if (!ok) return;
    }
    try {
      const res = await api("/api/apps/" + encodeURIComponent(appId) + "/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action: action.id, path, args: { ...(action.args || {}), ...(args || {}) } }),
      });
      if (res && res.toast) toast(res.toast, "ok");
      if (res && res.view) {
        const v = normalizeView(res.view);
        if (v) { setView(v); setUnsupported(false); }
        else setUnsupported(true);
      } else if (res && typeof res.path === "string" && res.path !== path) {
        setPath(res.path);
      }
    } catch (e) {
      toastError(e);
    }
  }

  const title = (manifest && manifest.name) || appId;
  const badVersion = manifest && !supportedApp(manifest);
  const split = !!view && view.layout === "split";
  const listBlocks = split ? view.blocks.filter((b) => b.pane === "list") : [];
  const detailBlocks = split ? view.blocks.filter((b) => b.pane !== "list") : [];
  const ctx = { onNavigate: setPath, onAction: fire, selected: path };
  // The detail header repeats the selected row's kind lozenge so the two
  // panes agree — read off the list the app already sent, no new field.
  const selectedRow = split
    ? listBlocks.flatMap((b) => b.items).find((it) => it.path && it.path === path)
    : null;

  return (
    <section className={"app-surface" + (split ? " app-surface-split" : "")} aria-label={title} hidden={!!hidden} ref={rootRef}>
      <header className="ft-head">
        {path && !split ? (
          <button type="button" className="btn btn-sm btn-ghost app-back" title="Back" onClick={() => setPath("")}>
            <IconChevronLeft size={13} />
          </button>
        ) : null}
        <span className="app-head-icon"><AppIcon name={manifest ? manifest.icon : ""} label={title} size={14} /></span>
        <h2 className="ft-title" title={title}>{title}</h2>
        <span className="ft-spacer" />
        <button type="button" className="btn btn-sm btn-ghost" onClick={() => load(path)} disabled={busy}>
          Refresh
        </button>
        {onClose ? (
          <button type="button" className="btn btn-sm btn-ghost" onClick={onClose}>
            Close
          </button>
        ) : null}
      </header>

      {unsupported || badVersion ? (
        <p className="ft-msg">
          This app needs a newer PiCode — it speaks primitives v{badVersion ? manifest.apiVersion : "?"}, this build renders v{SUPPORTED_API}.
        </p>
      ) : error ? (
        <p className="ft-msg">
          {error}{" "}
          <button type="button" className="btn btn-sm" onClick={() => load(path)}>Try again</button>
        </p>
      ) : view === null ? (
        <Skeleton />
      ) : split ? (
        <div className="app-split">
          <div className="app-pane app-pane-list">
            {listBlocks.map((b, i) => <AppBlock key={i} block={b} {...ctx} />)}
          </div>
          <div className="app-pane app-pane-detail" ref={detailRef}>
            {detailBlocks.length === 0 ? (
              <Blank icon={manifest ? manifest.icon : ""} label={title} text="Nothing selected — Pick an item on the left." />
            ) : (
              <PaneBlocks blocks={detailBlocks} ctx={ctx} badge={selectedRow} />
            )}
          </div>
        </div>
      ) : (
        <div className="app-body">
          {view.blocks.map((b, i) => <AppBlock key={i} block={b} {...ctx} />)}
          {view.blocks.length === 0 ? <Blank icon={manifest ? manifest.icon : ""} label={title} text={view.empty} /> : null}
        </div>
      )}
    </section>
  );
}

// A form and the actions that follow it are one decision, so they share
// one button row: submit first, then the secondary and destructive ones.
function PaneBlocks({ blocks, ctx, badge }) {
  const formAt = blocks.findIndex((b) => b.type === "form");
  const actionsAt = blocks.findIndex((b) => b.type === "actions");
  const merged = formAt >= 0 && actionsAt > formAt;
  return (
    <>
      {blocks.map((b, i) => {
        if (merged && i === actionsAt) return null;
        const extra = merged && i === formAt ? blocks[actionsAt].actions : undefined;
        return <AppBlock key={i} block={b} {...ctx} extraActions={extra} badge={i === 0 ? badge : null} />;
      })}
    </>
  );
}

function Skeleton() {
  return (
    <div className="app-body" aria-busy="true">
      {Array.from({ length: SKELETON_ROWS }, (_, i) => (
        <div key={i} className="app-skel-row">
          <span className="skel-line app-skel-title" style={{ width: 40 + ((i * 17) % 40) + "%" }} />
          <span className="skel-line app-skel-meta" />
        </div>
      ))}
    </div>
  );
}

// The app names its own emptiness (view.empty); the host owns how a
// blankslate looks — a quiet mark, one line, no call to action.
function Blank({ icon, label, text }) {
  const line = text || "Nothing here yet.";
  const [head, ...rest] = line.split(" — ");
  return (
    <div className="app-blank">
      {icon ? <AppIcon name={icon} label={label} size={24} /> : <IconInbox size={24} />}
      <p className="app-blank-title">{head}</p>
      {rest.length ? <p className="app-blank-sub">{rest.join(" — ")}</p> : null}
    </div>
  );
}

// A block's optional header: a section label with the count, plus the
// meta strip (drawn separators, so an empty chip leaves no gap).
function BlockHead({ block, count, badge }) {
  if (!block.title && block.meta.length === 0 && !block.at) return null;
  const heading = block.pane === "detail";
  return (
    <div className={heading ? "app-detail-head" : "app-sect"}>
      {block.title ? <span className={heading ? "app-detail-title" : "app-sect-title"}>{block.title}</span> : null}
      {!heading && typeof count === "number" ? <span className="app-sect-count">{count}</span> : null}
      {block.meta.length || block.at || badge ? (
        <span className="app-meta">
          {badge && badge.badge ? (
            <span className={"app-kind" + (badge.tone ? " tone-" + badge.tone : "")}>{badge.badge}</span>
          ) : null}
          {block.meta.map((m, i) => <span key={i}>{m}</span>)}
          {block.at ? <span className="app-when" title={absTime(block.at)}>{relTime(block.at)}</span> : null}
        </span>
      ) : null}
    </div>
  );
}

function AppBlock({ block, onNavigate, onAction, selected, extraActions, badge }) {
  if (block.type === "detail") {
    return (
      <div className="app-block">
        <BlockHead block={block} badge={badge} />
        <div className="app-detail md">
          <Markdown remarkPlugins={[remarkGfm]}>{block.markdown}</Markdown>
        </div>
      </div>
    );
  }
  if (block.type === "list") {
    return (
      <div className="app-block">
        <BlockHead block={block} count={block.items.length} />
        <ul className="app-list">
          {block.items.map((it) => (
            <Row key={it.id} item={it} onNavigate={onNavigate} onAction={onAction} active={!!it.path && it.path === selected} />
          ))}
        </ul>
      </div>
    );
  }
  if (block.type === "form") {
    return (
      <div className="app-block">
        <BlockHead block={block} />
        <AppForm form={block.form} onAction={onAction} extraActions={extraActions} />
      </div>
    );
  }
  // actions
  return (
    <div className="app-block">
      <BlockHead block={block} />
      <div className="app-actions">
        {block.actions.map((a) => (
          <ActionButton key={a.id} action={a} onAction={onAction} />
        ))}
      </div>
    </div>
  );
}

// Emphasis is the app's call (Action.primary), not a guess from position:
// on an approval the decision deserves the fill, on a result nothing does.
function ActionButton({ action, onAction }) {
  return (
    <button
      type="button"
      className={"btn btn-sm" + (action.danger ? " btn-danger" : action.primary ? " btn-primary" : "")}
      onClick={() => onAction(action)}
    >
      {action.label}
    </button>
  );
}

const ROW_ICONS = { check: IconCheck, clock: IconClock };

// One dense row: unread dot, title, relative time, then a meta strip.
// Row actions stay hidden until hover or keyboard focus, so they cost no
// width and never set the row's height.
function Row({ item, onNavigate, onAction, active }) {
  const has = item.path && onNavigate;
  return (
    <li className={"app-row" + (active ? " app-row-on" : "") + (item.unread ? " app-row-unread" : "")}>
      <button
        type="button"
        className="app-row-main"
        onClick={() => { if (has) onNavigate(item.path); }}
        disabled={!has}
      >
        <span className="app-row-dot" aria-hidden="true" />
        <span className="app-row-line1">
          <span className="app-row-title">{item.title}</span>
          {item.at ? <span className="app-when" title={absTime(item.at)}>{relTime(item.at)}</span> : null}
        </span>
        <span className="app-row-line2">
          {item.badge ? <span className={"app-kind" + (item.tone ? " tone-" + item.tone : "")}>{item.badge}</span> : null}
          <span className="app-meta">
            {item.meta.map((m, i) => <span key={i}>{m}</span>)}
            {item.subtitle ? <span>{item.subtitle}</span> : null}
          </span>
        </span>
      </button>
      {item.actions.length ? (
        <span className="app-row-actions">
          {item.actions.map((a) => {
            const Glyph = ROW_ICONS[a.icon];
            return (
              <button
                key={a.id}
                type="button"
                className={"ws-icon-btn" + (a.danger ? " danger" : "")}
                title={a.label}
                aria-label={a.label}
                onClick={() => onAction(a)}
              >
                {Glyph ? <Glyph size={13} /> : a.label}
              </button>
            );
          })}
        </span>
      ) : null}
    </li>
  );
}

function AppForm({ form, onAction, extraActions }) {
  const [values, setValues] = useState(() => {
    const v = {};
    for (const f of form.fields) v[f.name] = f.method === "confirm" ? "no" : (f.prefill || (f.method === "select" ? f.options[0] || "" : ""));
    return v;
  });
  const set = (name, val) => setValues((cur) => ({ ...cur, [name]: val }));
  const submit = () => onAction({ id: form.id, label: form.submit || "Submit", args: {} }, values);
  const hasEditor = form.fields.some((f) => f.method === "editor");
  const extraPrimary = (extraActions || []).some((a) => a.primary);
  return (
    <form
      className="app-form"
      onSubmit={(e) => { e.preventDefault(); submit(); }}
    >
      {form.fields.map((f) => (
        <label key={f.name} className="app-field">
          {f.title ? <span className="app-field-title">{f.title}</span> : null}
          {f.method === "select" ? (
            <select className="dlg-input" value={values[f.name]} onChange={(e) => set(f.name, e.target.value)}>
              {f.options.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          ) : f.method === "confirm" ? (
            <span className="app-field-confirm">
              <input type="checkbox" checked={values[f.name] === "yes"} onChange={(e) => set(f.name, e.target.checked ? "yes" : "no")} />
            </span>
          ) : f.method === "editor" ? (
            <textarea
              className="dlg-input app-field-editor"
              rows={3}
              value={values[f.name]}
              placeholder={f.placeholder}
              onChange={(e) => set(f.name, e.target.value)}
              onKeyDown={(e) => {
                // Send without reaching for the mouse — the shortcut every
                // composer in this app already answers to.
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); submit(); }
              }}
            />
          ) : (
            <input className="dlg-input" type="text" value={values[f.name]} placeholder={f.placeholder} onChange={(e) => set(f.name, e.target.value)} />
          )}
          {f.message ? <span className="app-field-msg">{f.message}</span> : null}
        </label>
      ))}
      <div className="app-actions">
        {/* Only one filled button per row: if the app marked a decision as
            primary, submitting the reply is the side channel. */}
        <button type="submit" className={"btn btn-sm" + (extraPrimary ? "" : " btn-primary")}>
          {form.submit || "Submit"}
        </button>
        {(extraActions || []).map((a) => (
          <ActionButton key={a.id} action={a} onAction={onAction} />
        ))}
        {hasEditor ? (
          <span className="app-field-hint"><span className="app-key">Ctrl</span>+<span className="app-key">Enter</span> to send</span>
        ) : null}
      </div>
    </form>
  );
}
