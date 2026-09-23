import { adoptOffer, terminalLaunchAgent } from "@picode/shared/domain/cliLaunch.js";
import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError, wsURL } from "@picode/shared/client/api.js";
import { bashLine } from "@picode/shared/domain/bashLine.js";
import { OPEN_LINK_EVENT } from "./lib/externalLinks.js";
import { installOpenUrlFeed } from "./lib/openUrlFeed.js";
import { bootReads } from "./lib/bootReads.js";
import { applyTheme, persistTheme, readThemeMode } from "@picode/shared/domain/theme.js";
import { readContextMenuPrefs, modifierHeld } from "./lib/contextMenuPrefs.js";
import { openBrowserChannel } from "./lib/browserChannel.js";
import { sessionHostTab } from "./lib/sessionBrowser.js";
import { enabledKeys } from "./lib/computerChannel.js";
import { matchAction } from "./lib/appKeys.js";
import { isReloadKey, desktopReloadAction } from "./lib/desktopReload.js";
import { DESKTOP_REQUIRED, webappTabId, webappIdFromTab, webappOpenPlan, webappChromeless, watchWebapps, removedWebappTabs, webappBadge, updateWebappMeta } from "./lib/webapps.js";
import { applyTermChrome } from "@picode/shared/domain/termTheme.js";
import { closeTerm } from "./lib/terms.js";
import { termWorkspaceId, workspaceForTerminal } from "./lib/termGroups.js";
import { containerIds, optimisticFleet, putSidebarOrder, sameIds } from "./lib/sidebarOrder.js";
import { closeShellTerm } from "./components/ShellTerm.jsx";
import { summarizeArgs } from "./components/Conversation.jsx";
import { fileChangeFromTool } from "@picode/shared/domain/diff.js";
import { applyCaptureFrame, captureOnEnd, captureState, updateCapture, toolResultDetail } from "@picode/shared/domain/toolPreview.js";
import { reconcileTranscript, liveSince, startTool, transcriptGate } from "@picode/shared/domain/transcriptMerge.js";
import { eventsToItems } from "@picode/shared/domain/replay.js";
import { readCompacting, writeCompacting } from "./lib/compact.js";
import Sidebar from "./components/Sidebar.jsx";
import WindowControls from "./components/WindowControls.jsx";
import RailTabs from "./components/RailTabs.jsx";
import { IconBrandMark, IconGlobe } from "./components/Icons.jsx";
import WebTabSurface from "./components/WebTab.jsx";
import AgentTabs from "./components/AgentTabs.jsx";
import DashboardView from "./components/DashboardView.jsx";
import SessionBar from "./components/SessionBar.jsx";
import ChatSurface from "./components/ChatSurface.jsx";
import TermSurface from "./components/TermSurface.jsx";
import { initialAgentView } from "@picode/shared/domain/agentTerminal.js";
import { readChatWanted, writeChatWanted } from "./lib/openTabs.js";
import FileSurface from "./components/FileSurface.jsx";
import Inspector, { InspectorToggle, useInspectorLayout } from "./components/Inspector.jsx";
import GitGraphSurface from "./components/GitGraphSurface.jsx";
import FileTreeSurface from "./components/FileTreeSurface.jsx";
import InstructionsSurface from "./components/InstructionsSurface.jsx";
import Settings from "./components/Settings.jsx";
import BrowserPage from "./components/BrowserPage.jsx";
import ComputerPage from "./components/ComputerPage.jsx";
import AgentClis from "./components/AgentClis.jsx";
import { cliSettingsHash } from "@picode/shared/domain/cliSettings.js";
import System from "./components/System.jsx";
import Integrations from "./components/Integrations.jsx";
import Devices from "./components/Devices.jsx";
import Automations from "./components/Automations.jsx";
import Snippets from "./components/Snippets.jsx";
import Outcomes from "./components/Outcomes.jsx";
import SnipCaptureSheet from "./components/SnipCaptureSheet.jsx";
import Palette from "./components/Palette.jsx";
import SnipRunSheet from "./components/SnipRunSheet.jsx";
import ContextMenu from "./components/ContextMenu.jsx";
import SessionHandoffDialog from "./components/SessionHandoffDialog.jsx";
import ForkAgentDialog from "./components/ForkAgentDialog.jsx";
import { terminalIsIdleShellAt } from "@picode/shared/domain/terminalCli.js";
import { sessionFromTerminal, terminalHandoffSourceCli, handoffTargets } from "@picode/shared/domain/sessionHandoff.js";
import FocusEdges, { FocusLeave } from "./components/FocusEdges.jsx";
import { focusAvailable } from "@picode/shared/domain/focusMode.js";
import { useFocusMode } from "./lib/useFocusMode.js";
import { graphActions, undoFor } from "@picode/shared/domain/graphActions.js";
import { ownerBase } from "@picode/shared/domain/gitOwner.js";
import { openRepoKeys, openTreeKeys, ownerIdOf, pickTarget } from "./lib/workspacePicker.js";
import GitActionDialog from "./components/GitActionDialog.jsx";
import { paneAt, paneSelection, paneLink, focusPane } from "./lib/termActions.js";
import { planAsk, paneCapabilities } from "./lib/termMenu.js";
import { globeBindId, globeClickAction } from "./lib/globeMenu.js";
import { promptDoorFor } from "@picode/shared/domain/termPrompt.js";
import SessionTree from "./components/SessionTree.jsx";
import SessionInfo from "./components/SessionInfo.jsx";
import CreateForm from "./components/CreateForm.jsx";
import NewCliPrincipal from "./components/NewCliPrincipal.jsx";
import { agentIsPi } from "@picode/shared/domain/managedPrincipal.js";
import { ownerLetter, parseRoute, go, agentRoute, workspaceHash, termRoute, termHash, termTabId, isTermTab, tabTermId, fileRoute, fileHash, fileTabId, isFileTab, parseFileTab, gitRoute, gitHash, gitTabId, gitTabKey, isGitTab, isAgentTab, treeRoute, treeHash, treeTabId, treeTabRoot, isTreeTab, appRoute, appHash, appPath, appTabId, isAppTab, tabAppId, renamedAppHash, isWebTab, tabWebId, webHash, webRoute, boundWorkTab, instructionsRoute, instructionsHash, instructionsTabId, isInstructionsTab, instructionsTabWorkspace } from "./lib/routes.js";
import { linkOpenTarget } from "./lib/openLink.js";
import AppSurface from "./components/AppSurface.jsx";
import NativeDemoSurface from "./components/NativeDemoSurface.jsx";
import { nativeApps, nativeSurfaceFor } from "./lib/nativeApps.js";
import { normalizeManifests } from "@picode/shared/contracts/appPrimitives.js";
const PinStudio = lazy(() => import("./components/PinStudio.jsx"));
// The Canvas surface is lazy for the reason ADR-0118 gives: a reader who
// never opens the app should not carry it. It is a registry entry like any
// other, and the tab mount wraps every native surface in a Suspense.
const CanvasSurface = lazy(() => import("./components/canvas/CanvasSurface.jsx"));
import { startPresence } from "@picode/shared/client/device.js";
import { startReconnectWatch } from "@picode/shared/client/reconnect.js";
import { startFeed, subscribeFeed, feedConnected } from "@picode/shared/client/feed.js";
import { applyFleet, applyTui, applyUsage, touches } from "@picode/shared/domain/feedReducers.js";
import { applyChecklists, indexChecklists } from "@picode/shared/domain/checklist.js";
import { workspaceStatusPath } from "@picode/shared/domain/statusbar.js";
import Reconnect from "./components/Reconnect.jsx";
import { setShell } from "@picode/shared/client/shell.js";
import { dismissNotice, notify, toast, toastError } from "./lib/toast.js";
import { resolveInteractiveTerminal } from "./lib/agentTerminalView.js";
import { terminalHost, agentPaneKeys, fleetAgents, closingPaneKeys } from "./lib/terminalHost.js";
import { watchReminders } from "@picode/shared/client/reminders.js";
import { agentFinishNotice, asksOnSurface, needsYouPlan } from "@picode/shared/domain/notice.js";
import { groupTurns } from "@picode/shared/domain/turns.js";
import { needsYou } from "./lib/needsYou.js";
import { pendingFollowUps, dropQueued, startEditQueued, saveEditQueued, cancelEditQueued } from "./lib/queue.js";
import { putAsk, answerAsk, timeoutAsk, cancelOpenAsks, askJustAnswered, backAsk, walkReply, noteAsk, unanswerAsk, slashNoteTarget, BACK } from "@picode/shared/domain/askForm.js";
import { writeAskMemory, mergeAskMemory } from "./lib/askMemory.js";
import { readDraft, writeDraft, clearDraft } from "./lib/draft.js";
import { askConfirm, fmtBytes } from "./lib/confirm.js";
import { exitRequestBody } from "@picode/shared/domain/agentExit.js";
import { stuckToBottom, pinToBottom } from "@picode/shared/domain/stickScroll.js";
import { alertFromPi } from "@picode/shared/domain/piError.js";
import { mergeAssistant } from "@picode/shared/domain/assistantMsg.js";
import { isSearchTool, hitsFromResult } from "@picode/shared/domain/searchCards.js";
import ConfirmDialog from "./components/ConfirmDialog.jsx";
import PromptDialog from "./components/PromptDialog.jsx";
import { askPrompt } from "./lib/prompt.js";
import { locate, ownerOfTerminal, firstAgentId, displayAgentName, mentionAgents } from "@picode/shared/domain/tree.js";
import { leafUserId } from "./lib/sessionCards.js";

function workspaceAPI(workspaces, freeAgents, selectedId, suffix) {
  const loc = locate(workspaces, freeAgents, selectedId);
  const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : "");
  if (!id) return "";
  const q = selectedId ? "?agent=" + encodeURIComponent(selectedId) : "";
  return "/api/workspaces/" + id + suffix + q;
}
import { extraSlash } from "@picode/shared/domain/slash.js";
import { isAutomateCommand, automatePrompt, parseAutomateReply } from "./lib/automateDraft.js";
import { writeAutomationDraft } from "./lib/automationDraft.js";
import { isValidCron } from "@picode/shared/domain/cron.js";
import { readOpenTabs, writeOpenTabs, filterOpenTabs, moveTab, pickNextTab, readTermWanted, writeTermWanted, readGitOwners, writeGitOwners, readTreeOwners, writeTreeOwners, readAgentSplits, writeAgentSplits, writeAgentSplitUrls, filterAgentSplits, readWebTabUrls, writeWebTabUrls, readFileWorktrees, writeFileWorktrees } from "./lib/openTabs.js";
import { anchorFor, askedNote, ownerExists, readInspectorPrefs, runFallbackNote, writeInspectorPrefs, INSPECTOR_MIN, maxInspectorWidth } from "./lib/inspector.js";
import Hotkeys from "./components/Hotkeys.jsx";
import Changelog from "./components/Changelog.jsx";
import WhatsNew from "./components/WhatsNew.jsx";
import RELEASE_NOTES from "@picode/shared/data/whats-new.json";
import { hasUnseenRelease, readSeenVersion, shouldAutoOpen, writeSeenVersion } from "./lib/whatsNew.js";
import ShareGist from "./components/ShareGist.jsx";
import LlamaPanel from "./components/LlamaPanel.jsx";
import TermSettingsPage from "./components/TermSettingsPage.jsx";
import { createWorkspaceSchema, createWorkspaceCloneSchema, createWsAgentSchema, parseForm } from "@picode/shared/contracts/schemas.js";
import { parentDir } from "@picode/shared/domain/cloneUrl.js";
import Toasts from "./components/Toasts.jsx";
import { useMedia } from "./lib/media.js";

// Native app surfaces this shell compiled in (ADR-0109), by manifest id:
// the hidden QA demo (the server lists it with PICODE_DEMO_APP=1) and the
// Canvas, which arrives as a chunk of its own.
const NATIVE_APPS = nativeApps({ "demo-native": NativeDemoSurface, canvas: CanvasSurface });
// One frozen object, so a shell with no app subject published never makes a
// new map and never re-runs the anchor effect.
const EMPTY_SUBJECTS = Object.freeze({});

