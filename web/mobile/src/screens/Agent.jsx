import { useEffect, useRef, useState } from "react";
import * as Sheet from "../components/MobileSheet.jsx";
import PiSettings from "../components/PiSettings.jsx";
import ScreenHeader from "../components/ScreenHeader.jsx";
import StateChip, { agentState } from "../components/StateChip.jsx";
import Conversation from "../components/Conversation.jsx";
import Composer from "../components/Composer.jsx";
import TerminalDock from "../components/TerminalDock.jsx";
import { useAgentSocket } from "../hooks/useAgentSocket.js";
import { usePoll } from "../hooks/usePoll.js";
import { api } from "@picode/shared/client/api.js";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { formatMoney } from "@picode/shared/domain/providerUsage.js";
import { extraSlash } from "@picode/shared/domain/slash.js";
import { stuckToBottom } from "@picode/shared/domain/stickScroll.js";
import { eventsToItems } from "@picode/shared/domain/replay.js";
import { IconGit } from "../components/Icons.jsx";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { applyUsage } from "@picode/shared/domain/feedReducers.js";

// The pushed agent screen: header (name · state), a meta line (model ·
// cost · where), Chat or — for an agent living in a tmux TUI — Terminal,
// the mobile Conversation with its ask card, and the mobile Composer
// whose own Stop button is the abort. Start/Stop the agent from the
// header; one screen, no tabs of its own.
export default function Agent({ agent, workspace, catalog, workingIds, busy, onBack, onStart, onStop, onOpenChanges, onAgentConfig }) {
  const sock = useAgentSocket(agent);
  const [draft, setDraft] = useState("");
  const [kind, setKind] = useState("prompt");
  const [view, setView] = useState("chat");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [bar, setBar] = useState(null);
  const [slashExtra, setSlashExtra] = useState([]);
  const convRef = useRef(null);
  const nearBottom = useRef(true);

  const id = agent && agent.id;
  const mode = (agent && agent.mode) || "stopped";
  const stopped = mode === "stopped";
  const interactive = mode === "interactive";
  const managed = mode === "managed";
  const state = agentState(agent, workingIds);
  const name = agent ? displayAgentName(agent, workspace) : "";

  useEffect(() => { setView(interactive ? "term" : "chat"); }, [id, interactive]);

  useEffect(() => {
    if (!id || stopped) { setSlashExtra([]); return; }
    api("/api/agents/" + id + "/slash")
      .then((d) => setSlashExtra(extraSlash(d.skills, d.templates, d.commands)))
      .catch(() => setSlashExtra([]));
  }, [id, mode]);

  // History: the last 200 events of the agent's own session (the server
  // falls back to the agent's session path when none is given), replayed
  // under whatever the socket already showed. One fetch per agent.
  useEffect(() => {
    if (!id || !managed) return undefined;
    let stale = false;
    const wsId = workspace ? workspace.id : "ws_free";
    api("/api/workspaces/" + encodeURIComponent(wsId) + "/sessions/transcript?agent=" + encodeURIComponent(id) + "&tail=200")
      .then((t) => { if (!stale) sock.seed(eventsToItems(t.events || [])); })
      .catch(() => {});
    return () => { stale = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, managed]);

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
    bar && bar.cost > 0 ? formatMoney(bar.cost, "usd") : "",
    bar && bar.contextPercent != null ? "ctx " + Math.round(bar.contextPercent) + "%" : "",
    workspace ? workspace.name : "free agent",
  ].filter(Boolean).join(" · ");

  const right = stopped
    ? <button type="button" className="btn btn-primary btn-sm" disabled={busy} onClick={() => onStart(agent, workspace)}>Start</button>
    : <button type="button" className="btn btn-sm" disabled={busy} onClick={() => onStop(agent, workspace)}>Stop</button>;

  return (
    <div className="m-screen m-agent">
      <ScreenHeader title={name} sub={meta} onBack={onBack} right={right} />
      <div className="m-agent-state">
        <StateChip state={state} />
        {agent.git && agent.git.dirty ? (
          <button type="button" className="btn btn-sm m-changes-btn" title="Uncommitted changes" onClick={() => onOpenChanges("agent", id, name)}><IconGit size={13} /> {agent.git.dirty} changed</button>
        ) : null}
        {interactive ? (
          <div className="dash-range m-seg" role="radiogroup" aria-label="View">
            {[["chat", "Chat"], ["term", "Terminal"]].map(([v, label]) => (
              <label key={v} className="dash-range-opt">
                <input type="radio" name="m-agent-view" value={v} checked={view === v} onChange={() => setView(v)} />
                <span className="dash-range-face">{label}</span>
              </label>
            ))}
          </div>
        ) : null}
      </div>

      {view === "term" && interactive ? (
        <div className="m-term">
          {<TerminalDock key={"agent-term-" + id} open agent={agent} workspace={workspace} />}
        </div>
      ) : (
        <div className="m-chat chat-body">
          <div className="chat-main">
            {stopped ? (
              <div className="m-cta">
                <p className="m-empty-line">This agent is stopped. Start it to chat, or send a message and it starts on its own.</p>
              </div>
            ) : interactive ? (
              <div className="m-cta">
                <p className="m-empty-line">This agent runs in a terminal. Chat here is not connected — switch to Terminal to type into it.</p>
              </div>
            ) : null}
            <Conversation
              items={sock.state.items}
              onToggleTool={sock.toggleTool}
              onToggleFiles={() => {}}
              convRef={convRef}
              onScroll={() => { const el = convRef.current; if (el) nearBottom.current = stuckToBottom(el); }}
              hidden={interactive}
              streaming={sock.state.streaming}
              agentId={id}
              onReplyAsk={sock.replyAsk}
              onAbortBash={sock.abortBash}
            />
            <Composer
              kind={kind}
              onKind={setKind}
              value={draft}
              onChange={setDraft}
              onSend={(text, images) => { setDraft(""); return sock.send(typeof text === "string" ? text : draft, images, kind); }}
              status={stopped ? "idle" : sock.state.status}
              streaming={sock.state.streaming}
              waiting={sock.state.waiting}
              stopped={stopped || interactive}
              onToggleDock={() => { if (interactive) setView("term"); }}
              onStop={() => onStop(agent, workspace)}
              onAbort={sock.abort}
              agentId={id}
              slashExtra={slashExtra}
              statusBar={bar}
              onSettings={() => setSettingsOpen(true)}
            />
          </div>
        </div>
      )}
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
