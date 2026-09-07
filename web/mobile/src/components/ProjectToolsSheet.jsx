import * as Sheet from "./MobileSheet.jsx";
import { IconFolder, IconGit } from "./Icons.jsx";

export default function ProjectToolsSheet({ open, onOpenChange, title, onFiles, onGit }) {
  const choose = action => { onOpenChange(false); action(); };
  return <Sheet.Root open={open} onOpenChange={onOpenChange}><Sheet.Portal>
    <Sheet.Overlay className="dlg-overlay" />
    <Sheet.Content className="dlg m-project-tools" onCloseAutoFocus={event => event.preventDefault()}>
      <Sheet.Title className="dlg-title">Project tools</Sheet.Title>
      <Sheet.Description className="m-project-tools-name">{title}</Sheet.Description>
      <button type="button" className="m-tool-action" onClick={() => choose(onFiles)}><IconFolder size={18} /><span>Files</span></button>
      <button type="button" className="m-tool-action" onClick={() => choose(onGit)}><IconGit size={18} /><span>Git</span></button>
      <Sheet.Close asChild><button type="button" className="btn btn-sm">Done</button></Sheet.Close>
    </Sheet.Content>
  </Sheet.Portal></Sheet.Root>;
}
