// Cross-CLI session handoff (ADR-0088): pure helpers the Sessions tab and
// its dialog share. Everything here derives from what GET /api/clis
// advertises per CLI (`sessions: {list, read, write, prompt}`) — no CLI id
// is hardcoded, so a new Agent CLI appears the moment the server knows it.

// sessionClis lists the catalog CLIs that have a session source, in
// catalog order — the picker of the Sessions tab.
export function sessionClis(clis) {
  return (clis || []).filter((c) => c && c.sessions && c.sessions.list).map((c) => c.id);
}

// handoffModes is the server's decision table, client side, for one
// (source, target) pair: which modes a handoff can run in. Empty when the
// source cannot be read or the target can receive nothing.
export function handoffModes(sourceCap, targetCap) {
  if (!sourceCap || !sourceCap.read || !targetCap) return [];
  const modes = [];
  if (targetCap.write) modes.push("native");
  if (targetCap.prompt) modes.push("brief");
  return modes;
}

// handoffTargets lists where a session of sourceCli can continue:
// every other CLI with at least one mode, with whether it is installed.
export function handoffTargets(clis, sourceCli) {
  const list = clis || [];
  const source = list.find((c) => c.id === sourceCli);
  if (!source) return [];
  return list
    .filter((c) => c.id !== sourceCli)
    .map((c) => ({ id: c.id, name: c.name, installed: !!c.installed, modes: handoffModes(source.sessions, c.sessions) }))
    .filter((t) => t.modes.length > 0);
}

const DROP_LABELS = {
  thinking: "thinking",
  context: "injected instructions",
  "tool_result.synthesized": "unanswered tool calls",
  "tool_result.orphan": "stray tool results",
  image: "images",
  "reasoning.encrypted": "encrypted reasoning",
};

function dropLabel(key) {
  if (DROP_LABELS[key]) return DROP_LABELS[key];
  const tail = key.split(".").pop();
  return tail.replace(/[-_]/g, " ");
}

// handoffSummaryLine is the one-line digest of a preview: what travels,
// then what is left behind. Zero counts are omitted.
export function handoffSummaryLine(counts, manifest) {
  const c = counts || {};
  const parts = [];
  if (c.messages) parts.push(c.messages + (c.messages === 1 ? " message" : " messages"));
  if (c.toolCalls) parts.push(c.toolCalls + (c.toolCalls === 1 ? " tool call" : " tool calls"));
  const dropped = (manifest && manifest.dropped) || {};
  const keys = Object.keys(dropped).filter((k) => dropped[k] > 0).sort((a, b) => dropped[b] - dropped[a] || a.localeCompare(b));
  const named = keys.slice(0, 2).map((k) => dropLabel(k) + " left out (" + dropped[k] + ")");
  const rest = keys.slice(2).reduce((n, k) => n + dropped[k], 0);
  if (rest) named.push(rest + (rest === 1 ? " other item skipped" : " other items skipped"));
  return [...parts, ...named].join(" · ");
}

// handoffRequest builds the exact body the handoff endpoints accept —
// the server rejects unknown keys, so this is the single place that
// knows the shape.
export function handoffRequest(session, form, extra = {}) {
  return {
    id: session.id || "",
    path: session.path || "",
    cwd: session.cwd || "",
    to: form.to,
    mode: form.mode || "",
    window: form.window || "recent",
    tools: form.tools || "native",
    force: !!extra.force,
    workspaceId: session.workspaceId || "",
  };
}

// lineageBadges renders a row's handoff links as short labels.
export function lineageBadges(handoff, cliNames = {}) {
  const out = [];
  if (!handoff) return out;
  const name = (id) => cliNames[id] || id;
  if (handoff.from) out.push({ kind: "from", label: "from " + name(handoff.from.cli), ref: handoff.from });
  for (const to of handoff.to || []) out.push({ kind: "to", label: "continued in " + name(to.cli), ref: to });
  return out;
}
