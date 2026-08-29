import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { api, humanizeError, wsURL } from "../lib/api.js";
import { bashLine } from "../lib/bashLine.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { applyTermChrome } from "../lib/termTheme.js";
import { closeTerm } from "../components/TerminalDock.jsx";
import { closeShellTerm } from "../components/ShellTerm.jsx";
import { summarizeArgs } from "../components/Conversation.jsx";
import { fileChangeFromTool } from "../lib/diff.js";
import { eventsToItems } from "../lib/replay.js";
import Sidebar from "../components/Sidebar.jsx";
import AgentTabs from "../components/AgentTabs.jsx";
import SessionBar from "../components/SessionBar.jsx";
import ChatSurface from "../components/ChatSurface.jsx";
import TermSurface from "../components/TermSurface.jsx";
import FileSurface from "../components/FileSurface.jsx";
import TerminalDock from "../components/TerminalDock.jsx";
import Settings from "../components/Settings.jsx";
import PiSettings from "../components/PiSettings.jsx";
import System from "../components/System.jsx";
import Providers from "../components/Providers.jsx";
import Mcps from "../components/Mcps.jsx";
import Packages from "../components/Packages.jsx";
import Devices from "../components/Devices.jsx";
import Palette from "../components/Palette.jsx";
import SessionTree from "../components/SessionTree.jsx";
import SessionInfo from "../components/SessionInfo.jsx";
import CreateForm from "../components/CreateForm.jsx";
import { parseRoute, go, providersNew, providersLlama, agentRoute, workspaceHash, termRoute, termHash, termTabId, isTermTab, tabTermId, fileRoute, fileHash, fileTabId, isFileTab, parseFileTab } from "../lib/routes.js";
const PinStudio = lazy(() => import("../components/PinStudio.jsx"));
import { startPresence } from "../lib/device.js";
import { startReconnectWatch } from "../lib/reconnect.js";
import Reconnect from "../components/Reconnect.jsx";
import { setShell } from "../lib/shell.js";
import { toast, toastError } from "../lib/toast.js";
import { pendingFollowUps, dropQueued, startEditQueued, saveEditQueued, cancelEditQueued } from "../lib/queue.js";
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
import { readOpenTabs, writeOpenTabs, filterOpenTabs, moveTab } from "../lib/openTabs.js";
import Hotkeys from "../components/Hotkeys.jsx";
import Changelog from "../components/Changelog.jsx";
import ShareGist from "../components/ShareGist.jsx";
import LlamaDialog from "../components/LlamaDialog.jsx";
import { createWorkspaceSchema, createFreeAgentSchema, createWsAgentSchema, parseForm } from "../lib/schemas.js";
import Toasts from "../components/Toasts.jsx";

