import { useCallback, useEffect, useRef, useState } from "react";
import { api, humanizeError, wsURL } from "../lib/api.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { closeTerm } from "../components/TerminalDock.jsx";
import { summarizeArgs } from "../components/Conversation.jsx";
import { fileChangeFromTool } from "../lib/diff.js";
import { eventsToItems } from "../lib/replay.js";
import Sidebar from "../components/Sidebar.jsx";
import AgentTabs from "../components/AgentTabs.jsx";
import SessionBar from "../components/SessionBar.jsx";
import ChatSurface from "../components/ChatSurface.jsx";
import TerminalDock from "../components/TerminalDock.jsx";
import Settings from "../components/Settings.jsx";
import PiSettings from "../components/PiSettings.jsx";
import System from "../components/System.jsx";
import Providers from "../components/Providers.jsx";
import Mcps from "../components/Mcps.jsx";
import Packages from "../components/Packages.jsx";
import Devices from "../components/Devices.jsx";
import Palette from "../components/Palette.jsx";
import { parseRoute, go } from "../lib/routes.js";
import { startPresence } from "../lib/device.js";
import { setShell } from "../lib/shell.js";
import { toast, toastError } from "../lib/toast.js";
import { askConfirm } from "../lib/confirm.js";
import { stuckToBottom, pinToBottom } from "../lib/stickScroll.js";
import { alertFromPi } from "../lib/piError.js";
import ConfirmDialog from "../components/ConfirmDialog.jsx";
import PromptDialog from "../components/PromptDialog.jsx";
import { askPrompt } from "../lib/prompt.js";
import { locate, firstAgentId } from "../lib/tree.js";
import Toasts from "../components/Toasts.jsx";

