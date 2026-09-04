import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError, wsURL } from "../lib/api.js";
import { bashLine } from "../lib/bashLine.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { readContextMenuPrefs, modifierHeld } from "../lib/contextMenuPrefs.js";
import { matchAction } from "../lib/appKeys.js";
import { applyTermChrome } from "../lib/termTheme.js";
import { closeTerm } from "../lib/terms.js";
import { termWorkspaceId, workspaceForTerminal } from "../lib/termGroups.js";
import { closeShellTerm } from "../components/ShellTerm.jsx";
import { summarizeArgs } from "../components/Conversation.jsx";
import { fileChangeFromTool } from "../lib/diff.js";
import { previewFromDetails } from "../lib/toolPreview.js";
import { eventsToItems } from "../lib/replay.js";
import { readCompacting, writeCompacting } from "../lib/compact.js";
import Sidebar from "../components/Sidebar.jsx";
import AgentTabs from "../components/AgentTabs.jsx";
import DashboardView from "../components/DashboardView.jsx";
import SessionBar from "../components/SessionBar.jsx";
import ChatSurface from "../components/ChatSurface.jsx";
import TermSurface from "../components/TermSurface.jsx";
import FileSurface from "../components/FileSurface.jsx";
import GitGraphSurface from "../components/GitGraphSurface.jsx";
import FileTreeSurface from "../components/FileTreeSurface.jsx";
import Settings from "../components/Settings.jsx";
import PiSettings from "../components/PiSettings.jsx";
import System from "../components/System.jsx";
import Providers from "../components/Providers.jsx";
import Mcps from "../components/Mcps.jsx";
import Packages from "../components/Packages.jsx";
import Devices from "../components/Devices.jsx";
import Automations from "../components/Automations.jsx";
import Palette from "../components/Palette.jsx";
import ContextMenu from "../components/ContextMenu.jsx";
import SessionTree from "../components/SessionTree.jsx";
import SessionInfo from "../components/SessionInfo.jsx";
import CreateForm from "../components/CreateForm.jsx";
import { parseRoute, go, providersNew, providersLlama, agentRoute, workspaceHash, termRoute, termHash, termTabId, isTermTab, tabTermId, fileRoute, fileHash, fileTabId, isFileTab, parseFileTab, gitRoute, gitHash, gitTabId, isGitTab, treeRoute, treeHash, treeTabId, isTreeTab, appRoute, appHash, appTabId, isAppTab, tabAppId } from "../lib/routes.js";
import AppSurface from "../components/AppSurface.jsx";
import { normalizeManifests } from "../lib/appPrimitives.js";
const PinStudio = lazy(() => import("../components/PinStudio.jsx"));
import { startPresence } from "../lib/device.js";
import { startReconnectWatch } from "../lib/reconnect.js";
import { startFeed, subscribeFeed, feedConnected } from "../lib/feed.js";
import { applyFleet, applyTui, applyUsage, touches } from "../lib/feedReducers.js";
import { applyChecklists, indexChecklists } from "../lib/checklist.js";
import { workspaceStatusPath } from "../lib/statusbar.js";
import Reconnect from "../components/Reconnect.jsx";
import { setShell } from "../lib/shell.js";
import { toast, toastError } from "../lib/toast.js";
import { pendingFollowUps, dropQueued, startEditQueued, saveEditQueued, cancelEditQueued } from "../lib/queue.js";
import { putAsk, answerAsk, timeoutAsk, cancelOpenAsks, askJustAnswered, backAsk, walkReply, noteAsk, unanswerAsk, slashNoteTarget, BACK } from "../lib/askForm.js";
import { writeAskMemory, mergeAskMemory } from "../lib/askMemory.js";
import { readDraft, writeDraft, clearDraft } from "../lib/draft.js";
import { askConfirm, fmtBytes } from "../lib/confirm.js";
import { stuckToBottom, pinToBottom } from "../lib/stickScroll.js";
import { alertFromPi } from "../lib/piError.js";
import { mergeAssistant } from "../lib/assistantMsg.js";
import { isSearchTool, hitsFromResult } from "../lib/searchCards.js";
import ConfirmDialog from "../components/ConfirmDialog.jsx";
import PromptDialog from "../components/PromptDialog.jsx";
import { askPrompt } from "../lib/prompt.js";
import { locate, firstAgentId, displayAgentName, mentionAgents } from "../lib/tree.js";
import { leafUserId } from "../lib/sessionCards.js";

function workspaceAPI(workspaces, freeAgents, selectedId, suffix) {
  const loc = locate(workspaces, freeAgents, selectedId);
  const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : "");
  if (!id) return "";
  const q = selectedId ? "?agent=" + encodeURIComponent(selectedId) : "";
  return "/api/workspaces/" + id + suffix + q;
}
import { extraSlash } from "../lib/slash.js";
import { isAutomateCommand, automatePrompt, parseAutomateReply } from "../lib/automateDraft.js";
import { writeAutomationDraft } from "../lib/automationDraft.js";
import { isValidCron } from "../lib/cron.js";
import { readOpenTabs, writeOpenTabs, filterOpenTabs, moveTab, readTermWanted, writeTermWanted, readGitOwners, writeGitOwners, readTreeOwners, writeTreeOwners } from "../lib/openTabs.js";
import { sessionsHash, sessionsRoute } from "../lib/routes.js";
import SessionsView from "../components/SessionsView.jsx";
import Hotkeys from "../components/Hotkeys.jsx";
import Changelog from "../components/Changelog.jsx";
import ShareGist from "../components/ShareGist.jsx";
import LlamaDialog from "../components/LlamaDialog.jsx";
import TermSettingsPage from "../components/TermSettingsPage.jsx";
import { createWorkspaceSchema, createWorkspaceCloneSchema, createFreeAgentSchema, createWsAgentSchema, parseForm } from "../lib/schemas.js";
import { parentDir } from "../lib/cloneUrl.js";
import Toasts from "../components/Toasts.jsx";

