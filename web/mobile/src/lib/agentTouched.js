import { useSyncExternalStore } from "react";

// Session-only record of the paths each agent's edit/write tools named —
// the mobile twin of the desktop's touchedPaths (ADR-0078 "This agent"
// scope). The agent screen writes it from its conversation items; the
// Inspector screen reads it when the anchor is that agent. A deep link
// straight into the Inspector has none: the scope chips simply do not
// render, like the desktop when it has no session paths. A reload
// deliberately starts empty, like agentDrafts.

const paths = new Map();
const listeners = new Set();

function emit() {
  for (const listener of listeners) listener();
}

// The Map holds the parsed array, not a JSON string: useSyncExternalStore
// compares snapshots by identity, so a re-parse per read would hand React
// a new array every render and loop the commit (react.dev/errors/185).
export function setAgentTouched(id, list) {
  if (!id) return;
  const next = [...(list || [])];
  const current = paths.get(id);
  if (current && current.length === next.length && next.every((p, i) => current[i] === p)) return;
  paths.set(id, next);
  emit();
}

export function readAgentTouched(id) {
  return paths.get(id) || null;
}

function subscribe(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useAgentTouched(id) {
  return useSyncExternalStore(subscribe, () => readAgentTouched(id));
}
