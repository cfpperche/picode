import Conversation from "./Conversation.jsx";
import Composer from "./Composer.jsx";

export default function ChatSurface({
  hidden, stopped, items, onToggleTool, onToggleFiles, convRef, onScroll,
  composer, onRun, onOpenTerm, catalog, agent, onConfig, onSlash,
}) {
  const hasChat = items.some((it) => it.kind === "block" || it.kind === "tool");
  const cfg = {
    provider: (agent && agent.provider) || "",
    model: (agent && agent.model) || "",
    thinking: (agent && agent.thinking) || "",
    opMode: (agent && agent.opMode) || "full",
  };
  return (
    <section id="chat-surface" className="chat-surface" hidden={hidden}>
      <div className="chat-body">
        <div id="run-cta" className="run-cta" hidden={!stopped}>
          <div className="run-cta-card">
            <p>Agent is stopped.</p>
            <div className="run-cta-actions">
              <button id="btn-run-agent" className="btn btn-primary" onClick={onRun}>Run agent</button>
              <button id="btn-term-agent" className="btn" onClick={onOpenTerm}>Open terminal</button>
            </div>
          </div>
        </div>
        <Conversation items={items} onToggleTool={onToggleTool} onToggleFiles={onToggleFiles} convRef={convRef} onScroll={onScroll} hidden={stopped && !hasChat} />
        <Composer {...composer} stopped={stopped} catalog={catalog} cfg={cfg} onConfig={onConfig} onSlash={onSlash} />
      </div>
    </section>
  );
}
