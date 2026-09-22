import { agentTerminalKeys, agentTerminalActionTarget } from "@picode/shared/domain/agentTerminal.js";
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api } from "@picode/shared/client/api.js";
import { applyTheme, persistTheme, readThemeMode, resolvedTheme } from "@picode/shared/domain/theme.js";
import { startPresence } from "@picode/shared/client/device.js";
import { startFeed, subscribeFeed } from "@picode/shared/client/feed.js";
import { applyTui, touches } from "@picode/shared/domain/feedReducers.js";
import { agentIsPi } from "@picode/shared/domain/managedPrincipal.js";
import { agentHandoffTerm } from "@picode/shared/domain/agentRowMenu.js";
import { applyChecklists, indexChecklists } from "@picode/shared/domain/checklist.js";
import { startReconnectWatch } from "@picode/shared/client/reconnect.js";
import { normalizeManifests, nativeApp } from "@picode/shared/contracts/appPrimitives.js";
import { needsYou } from "./lib/needsYou.js";
import { asksOnSurface, needsYouPlan } from "@picode/shared/domain/notice.js";
import { workspaceHash } from "./lib/routes.js";
import { dismissNotice, notify, toast, toastError } from "./lib/toast.js";
import { watchReminders } from "@picode/shared/client/reminders.js";
import { closeTerm } from "./lib/terms.js";
import { mobileHash, toolHash, tabOf, readWorkSection, writeWorkSection } from "./lib/mobileRoutes.js";
import { agentOwnerWs, termOwnerWs } from "./lib/workBack.js";
import { askConfirm } from "./lib/confirm.js";
import { askPrompt } from "./lib/prompt.js";
import { sessionFromTerminal, terminalHandoffSourceCli } from "@picode/shared/domain/sessionHandoff.js";
import Reconnect from "./components/Reconnect.jsx";
import ShareDrawer, { OPEN_EVENT } from "./components/ShareDrawer.jsx";
import Toasts from "./components/Toasts.jsx";
import ConfirmDialog from "./components/ConfirmDialog.jsx";
import PromptDialog from "./components/PromptDialog.jsx";
import SessionHandoffDialog from "./components/SessionHandoffDialog.jsx";
import WhatsNew from "./components/WhatsNew.jsx";
import RELEASE_NOTES from "@picode/shared/data/whats-new.json";
import TabBar from "./components/TabBar.jsx";
import CreateSheet from "./components/CreateSheet.jsx";
import NewCliPrincipal from "./components/NewCliPrincipal.jsx";
import { agentState } from "./components/StateChip.jsx";
import Now from "./screens/Now.jsx";
const Inbox = lazy(() => import("./screens/Inbox.jsx"));
const InboxItem = lazy(() => import("./screens/Inbox.jsx").then(module => ({ default: module.InboxItem })));
const PinScreen = lazy(() => import("./screens/Pin.jsx"));
const PinEdit = lazy(() => import("./screens/PinEdit.jsx"));
const SnipScreen = lazy(() => import("./screens/Snippet.jsx"));
const SnipEdit = lazy(() => import("./screens/SnippetEdit.jsx"));
import Work from "./screens/Work.jsx";
const Agent = lazy(() => import("./screens/Agent.jsx"));
const TerminalScreen = lazy(() => import("./screens/Terminal.jsx"));
const Files = lazy(() => import("./screens/Files.jsx"));
const Git = lazy(() => import("./screens/Git.jsx"));
const Inspector = lazy(() => import("./screens/Inspector.jsx"));
const InspectorDrawer = lazy(() => import("./components/InspectorDrawer.jsx"));
const More = lazy(() => import("./screens/More.jsx"));
const AppSurface = lazy(() => import("./components/AppSurface.jsx"));
import { useHashRoute, goTab, push, goBack } from "./hooks/useHashRoute.js";
import { useFleet, flatAgents, findAgent } from "./hooks/useFleet.js";
import { usePoll } from "./hooks/usePoll.js";
import { useVisualViewport } from "./hooks/useVisualViewport.js";
import { hasUnseenRelease, shouldAutoOpen, readSeenVersion, writeSeenVersion } from "./lib/whatsNew.js";
import ScreenBoundary, { ScreenError, ScreenLoading } from "./components/ScreenBoundary.jsx";
import NativeAppNotice from "./components/NativeAppNotice.jsx";
import { saveAgentConfig } from "./lib/agentConfig.js";
import { fleetRouteReady } from "./lib/fleetReads.js";
import { deliverGitCommand } from "./lib/gitDelivery.js";
import "./mobile.css";

const LAST_AGENT_KEY = "picode-mobile-last-agent";

