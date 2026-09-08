import { useCallback, useEffect, useMemo, useState } from "react";
import * as Dialog from "./ResponsiveDialog.jsx";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { handoffTargets, lineageBadges, sessionClis } from "@picode/shared/domain/sessionHandoff.js";
import { askConfirm, fmtBytes } from "../lib/confirm.js";
import { toast, toastError } from "../lib/toast.js";
import { termHash } from "../lib/routes.js";
import PageFrame from "./PageFrame.jsx";
import CliCombo from "./CliCombo.jsx";
import SessionHandoffDialog from "./SessionHandoffDialog.jsx";

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

// Lineage (ADR-0088): where a session came from and where it continued.
function LineageBadges({ s, cliNames, onOpenAgent }) {
  const badges = lineageBadges(s.handoff, cliNames);
  if (!badges.length) return null;
  return badges.map((b, i) => {
    const ref = b.ref || {};
    const title = (b.kind === "from" ? "Translated from " : "Continued in ") + (cliNames[ref.cli] || ref.cli) + (ref.id ? " · " + String(ref.id).slice(0, 8) : "") + (ref.mode ? " · " + ref.mode : "");
    let inner = b.label;
    if (ref.terminalId) inner = <a href={termHash(ref.terminalId)}>{b.label}</a>;
    else if (ref.agentId && onOpenAgent) inner = <a href="#" onClick={(e) => { e.preventDefault(); onOpenAgent(ref.agentId); }}>{b.label}</a>;
    return <span key={b.kind + i} className="sess-badge lineage" title={title}>{inner}</span>;
  });
}

// "Continue in <CLI>…" — one item per target the server advertises for
// this session's CLI (ADR-0088). Empty list: a disabled item says why.
function HandoffMenu({ s, targets, busy, onHandoff }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button type="button" className="btn btn-ghost btn-sm" aria-label={"More actions for " + (s.name || s.id)} disabled={busy}>•••</button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="um-popover" align="end" sideOffset={5} collisionPadding={12}>
          {targets.length ? targets.map((t) => (
            <DropdownMenu.Item key={t.id} className="um-item" disabled={!t.installed} title={t.installed ? "" : t.name + " is not installed"} onSelect={() => onHandoff(s, t)}>
              Continue in {t.name}…
            </DropdownMenu.Item>
          )) : (
            <DropdownMenu.Item className="um-item" disabled title="No other CLI can read this session and receive a handoff yet.">No other CLI can receive this session yet</DropdownMenu.Item>
          )}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

function PiRow({ s, agentsForOpen, busy, onOpen, onDelete, onCompact, targets, onHandoff, cliNames, onOpenAgent }) {
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
        <LineageBadges s={s} cliNames={cliNames} onOpenAgent={onOpenAgent} />
        {s.model ? <span className="sess-meta">{s.model}</span> : null}
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
        <HandoffMenu s={s} targets={targets} busy={busy} onHandoff={onHandoff} />
      </div>
    </li>
  );
}

function CliRow({ s, cliName, busy, onOpenTerminal, targets, onHandoff, cliNames, onOpenAgent }) {
  return (
    <li className="mcp-row sess-row orphan">
      <div className="mcp-row-main">
        <strong className="sess-name" title={s.name || s.path}>{s.name || s.id}</strong>
        {s.workspace ? <span className="sess-badge in-use">{s.workspace}</span> : null}
        <LineageBadges s={s} cliNames={cliNames} onOpenAgent={onOpenAgent} />
        {s.model ? <span className="sess-meta">{s.model}</span> : null}
      </div>
      <div className="sess-facts">
        <span title="Last update">{fmtAge(s.updatedAt)}</span>
        <span>{fmtBytes(s.size)}</span>
        <span>{s.messages ? s.messages.toLocaleString() + " msgs" : "—"}</span>
      </div>
      <div className="mcp-row-actions" data-align-row>
        {s.preview ? <span className="sess-preview" title={s.preview}>{s.preview}</span> : null}
        <button
          type="button"
          className="btn btn-ghost btn-sm"
          onClick={() => onOpenTerminal(s)}
          disabled={busy}
          title={"Launch a " + cliName + " terminal in this session's folder with " + (s.resumeArgs || []).join(" ")}
        >
          Open in terminal
        </button>
        <HandoffMenu s={s} targets={targets} busy={busy} onHandoff={onHandoff} />
      </div>
    </li>
  );
}

