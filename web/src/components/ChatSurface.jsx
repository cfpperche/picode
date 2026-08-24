import Conversation from "./Conversation.jsx";
import Composer from "./Composer.jsx";
import ProviderChip from "./ProviderChip.jsx";
import ModelChip from "./ModelChip.jsx";
import ThinkingChip from "./ThinkingChip.jsx";
import ModeChip from "./ModeChip.jsx";

export default function ChatSurface({
  hidden, stopped, items, onToggleTool, onToggleFiles, convRef, onScroll,
  composer, onRun, catalog, agent, onConfig, onSlash,
}) {
  const hasChat = items.some((it) => it.kind === "block" || it.kind === "tool");
  const cfg = {
    provider: (agent && agent.provider) || "",
    model: (agent && agent.model) || "",
    thinking: (agent && agent.thinking) || "",
    opMode: (agent && agent.opMode) || "full",
  };
  const onCfg = onConfig || (() => {});
  return (
    <section id="chat-surface" className="chat-surface" hidden={hidden}>
      <div className="chat-body">
        <Conversation items={items} onToggleTool={onToggleTool} onToggleFiles={onToggleFiles} convRef={convRef} onScroll={onScroll} hidden={stopped && !hasChat} />
        {stopped ? (
          <div className="composer-wrap">
            <div className="composer" id="run-cta">
              <p className="stopped-line">Agent is stopped. Run it to send a message.</p>
              <div className="composer-controls">
                <div className="composer-left">
                  <ProviderChip catalog={catalog} cfg={cfg} onChange={onCfg} />
                  <ModelChip catalog={catalog} cfg={cfg} onChange={onCfg} />
                  <ThinkingChip catalog={catalog} cfg={cfg} onChange={onCfg} />
                  <ModeChip cfg={cfg} onChange={onCfg} />
                </div>
                <div className="composer-right">
                  <button id="btn-run-agent" type="button" className="btn btn-primary btn-sm" onClick={onRun}>Run agent</button>
                </div>
              </div>
            </div>
          </div>
        ) : (
          <Composer {...composer} stopped={false} catalog={catalog} cfg={cfg} onConfig={onConfig} onSlash={onSlash} />
        )}
      </div>
    </section>
  );
}
