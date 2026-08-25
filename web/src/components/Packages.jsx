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
    <PageFrame id="packages-view" title="Packages" hidden={hidden} wide>
      <form className="pkg-by-source" onSubmit={(e) => { e.preventDefault(); installSource(source); }}>
        <input
          className="dlg-input"
          value={source}
          onChange={(e) => setSource(e.target.value)}
          placeholder="Install by source — npm:pi-web-search"
          disabled={busy}
          aria-label="Package source"
        />
        <button type="submit" className="btn btn-primary btn-sm" disabled={busy || !source.trim()}>Install</button>
      </form>
      <p className="pkg-fine">Packages run with full access. Only install what you review.</p>

      <section className="pkg-toolbar" data-align-row>
        <input
          className="pkg-search"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Filter packages…"
          aria-label="Search gallery"
        />
        <span className="pkg-count">{searching ? "Searching…" : hits.length ? hits.length + " shown" : "No matches"}</span>
        <a className="settings-link" href={gallery} target="_blank" rel="noopener noreferrer">pi.dev ↗</a>
      </section>

      {list.length > 0 ? (
        <section className="pkg-installed">
          <h3>Installed</h3>
          <ul className="pkg-chips">
            {list.map((p) => (
              <li key={p.scope + ":" + p.source} className="pkg-chip">
                <span className="pkg-src">{p.source}</span>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => remove(p)} disabled={busy}>Remove</button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <ul className="pkg-grid">
        {hits.map((h) => {
          const on = installed.has(h.source);
          return (
            <li key={h.source} className="pkg-card">
              <div className={"pkg-preview" + (h.image ? " has-media" : "")} aria-hidden="true">
                <div className="pkg-preview-frame">
                  {h.image ? <img src={h.image} alt="" loading="lazy" /> : <><span /><span /><span /></>}
                </div>
              </div>
              <div className="pkg-card-body">
              <div className="pkg-card-head">
                <span className="pkg-card-name">{h.name}</span>
                {h.kind ? <span className="pkg-type">{h.kind}</span> : null}
              </div>
              {h.description ? <p className="pkg-card-desc">{h.description}</p> : <p className="pkg-card-desc"> </p>}
              <div className="pkg-card-meta">
                {h.publisher ? <span>{h.publisher}</span> : null}
                {h.downloads ? <span>{fmtDown(h.downloads)}</span> : null}
                {h.updated ? <span>{fmtAge(h.updated)}</span> : null}
                {h.version ? <span>{h.version}</span> : null}
              </div>
              <div className="pkg-card-foot">
                <code className="pkg-cmd">pi install {h.source}</code>
                <button
                  type="button"
                  className="btn btn-primary btn-sm"
                  disabled={busy || on}
                  onClick={() => installSource(h.source)}
                >
                  {on ? "Installed" : busy ? "Working…" : "Install"}
                </button>
              </div>
              </div>
            </li>
          );
        })}
      </ul>

    </PageFrame>
  );
}

function fmtDown(n) {
  if (!n) return "";
  if (n >= 1e6) return trimNum(n / 1e6) + "M/mo";
  if (n >= 1000) return trimNum(n / 1000) + "k/mo";
  return n + "/mo";
}

function trimNum(n) {
  return n.toFixed(n >= 10 ? 0 : 1).replace(/\.0$/, "");
}

function fmtAge(iso) {
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const d = Date.now() - t;
  const day = 86400000;
  if (d < day) return "today";
  if (d < 2 * day) return "1d ago";
  if (d < 30 * day) return Math.floor(d / day) + "d ago";
  if (d < 365 * day) return Math.floor(d / (30 * day)) + "mo ago";
  return Math.floor(d / (365 * day)) + "y ago";
}