// Sessions surface (ADR-0079): one view per CLI. Pi keeps its management
// actions (open with, compact, delete, auto-clean); other CLIs list their
// on-disk sessions and open them in a terminal with the CLI's verified
// resume arguments. Any session can continue in another CLI (ADR-0088):
// the targets come from the capabilities /api/clis advertises, never from
// a list kept here.
export default function SessionsView({ wsId, workspace, agents, workspaces, onOpenAgent, onCompactAgent, embedded = false, cli = "pi", onCliChange, cliNames = {}, clis = [], wsReady = true }) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const [openPick, setOpenPick] = useState(null); // { session, resumeWsId }
  const [pickAgent, setPickAgent] = useState("");
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [handoff, setHandoff] = useState(null); // { session, target }
  const all = !wsId;
  const isPi = cli === "pi";
  const cliName = cliNames[cli] || cli;
  const pickerClis = useMemo(() => { const ids = sessionClis(clis); return ids.includes(cli) ? ids : [cli, ...ids]; }, [clis, cli]);
  const targets = useMemo(() => handoffTargets(clis, cli), [clis, cli]);

  const load = useCallback(async () => {
    setError("");
    try {
      if (isPi) {
        setData(all ? await api("/api/clis/pi/sessions") : await api("/api/clis/pi/sessions?workspace=" + encodeURIComponent(wsId)));
      } else {
        // Non-Pi scoping filters by folder, so the workspace must have
        // resolved first; a scope that never resolves is an honest error,
        // never a silently unfiltered list.
        if (!all && !(workspace && workspace.path)) {
          if (wsReady) { setError("That workspace is gone."); setData({ sessions: [] }); }
          return;
        }
        const q = workspace && workspace.path ? "?cwd=" + encodeURIComponent(workspace.path) : "";
        setData(await api("/api/clis/" + encodeURIComponent(cli) + "/sessions" + q));
      }
    } catch (e) {
      setError(e && e.message ? e.message : "Could not load sessions.");
      setData(null);
    }
  }, [wsId, all, isPi, cli, workspace, wsReady]);

  useEffect(() => { setData(null); setQuery(""); load(); }, [load]);

  // A handoff anywhere changes lineage badges here; refetch on its event.
  useEffect(() => {
    let timer;
    const unsub = subscribeFeed((e) => {
      if (e.type === "session.handoff" || e.type === "feed.reset") { clearTimeout(timer); timer = setTimeout(load, 120); }
    });
    return () => { unsub(); clearTimeout(timer); };
  }, [load]);

  const sessions = useMemo(() => (data && data.sessions) || [], [data]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return sessions;
    return sessions.filter((s) => [s.name, s.preview, s.cwd, s.path, s.id, s.model].some((v) => String(v || "").toLowerCase().includes(q)));
  }, [sessions, query]);

  // All mode: group by folder, tag workspace membership; Open with… uses the
  // agents of the workspace owning that folder (when there is one).
  const groups = useMemo(() => {
    if (!all) {
      return [{ key: "ws", label: "", items: filtered, agentsForOpen: agents || [], resumeWsId: wsId }];
    }
    const wsByPath = new Map((workspaces || []).map((w) => [w.path, w]));
    const byCwd = new Map();
    for (const s of filtered) {
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
  }, [all, filtered, agents, wsId, workspaces]);

  async function onDelete(s) {
    const ok = await askConfirm({
      title: "Delete session",
      message: baseName(s.path) + " (" + fmtBytes(s.size) + ") is deleted from disk. This cannot be undone.",
      confirmLabel: "Delete",
    });
    if (!ok) return;
    setBusy(true);
    try {
      await api("/api/clis/pi/sessions/delete", {
        method: "POST",
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
      const res = await api("/api/clis/pi/sessions/cleanup", {
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

  // Non-Pi resume: a CLI terminal born in the session's folder, launched
  // with the server-verified resume arguments for that session. Launch
  // argument overrides replace the CLI defaults for this one terminal.
  async function onOpenTerminal(s) {
    setBusy(true);
    try {
      const t = await api("/api/clis/" + encodeURIComponent(cli) + "/terminals", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: cliName + " · " + (s.name || s.id).slice(0, 40),
          cwd: s.cwd,
          workspaceId: s.workspaceId || "",
          overrides: { args: s.resumeArgs || [] },
        }),
      });
      if (t && t.launchError) toastError(new Error(t.launchError));
      else { toast.ok(cliName + " opening in this session's folder."); location.hash = termHash(t.id); }
      await load();
    } catch (e) {
      toastError(e);
    } finally {
      setBusy(false);
    }
  }

  // After a handoff: say where it went and go there (ADR-0088).
  function onHandoffDone(res, target) {
    setHandoff(null);
    const term = res && res.terminal;
    const agent = res && res.agent;
    if (term && term.launchError) {
      toastError(new Error(term.launchError));
    } else if (term && term.id) {
      toast.ok(target.name + " is opening with this conversation.");
      location.hash = termHash(term.id);
    } else if (agent && agent.id) {
      toast.ok("Continued as a Pi agent: " + agent.name + ".");
      if (onOpenAgent) onOpenAgent(agent.id);
    } else {
      toast.ok("Handoff recorded.");
    }
    load();
  }

  const total = data ? data.totalBytes : 0;

  return (
    <PageFrame id="sessions-view" title={(workspace ? workspace.name + " · " : "") + "Sessions"} wide embedded={embedded}>
      <div className="sessions-toolbar" data-align-row>
        <span className="sessions-total">{all ? "All folders · " : (workspace ? workspace.name + " · " : "")}{filtered.length}{query ? " of " + sessions.length : ""} {filtered.length === 1 ? "session" : "sessions"}{isPi ? " · " + fmtBytes(total) + " on disk" : ""}</span>
        <div className="sessions-actions" data-align-row>
          {!all ? (
            <a className="sessions-scope-link" href={"#/clis/sessions" + (cli !== "pi" ? "?cli=" + encodeURIComponent(cli) : "")} title={"Every " + cliName + " session on this machine, grouped by folder"}>All folders →</a>
          ) : null}
          <div className="sessions-cleanup" title="Which CLI's sessions are listed">
            CLI
            <CliCombo ariaLabel="Sessions CLI" value={cli} options={pickerClis.map((id) => ({ id, name: cliNames[id] }))} align="end" onChange={(id) => onCliChange && onCliChange(id)} />
          </div>
          {sessions.length > 6 ? (
            <input className="sessions-search" aria-label="Search sessions" placeholder="Find a session…" value={query} onChange={(e) => setQuery(e.target.value)} />
          ) : null}
          {isPi ? (
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
          ) : null}
          <button type="button" className="btn btn-sm" onClick={load} disabled={busy}>Refresh</button>
        </div>
      </div>

      {error ? (
        <p className="file-pane-msg">{error} <button type="button" className="btn btn-sm" onClick={load}>Retry</button></p>
      ) : !data ? (
        <p className="file-pane-msg">Loading sessions…</p>
      ) : filtered.length === 0 ? (
        <div className="empty-card">
          <h2>{query ? "No matching sessions" : "No " + cliName + " sessions yet"}</h2>
          <p>{query ? "Try a different search." : "Sessions appear here as " + cliName + " runs" + (all ? " in each folder" : " in this folder") + "."}</p>
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
              {g.items.map((s) => isPi ? (
                <PiRow
                  key={s.path}
                  s={s}
                  agentsForOpen={g.agentsForOpen}
                  busy={busy}
                  onOpen={(sess) => { setOpenPick({ session: sess, resumeWsId: g.resumeWsId, agents: g.agentsForOpen }); setPickAgent((g.agentsForOpen[0] || {}).id || ""); }}
                  onDelete={onDelete}
                  onCompact={onCompactAgent}
                  targets={targets}
                  onHandoff={(sess, target) => setHandoff({ session: sess, target })}
                  cliNames={cliNames}
                  onOpenAgent={onOpenAgent}
                />
              ) : (
                <CliRow key={s.cli + ":" + s.id + ":" + s.path} s={s} cliName={cliName} busy={busy} onOpenTerminal={onOpenTerminal} targets={targets} onHandoff={(sess, target) => setHandoff({ session: sess, target })} cliNames={cliNames} onOpenAgent={onOpenAgent} />
              ))}
            </ul>
          </section>
        ))
      )}

      <SessionHandoffDialog
        open={!!handoff}
        session={handoff ? handoff.session : null}
        sourceCli={cli}
        sourceName={cliName}
        target={handoff ? handoff.target : null}
        onClose={() => setHandoff(null)}
        onDone={onHandoffDone}
      />

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
