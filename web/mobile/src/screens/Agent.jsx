import { cliProvidersHash } from "@picode/shared/domain/cliProviders.js";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import * as Sheet from "../components/MobileSheet.jsx";
import { toast } from "../lib/toast.js";
import ProjectToolsSheet from "../components/ProjectToolsSheet.jsx";
import PiSettings from "../components/PiSettings.jsx";
import ScreenHeader from "../components/ScreenHeader.jsx";
import StateChip, { agentState } from "../components/StateChip.jsx";
import Conversation from "../components/Conversation.jsx";
import Composer from "../components/Composer.jsx";
import TerminalDock from "../components/TerminalDock.jsx";
import TermSurface from "../components/TermSurface.jsx";
import KeyBar from "../components/KeyBar.jsx";
import { useAgentSocket } from "../hooks/useAgentSocket.js";
import { usePoll } from "../hooks/usePoll.js";
import { useTermAccessory } from "../hooks/useTermAccessory.js";
import { api } from "@picode/shared/client/api.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { extraSlash } from "@picode/shared/domain/slash.js";
import { stuckToBottom } from "@picode/shared/domain/stickScroll.js";
import { IconFolder, IconKeyboard } from "../components/Icons.jsx";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { applyUsage } from "@picode/shared/domain/feedReducers.js";
import { terms } from "../lib/terms.js";
import { agentDrafts } from "../lib/agentDrafts.js";
import { resolveAgentCliTerminal } from "../lib/agentTerminal.js";
import "../styles/mobile-chat.css";
import "../styles/mobile-tools.css";

