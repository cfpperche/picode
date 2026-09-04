import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { IconChevronRight, IconEllipsis, IconTrash } from "../../components/Icons.jsx";
import { termLine } from "../../lib/repoLine.js";
import TerminalCliBadge from "../../components/TerminalCliBadge.jsx";
import { terminalActivityStamp, terminalCli, terminalCliLabel, terminalStatus, terminalStatusLabel } from "../../lib/terminalCli.js";
import { relTime } from "../../lib/relTime.js";

// One terminal, with the detected CLI as a stable identity and lifecycle as
// the right-hand status. No CLI means a plain shell, never an inferred agent.
export default function TermRow({ term, onOpen, onRemove, busy }) {
  const line = termLine(term);
  const cli = terminalCli(term);
  const status = terminalStatus(term);
  const stamp = terminalActivityStamp(term);
  const age = status === "working" && stamp ? " · " + relTime(stamp) : "";
  const subtitle = [cli ? terminalCliLabel(cli) : "Shell session", line.text].filter(Boolean).join(" · ");
  return (
    <li className={"m-row m-term-row is-" + status}>
      <button type="button" className="m-row-main" onClick={() => onOpen(term)}>
        <span className="m-row-face"><TerminalCliBadge term={term} /></span>
        <span className="m-row-text">
          <span className="m-row-title">{term.name || "Terminal"}</span>
          <span className="m-row-sub">{subtitle}</span>
        </span>
        <span className={"m-state is-" + status}>{terminalStatusLabel(term)}{age}</span>
        <IconChevronRight size={16} className="m-row-chev" />
      </button>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button type="button" className="m-row-act" title="Actions" aria-label={"Actions for " + (term.name || "terminal")}><IconEllipsis size={18} /></button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="ws-row-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
            <DropdownMenu.Item className="ws-row-menu-item danger" disabled={busy} onSelect={() => onRemove(term)}><IconTrash size={14} /> Remove terminal</DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  );
}
