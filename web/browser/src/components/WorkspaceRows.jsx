import { Fragment, useEffect, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconChat, IconChevronRight, IconEllipsis, IconFolder, IconFork, IconGit, IconMode, IconMoveDown, IconMoveUp, IconPencil, IconPlay, IconReload, IconSettings, IconStop, IconTerminal, IconX } from "./Icons.jsx";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { repoLine, termLine } from "@picode/shared/domain/repoLine.js";
import { relTime, absTime } from "@picode/shared/domain/relTime.js";
import { ProviderFace } from "./ProviderFaces.jsx";
import PiSpinner from "./PiSpinner.jsx";
import { checklistLine, checklistProgress, checklistRows, countDone } from "@picode/shared/domain/checklist.js";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { terminalActivityStamp, terminalCli, terminalCliLabel, terminalDisplayCli, terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { agentRowStatus, agentStatusLabel, agentTerm } from "@picode/shared/domain/agentStatus.js";
import { agentRowMenu, agentHandoffTerm } from "@picode/shared/domain/agentRowMenu.js";
import { agentSubtitle, forkLine } from "@picode/shared/domain/managedPrincipal.js";
import { termRowMenu } from "../lib/termRowMenu.js";

export function RowMenu({ label, children, onOpenChange, onCloseAutoFocus, triggerRef }) {
  return (
    <DropdownMenu.Root onOpenChange={onOpenChange}>
      <DropdownMenu.Trigger asChild>
        <button
          ref={triggerRef}
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
        <DropdownMenu.Content className="ws-row-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8} onCloseAutoFocus={onCloseAutoFocus}>
          {children}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}

export function RowMenuItem({ children, onSelect, danger = false, title }) {
  return <DropdownMenu.Item className={"ws-row-menu-item" + (danger ? " danger" : "")} onSelect={onSelect} title={title}>{children}</DropdownMenu.Item>;
}

export function RowMenuSep() {
  return <DropdownMenu.Separator className="ws-row-menu-sep" />;
}

// One icon per merged terminal-menu row (termRowMenu.js); the labels and
// order live there, so the sidebar and the Agent CLIs list cannot drift.
const TERM_ROW_MENU_ICONS = {
  rename: <IconPencil size={13} />,
  launch: <IconSettings size={14} />,
  settings: <IconMode size={14} />,
  handoff: <IconChat size={13} />,
  start: <IconPlay size={12} />,
  restart: <IconReload size={13} />,
  stop: <IconStop size={12} />,
  remove: <IconX size={13} />,
};

// Same contract as TERM_ROW_MENU_ICONS: labels and order live in
// agentRowMenu.js, this map is only the renderer's business.
const AGENT_ROW_MENU_ICONS = {
  start: <IconPlay size={14} />,
  stop: <IconStop size={13} />,
  restart: <IconReload size={13} />,
  launch: <IconSettings size={14} />,
  settings: <IconSettings size={14} />,
  fork: <IconFork size={13} />,
  handoff: <IconChat size={13} />,
  chat: <IconChat size={14} />,
  term: <IconTerminal size={14} />,
  rename: <IconPencil size={13} />,
  remove: <IconX size={13} />,
};

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
      <span>{agentStatusLabel(status)}</span>
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
export function OrderMoves({ up, down }) {
  if (!up && !down) return null;
  return (
    <>
      {up ? <RowMenuItem onSelect={up}><IconMoveUp size={13} /> Move up</RowMenuItem> : null}
      {down ? <RowMenuItem onSelect={down}><IconMoveDown size={13} /> Move down</RowMenuItem> : null}
      <RowMenuSep />
    </>
  );
}

export function AgentRow({
  agent: ag, ws,
  selectedId, onSelect,
  workingId, workingIds, waitingId, checklists,
  onFileTree, onGitGraph,
  actions = true, meta = false,
  onRenameAgent, onRun, onRemoveAgent, onRemove, onChat, onTerm, termView,
  clis, terms, onLaunchAction, onContinueTerm, onForkAgent,
  drag, onMoveUp, onMoveDown,
}) {
  const mode = ag.mode || "stopped";
  // A CLI agent's process is its bound terminal (ADR-0160): the status pill
  // and the lifecycle rows read the terminal, because runMode never sees
  // the terminal's tmux session.
  const term = agentTerm(ag, terms);
  const label = displayAgentName(ag, ws);
  const model = shortModel(ag.model || "");
  const title = model ? label + " — " + model : label;
  const repo = repoLine(ag, ws);
  const stamp = term && terminalActivityStamp(term) || ag.lastStatusAt || ag.lastStartedAt || ag.createdAt;
  const check = checklists && checklists[ag.id];
  const status = agentRowStatus(ag, { workingId, workingIds, waitingId, term });
  const select = () => onSelect(ag.id);
  const onMenuItem = (r) => {
    switch (r.id) {
      case "start": return onRun && onRun(ag.id);
      case "stop":
      case "restart": return onLaunchAction && onLaunchAction(term, r.id, ag);
      case "mission":
      case "settings":
      case "launch": location.hash = r.href; return;
      case "chat": return onChat && onChat(ag.id);
      case "term": return onTerm && onTerm(ag.id);
      case "fork": return onForkAgent && onForkAgent(ag, term);
      case "rename": return onRenameAgent && onRenameAgent(ag, label);
      case "remove": return onRemoveAgent ? onRemoveAgent(ag) : onRemove(ws);
      default: return undefined;
    }
  };
  return (
    <li
      ref={drag ? drag.setNodeRef : undefined}
      style={drag ? drag.style : undefined}
      data-drag-id={ag.id}
      className={"ws-item is-" + status + (ag.id === selectedId ? " active" : "") + (drag && drag.isDragging ? " is-placeholder" : "")}
    >
      <div className="ws-row-main">
        <div
          className="ws-row-hit"
          role="button"
          tabIndex={0}
          aria-current={ag.id === selectedId ? "page" : undefined}
          aria-label={title}
          aria-describedby={drag ? drag.describedBy : undefined}
          onClick={select}
          onKeyDown={(e) => openRow(e, select)}
          onPointerDown={drag ? drag.onPointerDown : undefined}
        >
          <span className="ws-identity-mark">
            <ProviderFace agent={ag} />
            {status === "working" ? <span className="ws-activity-dot" aria-hidden="true" /> : null}
          </span>
          <span className="ws-copy">
            <span className="ws-title" title={title}>{label}</span>
            <span className="ws-subtitle">{model || agentSubtitle(ag)}</span>
          </span>
          {term ? <TerminalStatus term={term} /> : <AgentStatus status={status} stamp={stamp} />}
        </div>
        {actions ? (
          <RowMenu label={label}>
            <OrderMoves up={onMoveUp} down={onMoveDown} />
            {agentRowMenu(ag, { clis, term }).map((r, i) => {
              if (r.sep) return <RowMenuSep key={"sep" + i} />;
              if (r.sub) return (
                <DropdownMenu.Sub key={r.id}>
                  <DropdownMenu.SubTrigger className="ws-row-menu-item" title={r.title}>
                    {AGENT_ROW_MENU_ICONS[r.id] || null} {r.label}
                    <IconChevronRight size={13} className="um-chev" />
                  </DropdownMenu.SubTrigger>
                  <DropdownMenu.Portal>
                    <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                      {r.sub.map((s) => s.sub ? (
                        <DropdownMenu.Sub key={s.id}>
                          <DropdownMenu.SubTrigger className="ws-row-menu-item" title={s.title}>
                            {s.label}
                            <IconChevronRight size={13} className="um-chev" />
                          </DropdownMenu.SubTrigger>
                          <DropdownMenu.Portal>
                            <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                              {s.sub.map((s2) => (
                                <DropdownMenu.Item
                                  key={s2.id}
                                  className="ws-row-menu-item"
                                  title={s2.title}
                                  onSelect={() => onContinueTerm && onContinueTerm(term || agentHandoffTerm(ag), s2.target)}
                                >
                                  {s2.label}
                                </DropdownMenu.Item>
                              ))}
                            </DropdownMenu.SubContent>
                          </DropdownMenu.Portal>
                        </DropdownMenu.Sub>
                      ) : (
                        <DropdownMenu.Item
                          key={s.id}
                          className="ws-row-menu-item"
                          title={s.title}
                          onSelect={() => onContinueTerm && onContinueTerm(term || agentHandoffTerm(ag), s.target)}
                        >
                          {s.label}
                        </DropdownMenu.Item>
                      ))}
                    </DropdownMenu.SubContent>
                  </DropdownMenu.Portal>
                </DropdownMenu.Sub>
              );
              return (
                <Fragment key={r.id}>
                  {r.danger ? <RowMenuSep /> : null}
                  <RowMenuItem title={r.title} danger={r.danger} onSelect={() => onMenuItem(r)}>
                    {AGENT_ROW_MENU_ICONS[r.id]} {r.label}
                  </RowMenuItem>
                </Fragment>
              );
            })}
          </RowMenu>
        ) : null}
      </div>
      {/* Its own line, as wide as the folder line: beside the status chip
          the subtitle has ~100px and cut "fork of <name>" to one word. */}
      {forkLine(ag) ? (
        <span className="ws-fork-line" title={"Forked from " + ag.forkedFrom.name + (ag.forkedFrom.gone ? ", which was removed" : "")}>
          <IconFork size={11} /><span>{forkLine(ag)}</span>
        </span>
      ) : null}
      <ChecklistDisclosure id={ag.id} check={check} />
      <ContextLine line={repo} ownerKind="agent" ownerId={ag.id} ownerLabel={label} onFileTree={onFileTree} onGitGraph={onGitGraph} />
      {meta && stamp ? <span className="ws-meta" title={absTime(stamp)}>{relTime(stamp)}</span> : null}
    </li>
  );
}

// Terminal rows use the same identity → activity → location rhythm as agent
// rows. `tui` is authoritative when present; legacy top-level cli/state is
// still rendered so older sessions degrade visibly rather than disappearing.
// A pi running inside the terminal (Agent CLIs, ADR-0069) publishes its
// checklist like a managed agent, so the card carries the same plan line.
export function TermRow({
  term: t,
  selectedId, onSelectTerm,
  onFileTree, onGitGraph,
  actions = true,
  onRenameTerm, onRemoveTerm, onLaunchAction, onContinueTerm, clis,
  drag, onMoveUp, onMoveDown,
}) {
  const line = termLine(t);
  const cli = terminalDisplayCli(t);
  const cliLabel = cli ? terminalCliLabel(cli) : "Terminal";
  const check = t.checklist;
  const selected = selectedId === "t:" + t.id;
  const select = () => onSelectTerm && onSelectTerm(t.id);
  return (
    <li
      ref={drag ? drag.setNodeRef : undefined}
      style={drag ? drag.style : undefined}
      data-drag-id={t.id}
      className={"ws-item is-terminal is-" + terminalStatus(t) + (selected ? " active" : "") + (drag && drag.isDragging ? " is-placeholder" : "")}
    >
      <div className="ws-row-main">
        <div className="ws-row-hit" role="button" tabIndex={0} aria-current={selected ? "page" : undefined} aria-describedby={drag ? drag.describedBy : undefined} aria-label={(t.name || "Terminal") + " — " + cliLabel} onClick={select} onKeyDown={(e) => openRow(e, select)} onPointerDown={drag ? drag.onPointerDown : undefined}>
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
            <OrderMoves up={onMoveUp} down={onMoveDown} />
            {termRowMenu(t, { clis }).map((r, i) => r.sep ? <RowMenuSep key={"sep" + i} /> : r.sub ? (
              <DropdownMenu.Sub key={r.id}>
                <DropdownMenu.SubTrigger className="ws-row-menu-item" title={r.title}>
                  {TERM_ROW_MENU_ICONS[r.id]} {r.label}
                  <IconChevronRight size={13} className="um-chev" />
                </DropdownMenu.SubTrigger>
                <DropdownMenu.Portal>
                  <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                    {r.sub.map((s) => s.sub ? (
                      <DropdownMenu.Sub key={s.id}>
                        <DropdownMenu.SubTrigger className="ws-row-menu-item" title={s.title}>
                          {s.label}
                          <IconChevronRight size={13} className="um-chev" />
                        </DropdownMenu.SubTrigger>
                        <DropdownMenu.Portal>
                          <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                            {s.sub.map((s2) => (
                              <DropdownMenu.Item
                                key={s2.id}
                                className="ws-row-menu-item"
                                disabled={!!s2.disabled}
                                title={s2.title}
                                onSelect={() => { if (!s2.disabled && onContinueTerm) onContinueTerm(t, s2.target); }}
                              >
                                {s2.label}
                              </DropdownMenu.Item>
                            ))}
                          </DropdownMenu.SubContent>
                        </DropdownMenu.Portal>
                      </DropdownMenu.Sub>
                    ) : (
                      <DropdownMenu.Item
                        key={s.id}
                        className="ws-row-menu-item"
                        disabled={!!s.disabled}
                        title={s.title}
                        onSelect={() => { if (!s.disabled && onContinueTerm) onContinueTerm(t, s.target); }}
                      >
                        {s.label}
                      </DropdownMenu.Item>
                    ))}
                  </DropdownMenu.SubContent>
                </DropdownMenu.Portal>
              </DropdownMenu.Sub>
            ) : (
              <RowMenuItem
                key={r.id}
                title={r.title}
                danger={r.danger}
                onSelect={() => {
                  if (r.id === "rename") return onRenameTerm && onRenameTerm(t);
                  if (r.id === "launch") { location.hash = "#/clis/terminal/" + encodeURIComponent(t.id); return; }
                  if (r.id === "settings") { location.hash = "#/termset/" + encodeURIComponent(t.id); return; }
                  if (r.id === "remove") return onRemoveTerm && onRemoveTerm(t);
                  return onLaunchAction && onLaunchAction(t, r.id); // start | restart | stop
                }}
              >
                {TERM_ROW_MENU_ICONS[r.id]} {r.label}
              </RowMenuItem>
            ))}
          </RowMenu>
        ) : null}
      </div>
      <ChecklistDisclosure id={t.id} check={check} />
      <ContextLine line={line} ownerKind="term" ownerId={t.id} ownerLabel={t.name || "Terminal"} onFileTree={onFileTree} onGitGraph={onGitGraph} />
    </li>
  );
}

