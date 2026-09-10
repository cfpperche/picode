import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconAgent, IconChat, IconEllipsis, IconFolder, IconGit, IconTerminal } from "./Icons.jsx";

// The five per-workspace actions collapse into one ellipsis menu on the
// group head (HIG), so no row inside Work can grow wider than the phone
// screen — the old action strip forced a page-wide horizontal scroll.
export default function WorkspaceMenu({ ws, onCreate, onNewTerm, onOpenFiles, onOpenGit }) {
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button type="button" className="m-work-menu-btn" aria-label={"Actions for " + ws.name}><IconEllipsis size={18} /></button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className="ws-row-menu m-settings-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
          <DropdownMenu.Item asChild><a className="ws-row-menu-item" href={"#/clis/messages/" + encodeURIComponent("workspace:" + ws.id)}><IconChat size={14} /> Communication</a></DropdownMenu.Item>
          <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => onOpenFiles({ kind: "workspace", id: ws.id })}><IconFolder size={14} /> Files</DropdownMenu.Item>
          <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => onOpenGit({ kind: "workspace", id: ws.id })}><IconGit size={14} /> Git</DropdownMenu.Item>
          <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => onCreate("agent", ws)}><IconAgent size={14} /> New agent</DropdownMenu.Item>
          <DropdownMenu.Item className="ws-row-menu-item" onSelect={() => onNewTerm(ws)}><IconTerminal size={14} /> New terminal</DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
