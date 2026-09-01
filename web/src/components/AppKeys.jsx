import { useEffect, useState } from "react";
import { CATALOG } from "../lib/appKeys.js";
import { readAppKeyOverrides, persistAppKeyOverride } from "../lib/appKeyPrefs.js";
import { effectiveKeys, fromEvent, isOverride, matchKeys } from "../lib/piKey.js";

export default function AppKeys({ hidden }) {
  const [user, setUser] = useState(readAppKeyOverrides);
  const [q, setQ] = useState("");
  const [listen, setListen] = useState("");

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
      const action = CATALOG.find((a) => a.id === listen);
      if (!action) return;
      const cur = effectiveKeys(action, user);
      if (cur.includes(chord)) { setListen(""); return; }
      save(listen, cur.concat(chord));
      setListen("");
    }
    window.addEventListener("keydown", onKey, true);
    return () => window.removeEventListener("keydown", onKey, true);
  }, [listen, user]);

  function save(actionId, keys, reset) {
    setUser(persistAppKeyOverride(actionId, reset ? null : keys));
  }

  const shown = CATALOG.filter((a) => matchKeys(a, user, q));
  const groups = [];
  for (const a of shown) {
    const last = groups[groups.length - 1];
    if (!last || last.name !== a.group) groups.push({ name: a.group, rows: [a] });
    else last.rows.push(a);
  }

  return (
    <section className="settings-section" hidden={hidden}>
      <h3>Keyboard</h3>
      <p className="settings-desc">This browser. Click Add, then press a key.</p>
      <form className="key-filter" noValidate onSubmit={(e) => e.preventDefault()}>
        <input
          id="app-keys-filter"
          type="search"
          placeholder="Filter shortcuts"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
      </form>
      {shown.length === 0 ? (
        <p className="settings-desc">No matching shortcuts. <button type="button" className="btn btn-ghost btn-sm" onClick={() => setQ("")}>Clear</button></p>
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
