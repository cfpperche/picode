import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { effectiveKeys, fromEvent, isOverride, matchKeys } from "../lib/piKey.js";

export default function PiKeys() {
  const [rep, setRep] = useState(null);
  const [err, setErr] = useState("");
  const [q, setQ] = useState("");
  const [listen, setListen] = useState("");

  function load() {
    setErr("");
    api("/api/pi-keys").then(setRep).catch((e) => { setRep(null); setErr(e.message || "Can't load keys."); });
  }

  useEffect(() => { load(); }, []);

  useEffect(() => {
    if (!listen) return undefined;
    function onKey(ev) {
      if (ev.key === "Escape") {
        ev.preventDefault();
        setListen("");
        return;
      }
      const chord = fromEvent(ev);
      if (!chord) return;
      ev.preventDefault();
      ev.stopPropagation();
      const action = (rep.actions || []).find((a) => a.id === listen);
      if (!action) return;
      const cur = effectiveKeys(action, rep.user);
      if (cur.includes(chord)) { setListen(""); return; }
      save(listen, cur.concat(chord));
      setListen("");
    }
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [listen, rep]);

  async function save(action, keys, reset) {
    try {
      const next = await api("/api/pi-keys", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(reset ? { action, reset: true } : { action, keys }),
      });
      setRep(next);
      toast.ok(reset ? "Back to Pi default." : "Saved. Restart the agent or /reload in Pi.");
    } catch (e) { toastError(e); }
  }

  if (err) {
    return (
      <section className="settings-section">
        <h3>Keys</h3>
        <p className="settings-desc">Can't load keys. <button type="button" className="btn btn-ghost btn-sm" onClick={load}>Retry</button></p>
      </section>
    );
  }
  if (!rep) {
    return (
      <section className="settings-section">
        <h3>Keys</h3>
        <p className="settings-desc">Loading keys…</p>
        <div className="key-skel" aria-hidden="true" />
        <div className="key-skel" aria-hidden="true" />
      </section>
    );
  }

  const actions = rep.actions || [];
  const user = rep.user || {};
  const shown = actions.filter((a) => matchKeys(a, user, q));
  const groups = [];
  for (const a of shown) {
    const last = groups[groups.length - 1];
    if (!last || last.name !== a.group) groups.push({ name: a.group, rows: [a] });
    else last.rows.push(a);
  }

  return (
    <section className="settings-section">
      <h3>Keys</h3>
      <p className="settings-desc">This machine. Click Add, then press a key. Same map as Pi.</p>
      <form className="key-filter" noValidate onSubmit={(e) => e.preventDefault()}>
        <input
          id="keys-filter"
          type="search"
          placeholder="Filter keys"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </form>
      {shown.length === 0 ? (
        <p className="settings-desc">No matching keys. <button type="button" className="btn btn-ghost btn-sm" onClick={() => setQ("")}>Clear</button></p>
      ) : groups.map((g) => (
        <div key={g.name} className="key-group">
          <h4 className="settings-h">{g.name}</h4>
          {g.rows.map((a) => {
            const keys = effectiveKeys(a, user);
            const dirty = isOverride(a, user);
            const waiting = listen === a.id;
            return (
              <div key={a.id} className="key-row" data-align-row>
                <span className="key-label">{a.label}</span>
                <div className="key-keys" data-align-row>
                  {waiting ? (
                    <span className="key-listen">Press a key</span>
                  ) : keys.length === 0 ? (
                    <span className="key-none">Off</span>
                  ) : keys.map((k) => (
                    <span key={k} className="key-chip">
                      {k}
                      <button type="button" className="key-x" aria-label={"Remove " + k} onClick={() => save(a.id, keys.filter((x) => x !== k))}>×</button>
                    </span>
                  ))}
                  {waiting ? (
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => setListen("")}>Cancel</button>
                  ) : (
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => setListen(a.id)}>Add</button>
                  )}
                  {dirty ? (
                    <button type="button" className="btn btn-ghost btn-sm" onClick={() => save(a.id, null, true)}>Reset</button>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>
      ))}
    </section>
  );
}
