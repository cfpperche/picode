import { useMemo } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Command } from "cmdk";

export default function Palette({ open, workspaces, apps, snips, agentId, onClose, onRun, focusable }) {
  const actions = useMemo(() => buildActions(workspaces, apps, focusable, snips, agentId), [workspaces, apps, focusable, snips, agentId]);
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

function buildActions(workspaces, apps, focusable, snips, agentId) {
  const out = [
    { id: "whats-new", label: "What’s new in PiCode", group: "app", kind: "whats-new" },
    { id: "settings", label: "CLI settings", group: "Agent CLIs", kind: "settings" },
    { id: "inspector", label: "Toggle inspector", group: "app", kind: "inspector" },
    // Only where the mode can live: a page route has no tab strip to
    // reveal, so the row would run and undo itself (focusMode.js).
    ...(focusable ? [{ id: "fullscreen", label: "Fullscreen", group: "app", kind: "fullscreen" }] : []),
    { id: "preferences", label: "Preferences", group: "app", kind: "preferences" },
    { id: "clis", label: "Agent CLIs", group: "app", kind: "clis" },
    { id: "cli-new", label: "New agent", group: "app", kind: "cli-new" },
    { id: "system", label: "System", group: "app", kind: "system" },
    { id: "providers", label: "Providers", group: "app", kind: "providers" },
    { id: "mcps", label: "Connectors", group: "app", kind: "mcps" },
    { id: "integrations", label: "Webhooks", group: "app", kind: "integrations" },
    { id: "packages", label: "Packages", group: "app", kind: "packages" },
    { id: "devices", label: "Devices", group: "app", kind: "devices" },
    { id: "automations", label: "Automations", group: "app", kind: "automations" },
    { id: "missions", label: "Missions", group: "app", kind: "missions" },
    { id: "snippets", label: "Snippets", group: "app", kind: "snippets" },
    { id: "outcomes", label: "Outcomes", group: "app", kind: "outcomes" },
  ];
  if (agentId) {
    for (const s of (snips || []).slice(0, 10)) {
      out.push({
        id: "snip-run-" + s.id,
        label: "Send snippet: " + (s.title || s.slug),
        group: "snippets",
        kind: "snip-run",
        snipId: s.id,
        target: { type: "agent", id: agentId },
      });
    }
  }
  for (const a of apps || []) {
    out.push({ id: "app-" + a.id, label: "Open " + a.name, group: "apps", kind: "app", appId: a.id });
  }
  for (const ws of workspaces) {
    out.push({ id: "cli-new-" + ws.id, label: "New agent · " + ws.name, group: ws.name, kind: "cli-new", wsId: ws.id });
    const mode = ws.agent ? ws.agent.mode : "stopped";
    out.push({ id: "open-" + ws.id, label: "Open " + ws.name, group: ws.name, kind: "open", wsId: ws.id });
    out.push({ id: "files-" + ws.id, label: "Files · " + ws.name, group: ws.name, kind: "files", wsId: ws.id, wsName: ws.name });
    if (mode === "stopped") {
      out.push({ id: "run-" + ws.id, label: "Run " + ws.name, group: ws.name, kind: "run", wsId: ws.id });
      out.push({ id: "term-" + ws.id, label: "Open terminal · " + ws.name, group: ws.name, kind: "term", wsId: ws.id });
    } else {
      out.push({ id: "stop-" + ws.id, label: "Stop " + ws.name, group: ws.name, kind: "stop", wsId: ws.id });
    }
  }
  return out;
}
