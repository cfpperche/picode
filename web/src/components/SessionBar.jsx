import { useState } from "react";
import * as Popover from "@radix-ui/react-popover";

export default function SessionBar({ sessions, current, onNew, onResume }) {
  const [open, setOpen] = useState(false);
  const cur = (sessions || []).find((s) => s.path === current);
  const label = cur ? (cur.name || cur.preview || "Current session") : "Sessions";

  return (
    <div className="session-bar">
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger asChild>
          <button type="button" id="session-picker" className="cockpit-chip" aria-expanded={open} title="Sessions">
            <span className="cockpit-chip-label">{label}</span>
            <span className="session-count">{(sessions || []).length}</span>
          </button>
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content className="cockpit-pop cockpit-combo-pop session-pop" side="bottom" align="start" sideOffset={6} avoidCollisions={false}>
            <div className="session-list">
              {(sessions || []).length === 0 && <div className="combo-empty">No sessions yet</div>}
              {(sessions || []).map((s) => (
                <button
                  key={s.path}
                  type="button"
                  className={"cockpit-opt" + (s.path === current ? " selected" : "")}
                  onClick={() => { if (s.path !== current) onResume(s.path); setOpen(false); }}
                >
                  <span className="session-name">{s.name || s.preview || "Untitled"}</span>
                  <span className="combo-hint">{shortDate(s.updatedAt || s.createdAt)}</span>
                </button>
              ))}
            </div>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
      <button type="button" className="cockpit-chip" onClick={onNew} title="New session">New</button>
    </div>
  );
}

function shortDate(iso) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}