// The phone shell (ADR-0044/0095): focused mobile workflows. Now (decisions, today, results) · Inbox · Work
// (workspaces / free agents / terminals, the desktop rail's three views)
// · More, plus the pushed agent and terminal screens. No header: the
// tab bar is the chrome. One fleet poll feeds every screen; only the
// agent screen opens an agent socket.
export default function MobileApp() {
  useVisualViewport();
  const route = useHashRoute();
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [catalog, setCatalog] = useState(null);
  const [clis, setClis] = useState([]);
  const [termHandoff, setTermHandoff] = useState(null);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [semver, setSemver] = useState("");
  const [releaseBuild, setReleaseBuild] = useState(false);
  const [whatsNewOpen, setWhatsNewOpen] = useState(false);
  const [whatsNewMode, setWhatsNewMode] = useState("manual");
  const [whatsNewSeen, setWhatsNewSeen] = useState(readSeenVersion);
  const [apps, setApps] = useState([]);
  const [inbox, setInbox] = useState([]);
  const [results, setResults] = useState([]);
  const [stats, setStats] = useState(null);
  const [attentionReady, setAttentionReady] = useState(false);
  const [attentionError, setAttentionError] = useState("");
  const [tuiWorking, setTuiWorking] = useState([]);
  const [checklists, setChecklists] = useState({});
  const [reconnect, setReconnect] = useState(false);
  // The agent screen's Inspector drawer: opens over the conversation, and
  // any navigation away (a file handed to the Files tool, a tab switch)
  // closes it — the drawer belongs to one agent section.
  const [inspDrawer, setInspDrawer] = useState(null);
  const [shareOpen, setShareOpen] = useState(false);
  useEffect(() => {
    const on = () => setShareOpen(true);
    window.addEventListener(OPEN_EVENT, on);
    return () => window.removeEventListener(OPEN_EVENT, on);
  }, []);
  const [create, setCreate] = useState(null); // { kind, workspace } | null
  const [cliPrincipalWs, setCliPrincipalWs] = useState(null);
  // Recovery links inside the sheet navigate to a full screen. Close it
  // only after navigation succeeds, so route guards can still cancel it.
  useEffect(() => { setCreate(null); }, [route]);
  const [busyId, setBusyId] = useState("");
  const [lastAgentId, setLastAgentId] = useState(() => { try { return localStorage.getItem(LAST_AGENT_KEY) || ""; } catch { return ""; } });

  const onNowOrWork = route.screen === "now" || route.screen === "work" || route.screen === "agent" || route.screen === "term";
  const fleet = useFleet(onNowOrWork ? 5000 : 15000);
  const { workspaces, freeAgents, terminals, loaded, error: fleetError, reload } = fleet;
  const [workSection, setWorkSection] = useState(readWorkSection);

  useEffect(() => {
    applyTheme(themeMode);
    // iOS 26 frosts a translucent status bar over the header. Opaque
    // (black / default) keeps the title at --text-primary.
    const bar = document.querySelector('meta[name="apple-mobile-web-app-status-bar-style"]');
    if (bar) bar.setAttribute("content", resolvedTheme(themeMode) === "light" ? "default" : "black");
  }, [themeMode]);
  useEffect(() => startPresence(), []);
  useEffect(() => startFeed(), []);
  // Zoom lock (owner): a supervision console is read at one scale. The
  // meta is rewritten here, not in index.html, so the desktop shell keeps
  // the browser's default; iOS ignores user-scalable, so pinch is also
  // cancelled at the gesture level and double-tap by touch-action in CSS.
  useEffect(() => {
    const meta = document.querySelector('meta[name="viewport"]');
    const prev = meta ? meta.getAttribute("content") : "";
    if (meta) meta.setAttribute("content", "width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover, interactive-widget=resizes-content");
    const stop = (e) => e.preventDefault();
    document.addEventListener("gesturestart", stop, { passive: false });
    document.addEventListener("gesturechange", stop, { passive: false });
    return () => {
      if (meta && prev) meta.setAttribute("content", prev);
      document.removeEventListener("gesturestart", stop);
      document.removeEventListener("gesturechange", stop);
    };
  }, []);
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
        setSemver(ver.semver || ver.version || "");
        setReleaseBuild(!!ver.release);
      } catch { /* offline */ }
      await loadCatalog();
      try {
        const d = await api("/api/clis");
        setClis(d.clis || []);
      } catch { /* Agent CLIs catalog is optional for Work */ }
    })();
  }, []);

  // Inbox (blocking items for Now, results for the feed) and app badges.
  const loadInbox = useCallback(async () => {
    const [blocking, all, appList] = await Promise.all([
      api("/api/inbox?blocking=1").catch(() => null),
      api("/api/inbox").catch(() => null),
      api("/api/apps").catch(() => null),
    ]);
    if (blocking) setInbox(blocking.items || []);
    if (all) setResults((all.items || []).filter((it) => it.kind === "result").slice(0, 5));
    if (appList) setApps(normalizeManifests(appList));
    if (blocking && all) setAttentionReady(true);
    setAttentionError(blocking && all && appList ? "" : "Couldn’t refresh activity. Try again for the latest updates.");
  }, []);
  usePoll(loadInbox, 15000);
  useEffect(() => subscribeFeed((ev) => { if (touches(ev, ["inbox", "docker"])) loadInbox().catch(() => {}); }), [loadInbox]);

  // Today's headline for the Now screen.
  usePoll(async () => { setStats(await api("/api/sessions/stats?range=today")); }, 60000, route.screen === "now");

  // Interactive (tmux) agents have no event channel: poll pi's Working state.
  const interactiveIds = useMemo(() => flatAgents(workspaces, freeAgents).filter((x) => x.agent.mode === "interactive").map((x) => x.agent.id), [workspaces, freeAgents]);
  usePoll(async () => {
    if (!interactiveIds.length) { setTuiWorking([]); return; }
    const d = await api("/api/tui-working?ids=" + encodeURIComponent(interactiveIds.join(",")));
    setTuiWorking(d.working || []);
  }, 3000, interactiveIds.length > 0);
  useEffect(() => subscribeFeed((ev) => { if (ev.type === "agent.tui") setTuiWorking((cur) => applyTui(cur, ev)); }), []);
  useEffect(() => {
    // Internal checklists (ADR-0055): one fetch per (re)open, then the feed.
    let stop = false;
    const load = () => api("/api/checklists").then((d) => { if (!stop) setChecklists(indexChecklists(d.checklists)); }).catch(() => {});
    load();
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") load();
      else if (ev.type === "agent.checklist" || ev.type === "agent.deleted") setChecklists((cur) => applyChecklists(cur, ev));
    });
    return () => { stop = true; unsub(); };
  }, []);

  const entries = useMemo(() => needsYou({ workspaces, freeAgents, inbox }), [workspaces, freeAgents, inbox]);
  const running = useMemo(() => flatAgents(workspaces, freeAgents).filter((x) => agentState(x.agent, tuiWorking) !== "stopped"), [workspaces, freeAgents, tuiWorking]);
  // Needs-you as a card, for the screens that are not the queue. On the
  // Now screen the queue itself is the answer, so an ask that arrives
  // there is recorded as seen without a toast on top of its own card.
  const announcedAsks = useRef(null);
  useEffect(() => {
    // "Nothing is waiting" and "nothing has been read yet" are different
    // answers: seeding on the empty first render made the first real read
    // look like an arrival, so a reload announced the whole backlog.
    if (!loaded) return;
    const plan = needsYouPlan(entries, announcedAsks.current, workspaceHash);
    announcedAsks.current = plan.keys;
    for (const key of plan.gone) dismissNotice(key);
    // The Now screen IS the queue, and an agent screen IS the answer:
    // on either, the standing cards are what the user already sees.
    const here = new Set(route.screen === "now"
      ? plan.keys
      : asksOnSurface(entries, route.screen === "agent" ? workspaceHash(route.id) : "", workspaceHash));
    for (const key of here) dismissNotice(key);
    for (const n of plan.fresh) if (!here.has(n.key)) notify(n);
  }, [entries, route.screen, route.id, loaded]);
  // Pin reminders (ADR-0100): the same sticky cards as the desk, mirroring
  // the open reminder items; Open lands on the read-only pin screen.
  useEffect(() => watchReminders({
    notify, dismiss: dismissNotice,
    pinHash: (id) => "#/pins/" + encodeURIComponent(id),
    inboxHash: "#/inbox",
  }), []);

  const fleetTotal = flatAgents(workspaces, freeAgents).length;
  const inboxApp = apps.find((a) => a.id === "inbox");
  const badges = { now: entries.length, inbox: inboxApp && inboxApp.badge ? inboxApp.badge.count || 0 : 0 };
  const whatsNewCurrent = semver || version;
  const whatsNewUnread = hasUnseenRelease({ release: releaseBuild, current: whatsNewCurrent, seen: whatsNewSeen, entries: RELEASE_NOTES });
  // A fresh install sees What's New too (ADR-0063, amendment 2026-09-11). The
  // product-state gate that used to stand here is gone; `blocked` below is
  // every other defer condition and all of them stay.
  useEffect(() => {
    if (!loaded || !releaseBuild || !whatsNewCurrent || whatsNewOpen) return;
    const blocked = reconnect || !!create || shareOpen || entries.length > 0;
    if (shouldAutoOpen({ release: releaseBuild, current: whatsNewCurrent, seen: whatsNewSeen, entries: RELEASE_NOTES, blocked })) {
      setWhatsNewMode("auto");
      setWhatsNewOpen(true);
    }
  }, [loaded, releaseBuild, whatsNewCurrent, whatsNewOpen, reconnect, create, shareOpen, entries.length, whatsNewSeen]);

  function openWhatsNew() { setWhatsNewMode("manual"); setWhatsNewOpen(true); }
  function closeWhatsNew() {
    if (releaseBuild && whatsNewCurrent) {
      writeSeenVersion(whatsNewCurrent);
      setWhatsNewSeen(whatsNewCurrent);
    }
    setWhatsNewOpen(false);
  }

  const current = route.screen === "agent" ? findAgent(workspaces, freeAgents, route.id) : null;
  const currentTerm = route.screen === "term" ? terminals.find((t) => t.id === route.id) || null : null;
  const terminalAgent = route.screen === "term" ? flatAgents(workspaces, freeAgents).find(({ agent }) => agent.terminalId === route.id) : null;
  const section = route.screen === "work" ? (route.section || workSection) : workSection;
  useEffect(() => {
    if (route.screen === "work" && route.section && route.section !== workSection) {
      setWorkSection(route.section);
      writeWorkSection(route.section);
    }
  }, [route.screen, route.section, workSection]);
  const last = findAgent(workspaces, freeAgents, lastAgentId) || (running[0] || null);

  // Agent CLI links and agent links are entrances to the same owning view.
  useEffect(() => {
    if (route.screen !== "term") return;
    const owner = flatAgents(workspaces, freeAgents).find(({ agent }) => agent.terminalId === route.id);
    if (owner) location.replace(mobileHash("agent", owner.agent.id, "", "terminal"));
  }, [route.screen, route.id, workspaces, freeAgents]);

  useEffect(() => {
    if (route.screen !== "agent" || !route.id) return;
    setLastAgentId(route.id);
    try { localStorage.setItem(LAST_AGENT_KEY, route.id); } catch { /* per-viewer nicety */ }
  }, [route.screen, route.id]);

  useEffect(() => {
    if (route.screen !== "agent" && route.screen !== "term") setInspDrawer(null);
  }, [route.screen]);
  function openAgent(id, view = "") {
    if (!id) { goTab("work"); return; }
    const target = findAgent(workspaces, freeAgents, id);
    const initial = view || (target && target.agent && target.agent.terminalId ? "terminal" : "");
    push(mobileHash("agent", id, "", initial));
  }
  function openTerm(id) {
    const owner = flatAgents(workspaces, freeAgents).find(({ agent }) => agent.terminalId === id);
    if (owner) { openAgent(owner.agent.id, "terminal"); return; }
    if (id) push(mobileHash("term", id));
  }
  async function onAppGoto(goto) {
    const value = String(goto || "");
    if (value.startsWith("agent:")) openAgent(value.slice("agent:".length));
    if (value.startsWith("pin:")) push("#/pins/" + encodeURIComponent(value.slice("pin:".length)));
    if (value.startsWith("term:")) openTerm(value.slice("term:".length));
  }
  function openInspector(owner, root = "") {
    if (owner && owner.id) push(toolHash("inspector", owner, { root }));
  }
  function openFiles(owner, options = {}) { push(toolHash("files", owner, options)); }
  function openGit(owner, root = "") { push(toolHash("git", owner, { root })); }
  async function prepareGit(owner, root, command, { run = false } = {}) {
    const context = owner.kind === "agent" ? findAgent(workspaces, freeAgents, owner.id) : null;
    const workspaceId = owner.kind === "workspace" ? owner.id : context?.workspace?.id || "";
    const result = await deliverGitCommand({ owner, root, command, run, terminals, workspaceId, api });
    await reload({ force: true }).catch(() => {});
    openTerm(result.id);
    toast.info(result.ran ? "Command started in the terminal." : (result.reason ? result.reason + " " : "") + "Command prepared. Press Enter in the terminal to run it.");
    return result;
  }
  async function askGit(who, text, root, action) {
    // A pi running as an Agent CLI terminal is asked through its receiver
    // (ADR-0089 amendment); an agent through its own channel (ADR-0078).
    const base = who?.kind === "terminal" ? "/api/terminals/" : "/api/agents/";
    const result = await api(base + encodeURIComponent(who.id) + "/ask", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ text, root }) });
    toast.info(result.mode === "interactive" ? "Sent to " + who.name + " in its terminal." : result.busy ? "Queued for " + who.name + " after the current turn." : "Sent to " + who.name + ".");
    return result;
  }
  // Pull-to-refresh: everything the visible screens read, at once.
  async function refreshAll() {
    await Promise.all([
      reload({ force: true }).catch(() => {}),
      loadInbox(),
      route.screen === "now" ? api("/api/sessions/stats?range=today").then(setStats).catch(() => {}) : Promise.resolve(),
    ]);
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
      await reload({ force: true });
      openTerm(page.id);
    } catch (e) { toastError(e); }
  }

  async function removeTerminal(t) {
    const bound = flatAgents(workspaces, freeAgents).some(({ agent }) => agent.terminalId === t.id);
    const ok = await askConfirm({ title: bound ? "Remove agent and terminal?" : "Remove terminal?", message: bound ? "This stops the terminal and removes its agent." : "This stops the terminal session.", confirmLabel: "Remove", danger: true });
    if (!ok) return;
    setBusyId(t.id);
    try {
      await api("/api/terminals/" + encodeURIComponent(t.id), { method: "DELETE" });
      closeTerm("sh:" + t.id);
      if (current?.agent.terminalId === t.id) agentTerminalKeys(current.agent).forEach(closeTerm);
      await reload({ force: true });
      if (route.screen === "term" && route.id === t.id) goBack(route, termOwnerWs(t));
      if (route.screen === "agent" && current?.agent.terminalId === t.id) goBack(route, agentOwnerWs(current));
    } catch (e) { toastError(e); } finally { setBusyId(""); }
  }

  async function renameTerminal(t) {
    if (!t) return;
    const name = await askPrompt({ title: "Rename terminal", defaultValue: t.name || "Terminal", confirmLabel: "Save" });
    if (!name) return;
    setBusyId(t.id);
    try {
      await api("/api/terminals/" + encodeURIComponent(t.id), {
        method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name }),
      });
      await reload({ force: true });
    } catch (e) { toastError(e); } finally { setBusyId(""); }
  }

  async function launchTerminalAction(t, op) {
    if (!t || !op) return;
    const destructive = t.running && op !== "start";
    if (destructive && !(await askConfirm({
      title: `${op === "stop" ? "Stop" : "Restart"} ${t.name || "terminal"}?`,
      message: op === "restart"
        ? (t.lastSession
          ? "This ends the processes running in this terminal, then reopens the same conversation."
          : "This ends the processes running in this terminal.")
        : "This ends the processes running in this terminal.",
      confirmLabel: op === "stop" ? "Stop terminal" : "Restart terminal",
      danger: true,
    }))) return;
    setBusyId(t.id);
    try {
      await api(`/api/terminals/${encodeURIComponent(t.id)}/launch/${op}`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ confirm: destructive }),
      });
      await reload({ force: true });
      if (op === "start") openTerm(t.id);
      else toast.ok(op === "stop" ? "Terminal stopped." : "Terminal restarted.");
    } catch (e) { toastError(e); } finally { setBusyId(""); }
  }

  function openTermHandoff(term, target) {
    const session = sessionFromTerminal(term);
    const sourceCli = terminalHandoffSourceCli(term);
    if (!session || !sourceCli || !target || (target.landing !== "agent" && !target.installed)) return;
    const source = clis.find((c) => c.id === sourceCli);
    setTermHandoff({
      session,
      sourceCli,
      sourceName: (source && source.name) || sourceCli,
      target,
    });
  }

  function onTermHandoffDone(res, target) {
    setTermHandoff(null);
    const next = res && res.terminal;
    const adopted = res && res.agent;
    if (next && next.launchError) {
      toastError(new Error(next.launchError));
    } else if (next && next.id) {
      toast.ok(target.name + " is opening with this conversation.");
      openTerm(next.id);
    } else if (adopted && adopted.id) {
      toast.ok(res.brief ? "Continued as a Pi agent: " + adopted.name + ". Its first message reads the brief." : "Continued as a Pi agent: " + adopted.name + ".");
      openAgent(adopted.id);
    } else {
      toast.ok("Handoff recorded.");
    }
  }

  function onTermAction(term, id, extra) {
    if (id === "rename") return renameTerminal(term);
    if (id === "remove") return removeTerminal(term);
    if (id === "handoff") return openTermHandoff(term, extra);
    if (id === "start" || id === "restart" || id === "stop") return launchTerminalAction(term, id);
  }

  async function onAgentAction(agent, id, extra) {
    if (id === "chat") return openAgent(agent.id, "chat");
    if (id === "term") return openAgent(agent.id, "terminal");
    if (id === "start") return startAgent(agent);
    if (id === "stop") return stopAgent(agent);
    if (id === "restart") return agentIsPi(agent)
      ? withBusy(agent, async () => {
          await api("/api/agents/" + agent.id + "/managed/stop", { method: "POST" });
          await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
        })
      : launchTerminalAction(agentTerm(agent), "restart");
    if (id === "rename") return renameAgent(agent);
    if (id === "remove") return removeAgent(agent);
    if (id === "handoff") {
      const term = terminals.find((t) => t.id === agent.terminalId) || null;
      return openTermHandoff(term || agentHandoffTerm(agent), extra);
    }
  }

  async function renameAgent(agent) {
    const name = await askPrompt({ title: "Rename agent", defaultValue: agent.name && agent.name !== "default" ? agent.name : "", confirmLabel: "Save" });
    if (!name) return;
    await withBusy(agent, async () => {
      await api("/api/agents/" + encodeURIComponent(agent.id), {
        method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name }),
      });
    });
  }

  async function removeAgent(agent) {
    const ok = await askConfirm({ title: "Remove " + (agent.name || "agent") + "?", message: "The agent and its terminal are removed. The project folder is not deleted.", confirmLabel: "Remove", danger: true });
    if (!ok) return;
    setBusyId(agent.id);
    try {
      await api("/api/agents/" + encodeURIComponent(agent.id), { method: "DELETE" });
      if (agent.terminalId) {
        // Deleting the agent only nulls its terminal_id: the bound
        // terminal — and the CLI process inside it — would survive as an
        // orphan Work card. The menu promises both go away.
        await api("/api/terminals/" + encodeURIComponent(agent.terminalId), { method: "DELETE" }).catch(() => {});
        closeTerm("sh:" + agent.terminalId);
      }
      await reload({ force: true });
    } catch (e) {
      // As with desktop: a 404 just means the agent is already gone.
      if (!e || e.status !== 404) toastError(e);
    } finally { setBusyId(""); }
  }

  async function withBusy(agent, fn) {
    setBusyId(agent.id);
    try { await fn(); await reload({ force: true }); } catch (e) { toastError(e); } finally { setBusyId(""); }
  }

  function agentTerm(agent) {
    return agentTerminalActionTarget(agent, terminals.find((t) => t.id === agent.terminalId));
  }

  function startAgent(agent, workspace) {
    // A CLI agent's process is its bound terminal (ADR-0160): start the CLI
    // launch, not the managed runtime — and open the TUI, which is the
    // conversation.
    if (!agentIsPi(agent) && agent.terminalId) return launchTerminalAction(agentTerm(agent), "start");
    return withBusy(agent, async () => {
      await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
    });
  }

  async function stopAgent(agent, workspace) {
    if (!agentIsPi(agent) && agent.terminalId) return launchTerminalAction(agentTerm(agent), "stop");
    if (agent.mode === "interactive" && !(await askConfirm({ title: "Stop agent?", message: "This ends the processes running in this terminal.", confirmLabel: "Stop", danger: true }))) return;
    return withBusy(agent, async () => {
      if (agent.mode === "interactive") {
        // Agent-scoped control matters in multi-agent workspaces; the
        // workspace endpoint only targets its default.
        await api("/api/agents/" + agent.id + "/close", { method: "POST" });
        agentTerminalKeys(agent).forEach(closeTerm);
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
      await reload({ force: true });
    } catch (e) { toastError(e); await reload({ force: true }); }
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
    reload({ force: true }).then(() => {
      if (res && res.kind !== "workspace" && res.created && res.created.id) openAgent(res.created.id);
      else if (res && res.kind === "workspace") setSection("workspaces");
    });
  }

  async function patchAgent(agent, cfg) {
    try { await saveAgentConfig(agent, cfg, api, closeTerm); }
    finally { await reload({ force: true }).catch(toastError); }
  }

  const tab = tabOf(route);
  // A pushed screen (it has the ← header) owns the whole height: the tab
  // bar goes away, Back is the way out.
  const pushed = route.screen === "app" || route.screen === "agent" || route.screen === "term" || ["inspector", "files", "git", "pin", "pinEdit"].includes(route.screen) || (route.screen === "more" && !!route.section) || (route.screen === "inbox" && !!route.id);
  // iOS standalone: the unreachable bottom strip continues the surface it
  // sits under — the tab bar's panel, or plain content when pushed.
  useEffect(() => { document.documentElement.dataset.pushed = pushed ? "1" : ""; }, [pushed]);
  const changeOwner = !["inspector", "files", "git"].includes(route.screen) ? null : route.section === "agent" ? findAgent(workspaces, freeAgents, route.id)
    : route.section === "term" ? { term: terminals.find((t) => t.id === route.id) }
    : { workspace: workspaces.find((w) => w.id === route.id) };
  const resourceFound = current || currentTerm || changeOwner?.agent || changeOwner?.term || changeOwner?.workspace;
  let body = null;
  if (!fleetRouteReady(route, fleet.known, !!resourceFound)) {
    body = fleetError ? <ScreenError message={fleetError} onRetry={reload} /> : <ScreenLoading />;
  } else if (route.screen === "app") {
    // A native app (ADR-0109) has no body on the phone: one line and Back,
    // not a primitives view that would only say it opens on the desktop.
    const manifest = apps.find((a) => a.id === route.id);
    body = nativeApp(manifest)
      ? <NativeAppNotice manifest={manifest} onBack={() => goBack(route)} />
      : <div className="m-screen m-app-screen"><AppSurface key={route.id} appId={route.id} initialPath={route.path || ""} manifest={manifest} hidden={false} onClose={() => goBack(route)} onGoto={onAppGoto} onPathChange={(path) => { location.hash = "#/app/" + encodeURIComponent(route.id) + (path ? "/" + path.split("/").map(encodeURIComponent).join("/") : ""); }} /></div>;
  } else if (route.screen === "files" || route.screen === "git") {
    const owner = { kind: route.section, id: route.id };
    const title = changeOwner?.agent?.name || changeOwner?.term?.name || changeOwner?.workspace?.name || "Project";
    body = route.screen === "files"
      ? <Files key={JSON.stringify([route.section, route.id, route.path, route.root, route.navigation])} owner={owner} title={title} root={route.root || ""} initialPath={route.path || ""} onBack={() => goBack(route)} onPathChange={(path, root) => { history.replaceState(history.state, "", toolHash("files", owner, { path, root })); }} onOpenGit={(target, root) => openGit(target || owner, root)} />
      : <Git key={JSON.stringify([route.section, route.id, route.root, route.commit, route.navigation])} owner={owner} title={title} root={route.root || ""} initialCommit={route.commit || ""} onBack={() => goBack(route)} onOpenFile={({ owner: target, path, root }) => openFiles(target || owner, { path, root })} onOpenTerminal={prepareGit} onAskAgent={askGit} onPickWorkspace={wsId => openGit({ kind: "workspace", id: wsId })} onOpen={(kind, id) => { location.hash = (kind === "term" ? "#/term/" : "#/agent/") + encodeURIComponent(id); }} workspaces={workspaces} freeAgents={freeAgents} terminals={terminals} />;
  } else if (route.screen === "inspector") {
    const owner = { kind: route.section, id: route.id };
    const title = changeOwner?.agent?.name || changeOwner?.term?.name || changeOwner?.workspace?.name || "Project";
    body = <Inspector key={JSON.stringify([route.section, route.id, route.root, route.view, route.navigation])} owner={owner} title={title} root={route.root || ""} initialView={route.view || ""} onBack={() => goBack(route)} onOpenFile={({ owner: target, path, root }) => openFiles(target || owner, { path, root })} onOpenTerminal={prepareGit} onAskAgent={askGit} workspaces={workspaces} freeAgents={freeAgents} terminals={terminals} />;
  } else if (route.screen === "term") {
    body = terminalAgent
      ? <div className="m-tool-state m-tool-loading" role="status" aria-busy="true"><p>Opening agent…</p><span className="gg-skel" /><span className="gg-skel" /></div>
      : <TerminalScreen key={route.id} term={currentTerm} onBack={() => goBack(route, termOwnerWs(currentTerm))} onRemove={removeTerminal} busy={!!currentTerm && busyId === currentTerm.id} onOpenFiles={openFiles} onOpenGit={openGit} onOpenInspector={() => setInspDrawer({ owner: { kind: "term", id: route.id }, root: "", title: currentTerm ? currentTerm.name : "Inspector" })} />;
  } else if (route.screen === "agent") {
    body = (
      <Agent
        agent={current ? current.agent : null}
        workspace={current ? current.workspace : null}
        catalog={catalog}
        workingIds={tuiWorking}
        busy={!!current && (busyId === current.agent.id || busyId === current.agent.terminalId)}
        terminal={current ? terminals.find((t) => t.id === current.agent.terminalId) || current.agent.terminal || null : null}
        initialView={route.view}
        onViewChange={(view) => { location.replace(mobileHash("agent", route.id, "", view)); }}
        onBack={() => goBack(route, agentOwnerWs(current))}
        onStart={startAgent}
        onStop={stopAgent}
        onOpenFiles={openFiles}
        onOpenGit={openGit}
        onOpenInspector={() => setInspDrawer({ owner: { kind: "agent", id: route.id }, root: "", title: current && current.agent ? (current.agent.name && current.agent.name !== "default" ? current.agent.name : (current.workspace ? current.workspace.name : current.agent.name)) : "Inspector" })}
        onAgentConfig={patchAgent}
        onRemoveTerminal={removeTerminal}
      />
    );
  } else if (route.screen === "pin") {
    body = <PinScreen key={route.id} pinId={route.id} onBack={() => goBack(route)} onEdit={(id) => push("#/pins/" + encodeURIComponent(id) + "/edit")} />;
  } else if (route.screen === "pinEdit") {
    body = <PinEdit key={route.id || "new"} pinId={route.id} onBack={() => goBack(route)} onSaved={(id) => { location.replace(location.pathname + location.search + "#/pins/" + encodeURIComponent(id)); }} />;
  } else if (route.screen === "snip") {
    body = <SnipScreen key={route.id} snipId={route.id} onBack={() => goBack(route)} onEdit={(id) => push("#/snippets/" + encodeURIComponent(id) + "/edit")} />;
  } else if (route.screen === "snipEdit") {
    body = <SnipEdit key={route.id || "new"} snipId={route.id} onBack={() => goBack(route)} onSaved={(id) => { location.replace(location.pathname + location.search + "#/snippets/" + encodeURIComponent(id)); }} />;
  } else if (route.screen === "inbox" && route.id) {
    body = <InboxItem manifest={inboxApp} itemId={route.id} onBack={() => goBack(route)} onGoto={onAppGoto} />;
  } else if (route.screen === "inbox") {
    body = <Inbox manifest={inboxApp} onOpenItem={(id) => push(mobileHash("inbox", id))} />;
  } else if (route.screen === "work") {
    body = (
      <Work section={section} focusWs={section === "workspaces" ? route.id : ""} onSection={setSection} loaded={loaded} error={fleetError} workspaces={workspaces} freeAgents={freeAgents} terminals={terminals}
        workingIds={tuiWorking} busyId={busyId} checklists={checklists}
        onOpenAgent={(a) => openAgent(a.id)} onOpenTerm={(t) => openTerm(t.id)} onTermAction={onTermAction} onAgentAction={onAgentAction} clis={clis}
        onCreate={(kind, ws) => (kind === "free" ? setCliPrincipalWs({ free: true }) : setCreate({ kind, workspace: ws || (kind === "agent" ? (workspaces[0] || null) : null) }))} onNewTerm={newTerminal} onNewCliPrincipal={setCliPrincipalWs}
        onOpenInspector={openInspector} onOpenFiles={openFiles} onOpenGit={openGit} onRefresh={refreshAll} />
    );
  } else if (route.screen === "more") {
    body = (
      <More fleetReady={loaded} legacyAgentId={lastAgentId} section={route.section} apps={apps} catalog={catalog} clis={clis} system={system} version={version} themeMode={themeMode} workspaces={workspaces} freeAgents={freeAgents}
        onAgentConfig={patchAgent}
        onTheme={(m) => { persistTheme(m); setThemeMode(m); }} last={last} onRefreshCatalog={loadCatalog} onCatalogChange={setCatalog}
        onShare={() => setShareOpen(true)} onWhatsNew={openWhatsNew} whatsNewUnread={whatsNewUnread} onBack={() => goBack(route)} />
    );
  } else {
    body = (
      <Now loaded={loaded && attentionReady} error={fleetError || attentionError} entries={entries} stats={stats} results={results}
        fleetTotal={fleetTotal + terminals.length} onAnswer={answerAsk} onRespond={respondInbox}
        onOpenAgent={openAgent} onOpenInbox={(id) => push(mobileHash("inbox", id))} onRefresh={refreshAll}
        onCreate={(kind) => (kind === "free" ? setCliPrincipalWs({ free: true }) : setCreate({ kind, workspace: null }))} />
    );
  }

  return (
    <div id="m-app" data-screen={route.screen} className={pushed ? "is-pushed" : ""}>
      <div className="m-body"><ScreenBoundary key={route.screen + ":" + route.id + ":" + route.section}><Suspense fallback={<ScreenLoading />}>{body}</Suspense></ScreenBoundary></div>
      {pushed ? null : <TabBar active={tab} badges={badges} />}
      <CreateSheet open={!!create} kind={create ? create.kind : "workspace"} workspace={create ? create.workspace : null} catalog={catalog}
        onClose={() => setCreate(null)} onCreated={onCreated} />
      <Suspense fallback={null}>
        <InspectorDrawer drawer={inspDrawer} onClose={() => setInspDrawer(null)}
          onOpenFile={({ owner: target, path, root }) => openFiles(target || (inspDrawer && inspDrawer.owner) || { kind: "workspace", id: "" }, { path, root })}
          onOpenTerminal={prepareGit} onAskAgent={askGit}
          workspaces={workspaces} freeAgents={freeAgents} terminals={terminals} />
      </Suspense>
      <NewCliPrincipal
        open={!!cliPrincipalWs}
        workspace={cliPrincipalWs}
        onClose={() => setCliPrincipalWs(null)}
        onCreated={async (created) => {
          setCliPrincipalWs(null);
          if (!created || !created.id) return;
          if (created.cli && created.cli !== "pi" && created.terminalId) {
            try {
              await api("/api/terminals/" + encodeURIComponent(created.terminalId) + "/launch/start", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ confirm: false }),
              });
            } catch (e) { toastError(e); }
            await reload({ force: true });
            openTerm(created.terminalId);
            return;
          }
          await reload({ force: true });
          openAgent(created.id);
        }}
      />
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
      <Toasts />
      {reconnect ? <Reconnect onReload={() => location.reload()} /> : null}
      <WhatsNew open={whatsNewOpen} onClose={closeWhatsNew} currentSemver={whatsNewCurrent} seenVersion={whatsNewSeen} notes={RELEASE_NOTES} unseenOnly={whatsNewMode === "auto"} />
      <ConfirmDialog />
      <PromptDialog />
      <SessionHandoffDialog
        open={!!termHandoff}
        session={termHandoff ? termHandoff.session : null}
        sourceCli={termHandoff ? termHandoff.sourceCli : ""}
        sourceName={termHandoff ? termHandoff.sourceName : ""}
        target={termHandoff ? termHandoff.target : null}
        onClose={() => setTermHandoff(null)}
        onDone={onTermHandoffDone}
      />
    </div>
  );
}
