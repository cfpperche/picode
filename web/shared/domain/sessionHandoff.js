// Cross-CLI session handoff (ADR-0088): pure helpers the Sessions pane and
// its dialog share. Everything here derives from what GET /api/clis
// advertises per CLI (`sessions: {list, read, write, prompt}`) — no CLI id
// is hardcoded, so a new Agent CLI appears the moment the server knows it.

// sessionClis lists the catalog CLIs that have a session source, in
// catalog order. The catalog on the CLIs page is the picker.
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

// Only kinds with a human label are named. The rest are a CLI's own record
// types (claude.bridge-session, codex.token_usage_record) and mean nothing
// to a reader, so they are counted rather than spelled out.
function dropLabel(key) {
  return DROP_LABELS[key] || "";
}

// handoffSessionLabel names the session in the dialog's opening line: its
// own title when it has one, else a short id — a full UUID wraps the line
// and tells the reader nothing.
export function handoffSessionLabel(session) {
  const name = ((session && session.name) || "").trim();
  if (name) return name.slice(0, 60);
  const id = ((session && session.id) || "").trim();
  return id.length > 12 ? id.slice(0, 8) : id;
}

// handoffSummaryLine is the one-line digest of a preview: what travels,
// then what is left behind. Zero counts are omitted.
export function handoffSummaryLine(counts, manifest) {
  const c = counts || {};
  const parts = [];
  if (c.messages) parts.push(c.messages + (c.messages === 1 ? " message" : " messages"));
  if (c.toolCalls) parts.push(c.toolCalls + (c.toolCalls === 1 ? " tool call" : " tool calls"));
  const dropped = (manifest && manifest.dropped) || {};
  const keys = Object.keys(dropped).filter((k) => dropped[k] > 0 && dropLabel(k)).sort((a, b) => dropped[b] - dropped[a] || a.localeCompare(b));
  const named = keys.slice(0, 2).map((k) => dropLabel(k) + " left out (" + dropped[k] + ")");
  const shown = new Set(keys.slice(0, 2));
  const rest = Object.keys(dropped).filter((k) => dropped[k] > 0 && !shown.has(k)).reduce((n, k) => n + dropped[k], 0);
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
