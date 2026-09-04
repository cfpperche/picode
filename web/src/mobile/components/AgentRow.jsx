import { ProviderFace } from "../../components/ProviderFaces.jsx";
import { IconChevronRight, IconPlay, IconStop } from "../../components/Icons.jsx";
import { displayAgentName } from "../../lib/tree.js";
import { shortModel } from "../../lib/chip.js";
import { shortPath } from "../../lib/repoLine.js";
import StateChip, { agentState } from "./StateChip.jsx";
import { checklistLine } from "../../lib/checklist.js";

// Mobile keeps the same identity → activity → location hierarchy as the
// desktop rail, but retains a 44px Start/Stop target at the edge.
export default function AgentRow({ agent, workspace, workingIds, checklist, onOpen, onStart, onStop, busy }) {
  const state = agentState(agent, workingIds);
  const name = displayAgentName(agent, workspace);
  const model = shortModel(agent.model || "");
  const stopped = state === "stopped";
  const check = checklistLine(checklist);
  const path = agent.workPath || (workspace && workspace.path) || "";
  const context = [model, shortPath(path)].filter(Boolean).join(" · ");
  return (
    <li className={"m-row m-agent-row is-" + state}>
      <button type="button" className="m-row-main" onClick={() => onOpen(agent)}>
        <span className="m-row-face"><ProviderFace agent={agent} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{name}</span>
          <span className="m-row-sub">{check && check.kind === "step" ? "(" + check.position + "/" + check.total + ") " + check.text : context || (check ? "No checklist" : "Pi agent")}</span>
          {check && check.kind === "step" ? <span className="m-row-context">{context || "Pi agent"}</span> : null}
        </span>
        <StateChip state={state} />
        <IconChevronRight size={16} className="m-row-chev" />
      </button>
      {stopped ? (
        <button type="button" className="m-row-act is-start" title="Start" aria-label={"Start " + name} disabled={busy} onClick={() => onStart(agent, workspace)}><IconPlay size={18} /></button>
      ) : (
        <button type="button" className="m-row-act is-stop" title="Stop" aria-label={"Stop " + name} disabled={busy} onClick={() => onStop(agent, workspace)}><IconStop size={16} /></button>
      )}
    </li>
  );
}