export default function App({ shellChrome = false } = {}) {
  // The sidebar rail's active view — lifted here because the shell's merged
  // top row renders the rail tabs too (the browser renders them inside the
  // sidebar). Same storage key and fallbacks the sidebar always used.
  const [sideTab, setSideTab] = useState(() => {
    try {
      const v = localStorage.getItem("picode-side-tab");
      if (v === "pins" || v === "terms" || v === "agents" || v === "apps") return v;
      return "workspaces";
    } catch { return "workspaces"; }
  });
  const selectSideTab = (next) => {
    setSideTab(next);
    try { localStorage.setItem("picode-side-tab", next); } catch { /* ignore */ }
  };
  const narrow = useMedia("(max-width: 767px)");
  const [navigationOpen, setNavigationOpen] = useState(false);
  const [workspaces, setWorkspaces] = useState([]);
  const workspacesRef = useRef([]);
  workspacesRef.current = workspaces;
  const [freeAgents, setFreeAgents] = useState([]);
  const freeAgentsRef = useRef([]);
  freeAgentsRef.current = freeAgents;
  const [selectedId, setSelectedId] = useState(() => readOpenTabs().selected);
  const [formKind, setFormKind] = useState("workspace");
  const [formWs, setFormWs] = useState("");
  const [tabs, setTabs] = useState(() => readOpenTabs().ids);
  const tabsRef = useRef([]);
  tabsRef.current = tabs;
  const [tabsReady, setTabsReady] = useState(false);
  const [fleetLoaded, setFleetLoaded] = useState(false);
  const [dashboardPinned, setDashboardPinned] = useState(false);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [semver, setSemver] = useState("");
  const [releaseBuild, setReleaseBuild] = useState(false);
  const [bootstrapped, setBootstrapped] = useState(false);
  const [webapps, setWebapps] = useState([]);
  const [webappsErr, setWebappsErr] = useState("");
  const [iconVersions, setIconVersions] = useState({});
  const [whatsNewOpen, setWhatsNewOpen] = useState(false);
  const [whatsNewMode, setWhatsNewMode] = useState("manual");
  const [whatsNewSeen, setWhatsNewSeen] = useState(readSeenVersion);
  const [host, setHost] = useState("local");
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [route, setRoute] = useState(() => parseRoute());
  useEffect(() => { setNavigationOpen(false); }, [route, selectedId]);
  const [hash, setHash] = useState(() => (typeof location !== "undefined" ? location.hash : "#/"));
  const [goneId, setGoneId] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  // Inspector rail: a per-viewer preference like the sidebar width (no hash
  // route). The anchor follows the selected tab's owner; file tabs remember
  // which view (editor or diff) the rail opened them in.
  const [inspectorPrefs, setInspectorPrefs] = useState(readInspectorPrefs);
  const [inspectorAnchor, setInspectorAnchor] = useState(null);
  const [fileViews, setFileViews] = useState({});
  // Which checkout each file tab reads through (a sibling worktree's {ref,
  // root}, or nothing for the anchor folder). Persisted beside the strip: a
  // reload must reopen the same checkout, not the same path in another tree.
  const [fileWorktrees, setFileWorktrees] = useState(readFileWorktrees);
  const [inspectorChanged, setInspectorChanged] = useState(null);
  const inspectorLayout = useInspectorLayout({ open: inspectorPrefs.open, width: inspectorPrefs.width, narrow });
  const inspectorWantOpenRef = useRef(inspectorLayout.wantOpen);
  inspectorWantOpenRef.current = inspectorLayout.wantOpen;
  const rememberInspector = useCallback((patch) => {
    setInspectorPrefs((p) => {
      const next = { ...p, ...patch };
      writeInspectorPrefs(next);
      return next;
    });
  }, []);
  // Fullscreen (focus) mode: one shell-wide, per-viewer preference like
  // the rail's, not a route. The rail's current state decides whether the
  // right edge strip exists at all once the mode starts.
  const focusOk = focusAvailable(narrow, route !== "workspace");
  const focus = useFocusMode({ railOpen: inspectorLayout.shown, available: focusOk });
  // The mode reads the rail through refs so this callback keeps one identity
  // for the keydown listener below, which unsubscribes on focusOk alone.
  const focusRef = useRef(focus);
  focusRef.current = focus;
  // Whether the rail fits at all, whatever the preference says:
  // inspectorLayout only reports "squeezed" once wantOpen is true, and the
  // toggle has to be honest before the click, not after it.
  const railFits = maxInspectorWidth(inspectorLayout.appWidth, inspectorLayout.sidebarWidth) >= INSPECTOR_MIN;
  const railFitsRef = useRef(railFits);
  railFitsRef.current = railFits;
  // The toggle: dock and undock everywhere, like any other control that
  // changes the layout. In fullscreen it also shows or hides the panel now,
  // because the mode keeps the rail off screen — flipping the preference
  // alone displayed nothing at all. Showing it docks the rail when it was
  // closed (the panel has to exist) and reveals it; hiding only takes the
  // overlay down, so the mode never undocks what the viewer set outside it.
  const toggleInspector = useCallback(() => {
    const mode = focusRef.current;
    if (mode.on) {
      if (mode.reveal === "right") { mode.show(null); return; }
      if (!railFitsRef.current) return;
      if (!inspectorWantOpenRef.current) setInspectorPrefs((p) => { const next = { ...p, open: true }; writeInspectorPrefs(next); return next; });
      mode.show("right");
      return;
    }
    setInspectorPrefs((p) => {
      const next = { ...p, open: !inspectorWantOpenRef.current };
      writeInspectorPrefs(next);
      return next;
    });
  }, []);
  const [ctxMenu, setCtxMenu] = useState(null);
  // The server's action catalog (ADR-0096): which git actions exist and what
  // risk tier each carries. Read once; with none, the graph offers no write
  // action at all rather than guessing a tier.
  const [gitCatalog, setGitCatalog] = useState(null);
  const [gitAction, setGitAction] = useState(null);
  const [gitActionTick, setGitActionTick] = useState(0);
  // What the last delivered action was, and how to put it back if it has an
  // honest inverse (ADR-0096 phase 4).
  const [gitActionDone, setGitActionDone] = useState(null);
  const [treeOpen, setTreeOpen] = useState(false);
  const [sessionOpen, setSessionOpen] = useState(false);
  const [treeMode, setTreeMode] = useState("tree");
  const [treeData, setTreeData] = useState({ tree: [], leafId: "" });
  const [catalog, setCatalog] = useState({ providers: [], thinking: [] });
  const [newCfg, setNewCfg] = useState({ provider: "", model: "", thinking: "" });
  const [showForm, setShowForm] = useState(false);
  const [cliPrincipalWs, setCliPrincipalWs] = useState(null);
  const [formError, setFormError] = useState("");
  const [formBusy, setFormBusy] = useState(false);
  const [termWanted, setTermWanted] = useState(() => new Set(readTermWanted()));
  const [chatWanted, setChatWanted] = useState(() => new Set(readChatWanted()));
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
  const historyGate = useRef(transcriptGate());
  const itemsSessionRef = useRef("");
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
  const [snipPicker, setSnipPicker] = useState([]);
  const [snipRun, setSnipRun] = useState(null);
  const [snipCapture, setSnipCapture] = useState(null); // text of a selection being saved as a snippet
  const [hotkeysOpen, setHotkeysOpen] = useState(false);
  const [changelogOpen, setChangelogOpen] = useState(false);
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
  const [clis, setClis] = useState([]);
  const [clisState, setClisState] = useState("loading"); // loading | ok | error
  const clisRef = useRef([]);
  clisRef.current = clis;
  const [termHandoff, setTermHandoff] = useState(null);
  const [forkSource, setForkSource] = useState(null);
  const terminalsRef = useRef([]);
  terminalsRef.current = terminals;
  const [termAttach, setTermAttach] = useState(null); // {id, token, text, files}: the message bar this terminal opened
  const [termFind, setTermFind] = useState(""); // the pane whose find field is open (one at a time)
  // Apps host (ADR-0036): manifests + badges from GET /api/apps.
  const [apps, setApps] = useState([]);
  const [appsLoaded, setAppsLoaded] = useState(false);
  // A graph tab is named by its repository, but only an owner can be asked for
  // it, so remember which owner opened each one (ADR-0022).
  const [gitOwners, setGitOwners] = useState(() => readGitOwners());
  const [treeOwners, setTreeOwners] = useState(() => readTreeOwners());
  // What a native app tab currently has in focus, by tab id (ADR-0109,
  // amendment 2026-09-12). The app publishes a fact about its own content —
  // "the panel in focus here is this terminal" — and the host decides what
  // that means; today it means the Inspector follows it, the way it follows
  // a graph tab through `gitOwners`. Not persisted: it is where a viewer is
  // looking right now, not a tab's identity.
  const [appSubjects, setAppSubjects] = useState(EMPTY_SUBJECTS);
  const publishSubject = useCallback((tabId, owner) => {
    setAppSubjects((cur) => {
      const had = cur[tabId] || null;
      const next = owner && owner.id ? { kind: owner.kind, id: owner.id } : null;
      if (had === next || (had && next && had.kind === next.kind && had.id === next.id)) return cur;
      if (!next && !(tabId in cur)) return cur;
      return { ...cur, [tabId]: next };
    });
  }, []);
  const treeCloseGuards = useRef(new Map());
  const registerTreeCloseGuard = useCallback((id, guard) => {
    treeCloseGuards.current.set(id, guard);
    return () => { if (treeCloseGuards.current.get(id) === guard) treeCloseGuards.current.delete(id); };
  }, []);
  const [termError, setTermError] = useState("");
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const panelRef = useRef(null);
  const pendingPayload = useRef("");

  const selectedRef = useRef(null);
  if (selectedRef.current !== selectedId) historyGate.current.invalidate();
  selectedRef.current = selectedId;

  useEffect(() => {
    setInspectorAnchor((last) => anchorFor(selectedId, { workspaces, freeAgents, terminals, gitOwners, treeOwners, appSubjects }, last));
  }, [selectedId, workspaces, freeAgents, terminals, gitOwners, treeOwners, appSubjects]);
  const fileTabInfo = isFileTab(selectedId) ? parseFileTab(selectedId) : null;
  const fileTabWT = fileTabInfo ? fileWorktrees[selectedId] || null : null;
  const fileTabOnAnchor = !!(fileTabInfo && inspectorAnchor && fileTabInfo.kind === inspectorAnchor.kind && fileTabInfo.id === inspectorAnchor.id);
  const inspectorActivePath = fileTabOnAnchor ? fileTabInfo.path : "";
  const fileTabChanged = !!(fileTabInfo && inspectorChanged && inspectorChanged.owner
    && inspectorChanged.owner.kind === fileTabInfo.kind && inspectorChanged.owner.id === fileTabInfo.id
    && (fileTabWT && fileTabWT.root
      ? (inspectorChanged.worktrees || []).some((w) => w.root === fileTabWT.root && w.paths.has(fileTabInfo.path))
      : inspectorChanged.paths.has(fileTabInfo.path)));
  // Paths this session's edit/write tools named — the Inspector's "This
  // agent" scope intersects the working tree with them.
  const touchedPaths = useMemo(() => {
    const out = [];
    for (const it of items) if (it && it.kind === "tool" && it.change && it.change.path) out.push(it.change.path);
    return out;
  }, [items]);

  const located = locate(workspaces, freeAgents, selectedId);
  const selected = located && located.workspace;
  const agent = located && located.agent;
  // A CLI agent opens as its bound terminal's tab (ADR-0160), so "the selected
  // agent" for the CLI panes is the agent that owns the selected terminal.
  const ctxAgent = agent || (isTermTab(selectedId) ? ownerOfTerminal(workspaces, freeAgents, tabTermId(selectedId)) : null);
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
  const termView = !!agent && initialAgentView(agent, chatWanted.has(agent.id) ? "chat" : (termWanted.has(agent.id) ? "terminal" : "")) === "term";
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
      if (matchAction("app.inspector.toggle", e)) {
        e.preventDefault();
        toggleInspector();
      }
      if (focusOk && matchAction("app.fullscreen.toggle", e)) {
        e.preventDefault();
        focus.toggle();
      }
      // Find is the pane's, not the app's: with the caret anywhere else the
      // chord is left alone, and the browser's own find still works there.
      if (matchAction("app.terminal.find", e)) {
        const pane = paneAt(document.activeElement);
        const field = document.querySelector(".term-find-input");
        if (pane) {
          e.preventDefault();
          setTermFind(pane.id);
          if (field) field.select();
        } else if (field && field.contains && document.activeElement === field) {
          e.preventDefault();
          field.select();
        }
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [focusOk, focus.toggle]);

  useEffect(() => {
    // A right-click inside a pane costs the user their selection before any
    // `contextmenu` listener runs: xterm drops it on mousedown when the CLI
    // has mouse reporting on (every agent TUI does), and rightClickSelectsWord
    // then replaces whatever is left with the word under the cursor. So the
    // selection is read on the mousedown capture — the earliest hook there is
    // — and used when xterm ends up with nothing. A word xterm did pick still
    // wins: it is the more precise intent.
    let held = "";
    function onCapture(e) {
      if (e.button !== 2) return;
      const pane = paneAt(e.target);
      held = pane ? paneSelection(pane.entry) : "";
      // The right button belongs to this menu. Without this, xterm forwards
      // the press to the pane as a mouse report and tmux answers it with its
      // own menu drawn inside the terminal — two menus over one click. Only
      // propagation stops: the browser still raises `contextmenu`, which is
      // where xterm selects the word under the cursor and where the menu is
      // built (or, with the bypass modifier, where the browser's own menu
      // takes over).
      if (pane) e.stopPropagation();
    }
    function onContextMenu(e) {
      if (paletteOpen) return; // avoid stacking on top of the palette
      const { bypassModifier } = readContextMenuPrefs();
      if (modifierHeld(bypassModifier, e)) return; // let the native/system menu show
      e.preventDefault();
      // The header globe is a launcher, not chrome: its menu is the work
      // browser's (Open browser / Open in new tab), not Copy / Reload.
      const globeBtn = e.target && e.target.closest && e.target.closest("[data-browser-globe]");
      if (globeBtn) {
        const selected = selectedRef.current;
        const term = isTermTab(selected) ? terminalsRef.current.find((t) => t.id === tabTermId(selected)) : null;
        const bindId = globeBindId(selected, term);
        setCtxMenu({
          x: e.clientX,
          y: e.clientY,
          target: e.target,
          globe: { tabId: bindId, splitOn: !!agentPanesRef.current[bindId] },
        });
        return;
      }
      // A terminal pane answers for itself: xterm owns the selection, and
      // what the pane is — a bare shell, a launched CLI, an agent's TUI —
      // decides which rows exist at all (lib/termMenu.js). Everything else
      // in the app gets the generic menu.
      const pane = paneAt(e.target);
      if (pane) {
        const owners = fleetAgents({ freeAgents: freeAgentsRef.current, workspaces: workspacesRef.current });
        const mapping = terminalHost(pane, owners, tabsRef.current);
        const termRecord = pane.kind === "term" ? terminalsRef.current.find((t) => t.id === pane.id) || (mapping.agent?.terminal?.id === pane.id ? mapping.agent.terminal : null) : null;
        const loc = mapping.agent ? locate(workspacesRef.current, freeAgentsRef.current, mapping.agent.id) : null;
        const splitOn = !!agentPanesRef.current[mapping.tabId];
        setCtxMenu({
          x: e.clientX,
          y: e.clientY,
          target: e.target,
          selection: paneSelection(pane.entry) || held,
          link: paneLink(pane.entry, e, pane.cwd),
          term: paneCapabilities({
            ...mapping,
            host: pane.kind,
            localScrollback: pane.entry?.term?.buffer?.active?.type !== "alternate",
            termRecord,
            agent: loc && loc.agent,
            workspace: loc && loc.workspace,
            paneCwd: pane.cwd,
            tabs: tabsRef.current,
            clis: clisRef.current,
            splitOn,
          }),
        });
        return;
      }
      // A selection inside a text field is not in the document selection,
      // so the menu reads the field itself: Copy and "Save selection as
      // snippet" must see what the reader highlighted in the composer.
      const el = e.target;
      const inField = el && typeof el.selectionStart === "number" && el.selectionEnd > el.selectionStart;
      const picked = inField ? String(el.value || "").slice(el.selectionStart, el.selectionEnd) : window.getSelection().toString();
      setCtxMenu({ x: e.clientX, y: e.clientY, selection: picked, target: e.target });
    }
    document.addEventListener("mousedown", onCapture, true);
    document.addEventListener("contextmenu", onContextMenu);
    return () => {
      document.removeEventListener("mousedown", onCapture, true);
      document.removeEventListener("contextmenu", onContextMenu);
    };
  }, [paletteOpen]);

  const loadWorkspaces = useCallback(async () => {
    const list = await api("/api/workspaces");
    setWorkspaces(list);
    try { setFreeAgents(await api("/api/agents?free=1")); }
    catch { setFreeAgents([]); }
    // The needs-you pass has to tell "nothing is waiting" from "nothing
    // has been read yet": before this flag the first render's empty fleet
    // was the seed, so the first real answer looked like an arrival and a
    // reload announced the whole backlog.
    setFleetLoaded(true);
    return list;
  }, []);

  const loadSessions = useCallback(async (wsId, opts) => {
    const ticket = historyGate.current.begin();
    const baseline = itemsRef.current;
    const current = () => historyGate.current.current(ticket) && selectedRef.current === selectedId;
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setSessions([]); setSessionCurrent(""); return; }
    try {
      const q = selectedId ? "?agent=" + encodeURIComponent(selectedId) : "";
      const data = await api("/api/workspaces/" + id + "/sessions" + q);
      if (!current()) return;
      setSessions(data.sessions || []);
      const newest = (data.sessions || [])[0] && (data.sessions || [])[0].path;
      const cur = (opts && opts.preferNewest && newest) ? newest : (data.current || "");
      if (!cur) {
        setSessionCurrent("");
        // No session file yet (extension commands only) — the thread
        // still restores from the agent's live ask-memory slot.
        let live = mergeAskMemory(selectedId, "", []);
        const snapLive = snapWaitingRef.current;
        if (snapLive.agentId === selectedId && !snapLive.waiting) live = cancelOpenAsks(live);
        itemsAgentRef.current = selectedId || "";
        setItems((items) => current() ? reconcileTranscript(live, items) : items);
        setEarlierRemaining(0);
        return;
      }
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur) + (selectedId ? "&agent=" + encodeURIComponent(selectedId) : "") + "&tail=200");
      if (!current()) return;
      const sameSession = !itemsSessionRef.current || itemsSessionRef.current === cur;
      itemsSessionRef.current = cur;
      setSessionCurrent(cur);
      const ev = t.events || [];
      earlierSkipRef.current = 0;
      setEarlierRemaining(t.remaining || 0);
      let merged = mergeAskMemory(selectedId, cur, ev.length ? eventsToItems(ev) : []);
      // A restored open stepper whose dialog died with the flow is a ghost:
      // the snapshot said nothing is waiting, so close it quietly.
      const snap = snapWaitingRef.current;
      if (snap.agentId === selectedId && !snap.waiting) merged = cancelOpenAsks(merged);
      itemsAgentRef.current = selectedId || "";
      setItems((items) => {
        if (!current()) return items;
        const reconciled = sameSession ? reconcileTranscript(merged, liveSince(items, baseline)) : merged;
        return snap.agentId === selectedId && !snap.waiting ? cancelOpenAsks(reconciled) : reconciled;
      });
      scrollToEnd();
      if ((t.bytes || 0) > 32 * 1024 * 1024) {
        if (t.compacted) {
          toast.info("Compacted — the session file stays large on disk, so this agent keeps loading slowly after a restart.");
        } else {
          toast.info("Huge session — run /compact to shrink future boots.");
        }
      }
    } catch { /* A failed read cannot clear the live conversation or session identity. */ }
  }, [selectedId, workspaces, freeAgents]);

  const fetchEarlier = useCallback(async () => {
    if (earlierLoadingRef.current || earlierSkipRef.current < 0) return;
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id || !selectedId) return;
    earlierLoadingRef.current = true;
    const ticket = historyGate.current.token();
    try {
      const cur = sessionCurrent || "";
      if (!cur) return;
      const skip = earlierSkipRef.current + 200;
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur) + (selectedId ? "&agent=" + encodeURIComponent(selectedId) : "") + "&tail=200&skip=" + skip);
      if (!historyGate.current.current(ticket) || selectedRef.current !== selectedId || itemsSessionRef.current !== cur) return;
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
    const ticket = historyGate.current.token();
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id || !selectedId) return;
    try {
      const data = await api("/api/workspaces/" + id + "/sessions?agent=" + encodeURIComponent(selectedId));
      if (!historyGate.current.current(ticket) || selectedRef.current !== selectedId) return;
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
    if (!isAgentTab(id)) { setRoleState(null); return; }
    try {
      const d = await api("/api/agents/" + id + "/role-state");
      if (selectedRef.current === id) setRoleState((d && d.state) || null);
    } catch { /* keep the last known state */ }
  }, []);
  useEffect(() => {
    setRoleState(null);
    if (isAgentTab(selectedId)) fetchRoleState();
  }, [selectedId, fetchRoleState]);

  useEffect(() => {
    if (itemsAgentRef.current !== (selectedId || "")) {
      itemsAgentRef.current = selectedId || "";
      itemsSessionRef.current = "";
      foreignTurnRef.current = false;
      itemsRef.current = [];
      setItems([]);
      setSessions([]);
      setSessionCurrent("");
    }
    loadSessions();
  }, [selectedId, workspaces.length, freeAgents.length]);
  useEffect(() => {
    if (!isAgentTab(selectedId)) { setSlashExtra([]); return; }
    api("/api/agents/" + selectedId + "/slash")
      .then((d) => setSlashExtra(extraSlash(d.skills, d.templates, d.commands, snipPicker)))
      .catch(() => setSlashExtra(extraSlash([], [], [], snipPicker)));
  }, [selectedId, agent && agent.mode, snipPicker]);
  useEffect(() => {
    function loadSnips() {
      api("/api/snips/picker").then((d) => setSnipPicker(d.snips || [])).catch(() => setSnipPicker([]));
    }
    loadSnips();
    return subscribeFeed((ev) => {
      if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("snip."))) loadSnips();
    });
  }, []);
  useEffect(() => { scrollConv(); }, [items]);
  useEffect(() => {
    // Only when the items on screen belong to this agent — on a tab switch
    // this effect fires with the previous agent's thread still in state.
    if (selectedId && itemsAgentRef.current === selectedId && itemsRef.current === items) {
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
      // Every read starts now and only the fleet is awaited (bootReads.js):
      // the Pi model catalog runs `pi --list-models` for seconds, and the
      // sidebar used to wait for it on every cold load (ADR-0179).
      const { fleet, side } = bootReads(api);
      side.clis.then((c) => { setClis(c.clis || []); setClisState("ok"); }, () => setClisState("error"));
      side.system.then(({ system: sys, version: ver }) => {
        setSystem(sys);
        setVersion(ver.version);
        setSemver(ver.semver || ver.version || "");
        setReleaseBuild(!!ver.release);
        setHost((sys.host && sys.host.name) || "local");
      }, () => { /* offline */ });
      side.catalog.then(setCatalog, () => { /* pi missing */ });
      try {
        const got = await fleet;
        const list = got.workspaces;
        const free = got.freeAgents;
        const terms = got.terminals;
        setWorkspaces(list);
        setFreeAgents(free);
        setFleetLoaded(true);
        setTerminals(terms);
        let appsOk = got.appsOk;
        let appList = [];
        try { if (appsOk) appList = normalizeManifests(got.apps); } catch { appsOk = false; }
        setApps(appList);
        setAppsLoaded(appsOk);
        const owners = readGitOwners();
        const towners = readTreeOwners();
        const ownerAlive = (o) => ownerExists(o, { workspaces: list, freeAgents: free, terminals: terms });
        const exists = (id) => {
          if (isTermTab(id)) return terms.some((t) => t.id === tabTermId(id));
          // A failed /api/apps fetch must not wipe persisted app tabs.
          if (isAppTab(id)) return appsOk ? appList.some((a) => a.id === tabAppId(id)) : true;
          if (isGitTab(id)) return ownerAlive(owners[id]);
          if (isTreeTab(id)) return ownerAlive(towners[id]);
          if (isInstructionsTab(id)) return list.some((w) => w && w.id === instructionsTabWorkspace(id));
          if (isFileTab(id)) {
            const f = parseFileTab(id);
            if (!f) return false;
            return ownerAlive(f);
          }
          return !!locate(list, free, id);
        };
        const next = filterOpenTabs(readOpenTabs(), exists);
        setTabs(next.ids);
        setFileWorktrees((cur) => {
          const keep = new Set(next.ids);
          const pruned = {};
          for (const [id, wt] of Object.entries(cur)) if (keep.has(id)) pruned[id] = wt;
          return Object.keys(pruned).length === Object.keys(cur).length ? cur : pruned;
        });
        // A split pane only lives while its host tab does: an agent that did
        // not come back takes its pane with it, and the pane's url goes too.
        // The webviews are recreated later — once the host tab is actually on
        // screen (the adoption effect below) — because `ensure` would
        // otherwise land them at its default rect, on top of the UI.
        const splits = filterAgentSplits(split0, (key) => next.ids.includes(key));
        setAgentPanes(splits.panes);
        setPaneRatios(splits.ratios);
        setPaneMax(splits.max);
        splitUrlsRef.current = splits.urls;
        writeAgentSplitUrls(splits.urls);
        const fromTerm = parseRoute() === "workspace" ? termRoute() : null;
        const fromFile = parseRoute() === "workspace" ? fileRoute() : null;
        const fromGit = parseRoute() === "workspace" ? gitRoute() : null;
        const fromTree = parseRoute() === "workspace" ? treeRoute() : null;
        const fromInstr = parseRoute() === "workspace" ? instructionsRoute() : null;
        const fromHash = parseRoute() === "workspace" ? agentRoute() : null;
        // The boot fetch outlives the redirect effect, so location.hash is
        // already canonical here; reading through it costs nothing and makes
        // that independent of which resolves first.
        const fromApp = parseRoute() === "workspace" ? appRoute(renamedAppHash() || location.hash) : null;
        if (fromApp) {
          if (appList.some((a) => a.id === fromApp) || !appsOk) openTab(appTabId(fromApp));
          else { setGoneId(appTabId(fromApp)); setSelectedId(null); }
        } else if (fromInstr) {
          if (list.some((w) => w && w.id === fromInstr)) openInstructionsTab(fromInstr);
          else { setGoneId(instructionsTabId(fromInstr)); setSelectedId(null); }
        } else if (fromTree) {
          if (ownerAlive(fromTree)) openTreeTab(fromTree.kind, fromTree.id);
          else { setGoneId(provisionalTreeId(fromTree.kind, fromTree.id)); setSelectedId(null); }
        } else if (fromGit) {
          if (ownerAlive(fromGit)) openGitTab(fromGit.kind, fromGit.id, "", fromGit.view);
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
      setBootstrapped(true);
    })();
  }, []);

  const whatsNewCurrent = semver || version;
  const whatsNewUnread = hasUnseenRelease({ release: releaseBuild, current: whatsNewCurrent, seen: whatsNewSeen, entries: RELEASE_NOTES });
  const inboxBadge = (apps.find((app) => app.id === "inbox") || {}).badge || {};
  const inboxNeedsYou = Number(inboxBadge.count) > 0 || !!inboxBadge.dot;
  // The dashboard's attention line wants the count, not the boolean: it is the
  // only place on the desktop that says how many questions are waiting.
  const inboxWaiting = Number(inboxBadge.count) || 0;

  // A fresh install sees What's New too (ADR-0063, amendment 2026-09-11). The
  // product-state gate that used to stand here — a workspace, an agent or a
  // terminal had to exist first — is gone; `blocked` below is every other
  // defer condition and all of them stay.
  useEffect(() => {
    if (!bootstrapped || !releaseBuild || !whatsNewCurrent || whatsNewOpen) return;
    const blocked = reconnect || showForm || paletteOpen || !!ctxMenu || treeOpen || sessionOpen || hotkeysOpen || shareOpen || waiting || inboxNeedsYou;
    if (shouldAutoOpen({ release: releaseBuild, current: whatsNewCurrent, seen: whatsNewSeen, entries: RELEASE_NOTES, blocked })) {
      setWhatsNewMode("auto");
      setWhatsNewOpen(true);
    }
  }, [bootstrapped, releaseBuild, whatsNewCurrent, whatsNewOpen, reconnect, showForm, paletteOpen, ctxMenu, treeOpen, sessionOpen, hotkeysOpen, shareOpen, waiting, inboxNeedsYou, whatsNewSeen]);

  function openWhatsNew() { setWhatsNewMode("manual"); setWhatsNewOpen(true); }
  function closeWhatsNew() {
    if (releaseBuild && whatsNewCurrent) {
      writeSeenVersion(whatsNewCurrent);
      setWhatsNewSeen(whatsNewCurrent);
    }
    setWhatsNewOpen(false);
  }

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
      if (ev.type === "feed.open" || ev.type === "feed.reset" || touches(ev, ["inbox", "docker"])) poll(true);
    });
    return () => { stop = true; clearInterval(t); unsub(); };
  }, []);
  // Change feed (ADR-0048): one stream per shell. Fleet events patch the
  // sidebar in place; anything the reducer cannot apply faithfully
  // refetches the fleet; (re)open and reset refetch everything.
  const fleetRef = useRef({ workspaces: [], freeAgents: [], terminals: [] });
  fleetRef.current = { workspaces, freeAgents, terminals };
  const orderTicket = useRef(0);
  const reorderSidebar = useCallback(async (kind, workspaceId, ids) => {
    const before = fleetRef.current;
    const next = optimisticFleet(before, kind, workspaceId, ids);
    if (!next || next === before) return;
    const ticket = ++orderTicket.current;
    setWorkspaces(next.workspaces);
    setFreeAgents(next.freeAgents);
    setTerminals(next.terminals);
    try {
      await putSidebarOrder(kind, workspaceId, ids);
    } catch {
      if (orderTicket.current !== ticket) return;
      const cur = fleetRef.current;
      if (!sameIds(containerIds(cur, kind, workspaceId), ids)) return;
      const restored = optimisticFleet(cur, kind, workspaceId, containerIds(before, kind, workspaceId));
      if (!restored || restored === cur) return;
      setWorkspaces(restored.workspaces);
      setFreeAgents(restored.freeAgents);
      setTerminals(restored.terminals);
    }
  }, []);
  useEffect(() => startFeed(), []);
  const webappsWatchRef = useRef(null);
  const [webappsLoaded, setWebappsLoaded] = useState(false);
  useEffect(() => {
    const watcher = watchWebapps({
      api,
      subscribe: subscribeFeed,
      onList: (list) => { setWebapps(list); setWebappsLoaded(true); },
      onError: setWebappsErr,
      onRemoved: (id) => {
        setWebapps((list) => list.filter((a) => a.id !== id));
        closeInstalledTab(webappTabId(id));
      },
    });
    webappsWatchRef.current = watcher;
    return () => { watcher.stop(); webappsWatchRef.current = null; };
  }, []);
  useEffect(() => {
    if (!webappsLoaded || !tabsReady) return;
    removedWebappTabs(tabs, webapps).forEach(closeInstalledTab);
  }, [webapps, webappsLoaded, tabs, tabsReady]);
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("cli."))) {
      api("/api/clis").then((d) => { setClis(d.clis || []); setClisState("ok"); }).catch(() => setClisState((s) => (s === "ok" ? s : "error")));
    }
  }), []);
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "feed.open" || ev.type === "feed.reset") {
      if (!ev.data || !ev.data.first) loadWorkspaces().catch(() => {});
      return;
    }
    if (!touches(ev, ["workspace", "agent", "terminal", "cli", "git"])) return;
    const next = applyFleet(fleetRef.current, ev);
    if (next === null) { loadWorkspaces().catch(() => {}); return; }
    if (next === fleetRef.current) return;
    setWorkspaces(next.workspaces);
    setFreeAgents(next.freeAgents);
    setTerminals(next.terminals);
  }), [loadWorkspaces]);
  // Needs-you: the fleet's live dialogs, announced as they arrive and
  // withdrawn as they are answered. `agent.state` (ADR-0048) carries every
  // dialog edge of every managed agent, so this reaches the agents whose
  // socket nobody has open — the gap the finish card leaves. Inbox items
  // stay out: they have a badge, a queue and a push of their own.
  const announcedAsks = useRef(null);
  useEffect(() => {
    if (!fleetLoaded) return;
    const entries = needsYou({ workspaces, freeAgents, inbox: [] });
    const plan = needsYouPlan(entries, announcedAsks.current, workspaceHash);
    announcedAsks.current = plan.keys;
    for (const key of plan.gone) dismissNotice(key);
    // Opening the conversation answers the card: a sticky notice has to
    // keep obeying suppression after it is already on screen.
    const here = new Set(asksOnSurface(entries, hash, workspaceHash));
    for (const key of here) dismissNotice(key);
    for (const n of plan.fresh) if (!here.has(n.key)) notify(n);
  }, [workspaces, freeAgents, hash, fleetLoaded]);
  // Pin reminders (ADR-0100): sticky cards that mirror the open reminder
  // items in the Inbox — raised on load and on every fire, withdrawn when
  // the item is closed or snoozed anywhere.
  useEffect(() => watchReminders({
    notify, dismiss: dismissNotice,
    pinHash: (id) => "#/pins/" + encodeURIComponent(id),
    inboxHash: "#/app/inbox",
  }), []);

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
  useEffect(() => { writeChatWanted([...chatWanted]); }, [chatWanted]);
  useEffect(() => {
    writeFileWorktrees(fileWorktrees);
  }, [fileWorktrees]);
  // ADR-0118: #/app/matrix[/<id>] is the Canvas app's old address. Replaced,
  // never pushed — the way ADR-0101/0102/0103 moved their surfaces and the
  // way #/sessions* still lands on #/clis/<cli>/sessions* — so a bookmark costs the
  // reader no extra Back step and the address bar shows the link that works
  // now. It is its own effect, declared before the one that resolves a hash
  // into a tab, so the old id never reaches the "that app is gone" branch.
  useEffect(() => {
    const next = renamedAppHash(hash);
    if (next) location.replace(next);
  }, [hash]);
  useEffect(() => {
    if (!tabsReady) return;
    if (parseRoute(hash) !== "workspace") return;
    // The effect above is replacing this hash; resolving it would flash the
    // gone tab for the one commit before `hashchange` arrives.
    if (renamedAppHash(hash)) return;
    const fromWeb = webRoute(hash);
    if (fromWeb) {
      // A web tab address selects (or restores) that tab. The id may exceed
      // this session's sequence after a reload — adopt it so the next
      // openWebTab never mints a colliding id.
      const wid = "w:" + fromWeb;
      setGoneId((g) => (g ? "" : g));
      if (selectedRef.current !== wid) {
        setTabs((t) => (t.includes(wid) ? t : [...t, wid]));
        setWebTabs((m) => (m[fromWeb] ? m : { ...m, [fromWeb]: { url: "", title: "" } }));
        webSeqRef.current = Math.max(webSeqRef.current, Number(fromWeb) || 0);
        setSelectedId(wid);
      }
      return;
    }
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
      const ok = ownerExists(fromFile, { workspaces, freeAgents, terminals });
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
      const ok = ownerExists(fromGit, { workspaces, freeAgents, terminals });
      if (ok) {
        setGoneId((g) => (g ? "" : g));
        const known = Object.entries(gitOwners).find(([, o]) => o && o.kind === fromGit.kind && o.id === fromGit.id);
        if (!known || selectedRef.current !== known[0]) openGitTab(fromGit.kind, fromGit.id, "", fromGit.view);
        else if ((known[1].view || "") !== (fromGit.view || "")) setGitOwners(m => ({ ...m, [known[0]]: { ...m[known[0]], view: fromGit.view || "" } }));
      } else {
        const gid = provisionalGitId(fromGit.kind, fromGit.id);
        setGoneId((g) => (g === gid ? g : gid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromInstr = instructionsRoute(hash);
    if (fromInstr) {
      if (workspaces.some((w) => w && w.id === fromInstr)) {
        setGoneId((g) => (g ? "" : g));
        if (selectedRef.current !== instructionsTabId(fromInstr)) openInstructionsTab(fromInstr);
      } else {
        const iid = instructionsTabId(fromInstr);
        setGoneId((g) => (g === iid ? g : iid));
        if (selectedRef.current) setSelectedId(null);
      }
      return;
    }
    const fromTree = treeRoute(hash);
    if (fromTree) {
      const ok = ownerExists(fromTree, { workspaces, freeAgents, terminals });
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
    // Web tabs are session-scoped native surfaces, but they own an address
    // now (#/web/<id>): skipping the write entirely (2026-09-13) stopped the
    // "That agent is gone" loop but left the read side re-resolving the
    // previous tab's hash and yanking the selection back (2026-09-14).
    const want = isWebTab(selectedId)
      ? webHash(tabWebId(selectedId))
      : isTermTab(selectedId)
      ? termHash(tabTermId(selectedId))
      : isAppTab(selectedId)
        ? appHash(tabAppId(selectedId), appRoute(location.hash) === tabAppId(selectedId) ? appPath(location.hash) : "")
        : gitOwner
        ? gitHash(gitOwner.kind, gitOwner.id, gitOwner.view)
        : treeOwner
          ? treeHash(treeOwner.kind, treeOwner.id)
          : isInstructionsTab(selectedId)
            ? instructionsHash(instructionsTabWorkspace(selectedId))
          : file
            ? fileHash(file.kind, file.id, file.path)
            : workspaceHash(selectedId);
    if (location.hash === want) return;
    // Only manage workspace-ish hashes. Route hashes (#/automations,
    // #/settings, #/devices, …) are deep links: replacing them here bounced
    // them to #/ once the fleet loaded — found by scripts/docs-shots.mjs,
    // which navigates cold to #/automations and never saw the view.
    if (parseRoute(location.hash) !== "workspace") return;
    if (!agentRoute(location.hash) && !termRoute(location.hash) && !fileRoute(location.hash) && !gitRoute(location.hash) && !treeRoute(location.hash) && !appRoute(location.hash) && !webRoute(location.hash) && selectedId) {
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
    if (isWebTab(id)) {
      // Session-scoped native surface: no owner to locate — the 2026-09-14
      // regression fell through to the agent lookup and returned silently,
      // so clicking a browser tab in the strip never selected it.
      setGoneId("");
      setSelectedId(id);
      setTabs((t) => (t.includes(id) ? t : [...t, id]));
      return;
    }
    if (isGitTab(id) || isTreeTab(id) || isAppTab(id) || isInstructionsTab(id)) {
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

  const [gitCatalogFailed, setGitCatalogFailed] = useState(false);
  const loadGitCatalog = useCallback(() => {
    return api("/api/git/actions")
      .then((page) => {
        const byId = {};
        for (const a of (page && page.actions) || []) if (a && a.id) byId[a.id] = a;
        setGitCatalog(byId);
        setGitCatalogFailed(false);
      })
      // No catalog, no write actions — the graph fails closed, and the menu
      // says why rather than silently offering less.
      .catch(() => setGitCatalogFailed(true));
  }, []);
  useEffect(() => { loadGitCatalog(); }, [loadGitCatalog]);

  function revealAgent(id, list) {
    openTab(id, list);
    go("workspace", id);
  }

  // Capture (snippets v2, F3): a selection in the composer, or any text the
  // context menu was opened over. A blank selection never opens the sheet.
  function openSnipCapture(text) {
    const body = String(text == null ? "" : text);
    if (!body.trim()) return;
    setSnipCapture(body);
  }

  // The git graph's context menu (ADR-0096). The graph resolves what the
  // reader pointed at; the app knows which agents are alive, so the menu is
  // composed here and rendered by the one PiCode context menu.
  // A write row carries everything the form needs: the action, its tier and
  // the target. The owner and the folder come from the tab the menu was
  // opened on, so a command is never composed for another repository.
  function openGraphAction(item, ctx) {
    if (!item || !ctx || !ctx.owner) return;
    // The ref the row points at, by name *and* kind: a tag and a branch may
    // share a name, and an undo has to put back the right one. Its hash is
    // what the graph showed — a deleted ref cannot be re-read afterwards.
    const ref = (ctx.refs || []).find((r) => r && r.name === item.target && (!item.refKind || r.kind === item.refKind)) || null;
    setGitAction({
      item, owner: ctx.owner, root: ctx.root || "", agents: ctx.agents || [],
      ref: ref ? { name: ref.name, hash: ref.hash } : null,
    });
  }

  // The token the repository had before an action was delivered. Read right
  // before delivery, never from the graph's last load: a second action sent
  // while the first is still pending would otherwise compare against a
  // baseline the first already moved, and report done at once.
  async function gitTokenBefore(owner) {
    try {
      const head = await api(ownerBase(owner) + encodeURIComponent(owner.id) + "/git/head");
      return (head && head.token) || "";
    } catch {
      return "";
    }
  }

  // "Also start an agent here" (ADR-0096 phase 4). The worktree is created by
  // a command in a terminal, so its folder appears when git gets there — not
  // when the POST returns. This waits for the graph to show it, then creates
  // the agent that lives in it. A worktree that never appears is said so,
  // rather than an agent pointed at a folder that is not there.
  async function startAgentInWorktree(owner, slug, agentName) {
    const wsId = owner.kind === "workspace"
      ? owner.id
      : ((locate(workspaces, freeAgents, owner.id) || {}).workspace || {}).id || "";
    if (!wsId) {
      toast.info("The worktree command was sent. Start an agent from its row once it exists.");
      return;
    }
    const base = ownerBase(owner) + encodeURIComponent(owner.id) + "/git";
    for (let i = 0; i < 15; i += 1) {
      await new Promise((r) => setTimeout(r, 2000));
      let found = "";
      try {
        const g = await api(base + "?limit=1");
        found = ((g && g.worktrees) || []).map((w) => w.path || "").find((p) => p.endsWith("/.worktrees/" + slug)) || "";
      } catch {
        /* a poll that fails is not an answer; the deadline still applies */
      }
      if (!found) continue;
      try {
        const agent = await api("/api/workspaces/" + encodeURIComponent(wsId) + "/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name: agentName || slug, workPath: found }),
        });
        await loadWorkspaces();
        toast.ok("Agent " + (agent.name || slug) + " lives in " + slug + " now.");
        if (agent.id) revealAgent(agent.id);
      } catch (e) {
        toastError(e);
      }
      return;
    }
    toast.info("The worktree has not appeared yet. Start an agent from its row when it does.");
  }

  function openGraphMenu({ x, y, target, graph, owner, root }) {
    // A catalog that never arrived is asked for again on every menu, so a
    // transient failure at boot does not leave the graph read-only for the
    // rest of the session.
    if (!gitCatalog) loadGitCatalog();
    const menu = graphActions(target, graph, { workspaces, freeAgents, terminals, catalog: gitCatalog });
    if (!menu.title) return;
    if (!gitCatalog && gitCatalogFailed) menu.state = [menu.state, "Git actions are unavailable: the server did not answer the action catalog."].filter(Boolean).join(" ");
    const here = (menu.occupants || []).filter((o) => o.running !== false);
    setCtxMenu({ x, y, graph: menu, graphCtx: { owner, root, agents: here, head: graph.head, refs: graph.refs } });
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

  function openFileTab(kind, ownerId, path, view = "file", wt = null) {
    setDashboardPinned(false);
    if (!ownerId || !path) return;
    const id = fileTabId(kind, ownerId, path);
    setFileViews((v) => (v[id] === view ? v : { ...v, [id]: view }));
    // The checkout is set-or-cleared on every open: a tab id is owner+path,
    // so reopening the same path from another checkout must retarget the
    // tab, never inherit a stale one.
    setFileWorktrees((v) => {
      const cur = v[id] || null;
      const same = (!cur && !wt) || (!!cur && !!wt && cur.ref === wt.ref && cur.root === wt.root);
      if (same) return v;
      const next = { ...v };
      if (wt && wt.ref && wt.root) next[id] = { ref: wt.ref, root: wt.root, branch: wt.branch || "" };
      else delete next[id];
      return next;
    });
    setGoneId("");
    setSelectedId(id);
    setTabs((t) => (t.includes(id) ? t : [...t, id]));
  }

  // The Inspector's PR tab pre-fills a gh command in a terminal the human
  // submits themselves (ADR-0078): the owner's own terminal when it is one,
  // otherwise a new terminal born in the anchored folder.
  // Stage 3 (ADR-0078): the Inspector asks a running agent through the
  // channel that already carries prompts to it. The server picks the door
  // (task queue for a managed agent, receiver or paste for a TUI) and says
  // which, so the note is honest about when the agent acts. The view is not
  // retargeted: the agent's own tab shows the turn.
  // With `quiet` the failure is thrown to the caller, which shows it in its
  // own form, instead of also being toasted — one report per failure.
  async function askAgentGit(who, text, root, action, verb, { quiet = false } = {}) {
    try {
      // A pi running as an Agent CLI terminal is asked through its receiver
      // (ADR-0089 amendment); an agent through its own channel (ADR-0078).
      const base = who.kind === "terminal" ? "/api/terminals/" : "/api/agents/";
      const res = await api(base + encodeURIComponent(who.id) + "/ask", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text, root }),
      });
      // The graph composes thirty-odd actions the rail's own verb table never
      // knew; when the server named one, that name is what the toast says.
      toast.info(askedNote(who.name, action, res, verb));
    } catch (e) {
      if (quiet) throw e;
      toastError(e);
    }
  }

  async function typeIntoTerminal(owner, root, command, { run = false, quiet = false, prepareNote = true } = {}) {
    if (!owner || !command) return;
    const post = (tid, resource, body) => api("/api/terminals/" + encodeURIComponent(tid) + "/" + resource, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
    });
    // Stage 2 (ADR-0078): with the run preference on, the server presses
    // Enter only when its interlock finds nobody else writing the repository;
    // a busy answer becomes a prepared command plus a note saying why.
    // The outcome is returned — "ran", or "prepared" when the command waits
    // for Enter — so a caller watching for its effect can say which.
    const deliver = async (tid) => {
      if (run) {
        try {
          await post(tid, "run", { text: command, root });
          return "ran";
        } catch (err) {
          // Only the target terminal's own refusals ("This terminal moved
          // to…", "This terminal is running…") send the command elsewhere;
          // a busy repository is prepared right here, with the reason.
          const msg = (err && err.message) || "";
          if (/^(This terminal (moved to|is running)|This is an Agent CLI)/i.test(msg)) throw err;
          if (prepareNote) toast.info(runFallbackNote(humanizeError(msg)));
        }
      }
      await post(tid, "type", { text: command, root });
      return "prepared";
    };
    const create = async () => {
      const loc = owner.kind === "agent" ? locate(workspaces, freeAgents, owner.id) : null;
      const wsId = owner.kind === "workspace" ? owner.id : (loc && loc.workspace ? loc.workspace.id : "");
      const page = await api("/api/terminals", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: command.trim().split(/\s+/)[0] || "git", cwd: root || "", workspaceId: wsId }),
      });
      setTerminals((cur) => (cur.some((x) => x.id === page.id) ? cur : [...cur, page]));
      await openTermTab(page.id);
      // A shell born this instant has not drawn its prompt yet; keystrokes
      // that arrive before it echo twice. Give it a beat before typing.
      await new Promise((resolve) => setTimeout(resolve, 900));
      return page.id;
    };
    try {
      let tid = owner.kind === "term" ? owner.id : "";
      if (!tid) {
        // Reuse a plain, idle shell already sitting in that folder; never a
        // terminal hosting a CLI (Claude Code, Codex…) whose TUI would eat
        // the keystrokes, and never one that is working.
        const agentTerms = new Set([...workspaces.flatMap((w) => w.agents || []), ...freeAgents].map((a) => a && a.terminalId).filter(Boolean));
        const idle = terminals.find((t) => terminalIsIdleShellAt(t, root, agentTerms));
        if (idle) tid = idle.id;
      }
      if (!tid) return await deliver(await create());
      await openTermTab(tid);
      try {
        return await deliver(tid);
      } catch (err) {
        // The chosen terminal moved away, holds a foreground program or
        // turned out to be an Agent CLI: a fresh terminal in the folder
        // takes the command instead.
        if (!/^(This terminal (moved to|is running)|This is an Agent CLI)/i.test((err && err.message) || "")) throw err;
        return await deliver(await create());
      }
    } catch (err) {
      // A caller that announces the outcome (the graph's pending banner)
      // must hear about the failure, or it would say "Sent" over a 409.
      if (quiet) throw err;
      toastError(err);
    }
  }

  function provisionalGitId(kind, ownerId) {
    return gitTabId("@" + ownerLetter(kind) + ":" + ownerId);
  }

  // The repository is unknown until the server answers, so a graph tab opens
  // under a provisional id and onGitKey renames it to g:<key> — which is also
  // where two owners of the same repo collapse onto one tab (ADR-0022).
  function openGitTab(kind, ownerId, ownerName, view = "") {
    setDashboardPinned(false);
    if (!ownerId) return;
    const known = Object.entries(gitOwners).find(([, o]) => o && o.kind === kind && o.id === ownerId);
    const id = known ? known[0] : provisionalGitId(kind, ownerId);
    setGitOwners((m) => {
      const next = { ...m, [id]: { kind, id: ownerId, name: ownerName || "", view } };
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

  // The graph toolbar's workspace picker (ADR-0022 amendment): the reader
  // chooses the owner the history is read through, from inside the tab. The
  // repository is still the tab's identity, so a pick resolves where it lands
  // before anything moves — the key comes from the picked workspace's own
  // /git/head, never from a path in a URL (ADR-0022) — and then one of three
  // moves happens: the owner changes in place (sibling worktrees of this
  // repository), this tab becomes the other repository's (`g:<key>`, the same
  // rename the first load performs), or the tab that already holds that
  // repository is selected and takes the picked owner, because an explicit
  // pick wins over whoever happened to open it first. One tab per repository
  // holds in all three. A pick that cannot resolve — the folder stopped being
  // a repository, the daemon said no — leaves the tab exactly as it was and
  // says so: retargeting silently would show another repository's history
  // under the old tab's name.
  async function switchGraphWorkspace(tabId, wsId) {
    if (!isGitTab(tabId)) return;
    const ws = workspaces.find((w) => w && w.id === wsId);
    if (!ws) return;
    const owner = { kind: "workspace", id: wsId, name: ws.name || "", view: gitOwners[tabId]?.view || "" };
    // The resolution and its two failure rows (a request that throws, an answer
    // that names no repository) live in the pure module, where they are tests.
    const { key, action, error } = await pickTarget({
      tabKey: gitTabKey(tabId),
      openKeys: openRepoKeys(tabs),
      readTarget: async () => {
        const head = await api(ownerBase(owner) + encodeURIComponent(wsId) + "/git/head");
        return { key: (head && head.key) || "" };
      },
    });
    if (error) {
      toastError(error);
      return;
    }
    if (action === "none") return;
    const target = gitTabId(key);
    setGitOwners((m) => {
      const next = { ...m };
      if (action === "rename") delete next[tabId];
      next[target] = owner;
      writeGitOwners(next);
      return next;
    });
    if (action === "rename") {
      setTabs((t) => {
        const swapped = t.map((x) => (x === tabId ? target : x));
        return swapped.filter((x, i) => swapped.indexOf(x) === i);
      });
    }
    setGoneId("");
    setSelectedId(target);
  }

  // The file tree's folder line is a workspace picker (ADR-0030 amendment): the
  // reader chooses the folder this tree reads, from inside the tab. The tab is
  // still the canonical folder (`d:<root>`), so a pick resolves the target
  // first — /browse answers the canonical root, never a path from a URL — and
  // pickAction turns it into the same three moves the graph's pick makes: the
  // same folder swaps the owner in place, another folder renames this tab to
  // it (or hands the pick to the tab that already holds it, an explicit pick
  // winning over whoever opened it first). One tab per folder holds in all
  // three. A pick that cannot resolve leaves the tab exactly as it was.
  async function switchTreeWorkspace(tabId, wsId) {
    if (!isTreeTab(tabId)) return;
    const ws = workspaces.find((w) => w && w.id === wsId);
    if (!ws) return;
    const owner = { kind: "workspace", id: wsId, name: ws.name || "" };
    const { key, action, error } = await pickTarget({
      tabKey: treeTabRoot(tabId),
      openKeys: openTreeKeys(tabs),
      readTarget: async () => {
        const page = await api("/api/workspaces/" + encodeURIComponent(wsId) + "/browse");
        return { key: (page && page.root) || "" };
      },
    });
    if (error) {
      toastError(error);
      return;
    }
    if (action === "none") return;
    const target = treeTabId(key);
    setTreeOwners((m) => {
      const next = { ...m };
      if (action === "rename") delete next[tabId];
      next[target] = owner;
      writeTreeOwners(next);
      return next;
    });
    if (action === "rename") {
      setTabs((t) => {
        const swapped = t.map((x) => (x === tabId ? target : x));
        return swapped.filter((x, i) => swapped.indexOf(x) === i);
      });
    }
    setGoneId("");
    setSelectedId(target);
  }

  function provisionalTreeId(kind, ownerId) {
    return treeTabId("@" + ownerLetter(kind) + ":" + ownerId);
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

  // Instructions (docs/architecture/cli-instructions.md): one tab per
  // workspace, keyed by the workspace's own id — nothing to resolve first.
  function openInstructionsTab(wsId) {
    setDashboardPinned(false);
    if (!wsId) return;
    const id = instructionsTabId(wsId);
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

  // Work browser tabs (Phase 3 slice 1): "w:<id>" editor tabs backed by a
  // native WebView2 child. Desktop-shell only — the component shows a notice
  // elsewhere. Meta (url/title) is mirrored up for the tab strip.
  // A shell without a webview restores what its tabs opened; the desktop shell
  // reads the address back from WebView2 instead (lib/openTabs.js).
  const [webTabs, setWebTabs] = useState(readWebTabUrls);
  useEffect(() => { writeWebTabUrls(webTabs); }, [webTabs]);
  // The Ask prompts the shell is holding (slice 3, Browser permissions): one
  // entry per held request, shown in its own tab and answered through
  // btab_permission_answer. The shell times an unanswered prompt out and
  // reports the outcome with the same ask id, which drops the bar.
  const [permissionAsks, setPermissionAsks] = useState([]);
  // Agent split (ADR-0135): an agent tab can host a work-browser pane beside
  // it. agentId (or "term:<id>") -> web id; ratios/max are the layout, and
  // both the layout and the last url per pane are persisted, so a shell
  // relaunch brings the panes back at the same page (owner directive
  // 2026-09-14) instead of asking the user to reopen every split by hand.
  const split0 = useMemo(readAgentSplits, []);
  const webSeqRef = useRef(
    Math.max(0, ...Object.values(split0.panes).map((n) => Number(n)).filter(Number.isFinite)),
  );
  const [agentPanes, setAgentPanes] = useState(split0.panes);
  const [paneRatios, setPaneRatios] = useState(split0.ratios);
  const [paneMax, setPaneMax] = useState(split0.max);
  const splitUrlsRef = useRef(split0.urls);
  const agentPanesRef = useRef(agentPanes);
  agentPanesRef.current = agentPanes;
  const ensureSessionRef = useRef(null);
  useEffect(() => {
    writeAgentSplits({ panes: agentPanes, ratios: paneRatios, max: paneMax });
  }, [agentPanes, paneRatios, paneMax]);
  // A relaunch kills every webview. The first time a restored pane's host tab
  // is on screen, recreate it at its last url — the surface's own bounds
  // effect has already stored the rect (child effects run before the
  // parent's), so the webview lands where the pane is.
  const splitAdoptedRef = useRef(new Set());
  useEffect(() => {
    if (!bootstrapped) return;
    const id = agentPanes[selectedId];
    if (!id || splitAdoptedRef.current.has(id)) return;
    splitAdoptedRef.current.add(id);
    const url = splitUrlsRef.current[id] || "";
    if (url) window.__TAURI__?.core.invoke("btab_navigate", { id, url }).catch(() => {});
  }, [bootstrapped, selectedId, agentPanes]);
  function openAgentSplit(agentId) {
    if (!agentId) return "";
    const existing = agentPanesRef.current[agentId];
    if (existing) return existing;
    webSeqRef.current += 1;
    const id = String(webSeqRef.current);
    agentPanesRef.current = { ...agentPanesRef.current, [agentId]: id };
    setWebTabs((m) => ({ ...m, [id]: { url: "", title: "" } }));
    setAgentPanes((p) => ({ ...p, [agentId]: id }));
    return id;
  }
  // ADR-0172: a session drive opens the split beside that principal's tab
  // and selects it, so the human sees the page the agent is driving.
  function ensureSession(cmd) {
    const agents = fleetAgents({ freeAgents: freeAgentsRef.current, workspaces: workspacesRef.current });
    const ag = agents.find((a) => a && a.id === cmd?.agent);
    const host = sessionHostTab({
      agentId: cmd?.agent || "",
      termId: cmd?.term || "",
      openTabs: tabsRef.current,
      agentTerminalId: ag?.terminalId || "",
    });
    if (!host) return { error: "this caller has no session tab to bind a browser to" };
    if (!tabsRef.current.includes(host)) {
      if (isTermTab(host)) openTermTab(tabTermId(host));
      else openTab(host);
    } else if (selectedRef.current !== host) {
      setSelectedId(host);
    }
    const id = openAgentSplit(host);
    return id ? { id } : { error: "could not open the browser beside this session" };
  }
  ensureSessionRef.current = ensureSession;
  function closeAgentSplit(key) {
    const wid = agentPanesRef.current[key];
    if (wid) {
      window.__TAURI__?.core.invoke("btab_close", { id: wid }).catch(() => {});
      setPermissionAsks((cur) => cur.filter((a) => a.tab !== wid));
    }
    const next = { ...agentPanesRef.current };
    delete next[key];
    agentPanesRef.current = next;
    setAgentPanes(({ [key]: _gone, ...rest }) => rest);
  }
  function openWebTab(url) {
    if (parseRoute(location.hash) !== "workspace") location.hash = "#/";
    webSeqRef.current += 1;
    const id = String(webSeqRef.current);
    setTabs((t) => [...t, "w:" + id]);
    setSelectedId("w:" + id);
    setWebTabs((m) => ({ ...m, [id]: { url: url || "", title: "" } }));
    if (url) window.__TAURI__?.core.invoke("btab_navigate", { id, url }).catch(() => {});
    return "w:" + id;
  }
  const openWebTabRef = useRef(openWebTab);
  openWebTabRef.current = openWebTab;

  function onGlobeClick(e) {
    const selected = selectedId;
    const term = isTermTab(selected) ? terminals.find((t) => t.id === tabTermId(selected)) : null;
    const bindId = globeBindId(selected, term);
    const action = globeClickAction({
      shiftKey: !!e.shiftKey,
      bindId,
      splitOn: !!agentPanesRef.current[bindId],
    });
    if (action === "new-tab") openWebTab("");
    else if (action === "split") openAgentSplit(bindId);
  }

  function closeInstalledTab(tab) {
    if (!webappIdFromTab(tab)) return;
    const id = tab.slice(2);
    const next = tabsRef.current.filter((t) => t !== tab);
    tabsRef.current = next;
    setTabs((tabs) => tabs.filter((t) => t !== tab));
    setSelectedId((selected) => selected === tab ? next.at(-1) || null : selected);
    setWebTabs(({ [id]: _gone, ...rest }) => rest);
    setPermissionAsks((asks) => asks.filter((ask) => ask.tab !== id));
    window.__TAURI__?.core.invoke("btab_close", { id }).catch((error) => toastError(error));
  }

  function savedWebapp(app) {
    setWebapps((list) => [...list.filter((a) => a.id !== app.id), app]);
    webappsWatchRef.current?.refresh();
  }

  async function removeWebapp(app) {
    // The partition folder lives on the shell: best effort to remove it
    // with the rest (a browser-UI removal cannot — an orphan folder is
    // the accepted leftover, reclaimed on re-install).
    try {
      await window.__TAURI__?.core.invoke("btab_clear_app_data", { id: "app-" + app.id });
    } catch { /* row removal proceeds regardless */ }
    await api("/api/webapps/" + encodeURIComponent(app.id), { method: "DELETE" });
    setWebapps((list) => list.filter((a) => a.id !== app.id));
    closeInstalledTab(webappTabId(app.id));
    webappsWatchRef.current?.refresh();
  }

  async function refreshWebapp(app) {
    const updated = await api("/api/webapps/" + encodeURIComponent(app.id) + "/refresh", { method: "POST" });
    setWebapps((list) => list.map((a) => (a.id === app.id ? updated : a)));
    setIconVersions((v) => ({ ...v, [app.id]: Date.now() }));
    webappsWatchRef.current?.refresh();
    toast.ok(app.name + " updated");
  }

  async function clearWebappData(app) {
    await window.__TAURI__?.core.invoke("btab_clear_app_data", { id: "app-" + app.id });
    setIconVersions((v) => ({ ...v, [app.id]: Date.now() }));
    toast.ok(app.name + "'s data cleared — sign in again next time");
  }

  function openWebapp(app) {
    const plan = webappOpenPlan(app, tabsRef.current, shellChrome && !!window.__TAURI__);
    if (plan.action === "invalid") return;
    if (plan.action === "desktop-required") {
      toast.info(DESKTOP_REQUIRED);
      return;
    }
    if (plan.action === "focus") {
      setSelectedId(plan.tab);
      if (parseRoute(location.hash) !== "workspace") location.hash = "#/";
      return;
    }
    if (parseRoute(location.hash) !== "workspace") location.hash = "#/";
    tabsRef.current = [...tabsRef.current, plan.tab];
    setTabs((t) => (t.includes(plan.tab) ? t : [...t, plan.tab]));
    setSelectedId(plan.tab);
    setWebTabs((m) => ({ ...m, [plan.id]: { url: plan.url, title: "" } }));
    window.__TAURI__?.core.invoke("btab_navigate", { id: plan.id, url: plan.url }).catch(toastError);
  }

  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const un = window.__TAURI__.event.listen("btab://new", (e) => {
      const url = typeof e.payload === "string" ? e.payload : "";
      if (!url) { openWebTabRef.current(""); return; }
      let host = "";
      try { host = new URL(url).hostname.toLowerCase(); } catch { /* fall through to the app */ }
      const isLocal = ["localhost", "127.0.0.1", "::1"].includes(host);
      // Open destinations (slice 3): the pref decides whether a new-tab
      // request adopts as a tab or hands off to the system default browser.
      // One fresh read per request — a `target=_blank` is rare and the pref
      // must be current.
      fetch("/api/browser/prefs")
        .then((r) => r.json())
        .then((p) => {
          const dest = isLocal ? (p.localOpenDest || "app") : (p.webOpenDest || "app");
          if (dest === "external") return window.__TAURI__.core.invoke("btab_open_external", { url });
          openWebTabRef.current(url);
        })
        .catch(() => openWebTabRef.current(url));
    });
    return () => un.then((f) => f());
  }, [shellChrome]);

  // Downloads (slice 3.3d): the shell reports each download as it starts and
  // again when it lands or breaks. The row lives in the daemon's store, so
  // the report goes through the API (the shell never talks to the store) and
  // Settings ▸ Browser reads it back.
  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const un = window.__TAURI__.event.listen("btab://download", (e) => {
      const d = e.payload ?? {};
      const url = d.status === "started" ? "/api/browser/downloads" : "/api/browser/downloads/status";
      fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(d) }).catch(() => {});
    });
    return () => un.then((f) => f());
  }, [shellChrome]);

  // Site permissions: the shell reports every decision it made (policy,
  // platform default, or an answered Ask prompt). The row lives in the
  // daemon's store, so the report goes through the API and Settings ▸
  // Browser reads it back. An answered prompt also carries its ask id,
  // which dismisses the bar even when the shell's own timeout was the one
  // that answered.
  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const un = window.__TAURI__.event.listen("btab://permission", (e) => {
      const p = e.payload ?? {};
      if (p.ask) setPermissionAsks((cur) => cur.filter((x) => x.id !== p.ask));
      if (!p.origin || !p.kind) return;
      fetch("/api/browser/permissions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(p) }).catch(() => {});
    });
    return () => un.then((f) => f());
  }, [shellChrome]);

  // The Ask prompt's other half: the shell holds a site's request and names
  // it here, the tab renders the bar, and the answer command completes the
  // deferral. "Always allow" is `remember` — the shell writes the site's
  // standing, which lands back in the store through the report above.
  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const un = window.__TAURI__.event.listen("btab://permission-ask", (e) => {
      const a = e.payload ?? {};
      if (!a.id || !a.tab) return;
      setPermissionAsks((cur) => [
        ...cur.filter((x) => x.id !== a.id),
        { id: a.id, tab: String(a.tab), origin: String(a.origin || ""), kind: String(a.kind || "unknown") },
      ]);
    });
    return () => un.then((f) => f());
  }, [shellChrome]);

  const answerPermission = (ask, state, remember) => {
    setPermissionAsks((cur) => cur.filter((x) => x.id !== ask.id));
    window.__TAURI__?.core.invoke("btab_permission_answer", { id: ask.id, state, remember })
      .catch((e) => toast("Answering the permission failed: " + (e?.message || e)));
  };

  // The daemon's work-browser command channel (ADR-0132): one stream per shell
  // window. A command runs against the work-browser tab on screen; the ref
  // keeps the listener on the current tab without reopening the stream.
  const selectedTabRef = useRef(selectedId);
  selectedTabRef.current = selectedId;
  useEffect(() => {
    if (!shellChrome) return undefined;
    return openBrowserChannel(
      () => boundWorkTab(selectedTabRef.current, agentPanesRef.current),
      (cmd) => ensureSessionRef.current?.(cmd),
    );
  }, [shellChrome]);
  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const invoke = window.__TAURI__.core.invoke;
    const onKey = (e) => {
      if (!isReloadKey(e) || e.defaultPrevented) return;
      const action = desktopReloadAction({
        inTerminal: !!paneAt(document.activeElement),
        selectedTab: selectedTabRef.current,
        panes: agentPanesRef.current,
        onPane: parseRoute() !== "workspace",
      });
      if (!action) return;
      e.preventDefault();
      if (action.kind === "page") invoke("btab_reload", { id: action.id }).catch(() => {});
      else location.reload();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [shellChrome]);

  // ADR-0148: the shell keeps a mirror of the computer grants (one bit per
  // principal) and refuses on its own copy too. Pushed at load and whenever a
  // setting changes; the daemon stays the decision point.
  useEffect(() => {
    if (!shellChrome || !window.__TAURI__) return undefined;
    const push = async () => {
      try {
        const r = await fetch("/api/computer/policies");
        if (!r.ok) return;
        const data = await r.json();
        await window.__TAURI__.core.invoke("computer_set_grants", { keys: enabledKeys(data.policies) });
      } catch {
        /* the next setting change retries */
      }
    };
    push();
    return subscribeFeed((ev) => {
      if (ev.type === "setting.updated") push();
    });
  }, [shellChrome]);

  // Select a surviving tab after one left the strip (closeTab, removeAgent,
  // removeWorkspace): terminals go through openTermTab so the shell pane
  // comes back, agent tabs prepare their surface like a click would, and
  // null means none survived — the dashboard takes over (showHome).
  function adoptTab(id) {
    if (!id) { setSelectedId(null); return; }
    if (isTermTab(id)) { openTermTab(tabTermId(id)); return; }
    setSelectedId(id);
    const loc = locate(workspaces, freeAgents, id);
    if (loc && loc.agent) prepareSurface(loc.agent);
  }

  async function closeTab(id) {
    const guard = treeCloseGuards.current.get(id);
    if (guard && !await guard()) return;
    setFileViews((v) => {
      if (!(id in v)) return v;
      const next = { ...v };
      delete next[id];
      return next;
    });
    setFileWorktrees((v) => {
      if (!(id in v)) return v;
      const next = { ...v };
      delete next[id];
      return next;
    });
    if (isWebTab(id)) {
      const wid = tabWebId(id);
      window.__TAURI__?.core.invoke("btab_close", { id: wid }).catch(() => {});
      setPermissionAsks((cur) => cur.filter((a) => a.tab !== wid));
      setWebTabs(({ [wid]: _gone, ...rest }) => rest);
    }
    for (const key of closingPaneKeys(id, fleetAgents({ workspaces, freeAgents }), tabs)) closeShellTerm(key);
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
    // The neighbour is picked from the strip as the reader sees it — before
    // the filter below lands — so the answer anchors on `id`'s position.
    const next = selectedRef.current === id ? pickNextTab(tabsRef.current, [id]) : null;
    setTabs((t) => t.filter((x) => x !== id));
    setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
    if (ws && ws.agent) closeAgentShell(ws.agent);
    if (panelRef.current && ws && ws.agent && panelRef.current.agentId === ws.agent.id) closePanel();
    if (selectedRef.current === id) adoptTab(next);
  }

  function closePanel() {
    const p = panelRef.current;
    if (!p) return;
    p.stopped = true;
    try { p.sock.close(); } catch { /* ignore */ }
    panelRef.current = null;
  }

  function closeAgentShell(ag) {
    for (const id of agentPaneKeys(ag)) closeShellTerm(id);
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

  // A turn that finished while the user was looking somewhere else is the
  // one notice worth interrupting for (study:
  // docs/benchmarks/2026-09-07-superset-notifications.md). Everything the
  // card shows — how long the turn took, which files it touched, what the
  // agent said last — is already in `items`; notify() drops it when this
  // agent's conversation is the focused surface.
  function announceFinish(agentId) {
    if (!agentId) return;
    const turns = groupTurns(itemsRef.current);
    let last = null;
    for (const t of turns) if (t.kind === "turn" && t.replies.length) last = t;
    if (!last) return;
    const loc = locate(fleetRef.current.workspaces, fleetRef.current.freeAgents, agentId);
    const found = loc && loc.agent;
    notify(agentFinishNotice({
      agent: { id: agentId, name: found ? displayAgentName(found) : "Agent", cli: (found && found.cli) || "pi" },
      turn: last,
      target: workspaceHash(agentId),
    }));
  }

  function handleEvent(env, panel) {
    if (panelRef.current !== panel || panel.stopped || selectedRef.current !== panel.agentId) return;
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
        announceFinish(panel.agentId);
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
        setItems((cur) => startTool(cur, {
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
        }));
        queueMicrotask(scrollConv);
        break;
      }
      case "capture_frame": {
        setItems((cur) => cur.map((it) => applyCaptureFrame(it, ev)));
        break;
      }
      case "tool_execution_update": {
        setItems((cur) => cur.map((it) => (it.kind === "tool" && it.id === ev.toolCallId
          ? updateCapture(it, ev.partialResult?.details) : it)));
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
            detail: toolResultDetail(ev.result),
            result: ev.result,
            expanded: it.expanded || searchHits.length > 0,
            change,
            ...captureOnEnd(it, ev.result?.details),
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

  async function startManaged(id, opts) {
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) {
      if (opts?.throwErrors) throw new Error("This agent is no longer available.");
      return;
    }
    try {
      if (!agentIsPi(loc.agent) && loc.agent.terminalId) {
        await api("/api/terminals/" + encodeURIComponent(loc.agent.terminalId) + "/launch/start", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ confirm: false }),
        });
        await openTermTab(loc.agent.terminalId);
        return;
      }
      await api(`/api/agents/${loc.agent.id}/managed/start`, { method: "POST" });
      const list = await refreshFleetFallback();
      openTab(loc.agent.id, list);
    } catch (err) { if (opts?.throwErrors) throw err; toastError(err); }
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

  // The menu's own doors. `ask` seeds the bar with the selection: one line
  // becomes the message, anything longer is staged as selection.txt so the
  // CLI reads a whole file instead of a mangled one-line paste
  // (lib/termMenu.js planAsk, ADR-0089 delivery by path).
  function openTermAttach(ctx, selection) {
    const plan = planAsk(selection || "");
    setTermAttach({
      id: ctx.id,
      token: String(Date.now()),
      text: plan.mode === "text" ? plan.text : "",
      files: plan.mode === "file" ? [new File([plan.body], plan.name, { type: "text/plain" })] : [],
    });
  }
  // A files-paste (screenshot, artifact) on a terminal pane opens the same
  // bar the Attach menu does, seeded with the pasted files and any
  // accompanying text as the message — a bar already open for this pane
  // keeps its text while the new files stage alongside (the seed token
  // re-runs addFiles; the seed effect appends the text with a space).
  function openTermAttachFiles(id, files, text) {
    if (!id || !files || !files.length) return;
    setTermAttach({ id, token: String(Date.now()), text: text || "", files: [...files] });
  }

  function openTermHandoff(term, target) {
    const session = sessionFromTerminal(term);
    const sourceCli = terminalHandoffSourceCli(term);
    // The agent landing needs no installed CLI — only a terminal does.
    if (!session || !sourceCli || !target || (target.landing !== "agent" && !target.installed)) return;
    const source = clis.find((c) => c.id === sourceCli);
    setTermHandoff({
      session,
      sourceCli,
      sourceName: (source && source.name) || sourceCli,
      target,
    });
  }

  // Fork agent…: the dialog runs the flow; App lends it the git door so a
  // new worktree is made by a visible command (ADR-0096), and opens the
  // fork's terminal when it starts.
  function openForkAgent(ag, term) {
    const cliId = (term && term.lastSession && term.lastSession.cli) || ag.cli;
    const cli = clis.find((c) => c.id === cliId);
    setForkSource({ agent: ag, cliName: (cli && cli.name) || cliId });
  }

  function onForkAgentDone(res) {
    const forked = res && res.agent;
    const next = res && res.terminal;
    setForkSource(null);
    loadWorkspaces();
    if (next && next.launchError) {
      // The agent exists (ADR-0184); its stopped terminal says why and offers Start.
      toastError(new Error(next.launchError));
    } else if (forked) {
      toast.ok((forked.name || "The fork") + " is starting on a copy of the conversation.");
    }
    if (next && next.id) {
      openTermTab(next.id);
      location.hash = termHash(next.id);
    }
  }

  function onTermHandoffDone(res, target) {
    setTermHandoff(null);
    const next = res && res.terminal;
    const adopted = res && res.agent;
    const brief = res && res.brief;
    if (next && next.launchError) {
      // The agent exists (ADR-0184); its stopped terminal says why and offers Start.
      toastError(new Error(next.launchError));
      if (next.id) location.hash = termHash(next.id);
    } else if (next && next.id) {
      toast.ok(target.name + " is opening with this conversation.");
      location.hash = termHash(next.id);
    } else if (adopted && adopted.id) {
      toast.ok(brief ? "Continued as a Pi agent: " + adopted.name + ". Its first message reads the brief." : "Continued as a Pi agent: " + adopted.name + ".");
      revealAgent(adopted.id);
    } else {
      toast.ok("Handoff recorded.");
    }
  }

  const termMenuHandlers = {
    fullscreen: () => focus.toggle(),
    ask: (ctx) => openTermAttach(ctx, ctx.selection),
    attach: (ctx) => openTermAttach(ctx, ""),
    pasteFiles: (ctx, files, text) => openTermAttachFiles(ctx.id, files, text),
    snippet: (ctx) => setSnipRun({
      target: { type: ctx.kind === "agent" ? "agent" : "terminal", id: ctx.id },
      targetName: ctx.record && ctx.record.name,
      onlyKind: "prompt",
      via: ctx.kind === "agent" ? "tui" : "cli",
    }),
    "snippet-cmd": (ctx) => setSnipRun({ target: { type: "terminal", id: ctx.id }, targetName: ctx.record && ctx.record.name, onlyKind: "shell" }),
    find: (ctx) => setTermFind(ctx.id),
    handoff: (ctx, targetId) => {
      // Pane menu ids are "handoff:<targetCli>" or "handoff:<targetCli>:<landing>".
      const record = ctx && ctx.record;
      if (!record || !targetId) return;
      const [cliId, landing] = String(targetId).split(":");
      const target = handoffTargets(clisRef.current, terminalHandoffSourceCli(record)).find((t) => t.id === cliId);
      if (target) openTermHandoff(record, landing ? { ...target, landing } : target);
    },
    "open-link": (ctx) => {
      if (!ctx.link) return;
      // A dev server URL printed in a terminal opens in PiCode's own browser
      // surface — the page being developed is the one the terminal just
      // printed. Where local links open is a user preference (Browser
      // settings, "Local development sites"), so it decides here too: the
      // decision table lives in lib/openLink.js. Anything not local, and any
      // file path, behaves exactly as before.
      const first = linkOpenTarget(ctx.link, "app", "app");
      if (!first) return;
      if (first.action === "file") { openFileTab(ctx.kind === "agent" ? "agent" : "term", ctx.id, first.path); return; }
      if (first.action === "external") { window.open(first.url, "_blank", "noopener,noreferrer"); return; }
      // One fresh read per link, like the shell's popup door: links are rare
      // and the preference must be current. A preference we cannot read keeps
      // the feature's default (PiCode's own surface).
      fetch("/api/browser/prefs")
        .then((r) => r.json())
        .then((p) => {
          const dest = linkOpenTarget(ctx.link, p.localOpenDest, p.webOpenDest);
          if (dest.action === "external") window.open(dest.url, "_blank", "noopener,noreferrer");
          else openWebTab(dest.url);
        })
        .catch(() => openWebTab(first.url));
    },
    rename: (ctx) => {
      if (ctx.ownerKind === "agent") {
        const loc = locate(workspacesRef.current, freeAgentsRef.current, ctx.ownerId);
        if (loc && loc.agent) renameAgent(loc.agent);
        return;
      }
      renameTerminal(ctx.record);
    },
    settings: (ctx) => {
      location.hash = ctx.kind === "agent" ? "#/termset" : "#/termset/" + encodeURIComponent(ctx.id);
    },
    files: (ctx) => openTreeTab(ctx.kind === "agent" ? "agent" : "term", ctx.id, ctx.record ? ctx.record.name : ""),
    "close-tab": (ctx) => closeTab(ctx.tabId),
    remove: (ctx) => {
      if (ctx.ownerKind === "agent") {
        const loc = locate(workspacesRef.current, freeAgentsRef.current, ctx.ownerId);
        if (loc && loc.agent) removeAgent(loc.agent);
        return;
      }
      removeTerminal(ctx.record);
    },
    "open-browser": (ctx) => openAgentSplit(ctx.tabId),
    "close-browser": (ctx) => closeAgentSplit(ctx.tabId),
    "new-tab": () => openWebTab(""),
  };

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

  // Start / stop / restart from the sidebar's terminal menu — the same
  // launch endpoint the Agent CLIs list uses, so one terminal answers to
  // one set of actions from either surface. The fleet feed patches the
  // sidebar in place; start opens the terminal like the CLIs list does.
  async function launchTerminalAction(t, op, agent) {
    if (agent && agentIsPi(agent)) {
      if (!(await askConfirm({
        title: `${op === "restart" ? "Restart" : "Stop"} ${agent.name || "agent"}?`,
        message: op === "restart"
          ? "This interrupts the current work and restarts the agent in the same mode."
          : "This interrupts the agent's current work.",
        confirmLabel: op === "restart" ? "Restart agent" : "Stop agent",
        danger: true,
      }))) return;
      const pending = notify({ level: "busy", title: op === "restart" ? "Restarting agent…" : "Stopping agent…", duration: Infinity });
      try {
        if (op === "restart" && agent.mode === "interactive") {
          await openInteractive(agent.id, { restart: true, throwErrors: true });
        } else {
          await stopAgent(agent.id, { throwErrors: true });
          if (op === "restart") await startManaged(agent.id, { throwErrors: true });
        }
        toast.ok(op === "restart" ? "Agent restarted." : "Agent stopped.");
      } catch (err) { toastError(err); }
      finally { dismissNotice(pending); }
      return;
    }
    if (!t || !op) return;
    const noun = agent ? "agent" : "terminal";
    const destructive = op === "remove" || (t.running && op !== "start");
    if (destructive && !(await askConfirm({
      title: `${op === "stop" ? "Stop" : op === "restart" ? "Restart" : "Remove"} ${agent?.name || t.name || noun}?`,
      message: agent
        ? (op === "restart"
          ? (t.lastSession ? "This interrupts the agent's current work, then reopens the same conversation." : "This interrupts the current work and restarts the agent.")
          : "This interrupts the agent's current work.")
        : op === "restart"
        ? (t.lastSession
          ? "This ends the processes running in this terminal, then reopens the same conversation."
          : "This ends the processes running in this terminal.")
        : t.running ? "This ends the processes running in this terminal." : "Remove this saved terminal and its launch settings?",
      confirmLabel: `${op === "stop" ? "Stop" : op === "restart" ? "Restart" : "Remove"} ${noun}`,
      danger: true,
    }))) return;
    const pending = agent ? notify({ level: "busy", title: op === "restart" ? "Restarting agent…" : "Stopping agent…", duration: Infinity }) : null;
    try {
      await api(`/api/terminals/${encodeURIComponent(t.id)}/launch/${op}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ confirm: destructive }),
      });
      if (op === "start") location.hash = termHash(t.id);
      else toast.ok(`${agent ? "Agent" : "Terminal"} ${op === "stop" ? "stopped" : op === "restart" ? "restarted" : "removed"}.`);
    } catch (err) { toastError(err); }
    finally { dismissNotice(pending); }
  }

  async function openInteractive(id, opts) {
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) {
      if (opts?.throwErrors) throw new Error("This agent is no longer available.");
      return;
    }
    try {
      const forceRestart = !!(opts && opts.restart);
      const restart = forceRestart ? "?restart=1" : "";
      await api(`/api/agents/${loc.agent.id}/open${restart}`, { method: "POST" });
      if (forceRestart) {
        closeAgentShell(loc.agent);
        setTermEpochs((cur) => ({ ...cur, [loc.agent.id]: (cur[loc.agent.id] || 0) + 1 }));
      }
      const list = await refreshFleetFallback();
      openTab(loc.agent.id, list);
      if (!opts || opts.dock !== false) {
        setChatWanted((s) => { const next = new Set(s); next.delete(loc.agent.id); return next; });
        setTermWanted((s) => new Set(s).add(loc.agent.id));
      }
    } catch (err) { if (opts?.throwErrors) throw err; toastError(err); }
  }

  async function stopAgent(id, opts) {
    if (automateRef.current && automateRef.current.agentId === id) automateRef.current = null;
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) {
      if (opts?.throwErrors) throw new Error("This agent is no longer available.");
      return;
    }
    try {
      if (!agentIsPi(loc.agent) && loc.agent.terminalId) {
        await api("/api/terminals/" + encodeURIComponent(loc.agent.terminalId) + "/launch/stop", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ confirm: true }),
        });
        await refreshFleetFallback();
        return;
      }
      await api(`/api/agents/${loc.agent.id}/close`, { method: "POST" });
      closeAgentShell(loc.agent);
      if (panelRef.current && panelRef.current.agentId === loc.agent.id) panelRef.current.stopped = true;
      optimisticRef.current = false;
      setStreaming(false);
      streamingRef.current = false;
      setWaiting(false);
      setStatus("stopped");
      setItems((cur) => cancelOpenAsks(cur));
      await refreshFleetFallback();
    } catch (err) { if (opts?.throwErrors) throw err; toastError(err); }
  }

  async function confirmCleanup({ title, message, path, extraChoices }) {
    let preview = { lastOccupant: false, sessions: 0, sessionBytes: 0, canPurgeWork: false };
    try { preview = await api(path); } catch { /* unregister still allowed */ }
    if (preview.terminals > 0) {
      const n = preview.terminals;
      message += ` This also removes ${n} terminal${n === 1 ? "" : "s"} and stops ${n === 1 ? "its" : "their"} tmux session${n === 1 ? "" : "s"}.`;
    }
    // An agent preview says whether the dialog asks how it went (ADR-0194);
    // a workspace preview has no exit block.
    const exit = preview.exit || null;
    const choices = [];
    if (preview.lastOccupant && preview.sessions > 0) {
      const n = preview.sessions;
      choices.push({
        id: "sessions",
        label: `Also delete ${n} session${n === 1 ? "" : "s"} (${fmtBytes(preview.sessionBytes)}) for this folder`,
        checkedHint: exit ? "A later review of how this agent went can only read the sessions you keep." : "",
      });
    }
    if (preview.canPurgeWork) {
      choices.push({ id: "work", label: "Also delete the work folder" });
    }
    for (const c of extraChoices || []) choices.push(c);
    const feedback = exit && exit.ask ? { taxonomy: exit.taxonomy } : null;
    const ok = await askConfirm({ title, message, confirmLabel: "Remove", danger: true, choices, feedback });
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
    return { query: qs ? "?" + qs : "", exit, feedback: picked.feedback || null };
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

  // Best-effort reversal of removeAgent: the old row's id is gone for good
  // (automations pointing at it stay broken), but the agent comes back in
  // the same workspace with the same name, cli and config, and its session
  // history re-attached — unless the dialog's purge was picked, in which
  // case the sessions are already deleted and nothing pretends otherwise.
  // Per-agent package selection rides the packages surface, not PATCH, so
  // it is the one setting this cannot carry over.
  async function undoRemoveAgent(snap) {
    try {
      let created;
      if (snap.workspaceId && snap.workspaceId !== "ws_free") {
        created = await api("/api/workspaces/" + snap.workspaceId + "/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            cli: snap.cli, name: snap.name, workPath: snap.workPath || "",
            provider: snap.provider || "", model: snap.model || "", thinking: snap.thinking || "",
          }),
        });
      } else {
        created = await api("/api/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            cli: snap.cli, name: snap.name, path: snap.workPath || "",
            provider: snap.provider || "", model: snap.model || "", thinking: snap.thinking || "",
          }),
        });
      }
      const patch = {};
      if (snap.opMode) patch.opMode = snap.opMode;
      if (snap.checklist) patch.checklist = snap.checklist;
      if (snap.extraPrompt) patch.extraPrompt = snap.extraPrompt;
      if (snap.packagesIsolated) patch.packagesIsolated = true;
      if (!snap.sessionsPurged && snap.sessionPath) patch.sessionPath = snap.sessionPath;
      if (Object.keys(patch).length) {
        await api("/api/agents/" + created.id, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(patch),
        });
      }
      // The exit stays for the record and leaves every count (ADR-0194).
      if (snap.exitId) {
        api("/api/agent-exits/" + encodeURIComponent(snap.exitId) + "/undo", {
          method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ agentId: created.id }),
        }).catch(() => { /* the agent is back either way */ });
      }
      // The feed may not have delivered the brand-new row yet (and
      // refreshFleetFallback returns nothing when it is live), so undo
      // refetches and hands the fresh list to openTab itself.
      const list = await loadWorkspaces();
      openTab(created.id, list);
      toast.ok(`"${snap.name}" is back.`);
    } catch (err) { toastError(err); }
  }

  async function removeAgent(ag) {
    const choice = await confirmCleanup({
      title: "Remove agent",
      // A bound terminal goes with its agent — for a shell made an agent
      // (ADR-0184), that is the shell and whatever runs in it.
      message: `Remove "${ag.name}"? ` + (ag.terminalId ? "Its terminal closes with it, and anything running there stops. " : "") + "The project folder is not deleted.",
      path: "/api/agents/" + ag.id + "/cleanup",
    });
    if (!choice) return;
    // An agent's removal can take two tabs (its own and, for a CLI agent,
    // its bound terminal's — ADR-0160). The neighbour is picked before any
    // filter lands, while the strip still anchors the reader's place.
    const removedTabs = [ag.id, ag.terminalId ? termTabId(ag.terminalId) : ""].filter(Boolean);
    const next = removedTabs.includes(selectedRef.current) ? pickNextTab(tabsRef.current, removedTabs) : null;
    const undoSnapshot = {
      workspaceId: ag.workspaceId || "",
      cli: ag.cli || "pi",
      name: ag.name,
      workPath: ag.workPath || "",
      provider: ag.provider || "",
      model: ag.model || "",
      thinking: ag.thinking || "",
      opMode: ag.opMode || "",
      checklist: ag.checklist || "",
      sessionPath: ag.sessionPath || "",
      extraPrompt: ag.extraPrompt || "",
      packagesIsolated: !!ag.packagesIsolated,
      sessionsPurged: (choice.query || "").includes("sessions=1"),
      exitId: "",
    };
    // The answer rides the removal itself (ADR-0194): the exit is written
    // in the same transaction as the delete, so nothing is left to send.
    const fb = choice.feedback;
    const body = exitRequestBody({ taxonomy: choice.exit && choice.exit.taxonomy, draft: fb && fb.draft, shown: !!(fb && fb.shown), origin: "desktop" });
    try {
      const removed = await api("/api/agents/" + ag.id + choice.query, {
        method: "DELETE", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body),
      });
      undoSnapshot.exitId = (removed && removed.exit && removed.exit.id) || "";
    } catch (err) {
      // The server may have completed the delete before the response was
      // lost (a second click, a flaky moment): 404 means the agent is
      // gone — drop the row instead of scolding the operator twice.
      if (err && err.status !== 404) { toastError(err); return; }
    }
    if (fb && fb.stopAsking) {
      api("/api/agent-exits/prefs", {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ ask: false }),
      }).catch(toastError);
    }
    closeAgentShell(ag);
    setTabs((t) => t.filter((x) => x !== ag.id));
    // A CLI agent's process is its bound terminal (ADR-0160): the delete
    // above only nulls terminal_id, which would leave the terminal — and
    // the live CLI inside it — as an orphan Work card. The row menu
    // promises both go away. Pi keeps its terminal (its menu says so).
    if (!agentIsPi(ag) && ag.terminalId) {
      try {
        await api("/api/terminals/" + encodeURIComponent(ag.terminalId), { method: "DELETE" });
      } catch { /* already gone */ }
      closeShellTerm(ag.terminalId);
      setTabs((t) => t.filter((x) => x !== termTabId(ag.terminalId)));
    }
    if (removedTabs.includes(selectedRef.current)) adoptTab(next);
    // A removal is rare and final: refetch even when the feed is live, so
    // the row leaves on this answer and not only on the next event.
    await loadWorkspaces();
    // Feedback + the way back. An action carries the notice model's floor
    // (8s), so the Undo is readable without pinning the screen.
    notify({
      level: "ok",
      title: `Removed "${ag.name}".`,
      actions: [{ label: "Undo", primary: true, run: () => undoRemoveAgent(undoSnapshot) }],
    });
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
      // Its agents end with it and each gets an exit (ADR-0194); the body
      // only says which face removed them.
      await api("/api/workspaces/" + ws.id + choice.query, {
        method: "DELETE", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ exit: { origin: "desktop" } }),
      });
      const ids = (ws.agents || []).map((a) => a.id);
      if (ws.agent) ids.push(ws.agent.id);
      // The workspace's terminals died with it (ADR-0026): drop them from
      // the list and close their tabs, like removeTerminal does.
      const deadTerms = terminals.filter((t) => termWorkspaceId(t) === ws.id);
      // The neighbour is picked before any filter lands, while the strip
      // still anchors the reader's place: workspace tab, its agents' tabs
      // and its terminals' tabs all leave together.
      const removedTabs = [ws.id, ...ids, ...deadTerms.map((t) => termTabId(t.id))];
      const next = removedTabs.includes(selectedRef.current) ? pickNextTab(tabsRef.current, removedTabs) : null;
      for (const id of [...new Set(ids)]) {
        const ownedAgent = (ws.agents || []).find((candidate) => candidate.id === id) || (ws.agent && ws.agent.id === id ? ws.agent : null);
        closeAgentShell(ownedAgent || { id });
        if (panelRef.current && panelRef.current.agentId === id) closePanel();
      }
      setTerminals((cur) => cur.filter((t) => termWorkspaceId(t) !== ws.id));
      for (const t of deadTerms) {
        setTabs((cur) => cur.filter((id) => {
          const f = parseFileTab(id);
          return !(f && f.kind === "term" && f.id === t.id);
        }));
        closeTab(termTabId(t.id));
      }
      setTabs((t) => t.filter((x) => x !== ws.id && !ids.includes(x)));
      if (removedTabs.includes(selectedRef.current)) adoptTab(next);
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
    const schema = formKind === "workspace" ? createWorkspaceSchema : createWsAgentSchema;
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
    setChatWanted(s => { const n = new Set(s); n.delete(selectedId); return n; });
    setTermWanted((s) => new Set(s).add(selectedId));
    if (!interactive) openInteractive(selectedId);
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

  async function patchAgent(cfg, target = agent, reportError = true) {
    if (!target) return;
    const agent = target;
    const modeChanged = Object.prototype.hasOwnProperty.call(cfg, "opMode")
      && (cfg.opMode || "full") !== (agent.opMode || "full");
    const was = agent.mode;
    const dockWasOpen = termWanted.has(agent.id);
    try {
      await api("/api/agents/" + agent.id, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(cfg),
      });
      await loadWorkspaces();
      if (modeChanged && was && was !== "stopped") {
        try {
          await stopAgent(agent.id, { throwErrors: true });
          if (was === "managed") await startManaged(agent.id, { throwErrors: true });
          else if (was === "interactive") await openInteractive(agent.id, { dock: dockWasOpen, throwErrors: true });
        } catch (error) {
          throw new Error("Settings saved, but the agent could not restart. Check its state and use Start. " + error.message);
        }
      }
    } catch (e) { if (reportError) toastError(e); else throw e; }
  }

  // A link printed in a terminal is a browsing action inside PiCode: Ctrl+click
  // and the pane menu's Open both land here, and the human's own preference
  // decides (Browser settings — local dev sites vs web URLs, both defaulting to
  // PiCode's surface). The system browser stays reachable by setting either to
  // "Default browser"; before 2026-09-17 every non-loopback link left the app
  // from the terminal, which is the one door that ignored that preference.
  // The shell's external-links bridge (lib/externalLinks.js) hands every
  // chrome link that leaves the app to this event — Documentation, guides,
  // changelog — and the owner wants them in a PiCode browser tab, not the
  // system browser (2026-09-21). Deliberately not openTermLink: the browser
  // prefs govern links clicked inside a browsed page, not the app's own
  // chrome.
  useEffect(() => {
    const onOpen = (e) => {
      const url = typeof e.detail === "string" ? e.detail : "";
      if (!url) return;
      // A remote page hosts only in the shell's WebView2; a plain browser
      // keeps the system browser (its own caption says exactly that).
      if (shellChrome && window.__TAURI__) openWebTabRef.current(url);
      else window.open(url, "_blank", "noopener,noreferrer");
    };
    window.addEventListener(OPEN_LINK_EVENT, onOpen);
    return () => window.removeEventListener(OPEN_LINK_EVENT, onOpen);
  }, []);

  const openTermLink = useCallback((href) => {
    const url = String(href || "").trim();
    if (!url) return;
    fetch("/api/browser/prefs")
      .then((r) => r.json())
      .then((p) => {
        const target = linkOpenTarget({ kind: "http", href: url }, p.localOpenDest, p.webOpenDest);
        if (!target) return;
        if (target.action === "external") window.open(target.url, "_blank", "noopener,noreferrer");
        else openWebTab(target.url);
      })
      .catch(() => openWebTab(url));
  }, []);

  // A CLI's "open in the browser" (ADR-0180: login pages above all) comes
  // in over the feed — the wrapper in the terminal posts to the daemon,
  // which turns it into terminal.open_url. The client actually on screen
  // answers through the clicked-link path, so the destination preference
  // governs it too; a hidden client stays out (the daemon's host fallback
  // covers the nobody-watching case).
  useEffect(() => installOpenUrlFeed({
    subscribe: subscribeFeed,
    open: openTermLink,
    hidden: () => typeof document !== "undefined" && document.visibilityState !== "visible",
  }), [openTermLink]);

  // The wordmark's one action, shared by the shell's top row and the
  // browser's sidebar: show the dashboard. Pinning alone was not enough.
  //  - The dashboard is rendered inside the workspace view, which the
  //    non-workspace routes keep hidden, so from Agent CLIs, Browser,
  //    Preferences or Devices the click looked dead until the human
  //    navigated back on their own. It now comes back to the workspace
  //    route (the hash a tab already owns is left alone).
  //  - In the shell the brand is a button inside a drag region; nothing
  //    inside it may carry data-tauri-drag-region, or Tauri answers the
  //    mousedown with a native window drag and the click never fires
  //    (2026-09-17: the wordmark dragged the window instead of opening
  //    the dashboard).
  const openDashboard = useCallback(() => {
    setDashboardPinned(true);
    setNavigationOpen(false);
    if (parseRoute() !== "workspace") go("workspace");
  }, []);

  const onPane = route !== "workspace";
  const missing = !!goneId;
  const noTabs = tabs.length === 0 && !missing;
  const hasData = (workspaces.length + freeAgents.length + terminals.length) > 0;
  const showHome = (noTabs || dashboardPinned) && hasData && !isWebTab(selectedId);
  const badgeTabs = tabsRef.current;
  const webappsBadged = webapps.map((a) => ({ ...a, badge: webappBadge(a, badgeTabs, webTabs) }));

  const tabsStrip = (
<AgentTabs
    tabs={tabs}
    workspaces={workspaces}
    freeAgents={freeAgents}
    terminals={terminals}
    apps={apps}
    webapps={webapps}
    webTabs={webTabs}
    fileWorktrees={fileWorktrees}
    selectedId={selectedId}
    onSelect={(id) => openTab(id)}
    onClose={closeTab}
    onReorder={(from, to) => setTabs((t) => moveTab(t, from, to))}
    keepVisible={focus.on}
    endSlot={!narrow ? (
      <>
        <button type="button" className="insp-toggle" data-browser-globe="" aria-label="Open browser" title="Open browser" onClick={onGlobeClick}>
          <IconGlobe />
        </button>
        <InspectorToggle
          shown={focus.on ? focus.reveal === "right" : inspectorLayout.shown}
          reason={railFits ? inspectorLayout.reason : (narrow ? "narrow" : "squeezed")}
          onToggle={toggleInspector}
        />
        {focus.on ? <FocusLeave onLeave={focus.leave} /> : null}
      </>
    ) : null}
  />
  );

  return (
    <div id="app" className={[shellChrome ? "app-shell" : "", navigationOpen ? "navigation-open" : "", focus.classes].filter(Boolean).join(" ")}>
      {shellChrome && (
        <header className="shell-row" data-tauri-drag-region>
          <div className="shell-brand-cluster" data-tauri-drag-region>
            {/* No data-tauri-drag-region inside this button: Tauri starts
                a window drag on the element the mousedown lands on, and a
                span carrying the attribute swallows the click before the
                button sees it. The row around it stays draggable. */}
            <button type="button" className="shell-brand" title="Dashboard" onClick={openDashboard}>
              <span className="shell-mark"><IconBrandMark /></span>
              <span className="shell-name">PiCode</span>
            </button>
            <RailTabs tab={sideTab} selectTab={selectSideTab} apps={apps} pkgUpdates={pkgUpdates} onOpenClis={() => { go("clis"); setNavigationOpen(false); }} />
          </div>
          {tabsStrip}
          <WindowControls />
        </header>
      )}
      <header className="desktop-compact-bar">
        <button type="button" className="btn btn-ghost" aria-expanded={navigationOpen} aria-controls="desktop-navigation" onClick={() => setNavigationOpen(open => !open)}>
          {navigationOpen ? "Close navigation" : "Navigation"}
        </button>
        <span>PiCode</span>
      </header>
      <div id="desktop-navigation" inert={narrow && !navigationOpen} onKeyDown={event => {
        // Only Escape pressed inside the drawer itself: a dialog or menu the
        // sidebar opens lives in a portal, and React still bubbles its keys
        // here — one Escape closed both the dialog and the drawer under it.
        if (event.key === "Escape" && event.currentTarget.contains(event.target)) {
          setNavigationOpen(false);
          document.querySelector('[aria-controls="desktop-navigation"]')?.focus();
        }
      }}>
      <Sidebar
        fleetLoaded={bootstrapped}
        onOpenDocs={() => {
          // The work browser tab hosts pages only in the shell (WebView2);
          // in a plain browser the docs keep going to the system browser,
          // which is where this same link landed before. Same test
          // webappOpenPlan makes: the Tauri bridge, not the boot flag.
          if (shellChrome && window.__TAURI__) openWebTabRef.current("https://cfpperche.github.io/picode/");
          else window.open("https://cfpperche.github.io/picode/", "_blank", "noopener,noreferrer");
        }}
        inShell={shellChrome}
        tab={sideTab}
        selectTab={selectSideTab}
        workspaces={workspaces}
        selectedId={selectedId}
        onNew={() => { setFormKind("workspace"); setShowForm(true); }}
        onNewFree={() => { setCliPrincipalWs({ free: true }); }}
        onNewAgent={(id) => {
          const ws = workspaces.find((w) => w.id === id);
          if (ws && ws.id !== "ws_free") setCliPrincipalWs(ws);
        }}
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
        onLaunchAction={launchTerminalAction}
        onContinueTerm={openTermHandoff}
        onForkAgent={openForkAgent}
        clis={clis}
        onRenameTerm={renameTerminal}
        onReorder={reorderSidebar}
        onGitGraph={openGitTab}
        onFileTree={openTreeTab}
        onInstructions={openInstructionsTab}
        onOpenDashboard={openDashboard}
        onOpenClis={() => { go("clis"); setNavigationOpen(false); }}
        apps={apps}
        nativeApps={NATIVE_APPS}
        onOpenApp={(id) => { openTab(appTabId(id)); if (parseRoute() !== "workspace") location.hash = appHash(id); }}
        webapps={webappsBadged}
        webappsErr={webappsErr}
        webappsLoaded={webappsLoaded}
        onRetryWebapps={() => webappsWatchRef.current?.refresh()}
        onOpenWebapp={openWebapp}
        onSavedWebapp={savedWebapp}
        onRemoveWebapp={removeWebapp}
        onRefreshWebapp={refreshWebapp}
        onClearWebappData={clearWebappData}
        iconVersions={iconVersions}
        desktop={shellChrome && !!window.__TAURI__}
        onChat={(id) => {
          revealAgent(id);
          setChatWanted(s => new Set(s).add(id));
          setTermWanted((s) => {
            const n = new Set(s);
            n.delete(id);
            return n;
          });
        }}
        onTerm={(id) => {
          revealAgent(id);
          setChatWanted(s => { const n = new Set(s); n.delete(id); return n; });
          setTermWanted((s) => new Set(s).add(id));
          const loc = locate(workspaces, freeAgents, id);
          if (loc && loc.agent && loc.agent.mode !== "interactive") openInteractive(id);
        }}
        userMenu={{
          host,
          version,
          inShell: shellChrome,
          themeMode,
          onTheme: setTheme,
          onNavigate: (kind) => go(kind, ctxAgent?.id, { workspaceId: paneWs?.id, cli: ctxAgent?.cli }),
          onWhatsNew: openWhatsNew,
          whatsNewUnread,
          pkgUpdates,
        }}
      />
      </div>

      <main id="main">
        <div id="workspace-view" className={"workspace-view" + (isTermTab(selectedId) ? " term-on" : "") + (isFileTab(selectedId) ? " file-on" : "") + (isGitTab(selectedId) ? " git-on" : "") + (isTreeTab(selectedId) ? " tree-on" : "") + (isAppTab(selectedId) ? " app-on" : "") + (showHome ? " dashboard-on" : "") + (agentPanes[selectedId] ? " agent-split-on" : "") + (agentPanes[selectedId] && paneMax[selectedId] ? " agent-split-max" : "")} hidden={onPane} style={agentPanes[selectedId] ? { "--split-w": (paneMax[selectedId] ? 100 : (paneRatios[selectedId] ?? 50)) + "%" } : undefined}>
          {!shellChrome && tabsStrip}

          <div id="empty" className="empty" hidden={!(missing || (noTabs && !hasData))}>
            <div className="empty-card">
              {missing ? (
                <>
                  <h2>{isAppTab(goneId) ? "That app is gone." : isInstructionsTab(goneId) ? "That workspace is gone." : isFileTab(goneId) ? "That file is gone." : isTermTab(goneId) || (isGitTab(goneId) && goneId.startsWith("g:@t:")) || (isTreeTab(goneId) && goneId.startsWith("d:@t:")) ? "That terminal is gone." : (isTreeTab(goneId) && goneId.startsWith("d:@w:")) || (isGitTab(goneId) && goneId.startsWith("g:@w:")) ? "That workspace is gone." : "That agent is gone."}</h2>
                  {hasData ? (
                    <p>Pick another from the sidebar.</p>
                  ) : (
                    <>
                      <p>Add a project folder to create your first agent.</p>
                      <button id="btn-new-empty" className="btn btn-primary" onClick={() => setShowForm(true)}>Add workspace</button>
                    </>
                  )}
                </>
              ) : !bootstrapped ? (
                // Before the boot has read the fleet, "nothing yet" is not
                // known: the card waits like the sidebar does.
                <div className="empty-skel" aria-label="Loading your work"><div className="skel-line w-70" /><div className="skel-line w-90" /><div className="skel-line w-40" /></div>
              ) : (
                <>
                  <h2>No agents yet</h2>
                  <p>Add a project folder to create your first agent.</p>
                  <button id="btn-new-empty" className="btn btn-primary" onClick={() => setShowForm(true)}>Add workspace</button>
                </>
              )}
            </div>
          </div>

          {showHome ? <DashboardView workspaces={workspaces} freeAgents={freeAgents} terminals={terminals} workingIds={tuiWorking} waitingId={waiting ? selectedId : null} onOpen={(id) => openTab(id)} inboxWaiting={inboxWaiting} onOpenApp={(id) => { openTab(appTabId(id)); if (parseRoute() !== "workspace") location.hash = appHash(id); }} /> : null}

          {tabs.filter(isTermTab).map((id) => {
            const tid = tabTermId(id);
            const t = terminals.find((x) => x.id === tid);
            if (!t) return null;
            return (
              <TermSurface
                key={id}
                adopt={adoptOffer(t, terminalLaunchAgent(t.id, workspaces, freeAgents))}
                term={t}
                tabId={id}
                hidden={selectedId !== id}
                error={selectedId === id ? termError : ""}
                onOpenFile={(p) => openFileTab("term", tid, p)}
                onOpenLink={openTermLink}
                attach={termAttach && termAttach.id === tid ? termAttach : null}
                onAttachClose={() => setTermAttach(null)}
                find={termFind === tid}
                onFindClose={() => { setTermFind(""); focusPane(tid); }}
                onPasteFiles={openTermAttachFiles}
                promptDoor={promptDoorFor({ kind: "term", term: t })}
              />
            );
          })}
          {tabs.filter(isWebTab).map((id) => {
            const app = webapps.find((a) => a.id === webappIdFromTab(id));
            return (
              <WebTabSurface
                key={id}
                tabId={id}
                url={(webTabs[tabWebId(id)] && webTabs[tabWebId(id)].url) || app?.url || ""}
                active={selectedId === id}
                hidden={selectedId !== id || onPane}
                chromeless={!!app && webappChromeless(app.display)}
                asks={permissionAsks.filter((a) => a.tab === tabWebId(id))}
                onAnswerAsk={answerPermission}
                onMeta={(m) => setWebTabs((cur) => updateWebappMeta(cur, tabsRef.current, id, m))}
                onNew={() => openWebTab("")}
                onBrowserSettings={() => go("browser")}
              />
            );
          })}

          <FileSurface
            owner={fileTabInfo}
            path={fileTabInfo ? fileTabInfo.path : ""}
            onClose={() => isFileTab(selectedId) && closeTab(selectedId)}
            view={fileViews[selectedId] || "file"}
            onView={(view) => setFileViews((v) => ({ ...v, [selectedId]: view }))}
            changed={fileTabChanged}
            root={(fileTabWT && fileTabWT.root) || ""}
            worktree={(fileTabWT && fileTabWT.ref) || ""}
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
                onView={view => setGitOwners(m => ({ ...m, [id]: { ...m[id], view } }))}
                workspaces={workspaces}
                freeAgents={freeAgents}
                terminals={terminals}
                onPickWorkspace={(wsId) => switchGraphWorkspace(id, wsId)}
                hidden={selectedId !== id}
                onKey={(key) => onGitKey(id, key)}
                onClose={() => closeTab(id)}
                onMenu={openGraphMenu}
                actionTick={gitActionTick}
                done={gitActionDone && gitActionDone.ownerId === ownerIdOf(o) ? gitActionDone : null}
                onUndo={(undo) => openGraphAction(
                  { id: "act:" + undo.action, kind: "action", action: undo.action, label: "Undo", tier: "B", needs: undo.name ? ["target", "name"] : ["target"], target: undo.target, name: undo.name },
                  { owner: o, root: (gitAction && gitAction.root) || "", agents: [], refs: [] },
                )}
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
                tabId={id}
                owner={o}
                workspaces={workspaces}
                freeAgents={freeAgents}
                terminals={terminals}
                onPickWorkspace={(wsId) => switchTreeWorkspace(id, wsId)}
                hidden={selectedId !== id}
                onKey={(root) => onTreeKey(id, root)}
                registerCloseGuard={registerTreeCloseGuard}
                onClose={() => closeTab(id)}
              />
            );
          })}
          {tabs.filter(isInstructionsTab).map((id) => {
            const ws = workspaces.find((w) => w && w.id === instructionsTabWorkspace(id));
            if (!ws) return null;
            return (
              <InstructionsSurface
                key={id}
                workspace={ws}
                hidden={selectedId !== id}
                onOpenFile={(p) => openFileTab("workspace", ws.id, p)}
              />
            );
          })}
          {tabs.filter(isAppTab).map((id) => {
            const appId = tabAppId(id);
            const manifest = apps.find((a) => a.id === appId) || null;
            // A native surface (ADR-0109) mounts the registered component
            // instead of AppSurface — same key, kept mounted and hidden like
            // every tab — with `host`: the client twin of Go's apps.Host and
            // the whole API a native app gets. Rebuilt per render like any
            // prop object; the app reads it and never imports App state.
            const Native = nativeSurfaceFor(manifest, NATIVE_APPS);
            if (Native) {
              return (
                // A registered surface may be a lazy chunk (the Canvas is,
                // ADR-0118), so the mount carries the boundary. `fallback`
                // is null on purpose: a tab that is opening shows nothing
                // for the length of one fetch, never a skeleton of invented
                // rows, and the surface draws its own the moment it lands.
                <Suspense key={id} fallback={null}>
                  <Native
                    manifest={manifest}
                    hidden={selectedId !== id}
                    onClose={() => closeTab(id)}
                    initialPath={appRoute(hash) === appId ? appPath(hash) : undefined}
                    onPathChange={(path) => {
                      if (selectedId !== id) return;
                      const next = appHash(appId, path);
                      if (location.hash !== next) { history.replaceState(null, "", next); setHash(next); }
                    }}
                    host={{
                      // `loaded` says the boot fetch is done: before it, an
                      // empty fleet means "not read yet", not "deleted" — a
                      // native surface must not draw gone rows over it.
                      fleet: { workspaces, freeAgents, terminals, termEpochs, loaded: bootstrapped },
                      openTabs: tabs,
                      openTab, openInteractive, revealAgent, openFileTab,
                      feed: subscribeFeed,
                      termAttach,
                      termFind,
                      onFindClose: (id) => { setTermFind(""); focusPane(id); },
                      onAttachClose: () => setTermAttach(null),
                      onPasteFiles: (id, files, text) => openTermAttachFiles(id, files, text),
                      // The app says what it has in focus; the host decides
                      // what follows from that (ADR-0109, amendment
                      // 2026-09-12). An app never names the Inspector.
                      subject: (owner) => publishSubject(id, owner),
                    }}
                  />
                </Suspense>
              );
            }
            return (
              <AppSurface
                key={id}
                appId={appId}
                nativeSurfaces={NATIVE_APPS.ids}
                initialPath={appRoute(hash) === appId ? appPath(hash) : undefined}
                onPathChange={(path) => {
                  if (selectedId !== id) return;
                  const next = appHash(appId, path);
                  if (location.hash !== next) { history.replaceState(null, "", next); setHash(next); }
                }}
                hidden={selectedId !== id}
                manifest={manifest}
                onClose={() => closeTab(id)}
                // External http(s) links clicked inside the surface open as
                // work-browser tabs — the same flow the address bar uses,
                // which also flips to the workspace pane — never as
                // navigation of this document (2026-09-20/21 incident).
                onOpenUrl={openWebTab}
                onGoto={(g) => {
                  // Apps can focus an agent's existing tab; "agent:" opens its
                  // interactive TUI (replies land in the terminal itself now).
                  if (g.startsWith("agent:")) openInteractive(g.slice("agent:".length));
                  // A reminder's "Open pin" lands in the studio (ADR-0100).
                  if (g.startsWith("pin:")) location.hash = "#/pins/" + encodeURIComponent(g.slice("pin:".length));
                  if (g.startsWith("term:")) {
                    const tid = g.slice("term:".length);
                    openTermTab(tid);
                    location.hash = termHash(tid);
                  }
                }}
              />
            );
          })}
          <ChatSurface
            hidden={noTabs || missing || termView || isTermTab(selectedId) || isFileTab(selectedId) || isGitTab(selectedId) || isTreeTab(selectedId) || isInstructionsTab(selectedId) || isAppTab(selectedId) || isWebTab(selectedId)}
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
              if (cmd.run === "llama") { go("llama"); return; }
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
                location.hash = cliSettingsHash("pi", { agentId: agent?.id, focus: cmd.run === "go-scoped" ? "scoped-models" : "" });
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
              slashExtra, atAgents, onAgentPage: (name) => go(name, agent?.id, { workspaceId: paneWs?.id }), pkgUpdates,
              onCaptureSnippet: openSnipCapture,
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

          {agentPanes[selectedId] && !webappIdFromTab(selectedId) ? (
            <>
              <div
                className="split-divider"
                role="separator"
                aria-orientation="vertical"
                aria-label="Resize panes"
                onPointerDown={(e) => {
                  e.preventDefault();
                  const view = document.getElementById("workspace-view");
                  if (!view) return;
                  const handle = e.currentTarget;
                  handle.classList.add("dragging");
                  const move = (ev) => {
                    const r = view.getBoundingClientRect();
                    const pct = Math.min(80, Math.max(20, Math.round(((r.right - ev.clientX) / r.width) * 100)));
                    setPaneRatios((p) => ({ ...p, [selectedId]: pct }));
                  };
                  const up = () => {
                    handle.classList.remove("dragging");
                    window.removeEventListener("pointermove", move);
                    window.removeEventListener("pointerup", up);
                  };
                  window.addEventListener("pointermove", move);
                  window.addEventListener("pointerup", up);
                }}
              />
              <div className="agent-split-pane">
                <WebTabSurface
                  key={"split-" + selectedId}
                  tabId={"w:" + agentPanes[selectedId]}
                  boundSession={selectedId}
                  active={true}
                  hidden={noTabs || missing || isFileTab(selectedId) || isGitTab(selectedId) || isTreeTab(selectedId) || isInstructionsTab(selectedId) || isAppTab(selectedId) || isWebTab(selectedId) || onPane}
                  expanded={!!paneMax[selectedId]}
                  asks={permissionAsks.filter((a) => a.tab === agentPanes[selectedId])}
                  onAnswerAsk={answerPermission}
                  onBrowserSettings={() => go("browser")}
                  onToggleExpand={() => setPaneMax((p) => ({ ...p, [selectedId]: !p[selectedId] }))}
                  onClose={() => closeAgentSplit(selectedId)}
                  onMeta={(m) => {
                    const wid = agentPanes[selectedId];
                    setWebTabs((cur) => ({ ...cur, [wid]: { ...cur[wid], ...m } }));
                    // The pane's url is what a relaunch navigates back to; the
                    // 800 ms meta poll repeats it, so only a change writes.
                    if (m.url && splitUrlsRef.current[wid] !== m.url) {
                      splitUrlsRef.current = { ...splitUrlsRef.current, [wid]: m.url };
                      writeAgentSplitUrls(splitUrlsRef.current);
                    }
                  }}
                />
              </div>
            </>
          ) : null}

          {termView && !onPane && agent ? (
            agent.mode !== "managed" && (agent.mode === "interactive" || agent.terminalId) ? (
              (() => {
                const resolved = resolveInteractiveTerminal(agent, terminals, selected && selected.path);
                if (!resolved) return <section className="term-surface"><p className="file-pane-msg">Terminal unavailable. <button type="button" className="btn btn-sm" onClick={() => openInteractive(agent.id)}>Open terminal</button></p></section>;
                const term = resolved.term;
                const termId = resolved.id;
                return (
                  <TermSurface
                    key={"agterm-" + termId + "-" + (termEpochs[agent.id] || 0)}
                    term={term}
                    tabId={agent.id}
                    cwdKind={resolved.cwdKind}
                    onOpenFile={(p) => openFileTab(resolved.canonical ? "term" : "agent", termId, p)}
                    onOpenLink={openTermLink}
                    attach={termAttach && termAttach.id === termId ? termAttach : null}
                    onAttachClose={() => setTermAttach(null)}
                    find={termFind === termId}
                    onFindClose={() => { setTermFind(""); focusPane(termId); }}
                    onPasteFiles={openTermAttachFiles}
                    promptDoor={promptDoorFor({ kind: "agent", agentMode: agent.mode, term })}
                  />
                );
              })()
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

        <AgentClis catalog={catalog} onCatalogChange={setCatalog} legacyContextReady={bootstrapped} legacyPackageContext={{ workspaceId: paneWs?.id || "", agentId: agent?.id || (isAgentTab(selectedId) ? selectedId : "") }} packageUpdates={pkgUpdates} onPackageUpdates={(updates, workspaceId) => { if ((paneWs?.id || "") === workspaceId) setPkgUpdates(updates); }} legacyAgentId={agent?.id || (isAgentTab(selectedId) ? selectedId : "")} onAgentConfig={(target, cfg) => patchAgent(cfg, target, false)} hidden={route !== "clis"} onOpenAgent={(id) => revealAgent(id)} onCompactAgent={compactAgentById} onRenameTerm={renameTerminal} onContinueTerm={openTermHandoff} onReloadAgent={async (id) => {
            const loc = locate(workspaces, freeAgents, id);
            const target = loc && loc.agent;
            if (!target || target.mode === "stopped") return;
            const was = target.mode;
            await stopAgent(target.id);
            if (was === "interactive") await openInteractive(target.id);
            else await startManaged(target.id);
          }} />
        <Settings
          hidden={route !== "preferences"}
          themeMode={themeMode}
          onTheme={setTheme}
          workspaces={workspaces}
          workspacesLoaded={fleetLoaded}
        />
        <System hidden={route !== "system"} version={version} system={system} clis={clis} clisState={clisState} />
        {route === "llama" ? <LlamaPanel onRefresh={async () => { try { setCatalog(await api("/api/catalog")); } catch { /* pi missing */ } }} /> : null}
        <Integrations hidden={route !== "integrations"} />
        <Devices hidden={route !== "devices"} />
        <BrowserPage hidden={route !== "browser"} onCreateAgent={() => { selectSideTab("agents"); go("workspace"); setCliPrincipalWs({ free: true }); }} />
        <ComputerPage hidden={route !== "computer"} onCreateAgent={() => { selectSideTab("agents"); go("workspace"); setCliPrincipalWs({ free: true }); }} />
        <Automations hidden={route !== "automations"} catalog={catalog} workspaces={workspaces} freeAgents={freeAgents} system={system} clis={clis} clisLoaded={clisState === "ok"} />
        <Snippets hidden={route !== "snippets"} />
        <Outcomes hidden={route !== "outcomes"} workspaces={workspaces} />
        <TermSettingsPage hidden={route !== "termset"} terminals={terminals} />
        {route === "pins" ? <Suspense fallback={null}><PinStudio /></Suspense> : null}
      </main>
      <Inspector
        hidden={onPane || showHome || !inspectorLayout.shown}
        anchor={inspectorAnchor}
        workspaces={workspaces}
        freeAgents={freeAgents}
        terminals={terminals}
        touchedPaths={inspectorAnchor && inspectorAnchor.kind === "agent" && agent && agent.id === inspectorAnchor.id ? touchedPaths : null}
        tab={inspectorPrefs.tab}
        onTab={(tab) => rememberInspector({ tab })}
        width={inspectorLayout.width}
        maxWidth={inspectorLayout.maxWidth}
        onWidth={(width) => rememberInspector({ width })}
        activePath={inspectorActivePath}
        activeWorktree={(fileTabWT && fileTabWT.ref) || ""}
        onOpenFile={(o, path) => openFileTab(o.kind, o.id, path, "file")}
        onOpenDiff={(o, path, wt) => openFileTab(o.kind, o.id, path, "diff", wt || null)}
        onOpenGraph={(o, name) => openGitTab(o.kind, o.id, name)}
        onOpenTree={(o, name) => openTreeTab(o.kind, o.id, name)}
        onOpenTerminal={typeIntoTerminal}
        onChanges={setInspectorChanged}
        runMode={!!inspectorPrefs.run}
        onRunMode={(run) => rememberInspector({ run })}
        onAskAgent={askAgentGit}
        onOpenUrl={(url, title) => {
          const id = openWebTab(url || "");
          if (title) setWebTabs((m) => ({ ...m, [tabWebId(id)]: { ...m[tabWebId(id)], title } }));
        }}
        onOpenOwner={(owner) => {
          // The Servers panel's "Show terminal": the row already answered
          // whose process holds the port, so this only brings that owner's
          // tab forward — no lookup, no guessing by name.
          if (!owner || !owner.id) return;
          if (owner.kind === "agent") { revealAgent(owner.id); return; }
          openTermTab(owner.id);
          if (parseRoute() !== "workspace") location.hash = termHash(owner.id);
        }}
      />

      <FocusEdges zones={focus.zones} />

      <Palette
        open={paletteOpen}
        workspaces={workspaces}
        apps={apps}
        snips={snipPicker}
        agentId={!isTermTab(selectedId) ? (selectedId || "") : ""}
        focusable={focusOk}
        onClose={() => setPaletteOpen(false)}
        onRun={(a) => {
          if (a.kind === "whats-new") { openWhatsNew(); return; }
          if (a.kind === "inspector") { toggleInspector(); return; }
          if (a.kind === "fullscreen") { focus.toggle(); return; }
          if (a.kind === "cli-new") {
            if (a.wsId) {
              const ws = workspacesRef.current.find((w) => w.id === a.wsId);
              if (ws && ws.id !== "ws_free") { setCliPrincipalWs(ws); return; }
            }
            setCliPrincipalWs({ free: true }); // ADR-0184: a launch is an agent
            return;
          }
          if (a.kind === "settings" || a.kind === "preferences" || a.kind === "clis" || a.kind === "system" || a.kind === "providers" || a.kind === "mcps" || a.kind === "connectors" || a.kind === "integrations" || a.kind === "packages" || a.kind === "skills" || a.kind === "devices" || a.kind === "automations" || a.kind === "snippets" || a.kind === "outcomes") { go(a.kind, ctxAgent?.id, { workspaceId: paneWs?.id, cli: ctxAgent?.cli }); return; }
          if (a.kind === "snip-run") {
            const loc = locate(workspacesRef.current, freeAgentsRef.current, a.target && a.target.id);
            const via = loc && loc.agent && loc.agent.mode === "interactive" ? "tui" : undefined;
            setSnipRun({ snipId: a.snipId, target: a.target, via });
            return;
          }
          if (a.kind === "app") { openTab(appTabId(a.appId)); if (parseRoute() !== "workspace") location.hash = appHash(a.appId); return; }
          if (a.kind === "open") revealAgent(a.wsId);
          if (a.kind === "files") openTreeTab("workspace", a.wsId, a.wsName);
          if (a.kind === "run") startManaged(a.wsId);
          if (a.kind === "term") openInteractive(a.wsId);
          if (a.kind === "stop") stopAgent(a.wsId);
        }}
      />
      <SnipRunSheet
        open={!!snipRun}
        mode="run"
        snipId={snipRun && snipRun.snipId}
        target={snipRun && snipRun.target}
        targetName={snipRun && snipRun.targetName}
        onlyKind={snipRun && snipRun.onlyKind}
        via={snipRun && snipRun.via}
        onRan={() => setSnipRun(null)}
        onClose={() => setSnipRun(null)}
      />
      <SnipCaptureSheet
        open={snipCapture != null}
        text={snipCapture || ""}
        onClose={() => setSnipCapture(null)}
        onSaved={() => setSnipCapture(null)}
      />
      <ContextMenu
        state={ctxMenu}
        onClose={() => setCtxMenu(null)}
        themeMode={themeMode}
        onTheme={setTheme}
        focusOn={focus.on}
        focusable={focusOk}
        onFullscreen={focus.toggle}
        termHandlers={termMenuHandlers}
        clis={clis}
        onOpenAgent={revealAgent}
        onOpenTerminal={openTermTab}
        onGraphAction={openGraphAction}
        onSaveSnippet={openSnipCapture}
      />
      <SessionHandoffDialog
        open={!!termHandoff}
        session={termHandoff ? termHandoff.session : null}
        sourceCli={termHandoff ? termHandoff.sourceCli : ""}
        sourceName={termHandoff ? termHandoff.sourceName : ""}
        target={termHandoff ? termHandoff.target : null}
        onClose={() => setTermHandoff(null)}
        onDone={onTermHandoffDone}
      />

      <ForkAgentDialog
        open={!!forkSource}
        agent={forkSource ? forkSource.agent : null}
        cliName={forkSource ? forkSource.cliName : ""}
        deliverGit={(command, root) => typeIntoTerminal({ kind: "agent", id: forkSource.agent.id }, root, command, { run: true, quiet: true, prepareNote: false })}
        onClose={() => setForkSource(null)}
        onDone={onForkAgentDone}
        onBackground={(name) => {
          // Another agent is writing the repository, so the worktree command
          // waits for the person's Enter (ADR-0078). The git terminal is the
          // open tab; the dialog steps aside so the key can reach it.
          setForkSource(null);
          toast.info("Press Enter in the git terminal to create the worktree. " + (name || "The fork") + " starts when it appears.");
        }}
        onBackgroundError={(e) => toastError(e)}
      />

      <GitActionDialog
        open={!!gitAction}
        owner={gitAction ? gitAction.owner : null}
        root={gitAction ? gitAction.root : ""}
        item={gitAction ? gitAction.item : null}
        agents={gitAction ? gitAction.agents : []}
        run={!!inspectorPrefs.run}
        onClose={() => setGitAction(null)}
        onDeliver={async (command, opts) => {
          const { owner, root, item, ref } = gitAction;
          const token = await gitTokenBefore(owner);
          // A failure is thrown into the form, which stays open and shows it;
          // nothing below runs, so the graph never says "Sent" over a 409.
          await typeIntoTerminal(owner, root, command, { run: opts.run, quiet: true });
          // The graph learns the outcome by watching, not by return value:
          // the command was typed or submitted, not finished (ADR-0096).
          const before = { head: opts.head || "", ref };
          setGitActionDone({ ownerId: ownerIdOf(owner), verb: opts.verb, door: opts.run ? "run" : "prepare", token, undo: undoFor(item.action, before) });
          setGitActionTick((n) => n + 1);
          if (opts.alsoAgent && item.action === "create-worktree") {
            startAgentInWorktree(owner, opts.name, opts.agentName);
          }
        }}
        onAsk={async (who, text, action, verb, head) => {
          const { owner, root, item, ref } = gitAction;
          const token = await gitTokenBefore(owner);
          await askAgentGit(who, text, root, action, verb, { quiet: true });
          setGitActionDone({ ownerId: ownerIdOf(owner), verb, door: "ask", token, undo: undoFor(item.action, { head: head || "", ref }) });
          setGitActionTick((n) => n + 1);
        }}
      />
      <Toasts />
      <CreateForm
        open={showForm}
        kind={formKind}
        workspaceName={(workspaces.find((w) => w.id === formWs) || {}).name}
        catalog={catalog}
        cfg={newCfg}
        onCfg={setNewCfg}
        error={formError}
        onSubmit={submitNew}
        onClose={() => { setShowForm(false); setFormError(""); }}
        busy={formBusy}
      />
      <NewCliPrincipal
        open={!!cliPrincipalWs}
        workspace={cliPrincipalWs}
        onClose={() => setCliPrincipalWs(null)}
        onCreated={async (created) => {
          setCliPrincipalWs(null);
          if (!created || !created.id) return;
          if (agentIsPi(created) || !created.terminalId) {
            await refreshFleetFallback();
            openTab(created.id);
            return;
          }
          try {
            await api("/api/terminals/" + encodeURIComponent(created.terminalId) + "/launch/start", {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ confirm: false }),
            });
          } catch (err) {
            toastError(err);
          }
          await openTermTab(created.terminalId);
          location.hash = termHash(created.terminalId);
        }}
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

      <ShareGist open={shareOpen} gist={shareLinks.gist} viewer={shareLinks.viewer} onClose={() => setShareOpen(false)} />
      <Hotkeys open={hotkeysOpen} onClose={() => setHotkeysOpen(false)} />
      {reconnect ? <Reconnect onReload={() => location.reload()} /> : null}
      <Changelog open={changelogOpen} onClose={() => setChangelogOpen(false)} />
      <WhatsNew open={whatsNewOpen} onClose={closeWhatsNew} currentSemver={whatsNewCurrent} seenVersion={whatsNewSeen} notes={RELEASE_NOTES} unseenOnly={whatsNewMode === "auto"} />
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
