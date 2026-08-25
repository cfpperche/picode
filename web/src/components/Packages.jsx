import { useEffect, useState } from "react";
import { api, humanizeError } from "../lib/api.js";
import { askConfirm } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";

export default function Packages({ hidden }) {
  const [data, setData] = useState(null);
  const [source, setSource] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    try { setData(await api("/api/packages")); }
    catch { setData({ packages: [], capabilities: {}, gallery: "https://pi.dev/packages" }); }
  }

  useEffect(() => { if (!hidden) load(); }, [hidden]);

  async function install(e) {
    e.preventDefault();
    const src = source.trim();
    if (!src || busy) return;
    setBusy(true);
    try {
      const next = await api("/api/packages", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source: src }),
      });
      setData(next);
      setSource("");
      toast.ok("Installed " + src);
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
        <h3>Pi packages</h3>
        <p className="pkg-lead">
          Extensions, skills, prompts, and themes from npm or git.
          They run with <strong>full access</strong> — only install what you review.
        </p>
        <p className="pkg-lead">
          <a className="settings-link" href={gallery} target="_blank" rel="noopener noreferrer">Browse the gallery ↗</a>
        </p>
        <form className="pkg-install" onSubmit={install}>
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
    </PageFrame>
  );
}
