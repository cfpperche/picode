import { IconChat, IconTerminal, IconPlay, IconStop, IconX, IconFolder, IconGit, IconSettings } from "./Icons.jsx";
import { displayAgentName } from "../lib/tree.js";
import { shortModel } from "../lib/chip.js";
import { repoLine, termLine } from "../lib/repoLine.js";
import { relTime, absTime } from "../lib/relTime.js";
import { ProviderFace } from "./ProviderFaces.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { checklistLine } from "../lib/checklist.js";

// One row shape for an agent, shared by the sidebar (dense, every action
// live) and the home dashboard (click-to-open only, a last-active stamp
// instead of the action cluster). `actions=false` hides run/stop/remove/
// rename/chat/terminal; `meta=true` adds the relative-time stamp.
export function AgentRow({
  agent: ag, ws,
  selectedId, onSelect,
  workingId, workingIds, waitingId, checklists,
  onFileTree, onGitGraph,
  actions = true, meta = false,
  onRenameAgent, onRun, onStop, onRemoveAgent, onRemove, onChat, onTerm, termView,
}) {
  const mode = ag.mode || "stopped";
  const label = displayAgentName(ag, ws);
  const model = shortModel(ag.model || "");
  const title = model ? label + " - " + model : label;
  const repo = repoLine(ag, ws);
  const stamp = ag.lastStatusAt || ag.lastStartedAt || ag.createdAt;
  const check = checklistLine(checklists && checklists[ag.id]);
  return (
    <li
      className={"ws-item" + (ag.id === selectedId ? " active" : "")}
      onClick={(e) => { if (e.target.closest("button")) return; onSelect(ag.id); }}
    >
      <div className="ws-row1">
        {ag.id === workingId || (workingIds || []).includes(ag.id) ? <PiSpinner /> : <ProviderFace agent={ag} />}
        <span className="ws-name" title={title}>
          {actions ? (
            <button type="button" className="ws-name-btn" title="Rename" onClick={() => onRenameAgent && onRenameAgent(ag, label)}>{label}</button>
          ) : label}
          {model ? <span className="ws-model"> - {model}</span> : null}
        </span>
        {ag.id === waitingId ? <span className="ws-wait">Waiting</span> : null}
        {meta && stamp ? <span className="ws-meta" title={absTime(stamp)}>{relTime(stamp)}</span> : null}
      </div>
      {check ? <ChecklistLine line={check} /> : null}
      <div className="ws-row2">
        <button
          type="button"
          className="ws-pill"
          title={"Files — " + (ws ? ws.path : (ag.workPath || ""))}
          onClick={(e) => { e.stopPropagation(); onFileTree && onFileTree("agent", ag.id, label); }}
        >
          <IconFolder size={12} /><span className="ws-pill-text">{repo.dir}</span>
        </button>
      </div>
      {repo.git ? (
        <div className="ws-row2">
          <button
            type="button"
            className="ws-pill"
            title={"Git graph" + (repo.git.branch ? " — " + repo.git.branch : "")}
            onClick={(e) => { e.stopPropagation(); onGitGraph && onGitGraph("agent", ag.id, label); }}
          >
            <IconGit size={12} /><span className="ws-pill-text">{repo.git.branch || "git"}</span>{repo.git.dirty ? <span className="ws-pill-badge">{repo.git.dirty}</span> : null}
          </button>
        </div>
      ) : null}
      {actions ? (
        <span className="ws-actions">
          {mode === "stopped"
            ? <button type="button" className="ws-icon-btn" title="Run" onClick={() => onRun(ag.id)}><IconPlay /></button>
            : <button type="button" className="ws-icon-btn" title="Stop" onClick={() => onStop(ag.id)}><IconStop size={12} /></button>}
          <button type="button" className="ws-icon-btn danger" title="Remove agent" onClick={() => onRemoveAgent ? onRemoveAgent(ag) : onRemove(ws)}><IconX size={12} /></button>
          <button type="button" className="ws-icon-btn" title="Chat" aria-pressed={ag.id === selectedId && !termView} onClick={(e) => { e.stopPropagation(); onChat && onChat(ag.id); }}><IconChat size={14} /></button>
          <button type="button" className="ws-icon-btn" title="Terminal" aria-pressed={ag.id === selectedId && !!termView} onClick={(e) => { e.stopPropagation(); onTerm && onTerm(ag.id); }}><IconTerminal size={14} /></button>
        </span>
      ) : null}
    </li>
  );
}

// Same shape for a terminal. Terminals have no live-status field beyond
// `createdAt`, so the meta stamp reads as "created", not "last active".
export function TermRow({
  term: t,
  selectedId, onSelectTerm,
  onFileTree, onGitGraph,
  actions = true, meta = false,
  onRenameTerm, onRemoveTerm,
}) {
  const line = termLine(t);
  const stamp = t.createdAt;
  return (
    <li
      className={"ws-item" + (selectedId === "t:" + t.id ? " active" : "")}
      onClick={(e) => { if (e.target.closest("button")) return; onSelectTerm && onSelectTerm(t.id); }}
    >
      <div className="ws-row1">
        <span className="tree-icon"><IconTerminal size={14} /></span>
        {actions ? (
          <button type="button" className="ws-name ws-name-btn" title="Rename" onClick={() => onRenameTerm && onRenameTerm(t)}>{t.name}</button>
        ) : <span className="ws-name">{t.name}</span>}
        {meta && stamp ? <span className="ws-meta" title={absTime(stamp)}>{relTime(stamp)}</span> : null}
      </div>
      <div className="ws-row2">
        <button type="button" className="ws-pill" title={"Files — " + t.cwd} onClick={(e) => { e.stopPropagation(); onFileTree && onFileTree("term", t.id, t.name); }}>
          <IconFolder size={12} /><span className="ws-pill-text">{line.dir}</span>
        </button>
      </div>
      {line.git ? (
        <div className="ws-row2">
          <button type="button" className="ws-pill" title={"Git graph" + (line.git.branch ? " — " + line.git.branch : "")} onClick={(e) => { e.stopPropagation(); onGitGraph && onGitGraph("term", t.id, t.name); }}>
            <IconGit size={12} /><span className="ws-pill-text">{line.git.branch || "git"}</span>{line.git.dirty ? <span className="ws-pill-badge">{line.git.dirty}</span> : null}
          </button>
        </div>
      ) : null}
      {actions ? (
        <span className="ws-actions">
          <button type="button" className="ws-icon-btn danger" title="Remove terminal" onClick={() => onRemoveTerm && onRemoveTerm(t)}><IconX size={12} /></button>
          <button type="button" className="ws-icon-btn" title="Settings" onClick={() => { location.hash = "#/termset/" + encodeURIComponent(t.id); }}><IconSettings /></button>
        </span>
      ) : null}
    </li>
  );
}

// The agent's internal checklist as one operator line (ADR-0055): the
// current step with its position, or a discrete "No checklist" when the
// contract was not met. Nothing known → nothing shown.
export function ChecklistLine({ line }) {
  if (!line) return null;
  if (line.kind === "absent") {
    return <div className="ws-row2 ws-check absent" title="The task required a checklist and none was written"><span className="ws-check-text">No checklist</span></div>;
  }
  const pos = "(" + line.position + "/" + line.total + ")";
  return (
    <div className="ws-row2 ws-check" title={pos + " " + line.text}>
      <span className="ws-check-pos">{pos}</span>
      <span className="ws-check-text">{line.text}</span>
    </div>
  );
}
