import { useEffect, useMemo, useRef, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { IconChevronRight } from "./Icons.jsx";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { terminalCliLabel } from "@picode/shared/domain/terminalCli.js";
import { historyWorkspaceChoice, historyWorkspaceOptions } from "@picode/shared/domain/agentHistory.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm, fmtBytes } from "../lib/confirm.js";
import { termHash } from "../lib/routes.js";
import "./outcomes.css";
import "./agentHistory.css";

const FREE = "ws_free";

function cliName(id) {
  const label = terminalCliLabel(id || "pi");
  return label === "Terminal" ? id : label;
}

function post(body) {
  return { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body || {}) };
}

async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    toast.ok("Copied.");
  } catch {
    toast.error("Couldn't copy — select the path and copy it.");
  }
}

// Agent history (ADR-0205): removed agents whose conversation is still on
// disk. Each one can come back — in its workspace or another — resuming
// that conversation, or leave the history. The list is the exit records
// the server still finds a transcript for, so an entry whose file is gone
// is simply not here.
export default function AgentHistory({ hidden, workspaces = [], onOpenAgent }) {
  const [entries, setEntries] = useState(null);
  const [loadErr, setLoadErr] = useState("");
  const [open, setOpen] = useState("");
  const [ws, setWs] = useState("");
  const [cli, setCli] = useState("");
  const [target, setTarget] = useState({}); // exit id → chosen workspace id
  const [busy, setBusy] = useState(""); // exit id being restored or forgotten
  const seq = useRef(0);
  const hiddenRef = useRef(hidden);
  hiddenRef.current = hidden;

  function load() {
    const n = ++seq.current;
    api("/api/agent-history").then((res) => {
      if (n !== seq.current) return;
      setEntries(res.entries || []);
      setLoadErr("");
    }).catch(() => {
      if (n !== seq.current) return;
      setLoadErr("Couldn't load the history.");
    });
  }

  useEffect(() => { if (!hidden) load(); }, [hidden]);

  // A removal writes an exit, a restore or forget updates one (ADR-0048).
  useEffect(() => subscribeFeed((ev) => {
    if (hiddenRef.current) return;
    const t = String(ev.type || "");
    if (t === "feed.open" || t === "feed.reset" || t.startsWith("agent_exit.")) load();
  }), []);

  const wsChoices = useMemo(() => historyWorkspaceOptions(workspaces), [workspaces]);
  const filterWs = useMemo(() => {
    const seen = new Map();
    for (const e of entries || []) {
      const ex = e.exit;
      if (!seen.has(ex.workspaceId)) seen.set(ex.workspaceId, ex.workspaceId === FREE ? "Free agents" : ex.workspaceName || ex.workspaceId);
    }
    return [...seen.entries()];
  }, [entries]);
  const filterCli = useMemo(() => [...new Set((entries || []).map((e) => e.exit.cli || "pi"))], [entries]);
  const shown = (entries || []).filter((e) => (!ws || e.exit.workspaceId === ws) && (!cli || (e.exit.cli || "pi") === cli));
  const filtered = !!(ws || cli);

  async function restore(e) {
    const ex = e.exit;
    const wsId = historyWorkspaceChoice(e, target[ex.id]);
    if (!wsId) return;
    setBusy(ex.id);
    try {
      const res = await api("/api/agent-history/" + encodeURIComponent(ex.id) + "/restore", post({ workspaceId: wsId }));
      setEntries((list) => (list || []).filter((x) => x.exit.id !== ex.id));
      setOpen("");
      const keys = res.envKeys || [];
      const note = keys.length ? ` Set ${keys.join(", ")} again in its launch settings.` : "";
      if (res.resume && res.terminalId) {
        try {
          await api("/api/terminals/" + encodeURIComponent(res.terminalId) + "/launch/start", post({ confirm: false, resume: true }));
          toast.ok(`"${ex.agentName}" is back, resuming its conversation.` + note);
        } catch (err) {
          toast.error(`"${ex.agentName}" is back, but its CLI didn't start: ${(err && err.message) || err}`);
        }
        location.hash = termHash(res.terminalId);
      } else if (res.agent) {
        // A stopped Pi agent shows an empty chat; starting it loads the
        // conversation the toast promises.
        try {
          await api("/api/agents/" + encodeURIComponent(res.agent.id) + "/managed/start", { method: "POST" });
          toast.ok(`"${ex.agentName}" is back with its conversation.` + note);
        } catch (err) {
          toast.error(`"${ex.agentName}" is back, but it didn't start: ${(err && err.message) || err}. Press Run to open its conversation.`);
        }
        if (onOpenAgent) onOpenAgent(res.agent.id);
      }
    } catch (err) {
      toastError(err);
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
      if (open === ex.id) setOpen("");
    } catch (err) {
      toastError(err);
    } finally {
      setBusy("");
    }
  }

  const firstLoad = !entries && !loadErr;
  const empty = entries && shown.length === 0;

  return (
    <PageFrame id="history-view" title="Agent history" hidden={hidden}>
      <p className="hist-lede">Removed agents whose conversation is still on disk. Bring one back to pick up where it stopped.</p>

      {loadErr ? (
        <p className="outc-err" role="alert">{loadErr} <button type="button" className="btn-link" onClick={load}>Retry</button></p>
      ) : null}

      {entries && entries.length ? (
        <div className="outc-filters" role="group" aria-label="Filter removed agents">
          <select className="set-select" value={ws} onChange={(ev) => setWs(ev.target.value)} aria-label="Workspace">
            <option value="">All workspaces</option>
            {filterWs.map(([id, name]) => <option key={id} value={id}>{name}</option>)}
          </select>
          <select className="set-select" value={cli} onChange={(ev) => setCli(ev.target.value)} aria-label="CLI">
            <option value="">All CLIs</option>
            {filterCli.map((c) => <option key={c} value={c}>{cliName(c)}</option>)}
          </select>
        </div>
      ) : null}

      {firstLoad ? (
        <ul className="outc-list" aria-hidden="true">
          {[0, 1, 2].map((i) => <li key={i} className="outc-row outc-skel"><span className="skel-line w-70" /></li>)}
        </ul>
      ) : empty ? (
        filtered ? (
          <p className="outc-empty">Nothing matches these filters. <button type="button" className="btn-link" onClick={() => { setWs(""); setCli(""); }}>Clear filters</button></p>
        ) : (
          <p className="outc-empty">No removed agents to bring back — when you remove one, it waits here while its conversation exists. <a className="btn-link" href="#/outcomes">Open Outcomes</a></p>
        )
      ) : entries ? (
        <>
          <div className="outc-row-head outc-colhead hist-row-head" aria-hidden="true">
            <span />
            <span>Agent</span>
            <span>CLI</span>
            <span>Workspace</span>
            <span>Conversation</span>
            <span className="outc-when">Removed</span>
          </div>
          <ul className="outc-list">
            {shown.map((e) => {
              const ex = e.exit;
              const isOpen = open === ex.id;
              const s = e.session || {};
              const title = s.name || s.preview || "";
              return (
                <li key={ex.id} className={"outc-row" + (isOpen ? " is-open" : "") + (busy === ex.id ? " is-busy" : "")}>
                  <button type="button" className="outc-row-head hist-row-head" aria-expanded={isOpen} onClick={() => setOpen(isOpen ? "" : ex.id)}>
                    <IconChevronRight className="outc-chev" aria-hidden="true" />
                    <span className="hist-name">
                      <span className="outc-name" title={ex.agentName}>{ex.agentName}</span>
                      {!e.folderExists ? <span className="hist-tag hist-tag-warn">folder missing</span> : !e.workspaceExists ? <span className="hist-tag">workspace removed</span> : null}
                    </span>
                    <span className="outc-cli">{cliName(ex.cli)}{ex.model ? <span className="outc-sub"> · {ex.model}</span> : null}</span>
                    <span className="outc-ws" title={ex.workspaceName}>
                      {ex.workspaceId === FREE ? "Free" : ex.workspaceName || "—"}
                    </span>
                    <span className="hist-conv" title={title}>{title || <span className="outc-sub">Untitled</span>}</span>
                    <span className="outc-when" title={absTime(ex.removedAt)}>{relTime(ex.removedAt)}</span>
                  </button>
                  {isOpen ? (
                    <HistoryDetail
                      entry={e}
                      wsChoices={wsChoices}
                      target={historyWorkspaceChoice(e, target[ex.id])}
                      onTarget={(id) => setTarget((t) => ({ ...t, [ex.id]: id }))}
                      busy={busy === ex.id}
                      onRestore={() => restore(e)}
                      onForget={() => forget(e)}
                    />
                  ) : null}
                </li>
              );
            })}
          </ul>
        </>
      ) : null}
    </PageFrame>
  );
}

