import { CHAT_LIVE_MAX, hasChat, hasPane, parseRef } from "@picode/shared/domain/canvas.js";
import { relTime } from "@picode/shared/domain/relTime.js";
import TerminalPanel from "./TerminalPanel.jsx";
import NotePanel from "./NotePanel.jsx";
import TextPanel from "./TextPanel.jsx";
import FilePanel from "./FilePanel.jsx";
import DiffPanel from "./DiffPanel.jsx";
import AgentChatPanel from "./AgentChatPanel.jsx";

// PanelBody — the slot under a panel's header (docs/plans/matrix-app.md
// §4.4, one row each; the kinds beyond a live pane are
// docs/plans/matrix-canvas.md §4.2). Loaded, the body is what the panel's
// **kind** binds: a terminal or an agent is the live pane (TerminalPanel), a
// note is its pin's markdown (NotePanel), a file is the editor in its
// embedded layout (FilePanel), a diff is that path's working-tree patch
// (DiffPanel). Unloaded: a muted placeholder with
// the feed's last state ("Working · 2 min") — the header stays live either
// way. A binding the fleet cannot serve is one line and one action, whether
// loaded or not.
// One line and one action. Which weight that action carries is a rule, not
// a taste: **the action that starts work is accented, the action that only
// navigates or tidies is not**. So Run on a stopped agent is the same
// accent button the agent tab offers for the same verb (ChatSurface's
// "Run agent") — a stopped panel's Run used to be a plain chip beside it,
// which read as the disabled one of the pair — while Open (go to the tab)
// and Remove (drop a binding whose target is gone) stay quiet.
function Line({ text, action, onAction, primary = false }) {
  return (
    <div className="cv-placeholder cv-state" role="status">
      <span>{text}</span>
      <button type="button" className={"btn btn-sm" + (primary ? " btn-primary" : "")} onClick={onAction}>{action}</button>
    </div>
  );
}

// hasPane and hasChat are the domain's (web/shared/domain/canvas.js): the
// rows whose body *is* a terminal, and the row whose body holds an agent
// socket. Re-exported here because this is where the components already ask.
export { hasChat, hasPane };

export default function PanelBody({ model, loaded, body = "live", hidden, focused, onOpen, onRemove, onRun, onOpenFile, attach, onAttachClose, find, onFindClose, onDirty, onSaveText }) {
  switch (model.state) {
    case "terminal-gone": return <Line text="That terminal is gone." action="Remove" onAction={onRemove} />;
    case "agent-gone": return <Line text="That agent is gone." action="Remove" onAction={onRemove} />;
    case "agent-stopped": return <Line text="Agent is stopped." action="Run" onAction={onRun} primary />;
    case "agent-managed":
      // The conversation, read-only, exactly while the loader says this body
      // is live — in the band, at zoom 0.4 or more, under the chat cap. The
      // socket's lifetime is this component's mount, so the three other
      // answers are each one line and one action, and none of them costs a
      // connection. The header says Needs you at all four, because the chip
      // is the fleet's and not this socket's.
      if (loaded && body === "live") {
        return (
          <AgentChatPanel
            agentId={model.ref}
            workspaceId={model.wsId}
            ask={model.ask}
            onOpen={onOpen}
            onOpenTab={(path) => onOpenFile(path)}
          />
        );
      }
      if (body === "quiet") {
        return <Line text={"Paused — " + CHAT_LIVE_MAX + " conversations are already live."} action="Open" onAction={onOpen} />;
      }
      return <Line text="Managed agent — open to read." action="Open" onAction={onOpen} />;
    case "note-gone": return <Line text="That pin is gone." action="Remove" onAction={onRemove} />;
    case "file-gone": return <Line text="Where this file was read from is gone." action="Remove" onAction={onRemove} />;
    case "diff-gone": return <Line text="Where this file was read from is gone." action="Remove" onAction={onRemove} />;
    default: break;
  }
  if (model.pending) return <div className="cv-placeholder"><span>Adding…</span></div>;
  const age = model.status === "working" && model.stamp ? relTime(model.stamp) : "";
  const placeholder = (
    <div className="cv-placeholder" aria-hidden="true">
      <span>{model.label}{age ? " · " + age : ""}</span>
    </div>
  );
  if (!loaded) return placeholder;
  if (model.kind === "agent" && !model.runtimeId) return <Line text="Terminal unavailable." action="Open" onAction={onOpen} />;
  // Text is the one body with nothing to fetch: its words came down with the
  // panel row, so it never shows the placeholder and never waits.
  if (model.kind === "text") return <TextPanel panelId={model.id} content={model.content} onSave={onSaveText} />;
  if (model.kind === "note") return <NotePanel pinId={model.ref} title={model.name} />;
  if (model.kind === "file" || model.kind === "diff") {
    const at = parseRef(model.kind, model.ref);
    if (!at) return placeholder;
    if (model.kind === "diff") {
      return <DiffPanel owner={at.owner} path={at.path} watch={model.cwd} onOpenFile={() => onOpenFile(at.path)} />;
    }
    return <FilePanel owner={at.owner} path={at.path} refKey={model.ref} hidden={hidden} onDirty={onDirty} />;
  }
  return (
    <TerminalPanel
      key={(model.runtimeId || model.ref) + ":" + (model.runtimeEpoch || 0)}
      kind={model.kind}
      target={model.target}
      terminals={model.terminals}
      cwd={model.cwd}
      hidden={hidden}
      focused={focused}
      owned={model.owned}
      onOpenFile={onOpenFile}
      attach={attach}
      onAttachClose={onAttachClose}
      find={find}
      onFindClose={onFindClose}
      placeholder={placeholder}
    />
  );
}
