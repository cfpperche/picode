import { useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis } from "recharts";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { agentsOf, displayAgentName } from "@picode/shared/domain/tree.js";
import { agentRowStatus, agentStatusLabel, agentTerm } from "@picode/shared/domain/agentStatus.js";
import { terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import PageFrame from "./PageFrame.jsx";
import DateRangePicker from "./DateRangePicker.jsx";
import "../styles/workspace-overview.css";

const empty = { git: null, pr: null, inbox: null, missions: null, stats: null };
const paths = (id, range, ws) => ({
  git: `/api/workspaces/${encodeURIComponent(id)}/gitstatus`,
  ...(ws.remote?.kind === "github" ? { pr: `/api/workspaces/${encodeURIComponent(id)}/pr` } : {}),
  inbox: "/api/inbox?blocking=1",
  missions: `/api/missions?workspace=${encodeURIComponent(id)}`,
  stats: `/api/workspaces/${encodeURIComponent(id)}/stats?range=${encodeURIComponent(range)}`,
});

function Block({ title, href, action, pending, error, retry, children }) {
  return <section className="wo-block" aria-label={title}>
    <div className="wo-block-head"><h3>{title}</h3>{href ? <a href={href}>{action || "View all"}</a> : null}</div>
    {pending ? <div className="wo-skeleton" aria-label={`Loading ${title}`}><span /><span /></div>
      : error ? <p className="wo-empty">Could not load {title.toLowerCase()}. <button type="button" onClick={retry}>Retry</button></p>
      : children}
  </section>;
}

export default function WorkspaceOverview({ id, workspaces, terminals, loaded, workingIds, waitingId, onOpenAgent, onOpenTerm, onOpenTree, onOpenGit, onNewAgent }) {
  const ws = workspaces.find((w) => w.id === id);
  const [range, setRange] = useState("7d");
  const [data, setData] = useState(empty);
  const [errors, setErrors] = useState({});
  const [refresh, setRefresh] = useState(0);
  const agents = useMemo(() => ws ? agentsOf(ws) : [], [ws]);
  const owned = new Set(agents.map((a) => a.terminalId).filter(Boolean));
  const shells = terminals.filter((t) => t.workspaceId === id && !owned.has(t.id));
  const retry = () => setRefresh((n) => n + 1);

  useEffect(() => {
    if (!ws) return;
    let alive = true;
    const endpoints = paths(id, range, ws);
    for (const [key, url] of Object.entries(endpoints)) {
      api(url).then((value) => {
        if (!alive) return;
        setData((cur) => ({ ...cur, [key]: value }));
        setErrors((cur) => ({ ...cur, [key]: false }));
      }).catch(() => { if (alive) setErrors((cur) => ({ ...cur, [key]: true })); });
    }
    return () => { alive = false; };
  }, [id, range, refresh, !!ws]);

  useEffect(() => {
    if (!ws) return;
    let timer;
    const unsub = subscribeFeed((event) => {
      const type = String(event.type || "");
      if (type === "feed.open" || type === "git.updated" || type.startsWith("inbox.") || type.startsWith("mission.")) {
        clearTimeout(timer);
        timer = setTimeout(retry, 300);
      }
    });
    const poll = setInterval(() => { if (!document.hidden) retry(); }, 60_000);
    return () => { unsub(); clearTimeout(timer); clearInterval(poll); };
  }, [id, !!ws]);

  if (!ws && !loaded) return <PageFrame id="workspace-overview" title="Loading workspace"><div className="wo-skeleton" aria-label="Loading workspace"><span /><span /></div></PageFrame>;
  if (!ws) return <PageFrame id="workspace-overview" title="Workspace unavailable"><p className="wo-empty">This workspace is no longer available. <a href="#/">Open dashboard</a></p></PageFrame>;
  const inbox = (data.inbox?.items || []).filter((item) => item.workspaceId === id && item.state !== "done").slice(0, 3);
  const missions = (data.missions?.missions || []).filter((m) => !m.archived && m.state !== "accepted").slice(0, 3);
  const git = data.git;
  const pr = data.pr?.status === "ok" ? data.pr.pr : null;
  const stats = data.stats;

  return <PageFrame id="workspace-overview" title={ws.name} context={ws.path}>
    <div className="wo-content">
      <div className="wo-grid">
        <Block title="Needs you" href="#/app/inbox" action="Open Inbox" pending={!data.inbox && !errors.inbox} error={errors.inbox} retry={retry}>
          {inbox.length ? <ul className="wo-list">{inbox.map((item) => <li key={item.id}><a href={`#/app/inbox/item/${encodeURIComponent(item.id)}`}><strong>{item.title}</strong><small>{relTime(item.createdAt)}</small></a></li>)}</ul>
            : <p className="wo-empty">Nothing needs your answer. <a href="#/app/inbox">Open Inbox</a></p>}
        </Block>
        <Block title="Project" pending={!git && !errors.git} error={errors.git} retry={retry}>
          {git?.git ? <div className="wo-project">
            <p><strong>{git.branch || "Detached checkout"}</strong><span>{git.totals?.files || 0} changed files</span></p>
            <p>{git.ahead ? `↑${git.ahead} ahead` : ""}{git.behind ? ` ↓${git.behind} behind` : ""}</p>
            {pr?.url ? <p><a href={pr.url} target="_blank" rel="noopener noreferrer">Pull request #{pr.number} · {pr.state}</a></p> : null}
            <div className="wo-actions"><button type="button" onClick={() => onOpenGit(id, ws.name)}>Open Git</button><button type="button" onClick={() => onOpenTree(id, ws.name)}>Open files</button></div>
          </div> : <p className="wo-empty">No Git repository here. <button type="button" onClick={() => onOpenTree(id, ws.name)}>Open files</button></p>}
        </Block>
        <Block title="Agents" pending={false} href={null}>
          {agents.length || shells.length ? <ul className="wo-list">{agents.map((agent) => {
            const term = agentTerm(agent, terminals);
            const status = agentRowStatus(agent, { workingIds, waitingId, term });
            return <li key={agent.id}><button type="button" onClick={() => onOpenAgent(agent.id)}><strong>{displayAgentName(agent, ws)}</strong><small>{agent.cli || "Pi"} · {agentStatusLabel(status)}</small></button></li>;
          })}{shells.map((term) => <li key={term.id}><button type="button" onClick={() => onOpenTerm(term.id)}><strong>{term.name || "Terminal"}</strong><small>{terminalStatusLabel(term) || terminalStatus(term)}</small></button></li>)}</ul>
            : <p className="wo-empty">No agents here yet. <button type="button" onClick={() => onNewAgent(id)}>Create agent</button></p>}
        </Block>
        <Block title="Missions" href={`#/missions?workspace=${encodeURIComponent(id)}`} pending={!data.missions && !errors.missions} error={errors.missions} retry={retry}>
          {missions.length ? <ul className="wo-list">{missions.map((m) => <li key={m.id}><a href={`#/mission/${encodeURIComponent(m.id)}`}><strong>{m.title}</strong><small>{m.state}</small></a></li>)}</ul>
            : <p className="wo-empty">No active missions. <a href={`#/missions?workspace=${encodeURIComponent(id)}`}>Open Missions</a></p>}
        </Block>
      </div>
      <section className="wo-usage" aria-label="Workspace usage">
        <div className="wo-usage-head"><h3>Agent activity</h3><DateRangePicker name="workspace-range" value={range} onChange={setRange} /></div>
        {!stats && !errors.stats ? <div className="wo-skeleton" aria-label="Loading agent activity"><span /><span /></div>
          : errors.stats ? <p className="wo-empty">Could not load agent activity. <button type="button" onClick={retry}>Retry</button></p>
          : <>
            {stats.range !== range ? <p className="wo-note" role="status">Showing the previous period while updating…</p> : null}
            <div className="wo-totals"><p><strong>{stats.current?.sessions || 0}</strong><span>sessions</span></p><p><strong>{stats.current?.messages || 0}</strong><span>messages</span></p><p><strong>{formatMoney(stats.current?.cost || 0, "usd")}</strong><span>reported or estimated cost</span></p></div>
            {stats.current?.messages ? <BarChart data={stats.series || []} responsive accessibilityLayer style={{ width: "100%", height: 180 }} margin={{ top: 12, right: 8, bottom: 0, left: 8 }}>
              <CartesianGrid vertical={false} stroke="var(--border)" /><XAxis dataKey="date" tickLine={false} axisLine={false} tickFormatter={(v) => v.slice(-5)} stroke="var(--text-secondary)" /><Tooltip /><Bar dataKey="messages" name="Messages" fill="var(--accent)" radius={[3, 3, 0, 0]} isAnimationActive={false} />
            </BarChart> : <p className="wo-empty">No recorded activity in this period. <button type="button" onClick={() => setRange("all")}>Show all time</button></p>}
            {stats.current?.estimated ? <p className="wo-note">{formatMoney(stats.current.estimated, "usd")} uses list price estimates.</p> : null}
            {stats.coverage?.some((row) => row.signals?.cost === "partial" || row.signals?.cost === "not-reported" || row.signals?.cost === "unavailable") ? <p className="wo-note">Some CLIs do not report complete costs.</p> : null}
          </>}
      </section>
    </div>
  </PageFrame>;
}