function PathFact({ label, path, gone }) {
  return (
    <>
      <dt>{label}</dt>
      <dd className="hist-path">
        <span className="outc-mono hist-path-text">{path || "—"}</span>
        {gone ? <span className="hist-tag hist-tag-warn">missing</span> : null}
        {path ? <button type="button" className="btn-link hist-copy" onClick={() => copyText(path)}>Copy</button> : null}
      </dd>
    </>
  );
}

function HistoryDetail({ entry, wsChoices, target, onTarget, busy, onRestore, onForget }) {
  const ex = entry.exit;
  const s = entry.session || {};
  const blocked = !entry.folderExists
    ? "The folder it worked in is gone. Its conversation resumes only there — recreate the folder to bring it back."
    : !target ? "Its workspace was removed — choose where to bring it back." : "";
  return (
    <div className="outc-detail">
      <div className="outc-group">
        <div className="outc-group-label">Where it lives</div>
        <dl className="outc-facts">
          <PathFact label="Conversation" path={s.path} />
          <PathFact label="Folder" path={entry.folder} gone={!entry.folderExists} />
          <dt>Last active</dt>
          <dd>{s.updatedAt ? <span title={absTime(s.updatedAt)}>{relTime(s.updatedAt)}</span> : "—"}{s.size ? <span className="outc-sub"> · {fmtBytes(s.size)}</span> : null}{s.messages ? <span className="outc-sub"> · {s.messages} {s.messages === 1 ? "message" : "messages"}</span> : null}</dd>
          {s.preview ? (<><dt>First message</dt><dd className="outc-note">{s.preview}</dd></>) : null}
        </dl>
      </div>
      <div className="hist-restore" data-align-row data-align-wrap>
        <label className="hist-target">
          <span>Bring back to</span>
          <select className="set-select" value={target} onChange={(ev) => onTarget(ev.target.value)} aria-label="Workspace to bring it back to" disabled={busy || !entry.folderExists}>
            {!target ? <option value="">Choose a workspace…</option> : null}
            {wsChoices.map(([id, name]) => <option key={id} value={id}>{name}{id === ex.workspaceId ? " (where it was)" : ""}</option>)}
          </select>
        </label>
        <button type="button" className="btn btn-primary" disabled={busy || !!blocked} onClick={onRestore}>{busy ? "Bringing back…" : "Bring back"}</button>
        <button type="button" className="btn btn-ghost btn-danger" disabled={busy} onClick={onForget}>Remove from history</button>
      </div>
      {blocked ? <p className="hist-blocked">{blocked}</p> : null}
    </div>
  );
}