// The pushed agent screen: header (name · state), a meta line (model ·
// cost · where), Chat or — for an agent living in a tmux TUI — Terminal,
// the mobile Conversation with its ask card, and the mobile Composer
// whose own Stop button is the abort. Start/Stop the agent from the
// header; one screen, no tabs of its own.
export default function Agent({ agent, workspace, terminal, catalog, workingIds, busy, initialView = "", onViewChange, onBack, onStart, onStop, onOpenFiles, onOpenGit, onAgentConfig }) {
  const id = agent && agent.id;
  const sock = useAgentSocket(agent, workspace?.id || "ws_free");
  const draft = useSyncExternalStore(agentDrafts.subscribe, () => agentDrafts.read(id));
  const [view, setView] = useState(initialView === "terminal" ? "term" : "chat");
  const [toolsOpen, setToolsOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [bar, setBar] = useState(null);
  const [slashExtra, setSlashExtra] = useState([]);
  const [snipExtra, setSnipExtra] = useState([]);
  const convRef = useRef(null);
  const termHostRef = useRef(null);
  const nearBottom = useRef(true);

  const mode = (agent && agent.mode) || "stopped";
  const stopped = mode === "stopped";
  const interactive = mode === "interactive";
  const managed = mode === "managed";
  const cliTerminal = resolveAgentCliTerminal(agent, terminal);
  // ShellTerm namespaces its xterm entries to avoid colliding with the
  // Pi-owned TerminalDock entry for the same agent identity.
  const terminalKey = cliTerminal ? "sh:" + cliTerminal.id : id;
  const keys = useTermAccessory(termHostRef, () => terms.get(terminalKey), view === "term" && interactive ? terminalKey : "", () => { toast.warn("Terminal reconnecting — tap the pane, then try again."); });
  const state = agentState(agent, workingIds);
  const name = agent ? displayAgentName(agent, workspace) : "";

  useEffect(() => {
    setView(initialView === "terminal" && interactive ? "term" : (interactive ? "term" : "chat"));
  }, [id, interactive, initialView]);

  function changeView(next) {
    setView(next);
    if (onViewChange) onViewChange(next === "term" ? "terminal" : "chat");
  }

  useEffect(() => {
    if (!id || stopped) { setSlashExtra([]); return; }
    api("/api/agents/" + id + "/slash")
      .then((d) => setSlashExtra(extraSlash(d.skills, d.templates, d.commands)))
      .catch(() => setSlashExtra([]));
  }, [id, mode]);
  // Snippets do not need a running agent: the picker rides its own state
  // and the snip.* feed, so /snip: lists while the agent is stopped.
  useEffect(() => subscribeFeed((ev) => {
    if (!(ev.type === "feed.open" || ev.type === "feed.reset" || (ev.type && ev.type.startsWith("snip.")))) return;
    api("/api/snips/picker").then((d) => setSnipExtra(extraSlash([], [], [], d.snips || []))).catch(() => setSnipExtra([]));
  }), []);

  usePoll(async () => {
    if (!id) return;
    const wsId = workspace ? workspace.id : "ws_free";
    setBar(await api("/api/workspaces/" + encodeURIComponent(wsId) + "/status?agent=" + encodeURIComponent(id)));
  }, 15000, !!id && !stopped);
  useEffect(() => subscribeFeed((ev) => { if (ev.type === "agent.usage" && ev.data && ev.data.agentId === id) setBar((bar) => applyUsage(bar, ev.data)); }), [id]);

  useEffect(() => {
    const el = convRef.current;
    if (!el || !nearBottom.current) return;
    el.scrollTop = el.scrollHeight;
  }, [sock.scrollTick, sock.state.items.length]);

  if (!agent) {
    return (
      <div className="m-screen">
        <ScreenHeader title="Agent" onBack={onBack} />
        <p className="m-empty-line m-pad">That agent is gone.</p>
      </div>
    );
  }

  const meta = [
    shortModel(agent.model || ""),
    workspace ? workspace.name : "free agent",
  ].filter(Boolean).join(" · ");

  const viewSwitch = interactive ? (
    <fieldset className="m-view-switch" aria-label="Agent view">
      <legend className="sr-only">Agent view</legend>
      {[['chat', 'Chat'], ['term', 'Terminal']].map(([value, label]) => (
        <label key={value} className="m-view-switch-option">
          <input type="radio" name="m-agent-view" value={value} checked={view === value} onChange={() => changeView(value)} />
          <span>{label}</span>
        </label>
      ))}
    </fieldset>
  ) : null;

  const right = (
    <>
      {viewSwitch}
      <button type="button" className="btn btn-sm m-changes-btn" title="Project tools" aria-label="Project tools" aria-haspopup="dialog" aria-expanded={toolsOpen} onClick={() => setToolsOpen(true)}><IconFolder size={16} /></button>
      {interactive && view === "term" ? (
        <button type="button" className={"btn btn-sm m-keys-btn" + (keys.visible ? " on" : "")} title={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-label={keys.visible ? "Hide keyboard" : "Show keyboard"} aria-pressed={keys.visible} onPointerDown={(e) => { if (keys.visible) e.preventDefault(); }} onClick={() => { keys.visible ? keys.hide() : keys.show(); }}>
          <IconKeyboard size={16} />
        </button>
      ) : null}
      {stopped
        ? <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={() => onStart(agent, workspace)}>Start</button>
        : <button type="button" className="btn btn-sm" disabled={busy} onClick={() => onStop(agent, workspace)}>Stop</button>}
    </>
  );

  return (
    <div className="m-screen m-agent">
      <ScreenHeader title={name} sub={<><StateChip state={state} /><span className="m-agent-meta" title={meta}>{meta}</span></>} onBack={onBack} right={right} />
      {view === "term" && interactive ? (
        <>
          <div className="m-term" ref={termHostRef}>
            {cliTerminal ? (
              <TermSurface key={"agent-cli-term-" + cliTerminal.id} term={cliTerminal} hidden={false} cwdKind="term" />
            ) : (
              <TerminalDock key={"agent-term-" + id} open agent={agent} workspace={workspace} />
            )}
          </div>
          {keys.visible ? <KeyBar armed={keys.armed} onArm={keys.armKey} onKey={keys.sendKey} onHide={keys.hide} /> : null}
        </>
      ) : (
        <div className="m-chat chat-body">
          <div className="chat-main">
            {interactive ? (
              <div className="m-cta">
                <p className="m-empty-line">Continue this conversation in the terminal.</p>
                <button type="button" className="btn btn-primary" onClick={() => changeView("term")}>Open terminal</button>
              </div>
            ) : null}
            {!interactive && !sock.state.items.length ? <div className="m-chat-empty"><p>{stopped ? "Send a message to start this agent." : "Start a conversation with this agent."}</p><button type="button" className="btn btn-ghost" onClick={() => document.getElementById("task-input")?.focus()}>Write a message</button></div> : null}
            <Conversation
              items={sock.state.items}
              onToggleTool={sock.toggleTool}
              onToggleFiles={sock.toggleFiles}
              convRef={convRef}
              onScroll={() => { const el = convRef.current; if (el) nearBottom.current = stuckToBottom(el); }}
              hidden={interactive}
              streaming={sock.state.streaming}
              agentId={id}
              onReplyAsk={sock.replyAsk}
              onAbortBash={sock.abortBash}
            />
            {!interactive ? <Composer
              key={id}
              kind={draft.kind}
              onKind={kind => agentDrafts.update(id, { kind })}
              value={draft.text}
              onChange={text => agentDrafts.update(id, { text })}
              images={draft.images}
              onImagesChange={change => agentDrafts.update(id, { images: typeof change === "function" ? change(agentDrafts.read(id).images) : change })}
              sending={draft.sending}
              sendError={draft.error}
              onSend={text => agentDrafts.submit(id, sock.send, text)}
              status={stopped ? "idle" : sock.state.status}
              streaming={sock.state.streaming}
              waiting={sock.state.waiting}
              stopped={stopped || interactive}
              onToggleDock={() => { if (interactive) changeView("term"); }}
              onStop={() => onStop(agent, workspace)}
              onAbort={sock.abort}
              agentId={id}
              slashExtra={[...slashExtra, ...snipExtra]}
              onSlash={cmd => { if (cmd.run === "go-providers" || cmd.run === "go-providers-new") location.hash = cliProvidersHash("pi", { add: cmd.run === "go-providers-new" }); }}
              statusBar={bar}
              lastReply={sock.state.items.findLast(it => it.kind === "block" && !it.cls && it.text)?.text || ""}
              onSettings={() => setSettingsOpen(true)}
            /> : null}
          </div>
        </div>
      )}
      <ProjectToolsSheet open={toolsOpen} onOpenChange={setToolsOpen} title={name} onFiles={() => onOpenFiles({ kind: "agent", id })} onGit={() => onOpenGit({ kind: "agent", id })} />
      <Sheet.Root open={settingsOpen} onOpenChange={setSettingsOpen}>
        <Sheet.Portal>
          <Sheet.Overlay className="dlg-overlay" />
          <Sheet.Content className="dlg m-agent-settings" aria-describedby={undefined}>
            <Sheet.Title className="dlg-title">Settings · {name}</Sheet.Title>
            <div className="m-agent-settings-body">
              <PiSettings agentOnly hidden={false} agent={agent} workspace={workspace} catalog={catalog} onAgentConfig={cfg => onAgentConfig(agent, cfg)} />
            </div>
            <div className="dlg-actions"><Sheet.Close asChild><button type="button" className="btn btn-primary">Done</button></Sheet.Close></div>
          </Sheet.Content>
        </Sheet.Portal>
      </Sheet.Root>
    </div>
  );
}
