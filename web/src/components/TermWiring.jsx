import { useEffect, useState } from "react";
import { api } from "../lib/api.js";
import { toast, toastError } from "../lib/toast.js";
import { DOCS_BASE } from "../lib/commandDocs.js";

// Guest CLI wiring (ADR-0056 tier 1): one row per supported CLI — its
// state and the one action. Claude Code is one click (PiCode writes the
// hooks and the reporter); Codex is one manual line in config.toml, so
// its row shows the state and the guide instead of a dead button.
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
      toast.ok(op === "enable" ? "Status signals on." : "Status signals off.");
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
              {row.manual ? row.note : row.installed ? row.configPath : "Not found on this machine's PATH — enable anyway and it works once installed."}
            </span>
          </label>
          <span style={{ display: "flex", alignItems: "center", justifyContent: "flex-end", gap: 8 }}>
            {row.wired ? <span className="ws-wait">On</span> : <span className="ws-meta">Off</span>}
            {row.manual ? (
              <a className="btn btn-ghost btn-sm" href={DOCS_BASE + "/guide/terminal-status"} target="_blank" rel="noreferrer">Guide</a>
            ) : row.wired ? (
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
