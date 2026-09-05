import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconChat, IconEllipsis, IconFolder, IconGit, IconPencil, IconPlay, IconSettings, IconStop, IconTerminal, IconX } from "./Icons.jsx";
import { displayAgentName } from "../lib/tree.js";
import { shortModel } from "../lib/chip.js";
import { repoLine, termLine } from "../lib/repoLine.js";
import { relTime, absTime } from "../lib/relTime.js";
import { ProviderFace } from "./ProviderFaces.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { checklistLine } from "../lib/checklist.js";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { terminalActivityStamp, terminalCli, terminalCliLabel, terminalStatus, terminalStatusLabel } from "../lib/terminalCli.js";

function RowMenu({ label, children }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="ws-row-menu-trigger"
          aria-label={"Actions for " + label}
          title="Actions"
          onClick={(e) => e.stopPropagation()}
        >
          <IconEllipsis size={15} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="ws-row-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
          {children}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

function RowMenuItem({ children, onSelect, danger = false }) {
  return <DropdownMenu.Item className={"ws-row-menu-item" + (danger ? " danger" : "")} onSelect={onSelect}>{children}</DropdownMenu.Item>;
}

function openRow(e, onSelect) {
  if (e.key === "Enter" || e.key === " ") {
    e.preventDefault();
    onSelect();
  }
}

function AgentStatus({ status, stamp }) {
  const active = status === "working";
  const age = active && stamp ? relTime(stamp) : "";
  return (
    <span className={"ws-status is-" + status}>
      {active ? <PiSpinner title="Working" /> : null}
      <span>{status === "needs-you" ? "Needs you" : status === "working" ? "Working" : status === "interactive" ? "In terminal" : status === "stopped" ? "Stopped" : "Ready"}</span>
      {age ? <span className="ws-status-age">{age}</span> : null}
    </span>
  );
}

function TerminalStatus({ term }) {
  const status = terminalStatus(term);
  const stamp = terminalActivityStamp(term);
  const age = status === "working" && stamp ? relTime(stamp) : "";
  return (
    <span className={"ws-status is-" + status}>
      {status === "working" ? <PiSpinner title="Working" /> : null}
      <span>{terminalStatusLabel(term)}</span>
      {age ? <span className="ws-status-age">{age}</span> : null}
    </span>
  );
}

function ContextLine({ line, ownerKind, ownerId, ownerLabel, onFileTree, onGitGraph }) {
  return (
    <div className="ws-context">
      <button
        type="button"
        className="ws-context-btn"
        title={"Files — " + line.dir}
        onClick={(e) => { e.stopPropagation(); onFileTree && onFileTree(ownerKind, ownerId, ownerLabel); }}
      >
        <IconFolder size={12} /><span>{line.dir}</span>
      </button>
      {line.git ? (
        <button
          type="button"
          className="ws-context-btn ws-context-git"
          title={"Git graph — " + (line.git.branch || "git")}
          onClick={(e) => { e.stopPropagation(); onGitGraph && onGitGraph(ownerKind, ownerId, ownerLabel); }}
        >
          <IconGit size={12} /><span>{line.git.branch || "git"}</span>{line.git.dirty ? <span className="ws-context-dirty">{line.git.dirty}</span> : null}
        </button>
      ) : null}
    </div>
  );
}

// Compact supervision row shared by the desktop sidebar and dashboard. The
// row is intentionally flat: identity and activity lead, location recedes,
// and secondary actions live behind one keyboard-accessible menu.
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
  const title = model ? label + " — " + model : label;
  const repo = repoLine(ag, ws);
  const stamp = ag.lastStatusAt || ag.lastStartedAt || ag.createdAt;
  const check = checklistLine(checklists && checklists[ag.id]);
  const waiting = ag.waiting || ag.id === waitingId;
  const working = !waiting && (ag.streaming || ag.id === workingId || (workingIds || []).includes(ag.id));
  const status = waiting ? "needs-you" : working ? "working" : mode === "interactive" ? "interactive" : mode === "stopped" ? "stopped" : "ready";
  const select = () => onSelect(ag.id);
  return (
    <li className={"ws-item is-" + status + (ag.id === selectedId ? " active" : "")}>
      <div className="ws-row-main">
        <div
          className="ws-row-hit"
          role="button"
          tabIndex={0}
          aria-current={ag.id === selectedId ? "page" : undefined}
          aria-label={title}
          onClick={select}
          onKeyDown={(e) => openRow(e, select)}
        >
          <span className="ws-identity-mark">
            <ProviderFace agent={ag} />
            {status === "working" ? <span className="ws-activity-dot" aria-hidden="true" /> : null}
          </span>
          <span className="ws-copy">
            <span className="ws-title" title={title}>{label}</span>
            <span className="ws-subtitle">{model || (mode === "interactive" ? "Interactive session" : "Pi agent")}</span>
          </span>
          <AgentStatus status={status} stamp={stamp} />
        </div>
        {actions ? (
          <RowMenu label={label}>
            {mode === "stopped"
              ? <RowMenuItem onSelect={() => onRun && onRun(ag.id)}><IconPlay size={14} /> Start agent</RowMenuItem>
              : <RowMenuItem onSelect={() => onStop && onStop(ag.id)}><IconStop size={13} /> Stop agent</RowMenuItem>}
            <RowMenuItem onSelect={() => onChat && onChat(ag.id)}><IconChat size={14} /> Open chat</RowMenuItem>
            <RowMenuItem onSelect={() => onTerm && onTerm(ag.id)}><IconTerminal size={14} /> Open terminal</RowMenuItem>
            <RowMenuItem onSelect={() => onRenameAgent && onRenameAgent(ag, label)}><IconPencil size={13} /> Rename</RowMenuItem>
            <RowMenuItem danger onSelect={() => onRemoveAgent ? onRemoveAgent(ag) : onRemove(ws)}><IconX size={13} /> Remove agent</RowMenuItem>
          </RowMenu>
        ) : null}
      </div>
      <ContextLine line={repo} ownerKind="agent" ownerId={ag.id} ownerLabel={label} onFileTree={onFileTree} onGitGraph={onGitGraph} />
      {check ? <ChecklistLine line={check} /> : null}
      {meta && stamp ? <span className="ws-meta" title={absTime(stamp)}>{relTime(stamp)}</span> : null}
    </li>
  );
}