// The agent's internal checklist as one operator line (ADR-0055): the
// current step with its position. Nothing known, and an absent marker (the
// contract was not met), both render nothing — silence is not absence and
// absence is not worth a line (ADR-0092, owner refinement). No parens
// around the counter — this is a list row, not a terminal (owner refinement,
// docs/plans/sidebar-checklist-expand.md). The terminal pane does not
// repeat this line (ADR-0081 amendment 2026-09-07).
export function ChecklistLine({ line }) {
  const progress = checklistProgress(line);
  if (!progress) return null;
  return (
    <div className="ws-check" title={progress.pos + " · " + progress.text}>
      <span className="ws-check-text">{progress.text}</span>
      <span className="ws-check-pos">{progress.pos}</span>
    </div>
  );
}

// The same plan as a disclosure (docs/plans/sidebar-checklist-expand.md):
// collapsed it is the operator line; expanded it lists every step — ☑ on
// finished ones, the braille spinner on the step being executed, ☐ on the
// rest. The items are already in client state and arrive live over the
// feed, so opening costs no fetch and updates render in place. Absent and
// unknown checklists render nothing (ADR-0092) — there is nothing to open.
//
// Layout matches ContextLine below it: the card's 23px metadata column,
// no parens around the counter, counter reads as a fixed column at the row's end
// (Linear/GitHub sub-issue idiom) instead of a prefix. A finished plan
// (position === total and every item completed) dims to the same weight as
// a completed step, so it stops reading as an open task.
export function ChecklistDisclosure({ id, check }) {
  const [open, setOpen] = useState(false);
  const line = checklistLine(check);
  const progress = checklistProgress(line);
  if (!progress) return null;
  const listId = "chk-" + id;
  const items = (check && check.items) || [];
  const done = line.position === line.total && items.length > 0 && countDone(items) === items.length;
  return (
    <div className="ws-check-disclosure">
      <button
        type="button"
        className={"ws-check-line" + (done ? " is-done" : "")}
        aria-expanded={open}
        aria-controls={listId}
        aria-label={"Plan, step " + progress.pos + ": " + progress.text}
        title={progress.pos + " · " + progress.text}
        onClick={(e) => { e.stopPropagation(); setOpen((v) => !v); }}
        onKeyDown={(e) => { if (e.key === "Escape") { e.stopPropagation(); setOpen(false); } }}
      >
        {/* No chevron (owner request 2026-09-09): the line reads as plan
            text, not a disclosure widget — the whole row stays the click
            target, so expanding needs no arrow. The line carries no mark
            of its own, so its text starts on the card's 23px metadata
            column — the same left edge as the title, subtitle and the
            folder/branch icons below (owner report 2026-09-09: the ghost
            slot read as an indent, not an alignment). The expanded steps'
            glyphs keep that column for the marks, 39px for their text. */}
        <span className="ws-check-text">{progress.text}</span>
        <span className="ws-check-pos">{progress.pos}</span>
      </button>
      <div id={listId} className={"ws-check-list" + (open ? " open" : "")} aria-hidden={!open}>
        <ul>
          {checklistRows(check.items).map((row) => (
            <li key={row.key} className={row.status + (row.current ? " current" : "")}>
              <span className="ws-check-step-glyph" aria-hidden="true">
                {row.current ? <PiSpinner title="In progress" /> : row.glyph}
              </span>
              <span className="ws-check-step-text" title={row.text}>{row.text}</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
