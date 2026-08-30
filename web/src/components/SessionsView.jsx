import { useCallback, useEffect, useMemo, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { api } from "../lib/api.js";
import { askConfirm, fmtBytes } from "../lib/confirm.js";
import { toast } from "../lib/toast.js";
import PageFrame from "./PageFrame.jsx";

function fmtAge(iso) {
  const t = Date.parse(iso || "");
  if (!t) return "—";
  const s = Math.max(0, (Date.now() - t) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return Math.round(s / 60) + "m ago";
  if (s < 86400) return Math.round(s / 3600) + "h ago";
  return Math.round(s / 86400) + "d ago";
}

function baseName(path) {
  return String(path || "").split("/").pop() || path;
}

const CLEANUP_OPTIONS = [
  { v: 0, label: "Off" },
  { v: 30, label: "30 days" },
  { v: 60, label: "60 days" },
  { v: 90, label: "90 days" },
];

// One view for every Pi session under the workspace folder (A) plus the
// orphan auto-clean preference (B). In-use = the current session of an
// agent; everything else is an orphan you can open, adopt or delete.
export default function SessionsView({ wsId, workspace, agents, onOpenAgent, onCompactAgent }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [openPick, setOpenPick] = useState(null); // session being opened
  const [pickAgent, setPickAgent] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!wsId) return;
    setError("");
    try {
      setData(await api("/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/manage"));
    } catch (e) {
      setError(e && e.message ? e.message : "Could not load sessions.");
      setData(null);
    }
  }, [wsId]);

  useEffect(() => { setData(null); load(); }, [load]);

  const sessions = useMemo(() => (data && data.sessions) || [], [data]);
  const agentsHere = useMemo(() => agents || [], [agents]);

  async function onDelete(s) {
    const ok = await askConfirm({
      title: "Delete session",
      message: baseName(s.path) + " (" + fmtBytes(s.size) + ") is deleted from disk. This cannot be undone.",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api("/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/manage", {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: s.path }),
      });
      toast.ok("Session deleted.");
      await load();
    } catch (e) {
      toast.error((e && e.message) || "Delete failed.");
    } finally {
      setBusy(false);
    }
  }

  async function onCleanup(days) {
    setBusy(true);
    try {
      const res = await api("/api/session-cleanup", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ days }),
      });
      toast.ok(days === 0
        ? "Auto-clean off. Sessions are kept forever."
        : "Auto-clean: orphan sessions untouched for " + days + " days are removed." + (res && res.removed ? " Removed " + res.removed + " now." : ""));
      await load();
    } catch (e) {
      toast.error((e && e.message) || "Could not save the auto-clean setting.");
    } finally {
      setBusy(false);
    }
  }

  async function doOpenWith() {
    if (!openPick || !pickAgent) return;
    setBusy(true);
    try {
      await api("/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/resume?agent=" + encodeURIComponent(pickAgent), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: openPick.path }),
      });
      setOpenPick(null);
      setPickAgent("");
      onOpenAgent(pickAgent);
    } catch (e) {
      toast.error((e && e.message) || "Could not open the session.");
    } finally {
      setBusy(false);
    }
  }

  const total = data ? data.totalBytes : 0;

  return (
    <PageFrame id="sessions-view" title={(workspace ? workspace.name + " · " : "") + "Sessions"} wide>
      <div className="sessions-toolbar" data-align-row>
        <span className="sessions-total">{sessions.length} {sessions.length === 1 ? "session" : "sessions"} · {fmtBytes(total)} on disk</span>
        <div className="sessions-actions" data-align-row>
          <label className="sessions-cleanup" title="Orphan sessions (not the current session of any agent) are deleted after this many days.">
            Auto-clean orphans
            <select
              value={data ? String(data.cleanupDays) : "0"}
              disabled={!data || busy}
              onChange={(e) => onCleanup(Number(e.target.value))}
            >
              {CLEANUP_OPTIONS.map((o) => <option key={o.v} value={String(o.v)}>{o.label}</option>)}
            </select>
          </label>
          <button type="button" className="btn btn-sm" onClick={load} disabled={busy}>Refresh</button>
        </div>
      </div>

      {error ? (
        <p className="file-pane-msg">{error} <button type="button" className="btn btn-sm" onClick={load}>Retry</button></p>
      ) : !data ? (
        <p className="file-pane-msg">Loading sessions…</p>
      ) : sessions.length === 0 ? (
        <div className="empty-card">
          <h2>No Pi sessions yet</h2>
          <p>Sessions appear here as agents chat in this folder.</p>
        </div>
      ) : (
        <ul className="mcp-list sessions-list">
          {sessions.map((s) => (
            <li key={s.path} className={"mcp-row sess-row" + (s.inUseBy ? "" : " orphan")}>
              <div className="mcp-row-main">
                <strong className="sess-name" title={s.path}>{baseName(s.path)}</strong>
                {s.inUseBy ? (
                  <span className="sess-badge in-use" title={"Current session of " + s.inUseBy.agentName}>in use · {s.inUseBy.agentName}</span>
                ) : (
                  <span className="sess-badge">free</span>
                )}
                {s.model ? <span className="sess-meta">{s.provider}/{s.model}</span> : null}
              </div>
              <div className="sess-facts">
                <span title="Last update">{fmtAge(s.updatedAt)}</span>
                <span>{fmtBytes(s.size)}</span>
                <span>{s.messages ? s.messages.toLocaleString() + " msgs" : "—"}</span>
                {s.cost > 0 ? <span>{"$" + s.cost.toFixed(2)}</span> : null}
              </div>
              <div className="mcp-row-actions" data-align-row>
                <button type="button" className="btn btn-ghost btn-sm" onClick={() => { setOpenPick(s); setPickAgent((agentsHere[0] || {}).id || ""); }} disabled={busy || agentsHere.length === 0} title={agentsHere.length ? "Resume this session with one of this workspace's agents" : "This workspace has no agents yet"}>Open with…</button>
                {s.inUseBy ? (
                  <button type="button" className="btn btn-ghost btn-sm" onClick={() => onCompactAgent(s.inUseBy.agentId)} title={"Compact " + s.inUseBy.agentName + " (summarizes older turns)"}>Compact</button>
                ) : null}
                <button
                  type="button"
                  className="btn btn-ghost btn-sm danger"
                  onClick={() => onDelete(s)}
                  disabled={!!s.inUseBy || busy}
                  title={s.inUseBy ? "In use by " + s.inUseBy.agentName + " — point that agent at another session first" : "Delete this file from disk"}
                >
                  Delete
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <Dialog.Root open={!!openPick} onOpenChange={(o) => { if (!o) setOpenPick(null); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-open-session" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">Open {openPick ? baseName(openPick.path) : ""}</Dialog.Title>
            <Dialog.Description className="dlg-body">The agent switches to this session (no copy is made).</Dialog.Description>
            <label className="sessions-pick">
              <span>Agent</span>
              <select value={pickAgent} onChange={(e) => setPickAgent(e.target.value)} autoFocus>
                {agentsHere.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
              </select>
            </label>
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-primary btn-sm" onClick={doOpenWith} disabled={!pickAgent || busy}>Open</button>
              <button type="button" className="btn btn-ghost btn-sm" onClick={() => setOpenPick(null)}>Cancel</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}
