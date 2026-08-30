import { useCallback, useEffect, useRef, useState } from "react";
import { api, wsURL } from "../lib/api.js";
import { applyTheme, persistTheme, readThemeMode } from "../lib/theme.js";
import { startPresence } from "../lib/device.js";
import { startReconnectWatch } from "../lib/reconnect.js";
import Reconnect from "../components/Reconnect.jsx";
import { setShell } from "../lib/shell.js";
import { summarizeArgs } from "../components/Conversation.jsx";
import { fileChangeFromTool } from "../lib/diff.js";
import Conversation from "../components/Conversation.jsx";
import Composer from "../components/Composer.jsx";
import { bashLine } from "../lib/bashLine.js";
import TerminalDock from "../components/TerminalDock.jsx";
import { closeTerm } from "../lib/terms.js";
import ShareDrawer from "../components/ShareDrawer.jsx";
import InstallButton from "../components/InstallButton.jsx";
import Devices from "../components/Devices.jsx";
import Settings from "../components/Settings.jsx";
import PiSettings from "../components/PiSettings.jsx";
import System from "../components/System.jsx";
import Providers from "../components/Providers.jsx";
import Mcps from "../components/Mcps.jsx";
import Packages from "../components/Packages.jsx";
import "./mobile.css";
import { toastError } from "../lib/toast.js";
import { mergeAssistant } from "../lib/assistantMsg.js";
import { isSearchTool, hitsFromResult } from "../lib/searchCards.js";
import { stuckToBottom } from "../lib/stickScroll.js";
import Toasts from "../components/Toasts.jsx";
import ConfirmDialog from "../components/ConfirmDialog.jsx";