// Terminal rows use the same identity → activity → location rhythm as agent
// rows. `tui` is authoritative when present; legacy top-level cli/state is
// still rendered so older sessions degrade visibly rather than disappearing.
export function TermRow({
  term: t,
  selectedId, onSelectTerm,
  onFileTree, onGitGraph,
  actions = true,
  onRenameTerm, onRemoveTerm,
}) {
  const line = termLine(t);
  const cli = terminalCli(t);
  const cliLabel = cli ? terminalCliLabel(cli) : "Terminal";
  const selected = selectedId === "t:" + t.id;
  const select = () => onSelectTerm && onSelectTerm(t.id);
  return (
    <li className={"ws-item is-terminal is-" + terminalStatus(t) + (selected ? " active" : "")}>
      <div className="ws-row-main">
        <div className="ws-row-hit" role="button" tabIndex={0} aria-current={selected ? "page" : undefined} aria-label={(t.name || "Terminal") + " — " + cliLabel} onClick={select} onKeyDown={(e) => openRow(e, select)}>
          <span className="ws-identity-mark">
            <TerminalCliBadge term={t} />
            {terminalStatus(t) === "working" ? <span className="ws-activity-dot" aria-hidden="true" /> : null}
          </span>
          <span className="ws-copy">
            <span className="ws-title" title={t.name || "Terminal"}>{t.name || "Terminal"}</span>
            <span className="ws-subtitle">{cli ? cliLabel : "Shell session"}</span>
          </span>
          <TerminalStatus term={t} />
        </div>
        {actions ? (
          <RowMenu label={t.name || "Terminal"}>
            <RowMenuItem onSelect={() => onRenameTerm && onRenameTerm(t)}><IconPencil size={13} /> Rename</RowMenuItem>
            <RowMenuItem onSelect={() => { location.hash = "#/termset/" + encodeURIComponent(t.id); }}><IconSettings size={14} /> Terminal settings</RowMenuItem>
            <RowMenuItem danger onSelect={() => onRemoveTerm && onRemoveTerm(t)}><IconX size={13} /> Remove terminal</RowMenuItem>
          </RowMenu>
        ) : null}
      </div>
      <ContextLine line={line} ownerKind="term" ownerId={t.id} ownerLabel={t.name || "Terminal"} onFileTree={onFileTree} onGitGraph={onGitGraph} />
    </li>
  );
}

// The agent's internal checklist as one operator line (ADR-0055): the
// current step with its position, or a discrete "No checklist" when the
// contract was not met. Nothing known → nothing shown.
export function ChecklistLine({ line }) {
  if (!line) return null;
  if (line.kind === "absent") {
    return <div className="ws-check absent" title="The task required a checklist and none was written"><span className="ws-check-text">No checklist</span></div>;
  }
  const pos = "(" + line.position + "/" + line.total + ")";
  return (
    <div className="ws-check" title={pos + " " + line.text}>
      <span className="ws-check-pos">{pos}</span>
      <span className="ws-check-text">{line.text}</span>
    </div>
  );
}
