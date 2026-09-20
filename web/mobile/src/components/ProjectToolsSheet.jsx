import * as Sheet from "./MobileSheet.jsx";
import { IconClip, IconFolder, IconGit, IconPanelRight } from "./Icons.jsx";

export default function ProjectToolsSheet({ open, onOpenChange, title, onInspect, onFiles, onGit, onAttach }) {
  const choose = action => { onOpenChange(false); action(); };
  return <Sheet.Root open={open} onOpenChange={onOpenChange}><Sheet.Portal>
    <Sheet.Overlay className="dlg-overlay" />
    <Sheet.Content className="dlg m-project-tools" onCloseAutoFocus={event => event.preventDefault()}>
      <Sheet.Title className="dlg-title">Project tools</Sheet.Title>
      <Sheet.Description className="m-project-tools-name">{title}</Sheet.Description>
      {onInspect ? <button type="button" className="m-tool-action" onClick={() => choose(onInspect)}><IconPanelRight size={18} /><span>Inspect changes</span></button> : null}
      {onAttach ? <button type="button" className="m-tool-action" onClick={() => choose(onAttach)}><IconClip size={18} /><span>Attach to terminal…</span></button> : null}
      <button type="button" className="m-tool-action" onClick={() => choose(onFiles)}><IconFolder size={18} /><span>Files</span></button>
      <button type="button" className="m-tool-action" onClick={() => choose(onGit)}><IconGit size={18} /><span>Git</span></button>
      <Sheet.Close asChild><button type="button" className="btn btn-sm">Done</button></Sheet.Close>
    </Sheet.Content>
  </Sheet.Portal></Sheet.Root>;
}
