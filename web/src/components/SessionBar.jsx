import { useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { Command } from "cmdk";
import { IconSession, IconPlus, IconChat } from "./Icons.jsx";
import { formatSessionCost } from "../lib/statusbar.js";

export default function SessionBar({ sessions, current, onNew, onResume, onRename, onChat, inline, cost }) {
  const [open, setOpen] = useState(false);
  const cur = (sessions || []).find((s) => s.path === current);
  const label = cur ? (cur.name || cur.preview || "Current session") : "Sessions";
  const list = sessions || [];
  const costText = formatSessionCost(cost);

  return (
    <div className={"session-bar" + (inline ? " inline" : "")}>
      <div className="chip-group">
        <Popover.Root open={open} onOpenChange={setOpen}>
          <Popover.Trigger asChild>
            <button type="button" id="session-picker" className="cockpit-chip" title={costText ? "Sessions · " + costText : "Sessions"}>
              <span className="cockpit-chip-icon"><IconSession /></span>
              <span className="cockpit-chip-label">{label}</span>
              {costText ? <span className="session-cost">{costText}</span> : null}
              <span className="session-count">{list.length}</span>
              <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" aria-hidden="true">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </button>
          </Popover.Trigger>
          <Popover.Portal>
            <Popover.Content className="session-pop" side="top" align="start" sideOffset={6} collisionPadding={8}>
              <Command loop>
                <Command.Input className="combo-input" placeholder="Search sessions" />
                <Command.List className="session-list">
                  <Command.Empty className="combo-empty">{list.length === 0 ? "No sessions yet" : "No matches"}</Command.Empty>
                  {list.map((s) => (
                    <Command.Item
                      key={s.path}
                      value={(s.name || "") + " " + (s.preview || "") + " " + s.path}
                      className={"cockpit-opt" + (s.path === current ? " selected" : "")}
                      onSelect={() => { if (s.path !== current) onResume(s.path); setOpen(false); }}
                    >
                      <span className="session-name">{s.name || s.preview || "Untitled"}</span>
                      <span className="combo-hint">{shortDate(s.updatedAt || s.createdAt)}</span>
                      {onRename ? (
                        <span
                          className="session-rename"
                          onPointerDown={(e) => { e.preventDefault(); e.stopPropagation(); }}
                          onClick={(e) => { e.stopPropagation(); onRename(s); }}
                        >Rename</span>
                      ) : null}
                    </Command.Item>
                  ))}
                </Command.List>
              </Command>
            </Popover.Content>
          </Popover.Portal>
        </Popover.Root>
        <button type="button" className="cockpit-chip" onClick={onNew} title="New session">
          <span className="cockpit-chip-icon"><IconPlus /></span>
          <span className="cockpit-chip-label">New</span>
        </button>
      </div>
      {onChat ? (
        <div className="session-bar-end">
          <button type="button" className="cockpit-chip" onClick={onChat} title="Chat">
            <span className="cockpit-chip-icon"><IconChat /></span>
            <span className="cockpit-chip-label">Chat</span>
          </button>
        </div>
      ) : null}
    </div>
  );
}

function shortDate(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}
