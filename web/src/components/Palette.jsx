import { useMemo } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";

export default function Palette({ open, workspaces, onClose, onRun }) {
  const actions = useMemo(() => buildActions(workspaces), [workspaces]);
  const groups = useMemo(() => {
    const m = new Map();
    for (const a of actions) {
      if (!m.has(a.group)) m.set(a.group, []);
      m.get(a.group).push(a);
    }
    return [...m.entries()];
  }, [actions]);

  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="palette-root" />
        <Dialog.Content className="palette" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="sr-only">Command palette</Dialog.Title>
          <Command loop>
            <Command.Input className="palette-input" placeholder="Switch agent, run, stop…" />
            <Command.List className="palette-list">
              <Command.Empty className="palette-empty">No matches</Command.Empty>
              {groups.map(([group, items]) => (
                <Command.Group key={group} heading={group} className="palette-group">
                  {items.map((a) => (
                    <Command.Item
                      key={a.id}
                      value={a.label + " " + a.group}
                      className="palette-item"
                      onSelect={() => { onClose(); onRun(a); }}
                    >
                      <span className="palette-label">{a.label}</span>
                      <span className="palette-group">{a.group}</span>
                    </Command.Item>
                  ))}
                </Command.Group>
              ))}
            </Command.List>
          </Command>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function buildActions(workspaces) {
  const out = [
    { id: "settings", label: "Settings", group: "app", kind: "settings" },
    { id: "preferences", label: "Preferences", group: "app", kind: "preferences" },
    { id: "system", label: "System", group: "app", kind: "system" },
    { id: "providers", label: "Providers", group: "app", kind: "providers" },
    { id: "mcps", label: "MCPs", group: "app", kind: "mcps" },
    { id: "packages", label: "Packages", group: "app", kind: "packages" },
    { id: "devices", label: "Devices", group: "app", kind: "devices" },
  ];
  for (const ws of workspaces) {
    const mode = ws.agent ? ws.agent.mode : "stopped";
    out.push({ id: "open-" + ws.id, label: "Open " + ws.name, group: ws.name, kind: "open", wsId: ws.id });
    if (mode === "stopped") {
      out.push({ id: "run-" + ws.id, label: "Run " + ws.name, group: ws.name, kind: "run", wsId: ws.id });
      out.push({ id: "term-" + ws.id, label: "Open terminal · " + ws.name, group: ws.name, kind: "term", wsId: ws.id });
    } else {
      out.push({ id: "stop-" + ws.id, label: "Stop " + ws.name, group: ws.name, kind: "stop", wsId: ws.id });
    }
  }
  return out;
}
