import { useMemo } from "react";
import { Command } from "cmdk";
import { agentsOf, displayAgentName } from "@picode/shared/domain/tree.js";
import { terminalCli, terminalCliLabel, terminalStatusLabel } from "@picode/shared/domain/terminalCli.js";
import { agentRowStatus, agentStatusLabel } from "@picode/shared/domain/agentStatus.js";
import * as Dialog from "../ResponsiveDialog.jsx";
import { ProviderFace } from "../ProviderFaces.jsx";
import TerminalCliBadge from "../TerminalCliBadge.jsx";
import { IconPin } from "../Icons.jsx";

// PanelPicker — Add panel: a cmdk list grouped by what a panel can be
// (docs/plans/matrix-canvas.md §4.2), with the faces and status words the
// sidebar uses. The same binding at most once per matrix (plan §4.6) starts
// here: what is already on it is not offered. Empty copy per plan §4.8, one
// line and one action per group.
//
// It lists what already exists and never browses: pins come from
// `GET /api/pins` (title and tags, the summary the sidebar shows).
export default function PanelPicker({ open, fleet, pins, onMatrix, workingIds, onPick, onNewPin, onClose }) {
  const { agents, terminals, notes, total, pinTotal } = useMemo(() => {
    const on = onMatrix || new Set();
    const agents = [];
    for (const ws of fleet.workspaces || []) for (const a of agentsOf(ws)) agents.push({ agent: a, ws });
    for (const a of fleet.freeAgents || []) if (a && a.id) agents.push({ agent: a, ws: null });
    const terminals = (fleet.terminals || []).filter((t) => t && t.id);
    const pinList = (Array.isArray(pins) ? pins : []).filter((p) => p && p.id);
    return {
      total: agents.length + terminals.length + pinList.length,
      pinTotal: pinList.length,
      agents: agents.filter((x) => !on.has("agent:" + x.agent.id)),
      terminals: terminals.filter((t) => !on.has("terminal:" + t.id)),
      notes: pinList.filter((p) => !on.has("note:" + p.id)),
    };
  }, [fleet.workspaces, fleet.freeAgents, fleet.terminals, pins, onMatrix]);
  const none = agents.length + terminals.length + notes.length === 0;
  const copy = total === 0 ? "No agents, terminals or pins yet." : "Everything is already on this matrix.";
  return (
    <Dialog.Root open={!!open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dlg-overlay" />
        <Dialog.Content className="dlg dlg-picker" onCloseAutoFocus={(e) => e.preventDefault()}>
          <Dialog.Title className="dlg-title">Add panel</Dialog.Title>
          <Dialog.Description className="sr-only">Pick an agent, a terminal or a pin to show on this matrix.</Dialog.Description>
          {none ? (
            <p className="mx-picker-empty dlg-body" role="status">
              <span>{copy}</span>
              {pinTotal === 0 ? <button type="button" className="btn btn-sm" onClick={onNewPin}>New pin</button> : null}
            </p>
          ) : (
            <Command label="Add panel" loop className="mx-picker">
              <Command.Input className="combo-input" placeholder="Search agents, terminals and pins" autoFocus />
              <Command.List className="combo-list mx-picker-list">
                <Command.Empty className="combo-empty">No matches</Command.Empty>
                {agents.length ? (
                  <Command.Group heading="Agents" className="palette-group">
                    {agents.map(({ agent, ws }) => {
                      const name = displayAgentName(agent, ws);
                      const where = ws ? ws.name : "Free agent";
                      return (
                        <Command.Item key={"agent:" + agent.id} value={name + " " + where + " agent " + agent.id} className="palette-item mx-picker-item" onSelect={() => onPick({ kind: "agent", ref: agent.id })}>
                          <span className="combo-opt-icon"><ProviderFace agent={agent} /></span>
                          <span className="mx-picker-name">{name}</span>
                          <span className="combo-hint">{where} · {agentStatusLabel(agentRowStatus(agent, { workingIds }))}</span>
                        </Command.Item>
                      );
                    })}
                  </Command.Group>
                ) : null}
                {terminals.length ? (
                  <Command.Group heading="Terminals" className="palette-group">
                    {terminals.map((t) => {
                      const cli = terminalCli(t);
                      return (
                        <Command.Item key={"terminal:" + t.id} value={(t.name || "Terminal") + " " + (cli ? terminalCliLabel(cli) : "shell") + " terminal " + t.id} className="palette-item mx-picker-item" onSelect={() => onPick({ kind: "terminal", ref: t.id })}>
                          <span className="combo-opt-icon"><TerminalCliBadge term={t} decorative /></span>
                          <span className="mx-picker-name">{t.name || "Terminal"}</span>
                          <span className="combo-hint">{cli ? terminalCliLabel(cli) : "Shell"} · {terminalStatusLabel(t)}</span>
                        </Command.Item>
                      );
                    })}
                  </Command.Group>
                ) : null}
                {notes.length ? (
                  <Command.Group heading="Pins" className="palette-group">
                    {notes.map((p) => {
                      const tags = (p.tags || []).join(" · ");
                      return (
                        <Command.Item key={"note:" + p.id} value={(p.title || "Untitled note") + " " + tags + " pin note " + p.id} className="palette-item mx-picker-item" onSelect={() => onPick({ kind: "note", ref: p.id })}>
                          <span className="combo-opt-icon"><IconPin size={13} /></span>
                          <span className="mx-picker-name">{p.title || "Untitled note"}</span>
                          {tags ? <span className="combo-hint">{tags}</span> : null}
                        </Command.Item>
                      );
                    })}
                  </Command.Group>
                ) : null}
              </Command.List>
            </Command>
          )}
          {!none && pinTotal === 0 ? (
            <p className="mx-picker-empty" role="status">
              <span>No pins yet.</span>
              <button type="button" className="btn btn-sm" onClick={onNewPin}>New pin</button>
            </p>
          ) : null}
          <div className="dlg-actions">
            <Dialog.Close asChild>
              <button type="button" className="btn btn-ghost btn-sm">Close</button>
            </Dialog.Close>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