export default function App() {
  const [workspaces, setWorkspaces] = useState([]);
  const [freeAgents, setFreeAgents] = useState([]);
  const [selectedId, setSelectedId] = useState(() => readOpenTabs().selected);
  const [formKind, setFormKind] = useState("workspace");
  const [formWs, setFormWs] = useState("");
  const [tabs, setTabs] = useState(() => readOpenTabs().ids);
  const [tabsReady, setTabsReady] = useState(false);
  const [dashboardPinned, setDashboardPinned] = useState(false);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [host, setHost] = useState("local");
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [route, setRoute] = useState(() => parseRoute());
  const [hash, setHash] = useState(() => (typeof location !== "undefined" ? location.hash : "#/"));
  const [goneId, setGoneId] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [ctxMenu, setCtxMenu] = useState(null);
  const [treeOpen, setTreeOpen] = useState(false);
  const [sessionOpen, setSessionOpen] = useState(false);
  const [treeMode, setTreeMode] = useState("tree");
  const [treeData, setTreeData] = useState({ tree: [], leafId: "" });
  const [catalog, setCatalog] = useState({ providers: [], thinking: [] });
  const [newCfg, setNewCfg] = useState({ provider: "", model: "", thinking: "" });
  const [showForm, setShowForm] = useState(false);
  const [formError, setFormError] = useState("");
  const [formBusy, setFormBusy] = useState(false);
  const [piSessions, setPiSessions] = useState(null);
  const [termWanted, setTermWanted] = useState(() => new Set(readTermWanted()));
  const [termEpochs, setTermEpochs] = useState({});
  const [tuiWorking, setTuiWorking] = useState([]);
  const [checklists, setChecklists] = useState({});
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const draftAgentRef = useRef(null);
  const [status, setStatus] = useState("idle");
  const [streaming, setStreaming] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const streamingRef = useRef(false);
  const foreignTurnRef = useRef(false); // a turn started by an automation or another tab
  // Working shown before the server confirmed a turn (extension commands
  // never confirm one) — cleared by the first event that says what is
  // actually happening, or by a short fallback after task_delivered.
  const optimisticRef = useRef(false);
  const waitingRef = useRef(false);
  waitingRef.current = waiting;
  const flushingRef = useRef(false);
  // Which agent the items on screen belong to (guards ask-memory writes).
  const itemsAgentRef = useRef("");
  // Active pi-roles state for the composer chip (null = no chip).
  const [roleState, setRoleState] = useState(null);
  // Last snapshot per panel: reconciles restored open asks against reality.
  const snapWaitingRef = useRef({ agentId: "", waiting: false });
  const [items, setItems] = useState([]);
  const itemsRef = useRef([]);
  itemsRef.current = items;
  const automateRef = useRef(null); // {agentId, description, agentName, workspaceId}: a /automate turn in flight
  const [earlierRemaining, setEarlierRemaining] = useState(0);
  const earlierSkipRef = useRef(0);
  const earlierLoadingRef = useRef(false);
  const [sessions, setSessions] = useState([]);
  const [sessionCurrent, setSessionCurrent] = useState("");
  const [slashExtra, setSlashExtra] = useState([]);
  const [hotkeysOpen, setHotkeysOpen] = useState(false);
  const [changelogOpen, setChangelogOpen] = useState(false);
  const [llamaOpen, setLlamaOpen] = useState(false);
  const [reconnect, setReconnect] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  const [shareLinks, setShareLinks] = useState({ gist: "", viewer: "" });
  const [statusBar, setStatusBar] = useState(null);
  // Compaction in flight, per agent: { agentId: startedAtMs }. Lives outside
  // the conversation items so a panel rebuild (TUI→managed switch) can't drop it.
  const [compacting, setCompacting] = useState(readCompacting);
  const compactingRef = useRef(compacting);
  compactingRef.current = compacting;
  const agentIdRef = useRef(null);

  function setCompact(id, since) {
    if (!id) return;
    setCompacting((cur) => {
      const n = { ...cur };
      if (since == null) delete n[id];
      else n[id] = since;
      writeCompacting(n);
      return n;
    });
  }

  const [pkgUpdates, setPkgUpdates] = useState([]);

  const [terminals, setTerminals] = useState([]);
  // Apps host (ADR-0036): manifests + badges from GET /api/apps.
  const [apps, setApps] = useState([]);
  const [appsLoaded, setAppsLoaded] = useState(false);
  // A graph tab is named by its repository, but only an owner can be asked for
  // it, so remember which owner opened each one (ADR-0022).
  const [gitOwners, setGitOwners] = useState(() => readGitOwners());
  const [treeOwners, setTreeOwners] = useState(() => readTreeOwners());
  const [termError, setTermError] = useState("");
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const panelRef = useRef(null);
  const pendingPayload = useRef("");

  const selectedRef = useRef(null);
  selectedRef.current = selectedId;

  const located = locate(workspaces, freeAgents, selectedId);
  const selected = located && located.workspace;
  const agent = located && located.agent;
  // A workspace terminal tab still has that folder as the packages/MCP context
  // (machine list must not disappear — same rule as GET /api/packages).
  const paneWs = selected || (isTermTab(selectedId) ? workspaceForTerminal(terminals, workspaces, tabTermId(selectedId)) : null);
  agentIdRef.current = (agent && agent.id) || null;
  // Compaction progress is a live line at the end of the chat (not the
  // composer statusbar); CompactLive owns its own per-second tick.
  const compactSince = (agent && compacting[agent.id]) || null;
  const stopped = !agent || agent.mode === "stopped";
  // An interactive (TUI) agent has no event channel: the server scrapes
  // the pane and publishes agent.tui (ADR-0048). When pi is busy there,
  // the chat says so instead of reading as idle.
  const tuiBusy = !!(agent && agent.mode === "interactive" && tuiWorking.includes(agent.id));
  const interactive = !!(agent && agent.mode === "interactive");
  const termView = !!(selectedId && termWanted.has(selectedId));
  const atAgents = useMemo(
    () => mentionAgents(workspaces, freeAgents, selectedId),
    [workspaces, freeAgents, selectedId],
  );

  useEffect(() => {
    function onOpen(ev) {
      const p = ev && ev.detail;
      const id = selectedRef.current;
      if (!id || !p) return;
      setFileByAgent((s) => ({ ...s, [id]: String(p) }));
    }
    window.addEventListener("picode-open-file", onOpen);
    return () => window.removeEventListener("picode-open-file", onOpen);
  }, []);

  useEffect(() => {
    function onKey(e) {
      if (matchAction("app.terminal.new", e)) {
        e.preventDefault();
        window.dispatchEvent(new Event("picode-new-term"));
      }
    }
    function onNew() { createTerminal(); }
    document.addEventListener("keydown", onKey);
    window.addEventListener("picode-new-term", onNew);
    return () => {
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("picode-new-term", onNew);
    };
  }, []);

  const pkgWs = paneWs ? paneWs.id : "";
  const pkgScope = pkgWs ? "workspace:" + pkgWs : "user";
  useEffect(() => {
    let stop = false;
    async function load() {
      try {
        const q = pkgWs ? "?workspace=" + encodeURIComponent(pkgWs) : "";
        const page = await api("/api/packages/updates" + q);
        if (!stop) setPkgUpdates(page.updates || []);
      } catch { /* keep last */ }
    }
    load();
    // Change feed (ADR-0048): the server scans on a slow ticker for the
    // whole fleet and publishes packages.updates per scope when the
    // result changes — the event carries the list, so applying it is
    // free. The interval below is only the feed-down fallback.
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") { load(); return; }
      if (ev.type === "packages.updates" && ev.data && ev.data.scope === pkgScope) setPkgUpdates(ev.data.updates || []);
    });
    const t = setInterval(() => { if (!feedConnected()) load(); }, 30 * 60 * 1000);
    return () => { stop = true; clearInterval(t); unsub(); };
  }, [pkgWs]);
  useEffect(() => {
    if (route !== "packages") return;
    const q = pkgWs ? "?workspace=" + encodeURIComponent(pkgWs) : "";
    api("/api/packages/updates" + q).then((p) => setPkgUpdates(p.updates || [])).catch(() => {});
  }, [route, pkgWs]);

  useEffect(() => { applyTheme(themeMode); }, [themeMode]);
  useEffect(() => { applyTermChrome(); }, []);
  useEffect(() => {
    const mq = matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => { if (themeMode === "system") applyTheme("system"); };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [themeMode]);

  useEffect(() => {
    const onHash = () => {
      setRoute(parseRoute());
      setHash(location.hash);
    };
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    const onKey = (e) => {
      if (matchAction("app.palette.toggle", e)) {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  useEffect(() => {
    function onContextMenu(e) {
      if (e.target.closest(".xterm, .term-pane")) return; // terminals: untouched for now
      if (paletteOpen) return; // avoid stacking on top of the palette
      const { bypassModifier } = readContextMenuPrefs();
      if (modifierHeld(bypassModifier, e)) return; // let the native/system menu show
      e.preventDefault();
      setCtxMenu({ x: e.clientX, y: e.clientY, selection: window.getSelection().toString(), target: e.target });
    }
    document.addEventListener("contextmenu", onContextMenu);
    return () => document.removeEventListener("contextmenu", onContextMenu);
  }, [paletteOpen]);

  const loadWorkspaces = useCallback(async () => {
    const list = await api("/api/workspaces");
    setWorkspaces(list);
    try { setFreeAgents(await api("/api/agents?free=1")); }
    catch { setFreeAgents([]); }
    return list;
  }, []);

  const loadSessions = useCallback(async (wsId, opts) => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setSessions([]); setSessionCurrent(""); return; }
    try {
      const q = selectedId ? "?agent=" + encodeURIComponent(selectedId) : "";
      const data = await api("/api/workspaces/" + id + "/sessions" + q);
      setSessions(data.sessions || []);
      const newest = (data.sessions || [])[0] && (data.sessions || [])[0].path;
      const cur = (opts && opts.preferNewest && newest) ? newest : (data.current || "");
      setSessionCurrent(cur);
      if (!cur) {
        // No session file yet (extension commands only) — the thread
        // still restores from the agent's live ask-memory slot.
        let live = mergeAskMemory(selectedId, "", []);
        const snapLive = snapWaitingRef.current;
        if (snapLive.agentId === selectedId && !snapLive.waiting) live = cancelOpenAsks(live);
        itemsAgentRef.current = selectedId || "";
        setItems(live);
        setEarlierRemaining(0);
        return;
      }
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur) + (selectedId ? "&agent=" + encodeURIComponent(selectedId) : "") + "&tail=200");
      const ev = t.events || [];
      earlierSkipRef.current = 0;
      setEarlierRemaining(t.remaining || 0);
      let merged = mergeAskMemory(selectedId, cur, ev.length ? eventsToItems(ev) : []);
      // A restored open stepper whose dialog died with the flow is a ghost:
      // the snapshot said nothing is waiting, so close it quietly.
      const snap = snapWaitingRef.current;
      if (snap.agentId === selectedId && !snap.waiting) merged = cancelOpenAsks(merged);
      itemsAgentRef.current = selectedId || "";
      setItems(merged);
      scrollToEnd();
      if ((t.bytes || 0) > 32 * 1024 * 1024) {
        if (t.compacted) {
          toast.info("Compacted — the file stays large on disk, so cold boots stay slow until pi loads sessions lazily.");
        } else {
          toast.info("Huge session — run /compact to shrink future boots.");
        }
      }
    } catch { setSessions([]); setSessionCurrent(""); }
  }, [selectedId, workspaces, freeAgents]);

  const fetchEarlier = useCallback(async () => {
    if (earlierLoadingRef.current || earlierSkipRef.current < 0) return;
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id || !selectedId) return;
    earlierLoadingRef.current = true;
    try {
      const cur = sessionCurrent || "";
      if (!cur) return;
      const skip = earlierSkipRef.current + 200;
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur) + (selectedId ? "&agent=" + encodeURIComponent(selectedId) : "") + "&tail=200&skip=" + skip);
      const ev = t.events || [];
      if (ev.length) {
        const older = eventsToItems(ev);
        setItems((c) => mergeAskMemory(selectedId, cur, older.concat(c)));
        earlierSkipRef.current = skip;
      }
      setEarlierRemaining(t.remaining || 0);
    } catch { /* keep what we have */ }
    finally { earlierLoadingRef.current = false; }
  }, [selectedId, workspaces, freeAgents, sessionCurrent]);

  const pinNewestSession = useCallback(async () => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id || !selectedId) return;
    try {
      const data = await api("/api/workspaces/" + id + "/sessions?agent=" + encodeURIComponent(selectedId));
      setSessions(data.sessions || []);
      const newest = (data.sessions || [])[0] && (data.sessions || [])[0].path;
      if (!newest) return;
      setSessionCurrent(newest);
      await api("/api/agents/" + selectedId, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sessionPath: newest }),
      });
    } catch { /* live chat stays */ }
  }, [selectedId, workspaces, freeAgents]);

  const fetchRoleState = useCallback(async () => {
    const id = selectedRef.current;
    if (!id) { setRoleState(null); return; }
    try {
      const d = await api("/api/agents/" + id + "/role-state");
      if (selectedRef.current === id) setRoleState((d && d.state) || null);
    } catch { /* keep the last known state */ }
  }, []);
  useEffect(() => {
    setRoleState(null);
    if (selectedId) fetchRoleState();
  }, [selectedId, fetchRoleState]);

  useEffect(() => { loadSessions(); }, [selectedId, workspaces.length, freeAgents.length]);
  useEffect(() => {
    if (!selectedId) { setSlashExtra([]); return; }
    api("/api/agents/" + selectedId + "/slash")
      .then((d) => setSlashExtra(extraSlash(d.skills, d.templates, d.commands)))
      .catch(() => setSlashExtra([]));
  }, [selectedId, agent && agent.mode]);
  useEffect(() => { scrollConv(); }, [items]);
  useEffect(() => {
    // Only when the items on screen belong to this agent — on a tab switch
    // this effect fires with the previous agent's thread still in state.
    if (selectedId && itemsAgentRef.current === selectedId) {
      writeAskMemory(selectedId, sessionCurrent, items);
    }
  }, [items, selectedId, sessionCurrent]);

  const loadStatus = useCallback(async (wsId) => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setStatusBar(null); return; }
    // The bar is the SELECTED agent's session. Without ?agent= the server
    // answers with the workspace's first agent, which read as another
    // agent's context, spend and cache on every later agent's screen.
    const barAgent = loc && loc.workspace && loc.agent ? loc.agent.id : "";
    try { setStatusBar(await api(workspaceStatusPath(id, barAgent))); }
    catch { setStatusBar(null); }
  }, [selectedId, workspaces, freeAgents]);

  useEffect(() => { loadStatus(); }, [selectedId, sessionCurrent, loadStatus]);
  useEffect(() => {
    if (!agent || agent.mode !== "managed") return;
    const t = setInterval(() => { if (!feedConnected()) loadStatus(); }, 15000);
    return () => clearInterval(t);
  }, [agent && agent.mode, selectedId, loadStatus]);
  // agent.usage (ADR-0048): add this message's tokens and cost to the bar
  // instead of rescanning the session file; a settle still refetches so
  // the file stays the authority.
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "agent.usage" && ev.data && ev.data.agentId === selectedRef.current) setStatusBar((bar) => applyUsage(bar, ev.data));
  }), []);
  // refreshFleetFallback: after a local mutation the store's own event
  // already patched the lists; only refetch when the feed is not there.
  const refreshFleetFallback = useCallback(async () => {
    if (!feedConnected()) await loadWorkspaces();
  }, [loadWorkspaces]);

  useEffect(() => {
    (async () => {
      try {
        const [sys, ver] = await Promise.all([api("/api/system"), api("/api/version")]);
        setSystem(sys);
        setVersion(ver.version);
        setHost((sys.host && sys.host.name) || "local");
      } catch { /* offline */ }
      try { setCatalog(await api("/api/catalog")); } catch { /* pi missing */ }
      try {
        const list = await loadWorkspaces();
        let free = [];
        try { free = await api("/api/agents?free=1"); } catch { free = []; }
        let terms = [];
        try { terms = (await api("/api/terminals")).terminals || []; } catch { terms = []; }
        setTerminals(terms);
        let appList = [];
        let appsOk = false;
        try {
          appList = normalizeManifests(await api("/api/apps"));
          appsOk = true;
        } catch { appList = []; }
        setApps(appList);
        setAppsLoaded(appsOk);
        const owners = readGitOwners();
        const towners = readTreeOwners();
        const ownerAlive = (o) =>
          !!o &&
          (o.kind === "term"
            ? terms.some((t) => t.id === o.id)
            : o.kind === "workspace"
              ? list.some((w) => w.id === o.id)
              : !!locate(list, free, o.id));
        const exists = (id) => {
          if (isTermTab(id)) return terms.some((t) => t.id === tabTermId(id));
          // A failed /api/apps fetch must not wipe persisted app tabs.
          if (isAppTab(id)) return appsOk ? appList.some((a) => a.id === tabAppId(id)) : true;
          if (isGitTab(id)) return ownerAlive(owners[id]);
          if (isTreeTab(id)) return ownerAlive(towners[id]);
          if (isFileTab(id)) {
            const f = parseFileTab(id);
            if (!f) return false;
            return ownerAlive(f);
          }
          return !!locate(list, free, id);
        };
        const next = filterOpenTabs(readOpenTabs(), exists);
        setTabs(next.ids);
        const fromTerm = parseRoute() === "workspace" ? termRoute() : null;
        const fromFile = parseRoute() === "workspace" ? fileRoute() : null;
        const fromGit = parseRoute() === "workspace" ? gitRoute() : null;
        const fromTree = parseRoute() === "workspace" ? treeRoute() : null;
        const fromHash = parseRoute() === "workspace" ? agentRoute() : null;
        const fromApp = parseRoute() === "workspace" ? appRoute() : null;
        if (fromApp) {
          if (appList.some((a) => a.id === fromApp) || !appsOk) openTab(appTabId(fromApp));
          else { setGoneId(appTabId(fromApp)); setSelectedId(null); }
        } else if (fromTree) {
          if (ownerAlive(fromTree)) openTreeTab(fromTree.kind, fromTree.id);
          else { setGoneId(provisionalTreeId(fromTree.kind, fromTree.id)); setSelectedId(null); }
        } else if (fromGit) {
          if (ownerAlive(fromGit)) openGitTab(fromGit.kind, fromGit.id);
          else { setGoneId(provisionalGitId(fromGit.kind, fromGit.id)); setSelectedId(null); }
        } else if (fromTerm) {
          if (terms.some((t) => t.id === fromTerm)) openTermTab(fromTerm);
          else { setGoneId(termTabId(fromTerm)); setSelectedId(null); }
        } else if (fromFile) {
          const fid = fileTabId(fromFile.kind, fromFile.id, fromFile.path);
          if (exists(fid)) {
            setSelectedId(fid);
            setTabs((t) => (t.includes(fid) ? t : [...t, fid]));
          } else { setGoneId(fid); setSelectedId(null); }
        } else if (fromHash) {
          if (exists(fromHash)) openTab(fromHash, list);
          else { setGoneId(fromHash); setSelectedId(null); }
        } else if (next.selected) openTab(next.selected, list);
        else setSelectedId(null);
        setTabsReady(true);
      } catch (e) {
        console.error("boot:", e);
        setTabsReady(true);
      }
    })();
  }, [loadWorkspaces]);

  useEffect(() => startPresence(), []);
  useEffect(() => {
    // The TUI has no event channel; poll tmux for pi's own Working state.
    let stop = false;
    async function poll() {
      const ids = new Set(freeAgents.map((a) => a.id));
      for (const w of workspaces) for (const a of w.agents || []) ids.add(a.id);
      if (selectedId && !isTermTab(selectedId) && !isFileTab(selectedId)) ids.add(selectedId);
      if (!ids.size) return;
      try {
        const d = await api("/api/tui-working?ids=" + encodeURIComponent([...ids].join(",")));
        if (!stop) setTuiWorking(d.working || []);
      } catch { /* transient */ }
    }
    poll();
    // The server scrapes tmux itself now and publishes agent.tui
    // (ADR-0048); this 3 s loop only runs while the feed is down, and
    // once on (re)open to reconcile.
    const t = setInterval(() => { if (!feedConnected()) poll(); }, 3000);
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") poll();
      else if (ev.type === "agent.tui") setTuiWorking((cur) => applyTui(cur, ev));
    });
    return () => { stop = true; clearInterval(t); unsub(); };
  }, [workspaces, freeAgents, selectedId]);
  useEffect(() => {
    // Internal checklists (ADR-0055): one fetch per (re)open, then the feed.
    let stop = false;
    async function load() {
      try {
        const d = await api("/api/checklists");
        if (!stop) setChecklists(indexChecklists(d.checklists));
      } catch { /* the sidebar shows nothing until the next open */ }
    }
    load();
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset") load();
      else if (ev.type === "agent.checklist" || ev.type === "agent.deleted") setChecklists((cur) => applyChecklists(cur, ev));
    });
    return () => { stop = true; unsub(); };
  }, []);
  useEffect(() => {
    // Badge refresh (ADR-0036). Gentler than the 3s tui-working loop; the
    // ADR accepts seconds of badge latency. Boot did the first fetch;
    // errors keep the last known list.
    let stop = false;
    async function poll(force) {
      if (document.hidden) return;
      if (!force && feedConnected()) return;
      try {
        const d = await api("/api/apps");
        if (!stop) { setApps(normalizeManifests(d)); setAppsLoaded(true); }
      } catch { /* transient */ }
    }
    const t = setInterval(() => poll(false), 15000);
    // Change feed (ADR-0048): badges follow inbox changes at once.
    const unsub = subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset" || touches(ev, ["inbox"])) poll(true);
    });
    return () => { stop = true; clearInterval(t); unsub(); };
  }, []);
  // Change feed (ADR-0048): one stream per shell. Fleet events patch the
  // sidebar in place; anything the reducer cannot apply faithfully
  // refetches the fleet; (re)open and reset refetch everything.
  const fleetRef = useRef({ workspaces: [], freeAgents: [], terminals: [] });
  fleetRef.current = { workspaces, freeAgents, terminals };
  useEffect(() => startFeed(), []);
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset") {
      if (!ev.data || !ev.data.first) loadWorkspaces().catch(() => {});
      return;
    }
    if (!touches(ev, ["workspace", "agent", "terminal", "git"])) return;
    const next = applyFleet(fleetRef.current, ev);
    if (next === null) { loadWorkspaces().catch(() => {}); return; }
    if (next === fleetRef.current) return;
    setWorkspaces(next.workspaces);
    setFreeAgents(next.freeAgents);
    setTerminals(next.terminals);
  }), [loadWorkspaces]);
  useEffect(() => startReconnectWatch({
    onState: (s) => { if (s === "down") setReconnect(true); },
  }), []);
  useEffect(() => {
    if (!tabsReady) return;
    writeOpenTabs(tabs, selectedId);
  }, [tabs, selectedId, tabsReady]);
  useEffect(() => {
    writeTermWanted([...termWanted]);
  }, [termWanted]);
  useEffect(() => {
    if (!tabsReady) return;
    if (parseRoute(hash) !== "workspace") return;
    const tid = termRoute(hash);
    if (tid) {
      if (terminals.some((t) => t.id === tid)) {
        setGoneId((g) => (g ? "" : g));
        if (selectedRef.current !== termTabId(tid)) openTermTab(tid);
      } else {
        setGoneId((g) => (g === termTabId(tid) ? g : termTabId(tid)));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromFile = fileRoute(hash);
    if (fromFile) {
      const fid = fileTabId(fromFile.kind, fromFile.id, fromFile.path);
      const ok = fromFile.kind === "term"
        ? terminals.some((t) => t.id === fromFile.id)
        : fromFile.kind === "workspace"
          ? workspaces.some((w) => w.id === fromFile.id)
          : !!locate(workspaces, freeAgents, fromFile.id);
      if (ok) {
        setGoneId((g) => (g ? "" : g));
        if (selectedRef.current !== fid) {
          setSelectedId(fid);
          setTabs((t) => (t.includes(fid) ? t : [...t, fid]));
        }
      } else {
        setGoneId((g) => (g === fid ? g : fid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromGit = gitRoute(hash);
    if (fromGit) {
      const ok = fromGit.kind === "term"
        ? terminals.some((t) => t.id === fromGit.id)
        : !!locate(workspaces, freeAgents, fromGit.id);
      if (ok) {
        setGoneId((g) => (g ? "" : g));
        const known = Object.entries(gitOwners).find(([, o]) => o && o.kind === fromGit.kind && o.id === fromGit.id);
        if (!known || selectedRef.current !== known[0]) openGitTab(fromGit.kind, fromGit.id);
      } else {
        const gid = provisionalGitId(fromGit.kind, fromGit.id);
        setGoneId((g) => (g === gid ? g : gid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromTree = treeRoute(hash);
    if (fromTree) {
      const ok = fromTree.kind === "term"
        ? terminals.some((t) => t.id === fromTree.id)
        : fromTree.kind === "workspace"
          ? workspaces.some((w) => w.id === fromTree.id)
          : !!locate(workspaces, freeAgents, fromTree.id);
      if (ok) {
        setGoneId((g) => (g ? "" : g));
        const known = Object.entries(treeOwners).find(([, o]) => o && o.kind === fromTree.kind && o.id === fromTree.id);
        if (!known || selectedRef.current !== known[0]) openTreeTab(fromTree.kind, fromTree.id);
      } else {
        const tid = provisionalTreeId(fromTree.kind, fromTree.id);
        setGoneId((g) => (g === tid ? g : tid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromApp = appRoute(hash);
    if (fromApp) {
      const aid = appTabId(fromApp);
      // Before /api/apps answers, trust the hash — the surface handles a
      // manifest that never shows up.
      if (apps.some((a) => a.id === fromApp) || !appsLoaded) {
        setGoneId((g) => (g ? "" : g));
        if (selectedRef.current !== aid) openTab(aid);
      } else {
        setGoneId((g) => (g === aid ? g : aid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const id = agentRoute(hash);
    if (!id) {
      setGoneId((g) => (g ? "" : g));
      return;
    }
    if (locate(workspaces, freeAgents, id)) {
      setGoneId((g) => (g ? "" : g));
      if (selectedRef.current !== id) openTab(id);
    } else {
      setGoneId((g) => (g === id ? g : id));
      if (selectedRef.current) setSelectedId(null);
    }
    // Hash is the only input. Putting selectedId here fights the write effect
    // and loops the tab strip (URL says A, tab says B).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hash, tabsReady]);
  useEffect(() => {
    if (!goneId) return;
    if (locate(workspaces, freeAgents, goneId)) openTab(goneId);
  }, [goneId, workspaces, freeAgents]);
  useEffect(() => {
    if (!tabsReady) return;
    if (route !== "workspace") return;
    if (goneId) return;
    const file = isFileTab(selectedId) ? parseFileTab(selectedId) : null;
    const gitOwner = isGitTab(selectedId) ? gitOwners[selectedId] : null;
    const treeOwner = isTreeTab(selectedId) ? treeOwners[selectedId] : null;
    // Without the owner there is no hash to write, and guessing one would send
    // the router to #/agent/g:… — wait for the map instead.
    if (isGitTab(selectedId) && !gitOwner) return;
    if (isTreeTab(selectedId) && !treeOwner) return;
    const want = isTermTab(selectedId)
      ? termHash(tabTermId(selectedId))
      : isAppTab(selectedId)
        ? appHash(tabAppId(selectedId))
        : gitOwner
        ? gitHash(gitOwner.kind, gitOwner.id)
        : treeOwner
          ? treeHash(treeOwner.kind, treeOwner.id)
          : file
            ? fileHash(file.kind, file.id, file.path)
            : workspaceHash(selectedId);
    if (location.hash === want) return;
    // Only manage workspace-ish hashes. Route hashes (#/automations,
    // #/settings, #/devices, …) are deep links: replacing them here bounced
    // them to #/ once the fleet loaded — found by scripts/docs-shots.mjs,
    // which navigates cold to #/automations and never saw the view.
    if (parseRoute(location.hash) !== "workspace") return;
    if (!agentRoute(location.hash) && !termRoute(location.hash) && !fileRoute(location.hash) && !gitRoute(location.hash) && !treeRoute(location.hash) && !appRoute(location.hash) && selectedId) {
      history.replaceState(null, "", want);
      setHash(want);
      return;
    }
    location.hash = want;
  }, [selectedId, tabsReady, goneId, route, gitOwners, treeOwners]);
  useEffect(() => {
    const prev = draftAgentRef.current;
    if (prev && prev !== selectedId) writeDraft(prev, draft, kind);
    draftAgentRef.current = selectedId || null;
    const d = readDraft(selectedId);
    setDraft(d.text);
    setKind(d.kind);
    // Only when the selected agent changes — not on every keystroke.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedId]);
  useEffect(() => {
    if (!selectedId) return;
    const t = setTimeout(() => writeDraft(selectedId, draft, kind), 200);
    return () => clearTimeout(t);
  }, [draft, kind, selectedId]);

  function openTab(id, list) {
    setDashboardPinned(false);
    if (isTermTab(id)) { openTermTab(tabTermId(id)); return; }
    if (isGitTab(id) || isTreeTab(id) || isAppTab(id)) {
      setGoneId("");
      setSelectedId(id);
      setTabs((t) => (t.includes(id) ? t : [...t, id]));
      return;
    }
    if (isFileTab(id)) {
      const f = parseFileTab(id);
      if (!f) return;
      setGoneId("");
      setSelectedId(id);
      setTabs((t) => (t.includes(id) ? t : [...t, id]));
      return;
    }
    const loc = locate(list || workspaces, freeAgents, id);
    if (!loc || !loc.agent) return;
    const aid = loc.agent.id;
    setGoneId("");
    setSelectedId(aid);
    setTabs((t) => t.includes(aid) ? t : [...t, aid]);
    prepareSurface(loc.agent);
  }

  function revealAgent(id, list) {
    openTab(id, list);
    go("workspace", id);
  }

  function prepareSurface(a) {
    if (!a || a.mode === "stopped") {
      setStatus("stopped");
      setStreaming(false);
      streamingRef.current = false;
      setWaiting(false);
      return;
    }
    if (a.mode === "interactive") {
      setStatus("interactive");
      setStreaming(false);
      streamingRef.current = false;
      setWaiting(false);
    }
  }

  function openFileTab(kind, ownerId, path) {
    setDashboardPinned(false);
    if (!ownerId || !path) return;
    const id = fileTabId(kind, ownerId, path);
    setGoneId("");
    setSelectedId(id);
    setTabs((t) => (t.includes(id) ? t : [...t, id]));
  }

  function provisionalGitId(kind, ownerId) {
    return gitTabId("@" + (kind === "term" ? "t" : "a") + ":" + ownerId);
  }

  // The repository is unknown until the server answers, so a graph tab opens
  // under a provisional id and onGitKey renames it to g:<key> — which is also
  // where two owners of the same repo collapse onto one tab (ADR-0022).
  function openGitTab(kind, ownerId, ownerName) {
    setDashboardPinned(false);
    if (!ownerId) return;
    const known = Object.entries(gitOwners).find(([, o]) => o && o.kind === kind && o.id === ownerId);
    const id = known ? known[0] : provisionalGitId(kind, ownerId);
    setGitOwners((m) => {
      const next = { ...m, [id]: { kind, id: ownerId, name: ownerName || "" } };
      writeGitOwners(next);
      return next;
    });
    setGoneId("");
    setSelectedId(id);
    setTabs((t) => (t.includes(id) ? t : [...t, id]));
  }

  function onGitKey(fromId, key) {
    const real = gitTabId(key);
    if (!key || real === fromId) return;
    setGitOwners((m) => {
      const owner = m[fromId];
      const next = { ...m };
      delete next[fromId];
      if (owner && !next[real]) next[real] = owner;
      writeGitOwners(next);
      return next;
    });
    setTabs((t) => {
      const swapped = t.map((x) => (x === fromId ? real : x));
      return swapped.filter((x, i) => swapped.indexOf(x) === i);
    });
    setSelectedId((s) => (s === fromId ? real : s));
  }

  function provisionalTreeId(kind, ownerId) {
    return treeTabId("@" + (kind === "term" ? "t" : kind === "workspace" ? "w" : "a") + ":" + ownerId);
  }

  // The root folder is unknown until the server answers, so a tree tab opens
  // under a provisional id and onTreeKey renames it to d:<root> — which is
  // also where two owners of the same folder collapse onto one tab (ADR-0030).
  function openTreeTab(kind, ownerId, ownerName) {
    setDashboardPinned(false);
    if (!ownerId) return;
    const known = Object.entries(treeOwners).find(([, o]) => o && o.kind === kind && o.id === ownerId);
    const id = known ? known[0] : provisionalTreeId(kind, ownerId);
    setTreeOwners((m) => {
      const next = { ...m, [id]: { kind, id: ownerId, name: ownerName || "" } };
      writeTreeOwners(next);
      return next;
    });
    setGoneId("");
    setSelectedId(id);
    setTabs((t) => (t.includes(id) ? t : [...t, id]));
  }

  function onTreeKey(fromId, root) {
    const real = treeTabId(root);
    if (!root || real === fromId) return;
    setTreeOwners((m) => {
      const owner = m[fromId];
      const next = { ...m };
      delete next[fromId];
      if (owner && !next[real]) next[real] = owner;
      writeTreeOwners(next);
      return next;
    });
    setTabs((t) => {
      const swapped = t.map((x) => (x === fromId ? real : x));
      return swapped.filter((x, i) => swapped.indexOf(x) === i);
    });
    setSelectedId((s) => (s === fromId ? real : s));
  }

  function closeTab(id) {
    if (isTermTab(id)) closeShellTerm(tabTermId(id));
    if (isGitTab(id)) {
      setGitOwners((m) => {
        const next = { ...m };
        delete next[id];
        writeGitOwners(next);
        return next;
      });
    }
    if (isTreeTab(id)) {
      setTreeOwners((m) => {
        const next = { ...m };
        delete next[id];
        writeTreeOwners(next);
        return next;
      });
    }
    const ws = workspaces.find((w) => w.id === id);
    setTabs((t) => t.filter((x) => x !== id));
    setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
    if (ws && ws.agent) closeShellTerm(ws.agent.id);
    if (panelRef.current && ws && ws.agent && panelRef.current.agentId === ws.agent.id) closePanel();
    if (selectedId === id) {
      setTabs((t) => {
        const next = t[t.length - 1];
        if (next) {
          if (isTermTab(next)) openTermTab(tabTermId(next));
          else if (isFileTab(next) || isGitTab(next) || isTreeTab(next) || isAppTab(next)) {
            setSelectedId(next);
          } else {
            setSelectedId(next);
            const loc = locate(workspaces, freeAgents, next);
            if (loc && loc.agent) prepareSurface(loc.agent);
          }
        } else {
          setSelectedId(null);
        }
        return t;
      });
    }
  }

  function closePanel() {
    const p = panelRef.current;
    if (!p) return;
    p.stopped = true;
    try { p.sock.close(); } catch { /* ignore */ }
    panelRef.current = null;
  }

  function scrollConv() {
    if (!nearBottom.current) return;
    const go = () => {
      if (nearBottom.current) pinToBottom(convRef.current);
    };
    go();
    queueMicrotask(go);
    requestAnimationFrame(() => requestAnimationFrame(go));
  }

  function scrollToEnd() {
    nearBottom.current = true;
    scrollConv();
  }

  function putAskItem(d, status) {
    setItems((cur) => putAsk(cur, d, status));
  }

  function connectPanel(agentId) {
    closePanel();
    setStatus("idle");
    optimisticRef.current = false;
    setStreaming(false);
    streamingRef.current = false;
    setWaiting(false);
    const sock = new WebSocket(wsURL(`/ws/agent?agent=${agentId}`));
    const panel = { agentId, sock, stopped: false };
    panelRef.current = panel;
    sock.onmessage = (ev) => {
      try { handleEvent(JSON.parse(ev.data), panel); } catch { /* ignore */ }
    };
    sock.onclose = () => {
      if (panelRef.current === panel && !panel.stopped) {
        setStatus("disconnected");
        setStreaming(false);
        streamingRef.current = false;
        setWaiting(false);
        setItems((cur) => [...cur, { kind: "sys", text: "— panel disconnected —", err: true }]);
        // A dead socket must not stand in for a live one: when the agent
        // comes back managed (an automation run, another tab), the mode
        // effect reconnects instead of finding "the same panel" here.
        panelRef.current = null;
      }
      if (window.__picodeKickHealth) window.__picodeKickHealth();
    };
  }

  function handleEvent(env, panel) {
    const ev = env.event || {};
    switch (ev.type) {
      case "snapshot":
        optimisticRef.current = false;
        fetchRoleState();
        snapWaitingRef.current = { agentId: panel.agentId, waiting: !!ev.waiting };
        // Joining a turn we did not start (an automation's prompt, a
        // send from another tab): the session file already holds the
        // prompt, so the thread shows it — and again, in order, on settle.
        if (ev.streaming) { foreignTurnRef.current = true; queueMicrotask(() => loadSessions(null, { preferNewest: true })); }
        setStreaming(!!ev.streaming);
        streamingRef.current = !!ev.streaming;
        setWaiting(!!ev.waiting);
        setStatus(ev.waiting ? "waiting" : ev.streaming ? "streaming" : "idle");
        if (ev.waiting && ev.dialog) putAskItem(ev.dialog, "open");
        // Nothing is waiting server-side: a restored open stepper is a ghost.
        else setItems((cur) => cancelOpenAsks(cur));
        break;
      case "agent_start":
        if (!optimisticRef.current) foreignTurnRef.current = true; // nobody here typed it
        optimisticRef.current = false;
        setStreaming(true);
        streamingRef.current = true;
        setStatus((s) => (s === "waiting" ? "waiting" : "streaming"));
        scrollToEnd();
        break;
      case "agent_settled": {
        optimisticRef.current = false;
        setStreaming(false);
        streamingRef.current = false;
        setStatus((s) => (s === "waiting" ? "waiting" : "idle"));
        if (foreignTurnRef.current) {
          foreignTurnRef.current = false;
          queueMicrotask(() => loadSessions(null, { preferNewest: true }));
        }
        if (automateRef.current) { const aid = env.agentId || (panel && panel.agentId); setTimeout(() => finishAutomate(aid), 0); }
        if (selectedId) loadStatus();
        fetchRoleState();
        pinNewestSession();
        queueMicrotask(() => flushFollowUp());
        break;
      }
      case "message_update": {
        const d = ev.assistantMessageEvent;
        if (!d) break;
        if (d.type === "text_delta") {
          setItems((cur) => appendDelta(cur, "", "agent", d.delta || ""));
        } else if (d.type === "thinking_delta") {
          setItems((cur) => appendDelta(cur, "thinking", "thinking", d.delta || ""));
        }
        queueMicrotask(scrollConv);
        break;
      }
      case "tool_execution_start": {
        const change = fileChangeFromTool(ev.toolName, ev.args, null);
        setItems((cur) => [...cur, {
          kind: "tool",
          id: ev.toolCallId,
          name: ev.toolName || "tool",
          args: summarizeArgs(ev.args),
          toolArgs: ev.args || {},
          status: "···",
          detail: JSON.stringify(ev.args || {}, null, 2),
          expanded: false,
          change,
          preview: null,
          ts: Date.now(),
        }]);
        queueMicrotask(scrollConv);
        break;
      }
      case "tool_execution_update": {
        // ADR-0057: latest preview frame wins; nothing else moves.
        const preview = previewFromDetails(ev.partialResult && ev.partialResult.details);
        if (!preview) break;
        setItems((cur) => cur.map((it) => (it.kind === "tool" && it.id === ev.toolCallId ? { ...it, preview } : it)));
        break;
      }
      case "tool_execution_end":
        setItems((cur) => cur.map((it) => {
          if (it.kind !== "tool" || it.id !== ev.toolCallId) return it;
          const change = fileChangeFromTool(ev.toolName || it.name, ev.args, ev.result) || it.change;
          const searchHits = isSearchTool(ev.toolName || it.name) ? hitsFromResult(ev.result) : [];
          return {
            ...it,
            status: ev.isError ? "error" : "ok",
            detail: JSON.stringify(ev.result || {}, null, 2),
            result: ev.result,
            expanded: it.expanded || searchHits.length > 0,
            change,
            preview: previewFromDetails(ev.result && ev.result.details) || it.preview,
          };
        }));
        break;
      case "bash_execution_update": {
        const chunk = ev.delta || "";
        if (!chunk) break;
        setItems((cur) => cur.map((it) =>
          it.kind === "bash" && it.status === "run" ? { ...it, output: (it.output || "") + chunk } : it));
        queueMicrotask(scrollConv);
        break;
      }
      case "enqueue_accepted": {
        const text = pendingPayload.current;
        pendingPayload.current = "";
        setDraft("");
        if (selectedRef.current) clearDraft(selectedRef.current);
        if (text) {
          setItems((cur) => [...cur, {
            kind: "block", cls: "user", actor: "You", chip: ev.kind || "prompt", text, ts: Date.now(),
          }]);
        }
        queueMicrotask(scrollConv);
        break;
      }
      case "task_delivered":
        // Extension commands (/roles …) never start a turn, so nothing would
        // ever clear the optimistic Working. Once delivery is confirmed, a
        // real turn announces itself within moments — if nothing does and no
        // dialog is up, the command finished silently: go idle.
        if (optimisticRef.current) {
          setTimeout(() => {
            if (!optimisticRef.current) return;
            optimisticRef.current = false;
            if (!waitingRef.current) {
              setStreaming(false);
              streamingRef.current = false;
              setStatus("idle");
            }
          }, 3000);
        }
        break;
      case "message_end": {
        const m = ev.message || {};
        if (m.role === "assistant") {
          setItems((cur) => mergeAssistant(cur, m));
          queueMicrotask(scrollConv);
        }
        const a = alertFromPi(ev);
        if (a) {
          setItems((cur) => [...cur, { kind: "alert", level: a.level, text: a.text, ts: Date.now() }]);
          if (a.level === "error") {
            setStreaming(false);
            toastError(a.text);
          }
          queueMicrotask(scrollConv);
        }
        break;
      }
      case "turn_end": {
        const m = ev.message || {};
        if (m.role === "assistant") {
          setItems((cur) => mergeAssistant(cur, m));
          queueMicrotask(scrollConv);
        }
        const te = alertFromPi(ev);
        if (te) {
          setItems((cur) => [...cur, { kind: "alert", level: te.level, text: te.text, ts: Date.now() }]);
          if (te.level === "error") { setStreaming(false); toastError(te.text); }
        }
        break;
      }
      case "agent_end": {
        const ae = alertFromPi(ev);
        if (ae) {
          setItems((cur) => [...cur, { kind: "alert", level: ae.level, text: ae.text, ts: Date.now() }]);
          if (ae.level === "error" && !ev.willRetry) { setStreaming(false); toastError(ae.text); }
        }
        queueMicrotask(scrollConv);
        break;
      }
      case "compaction_end": {
        // pi finished compacting (user-initiated or auto). Clear the live
        // chat line and fold the summary into the one-line compact card;
        // pi's TUI shows its own feedback otherwise.
        setCompact(agentIdRef.current, null);
        const sum = !ev.aborted && ev.result && ev.result.summary ? String(ev.result.summary) : "";
        if (sum) {
          setItems((cur) => (cur.some((it) => it.kind === "compaction" && it.text === sum)
            ? cur
            : [...cur, { kind: "compaction", text: sum, ts: Date.now() }]));
          queueMicrotask(scrollConv);
        }
        break;
      }
      case "auto_retry_start":
      case "auto_retry_end":
      case "extension_error": {
        const a = alertFromPi(ev);
        if (a) {
          setItems((cur) => [...cur, { kind: "alert", level: a.level, text: a.text, ts: Date.now() }]);
          if (a.level === "error") {
            if (ev.type !== "auto_retry_start") setStreaming(false);
            toastError(a.text);
          }
          queueMicrotask(scrollConv);
        }
        break;
      }
      case "task_failed":
        setItems((cur) => [...cur, { kind: "alert", level: "error", text: humanizeError(ev.error || "Task failed"), ts: Date.now() }]);
        setStreaming(false);
        break;
      case "enqueue_rejected":
        toastError(ev.error);
        break;
      case "extension_ui_request": {
        const method = ev.method || "";
        if (method === "select" || method === "confirm" || method === "input" || method === "editor") {
          // A dialog means the extension is asking, not a turn running:
          // an unconfirmed Working becomes the waiting state.
          if (optimisticRef.current) {
            optimisticRef.current = false;
            setStreaming(false);
            streamingRef.current = false;
          }
          setWaiting(true);
          setStatus("waiting");
          setItems((cur) => {
            // Going back to a clicked pill: answer BACK to the wrong fields
            // instead of showing them; show the target when it arrives.
            const back = walkReply(cur, ev);
            if (back) {
              const aid = panel && panel.agentId;
              if (aid && ev.id) {
                queueMicrotask(() => {
                  api("/api/agents/" + aid + "/ui", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ id: ev.id, value: back }),
                  }).catch(() => {});
                });
              }
              return cur;
            }
            return putAsk(cur, ev, "open");
          });
        } else if (method === "notify") {
          const msg = ev.message || "Notice";
          // Any roles/extension notify may mean the mode changed.
          fetchRoleState();
          // Any notify from an extension command ends an unconfirmed Working.
          if (optimisticRef.current) {
            optimisticRef.current = false;
            setStreaming(false);
            streamingRef.current = false;
            setStatus(waitingRef.current ? "waiting" : "idle");
          }
          const noteTs = Date.now();
          setItems((cur) => {
            // Right after a finished form the notify is its result
            // (model · thinking · why) — fold it into the card's
            // definition line instead of toasting.
            if (ev.notifyType !== "error" && askJustAnswered(cur)) return noteAsk(cur, msg);
            // A command that asked nothing (/vision, /auto, missing
            // config): its notify IS the result — keep it in the thread
            // as a line, not as a toast that fades.
            const cmd = slashNoteTarget(cur);
            if (cmd) {
              if (cur.some((n) => n.kind === "note" && n.ts === noteTs && n.text === msg)) return cur;
              return [...cur, { kind: "note", cmd, level: ev.notifyType || "info", text: msg, ts: noteTs }];
            }
            queueMicrotask(() => (ev.notifyType === "error" ? toastError(msg) : toast.info(msg)));
            return cur;
          });
        }
        queueMicrotask(scrollConv);
        break;
      }
      case "extension_ui_timeout":
        setWaiting(false);
        setStatus(streamingRef.current ? "streaming" : "idle");
        setItems((cur) => timeoutAsk(cur, ev.id));
        break;
      case "exit":
        // The pi process is gone: any open ask card is dead — close it
        // quietly so nothing clickable points at a dead dialog.
        optimisticRef.current = false;
        setStreaming(false);
        streamingRef.current = false;
        setWaiting(false);
        setItems((cur) => cancelOpenAsks(cur));
        break;
      default:
        break;
    }
  }

  useEffect(() => {
    // `selected` is the workspace, and a free agent has none: the panel
    // follows the agent, never the workspace. (Until 2026-09-02 a free
    // agent started by an automation never got its chat connected.)
    if (!agent) { closePanel(); return; }
    if (agent.mode === "managed") {
      const p = panelRef.current;
      if (!p || p.agentId !== agent.id || (p.sock && p.sock.readyState > 1)) connectPanel(agent.id);
    } else {
      closePanel();
    }
    // connectPanel/closePanel are stable enough for this surface switch
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedId, agent && agent.id, agent && agent.mode]);

  async function startManaged(id) {
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) return;
    try {
      await api(`/api/agents/${loc.agent.id}/managed/start`, { method: "POST" });
      const list = await refreshFleetFallback();
      openTab(loc.agent.id, list);
    } catch (err) { toastError(err); }
  }

  async function openTermTab(id) {
    setDashboardPinned(false);
    if (!id) return;
    const tab = termTabId(id);
    setGoneId("");
    setTermError("");
    setSelectedId(tab);
    setTabs((t) => (t.includes(tab) ? t : [...t, tab]));
    try {
      const page = await api("/api/terminals/" + id + "/open", { method: "POST" });
      setTerminals((cur) => {
        const i = cur.findIndex((x) => x.id === id);
        if (i < 0) return [...cur, page];
        const next = cur.slice();
        next[i] = { ...cur[i], ...page };
        return next;
      });
    } catch (err) {
      setTermError(humanizeError(err && err.message ? err.message : String(err)));
    }
  }

  async function createTerminal(wsId) {
    try {
      const body = wsId ? JSON.stringify({ workspaceId: wsId }) : "{}";
      const page = await api("/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body });
      setTerminals((cur) => (cur.some((x) => x.id === page.id) ? cur : [...cur, page]));
      await openTermTab(page.id);
    } catch (err) { toastError(err); }
  }

  async function renameTerminal(t) {
    if (!t) return;
    const name = await askPrompt({
      title: "Rename terminal",
      defaultValue: t.name || "Terminal",
      confirmLabel: "Save",
    });
    if (!name) return;
    try {
      const page = await api("/api/terminals/" + t.id, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      setTerminals((cur) => cur.map((x) => (x.id === t.id ? { ...x, ...page } : x)));
    } catch (err) { toastError(err); }
  }

  async function removeTerminal(t) {
    if (!t) return;
    const ok = await askConfirm({
      title: "Remove terminal?",
      message: "This stops the tmux session. The tab closes.",
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/terminals/" + t.id, { method: "DELETE" });
      setTerminals((cur) => cur.filter((x) => x.id !== t.id));
      setTabs((cur) => cur.filter((id) => {
        const f = parseFileTab(id);
        return !(f && f.kind === "term" && f.id === t.id);
      }));
      closeTab(termTabId(t.id));
    } catch (err) { toastError(err); }
  }

  async function openInteractive(id, opts) {
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) return;
    try {
      const forceRestart = !!(opts && opts.restart);
      const restart = forceRestart ? "?restart=1" : "";
      if (forceRestart) closeShellTerm(loc.agent.id);
      await api(`/api/agents/${loc.agent.id}/open${restart}`, { method: "POST" });
      if (forceRestart) {
        setTermEpochs((cur) => ({ ...cur, [loc.agent.id]: (cur[loc.agent.id] || 0) + 1 }));
      }
      const list = await refreshFleetFallback();
      openTab(loc.agent.id, list);
      if (!opts || opts.dock !== false) {
        setTermWanted((s) => new Set(s).add(loc.agent.id));
      }
    } catch (err) { toastError(err); }
  }

  async function stopAgent(id) {
    if (automateRef.current && automateRef.current.agentId === id) automateRef.current = null;
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) return;
    try {
      await api(`/api/agents/${loc.agent.id}/close`, { method: "POST" });
      closeShellTerm(loc.agent.id);
      if (panelRef.current && panelRef.current.agentId === loc.agent.id) panelRef.current.stopped = true;
      optimisticRef.current = false;
      setStreaming(false);
      streamingRef.current = false;
      setWaiting(false);
      setStatus("stopped");
      setItems((cur) => cancelOpenAsks(cur));
      await refreshFleetFallback();
    } catch (err) { toastError(err); }
  }

  async function confirmCleanup({ title, message, path, extraChoices }) {
    let preview = { lastOccupant: false, sessions: 0, sessionBytes: 0, canPurgeWork: false };
    try { preview = await api(path); } catch { /* unregister still allowed */ }
    if (preview.terminals > 0) {
      const n = preview.terminals;
      message += ` This also removes ${n} terminal${n === 1 ? "" : "s"} and stops ${n === 1 ? "its" : "their"} tmux session${n === 1 ? "" : "s"}.`;
    }
    const choices = [];
    if (preview.lastOccupant && preview.sessions > 0) {
      const n = preview.sessions;
      choices.push({
        id: "sessions",
        label: `Also delete ${n} session${n === 1 ? "" : "s"} (${fmtBytes(preview.sessionBytes)}) for this folder`,
      });
    }
    if (preview.canPurgeWork) {
      choices.push({ id: "work", label: "Also delete the work folder" });
    }
    for (const c of extraChoices || []) choices.push(c);
    const ok = await askConfirm({ title, message, confirmLabel: "Remove", danger: true, choices });
    if (!ok) return null;
    const picked = ok === true ? {} : ok;
    const q = new URLSearchParams();
    if (picked.sessions) q.set("sessions", "1");
    if (picked.work) q.set("work", "1");
    for (const c of extraChoices || []) {
      if (!picked[c.id]) continue;
      for (const [k, v] of Object.entries(c.params || {})) q.set(k, v);
    }
    const qs = q.toString();
    return { query: qs ? "?" + qs : "" };
  }

  async function renameAgent(ag, shown) {
    if (!ag) return;
    // A workspace agent still called "default" shows its workspace's name;
    // the field opens on what the card says, never blank.
    const name = await askPrompt({
      title: "Rename agent",
      defaultValue: ag.name && ag.name !== "default" ? ag.name : (shown || ""),
      confirmLabel: "Save",
    });
    if (!name) return;
    try {
      await api("/api/agents/" + ag.id, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      await refreshFleetFallback();
    } catch (err) { toastError(err); }
  }

  async function removeAgent(ag) {
    const choice = await confirmCleanup({
      title: "Remove agent",
      message: `Remove "${ag.name}"? The project folder is not deleted.`,
      path: "/api/agents/" + ag.id + "/cleanup",
    });
    if (!choice) return;
    try {
      await api("/api/agents/" + ag.id + choice.query, { method: "DELETE" });
      closeShellTerm(ag.id);
      setTabs((t) => t.filter((x) => x !== ag.id));
      if (selectedId === ag.id) setSelectedId(null);
      await refreshFleetFallback();
    } catch (err) { toastError(err); }
  }

  async function removeWorkspace(ws) {
    // Opt-in local deletion (ADR-0035): GitHub-style typed confirmation.
    // The remote repository (if any) is never touched.
    const folderName = String(ws.path || "").split("/").filter(Boolean).pop() || "";
    const choice = await confirmCleanup({
      title: "Remove workspace",
      message: `Remove "${ws.name}"? The project folder is kept unless you say otherwise below.`,
      path: "/api/workspaces/" + ws.id + "/cleanup",
      extraChoices: folderName ? [{
        id: "files",
        label: `Also delete the project folder on disk (${ws.path})`,
        typed: {
          expected: folderName,
          hint: `This permanently deletes local files. Type "${folderName}" to confirm — a remote repository is not touched.`,
        },
        params: { files: "1", confirm: folderName },
      }] : [],
    });
    if (!choice) return;
    try {
      await api("/api/workspaces/" + ws.id + choice.query, { method: "DELETE" });
      const ids = (ws.agents || []).map((a) => a.id);
      if (ws.agent) ids.push(ws.agent.id);
      for (const id of [...new Set(ids)]) {
        closeShellTerm(id);
        if (panelRef.current && panelRef.current.agentId === id) closePanel();
      }
      // The workspace's terminals died with it (ADR-0026): drop them from
      // the list and close their tabs, like removeTerminal does.
      const deadTerms = terminals.filter((t) => termWorkspaceId(t) === ws.id);
      setTerminals((cur) => cur.filter((t) => termWorkspaceId(t) !== ws.id));
      for (const t of deadTerms) {
        setTabs((cur) => cur.filter((id) => {
          const f = parseFileTab(id);
          return !(f && f.kind === "term" && f.id === t.id);
        }));
        closeTab(termTabId(t.id));
      }
      setTabs((t) => t.filter((x) => x !== ws.id && !ids.includes(x)));
      if (selectedId === ws.id || ids.includes(selectedId)) setSelectedId(null);
      await refreshFleetFallback();
    } catch (err) { toastError(err); }
  }

  async function submitNew(e) {
    e.preventDefault();
    setFormError("");
    const fd = new FormData(e.target);
    const name = String(fd.get("name") || "");
    const path = String(fd.get("path") || "");
    if (formKind === "workspace" && String(fd.get("source") || "") === "remote") {
      // Clone mode (ADR-0034): one blocking request; the button says
      // "Cloning…" until the server answers. Closing the dialog does not
      // cancel the clone — the workspace shows up on the next load.
      const url = String(fd.get("url") || "");
      const parsedClone = parseForm(createWorkspaceCloneSchema, { url, name, path });
      if (!parsedClone.ok) { setFormError(parsedClone.error); return; }
      setFormBusy(true);
      try {
        await api("/api/workspaces/clone", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(parsedClone.value),
        });
        const parent = parentDir(parsedClone.value.path);
        if (parent) { try { localStorage.setItem("picode.cloneParent", parent); } catch { /* per-viewer nicety */ } }
        await loadWorkspaces();
        e.target.reset();
        setShowForm(false);
      } catch (err) {
        setFormError(humanizeError(err.message));
      } finally {
        setFormBusy(false);
      }
      return;
    }
    const schema = formKind === "workspace" ? createWorkspaceSchema : formKind === "free" ? createFreeAgentSchema : createWsAgentSchema;
    const parsed = parseForm(schema, formKind === "workspace" ? { name, path } : { name, path, ...newCfg });
    if (!parsed.ok) { setFormError(parsed.error); return; }
    const body = parsed.value;
    try {
      if (formKind === "workspace") {
        // The workspace starts empty (ADR-0027): nothing to open yet.
        await api("/api/workspaces", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        await loadWorkspaces();
      } else if (formKind === "free") {
        const ag = await api("/api/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        await loadWorkspaces();
        openTab(ag.id);
      } else {
        if (!formWs) { setFormError("Name is required."); return; }
        const ag = await api("/api/workspaces/" + formWs + "/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name: body.name, provider: body.provider, model: body.model, thinking: body.thinking }),
        });
        await refreshFleetFallback();
        openTab(ag.id);
      }
      e.target.reset();
      setNewCfg({ provider: "", model: "", thinking: "" });
      setShowForm(false);
    } catch (err) {
      setFormError(humanizeError(err.message));
    }
  }

  async function newSession() {
    if (!selectedId) return;
    try {
      await api(workspaceAPI(workspaces, freeAgents, selectedId, "/sessions/new"), { method: "POST" });
      // Optimistic: the pane clears at once (the server pointer is now
      // empty — the fresh state); loadSessions reconciles below.
      itemsAgentRef.current = selectedId || "";
      setItems(mergeAskMemory(selectedId, "", []));
      setEarlierRemaining(0);
      setSessionCurrent("");
      await refreshFleetFallback();
      await loadSessions();
    } catch (e) { toastError(e); }
  }

  async function renameSession() {
    const cur = sessions.find((s) => s.path === sessionCurrent) || { path: sessionCurrent, name: "" };
    if (!cur.path) return;
    const name = await askPrompt({ title: "Rename session", defaultValue: cur.name || "", confirmLabel: "Save" });
    if (!name) return;
    try {
      await api(workspaceAPI(workspaces, freeAgents, selectedId, "/sessions/rename"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path: cur.path, name }),
      });
      await loadSessions();
      await loadStatus();
    } catch (e) { toastError(e); }
  }

  // Compact from the Sessions view: same flow as the chat one, but feedback
  // is the per-agent statusbar segment + toast (the conversation is not open).
  async function compactAgentById(id) {
    if (!id) return;
    const ok = await askConfirm({
      title: "Compact session",
      message: "Older turns become a summary. This cannot be undone in the chat, and can take a few minutes on huge sessions.",
      confirmLabel: "Compact",
    });
    if (!ok) return;
    setCompact(id, Date.now());
    try {
      const res = await api("/api/agents/" + id + "/compact", { method: "POST" });
      setCompact(id, null);
      toast.ok(res && res.already ? "Nothing left to compact." : "Session compacted.");
    } catch (e) {
      // Keep the statusbar segment up: pi may still finish server-side.
      toastError(e);
    }
  }

  async function compactSession() {
    if (!agent) return;
    const ok = await askConfirm({
      title: "Compact session",
      message: "Older turns become a summary. This cannot be undone in the chat, and can take a few minutes on huge sessions.",
      confirmLabel: "Compact",
    });
    if (!ok) return;
    // Progress is the live compact line at the end of the chat, which
    // survives the TUI→managed panel rebuild; the finished summary folds
    // into the one-line compact card (compaction_end event or replay).
    setCompact(agent.id, Date.now());
    try {
      if (selectedId) setTermWanted((s) => { const n = new Set(s); n.delete(selectedId); return n; });
      const res = await api("/api/agents/" + agent.id + "/compact", { method: "POST" });
      setCompact(agent.id, null);
      if (res && res.already) {
        setItems((cur) => [...cur, { kind: "alert", level: "info", text: "Nothing left to compact.", ts: Date.now() }]);
        toast.ok("Nothing left to compact.");
      } else {
        toast.ok("Session compacted.");
      }
      await refreshFleetFallback();
      await loadSessions(selectedId);
      await loadStatus();
    } catch (e) {
      // Leave the live line up: the compact may still be running
      // server-side; compaction_end clears it when pi finishes.
      setItems((cur) => [...cur, { kind: "alert", level: "error", text: "Compact failed — it may still be running; check the agent output.", ts: Date.now() }]);
      toastError(e);
    }
  }

  async function runBash(command) {
    if (!agent) return;
    const itemId = "bash-" + Date.now();
    try {
      try {
        await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
      } catch { /* already running or start failed; the bash call will say */ }
      if (!panelRef.current || panelRef.current.agentId !== agent.id || (panelRef.current.sock && panelRef.current.sock.readyState !== 1)) {
        connectPanel(agent.id);
      }
      setItems((cur) => [...cur, { kind: "bash", id: itemId, command, output: "", status: "run", ts: Date.now() }]);
      setDraft("");
      clearDraft(agent.id);
      pendingPayload.current = "";
      scrollToEnd();
      const res = await api("/api/agents/" + agent.id + "/bash", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ command }),
      });
      setItems((cur) => cur.map((it) => it.kind === "bash" && it.id === itemId ? {
        ...it,
        output: res.output || it.output,
        exit: res.exitCode,
        status: res.cancelled ? "cancelled" : (res.exitCode === 0 ? "ok" : "err"),
      } : it));
    } catch (e) {
      setItems((cur) => cur.map((it) => it.kind === "bash" && it.id === itemId
        ? { ...it, status: "err", output: it.output || humanizeError(e.message || String(e)) } : it));
    }
  }

  async function abortBash() {
    if (!agent) return;
    try {
      await api("/api/agents/" + agent.id + "/bash/abort", { method: "POST" });
    } catch { /* nothing running or already done */ }
  }

  async function abortTurn() {
    automateRef.current = null;
    if (!agent) return;
    optimisticRef.current = false;
    setStreaming(false);
    streamingRef.current = false;
    setWaiting(false);
    waitingRef.current = false;
    setStatus("idle");
    setItems((cur) => cancelOpenAsks(cur).map((it) => (
      it.kind === "block" && it.cls === "user" && it.chip === "steer" && !it.dropped
        ? { ...it, dropped: true }
        : it
    )));
    try {
      await api("/api/agents/" + agent.id + "/abort", { method: "POST" });
    } catch (e) { toastError(e); }
  }

  async function replyAsk(askId, body) {
    if (!agent || !askId) return;
    const cancelled = !!body.cancelled;
    const backTo = Number.isInteger(body.backTo) ? body.backTo : null;
    const payload = { id: askId, cancelled: body.cancelled, value: body.value, confirmed: body.confirmed };
    if (backTo != null) {
      // Going back: reopen the clicked pill and answer BACK to the open
      // dialog. The extension steps back; the walk in extension_ui_request
      // answers BACK again until the target field arrives. Still waiting.
      setItems((cur) => backAsk(cur, askId, backTo));
      try {
        await api("/api/agents/" + agent.id + "/ui", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ id: askId, value: BACK }),
        });
      } catch (e) { toastError(e); }
      return;
    }
    const answer = cancelled ? "Cancelled"
      : body.confirmed === true ? "Yes"
      : body.confirmed === false ? "No"
      : (body.value || "Answered");
    setItems((cur) => answerAsk(cur, askId, answer, cancelled));
    setWaiting(false);
    waitingRef.current = false;
    // Working stays only for a turn the server confirmed (agent_start);
    // an optimistic one would never be cleared by an extension command.
    if (optimisticRef.current) {
      optimisticRef.current = false;
      setStreaming(false);
      streamingRef.current = false;
    }
    setStatus(streamingRef.current ? "streaming" : "idle");
    queueMicrotask(() => flushFollowUp());
    try {
      await api("/api/agents/" + agent.id + "/ui", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
    } catch (e) {
      // The dialog is gone (walk race, restart): reopen the step honestly.
      setItems((cur) => unanswerAsk(cur, askId));
      toastError(e);
    }
  }

  function flushFollowUp() {
    if (streamingRef.current || waitingRef.current || flushingRef.current) return;
    const next = pendingFollowUps(itemsRef.current)[0];
    const id = selectedRef.current;
    if (!next || !id) return;
    flushingRef.current = true;
    const body = { kind: "prompt", message: next.text || "" };
    if (next.queueImages && next.queueImages.length) body.images = next.queueImages;
    // Mark sent NOW: an extension command answers this POST only when its
    // whole interactive flow ends, and a still-pending bubble would be
    // flushed a second time meanwhile (duplicate /roles).
    setItems((cur) => cur.map((it) => (it.qid === next.qid ? { ...it, pending: false, chip: "prompt" } : it)));
    api("/api/agents/" + id + "/prompt", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).catch((e) => {
      toastError(e);
      setItems((cur) => cur.map((it) => (it.qid === next.qid ? { ...it, pending: true, chip: "follow_up" } : it)));
    }).finally(() => { flushingRef.current = false; });
  }

  async function sendTask(text, images, opts) {
    const payload = (typeof text === "string" ? text : draft).trim();
    const pics = images || [];
    if ((!payload && !pics.length) || !agent) return;
    const shown = opts && opts.display ? opts.display : payload;
    if (!(opts && opts.display) && !pics.length) {
      const desc = isAutomateCommand(payload);
      if (desc !== null) { setDraft(""); clearDraft(agent.id); await startAutomate(desc); return; }
    }
    const busy = streamingRef.current || waitingRef.current;
    let sendKind = kind;
    if (busy && sendKind !== "steer" && sendKind !== "follow_up") sendKind = "follow_up";
    const bash = bashLine(payload);
    if (bash && bash.refused) {
      toast.info("!! runs without sending output — use the terminal for that.");
      return;
    }
    if (bash && !pics.length) {
      await runBash(bash.command);
      return;
    }
    try {
      if (busy && sendKind === "follow_up") {
        setItems((cur) => [...cur, {
          kind: "block", cls: "user", actor: "You", chip: "follow_up",
          pending: true, qid: "q-" + Date.now(),
          text: shown, images: pics.map((p) => p.url),
          queueImages: pics.map((p) => ({ mimeType: p.mime, data: p.data })),
          ts: Date.now(),
        }]);
        setDraft("");
        clearDraft(agent.id);
        pendingPayload.current = "";
        scrollToEnd();
        return;
      }
      // Optimistic UI: the agent may take seconds to boot (huge session);
      // show the turn now and reconcile when the server answers.
      const sentTs = Date.now();
      setItems((cur) => [...cur, { kind: "block", cls: "user", actor: "You", chip: sendKind, text: shown, images: pics.map((p) => p.url), ts: sentTs }]);
      setDraft("");
      clearDraft(agent.id);
      pendingPayload.current = "";
      scrollToEnd();
      if (!busy) {
        setStreaming(true);
        streamingRef.current = true;
        optimisticRef.current = true;
      }
      try {
        await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
      } catch { /* already running or start failed; enqueue still */ }
      if (!panelRef.current || panelRef.current.agentId !== agent.id || (panelRef.current.sock && panelRef.current.sock.readyState !== 1)) {
        connectPanel(agent.id);
      }
      try {
        if (pics.length || busy || sendKind === "steer" || sendKind === "follow_up") {
          await api("/api/agents/" + agent.id + "/prompt", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
              kind: sendKind,
              message: payload,
              images: pics.map((p) => ({ mimeType: p.mime, data: p.data })),
            }),
          });
        } else {
          await api("/api/agents/" + agent.id + "/tasks", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ kind: sendKind, payload, source: "user" }),
          });
        }
      } catch (e) {
        optimisticRef.current = false;
        setStreaming(false);
        streamingRef.current = false;
        setItems((cur) => cur.map((it) => (it.kind === "block" && it.cls === "user" && it.ts === sentTs ? { ...it, text: it.text + "\n\n— not delivered: " + (e && e.message ? e.message : e) } : it)));
        throw e;
      }
      if (agent.mode === "interactive") {
        setTermWanted((s) => { const n = new Set(s); n.delete(agent.id); return n; });
      }
    } catch (e) { toastError(e); }
  }

  // /automate (ADR-0045 v2): the current agent drafts the config; the
  // editor opens pre-filled once the turn settles. No server change — the
  // turn is correlated client-side, like slashNoteTarget for ask cards.
  async function startAutomate(description) {
    if (!agent) {
      toast.info("Open an agent first, or create the automation by hand.");
      go("automations");
      return;
    }
    let desc = (description || "").trim();
    if (!desc) {
      desc = await askPrompt({ title: "Describe the automation", message: "Example: every weekday at 9, summarize what changed in this repo since yesterday.", confirmLabel: "Draft" });
      desc = (desc || "").trim();
      if (!desc) return;
    }
    automateRef.current = { agentId: agent.id, description: desc, agentName: displayAgentName(agent, selected), workspaceId: agent.workspaceId };
    await sendTask(automatePrompt(desc, { workspaceName: selected ? selected.name : "" }), [], { display: "/automate " + desc });
  }

  function finishAutomate(agentId) {
    const pending = automateRef.current;
    if (!pending || pending.agentId !== agentId) return;
    automateRef.current = null;
    const cfg = parseAutomateReply(lastAssistantText(itemsRef.current), isValidCron);
    const ws = pending.workspaceId && pending.workspaceId !== "ws_free" ? pending.workspaceId : "";
    if (cfg) {
      writeAutomationDraft({ ...cfg, workspaceId: ws, source: "automate", sourceLabel: pending.agentName });
      toast.ok("Draft ready — review it, then Create.");
    } else {
      writeAutomationDraft({ prompt: pending.description, workspaceId: ws, source: "automate", sourceLabel: pending.agentName });
      toast.info("Couldn't read a config from the reply — the editor opens with your description.");
    }
    location.hash = "#/automations/new";
  }

  function setTheme(mode) {
    persistTheme(mode);
    setThemeMode(mode);
  }

  function showTerm() {
    if (!selectedId) return;
    setTermWanted((s) => new Set(s).add(selectedId));
    if (!interactive) openInteractive(selectedId);
  }

  function showChat() {
    if (!selectedId) return;
    setTermWanted((s) => { const n = new Set(s); n.delete(selectedId); return n; });
  }

  async function openTree(mode) {
    if (!agent) { toast.info("Select an agent first."); return; }
    setTreeMode(mode || "tree");
    setTreeOpen(true);
    try {
      setTreeData(await api("/api/agents/" + agent.id + "/tree"));
    } catch (e) { toastError(e); }
  }

  async function forkFrom(entryId) {
    if (!agent || !entryId) return;
    if (treeData && leafUserId(treeData.tree, treeData.leafId) === entryId) return;
    const ok = await askConfirm({
      title: "Continue from here",
      message: "Starts a new session from this prompt. This one stays.",
      confirmLabel: "Continue",
    });
    if (!ok) return;
    try {
      const res = await api("/api/agents/" + agent.id + "/fork", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ entryId }),
      });
      setTreeOpen(false);
      if (res && res.cancelled) { toast.info("Cancelled."); return; }
      toast.ok("Continued from that prompt.");
      await refreshFleetFallback();
      await loadSessions(selectedId);
    } catch (e) { toastError(e); }
  }

  async function cloneSession() {
    if (!agent) return;
    const ok = await askConfirm({
      title: "Duplicate session",
      message: "Copy this timeline into a new session. This one stays.",
      confirmLabel: "Duplicate",
    });
    if (!ok) return;
    try {
      const res = await api("/api/agents/" + agent.id + "/clone", { method: "POST" });
      setTreeOpen(false);
      if (res && res.cancelled) { toast.info("Cancelled."); return; }
      toast.ok("Duplicated.");
      await refreshFleetFallback();
      await loadSessions(selectedId);
    } catch (e) { toastError(e); }
  }

  async function patchAgent(cfg) {
    if (!agent) return;
    const modeChanged = Object.prototype.hasOwnProperty.call(cfg, "opMode")
      && (cfg.opMode || "full") !== (agent.opMode || "full");
    const was = agent.mode;
    const dockWasOpen = !!(selectedId && termWanted.has(selectedId));
    try {
      await api("/api/agents/" + agent.id, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cfg),
      });
      await loadWorkspaces();
      if (modeChanged && selectedId && was && was !== "stopped") {
        await stopAgent(selectedId);
        if (was === "managed") await startManaged(selectedId);
        else if (was === "interactive") await openInteractive(selectedId, { dock: dockWasOpen });
      }
    } catch (e) { toastError(e); }
  }

  const onPane = route !== "workspace";
  const sessionsWsId = route === "sessions" ? sessionsRoute() : null;
  const sessionsWs = sessionsWsId ? workspaces.find((w) => w.id === sessionsWsId) : null;
  const missing = !!goneId;
  const noTabs = tabs.length === 0 && !missing;
  const hasData = (workspaces.length + freeAgents.length + terminals.length) > 0;
  const showHome = (noTabs || dashboardPinned) && hasData;

  return (
    <div id="app">
      <Sidebar
        version={version}
        workspaces={workspaces}
        selectedId={selectedId}
        onNew={() => { setFormKind("workspace"); setShowForm(true); }}
        onNewFree={() => { setFormKind("free"); setShowForm(true); }}
        onNewAgent={(id) => { setFormKind("agent"); setFormWs(id); setShowForm(true); }}
        onSelect={(id) => revealAgent(id)}
        onRun={startManaged}
        onStop={stopAgent}
        onRemove={removeWorkspace}
        onRemoveAgent={removeAgent}
        onRenameAgent={renameAgent}
        freeAgents={freeAgents}
        workingId={streaming ? selectedId : null}
        workingIds={tuiWorking}
        checklists={checklists}
        waitingId={waiting ? selectedId : null}
        termView={termView}
        terminals={terminals}
        onNewTerm={createTerminal}
        onSelectTerm={(id) => { openTermTab(id); if (parseRoute() !== "workspace") location.hash = termHash(id); }}
        onRemoveTerm={removeTerminal}
        onSessions={(id) => { location.hash = sessionsHash(id); }}
        onRenameTerm={renameTerminal}
        onGitGraph={openGitTab}
        onFileTree={openTreeTab}
        onOpenDashboard={() => setDashboardPinned(true)}
        apps={apps}
        onOpenApp={(id) => { openTab(appTabId(id)); if (parseRoute() !== "workspace") location.hash = appHash(id); }}
        onChat={(id) => {
          revealAgent(id);
          setTermWanted((s) => {
            const n = new Set(s);
            n.delete(id);
            return n;
          });
        }}
        onTerm={(id) => {
          revealAgent(id);
          setTermWanted((s) => new Set(s).add(id));
          const loc = locate(workspaces, freeAgents, id);
          if (loc && loc.agent && loc.agent.mode !== "interactive") openInteractive(id);
        }}
        userMenu={{
          host,
          version,
          themeMode,
          onTheme: setTheme,
          onNavigate: go,
          pkgUpdates,
        }}
      />

      <main id="main">
        <div id="workspace-view" className={"workspace-view" + (isTermTab(selectedId) ? " term-on" : "") + (isFileTab(selectedId) ? " file-on" : "") + (isGitTab(selectedId) ? " git-on" : "") + (isTreeTab(selectedId) ? " tree-on" : "") + (isAppTab(selectedId) ? " app-on" : "") + (showHome ? " dashboard-on" : "")} hidden={onPane}>
          <AgentTabs
            tabs={tabs}
            workspaces={workspaces}
            freeAgents={freeAgents}
            terminals={terminals}
            apps={apps}
            selectedId={selectedId}
            onSelect={(id) => openTab(id)}
            onClose={closeTab}
            onReorder={(from, to) => setTabs((t) => moveTab(t, from, to))}
          />

          <div id="empty" className="empty" hidden={!(missing || (noTabs && !hasData))}>
            <div className="empty-card">
              {missing ? (
                <>
                  <h2>{isAppTab(goneId) ? "That app is gone." : isFileTab(goneId) ? "That file is gone." : isTermTab(goneId) || (isGitTab(goneId) && goneId.startsWith("g:@t:")) || (isTreeTab(goneId) && goneId.startsWith("d:@t:")) ? "That terminal is gone." : isTreeTab(goneId) && goneId.startsWith("d:@w:") ? "That workspace is gone." : "That agent is gone."}</h2>
                  {hasData ? (
                    <p>Pick another from the sidebar.</p>
                  ) : (
                    <>
                      <p>Add a project folder to create your first agent.</p>
                      <button id="btn-new-empty" className="btn btn-primary" onClick={() => setShowForm(true)}>Add workspace</button>
                    </>
                  )}
                </>
              ) : (
                <>
                  <h2>No agents yet</h2>
                  <p>Add a project folder to create your first agent.</p>
                  <button id="btn-new-empty" className="btn btn-primary" onClick={() => setShowForm(true)}>Add workspace</button>
                </>
              )}
            </div>
          </div>

          {showHome ? <DashboardView workspaces={workspaces} freeAgents={freeAgents} workingIds={tuiWorking} waitingId={waiting ? selectedId : null} /> : null}

          {tabs.filter(isTermTab).map((id) => {
            const tid = tabTermId(id);
            const t = terminals.find((x) => x.id === tid);
            if (!t) return null;
            return (
              <TermSurface
                key={id}
                term={t}
                hidden={selectedId !== id}
                error={selectedId === id ? termError : ""}
                onOpenFile={(p) => openFileTab("term", tid, p)}
              />
            );
          })}
          <FileSurface
            owner={isFileTab(selectedId) ? parseFileTab(selectedId) : null}
            path={isFileTab(selectedId) ? (parseFileTab(selectedId) || {}).path : ""}
            onClose={() => isFileTab(selectedId) && closeTab(selectedId)}
          />
          {/* Same rule as the trees below: the loaded history, the open
              commit, the search and the branch filter belong to the tab. */}
          {tabs.filter(isGitTab).map((id) => {
            const o = gitOwners[id];
            if (!o) return null;
            return (
              <GitGraphSurface
                key={id}
                owner={o}
                hidden={selectedId !== id}
                onKey={(key) => onGitKey(id, key)}
                onClose={() => closeTab(id)}
              />
            );
          })}
          {/* One mounted tree per tab, like the terminals above: leaving a tab
              must not collapse the folders the reader opened. State lives as
              long as the tab does — closing it is what forgets. */}
          {tabs.filter(isTreeTab).map((id) => {
            const o = treeOwners[id];
            if (!o) return null;
            return (
              <FileTreeSurface
                key={id}
                owner={o}
                hidden={selectedId !== id}
                onKey={(root) => onTreeKey(id, root)}
                onOpenFile={(p) => openFileTab(o.kind, o.id, p)}
                onClose={() => closeTab(id)}
              />
            );
          })}
          {tabs.filter(isAppTab).map((id) => (
            <AppSurface
              key={id}
              appId={tabAppId(id)}
              hidden={selectedId !== id}
              manifest={apps.find((a) => a.id === tabAppId(id)) || null}
              onClose={() => closeTab(id)}
              onGoto={(g) => {
                // Apps can focus an agent's existing tab; "agent:" opens its
                // interactive TUI (replies land in the terminal itself now).
                if (g.startsWith("agent:")) openInteractive(g.slice("agent:".length));
              }}
            />
          ))}
          <ChatSurface
            hidden={noTabs || missing || termView || isTermTab(selectedId) || isFileTab(selectedId) || isGitTab(selectedId) || isTreeTab(selectedId) || isAppTab(selectedId)}
            stopped={stopped}
            items={items}
            earlierRemaining={earlierRemaining}
            onFetchEarlier={fetchEarlier}
            onToggleTool={(id) => setItems((cur) => cur.map((it) => it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it))}
            onToggleFiles={(idx) => setItems((cur) => cur.map((it, i) => i === idx && it.kind === "files" ? { ...it, expanded: !it.expanded } : it))}
            convRef={convRef}
            onScroll={() => {
              const el = convRef.current;
              if (el) nearBottom.current = stuckToBottom(el);
            }}
            statusBar={statusBar}
            compactSince={compactSince}
            onCompact={compactSession}
            onAbortBash={abortBash}
            onReplyAsk={replyAsk}
            onPrefill={(t) => {
              setDraft(t);
              queueMicrotask(() => {
                const el = document.getElementById("task-input");
                if (el) el.focus();
              });
            }}
            onQueueRemove={(qid) => setItems((cur) => dropQueued(cur, qid))}
            onQueueEdit={(qid) => setItems((cur) => startEditQueued(cur, qid))}
            onQueueSave={(qid, text) => setItems((cur) => saveEditQueued(cur, qid, text))}
            onQueueCancelEdit={(qid) => setItems((cur) => cancelEditQueued(cur, qid))}
            onOpenTab={(p) => { if (agent && p) openFileTab("agent", agent.id, p); }}
            onRun={() => selectedId && startManaged(selectedId)}
            onOpenTerm={() => selectedId && openInteractive(selectedId)}
            catalog={catalog}
            agent={agent}
            onConfig={patchAgent}
            onSlash={async (cmd) => {
              if (cmd.run === "session-tree") { openTree("tree"); return; }
              if (cmd.run === "session-fork") { openTree("fork"); return; }
              if (cmd.run === "session-clone") { cloneSession(); return; }
              if (cmd.run === "go-providers") { go("providers"); return; }
              if (cmd.run === "go-providers-new") { go("providers-new"); return; }
              if (cmd.run === "llama") { setLlamaOpen(true); return; }
              if (cmd.run === "automate") { await startAutomate(""); return; }
              if (cmd.run === "session-info") { setSessionOpen(true); return; }
              if (cmd.run === "quit") {
                if (agent && agent.mode !== "stopped") await stopAgent(selectedId);
                if (selectedId) closeTab(selectedId);
                toast.ok("Agent stopped.");
                return;
              }
              if (cmd.run === "reload") {
                if (!agent || agent.mode === "stopped") { toast.info("Start the agent first."); return; }
                const ok = await askConfirm({
                  title: "Reload",
                  message: "Restart this agent so skills and config reload. The session file stays.",
                  confirmLabel: "Reload",
                });
                if (!ok) return;
                const was = agent.mode;
                await stopAgent(selectedId);
                if (was === "interactive") await openInteractive(selectedId);
                else await startManaged(selectedId);
                toast.ok("Reloaded.");
                return;
              }
              if (cmd.run === "trust") {
                if (!agent) { toast.info("Select an agent first."); return; }
                const cwd = agent.workPath || (selected && selected.path) || "this folder";
                const ok = await askConfirm({
                  title: "Trust this folder",
                  message: cwd + " — pi will load project settings and local skills.",
                  confirmLabel: "Trust",
                });
                if (!ok) return;
                try {
                  const res = await api("/api/agents/" + agent.id + "/trust", { method: "POST" });
                  toast.ok(res && res.already ? "Already trusted." : "Folder trusted.");
                } catch (e) { toastError(e); }
                return;
              }
              if (cmd.run === "go-settings" || cmd.run === "go-scoped") {
                go("settings");
                if (cmd.run === "go-scoped") {
                  requestAnimationFrame(() => document.getElementById("scoped-models")?.scrollIntoView({ block: "center" }));
                }
                return;
              }
              if (!agent) return;
              if (cmd.run === "focus-model") { document.getElementById("agent-model")?.focus(); return; }
              if (cmd.run === "focus-thinking") { document.getElementById("agent-thinking")?.focus(); return; }
              if (cmd.run === "focus-provider") { document.getElementById("agent-provider")?.focus(); return; }
              if (cmd.run === "session-new") { await newSession(); return; }
              if (cmd.run === "session-resume") {
                document.getElementById("session-picker")?.click();
                return;
              }
              if (cmd.run === "compact") { compactSession(); return; }
              if (cmd.run === "session-name") { await renameSession(); return; }
              if (cmd.run === "share") {
                if (!selectedId) return;
                const ok = await askConfirm({
                  title: "Share session",
                  message: "Create a secret GitHub gist. Needs gh logged in. Anyone with the link can read it.",
                  confirmLabel: "Share",
                });
                if (!ok) return;
                try {
                  const res = await api("/api/agents/" + selectedId + "/share", { method: "POST" });
                  setShareLinks({ gist: res.gist || "", viewer: res.viewer || "" });
                  setShareOpen(true);
                } catch (e) { toastError(e); }
                return;
              }
              if (cmd.run === "hotkeys") { setHotkeysOpen(true); return; }
              if (cmd.run === "changelog") { setChangelogOpen(true); return; }
              if (cmd.run === "export") {
                if (!selectedId) return;
                const a = document.createElement("a");
                a.href = "/api/agents/" + selectedId + "/export";
                a.download = "session.jsonl";
                a.click();
                return;
              }
              if (cmd.run === "import") {
                if (!selectedId) return;
                const inp = document.createElement("input");
                inp.type = "file";
                inp.accept = ".jsonl";
                inp.onchange = async () => {
                  const f = inp.files && inp.files[0];
                  if (!f) return;
                  const fd = new FormData();
                  fd.append("file", f);
                  try {
                    await api("/api/agents/" + selectedId + "/import", { method: "POST", body: fd });
                    toast.ok("Session imported.");
                    await refreshFleetFallback();
                    await loadSessions();
                  } catch (e) { toastError(e); }
                };
                inp.click();
                return;
              }
            }}
            composer={{
              kind, onKind: setKind, value: draft, onChange: setDraft, onSend: sendTask,
              roleState, onRoleCommand: (cmd) => sendTask(cmd),
              slashExtra, atAgents, onAgentPage: go, pkgUpdates,
              status, streaming, waiting, onToggleDock: showTerm, onStop: () => selectedId && stopAgent(selectedId),
              tuiWorking: tuiBusy,
              onAbort: abortTurn,
              lastReply: lastAssistantText(items),
              sessionBar: selectedId ? (
                <SessionBar
                  inline
                  sessions={sessions}
                  current={sessionCurrent}
                  cost={statusBar && statusBar.cost}
                  onNew={async () => {
                    try {
                      await api(workspaceAPI(workspaces, freeAgents, selectedId, "/sessions/new"), { method: "POST" });
                      await refreshFleetFallback();
                      await loadSessions();
                    } catch (e) { toastError(e); }
                  }}
                  onResume={async (path) => {
                    try {
                      await api(workspaceAPI(workspaces, freeAgents, selectedId, "/sessions/resume"), {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ path }),
                      });
                      await refreshFleetFallback();
                      await loadSessions();
                    } catch (e) { toastError(e); }
                  }}
                  onRename={async (s) => {
                    const name = await askPrompt({
                      title: "Rename session",
                      defaultValue: s.name || "",
                      confirmLabel: "Save",
                    });
                    if (!name) return;
                    try {
                      await api(workspaceAPI(workspaces, freeAgents, selectedId, "/sessions/rename"), {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ path: s.path, name }),
                      });
                      await loadSessions();
                      await loadStatus();
                    } catch (e) { toastError(e); }
                  }}
                />
              ) : null,
            }}
          />

          {termView && !onPane && agent ? (
            agent.mode === "interactive" ? (
              <>
                <TermSurface
                  key={"agterm-" + agent.id + "-" + (termEpochs[agent.id] || 0)}
                  term={{ id: agent.id, session: "picode-" + agent.id, name: agent.name + " · TUI", cwd: agent.workPath || (selected && selected.path) }}
                  cwdKind="agent"
                  onOpenFile={(p) => openFileTab("agent", agent.id, p)}
                />
              </>
            ) : (
              <section className="term-surface" aria-label="Agent terminal">
                <p className="file-pane-msg">
                  Agent is in managed mode (chat-driven).{" "}
                  <button type="button" className="btn btn-sm" onClick={() => openInteractive(agent.id)}>Open TUI</button>
                </p>
              </section>
            )
          ) : null}
        </div>

        <PiSettings hidden={route !== "settings"} agent={agent} workspace={selected} catalog={catalog} onAgentConfig={patchAgent} />
        <Settings
          hidden={route !== "preferences"}
          themeMode={themeMode}
          onTheme={setTheme}
        />
        <System hidden={route !== "system"} version={version} system={system} />
        {route === "sessions" ? (
          <SessionsView
            wsId={sessionsWsId}
            workspace={sessionsWs}
            agents={(sessionsWs && sessionsWs.agents) || []}
            workspaces={workspaces}
            onOpenAgent={(id) => revealAgent(id)}
            onCompactAgent={compactAgentById}
          />
        ) : null}
        <Providers
          hidden={route !== "providers"}
          catalog={catalog}
          wantAdd={providersNew()}
          wantLlama={providersLlama()}
          onRefresh={async () => { try { setCatalog(await api("/api/catalog")); } catch { /* pi missing */ } }}
          onSignOut={async (provider) => {
            const ok = await askConfirm({
              title: "Sign out " + provider,
              message: "Remove saved credentials for " + provider + " on this machine.",
              confirmLabel: "Sign out",
              danger: true,
            });
            if (!ok) return;
            try {
              await api("/api/providers/" + encodeURIComponent(provider), { method: "DELETE" });
              setCatalog(await api("/api/catalog"));
              toast.ok("Signed out of " + provider + ".");
            } catch (e) { toastError(e); }
          }}
        />
        <Mcps
          hidden={route !== "mcps"}
          workspaceId={paneWs ? paneWs.id : ""}
          workspaceName={paneWs ? paneWs.name : ""}
          workspacePath={paneWs ? paneWs.path : ""}
          agentId={agent ? agent.id : ""}
          agentName={displayAgentName(agent, selected)}
          agentWorkPath={agent && agent.workPath ? agent.workPath : ""}
          agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")}
          onReload={async () => {
            if (!agent || agent.mode === "stopped") return;
            const was = agent.mode;
            await stopAgent(agent.id);
            if (was === "interactive") await openInteractive(agent.id);
            else await startManaged(agent.id);
          }}
        />
        <Packages hidden={route !== "packages"} workspaceId={paneWs ? paneWs.id : ""} workspaceName={paneWs ? paneWs.name : ""} workspacePath={paneWs ? paneWs.path : ""} agentId={agent ? agent.id : ""} agentName={displayAgentName(agent, selected)} updates={pkgUpdates} onUpdates={setPkgUpdates} />
        <Devices hidden={route !== "devices"} />
        <Automations hidden={route !== "automations"} catalog={catalog} workspaces={workspaces} freeAgents={freeAgents} system={system} />
        <TermSettingsPage hidden={route !== "termset"} terminals={terminals} />
        {route === "pins" ? <Suspense fallback={null}><PinStudio /></Suspense> : null}
      </main>

      <Palette
        open={paletteOpen}
        workspaces={workspaces}
        apps={apps}
        onClose={() => setPaletteOpen(false)}
        onRun={(a) => {
          if (a.kind === "settings" || a.kind === "preferences" || a.kind === "system" || a.kind === "providers" || a.kind === "mcps" || a.kind === "packages" || a.kind === "devices" || a.kind === "automations") { go(a.kind); return; }
          if (a.kind === "app") { openTab(appTabId(a.appId)); if (parseRoute() !== "workspace") location.hash = appHash(a.appId); return; }
          if (a.kind === "open") revealAgent(a.wsId);
          if (a.kind === "files") openTreeTab("workspace", a.wsId, a.wsName);
          if (a.kind === "run") startManaged(a.wsId);
          if (a.kind === "term") openInteractive(a.wsId);
          if (a.kind === "stop") stopAgent(a.wsId);
        }}
      />
      <ContextMenu state={ctxMenu} onClose={() => setCtxMenu(null)} themeMode={themeMode} onTheme={setTheme} />
      <Toasts />
      <CreateForm
        open={showForm}
        kind={formKind}
        workspaceName={(workspaces.find((w) => w.id === formWs) || {}).name}
        catalog={catalog}
        cfg={newCfg}
        onCfg={setNewCfg}
        error={formError}
        sessions={piSessions}
        onKind={(k) => {
          setFormKind(k);
          setFormError("");
          if (k === "session") {
            setPiSessions(null);
            api("/api/pi-sessions").then((p) => setPiSessions((p && p.sessions) || [])).catch(() => setPiSessions([]));
          }
        }}
        onAdopt={async (path) => {
          setFormError("");
          try {
            const ag = await api("/api/pi-sessions/adopt", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ path }),
            });
            await refreshFleetFallback();
            setShowForm(false);
            if (ag && ag.id) openTab(ag.id);
          } catch (err) { setFormError(humanizeError(err && err.message ? err.message : String(err))); }
        }}
        onSubmit={submitNew}
        onClose={() => { setShowForm(false); setFormError(""); }}
        busy={formBusy}
      />
      <SessionInfo
        open={sessionOpen}
        onClose={() => setSessionOpen(false)}
        bar={statusBar}
        agent={agent}
        onRename={renameSession}
        onNew={newSession}
        onCompact={compactSession}
        onTree={() => openTree("tree")}
      />
      <SessionTree
        open={treeOpen}
        mode={treeMode}
        tree={treeData}
        onClose={() => setTreeOpen(false)}
        onFork={forkFrom}
        onClone={cloneSession}
      />
      <LlamaDialog open={llamaOpen} onClose={() => setLlamaOpen(false)} onRefresh={async () => { try { setCatalog(await api("/api/catalog")); } catch { /* pi missing */ } }} />
      <ShareGist open={shareOpen} gist={shareLinks.gist} viewer={shareLinks.viewer} onClose={() => setShareOpen(false)} />
      <Hotkeys open={hotkeysOpen} onClose={() => setHotkeysOpen(false)} />
      {reconnect ? <Reconnect onReload={() => location.reload()} /> : null}
      <Changelog open={changelogOpen} onClose={() => setChangelogOpen(false)} />
      <ConfirmDialog />
      <PromptDialog />
    </div>
  );
}

function lastAssistantText(items) {
  for (let i = (items || []).length - 1; i >= 0; i--) {
    const it = items[i];
    if (it && it.kind === "block" && it.cls !== "user" && it.cls !== "thinking" && it.text) return it.text;
  }
  return "";
}

function appendDelta(cur, cls, actor, delta) {
  const last = cur[cur.length - 1];
  if (last && last.kind === "block" && last.actor === actor && last.cls === cls) {
    const next = cur.slice();
    next[next.length - 1] = { ...last, text: last.text + delta };
    return next;
  }
  return [...cur, { kind: "block", cls, actor, text: delta, ts: Date.now() }];
}
