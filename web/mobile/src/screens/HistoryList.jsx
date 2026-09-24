import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { historyWorkspaceChoice, historyWorkspaceOptions, restoredToast } from "@picode/shared/domain/agentHistory.js";
import { restoreAgent } from "@picode/shared/client/launchAgent.js";
import { IconChevronRight } from "../components/Icons.jsx";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm, fmtBytes } from "../lib/confirm.js";

function cliName(id) {
  const label = terminalCliLabel(id || "pi");
  return label === "Terminal" ? id : label;
}

function post(body) {
  return { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body || {}) };
}

// Agent history on the phone (ADR-0205/0211): the same list the desktop page
// reads — removed agents whose conversation is still on disk — with Bring
// back and Remove from history. Filters stay on the desktop page.
export default function HistoryList({ workspaces = [] }) {
  const [entries, setEntries] = useState(null);
  const [err, setErr] = useState("");
  const [open, setOpen] = useState("");
  const [target, setTarget] = useState({});
  const [busy, setBusy] = useState("");
  const seq = useRef(0);

  function load() {
    const n = ++seq.current;
    api("/api/agent-history")
      .then((res) => { if (n === seq.current) { setEntries(res.entries || []); setErr(""); } })
      .catch(() => { if (n === seq.current) setErr("Couldn't load the history."); });
  }

  useEffect(load, []);
  useEffect(() => subscribeFeed((ev) => {
    const t = String(ev.type || "");
    if (t === "feed.open" || t === "feed.reset" || t.startsWith("agent_exit.")) load();
  }), []);

  const wsChoices = useMemo(() => historyWorkspaceOptions(workspaces), [workspaces]);

  async function restore(e) {
    const ex = e.exit;
    const wsId = historyWorkspaceChoice(e, target[ex.id]);
    if (!wsId) return;
    setBusy(ex.id);
    try {
      const out = await restoreAgent(ex.id, { workspaceId: wsId });
      setEntries((list) => (list || []).filter((x) => x.exit.id !== ex.id));
      setOpen("");
      const msg = restoredToast(ex.agentName, out);
      toast(msg.text, msg.ok ? "ok" : "err");
      if (out.terminalId) location.hash = "#/term/" + encodeURIComponent(out.terminalId);
      else if (out.agent) location.hash = "#/agent/" + encodeURIComponent(out.agent.id);
    } catch (error) {
      toastError(error);
      load();
    } finally {
      setBusy("");
    }
  }

  async function forget(e) {
    const ex = e.exit;
    const choice = await askConfirm({
      title: "Remove from history?",
      message: `"${ex.agentName}" can no longer be brought back from here. Its record stays in Outcomes.` + (e.canDeleteFile ? "" : ` The conversation file stays where ${cliName(ex.cli)} keeps it.`),
      confirmLabel: "Remove",
      danger: true,
      choices: e.canDeleteFile ? [{ id: "deleteFile", label: `Also delete the conversation file (${fmtBytes(e.session.size)})`, checked: false }] : [],
    });
    if (!choice) return;
    setBusy(ex.id);
    try {
      await api("/api/agent-history/" + encodeURIComponent(ex.id) + "/forget", post({ deleteFile: !!(choice && choice.deleteFile) }));
      setEntries((list) => (list || []).filter((x) => x.exit.id !== ex.id));
      setOpen("");
    } catch (error) {
      toastError(error);
    } finally {
      setBusy("");
    }
  }

  if (err && !entries) {
    return (
      <div className="m-v2-lists m-outcomes m-history">
        <div className="m-list-empty" role="alert"><p>{err}</p><button type="button" className="btn btn-sm" onClick={load}>Retry</button></div>
      </div>
    );
  }
  if (!entries) return <div className="m-v2-lists m-outcomes m-history"><p className="m-pin-msg">Loading…</p></div>;

  return (
    <div className="m-v2-lists m-outcomes m-history" aria-label="Agent history">
      {entries.length === 0 ? (
        <div className="m-list-empty" role="status">
          <p>No removed agents to bring back — when you remove one, it waits here while its conversation exists.</p>
          <a className="btn btn-sm" href="#/more/outcomes">Open Outcomes</a>
        </div>
      ) : (
        <section className="m-section">
          <p className="m-outc-facts">Removed agents whose conversation is still on disk.</p>
          <ul className="m-list m-menu m-group-list">
            {entries.map((e) => {
              const ex = e.exit;
              const s = e.session || {};
              const isOpen = open === ex.id;
              const wsId = historyWorkspaceChoice(e, target[ex.id]);
              const blocked = !e.folderExists
                ? "The folder it worked in is gone. Its conversation resumes only there — recreate the folder to bring it back."
                : !wsId ? "Its workspace was removed — choose where to bring it back." : "";
              return (
                <li key={ex.id} className={"m-row m-outc-row" + (isOpen ? " is-open" : "") + (busy === ex.id ? " is-busy" : "")}>
                  <button type="button" className="m-row-main" aria-expanded={isOpen} onClick={() => setOpen(isOpen ? "" : ex.id)}>
                    <span className="m-row-text">
                      <span className="m-row-title m-hist-title">
                        <span className="m-hist-name">{ex.agentName}</span>
                        {!e.folderExists ? <span className="m-hist-tag is-warn">folder missing</span> : !e.workspaceExists ? <span className="m-hist-tag">workspace removed</span> : null}
                      </span>
                      <span className="m-row-sub">{[relTime(ex.removedAt), cliName(ex.cli), s.name || s.preview].filter(Boolean).join(" · ")}</span>
                    </span>
                    <IconChevronRight size={16} className="m-row-chev m-outc-chev" />
                  </button>
                  {isOpen ? (
                    <div className="m-outc-detail">
                      <dl className="m-outc-facts-list">
                        <dt>Conversation</dt><dd className="m-hist-path">{s.path || "—"}</dd>
                        <dt>Folder</dt><dd className="m-hist-path">{e.folder || "—"}{!e.folderExists ? " (missing)" : ""}</dd>
                        <dt>Last active</dt><dd>{s.updatedAt ? relTime(s.updatedAt) : "—"}{s.size ? ` · ${fmtBytes(s.size)}` : ""}</dd>
                        <dt>Was in</dt><dd>{ex.workspaceId === "ws_free" ? "Free agents" : ex.workspaceName || "—"}</dd>
                      </dl>
                      <label className="m-hist-target">
                        <span>Bring back to</span>
                        <select className="dlg-input" value={wsId} disabled={busy === ex.id || !e.folderExists} onChange={(ev) => setTarget((t) => ({ ...t, [ex.id]: ev.target.value }))}>
                          {!wsId ? <option value="">Choose a workspace…</option> : null}
                          {wsChoices.map(([id, name]) => <option key={id} value={id}>{name}{id === ex.workspaceId ? " (where it was)" : ""}</option>)}
                        </select>
                      </label>
                      {blocked ? <p className="m-hist-blocked">{blocked}</p> : null}
                      <div className="m-outc-actions">
                        <button type="button" className="btn btn-sm btn-primary" disabled={busy === ex.id || !!blocked} onClick={() => restore(e)}>{busy === ex.id ? "Bringing back…" : "Bring back"}</button>
                        <button type="button" className="btn btn-sm btn-danger" disabled={busy === ex.id} onClick={() => forget(e)}>Remove from history</button>
                      </div>
                    </div>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </section>
      )}
    </div>
  );
}
