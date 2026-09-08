// The mobile action sheet over the git graph's shared vocabulary (ADR-0096,
// ADR-0095). Pure: what a sheet shows for a target, in an order a thumb can
// read, and which door it may take. Presentation stays in the component.
import { graphActions, repoActions, gateFor } from "@picode/shared/domain/graphActions.js";

// targetFor builds the target a history row or a detail view points at.
// A commit carries the refs drawn on it, so a branch's rows are reachable
// from the commit — a phone has no pill to long-press on its own.
export function targetFor({ commit, refs = [], worktree, uncommitted = false, repo = false }) {
  if (repo) return { kind: "repo" };
  if (worktree) return { kind: "worktree", worktree, uncommitted };
  if (commit && commit.hash) return { kind: "commit", commit, refs: refs.filter((r) => r && r.hash === commit.hash) };
  return null;
}

// sheetMenu is graphActions for every target kind, including the header's.
export function sheetMenu(target, graph, ctx) {
  if (!target) return null;
  if (target.kind === "repo") return repoActions(graph || {}, ctx || {});
  return graphActions(target, graph || {}, ctx || {});
}

// sheetGroups turns the flat, sectioned item list into groups a list can
// render with one heading each. Rows before the first section form an
// unnamed group.
export function sheetGroups(menu) {
  const groups = [];
  let current = { label: "", items: [] };
  for (const item of (menu && menu.items) || []) {
    if (item.kind === "section") {
      if (current.items.length) groups.push(current);
      current = { label: item.label, state: item.state || "", items: [] };
      continue;
    }
    current.items.push(item);
  }
  if (current.items.length) groups.push(current);
  return groups;
}

// doors lists what "Send through" offers: the two terminal doors, then one
// ask per running occupant — agents and pi terminals alike, labelled the
// same, since the reader asks a name, not a kind.
export function doors(occupants = []) {
  const out = [
    { value: "prepare", label: "Prepare in terminal" },
    { value: "run", label: "Run when idle" },
  ];
  for (const o of occupants) {
    if (!o || !o.id || o.running === false) continue;
    out.push({ value: "ask:" + o.id, label: "Ask " + (o.name || "agent"), who: o });
  }
  return out;
}

// gate re-exports the shared rule so the sheet and the desktop dialog can
// never disagree about when a typed phrase is required.
export const gate = gateFor;

// doorHint is the one line under the select, in the words the desktop
// dialog and the six-action sheet already use.
export function doorHint(door, who) {
  if (who) return `${who.name} does it in its own turn, with its own judgment.`;
  if (door === "run") return "Runs when no agent is working here; otherwise it is prepared for you.";
  return "The command waits in your terminal for you to press Enter.";
}
