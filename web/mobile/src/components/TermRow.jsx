import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import {
  IconChevronRight, IconEllipsis, IconTrash, IconPencil, IconPlay,
  IconReload, IconStop, IconChat,
} from "./Icons.jsx";
import { termLine } from "@picode/shared/domain/repoLine.js";
import { termRowMenu } from "@picode/shared/domain/termRowMenu.js";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { terminalActivityStamp, terminalCliLabel, terminalDisplayCli, terminalStatus, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { relTime } from "@picode/shared/domain/relTime.js";

const ICONS = {
  rename: <IconPencil size={14} />,
  handoff: <IconChat size={14} />,
  start: <IconPlay size={14} />,
  restart: <IconReload size={14} />,
  stop: <IconStop size={14} />,
  remove: <IconTrash size={14} />,
};

// One terminal on the Work list. The ⋯ is termRowMenu's phone surface:
// Rename, Continue in… when pinned, Start or Restart/Stop, Remove.
// Launch and Terminal settings stay on Agent CLIs / the open pane.
export default function TermRow({ term, onOpen, onAction = () => {}, busy, clis }) {
  const line = termLine(term);
  const cli = terminalDisplayCli(term);
  const status = terminalStatus(term);
  const stamp = terminalActivityStamp(term);
  const age = status === "working" && stamp ? " · " + relTime(stamp) : "";
  const subtitle = [cli ? terminalCliLabel(cli) : "Shell session", line.text].filter(Boolean).join(" · ");
  const rows = termRowMenu(term, { clis, surface: "phone" });
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
          <button type="button" className="m-row-act" title="Actions" aria-label={"Actions for " + (term.name || "terminal")} disabled={busy}><IconEllipsis size={18} /></button>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className="ws-row-menu m-settings-menu" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
            {rows.map((r, i) => r.sep ? <DropdownMenu.Separator key={"sep" + i} className="ws-row-menu-sep" /> : r.sub ? (
              <HandoffSub key={r.id} row={r} disabled={busy} onSelect={(target) => onAction(term, "handoff", target)} />
            ) : (
              <DropdownMenu.Item
                key={r.id}
                className={"ws-row-menu-item" + (r.danger ? " danger" : "")}
                disabled={busy || !!r.disabled}
                onSelect={() => onAction(term, r.id)}
              >
                {ICONS[r.id]} {r.label}
              </DropdownMenu.Item>
            ))}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </li>
  );
}

function HandoffSub({ row, disabled, onSelect }) {
  return (
    <DropdownMenu.Sub>
      <DropdownMenu.SubTrigger className="ws-row-menu-item" disabled={disabled}>
        {ICONS.handoff} {row.label}
        <IconChevronRight size={13} className="um-chev" />
      </DropdownMenu.SubTrigger>
      <DropdownMenu.Portal>
        <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
          {row.sub.map((s) => s.sub ? (
            <DropdownMenu.Sub key={s.id}>
              <DropdownMenu.SubTrigger className="ws-row-menu-item" disabled={disabled}>
                {s.label}
                <IconChevronRight size={13} className="um-chev" />
              </DropdownMenu.SubTrigger>
              <DropdownMenu.Portal>
                <DropdownMenu.SubContent className="ws-row-menu" sideOffset={4} alignOffset={-4} collisionPadding={8}>
                  {s.sub.map((s2) => (
                    <DropdownMenu.Item
                      key={s2.id}
                      className="ws-row-menu-item"
                      disabled={disabled || !!s2.disabled}
                      onSelect={() => { if (!s2.disabled) onSelect(s2.target); }}
                    >
                      {s2.label}
                    </DropdownMenu.Item>
                  ))}
                </DropdownMenu.SubContent>
              </DropdownMenu.Portal>
            </DropdownMenu.Sub>
          ) : (
            <DropdownMenu.Item
              key={s.id}
              className="ws-row-menu-item"
              disabled={disabled || !!s.disabled}
              onSelect={() => { if (!s.disabled) onSelect(s.target); }}
            >
              {s.label}
            </DropdownMenu.Item>
          ))}
        </DropdownMenu.SubContent>
      </DropdownMenu.Portal>
    </DropdownMenu.Sub>
  );
}