export default function App() {
  const [workspaces, setWorkspaces] = useState([]);
  const [freeAgents, setFreeAgents] = useState([]);
  const [selectedId, setSelectedId] = useState(() => readOpenTabs().selected);
  const [formKind, setFormKind] = useState("workspace");
  const [formWs, setFormWs] = useState("");
  const [tabs, setTabs] = useState(() => readOpenTabs().ids);
  const [tabsReady, setTabsReady] = useState(false);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [host, setHost] = useState("local");
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [route, setRoute] = useState(() => parseRoute());
  const [hash, setHash] = useState(() => (typeof location !== "undefined" ? location.hash : "#/"));
  const [goneId, setGoneId] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [treeOpen, setTreeOpen] = useState(false);
  const [sessionOpen, setSessionOpen] = useState(false);
  const [treeMode, setTreeMode] = useState("tree");
  const [treeData, setTreeData] = useState({ tree: [], leafId: "" });
  const [catalog, setCatalog] = useState({ providers: [], thinking: [] });
  const [newCfg, setNewCfg] = useState({ provider: "", model: "", thinking: "" });
  const [showForm, setShowForm] = useState(false);
  const [formError, setFormError] = useState("");
  const [termWanted, setTermWanted] = useState(() => new Set());
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const draftAgentRef = useRef(null);
  const [status, setStatus] = useState("idle");
  const [streaming, setStreaming] = useState(false);
  const [waiting, setWaiting] = useState(false);
  const streamingRef = useRef(false);
  const waitingRef = useRef(false);
  waitingRef.current = waiting;
  const flushingRef = useRef(false);
  const [items, setItems] = useState([]);
  const itemsRef = useRef([]);
  itemsRef.current = items;
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
  const [pkgUpdates, setPkgUpdates] = useState([]);

  const [terminals, setTerminals] = useState([]);
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
  const stopped = !agent || agent.mode === "stopped";
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
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key === "`") {
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

  const pkgWs = selected ? selected.id : "";
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
    const t = setInterval(load, 30 * 60 * 1000);
    return () => { stop = true; clearInterval(t); };
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
      const pal = (e.ctrlKey || e.metaKey) && !e.shiftKey && e.key.toLowerCase() === "k";
      if (pal) {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

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
        setItems([]);
        return;
      }
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur) + (selectedId ? "&agent=" + encodeURIComponent(selectedId) : ""));
      const ev = t.events || [];
      setItems(ev.length ? eventsToItems(ev) : []);
      scrollToEnd();
    } catch { setSessions([]); setSessionCurrent(""); }
  }, [selectedId, workspaces, freeAgents]);

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

  useEffect(() => { loadSessions(); }, [selectedId, workspaces.length, freeAgents.length]);
  useEffect(() => {
    if (!selectedId) { setSlashExtra([]); return; }
    api("/api/agents/" + selectedId + "/slash")
      .then((d) => setSlashExtra(extraSlash(d.skills, d.templates)))
      .catch(() => setSlashExtra([]));
  }, [selectedId]);
  useEffect(() => { scrollConv(); }, [items]);

  const loadStatus = useCallback(async (wsId) => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setStatusBar(null); return; }
    try { setStatusBar(await api("/api/workspaces/" + id + "/status")); }
    catch { setStatusBar(null); }
  }, [selectedId, workspaces, freeAgents]);

  useEffect(() => { loadStatus(); }, [selectedId, sessionCurrent, loadStatus]);
  useEffect(() => {
    if (!agent || agent.mode !== "managed") return;
    const t = setInterval(() => loadStatus(), 15000);
    return () => clearInterval(t);
  }, [agent && agent.mode, selectedId, loadStatus]);

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
        const exists = (id) => {
          if (isTermTab(id)) return terms.some((t) => t.id === tabTermId(id));
          if (isFileTab(id)) {
            const f = parseFileTab(id);
            if (!f) return false;
            if (f.kind === "term") return terms.some((t) => t.id === f.id);
            return !!locate(list, free, f.id);
          }
          return !!locate(list, free, id);
        };
        const next = filterOpenTabs(readOpenTabs(), exists);
        setTabs(next.ids);
        const fromTerm = parseRoute() === "workspace" ? termRoute() : null;
        const fromFile = parseRoute() === "workspace" ? fileRoute() : null;
        const fromHash = parseRoute() === "workspace" ? agentRoute() : null;
        if (fromTerm) {
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
  useEffect(() => startReconnectWatch({
    onState: (s) => { if (s === "down") setReconnect(true); },
  }), []);
  useEffect(() => {
    if (!tabsReady) return;
    writeOpenTabs(tabs, selectedId);
  }, [tabs, selectedId, tabsReady]);
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
    const want = isTermTab(selectedId)
      ? termHash(tabTermId(selectedId))
      : file
        ? fileHash(file.kind, file.id, file.path)
        : workspaceHash(selectedId);
    if (location.hash === want) return;
    if (!agentRoute(location.hash) && !termRoute(location.hash) && !fileRoute(location.hash) && selectedId) {
      history.replaceState(null, "", want);
      setHash(want);
      return;
    }
    location.hash = want;
  }, [selectedId, tabsReady, goneId, route]);
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
    if (isTermTab(id)) { openTermTab(tabTermId(id)); return; }
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
    if (!ownerId || !path) return;
    const id = fileTabId(kind, ownerId, path);
    setGoneId("");
    setSelectedId(id);
    setTabs((t) => (t.includes(id) ? t : [...t, id]));
  }

  function closeTab(id) {
    if (isTermTab(id)) closeShellTerm(tabTermId(id));
    const ws = workspaces.find((w) => w.id === id);
    setTabs((t) => t.filter((x) => x !== id));
    setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
    if (ws && ws.agent) closeTerm(ws.agent.id);
    if (panelRef.current && ws && ws.agent && panelRef.current.agentId === ws.agent.id) closePanel();
    if (selectedId === id) {
      setTabs((t) => {
        const next = t[t.length - 1];
        if (next) {
          if (isTermTab(next)) openTermTab(tabTermId(next));
          else if (isFileTab(next)) {
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
    const item = {
      kind: "ask",
      id: d.id,
      method: d.method,
      title: d.title || "",
      message: d.message || "",
      options: d.options || [],
      placeholder: d.placeholder || "",
      prefill: d.prefill || "",
      timeout: d.timeout || 0,
      status: status || "open",
      ts: Date.now(),
    };
    setItems((cur) => {
      const i = cur.findIndex((it) => it.kind === "ask" && it.id === item.id);
      if (i >= 0) {
        const next = cur.slice();
        next[i] = { ...cur[i], ...item };
        return next;
      }
      return [...cur, item];
    });
  }

  function connectPanel(ws) {
    closePanel();
    setStatus("idle");
    setStreaming(false);
    streamingRef.current = false;
    setWaiting(false);
    const sock = new WebSocket(wsURL(`/ws/agent?agent=${ws.agent.id}`));
    const panel = { agentId: ws.agent.id, sock, stopped: false };
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
      }
      if (window.__picodeKickHealth) window.__picodeKickHealth();
    };
  }

  function handleEvent(env, panel) {
    const ev = env.event || {};
    switch (ev.type) {
      case "snapshot":
        setStreaming(!!ev.streaming);
        streamingRef.current = !!ev.streaming;
        setWaiting(!!ev.waiting);
        setStatus(ev.waiting ? "waiting" : ev.streaming ? "streaming" : "idle");
        if (ev.waiting && ev.dialog) putAskItem(ev.dialog, "open");
        break;
      case "agent_start":
        setStreaming(true);
        streamingRef.current = true;
        setStatus((s) => (s === "waiting" ? "waiting" : "streaming"));
        scrollToEnd();
        break;
      case "agent_settled": {
        setStreaming(false);
        streamingRef.current = false;
        setStatus((s) => (s === "waiting" ? "waiting" : "idle"));
        if (selectedId) loadStatus();
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
          ts: Date.now(),
        }]);
        queueMicrotask(scrollConv);
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
      case "auto_retry_start":
      case "auto_retry_end":
      case "compaction_end":
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
          setWaiting(true);
          setStatus("waiting");
          putAskItem(ev, "open");
        } else if (method === "notify") {
          const msg = ev.message || "Notice";
          if (ev.notifyType === "error") toastError(msg);
          else toast.info(msg);
        }
        queueMicrotask(scrollConv);
        break;
      }
      case "extension_ui_timeout":
        setWaiting(false);
        setStatus(streamingRef.current ? "streaming" : "idle");
        setItems((cur) => cur.map((it) => (it.kind === "ask" && it.id === ev.id && it.status === "open" ? { ...it, status: "timeout" } : it)));
        break;
      default:
        break;
    }
  }

  useEffect(() => {
    if (!selected || !agent) { closePanel(); return; }
    if (agent.mode === "managed") {
      if (!panelRef.current || panelRef.current.agentId !== agent.id) connectPanel(selected);
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
      const list = await loadWorkspaces();
      openTab(loc.agent.id, list);
    } catch (err) { toastError(err); }
  }

  async function openTermTab(id) {
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

  async function createTerminal() {
    try {
      const page = await api("/api/terminals", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
      setTerminals((cur) => [...cur, page]);
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
      await api(`/api/agents/${loc.agent.id}/open`, { method: "POST" });
      const list = await loadWorkspaces();
      openTab(loc.agent.id, list);
      if (!opts || opts.dock !== false) {
        setTermWanted((s) => new Set(s).add(loc.agent.id));
      }
    } catch (err) { toastError(err); }
  }

  async function stopAgent(id) {
    const loc = locate(workspaces, freeAgents, id);
    if (!loc || !loc.agent) return;
    try {
      await api(`/api/agents/${loc.agent.id}/close`, { method: "POST" });
      closeTerm(loc.agent.id);
      if (panelRef.current && panelRef.current.agentId === loc.agent.id) panelRef.current.stopped = true;
      setStreaming(false);
      streamingRef.current = false;
      setWaiting(false);
      setStatus("stopped");
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function confirmCleanup({ title, message, path }) {
    let preview = { lastOccupant: false, sessions: 0, sessionBytes: 0, canPurgeWork: false };
    try { preview = await api(path); } catch { /* unregister still allowed */ }
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
    const ok = await askConfirm({ title, message, confirmLabel: "Remove", danger: true, choices });
    if (!ok) return null;
    const picked = ok === true ? {} : ok;
    const q = new URLSearchParams();
    if (picked.sessions) q.set("sessions", "1");
    if (picked.work) q.set("work", "1");
    const qs = q.toString();
    return { query: qs ? "?" + qs : "" };
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
      closeTerm(ag.id);
      setTabs((t) => t.filter((x) => x !== ag.id));
      if (selectedId === ag.id) setSelectedId(null);
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function removeWorkspace(ws) {
    const choice = await confirmCleanup({
      title: "Remove workspace",
      message: `Remove "${ws.name}"? The project folder is not deleted.`,
      path: "/api/workspaces/" + ws.id + "/cleanup",
    });
    if (!choice) return;
    try {
      await api("/api/workspaces/" + ws.id + choice.query, { method: "DELETE" });
      const ids = (ws.agents || []).map((a) => a.id);
      if (ws.agent) ids.push(ws.agent.id);
      for (const id of [...new Set(ids)]) {
        closeTerm(id);
        if (panelRef.current && panelRef.current.agentId === id) closePanel();
      }
      setTabs((t) => t.filter((x) => x !== ws.id && !ids.includes(x)));
      if (selectedId === ws.id || ids.includes(selectedId)) setSelectedId(null);
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function submitNew(e) {
    e.preventDefault();
    setFormError("");
    const fd = new FormData(e.target);
    const name = String(fd.get("name") || "");
    const path = String(fd.get("path") || "");
    const schema = formKind === "workspace" ? createWorkspaceSchema : formKind === "free" ? createFreeAgentSchema : createWsAgentSchema;
    const parsed = parseForm(schema, { name, path, ...newCfg });
    if (!parsed.ok) { setFormError(parsed.error); return; }
    const body = parsed.value;
    try {
      if (formKind === "workspace") {
        const ws = await api("/api/workspaces", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        const list = await loadWorkspaces();
        const aid = (ws.agents && ws.agents[0] && ws.agents[0].id) || (ws.agent && ws.agent.id);
        if (aid) openTab(aid, list);
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
        await loadWorkspaces();
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
      await loadWorkspaces();
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

  async function compactSession() {
    if (!agent) return;
    const ok = await askConfirm({
      title: "Compact session",
      message: "Older turns become a summary. This cannot be undone in the chat.",
      confirmLabel: "Compact",
    });
    if (!ok) return;
    try {
      if (selectedId) setTermWanted((s) => { const n = new Set(s); n.delete(selectedId); return n; });
      const res = await api("/api/agents/" + agent.id + "/compact", { method: "POST" });
      toast.ok(res && res.already ? "Nothing left to compact." : "Session compacted.");
      await loadWorkspaces();
      await loadSessions(selectedId);
      await loadStatus();
    } catch (e) { toastError(e); }
  }

  async function runBash(command) {
    if (!agent) return;
    const itemId = "bash-" + Date.now();
    try {
      try {
        await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
      } catch { /* already running or start failed; the bash call will say */ }
      if (!panelRef.current || panelRef.current.agentId !== agent.id || (panelRef.current.sock && panelRef.current.sock.readyState !== 1)) {
        const loc = locate(workspaces, freeAgents, agent.id);
        if (loc) connectPanel(loc);
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
    if (!agent) return;
    setItems((cur) => cur.map((it) => (
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
    const answer = cancelled ? "Cancelled"
      : body.confirmed === true ? "Yes"
      : body.confirmed === false ? "No"
      : (body.value || "Answered");
    setItems((cur) => cur.map((it) => (it.kind === "ask" && it.id === askId && it.status === "open"
      ? { ...it, status: cancelled ? "cancelled" : "answered", answer }
      : it)));
    setWaiting(false);
    waitingRef.current = false;
    setStatus(streamingRef.current ? "streaming" : "idle");
    queueMicrotask(() => flushFollowUp());
    try {
      await api("/api/agents/" + agent.id + "/ui", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: askId, ...body }),
      });
    } catch (e) {
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
    api("/api/agents/" + id + "/prompt", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).then(() => {
      setItems((cur) => cur.map((it) => (it.qid === next.qid ? { ...it, pending: false, chip: "prompt" } : it)));
    }).catch((e) => toastError(e)).finally(() => { flushingRef.current = false; });
  }

  async function sendTask(text, images) {
    const payload = (typeof text === "string" ? text : draft).trim();
    const pics = images || [];
    if ((!payload && !pics.length) || !agent) return;
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
          text: payload, images: pics.map((p) => p.url),
          queueImages: pics.map((p) => ({ mimeType: p.mime, data: p.data })),
          ts: Date.now(),
        }]);
        setDraft("");
        clearDraft(agent.id);
        pendingPayload.current = "";
        scrollToEnd();
        return;
      }
      try {
        await api("/api/agents/" + agent.id + "/managed/start", { method: "POST" });
      } catch { /* already running or start failed; enqueue still */ }
      if (!panelRef.current || panelRef.current.agentId !== agent.id || (panelRef.current.sock && panelRef.current.sock.readyState !== 1)) {
        const loc = locate(workspaces, freeAgents, agent.id);
        if (loc) connectPanel(loc);
      }
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
      setItems((cur) => [...cur, { kind: "block", cls: "user", actor: "You", chip: sendKind, text: payload, images: pics.map((p) => p.url), ts: Date.now() }]);
      setDraft("");
      clearDraft(agent.id);
      pendingPayload.current = "";
      scrollToEnd();
      if (agent.mode === "interactive") {
        setTermWanted((s) => { const n = new Set(s); n.delete(agent.id); return n; });
      }
    } catch (e) { toastError(e); }
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
      await loadWorkspaces();
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
      await loadWorkspaces();
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
  const missing = !!goneId;
  const noTabs = tabs.length === 0 && !missing;

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
        freeAgents={freeAgents}
        workingId={(streaming || waiting) ? selectedId : null}
        waitingId={waiting ? selectedId : null}
        termView={termView}
        terminals={terminals}
        onNewTerm={createTerminal}
        onSelectTerm={(id) => { openTermTab(id); if (parseRoute() !== "workspace") location.hash = termHash(id); }}
        onRemoveTerm={removeTerminal}
        onRenameTerm={renameTerminal}
        onChat={(id) => {
          revealAgent(id);
          setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
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
        <div id="workspace-view" className={"workspace-view" + (isTermTab(selectedId) ? " term-on" : "") + (isFileTab(selectedId) ? " file-on" : "")} hidden={onPane}>
          <AgentTabs
            tabs={tabs}
            workspaces={workspaces}
            freeAgents={freeAgents}
            terminals={terminals}
            selectedId={selectedId}
            onSelect={(id) => openTab(id)}
            onClose={closeTab}
            onReorder={(from, to) => setTabs((t) => moveTab(t, from, to))}
          />

          <div id="empty" className="empty" hidden={!noTabs && !missing}>
            <div className="empty-card">
              {missing ? (
                <>
                  <h2>{isFileTab(goneId) ? "That file is gone." : isTermTab(goneId) ? "That terminal is gone." : "That agent is gone."}</h2>
                  {(workspaces.length + freeAgents.length + terminals.length) > 0 ? (
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
          <ChatSurface
            hidden={noTabs || missing || termView || isTermTab(selectedId) || isFileTab(selectedId)}
            stopped={stopped}
            items={items}
            onToggleTool={(id) => setItems((cur) => cur.map((it) => it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it))}
            onToggleFiles={(idx) => setItems((cur) => cur.map((it, i) => i === idx && it.kind === "files" ? { ...it, expanded: !it.expanded } : it))}
            convRef={convRef}
            onScroll={() => {
              const el = convRef.current;
              if (el) nearBottom.current = stuckToBottom(el);
            }}
            statusBar={statusBar}
            onCompact={compactSession}
            onAbortBash={abortBash}
            onReplyAsk={replyAsk}
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
                    await loadWorkspaces();
                    await loadSessions();
                  } catch (e) { toastError(e); }
                };
                inp.click();
                return;
              }
            }}
            composer={{
              kind, onKind: setKind, value: draft, onChange: setDraft, onSend: sendTask,
              slashExtra, atAgents, onAgentPage: go, pkgUpdates,
              status, streaming, waiting, onToggleDock: showTerm, onStop: () => selectedId && stopAgent(selectedId),
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
                      await loadWorkspaces();
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
                      await loadWorkspaces();
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

          <TerminalDock
            open={termView && !onPane}
            agent={agent}
            workspace={selected}
            onOpenFile={(p) => { if (agent) openFileTab("agent", agent.id, p); }}
          />
        </div>

        <PiSettings hidden={route !== "settings"} agent={agent} workspace={selected} catalog={catalog} onAgentConfig={patchAgent} />
        <Settings
          hidden={route !== "preferences"}
          themeMode={themeMode}
          onTheme={setTheme}
        />
        <System hidden={route !== "system"} version={version} system={system} />
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
          workspaceId={selected ? selected.id : ""}
          workspaceName={selected ? selected.name : ""}
          workspacePath={selected ? selected.path : ""}
          agentId={selectedId || ""}
          agentName={displayAgentName(agent, selected)}
          agentWorkPath={agent && agent.workPath ? agent.workPath : ""}
          agentRunning={!!(agent && agent.mode && agent.mode !== "stopped")}
          onReload={async () => {
            if (!agent || agent.mode === "stopped") return;
            const was = agent.mode;
            await stopAgent(selectedId);
            if (was === "interactive") await openInteractive(selectedId);
            else await startManaged(selectedId);
          }}
        />
        <Packages hidden={route !== "packages"} workspaceId={selected ? selected.id : ""} workspaceName={selected ? selected.name : ""} workspacePath={selected ? selected.path : ""} agentId={selectedId || ""} agentName={displayAgentName(agent, selected)} updates={pkgUpdates} onUpdates={setPkgUpdates} />
        <Devices hidden={route !== "devices"} />
        {route === "pins" ? <Suspense fallback={null}><PinStudio /></Suspense> : null}
      </main>

      <Palette
        open={paletteOpen}
        workspaces={workspaces}
        onClose={() => setPaletteOpen(false)}
        onRun={(a) => {
          if (a.kind === "settings" || a.kind === "preferences" || a.kind === "system" || a.kind === "providers" || a.kind === "mcps" || a.kind === "packages" || a.kind === "devices") { go(a.kind); return; }
          if (a.kind === "open") revealAgent(a.wsId);
          if (a.kind === "run") startManaged(a.wsId);
          if (a.kind === "term") openInteractive(a.wsId);
          if (a.kind === "stop") stopAgent(a.wsId);
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
