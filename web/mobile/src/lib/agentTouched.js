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

export function setAgentTouched(id, list) {
  if (!id) return;
  const next = JSON.stringify(list || []);
  if (paths.get(id) === next) return;
  paths.set(id, next);
  emit();
}

export function readAgentTouched(id) {
  const raw = paths.get(id);
  if (raw == null) return null;
  try { return JSON.parse(raw); } catch { return null; }
}

function subscribe(listener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function useAgentTouched(id) {
  return useSyncExternalStore(subscribe, () => readAgentTouched(id));
}
