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

function SessionRow({ s, agentsForOpen, busy, onOpen, onDelete, onCompact }) {
  const canOpen = agentsForOpen.length > 0;
  return (
    <li className={"mcp-row sess-row" + (s.inUseBy ? "" : " orphan")}>
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
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={() => onOpen(s)}
          disabled={busy || !canOpen}
          title={canOpen ? "Switch one of this folder's agents to this session (no copy)" : "This folder is not a PiCode workspace"}
        >
          Open with…
        </button>
        {s.inUseBy ? (
          <button type="button" className="btn btn-ghost btn-sm" onClick={() => onCompact(s.inUseBy.agentId)} title={"Compact " + s.inUseBy.agentName + " (summarizes older turns)"}>Compact</button>
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
  );
}

// Two scopes, one view: a workspace folder (#/sessions/<id>) or every Pi
// session on this machine (#/sessions), grouped by folder. Orphan = not the
// current session of any agent; only orphans can be deleted.
export default function SessionsView({ wsId, workspace, agents, workspaces, onOpenAgent, onCompactAgent }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [openPick, setOpenPick] = useState(null); // { session, resumeWsId }
  const [pickAgent, setPickAgent] = useState("");
  const [busy, setBusy] = useState(false);
  const all = !wsId;

  const load = useCallback(async () => {
    setError("");
    try {
      setData(all ? await api("/api/sessions/all") : await api("/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/manage"));
    } catch (e) {
      setError(e && e.message ? e.message : "Could not load sessions.");
      setData(null);
    }
  }, [wsId, all]);

  useEffect(() => { setData(null); load(); }, [load]);

  const sessions = useMemo(() => (data && data.sessions) || [], [data]);

  // All mode: group by folder, tag workspace membership; Open with… uses the
  // agents of the workspace owning that folder (when there is one).
  const groups = useMemo(() => {
    if (!all) {
      return [{ key: "ws", label: "", items: sessions, agentsForOpen: agents || [], resumeWsId: wsId }];
    }
    const wsByPath = new Map((workspaces || []).map((w) => [w.path, w]));
    const byCwd = new Map();
    for (const s of sessions) {
      const k = s.cwd || "(unknown folder)";
      if (!byCwd.has(k)) byCwd.set(k, []);
      byCwd.get(k).push(s);
    }
    return [...byCwd.entries()].map(([cwd, items]) => {
      const ws = wsByPath.get(cwd) || null;
      return {
        key: cwd,
        label: (ws ? ws.name + " · " : "") + cwd,
        ws,
        items,
        agentsForOpen: (ws && ws.agents) || [],
        resumeWsId: ws ? ws.id : "",
      };
    }).sort((a, b) => (a.ws ? 0 : 1) - (b.ws ? 0 : 1) || b.items.length - a.items.length);
  }, [all, sessions, agents, wsId, workspaces]);

  async function onDelete(s) {
    const ok = await askConfirm({
      title: "Delete session",
      message: baseName(s.path) + " (" + fmtBytes(s.size) + ") is deleted from disk. This cannot be undone.",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api(all ? "/api/sessions/all" : "/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/manage", {
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
    if (!openPick || !pickAgent || !openPick.resumeWsId) return;
    setBusy(true);
    try {
      await api("/api/workspaces/" + encodeURIComponent(openPick.resumeWsId) + "/sessions/resume?agent=" + encodeURIComponent(pickAgent), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: openPick.session.path }),
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
        <span className="sessions-total">{all ? "All folders · " : ""}{sessions.length} {sessions.length === 1 ? "session" : "sessions"} · {fmtBytes(total)} on disk</span>
        <div className="sessions-actions" data-align-row>
          {!all ? (
            <a className="sessions-scope-link" href="#/sessions" title="Every Pi session on this machine, grouped by folder">All folders →</a>
          ) : null}
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
          <p>Sessions appear here as agents chat{all ? " in each folder" : " in this folder"}.</p>
        </div>
      ) : (
        groups.map((g) => (
          <section key={g.key} className="sessions-group">
            {all ? (
              <header className={"sessions-group-head" + (g.ws ? "" : " unknown")}>
                <span className="sessions-group-name" title={g.key}>{g.label}</span>
                {g.ws ? null : <span className="sess-badge">not a workspace</span>}
              </header>
            ) : null}
            <ul className="mcp-list sessions-list">
              {g.items.map((s) => (
                <SessionRow
                  key={s.path}
                  s={s}
                  agentsForOpen={g.agentsForOpen}
                  busy={busy}
                  onOpen={(sess) => { setOpenPick({ session: sess, resumeWsId: g.resumeWsId, agents: g.agentsForOpen }); setPickAgent((g.agentsForOpen[0] || {}).id || ""); }}
                  onDelete={onDelete}
                  onCompact={onCompactAgent}
                />
              ))}
            </ul>
          </section>
        ))
      )}

      <Dialog.Root open={!!openPick} onOpenChange={(o) => { if (!o) setOpenPick(null); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-open-session" onCloseAutoFocus={(e) => e.preventDefault()}>
            <Dialog.Title className="dlg-title">Open {openPick ? baseName(openPick.session.path) : ""}</Dialog.Title>
            <Dialog.Description className="dlg-body">The agent switches to this session (no copy is made).</Dialog.Description>
            <label className="sessions-pick">
              <span>Agent</span>
              <select value={pickAgent} onChange={(e) => setPickAgent(e.target.value)} autoFocus>
                {(openPick ? openPick.agents || [] : []).map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
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
