import { useEffect, useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, Tooltip, XAxis } from "recharts";
import { api } from "@picode/shared/client/api.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { agentsOf } from "@picode/shared/domain/tree.js";
import { agentRowStatus, agentStatusStamp, agentTerm } from "@picode/shared/domain/agentStatus.js";
import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import { activeMissions, activityLabel, agentSummaryRows, attentionItems, recentActivity } from "../lib/workspaceOverviewModel.js";
import PageFrame from "./PageFrame.jsx";
import DateRangePicker from "./DateRangePicker.jsx";
import WorkspaceAgents from "./WorkspaceAgents.jsx";
import "../styles/workspace-overview.css";

const empty = { git: null, graph: null, pr: null, inbox: null, missions: null, stats: null, activity: null };
const paths = (id, range, ws) => ({
  git: `/api/workspaces/${encodeURIComponent(id)}/gitstatus`,
  graph: `/api/workspaces/${encodeURIComponent(id)}/git?limit=10&remotes=0`,
  ...(ws.remote?.kind === "github" ? { pr: `/api/workspaces/${encodeURIComponent(id)}/pr` } : {}),
  inbox: "/api/inbox?blocking=1",
  missions: `/api/missions?workspace=${encodeURIComponent(id)}`,
  activity: `/api/workspaces/${encodeURIComponent(id)}/activity`,
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

export default function WorkspaceOverview({ id, workspaces, terminals, checklists, loaded, workingIds, waitingId, fullAgents = false, onOpenAgent, onOpenTerm, onOpenTree, onOpenGit, onNewAgent }) {
  const ws = workspaces.find((w) => w.id === id);
  const [range, setRange] = useState("7d");
  const [data, setData] = useState(empty);
  const [errors, setErrors] = useState({});
  const [refresh, setRefresh] = useState(0);
  const [lastVisit, setLastVisit] = useState(null);
  const agents = useMemo(() => ws ? agentsOf(ws) : [], [ws]);
  const owned = new Set(agents.map((a) => a.terminalId).filter(Boolean));
  const shells = terminals.filter((t) => t.workspaceId === id && !owned.has(t.id));
  const retry = () => setRefresh((n) => n + 1);

  useEffect(() => {
    const key = `picode.workspaceOverview.visit.${id}`;
    try {
      const previous = Number(window.localStorage.getItem(key));
      setLastVisit(previous > 0 ? previous : null);
      window.localStorage.setItem(key, String(Date.now()));
    } catch { setLastVisit(null); }
  }, [id]);

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
      if (type === "feed.open" || type === "git.updated" || type.startsWith("inbox.") || type.startsWith("mission.") || type.startsWith("agent.")) {
        clearTimeout(timer);
        timer = setTimeout(retry, 300);
      }
    });
    const poll = setInterval(() => { if (!document.hidden) retry(); }, 60_000);
    return () => { unsub(); clearTimeout(timer); clearInterval(poll); };
  }, [id, !!ws]);

  if (!ws && !loaded) return <PageFrame id="workspace-overview" title="Loading workspace"><div className="wo-skeleton" aria-label="Loading workspace"><span /><span /></div></PageFrame>;
  if (!ws) return <PageFrame id="workspace-overview" title="Workspace unavailable"><p className="wo-empty">This workspace is no longer available. <a href="#/">Open dashboard</a></p></PageFrame>;
  const inbox = (data.inbox?.items || []).filter((item) => item.workspaceId === id && item.state !== "done");
  const missions = activeMissions(data.missions?.missions || []);
  const git = data.git;
  const pr = data.pr?.status === "ok" ? data.pr.pr : null;
  const stats = data.stats;
  const statusOf = (agent) => agentRowStatus(agent, { workingIds, waitingId, term: agentTerm(agent, terminals) });
  const agentRows = agentSummaryRows({ agents, missions, checklists, inbox, statusOf, stampOf: (agent, status) => agentStatusStamp(status, agent, agentTerm(agent, terminals)) });
  const attention = attentionItems({ workspaceId: id, inbox, missions, agents, statusOf, pr });
  const activity = recentActivity(data.activity?.items || [], data.graph?.commits || [], lastVisit);
  const visitWithinRetention = lastVisit && lastVisit > Date.now() - 7 * 86400_000;
  const activityHref = (item) => item.kind === "mission" ? `#/mission/${encodeURIComponent(item.entityId)}`
    : item.kind === "inbox" ? `#/inbox/${encodeURIComponent(item.entityId)}`
      : null;

  if (fullAgents) return <PageFrame id="workspace-agents" title={`${ws.name} agents`} context={ws.path}>
    <div className="wo-content"><WorkspaceAgents workspace={ws} rows={agentRows} shells={shells} full missionsError={errors.missions} retry={retry} onOpenAgent={onOpenAgent} onOpenTerm={onOpenTerm} onNewAgent={onNewAgent} /></div>
  </PageFrame>;

  return <PageFrame id="workspace-overview" title={ws.name} context={ws.path}>
    <div className="wo-content">
      <section className="wo-focus" aria-label="Needs you">
        <div className="wo-block-head"><h3>Needs you</h3><a href="#/inbox">Open Inbox</a></div>
        {!data.inbox && !data.missions && !errors.inbox && !errors.missions ? <div className="wo-skeleton" aria-label="Loading attention"><span /><span /></div>
          : attention.length ? <ul className="wo-list">{attention.slice(0, 5).map((item) => <li key={item.key}>
            {item.href ? <a href={item.href} target={item.href.startsWith("http") ? "_blank" : undefined} rel={item.href.startsWith("http") ? "noopener noreferrer" : undefined}><strong>{item.label}</strong><small>{item.detail}</small></a>
              : <button type="button" onClick={() => onOpenAgent(item.agentId)}><strong>{item.label}</strong><small>{item.detail}</small></button>}
          </li>)}</ul>
            : !data.inbox && !data.missions ? <p className="wo-empty">Could not load attention. <button type="button" onClick={retry}>Retry</button></p>
            : <p className="wo-empty">Nothing needs your answer. <a href="#/inbox">Open Inbox</a></p>}
        {(errors.inbox || errors.missions) && (data.inbox || data.missions) ? <p className="wo-note">Some updates are unavailable. <button type="button" onClick={retry}>Retry</button></p> : null}
      </section>
      <div className="wo-grid">
        <Block title="Project" pending={!git && !errors.git} error={errors.git} retry={retry}>
          {git?.git ? <div className="wo-project">
            <p><strong>{git.branch || "Detached checkout"}</strong><span>{git.totals?.files || 0} changed files</span></p>
            <p>{git.ahead ? `↑${git.ahead} ahead` : ""}{git.behind ? ` ↓${git.behind} behind` : ""}</p>
            {pr?.url ? <p><a href={pr.url} target="_blank" rel="noopener noreferrer">Pull request #{pr.number} · {pr.state}</a></p> : null}
            {pr?.checks?.total ? <p><span>{pr.checks.failed ? `${pr.checks.failed} failed` : pr.checks.pending ? `${pr.checks.pending} running` : `${pr.checks.passed} passed`} checks</span><a href={pr.url} target="_blank" rel="noopener noreferrer">View checks</a></p> : null}
            <div className="wo-actions"><button type="button" onClick={() => onOpenGit(id, ws.name)}>Open Git</button><button type="button" onClick={() => onOpenTree(id, ws.name)}>Open files</button></div>
          </div> : <p className="wo-empty">No Git repository here. <button type="button" onClick={() => onOpenTree(id, ws.name)}>Open files</button></p>}
        </Block>
        <WorkspaceAgents workspace={ws} rows={agentRows} shells={shells} missionsError={errors.missions} retry={retry} onOpenAgent={onOpenAgent} onOpenTerm={onOpenTerm} onNewAgent={onNewAgent} />
        <Block title="Missions" href={`#/missions?workspace=${encodeURIComponent(id)}`} pending={!data.missions && !errors.missions} error={errors.missions} retry={retry}>
          {missions.length ? <ul className="wo-list">{missions.slice(0, 3).map((m) => <li key={m.id}><a href={`#/mission/${encodeURIComponent(m.id)}`}><span className="wo-mission"><strong>{m.title}</strong>{m.nextAction ? <span>{m.nextAction}</span> : null}</span><small>{m.state === "in-review" ? "Review" : m.state === "blocked" ? "Blocked" : m.assignment?.agentName || m.state}</small></a></li>)}</ul>
            : <p className="wo-empty">No active missions. <a href={`#/missions?workspace=${encodeURIComponent(id)}`}>Open Missions</a></p>}
        </Block>
      </div>
      <section className="wo-activity" aria-label="Recent changes">
        <div className="wo-block-head"><h3>{visitWithinRetention ? "Since your last visit" : "Recent changes"}</h3><span>Last 7 days</span></div>
        {(!data.activity && !errors.activity) || (data.activity && !activity.length && !data.graph && !errors.graph) ? <div className="wo-skeleton" aria-label="Loading recent changes"><span /><span /></div>
          : errors.activity ? <p className="wo-empty">Could not load recent changes. <button type="button" onClick={retry}>Retry</button></p>
            : activity.length ? <ul className="wo-list wo-timeline">{activity.map((item) => <li key={`${item.kind}:${item.id}`}>
              {activityHref(item) ? <a href={activityHref(item)}><span><small>{activityLabel(item)}</small><strong>{item.title}</strong></span><small>{relTime(new Date(item.at).toISOString())}</small></a>
                : <button type="button" onClick={() => item.kind === "commit" ? onOpenGit(id, ws.name) : onOpenAgent(item.entityId)}><span><small>{activityLabel(item)}</small><strong>{item.title}</strong></span><small>{relTime(new Date(item.at).toISOString())}</small></button>}
            </li>)}</ul>
              : <p className="wo-empty">{visitWithinRetention ? "No changes since your last visit." : "No recent changes here."} {git?.git ? <button type="button" onClick={() => onOpenGit(id, ws.name)}>Open Git</button> : <button type="button" onClick={() => onOpenTree(id, ws.name)}>Open files</button>}</p>}
      </section>
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
