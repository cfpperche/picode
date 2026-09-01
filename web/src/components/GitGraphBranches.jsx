import { useMemo, useState } from "react";
import * as Popover from "@radix-ui/react-popover";
import { Command } from "cmdk";
import { groupBranches, triggerLabel } from "../lib/gitgraphBranches.js";
import { IconGit, IconCheck, IconRemote } from "./Icons.jsx";

// Multi-select branch picker for the git graph header, inspired by VS Code's
// Git Graph extension. Deliberately a sibling of SearchCombo, not a mode of
// it: unlike every SearchCombo consumer (single pick, closes on select),
// this one stays open across multiple picks and groups Local/Remote — a
// different enough interaction shape that bolting it onto SearchCombo would
// risk its six other single-select consumers.
export default function GitGraphBranches({ refs, selected, showRemotes, onChange, onToggleRemotes }) {
  const [open, setOpen] = useState(false);
  const { local, remote } = useMemo(() => groupBranches(refs, showRemotes), [refs, showRemotes]);
  const label = useMemo(() => triggerLabel(selected), [selected]);
  const selectedSet = useMemo(() => new Set(selected || []), [selected]);

  function toggle(name) {
    onChange(selectedSet.has(name) ? (selected || []).filter((n) => n !== name) : [...(selected || []), name]);
  }

  function row(name, kind) {
    const on = selectedSet.has(name);
    return (
      <Command.Item
        key={kind + ":" + name}
        value={name}
        onSelect={() => toggle(name)}
        className={"cockpit-opt" + (on ? " selected" : "")}
      >
        <span className="gg-branch-check">{on ? <IconCheck /> : null}</span>
        {kind === "remote" ? <IconRemote /> : null}
        <span>{name}</span>
      </Command.Item>
    );
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button type="button" className="btn btn-sm gg-branches-trigger" aria-expanded={open}>
          <IconGit />
          <span className="gg-branches-label">{label}</span>
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          className="cockpit-pop cockpit-combo-pop gg-branches-pop"
          side="bottom"
          align="start"
          sideOffset={6}
          collisionPadding={8}
        >
          <Command label="Branches" loop>
            <Command.Input className="combo-input" placeholder="Filter branches" />
            <Command.List className="combo-list">
              <Command.Empty className="combo-empty">No matches</Command.Empty>
              <Command.Item
                value="show all"
                onSelect={() => { onChange([]); setOpen(false); }}
                className={"cockpit-opt" + (selectedSet.size === 0 ? " selected" : "")}
              >
                <span className="gg-branch-check">{selectedSet.size === 0 ? <IconCheck /> : null}</span>
                <span>Show All</span>
              </Command.Item>
              {local.length ? (
                <Command.Group heading="Local">{local.map((name) => row(name, "head"))}</Command.Group>
              ) : null}
              {remote.length ? (
                <Command.Group heading="Remote">{remote.map((name) => row(name, "remote"))}</Command.Group>
              ) : null}
            </Command.List>
          </Command>
          <label className="gg-branches-remotes">
            <input
              type="checkbox"
              checked={!!showRemotes}
              onChange={(e) => onToggleRemotes(e.target.checked)}
            />
            Show Remote Branches
          </label>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
