import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { startPresence } from "../lib/device.js";
import { startReconnectWatch } from "../lib/reconnect.js";
import { normalizeManifests } from "../lib/appPrimitives.js";
import { needsYou } from "../lib/needsYou.js";
import { toast, toastError } from "../lib/toast.js";
import { closeTerm } from "../lib/terms.js";
import { mobileHash } from "../lib/mobileRoutes.js";
import { tabOf } from "../lib/mobileRoutes.js";
import Reconnect from "../components/Reconnect.jsx";
import ShareDrawer from "../components/ShareDrawer.jsx";
import Toasts from "../components/Toasts.jsx";
import ConfirmDialog from "../components/ConfirmDialog.jsx";
import TabBar from "./components/TabBar.jsx";
import CreateSheet from "./components/CreateSheet.jsx";
import { agentState } from "./components/StateChip.jsx";
import Now from "./screens/Now.jsx";
import Inbox from "./screens/Inbox.jsx";
import Agents from "./screens/Agents.jsx";
import Agent from "./screens/Agent.jsx";
import More from "./screens/More.jsx";
import { useHashRoute, goTab, push, goBack } from "./hooks/useHashRoute.js";
import { useFleet, flatAgents, findAgent } from "./hooks/useFleet.js";
import { usePoll } from "./hooks/usePoll.js";
import { IconQR } from "../components/Icons.jsx";
import "./mobile.css";

const LAST_AGENT_KEY = "picode-mobile-last-agent";

