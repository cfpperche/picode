import { relTime } from "@picode/shared/domain/relTime.js";
import TerminalPanel from "./TerminalPanel.jsx";

// PanelBody — the slot under a panel's header (docs/plans/matrix-app.md
// §4.4, one row each). Loaded: the live pane (TerminalPanel). Unloaded: a
// muted placeholder with the feed's last state ("Working · 2 min") — the
// header stays live either way. A binding the fleet cannot serve is one
// line and one action, whether loaded or not.
function Line({ text, action, onAction }) {
  return (
    <div className="mx-placeholder mx-state" role="status">
      <span>{text}</span>
      <button type="button" className="btn btn-sm" onClick={onAction}>{action}</button>
    </div>
  );
}

export default function PanelBody({ model, loaded, hidden, focused, onOpen, onRemove, onRun, onOpenFile }) {
  switch (model.state) {
    case "terminal-gone": return <Line text="That terminal is gone." action="Remove" onAction={onRemove} />;
    case "agent-gone": return <Line text="That agent is gone." action="Remove" onAction={onRemove} />;
    case "agent-stopped": return <Line text="Agent is stopped." action="Run" onAction={onRun} />;
    case "agent-managed": return <Line text="Managed agent — open to read." action="Open" onAction={onOpen} />;
    default: break;
  }
  if (model.pending) return <div className="mx-placeholder"><span>Adding…</span></div>;
  const age = model.status === "working" && model.stamp ? relTime(model.stamp) : "";
  const placeholder = (
    <div className="mx-placeholder" aria-hidden="true">
      <span>{model.label}{age ? " · " + age : ""}</span>
    </div>
  );
  if (!loaded) return placeholder;
  return (
    <TerminalPanel
      kind={model.kind}
      target={model.target}
      cwd={model.cwd}
      hidden={hidden}
      focused={focused}
      owned={model.owned}
      onOpenFile={onOpenFile}
      placeholder={placeholder}
    />
  );
}
