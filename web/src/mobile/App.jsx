import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "../lib/api.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { startPresence } from "../lib/device.js";
import { startReconnectWatch } from "../lib/reconnect.js";
import { normalizeManifests } from "../lib/appPrimitives.js";
import { needsYou } from "../lib/needsYou.js";
import { toast, toastError } from "../lib/toast.js";
import { closeTerm } from "../lib/terms.js";
import { mobileHash, tabOf, readWorkSection, writeWorkSection } from "../lib/mobileRoutes.js";
import { askConfirm } from "../lib/confirm.js";
import Reconnect from "../components/Reconnect.jsx";
import ShareDrawer from "../components/ShareDrawer.jsx";
import Toasts from "../components/Toasts.jsx";
import ConfirmDialog from "../components/ConfirmDialog.jsx";
import TabBar from "./components/TabBar.jsx";
import CreateSheet from "./components/CreateSheet.jsx";
import { agentState } from "./components/StateChip.jsx";
import Now from "./screens/Now.jsx";
import Inbox from "./screens/Inbox.jsx";
import Work from "./screens/Work.jsx";
import Agent from "./screens/Agent.jsx";
import TerminalScreen from "./screens/Terminal.jsx";
import More from "./screens/More.jsx";
import { useHashRoute, goTab, push, goBack } from "./hooks/useHashRoute.js";
import { useFleet, flatAgents, findAgent } from "./hooks/useFleet.js";
import { usePoll } from "./hooks/usePoll.js";
import "./mobile.css";

const LAST_AGENT_KEY = "picode-mobile-last-agent";

// The phone shell (ADR-0044): a supervision console, not the desktop
// shrunk. Now (decisions, running, today, results) · Inbox · Work
// (workspaces / free agents / terminals, the desktop rail's three views)
// · More, plus the pushed agent and terminal screens. No header: the
// tab bar is the chrome. One fleet poll feeds every screen; only the
// agent screen opens an agent socket.
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

  const onNowOrWork = route.screen === "now" || route.screen === "work" || route.screen === "agent" || route.screen === "term";
  const fleet = useFleet(onNowOrWork ? 5000 : 15000);
  const { workspaces, freeAgents, terminals, loaded, reload } = fleet;
  const [workSection, setWorkSection] = useState(readWorkSection);

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
  const currentTerm = route.screen === "term" ? terminals.find((t) => t.id === route.id) || null : null;
  const liveTerms = terminals.filter((t) => t.running);
  const section = route.screen === "work" ? (route.section || workSection) : workSection;
  useEffect(() => {
    if (route.screen === "work" && route.section && route.section !== workSection) {
      setWorkSection(route.section);
      writeWorkSection(route.section);
    }
  }, [route.screen, route.section, workSection]);
  const last = findAgent(workspaces, freeAgents, lastAgentId) || (running[0] || null);

  useEffect(() => {
    if (route.screen !== "agent" || !route.id) return;
    setLastAgentId(route.id);
    try { localStorage.setItem(LAST_AGENT_KEY, route.id); } catch { /* per-viewer nicety */ }
  }, [route.screen, route.id]);

  function openAgent(id) {
    if (!id) { goTab("work"); return; }
    push(mobileHash("agent", id));
  }
  function openTerm(id) {
    if (id) push(mobileHash("term", id));
  }
  function setSection(sec) {
    setWorkSection(sec);
    writeWorkSection(sec);
    location.replace(mobileHash("work", sec));
  }

  async function newTerminal(workspace) {
    try {
      const body = workspace ? JSON.stringify({ workspaceId: workspace.id }) : "{}";
      const page = await api("/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body });
      await reload();
      openTerm(page.id);
    } catch (e) { toastError(e); }
  }

  async function removeTerminal(t) {
    const ok = await askConfirm({ title: "Remove terminal?", message: "This stops the tmux session.", confirmLabel: "Remove", danger: true });
    if (!ok) return;
    setBusyId(t.id);
    try {
      await api("/api/terminals/" + encodeURIComponent(t.id), { method: "DELETE" });
      closeTerm("sh:" + t.id);
      await reload();
      if (route.screen === "term" && route.id === t.id) goBack(route);
    } catch (e) { toastError(e); } finally { setBusyId(""); }
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
      else if (res && res.kind === "workspace") setSection("workspaces");
    });
  }

  const tab = tabOf(route);
  let body = null;
  if (route.screen === "term") {
    body = <TerminalScreen term={currentTerm} onBack={() => goBack(route)} onRemove={removeTerminal} busy={!!currentTerm && busyId === currentTerm.id} />;
  } else if (route.screen === "agent") {
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
  } else if (route.screen === "work") {
    body = (
      <Work section={section} onSection={setSection} loaded={loaded} workspaces={workspaces} freeAgents={freeAgents} terminals={terminals}
        workingIds={tuiWorking} busyId={busyId}
        onOpenAgent={(a) => openAgent(a.id)} onOpenTerm={(t) => openTerm(t.id)} onStart={startAgent} onStop={stopAgent} onRemoveTerm={removeTerminal}
        onCreate={(kind, ws) => setCreate({ kind, workspace: ws || (kind === "agent" ? (workspaces[0] || null) : null) })} onNewTerm={newTerminal} />
    );
  } else if (route.screen === "more") {
    body = (
      <More section={route.section} catalog={catalog} system={system} version={version} themeMode={themeMode}
        onTheme={(m) => { persistTheme(m); setThemeMode(m); }} last={last} onRefreshCatalog={loadCatalog}
        onShare={() => setShareOpen(true)} onBack={() => goBack(route)} />
    );
  } else {
    body = (
      <Now loaded={loaded} entries={entries} running={running} liveTerms={liveTerms} workingIds={tuiWorking} stats={stats} results={results}
        fleetTotal={fleetTotal + terminals.length} onAnswer={answerAsk} onRespond={respondInbox}
        onOpenAgent={openAgent} onOpenTerm={openTerm} onOpenInbox={(id) => push(mobileHash("inbox", id))}
        onCreate={(kind) => setCreate({ kind, workspace: null })} />
    );
  }

  return (
    <div id="m-app" data-screen={route.screen}>
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
