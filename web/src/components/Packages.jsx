import { useEffect, useMemo, useState } from "react";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";

export default function Packages({ hidden }) {
  const [data, setData] = useState(null);
  const [source, setSource] = useState("");
  const [q, setQ] = useState("");
  const [hits, setHits] = useState([]);
  const [searching, setSearching] = useState(false);
  const [busy, setBusy] = useState(false);

  async function load() {
    try { setData(await api("/api/packages")); }
    catch { setData({ packages: [], capabilities: {}, gallery: "https://pi.dev/packages" }); }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden]);

  useEffect(() => {
    if (hidden) return;
    const t = setTimeout(async () => {
      setSearching(true);
      try {
        const page = await api("/api/packages/gallery?q=" + encodeURIComponent(q.trim()));
        setHits(page.hits || []);
      } catch { setHits([]); }
      finally { setSearching(false); }
    }, q ? 280 : 0);
    return () => clearTimeout(t);
  }, [hidden, q]);

  const installed = useMemo(() => {
    const s = new Set();
    for (const p of (data && data.packages) || []) s.add(p.source);
    return s;
  }, [data]);

  async function installSource(src) {
    const nextSrc = (src || "").trim();
    if (!nextSrc || busy) return;
    setBusy(true);
    try {
      const next = await api("/api/packages", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source: nextSrc }),
      });
      setData(next);
      setSource("");
      toast.ok("Installed " + nextSrc);
    } catch (err) {
      toast.error(humanizeError(err.message || String(err)));
    } finally { setBusy(false); }
  }

  async function remove(pkg) {
    const ok = await askConfirm({
      title: "Remove package",
      message: "Remove " + pkg.source + " from pi? This does not uninstall pi itself.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok || busy) return;
    setBusy(true);
    try {
      const next = await api("/api/packages?source=" + encodeURIComponent(pkg.source), { method: "DELETE" });
      setData(next);
      toast.ok("Removed " + pkg.source);
    } catch (err) {
      toast.error(humanizeError(err.message || String(err)));
    } finally { setBusy(false); }
  }

  const list = data && data.packages ? data.packages : [];
  const gallery = (data && data.gallery) || "https://pi.dev/packages";

  return (
    <PageFrame id="packages-view" title="Packages" hidden={hidden}>
      <section className="settings-section">
        <h3>Installed</h3>
        {list.length === 0 ? (
          <p className="pkg-empty">No packages installed.</p>
        ) : (
          <ul className="pkg-list">
            {list.map((p) => (
              <li key={p.scope + ":" + p.source} className="pkg-row">
                <span className="pkg-src">{p.source}</span>
                <span className="pkg-meta">{p.kind} · {p.scope}{p.filtered ? " · filtered" : ""}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => remove(p)} disabled={busy}>Remove</button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="settings-section">
        <h3>Gallery</h3>
        <p className="pkg-lead">
          npm packages tagged <code>pi-package</code>. They run with <strong>full access</strong> — only install what you review.
          {" "}<a className="settings-link" href={gallery} target="_blank" rel="noopener noreferrer">pi.dev ↗</a>
        </p>
        <input
          className="dlg-input"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search packages"
          aria-label="Search gallery"
        />
        {searching ? <p className="pkg-empty">Searching…</p> : null}
        {!searching && hits.length === 0 ? <p className="pkg-empty">No matches.</p> : null}
        <ul className="pkg-list">
          {hits.map((h) => {
            const on = installed.has(h.source);
            return (
              <li key={h.source} className="pkg-hit">
                <div className="pkg-hit-top">
                  <span className="pkg-src">{h.name}</span>
                  <span className="pkg-meta">{h.version}</span>
                  <button
                    type="button"
                    className="btn btn-primary btn-sm"
                    disabled={busy || on}
                    onClick={() => installSource(h.source)}
                  >
                    {on ? "Installed" : busy ? "Working…" : "Install"}
                  </button>
                </div>
                {h.description ? <p className="pkg-desc">{h.description}</p> : null}
              </li>
            );
          })}
        </ul>
      </section>

      <section className="settings-section">
        <h3>Install by source</h3>
        <p className="pkg-lead">npm:, git:, or a local path — same as <code>pi install</code>.</p>
        <form className="pkg-install" onSubmit={(e) => { e.preventDefault(); installSource(source); }}>
          <input
            className="dlg-input"
            value={source}
            onChange={(e) => setSource(e.target.value)}
            placeholder="npm:pi-web-search"
            disabled={busy}
            aria-label="Package source"
          />
          <button type="submit" className="btn btn-primary btn-sm" disabled={busy || !source.trim()}>
            {busy ? "Working…" : "Install"}
          </button>
        </form>
      </section>
    </PageFrame>
  );
}
