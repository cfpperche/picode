import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { ProviderFace } from "./ProviderFaces.jsx";
import {
  IconChevronRight, IconEllipsis, IconChat, IconMonitor, IconPencil,
  IconPlay, IconReload, IconStop, IconTrash,
} from "./Icons.jsx";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import StateChip, { agentState } from "./StateChip.jsx";
import { checklistLine } from "@picode/shared/domain/checklist.js";
import { agentSubtitleWithFork, forkLine } from "@picode/shared/domain/managedPrincipal.js";
import { terminalActivityStamp } from "@picode/shared/domain/terminalCli.js";
import { agentRowMenu, agentHandoffTerm } from "@picode/shared/domain/agentRowMenu.js";
import { relTime } from "@picode/shared/domain/relTime.js";

const ICONS = {
  restart: <IconReload size={14} />,
  start: <IconPlay size={14} />,
  stop: <IconStop size={14} />,
  chat: <IconChat size={14} />,
  term: <IconMonitor size={14} />,
  rename: <IconPencil size={14} />,
  remove: <IconTrash size={14} />,
};

// One agent on the Work list, at term-row parity (owner, 2026-09-20):
// the same ⋯ menu the terminal card has (agentRowMenu — lifecycle,
// open chat/terminal, rename, remove, Continue in…), and a status chip
// that reports the agent's real state with an activity age — a CLI
// agent's process is its bound terminal (ADR-0160), so the chip reads
// the terminal the way the desktop rail does. The main hit opens the
// agent (chat, or its terminal for CLI agents).
export default function AgentRow({ agent, workspace, workingIds, checklist, busy, terms, clis, onOpen, onAction = () => {} }) {
  const term = (terms || []).find((t) => t.id === agent.terminalId) || agent.terminal || null;
  const status = agentState({ ...agent, terminal: term }, workingIds);
  const stamp = (term && terminalActivityStamp(term)) || agent.lastStatusAt || agent.lastStartedAt || "";
  const age = status === "working" && stamp ? " · " + relTime(stamp) : "";
  const name = displayAgentName(agent, workspace);
  const model = shortModel(agent.model || "");
  const check = checklistLine(checklist);
  const path = agent.workPath || (workspace && workspace.path) || "";
  // Model and folder when they say something; else the CLI. A fork adds
  // where it came from to whichever line is shown.
  const base = [model, path === workspace?.path ? "" : shortPath(path)].filter(Boolean).join(" · ");
  const context = base ? [base, forkLine(agent)].filter(Boolean).join(" · ") : "";
  // A legacy interactive Pi pane has no bound terminal to restart; the
  // row drops the item instead of offering a no-op. Fork agent… has no
  // phone sheet yet (docs/handoff/open/agent-fork.md), so it stays desktop.
  const rows = agentRowMenu(agent, { clis, term }).filter((r) => (r.id !== "restart" || agent.terminalId) && r.id !== "fork");
  return (
    <li className={"m-row m-agent-row is-" + status}>
      <button type="button" className="m-row-main" onClick={() => onOpen(agent)}>
        <span className="m-row-face"><ProviderFace agent={agent} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{name}</span>
          <span className="m-row-sub">{check && check.kind === "step" ? check.position + "/" + check.total + " · " + check.text : context || agentSubtitleWithFork(agent)}</span>
          {check && check.kind === "step" ? <span className="m-row-context">{context || agentSubtitleWithFork(agent)}</span> : null}
        </span>
        <StateChip state={status} age={age} />
        <IconChevronRight size={16} className="m-row-chev" />
      </button>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button type="button" className="m-row-act" title="Actions" aria-label={"Actions for " + name} disabled={busy}><IconEllipsis size={18} /></button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="ws-row-menu m-settings-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
            {rows.map((r, i) => r.sep ? <DropdownMenu.Separator key={"sep" + i} className="ws-row-menu-sep" /> : r.sub ? (
              <HandoffSub key={r.id} row={r} disabled={busy} onSelect={(target) => onAction(agent, "handoff", target)} />
            ) : (
              <DropdownMenu.Item
                key={r.id}
                className={"ws-row-menu-item" + (r.danger ? " danger" : "")}
                disabled={busy}
                onSelect={() => {
                  if (r.href) { location.hash = r.href; return; }
                  onAction(agent, r.id);
                }}
              >
                {ICONS[r.id] || null} {r.label}
              </DropdownMenu.Item>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  );
}

function HandoffSub({ row, disabled, onSelect }) {
  return (
    <DropdownMenu.Sub>
      <DropdownMenu.SubTrigger className="ws-row-menu-item" disabled={disabled}>
        {IconChat} {row.label}
        <IconChevronRight size={13} className="um-chev" />
      </DropdownMenu.SubTrigger>
      <DropdownMenu.Portal>
        <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
          {row.sub.map((s) => s.sub ? (
            <DropdownMenu.Sub key={s.id}>
              <DropdownMenu.SubTrigger className="ws-row-menu-item" disabled={disabled || !!s.disabled}>
                {s.label}
                <IconChevronRight size={13} className="um-chev" />
              </DropdownMenu.SubTrigger>
              <DropdownMenu.Portal>
                <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                  {s.sub.map((s2) => (
                    <DropdownMenu.Item key={s2.id} className="ws-row-menu-item" disabled={disabled || !!s2.disabled} onSelect={() => { if (!s2.disabled) onSelect(s2.target); }}>
                      {s2.label}
                    </DropdownMenu.Item>
                  ))}
                </DropdownMenu.SubContent>
              </DropdownMenu.Portal>
            </DropdownMenu.Sub>
          ) : (
            <DropdownMenu.Item key={s.id} className="ws-row-menu-item" disabled={disabled || !!s.disabled} onSelect={() => { if (!s.disabled) onSelect(s.target); }}>
              {s.label}
            </DropdownMenu.Item>
          ))}
        </DropdownMenu.SubContent>
      </DropdownMenu.Portal>
    </DropdownMenu.Sub>
  );
}
