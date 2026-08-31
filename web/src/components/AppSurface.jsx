import { useCallback, useEffect, useRef, useState } from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { api, humanizeError } from "../lib/api.js";
import { normalizeView, supportedApp, SUPPORTED_API } from "../lib/appPrimitives.js";
import { askConfirm } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import AppIcon from "./AppIcon.jsx";
import { IconChevronLeft } from "./Icons.jsx";

const SKELETON_ROWS = 6;

// One open app (ADR-0036). The app answers with a primitive tree; this
// surface renders it with host components — chrome (header, back, close)
// stays host-owned, and a tree this build can't speak is refused, never
// guessed at.
export default function AppSurface({ appId, manifest, onClose }) {
  const [path, setPath] = useState("");
  const [view, setView] = useState(null); // normalized tree
  const [unsupported, setUnsupported] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  // Latest-wins, never skip: a click can navigate while a focus-triggered
  // refresh is in flight — dropping that load would eat the navigation.
  const seqRef = useRef(0);

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
      if (seq === seqRef.current) setBusy(false);
    }
  }, [appId]);

  useEffect(() => { setView(null); load(path); }, [path, load]);
  useEffect(() => {
    // Like the file tree: refresh when the page comes back, no polling.
    const onVisible = () => { if (!document.hidden) load(path); };
    window.addEventListener("visibilitychange", onVisible);
    window.addEventListener("focus", onVisible);
    return () => {
      window.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("focus", onVisible);
    };
  }, [load, path]);

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

  const title = (view && view.title) || (manifest && manifest.name) || appId;
  const badVersion = manifest && !supportedApp(manifest);

  return (
    <section className="app-surface" aria-label={title}>
      <header className="ft-head">
        {path ? (
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
        <div className="app-body" aria-busy="true">
          {Array.from({ length: SKELETON_ROWS }, (_, i) => (
            <div key={i} className="skel-line" style={{ width: 30 + ((i * 23) % 50) + "%" }} />
          ))}
        </div>
      ) : (
        <div className="app-body">
          {view.blocks.map((b, i) => (
            <AppBlock key={i} block={b} onNavigate={setPath} onAction={fire} />
          ))}
          {view.blocks.length === 0 ? <p className="ft-msg">This app has nothing to show here.</p> : null}
        </div>
      )}
    </section>
  );
}

function AppBlock({ block, onNavigate, onAction }) {
  if (block.type === "detail") {
    return (
      <div className="app-detail md">
        <Markdown remarkPlugins={[remarkGfm]}>{block.markdown}</Markdown>
      </div>
    );
  }
  if (block.type === "list") {
    return (
      <ul className="app-list">
        {block.items.map((it) => (
          <li key={it.id} className="app-list-row">
            <button
              type="button"
              className={"app-row-main" + (it.path ? " app-row-link" : "")}
              onClick={() => { if (it.path) onNavigate(it.path); }}
              disabled={!it.path}
            >
              {it.icon ? <span className="app-row-icon"><AppIcon name={it.icon} label={it.title} size={13} /></span> : null}
              <span className="app-row-text">
                <span className="app-row-title">{it.title}</span>
                {it.subtitle ? <span className="app-row-sub">{it.subtitle}</span> : null}
              </span>
              {it.badge ? <span className="app-row-badge">{it.badge}</span> : null}
            </button>
            {it.actions.map((a) => (
              <button key={a.id} type="button" className={"btn btn-sm" + (a.danger ? " btn-danger" : "")} onClick={() => onAction(a)}>
                {a.label}
              </button>
            ))}
          </li>
        ))}
      </ul>
    );
  }
  if (block.type === "form") {
    return <AppForm form={block.form} onAction={onAction} />;
  }
  // actions
  return (
    <div className="app-actions">
      {block.actions.map((a) => (
        <button key={a.id} type="button" className={"btn btn-sm" + (a.danger ? " btn-danger" : "")} onClick={() => onAction(a)}>
          {a.label}
        </button>
      ))}
    </div>
  );
}

function AppForm({ form, onAction }) {
  const [values, setValues] = useState(() => {
    const v = {};
    for (const f of form.fields) v[f.name] = f.method === "confirm" ? "no" : (f.prefill || (f.method === "select" ? f.options[0] || "" : ""));
    return v;
  });
  const set = (name, val) => setValues((cur) => ({ ...cur, [name]: val }));
  return (
    <form
      className="app-form"
      onSubmit={(e) => {
        e.preventDefault();
        onAction({ id: form.id, label: form.submit || "Submit", args: {} }, values);
      }}
    >
      {form.fields.map((f) => (
        <label key={f.name} className="app-field">
          {f.title ? <span className="app-field-title">{f.title}</span> : null}
          {f.message ? <span className="app-field-msg">{f.message}</span> : null}
          {f.method === "select" ? (
            <select className="dlg-input" value={values[f.name]} onChange={(e) => set(f.name, e.target.value)}>
              {f.options.map((o) => <option key={o} value={o}>{o}</option>)}
            </select>
          ) : f.method === "confirm" ? (
            <span className="app-field-confirm">
              <input type="checkbox" checked={values[f.name] === "yes"} onChange={(e) => set(f.name, e.target.checked ? "yes" : "no")} />
            </span>
          ) : f.method === "editor" ? (
            <textarea className="dlg-input app-field-editor" rows={4} value={values[f.name]} placeholder={f.placeholder} onChange={(e) => set(f.name, e.target.value)} />
          ) : (
            <input className="dlg-input" type="text" value={values[f.name]} placeholder={f.placeholder} onChange={(e) => set(f.name, e.target.value)} />
          )}
        </label>
      ))}
      <div className="app-actions">
        <button type="submit" className="btn btn-sm btn-primary">{form.submit || "Submit"}</button>
      </div>
    </form>
  );
}