export default function MobileApp() {
  const [tab, setTab] = useState("agents");
  const [workspaces, setWorkspaces] = useState([]);
  const [selectedId, setSelectedId] = useState(null);
  const [themeMode, setThemeMode] = useState(readThemeMode);
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const [status, setStatus] = useState("idle");
  const [streaming, setStreaming] = useState(false);
  const [items, setItems] = useState([]);
  const [shareOpen, setShareOpen] = useState(false);
  const [reconnect, setReconnect] = useState(false);
  const [more, setMore] = useState("menu");
  const convRef = useRef(null);
  const nearBottom = useRef(true);
  const panelRef = useRef(null);
  const pendingPayload = useRef("");

  const selected = workspaces.find((w) => w.id === selectedId) || null;
  const agent = selected && selected.agent;
  const stopped = !agent || agent.mode === "stopped";
  const interactive = !!(agent && agent.mode === "interactive");

  useEffect(() => { applyTheme(themeMode); }, [themeMode]);
  useEffect(() => startPresence(), []);
  useEffect(() => startReconnectWatch({
    onState: (s) => { if (s === "down") setReconnect(true); },
  }), []);

  const load = useCallback(async () => {
    const list = await api("/api/workspaces");
    setWorkspaces(list);
    return list;
  }, []);

  useEffect(() => {
    load().then((list) => {
      const active = list.find((w) => w.agent && w.agent.mode !== "stopped") || list[0];
      if (active) setSelectedId(active.id);
    }).catch((e) => console.error("boot:", e));
  }, [load]);

  function closePanel() {
    const p = panelRef.current;
    if (!p) return;
    p.stopped = true;
    try { p.sock.close(); } catch { /* ignore */ }
    panelRef.current = null;
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
      try { handleEvent(JSON.parse(ev.data)); } catch { /* ignore */ }
    };
    sock.onclose = () => {
      if (panelRef.current === panel && !panel.stopped) {
        setStatus("disconnected");
        setStreaming(false);
      }
      if (window.__picodeKickHealth) window.__picodeKickHealth();
    };
  }

  function handleEvent(env) {
    const ev = env.event || {};
    switch (ev.type) {
      case "snapshot":
        setStatus(ev.streaming ? "streaming" : "idle");
        setStreaming(!!ev.streaming);
        break;
      case "agent_start":
        setStatus("streaming");
        setStreaming(true);
        break;
      case "agent_settled":
        setStatus("idle");
        setStreaming(false);
        break;
      case "message_update": {
        const d = ev.assistantMessageEvent;
        if (!d) break;
        if (d.type === "text_delta") setItems((cur) => appendDelta(cur, "", "agent", d.delta || ""));
        else if (d.type === "thinking_delta") setItems((cur) => appendDelta(cur, "thinking", "thinking", d.delta || ""));
        break;
      }
      case "tool_execution_start":
        setItems((cur) => [...cur, {
          kind: "tool", id: ev.toolCallId, name: ev.toolName || "tool",
          args: summarizeArgs(ev.args), toolArgs: ev.args || {}, status: "···",
          detail: JSON.stringify(ev.args || {}, null, 2), expanded: false,
          change: fileChangeFromTool(ev.toolName, ev.args, null),
        }]);
        break;
      case "tool_execution_end":
        setItems((cur) => cur.map((it) => it.kind === "tool" && it.id === ev.toolCallId
          ? { ...it, status: ev.isError ? "error" : "ok", detail: JSON.stringify(ev.result || {}, null, 2),
              result: ev.result,
              expanded: it.expanded || (isSearchTool(ev.toolName || it.name) && hitsFromResult(ev.result).length > 0),
              change: fileChangeFromTool(ev.toolName || it.name, ev.args, ev.result) || it.change }
          : it));
        break;
      case "enqueue_accepted":
        setItems((cur) => [...cur, { kind: "block", cls: "user", actor: "You", chip: ev.kind || "prompt", text: pendingPayload.current || "" }]);
        pendingPayload.current = "";
        setDraft("");
        break;
      case "message_end": {
        const m = ev.message || {};
        if (m.role === "assistant") setItems((cur) => mergeAssistant(cur, m));
        break;
      }
      default:
        break;
    }
  }

  useEffect(() => {
    if (!selected || !agent) { closePanel(); return; }
    if (agent.mode === "managed") {
      if (!panelRef.current || panelRef.current.agentId !== agent.id) connectPanel(selected);
    } else closePanel();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedId, agent && agent.id, agent && agent.mode]);

  async function startManaged(id) {
    const ws = workspaces.find((w) => w.id === id);
    if (!ws?.agent) return;
    try {
      await api(`/api/agents/${ws.agent.id}/managed/start`, { method: "POST" });
      await load();
      setSelectedId(id);
      setTab("chat");
    } catch (e) { toastError(e); }
  }

  async function openInteractive(id) {
    const ws = workspaces.find((w) => w.id === id);
    if (!ws?.agent) return;
    try {
      await api(`/api/workspaces/${ws.id}/open`, { method: "POST" });
      await load();
      setSelectedId(id);
      setTab("term");
    } catch (e) { toastError(e); }
  }

  async function stopAgent(id) {
    const ws = workspaces.find((w) => w.id === id);
    if (!ws?.agent) return;
    try {
      if (ws.agent.mode === "managed") await api(`/api/agents/${ws.agent.id}/managed/stop`, { method: "POST" });
      else {
        await api(`/api/workspaces/${ws.id}/close`, { method: "POST" });
        closeTerm(ws.agent.id);
      }
      if (panelRef.current?.agentId === ws.agent.id) panelRef.current.stopped = true;
      await load();
    } catch (e) { toastError(e); }
  }

  async function sendTask(text, images) {
    const payload = (typeof text === "string" ? text : draft).trim();
    const pics = images || [];
    if ((!payload && !pics.length) || !selected?.agent) return;
    const bash = bashLine(payload);
    if (bash && bash.refused) {
      toast.info("!! runs without sending output — use the terminal for that.");
      return;
    }
    if (bash && !pics.length) {
      const itemId = "bash-" + Date.now();
      try {
        try { await api("/api/agents/" + selected.agent.id + "/managed/start", { method: "POST" }); } catch { /* already */ }
        setItems((cur) => [...cur, { kind: "bash", id: itemId, command: bash.command, output: "", status: "run" }]);
        setDraft("");
        const res = await api("/api/agents/" + selected.agent.id + "/bash", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ command: bash.command }),
        });
        setItems((cur) => cur.map((it) => it.kind === "bash" && it.id === itemId ? {
          ...it, output: res.output || it.output, exit: res.exitCode,
          status: res.cancelled ? "cancelled" : (res.exitCode === 0 ? "ok" : "err"),
        } : it));
      } catch (e) {
        toastError(e);
        setItems((cur) => cur.map((it) => it.kind === "bash" && it.id === itemId ? { ...it, status: "err" } : it));
      }
      return;
    }
    try {
      try { await api("/api/agents/" + selected.agent.id + "/managed/start", { method: "POST" }); } catch { /* already */ }
      if (pics.length) {
        await api("/api/agents/" + selected.agent.id + "/prompt", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind, message: payload, images: pics.map((p) => ({ mimeType: p.mime, data: p.data })) }),
        });
      } else {
        await api("/api/agents/" + selected.agent.id + "/tasks", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ kind, payload, source: "user" }),
        });
      }
      setItems((cur) => [...cur, { kind: "block", cls: "user", actor: "You", chip: kind, text: payload, images: pics.map((p) => p.url) }]);
      setDraft("");
    } catch (e) { toastError(e); }
  }

  return (
    <div id="m-app">
      <header className="m-top">
        <span className="m-brand">PiCode</span>
        <button type="button" className="m-icon" onClick={() => setShareOpen(true)} aria-label="Open on phone">QR</button>
      </header>

      <div className="m-body">
        {tab === "agents" && (
          <ul className="m-list">
            {workspaces.map((ws) => {
              const mode = ws.agent ? ws.agent.mode : "stopped";
              return (
                <li key={ws.id} className={ws.id === selectedId ? "on" : ""}>
                  <button type="button" className="m-ws" onClick={() => { setSelectedId(ws.id); setTab("chat"); }}>
                    <span className={"m-dot" + (mode !== "stopped" ? " run" : "")} />
                    <span className="m-ws-name">{ws.name}</span>
                    <span className="m-ws-mode">{mode}</span>
                  </button>
                  <div className="m-ws-actions">
                    {mode === "stopped" ? (
                      <>
                        <button type="button" className="btn btn-primary btn-sm" onClick={() => startManaged(ws.id)}>Run</button>
                        <button type="button" className="btn btn-sm" onClick={() => openInteractive(ws.id)}>Terminal</button>
                      </>
                    ) : (
                      <button type="button" className="btn btn-sm btn-danger" onClick={() => stopAgent(ws.id)}>Stop</button>
                    )}
                  </div>
                </li>
              );
            })}
            {workspaces.length === 0 && <li className="m-empty">No agents on this machine.</li>}
          </ul>
        )}

        {tab === "chat" && (
          <section className="m-chat">
            {!selected ? <p className="m-empty">Pick an agent.</p> : (
              <>
                {stopped && (
                  <div className="m-cta">
                    <button type="button" className="btn btn-primary" onClick={() => startManaged(selectedId)}>Run agent</button>
                    <button type="button" className="btn" onClick={() => openInteractive(selectedId)}>Open terminal</button>
                  </div>
                )}
                <Conversation items={items} onToggleTool={(id) => setItems((cur) => cur.map((it) => it.kind === "tool" && it.id === id ? { ...it, expanded: !it.expanded } : it))}
                  onToggleFiles={() => {}} convRef={convRef} onScroll={() => {
                    const el = convRef.current;
                    if (el) nearBottom.current = stuckToBottom(el);
                  }} hidden={stopped} agentId={agent && agent.id} />
                <Composer kind={kind} onKind={setKind} value={draft} onChange={setDraft} onSend={sendTask}
                  status={status} streaming={streaming} stopped={stopped}
                  onToggleDock={() => setTab("term")} onStop={() => stopAgent(selectedId)}
                  agentId={selectedId} />
              </>
            )}
          </section>
        )}

        {tab === "term" && (
          <section className="m-term">
            {interactive && selected ? (
              <TerminalDock open maximized height={0} agent={agent} workspace={selected}
                onClose={() => setTab("chat")} onToggleMax={() => {}} onHeight={() => {}} />
            ) : (
              <div className="m-cta">
                <p>Agent is not in the terminal.</p>
                {selected && <button type="button" className="btn btn-primary" onClick={() => openInteractive(selectedId)}>Open terminal</button>}
              </div>
            )}
          </section>
        )}

        {tab === "more" && (
          <section className="m-more">
            {more === "menu" && (
              <ul className="m-list">
                <li><button type="button" className="m-ws" onClick={() => setMore("devices")}>Devices</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("settings")}>Settings</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("preferences")}>Preferences</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("system")}>System</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("providers")}>Providers</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("mcps")}>MCPs</button></li>
                <li><button type="button" className="m-ws" onClick={() => setMore("packages")}>Packages</button></li>
                <li><button type="button" className="m-ws" onClick={() => setShell("desktop")}>Desktop layout</button></li>
                <li style={{ paddingTop: 12 }}><InstallButton /></li>
              </ul>
            )}
            {more === "devices" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><Devices hidden={false} /></>}
            {more === "settings" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><PiSettings hidden={false} agent={agent} workspace={selected} /></>}
            {more === "preferences" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><Settings hidden={false} themeMode={themeMode} onTheme={(m) => { persistTheme(m); setThemeMode(m); }} /></>}
            {more === "system" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><System hidden={false} version="" system={null} /></>}
            {more === "providers" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><Providers hidden={false} catalog={{ providers: [] }} onSignIn={() => {}} /></>}
            {more === "mcps" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><Mcps hidden={false} mcp={{}} /></>}
            {more === "packages" && <><button type="button" className="btn btn-ghost btn-sm" onClick={() => setMore("menu")}>Back</button><Packages hidden={false} workspaceId={selectedId || ""} workspaceName={selected ? selected.name : ""} workspacePath={selected ? selected.path : ""} agentId={agent ? agent.id : ""} agentName={agent && agent.name && agent.name !== "default" ? agent.name : (selected ? selected.name : "")} /></>}
          </section>
        )}
      </div>

      <nav className="m-nav">
        <button type="button" className={tab === "agents" ? "on" : ""} onClick={() => setTab("agents")}>Agents</button>
        <button type="button" className={tab === "chat" ? "on" : ""} onClick={() => setTab("chat")}>Chat</button>
        <button type="button" className={tab === "term" ? "on" : ""} onClick={() => setTab("term")}>Term</button>
        <button type="button" className={tab === "more" ? "on" : ""} onClick={() => { setTab("more"); setMore("menu"); }}>More</button>
      </nav>
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
      <Toasts />
      {reconnect ? <Reconnect onReload={() => location.reload()} /> : null}
      <ConfirmDialog />
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
