import { ProviderFace } from "./ProviderFaces.jsx";
import { IconChevronRight, IconPlay, IconStop } from "./Icons.jsx";
import { displayAgentName } from "@picode/shared/domain/tree.js";
import { shortModel } from "@picode/shared/domain/chip.js";
import { shortPath } from "@picode/shared/domain/repoLine.js";
import StateChip, { agentState } from "./StateChip.jsx";
import { checklistLine } from "@picode/shared/domain/checklist.js";
import { agentIsPi, agentSubtitle } from "@picode/shared/domain/managedPrincipal.js";
import { terminalStatus } from "@picode/shared/domain/terminalCli.js";

// Mobile keeps the same identity → activity → location hierarchy as the
// desktop rail, but retains a 44px Start/Stop target at the edge.
export default function AgentRow({ agent, workspace, workingIds, checklist, onOpen, onStart, onStop, busy, terms }) {
  const state = agentState(agent, workingIds);
  // A CLI agent's process is its bound terminal (ADR-0160); runMode never sees
  // the terminal's tmux session, so the chip and the Start/Stop button read
  // the terminal when there is one.
  const cliTerm = !agentIsPi(agent) && agent.terminalId ? (terms || []).find((t) => t.id === agent.terminalId) : null;
  const cliRunning = !!cliTerm && terminalStatus(cliTerm) !== "stopped";
  const effective = cliRunning ? "idle" : state;
  const name = displayAgentName(agent, workspace);
  const model = shortModel(agent.model || "");
  const check = checklistLine(checklist);
  const path = agent.workPath || (workspace && workspace.path) || "";
  const context = [model, path === workspace?.path ? "" : shortPath(path)].filter(Boolean).join(" · ");
  return (
    <li className={"m-row m-agent-row is-" + effective}>
      <button type="button" className="m-row-main" onClick={() => onOpen(agent)}>
        <span className="m-row-face"><ProviderFace agent={agent} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{name}</span>
          <span className="m-row-sub">{check && check.kind === "step" ? check.position + "/" + check.total + " · " + check.text : context || agentSubtitle(agent)}</span>
          {check && check.kind === "step" ? <span className="m-row-context">{context || agentSubtitle(agent)}</span> : null}
        </span>
        <StateChip state={effective} />
        <IconChevronRight size={16} className="m-row-chev" />
      </button>
      {effective === "stopped" ? (
        <button type="button" className="m-row-act is-start" title="Start" aria-label={"Start " + name} disabled={busy} onClick={() => onStart(agent, workspace)}><IconPlay size={18} /></button>
      ) : (
        <button type="button" className="m-row-act is-stop" title="Stop" aria-label={"Stop " + name} disabled={busy} onClick={() => onStop(agent, workspace)}><IconStop size={16} /></button>
      )}
    </li>
  );
}
