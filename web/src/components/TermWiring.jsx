import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";

// Terminal CLI intercept (ADR-0056): one row per CLI. Turn on drops a
// wrapper in PiCode's data dir and prepends it to PATH *inside PiCode
// terminals only*. Nothing is written to ~/.claude, ~/.codex, ~/.grok, or ~/.pi.
export default function TermWiring({ hidden }) {
  const [rows, setRows] = useState(null);
  const [busy, setBusy] = useState("");

  const load = () => api("/api/terminals/wiring").then((d) => setRows(d.clis || [])).catch((e) => toastError(e));
  useEffect(() => {
    if (!hidden && !rows) load();
  }, [hidden]); // eslint-disable-line react-hooks/exhaustive-deps

  async function act(cli, op) {
    setBusy(cli + ":" + op);
    try {
      const d = await api(`/api/terminals/wiring/${cli}/${op}`, { method: "POST" });
      setRows(d.clis || []);
      toast.ok(op === "enable" ? "Intercept on in PiCode terminals." : "Intercept off.");
    } catch (e) {
      toastError(e);
    } finally {
      setBusy("");
    }
  }

  return (
    <div className="set-rows" hidden={hidden}>
      {rows === null ? <p className="side-empty">Reading…</p> : null}
      {(rows || []).map((row) => (
        <div className="set-row" data-align-row style={{ alignItems: "stretch" }} key={row.id}>
          <label htmlFor={"wiring-" + row.id}>
            {row.label}
            <span className="ws-meta" style={{ display: "block", fontWeight: 400 }}>
              {row.note}
              {!row.installed ? " Not on PATH yet — intercept waits until it is." : ""}
            </span>
          </label>
          <span style={{ display: "flex", alignItems: "center", justifyContent: "flex-end", gap: 8 }}>
            {row.wired ? <span className="ws-wait">On</span> : <span className="ws-meta">Off</span>}
            {row.wired ? (
              <button id={"wiring-" + row.id} type="button" className="btn btn-ghost btn-sm" disabled={busy === row.id + ":disable"} onClick={() => act(row.id, "disable")}>Turn off</button>
            ) : (
              <button id={"wiring-" + row.id} type="button" className="btn btn-primary btn-sm" disabled={busy === row.id + ":enable"} onClick={() => act(row.id, "enable")}>Turn on</button>
            )}
          </span>
        </div>
      ))}
    </div>
  );
}
