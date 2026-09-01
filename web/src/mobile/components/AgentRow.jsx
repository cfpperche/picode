import { ProviderFace } from "../../components/ProviderFaces.jsx";
import { IconChevronRight, IconPlay, IconStop } from "../../components/Icons.jsx";
import { displayAgentName } from "../../lib/tree.js";
import { shortModel } from "../../lib/chip.js";
import StateChip, { agentState } from "./StateChip.jsx";

// One agent, 56px, whole row opens it; Start/Stop is the one action that
// does not need the screen (44px target, right edge).
export default function AgentRow({ agent, workspace, workingIds, onOpen, onStart, onStop, busy }) {
  const state = agentState(agent, workingIds);
  const name = displayAgentName(agent, workspace);
  const model = shortModel(agent.model || "");
  const stopped = state === "stopped";
  return (
    <li className={"m-row m-agent-row is-" + state}>
      <button type="button" className="m-row-main" onClick={() => onOpen(agent)}>
        <span className="m-row-face"><ProviderFace agent={agent} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{name}</span>
          <span className="m-row-sub">{[model, workspace ? workspace.name : "free agent"].filter(Boolean).join(" · ")}</span>
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
