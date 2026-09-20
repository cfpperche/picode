import { IconChat, IconTerminal } from "./Icons.jsx";
import { agentIsPi } from "@picode/shared/domain/managedPrincipal.js";

// Pi is the only managed agent with both conversation surfaces. CLI agents
// already use their bound terminal as the conversation and do not expose a
// second, empty chat surface (ADR-0160).
export default function AgentViewToolbar({ agent, view, onView }) {
  if (!agent || !agentIsPi(agent) || !onView) return null;
  return (
    <div className="agent-view-toolbar" role="toolbar" aria-label="Agent view">
      <div className="agent-view-switch" role="tablist" aria-label="Agent view">
        <button type="button" role="tab" aria-label="Chat" title="Chat" aria-selected={view === "chat"} className="agent-view-tab" data-active={view === "chat" ? "true" : undefined} onClick={() => onView("chat")}>
          <IconChat size={14} />
        </button>
        <button type="button" role="tab" aria-label="Terminal" title="Terminal" aria-selected={view === "terminal"} className="agent-view-tab" data-active={view === "terminal" ? "true" : undefined} onClick={() => onView("terminal")}>
          <IconTerminal size={14} />
        </button>
      </div>
    </div>
  );
}