export default function App() {
  const [workspaces, setWorkspaces] = useState([]);
  const [freeAgents, setFreeAgents] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [formKind, setFormKind] = useState("workspace");
  const [formWs, setFormWs] = useState("");
  const [tabs, setTabs] = useState([]);
  const [system, setSystem] = useState(null);
  const [version, setVersion] = useState("");
  const [host, setHost] = useState("local");
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [route, setRoute] = useState(() => parseRoute());
  const [menuOpen, setMenuOpen] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [catalog, setCatalog] = useState({ providers: [], thinking: [] });
  const [mcp, setMcp] = useState({ configured: false, path: "" });
  const [newCfg, setNewCfg] = useState({ provider: "", model: "", thinking: "" });
  const [showForm, setShowForm] = useState(false);
  const [formError, setFormError] = useState("");
  const [termWanted, setTermWanted] = useState(() => new Set());
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const [status, setStatus] = useState("idle");
  const [streaming, setStreaming] = useState(false);
  const [items, setItems] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [sessionCurrent, setSessionCurrent] = useState("");
  const [statusBar, setStatusBar] = useState(null);
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const panelRef = useRef(null);
  const pendingPayload = useRef("");
  const turnFiles = useRef(new Set());
  const selectedRef = useRef(null);
  selectedRef.current = selectedId;

  const located = locate(workspaces, freeAgents, selectedId);
  const selected = located && located.workspace;
  const agent = located && located.agent;
  const stopped = !agent || agent.mode === "stopped";
  const interactive = !!(agent && agent.mode === "interactive");
  const termView = !!(selectedId && termWanted.has(selectedId));

  useEffect(() => { applyTheme(themeMode); }, [themeMode]);
  useEffect(() => {
    const mq = matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => { if (themeMode === "system") applyTheme("system"); };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [themeMode]);

  useEffect(() => {
    const onHash = () => {
      setRoute(parseRoute());
      setMenuOpen(false);
    };
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  useEffect(() => {
    const onDoc = (e) => { if (!e.target.closest("#usermenu")) setMenuOpen(false); };
    const onKey = (e) => {
      if (e.key === "Escape") setMenuOpen(false);
      const pal = (e.ctrlKey || e.metaKey) && !e.shiftKey && e.key.toLowerCase() === "k";
      if (pal) {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    };
    document.addEventListener("click", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("click", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, []);

  const loadWorkspaces = useCallback(async () => {
    const list = await api("/api/workspaces");
    setWorkspaces(list);
    try { setFreeAgents(await api("/api/agents?free=1")); }
    catch { setFreeAgents([]); }
    return list;
  }, []);

  const loadSessions = useCallback(async (wsId) => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setSessions([]); setSessionCurrent(""); return; }
    try {
      const data = await api("/api/workspaces/" + id + "/sessions");
      setSessions(data.sessions || []);
      const cur = data.current || "";
      setSessionCurrent(cur);
      if (!cur) {
        setItems([]);
        return;
      }
      const t = await api("/api/workspaces/" + id + "/sessions/transcript?path=" + encodeURIComponent(cur));
      const ev = t.events || [];
      setItems(ev.length ? eventsToItems(ev) : []);
      scrollToEnd();
    } catch { setSessions([]); setSessionCurrent(""); }
  }, [selectedId, workspaces, freeAgents]);

  useEffect(() => { loadSessions(); }, [selectedId, loadSessions]);
  useEffect(() => { scrollConv(); }, [items]);

  const loadStatus = useCallback(async (wsId) => {
    const loc = locate(workspaces, freeAgents, selectedId);
    const id = wsId || (loc && loc.workspace && loc.workspace.id) || (loc && loc.agent ? "ws_free" : null);
    if (!id) { setStatusBar(null); return; }
    try { setStatusBar(await api("/api/workspaces/" + id + "/status")); }
    catch { setStatusBar(null); }
  }, [selectedId, workspaces, freeAgents]);

  useEffect(() => { loadStatus(selectedId); }, [selectedId, sessionCurrent, loadStatus]);
  useEffect(() => {
    if (!agent || agent.mode !== "managed") return;
    const t = setInterval(() => loadStatus(selectedId), 15000);
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
      try { setMcp(await api("/api/mcp")); } catch { /* ignore */ }
      try {
        const list = await loadWorkspaces();
        const active = list.find((w) => w.agent && w.agent.mode !== "stopped") || list[0];
        if (active) openTab(active.id, list);
      } catch (e) {
        console.error("boot:", e);
      }
    })();
  }, [loadWorkspaces]);

  useEffect(() => startPresence(), []);

  function openTab(id, list) {
    const loc = locate(list || workspaces, freeAgents, id);
    const aid = loc && loc.agent ? loc.agent.id : id;
    setSelectedId(aid);
    setTabs((t) => t.includes(aid) ? t : [...t, aid]);
    if (loc) prepareSurface(loc.agent);
  }

  function prepareSurface(a) {
    if (!a || a.mode === "stopped") {
      setStatus("stopped");
      setStreaming(false);
      return;
    }
    if (a.mode === "interactive") {
      setStatus("interactive");
      setStreaming(false);
    }
  }

  function closeTab(id) {
    const ws = workspaces.find((w) => w.id === id);
    setTabs((t) => t.filter((x) => x !== id));
    setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
    if (ws && ws.agent) closeTerm(ws.agent.id);
    if (panelRef.current && ws && ws.agent && panelRef.current.agentId === ws.agent.id) closePanel();
    if (selectedId === id) {
      setTabs((t) => {
        const next = t[t.length - 1];
        if (next) {
          setSelectedId(next);
          const ws2 = workspaces.find((w) => w.id === next);
          if (ws2) prepareSurface(ws2);
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

  function connectPanel(ws) {
    closePanel();
    setStatus("idle");
    setStreaming(false);
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
        setItems((cur) => [...cur, { kind: "sys", text: "— panel disconnected —", err: true }]);
      }
    };
  }

  function handleEvent(env, panel) {
    const ev = env.event || {};
    switch (ev.type) {
      case "snapshot":
        setStatus(ev.streaming ? "streaming" : "idle");
        setStreaming(!!ev.streaming);
        break;
      case "agent_start":
        setStatus("streaming");
        setStreaming(true);
        turnFiles.current = new Set();
        scrollToEnd();
        break;
      case "agent_settled": {
        setStatus("idle");
        setStreaming(false);
        const paths = [...turnFiles.current];
        turnFiles.current = new Set();
        if (paths.length) {
          setItems((cur) => [...cur, { kind: "files", paths, expanded: false }]);
        }
        if (selectedId) loadStatus(selectedId);
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
        if (change) turnFiles.current.add(change.path);
        setItems((cur) => [...cur, {
          kind: "tool",
          id: ev.toolCallId,
          name: ev.toolName || "tool",
          args: summarizeArgs(ev.args),
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
          if (change) turnFiles.current.add(change.path);
          return {
            ...it,
            status: ev.isError ? "error" : "ok",
            detail: JSON.stringify(ev.result || {}, null, 2),
            change,
          };
        }));
        break;
      case "enqueue_accepted": {
        const text = pendingPayload.current;
        pendingPayload.current = "";
        setDraft("");
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
      case "message_end":
      case "turn_end":
      case "agent_end":
      case "auto_retry_start":
      case "auto_retry_end":
      case "compaction_end":
      case "extension_error": {
        const a = alertFromPi(ev);
        if (a) {
          setItems((cur) => [...cur, { kind: "alert", level: a.level, text: a.text, ts: Date.now() }]);
          if (a.level === "error" && ev.type !== "auto_retry_start") setStreaming(false);
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
      setStatus("stopped");
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function removeAgent(ag) {
    const ok = await askConfirm({
      title: "Remove agent",
      message: `Remove "${ag.name}"? The project folder is not deleted.`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await api("/api/agents/" + ag.id, { method: "DELETE" });
      closeTerm(ag.id);
      setTabs((t) => t.filter((x) => x !== ag.id));
      if (selectedId === ag.id) setSelectedId(null);
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function removeWorkspace(ws) {
    const ok = await askConfirm({
      title: "Remove workspace",
      message: `Remove "${ws.name}"? The project folder is not deleted.`,
      confirmLabel: "Remove",
      danger: true,
    });
    if (!ok) return;
    try {
      await api(`/api/workspaces/${ws.id}`, { method: "DELETE" });
      if (ws.agent) closeTerm(ws.agent.id);
      if (panelRef.current && ws.agent && panelRef.current.agentId === ws.agent.id) closePanel();
      setTabs((t) => t.filter((x) => x !== ws.id));
      if (selectedId === ws.id) setSelectedId(null);
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function submitNew(e) {
    e.preventDefault();
    setFormError("");
    const fd = new FormData(e.target);
    const name = String(fd.get("name") || "").trim();
    const path = String(fd.get("path") || "").trim();
    if (!newCfg.provider || !newCfg.model || !newCfg.thinking) {
      setFormError("Provider, model, and thinking are required.");
      return;
    }
    try {
      if (formKind === "workspace") {
        if (!name || !path) {
          setFormError("Name and folder path are required.");
          return;
        }
        const ws = await api("/api/workspaces", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, path, ...newCfg }),
        });
        const list = await loadWorkspaces();
        const aid = (ws.agents && ws.agents[0] && ws.agents[0].id) || (ws.agent && ws.agent.id);
        if (aid) openTab(aid, list);
      } else if (formKind === "free") {
        if (!name) { setFormError("Name is required."); return; }
        const ag = await api("/api/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, path, ...newCfg }),
        });
        await loadWorkspaces();
        openTab(ag.id);
      } else {
        if (!name || !formWs) { setFormError("Name is required."); return; }
        const ag = await api("/api/workspaces/" + formWs + "/agents", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, ...newCfg }),
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
      await loadStatus(selectedId);
    } catch (e) { toastError(e); }
  }

  async function abortTurn() {
    if (!agent) return;
    try {
      await api("/api/agents/" + agent.id + "/abort", { method: "POST" });
    } catch (e) { toastError(e); }
  }

  async function sendTask(text) {
    const payload = (typeof text === "string" ? text : draft).trim();
    if (!payload || !agent) return;
    try {
      await api("/api/agents/" + agent.id + "/tasks", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ kind, payload, source: "user" }),
      });
      setItems((cur) => [...cur, { kind: "block", cls: "user", actor: "You", chip: kind, text: payload, ts: Date.now() }]);
      setDraft("");
      pendingPayload.current = "";
      scrollToEnd();
      if (agent.mode === "interactive") {
        setTermWanted((s) => { const n = new Set(s); n.delete(agent.id); return n; });
      }
      if (agent.mode !== "managed") {
        await startManaged(agent.id);
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
  const noTabs = tabs.length === 0;

  return (
    <div id="app">
      <Sidebar
        version={version}
        workspaces={workspaces}
        selectedId={selectedId}
        showForm={showForm}
        formError={formError}
        onNew={() => { setFormKind("workspace"); setShowForm(true); }}
        onNewFree={() => { setFormKind("free"); setShowForm(true); }}
        onNewAgent={(id) => { setFormKind("agent"); setFormWs(id); setShowForm(true); }}
        onCancel={() => { setShowForm(false); setFormError(""); }}
        onSubmit={submitNew}
        onSelect={(id) => openTab(id)}
        onRun={startManaged}
        onStop={stopAgent}
        onRemove={removeWorkspace}
        onRemoveAgent={removeAgent}
        freeAgents={freeAgents}
        formKind={formKind}
        formWs={formWs}
        termView={termView}
        onChat={(id) => {
          openTab(id);
          setTermWanted((s) => { const n = new Set(s); n.delete(id); return n; });
        }}
        onTerm={(id) => {
          openTab(id);
          setTermWanted((s) => new Set(s).add(id));
          const loc = locate(workspaces, freeAgents, id);
          if (loc && loc.agent && loc.agent.mode !== "interactive") openInteractive(id);
        }}
        catalog={catalog}
        newCfg={newCfg}
        onNewCfg={setNewCfg}
        userMenu={{
          open: menuOpen,
          onToggle: () => setMenuOpen((v) => !v),
          onClose: () => setMenuOpen(false),
          host,
          version,
          themeMode,
          onTheme: setTheme,
          onNavigate: (name) => { go(name); setMenuOpen(false); },
        }}
      />

      <main id="main">
        <div id="workspace-view" className="workspace-view" hidden={onPane}>
          <AgentTabs
            tabs={tabs}
            workspaces={workspaces}
            freeAgents={freeAgents}
            selectedId={selectedId}
            onSelect={(id) => openTab(id)}
            onClose={closeTab}
          />

          <div id="empty" className="empty" hidden={!noTabs}>
            <div className="empty-card">
              <h2>No agents yet</h2>
              <p>Add a project folder to create your first agent.</p>
              <button id="btn-new-empty" className="btn btn-primary" onClick={() => setShowForm(true)}>Add workspace</button>
            </div>
          </div>

          <ChatSurface
            hidden={noTabs || termView}
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
            onRun={() => selectedId && startManaged(selectedId)}
            onOpenTerm={() => selectedId && openInteractive(selectedId)}
            catalog={catalog}
            agent={agent}
            onConfig={patchAgent}
            onSlash={async (cmd) => {
              if (cmd.run === "go-settings" || cmd.run === "go-scoped") {
                if (!agent) { toast.info("Select an agent first."); return; }
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
              if (cmd.run === "session-new") {
                try {
                  await api("/api/workspaces/" + selectedId + "/sessions/new", { method: "POST" });
                  await loadWorkspaces();
                  await loadSessions();
                } catch (e) { toastError(e); }
                return;
              }
              if (cmd.run === "session-resume") {
                document.getElementById("session-picker")?.click();
                return;
              }
              if (cmd.run === "compact") { compactSession(); return; }
              if (cmd.run === "session-name") {
                const cur = sessions.find((s) => s.path === sessionCurrent) || { path: sessionCurrent, name: "" };
                if (!cur.path) return;
                const name = await askPrompt({ title: "Rename session", defaultValue: cur.name || "", confirmLabel: "Save" });
                if (!name) return;
                try {
                  await api("/api/workspaces/" + selectedId + "/sessions/rename", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ path: cur.path, name }),
                  });
                  await loadSessions();
                  await loadStatus(selectedId);
                } catch (e) { toastError(e); }
                return;
              }
              try {
                if (cmd.run === "login") {
                  await api("/api/agents/" + agent.id + "/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
                } else if (cmd.run === "term") {
                  await api("/api/agents/" + agent.id + "/command", {
                    method: "POST",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ text: cmd.label }),
                  });
                }
                await loadWorkspaces();
                if (selectedId) setTermWanted((s) => new Set(s).add(selectedId));
              } catch (e) { toastError(e); }
            }}
            composer={{
              kind, onKind: setKind, value: draft, onChange: setDraft, onSend: sendTask,
              status, streaming, onToggleDock: showTerm, onStop: () => selectedId && stopAgent(selectedId),
              onAbort: abortTurn,
              lastReply: lastAssistantText(items),
              sessionBar: selectedId ? (
                <SessionBar
                  inline
                  sessions={sessions}
                  current={sessionCurrent}
                  onNew={async () => {
                    try {
                      await api("/api/workspaces/" + selectedId + "/sessions/new", { method: "POST" });
                      await loadWorkspaces();
                      await loadSessions();
                    } catch (e) { toastError(e); }
                  }}
                  onResume={async (path) => {
                    try {
                      await api("/api/workspaces/" + selectedId + "/sessions/resume", {
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
                      await api("/api/workspaces/" + selectedId + "/sessions/rename", {
                        method: "POST",
                        headers: { "Content-Type": "application/json" },
                        body: JSON.stringify({ path: s.path, name }),
                      });
                      await loadSessions();
                      await loadStatus(selectedId);
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
          onSignIn={async (provider) => {
            const ws = selected || workspaces[0];
            if (!ws || !ws.agent) { toast.info("Add a workspace first."); return; }
            try {
              await api("/api/agents/" + ws.agent.id + "/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ provider }),
              });
              const list = await loadWorkspaces();
              openTab(ws.id, list);
              setTermWanted((s) => new Set(s).add(ws.id));
              go("workspace");
            } catch (e) { toastError(e); }
          }}
        />
        <Mcps hidden={route !== "mcps"} mcp={mcp} />
        <Packages hidden={route !== "packages"} workspaceId={selected ? selected.id : ""} workspaceName={selected ? selected.name : ""} workspacePath={selected ? selected.path : ""} />
        <Devices hidden={route !== "devices"} />
      </main>

      <Palette
        open={paletteOpen}
        workspaces={workspaces}
        onClose={() => setPaletteOpen(false)}
        onRun={(a) => {
          if (a.kind === "settings" || a.kind === "preferences" || a.kind === "system" || a.kind === "providers" || a.kind === "mcps" || a.kind === "packages" || a.kind === "devices") { go(a.kind); return; }
          if (a.kind === "open") openTab(a.wsId);
          if (a.kind === "run") startManaged(a.wsId);
          if (a.kind === "term") openInteractive(a.wsId);
          if (a.kind === "stop") stopAgent(a.wsId);
        }}
      />
      <Toasts />
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