// The phone shell (ADR-0044): a supervision console, not the desktop
// shrunk. Now (decisions, running, today, results) · Inbox · Agents ·
// More, plus the pushed agent screen. One fleet poll feeds every screen;
// only the agent screen opens a socket.
export default function MobileApp() {
  const route = useHashRoute();
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [catalog, setCatalog] = useState(null);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [apps, setApps] = useState([]);
  const [inbox, setInbox] = useState([]);
  const [results, setResults] = useState([]);
  const [stats, setStats] = useState(null);
  const [tuiWorking, setTuiWorking] = useState([]);
  const [reconnect, setReconnect] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  const [create, setCreate] = useState(null); // { kind, workspace } | null
  const [busyId, setBusyId] = useState("");
  const [lastAgentId, setLastAgentId] = useState(() => { try { return localStorage.getItem(LAST_AGENT_KEY) || ""; } catch { return ""; } });

  const onNowOrAgents = route.screen === "now" || route.screen === "agents" || route.screen === "agent";
  const fleet = useFleet(onNowOrAgents ? 5000 : 15000);
  const { workspaces, freeAgents, loaded, reload } = fleet;

  useEffect(() => { applyTheme(themeMode); }, [themeMode]);
  useEffect(() => startPresence(), []);
  useEffect(() => startReconnectWatch({ onState: (s) => { if (s === "down") setReconnect(true); } }), []);

  async function loadCatalog() {
    try { setCatalog(await api("/api/catalog")); } catch { /* pi missing */ }
  }
  useEffect(() => {
    (async () => {
      try {
        const [sys, ver] = await Promise.all([api("/api/system"), api("/api/version")]);
        setSystem(sys);
        setVersion(ver.version || "");
      } catch { /* offline */ }
      await loadCatalog();
    })();
  }, []);

  // Inbox (blocking items for Now, results for the feed) and app badges.
  usePoll(async () => {
    const [blocking, all, appList] = await Promise.all([
      api("/api/inbox?blocking=1").catch(() => null),
      api("/api/inbox").catch(() => null),
      api("/api/apps").catch(() => null),
    ]);
    if (blocking) setInbox(blocking.items || []);
    if (all) setResults((all.items || []).filter((it) => it.kind === "result").slice(0, 5));
    if (appList) setApps(normalizeManifests(appList));
  }, 15000);

  // Today's headline for the Now screen.
  usePoll(async () => { setStats(await api("/api/sessions/stats?range=today")); }, 60000, route.screen === "now");

  // Interactive (tmux) agents have no event channel: poll pi's Working state.
  const interactiveIds = useMemo(() => flatAgents(workspaces, freeAgents).filter((x) => x.agent.mode === "interactive").map((x) => x.agent.id), [workspaces, freeAgents]);
  usePoll(async () => {
    if (!interactiveIds.length) { setTuiWorking([]); return; }
    const d = await api("/api/tui-working?ids=" + encodeURIComponent(interactiveIds.join(",")));
    setTuiWorking(d.working || []);
  }, 3000, interactiveIds.length > 0);

  const entries = useMemo(() => needsYou({ workspaces, freeAgents, inbox }), [workspaces, freeAgents, inbox]);
  const running = useMemo(() => flatAgents(workspaces, freeAgents).filter((x) => agentState(x.agent, tuiWorking) !== "stopped"), [workspaces, freeAgents, tuiWorking]);
  const fleetTotal = flatAgents(workspaces, freeAgents).length;
  const inboxApp = apps.find((a) => a.id === "inbox");
  const badges = { now: entries.length, inbox: inboxApp && inboxApp.badge ? inboxApp.badge.count || 0 : 0 };

  const current = route.screen === "agent" ? findAgent(workspaces, freeAgents, route.id) : null;
  const last = findAgent(workspaces, freeAgents, lastAgentId) || (running[0] || null);

  useEffect(() => {
    if (route.screen !== "agent" || !route.id) return;
    setLastAgentId(route.id);
    try { localStorage.setItem(LAST_AGENT_KEY, route.id); } catch { /* per-viewer nicety */ }
  }, [route.screen, route.id]);

  function openAgent(id) {
    if (!id) { goTab("agents"); return; }
    push(mobileHash("agent", id));
  }

  async function withBusy(agent, fn) {
    setBusyId(agent.id);
    try { await fn(); await reload(); } catch (e) { toastError(e); } finally { setBusyId(""); }
  }

  function startAgent(agent, workspace) {
    return withBusy(agent, async () => {
      await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
    });
  }

  function stopAgent(agent, workspace) {
    return withBusy(agent, async () => {
      if (agent.mode === "interactive") {
        if (workspace) await api("/api/workspaces/" + workspace.id + "/close", { method: "POST" });
        else await api("/api/agents/" + agent.id + "/close", { method: "POST" });
        closeTerm(agent.id);
      } else {
        await api("/api/agents/" + agent.id + "/managed/stop", { method: "POST" });
      }
    });
  }

  async function answerAsk(entry, body) {
    try {
      await api("/api/agents/" + entry.agentId + "/ui", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: entry.dialogId, cancelled: body.cancelled, value: body.value, confirmed: body.confirmed }),
      });
      await reload();
    } catch (e) { toastError(e); await reload(); }
  }

  async function respondInbox(entry, verb, text) {
    try {
      await api("/api/inbox/" + encodeURIComponent(entry.itemId) + "/respond", {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ verb, text }),
      });
      setInbox((cur) => cur.filter((it) => it.id !== entry.itemId));
      toast.ok(verb === "ignore" ? "Ignored." : "Sent.");
    } catch (e) { toastError(e); }
  }

  function onCreated(res) {
    setCreate(null);
    reload().then(() => {
      if (res && res.kind !== "workspace" && res.created && res.created.id) openAgent(res.created.id);
      else if (res && res.kind === "workspace") goTab("agents");
    });
  }

  const tab = tabOf(route);
  let body = null;
  if (route.screen === "agent") {
    body = (
      <Agent
        agent={current ? current.agent : null}
        workspace={current ? current.workspace : null}
        catalog={catalog}
        workingIds={tuiWorking}
        busy={!!current && busyId === current.agent.id}
        onBack={() => goBack(route)}
        onStart={startAgent}
        onStop={stopAgent}
      />
    );
  } else if (route.screen === "inbox") {
    body = <Inbox manifest={inboxApp} itemId={route.id} />;
  } else if (route.screen === "agents") {
    body = (
      <Agents loaded={loaded} workspaces={workspaces} freeAgents={freeAgents} workingIds={tuiWorking} busyId={busyId}
        onOpen={(a) => openAgent(a.id)} onStart={startAgent} onStop={stopAgent}
        onCreate={(kind, ws) => setCreate({ kind, workspace: ws || (kind === "agent" ? (workspaces[0] || null) : null) })} />
    );
  } else if (route.screen === "more") {
    body = (
      <More section={route.section} catalog={catalog} system={system} version={version} themeMode={themeMode}
        onTheme={(m) => { persistTheme(m); setThemeMode(m); }} last={last} onRefreshCatalog={loadCatalog}
        onShare={() => setShareOpen(true)} onBack={() => goBack(route)} />
    );
  } else {
    body = (
      <Now loaded={loaded} entries={entries} running={running} workingIds={tuiWorking} stats={stats} results={results}
        fleetTotal={fleetTotal} onAnswer={answerAsk} onRespond={respondInbox}
        onOpenAgent={openAgent} onOpenInbox={(id) => push(mobileHash("inbox", id))}
        onCreate={(kind) => setCreate({ kind, workspace: null })} />
    );
  }

  return (
    <div id="m-app" data-screen={route.screen}>
      {route.screen === "agent" || (route.screen === "more" && route.section) ? null : (
        <header className="m-top">
          <span className="m-brand">PiCode</span>
          <button type="button" className="m-icon" onClick={() => setShareOpen(true)} aria-label="Open on another phone" title="Open on another phone"><IconQR size={18} /></button>
        </header>
      )}
      <div className="m-body">{body}</div>
      <TabBar active={tab} badges={badges} />
      <CreateSheet open={!!create} kind={create ? create.kind : "workspace"} workspace={create ? create.workspace : null} catalog={catalog}
        onClose={() => setCreate(null)} onCreated={onCreated} />
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
      <Toasts />
      {reconnect ? <Reconnect onReload={() => location.reload()} /> : null}
      <ConfirmDialog />
    </div>
  );
}
