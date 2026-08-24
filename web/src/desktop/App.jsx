import { useCallback, useEffect, useRef, useState } from "react";
import { api, humanizeError, wsURL } from "../lib/api.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { closeTerm } from "../components/TerminalDock.jsx";
import { summarizeArgs } from "../components/Conversation.jsx";
import { fileChangeFromTool } from "../lib/diff.js";
import Sidebar from "../components/Sidebar.jsx";
import AgentTabs from "../components/AgentTabs.jsx";
import ChatSurface from "../components/ChatSurface.jsx";
import TerminalDock from "../components/TerminalDock.jsx";
import Settings from "../components/Settings.jsx";
import Providers from "../components/Providers.jsx";
import Mcps from "../components/Mcps.jsx";
import Devices from "../components/Devices.jsx";
import Palette from "../components/Palette.jsx";
import { parseRoute, go } from "../lib/routes.js";
import { startPresence } from "../lib/device.js";
import { setShell } from "../lib/shell.js";
import { toast, toastError } from "../lib/toast.js";
import Toasts from "../components/Toasts.jsx";

export default function App() {
  const [workspaces, setWorkspaces] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
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
  const [dockWanted, setDockWanted] = useState(() => new Set());
  const [dockMax, setDockMax] = useState(() => new Set());
  const [dockH, setDockH] = useState(() => parseInt(localStorage.getItem("picode-dock-h") || "", 10) || 0);
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const [status, setStatus] = useState("idle");
  const [streaming, setStreaming] = useState(false);
  const [items, setItems] = useState([]);
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const panelRef = useRef(null);
  const pendingPayload = useRef("");
  const turnFiles = useRef(new Set());
  const selectedRef = useRef(null);
  selectedRef.current = selectedId;

  const selected = workspaces.find((w) => w.id === selectedId) || null;
  const agent = selected && selected.agent;
  const stopped = !agent || agent.mode === "stopped";
  const interactive = !!(agent && agent.mode === "interactive");
  const dockOpen = !!(selected && dockWanted.has(selected.id) && interactive);
  const dockMaximized = !!(selected && dockMax.has(selected.id) && dockOpen);

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
    return list;
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const [sys, ver] = await Promise.all([api("/api/system"), api("/api/version")]);
        setSystem(sys);
        setVersion(ver.version);
        setHost(sys.host || "local");
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
    setSelectedId(id);
    setTabs((t) => t.includes(id) ? t : [...t, id]);
    const ws = (list || workspaces).find((w) => w.id === id);
    if (ws) prepareSurface(ws);
  }

  function prepareSurface(ws) {
    const a = ws.agent;
    if (!a || a.mode === "stopped") {
      setStatus("stopped");
      setStreaming(false);
      return;
    }
    if (a.mode === "interactive") {
      setStatus("interactive");
      setStreaming(false);
      setItems((cur) => cur.length ? cur : [{ kind: "sys", text: "Agent is running in the terminal. Use the Terminal button to pair with it." }]);
    }
  }

  function closeTab(id) {
    const ws = workspaces.find((w) => w.id === id);
    setTabs((t) => t.filter((x) => x !== id));
    setDockWanted((s) => { const n = new Set(s); n.delete(id); return n; });
    setDockMax((s) => { const n = new Set(s); n.delete(id); return n; });
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
    const el = convRef.current;
    if (el && nearBottom.current) el.scrollTop = el.scrollHeight;
  }

  function connectPanel(ws) {
    closePanel();
    setItems([{ kind: "sys", text: "Connected. Send a task to start." }]);
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
        break;
      case "agent_settled": {
        setStatus("idle");
        setStreaming(false);
        const paths = [...turnFiles.current];
        turnFiles.current = new Set();
        if (paths.length) {
          setItems((cur) => [...cur, { kind: "files", paths, expanded: false }]);
        }
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
      case "enqueue_accepted":
        setItems((cur) => [...cur, {
          kind: "block", cls: "user", actor: "You", chip: ev.kind || "prompt",
          text: pendingPayload.current || "",
        }]);
        pendingPayload.current = "";
        setDraft("");
        queueMicrotask(scrollConv);
        break;
      case "task_delivered":
        setItems((cur) => [...cur, { kind: "sys", text: `✓ delivered (${ev.kind})` }]);
        break;
      case "task_failed":
        setItems((cur) => [...cur, { kind: "sys", text: `✗ failed: ${ev.error}`, err: true }]);
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
    const ws = workspaces.find((w) => w.id === id);
    if (!ws || !ws.agent) return;
    try {
      await api(`/api/agents/${ws.agent.id}/managed/start`, { method: "POST" });
      const list = await loadWorkspaces();
      openTab(id, list);
    } catch (err) { toastError(err); }
  }

  async function openInteractive(id) {
    const ws = workspaces.find((w) => w.id === id);
    if (!ws || !ws.agent) return;
    try {
      await api(`/api/workspaces/${ws.id}/open`, { method: "POST" });
      const list = await loadWorkspaces();
      openTab(id, list);
      setDockWanted((s) => new Set(s).add(id));
    } catch (err) { toastError(err); }
  }

  async function stopAgent(id) {
    const ws = workspaces.find((w) => w.id === id);
    if (!ws || !ws.agent) return;
    try {
      if (ws.agent.mode === "managed") {
        await api(`/api/agents/${ws.agent.id}/managed/stop`, { method: "POST" });
      } else {
        await api(`/api/workspaces/${ws.id}/close`, { method: "POST" });
        closeTerm(ws.agent.id);
      }
      if (panelRef.current && panelRef.current.agentId === ws.agent.id) panelRef.current.stopped = true;
      await loadWorkspaces();
    } catch (err) { toastError(err); }
  }

  async function removeWorkspace(ws) {
    if (!confirm(`Remove workspace "${ws.name}"?\n(The project folder is not deleted.)`)) return;
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
    if (!name || !path) {
      setFormError("Name and folder path are required.");
      return;
    }
    try {
      const ws = await api("/api/workspaces", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, path, ...newCfg }),
      });
      e.target.reset();
      setNewCfg({ provider: "", model: "", thinking: "" });
      setShowForm(false);
      const list = await loadWorkspaces();
      openTab(ws.id, list);
    } catch (err) {
      setFormError(humanizeError(err.message));
    }
  }

  function sendTask() {
    const p = panelRef.current;
    const payload = draft.trim();
    if (!p || !payload || p.sock.readyState !== WebSocket.OPEN) return;
    pendingPayload.current = payload;
    p.sock.send(JSON.stringify({ type: "enqueue", kind, payload }));
  }

  function setTheme(mode) {
    persistTheme(mode);
    setThemeMode(mode);
  }

  function toggleDock() {
    if (!selectedId) return;
    // Dock is the tmux TUI. Managed has no PTY — switch modes first.
    if (!interactive) {
      openInteractive(selectedId);
      return;
    }
    setDockWanted((s) => {
      const n = new Set(s);
      if (n.has(selectedId)) n.delete(selectedId);
      else n.add(selectedId);
      return n;
    });
  }

  function hideDock() {
    if (!selectedId) return;
    setDockWanted((s) => { const n = new Set(s); n.delete(selectedId); return n; });
    setDockMax((s) => { const n = new Set(s); n.delete(selectedId); return n; });
  }

  function toggleDockMax() {
    if (!selectedId) return;
    setDockMax((s) => {
      const n = new Set(s);
      if (n.has(selectedId)) n.delete(selectedId);
      else n.add(selectedId);
      return n;
    });
  }

  function setHeight(h) {
    setDockH(h);
    localStorage.setItem("picode-dock-h", String(h));
    if (selectedId) {
      setDockMax((s) => { const n = new Set(s); n.delete(selectedId); return n; });
    }
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
        onNew={() => setShowForm(true)}
        onCancel={() => { setShowForm(false); setFormError(""); }}
        onSubmit={submitNew}
        onSelect={(id) => openTab(id)}
        onRun={startManaged}
        onStop={stopAgent}
        onRemove={removeWorkspace}
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
            hidden={noTabs || dockMaximized}
            stopped={stopped}
            items={items}
            onToggleTool={(id) => setItems((cur) => cur.map((it) => it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it))}
            onToggleFiles={(idx) => setItems((cur) => cur.map((it, i) => i === idx && it.kind === "files" ? { ...it, expanded: !it.expanded } : it))}
            convRef={convRef}
            onScroll={() => {
              const el = convRef.current;
              if (el) nearBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 48;
            }}
            onRun={() => selectedId && startManaged(selectedId)}
            onOpenTerm={() => selectedId && openInteractive(selectedId)}
            catalog={catalog}
            agent={agent}
            onConfig={async (cfg) => {
              if (!agent) return;
              const modeChanged = Object.prototype.hasOwnProperty.call(cfg, "opMode")
                && (cfg.opMode || "full") !== (agent.opMode || "full");
              const was = agent.mode;
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
                  else if (was === "interactive") await openInteractive(selectedId);
                }
              } catch (e) { toastError(e); }
            }}
            onSlash={async (cmd) => {
              if (!agent) return;
              if (cmd.run === "focus-model") { document.getElementById("agent-model")?.focus(); return; }
              if (cmd.run === "focus-thinking") { document.getElementById("agent-thinking")?.focus(); return; }
              if (cmd.run === "focus-provider") { document.getElementById("agent-provider")?.focus(); return; }
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
                if (selectedId) setDockWanted((s) => new Set(s).add(selectedId));
              } catch (e) { toastError(e); }
            }}
            composer={{
              kind, onKind: setKind, value: draft, onChange: setDraft, onSend: sendTask,
              status, streaming, onToggleDock: toggleDock, onStop: () => selectedId && stopAgent(selectedId),
            }}
          />

          <TerminalDock
            open={dockOpen}
            maximized={dockMaximized}
            height={dockH}
            agent={agent}
            workspace={selected}
            onClose={hideDock}
            onToggleMax={toggleDockMax}
            onHeight={setHeight}
          />
        </div>

        <Settings
          hidden={route !== "settings"}
          version={version}
          themeMode={themeMode}
          onTheme={setTheme}
          system={system}
        />
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
              setDockWanted((s) => new Set(s).add(ws.id));
              go("workspace");
            } catch (e) { toastError(e); }
          }}
        />
        <Mcps hidden={route !== "mcps"} mcp={mcp} />
        <Devices hidden={route !== "devices"} />
      </main>

      <Palette
        open={paletteOpen}
        workspaces={workspaces}
        onClose={() => setPaletteOpen(false)}
        onRun={(a) => {
          if (a.kind === "settings" || a.kind === "providers" || a.kind === "mcps" || a.kind === "devices") { go(a.kind); return; }
          if (a.kind === "open") openTab(a.wsId);
          if (a.kind === "run") startManaged(a.wsId);
          if (a.kind === "term") openInteractive(a.wsId);
          if (a.kind === "stop") stopAgent(a.wsId);
        }}
      />
      <Toasts />
    </div>
  );
}

function appendDelta(cur, cls, actor, delta) {
  const last = cur[cur.length - 1];
  if (last && last.kind === "block" && last.actor === actor && last.cls === cls) {
    const next = cur.slice();
    next[next.length - 1] = { ...last, text: last.text + delta };
    return next;
  }
  return [...cur, { kind: "block", cls, actor, text: delta }];
}
