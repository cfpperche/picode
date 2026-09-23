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

// handoffLandings lists where a handoff to this target can open: a CLI
// terminal always; a CLI that is also the platform's managed agent (the
// server's `sessions.agent`, pi today) also lands as a stopped agent in
// the app — the second menu level. Agent first: it is the app-native
// surface and needs no installed CLI.
export function handoffLandings(targetCap) {
  if (!targetCap) return [];
  const landings = ["terminal"];
  if (targetCap.agent) landings.unshift("agent");
  return landings;
}

// handoffTargets lists where a session of sourceCli can continue:
// every other CLI with at least one mode, with whether it is installed
// and where it can land.
export function handoffTargets(clis, sourceCli) {
  const list = clis || [];
  const source = list.find((c) => c.id === sourceCli);
  if (!source) return [];
  return list
    .filter((c) => c.id !== sourceCli)
    .map((c) => ({ id: c.id, name: c.name, installed: !!c.installed, modes: handoffModes(source.sessions, c.sessions), landings: handoffLandings(c.sessions) }))
    .filter((t) => t.modes.length > 0);
}

// landingLabel names a landing in the menus and the dialog.
export function landingLabel(target, landing) {
  return landing === "agent" ? target.name + " agent · chat" : target.name + " agent · terminal";
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
    landing: form.landing || "",
    window: form.window || "recent",
    tools: form.tools || "native",
    force: !!extra.force,
    workspaceId: session.workspaceId || "",
  };
}

// sessionFromTerminal maps an Agent CLI terminal's pin (ADR-0084) into the
// session object the handoff endpoints accept. Null when this terminal has
// no recorded conversation yet — Continue is then dropped, same as a
// missing Resume button.
export function sessionFromTerminal(term) {
  const ls = term && term.lastSession;
  const id = String((ls && ls.sessionId) || "").trim();
  const path = String((ls && ls.path) || "").trim();
  // Path-only is enough for the handoff HTTP route ("session id or path
  // required"). Agent TUI pins often have sessionPath and no sessionId.
  if (!id && !path) return null;
  return {
    id,
    path,
    cwd: String(ls.cwd || (term && term.cwd) || "").trim(),
    name: String(ls.name || (term && term.name) || "").trim(),
    workspaceId: (term && term.workspaceId) || "",
  };
}

// agentPin is the Continue-in record for a managed agent's TUI pane.
// Workspace agents have no workPath; they inherit the workspace folder
// (store.AgentCwd). Path-only is a valid pin.
export function agentPin(agent, workspace, paneCwd) {
  if (!agent) return null;
  const path = String(agent.sessionPath || "").trim();
  if (!path) return null;
  const cwd = String(agent.workPath || (workspace && workspace.path) || paneCwd || "").trim();
  return {
    id: agent.id,
    name: agent.name,
    cwd,
    workspaceId: agent.workspaceId || (workspace && workspace.id) || "",
    lastSession: {
      cli: "pi",
      path,
      cwd,
    },
  };
}

export function terminalHandoffSourceCli(term) {
  return String((term && term.lastSession && term.lastSession.cli) || "").trim();
}

// terminalCanFork: the terminal's pinned conversation can be forked into a
// new agent of the same CLI — the CLI advertises a command-line fork
// (sessions.fork in GET /api/clis) and the pin names a session.
export function terminalCanFork(term, clis) {
  if (!sessionFromTerminal(term)) return false;
  const sourceCli = terminalHandoffSourceCli(term);
  const cli = (clis || []).find((c) => c && c.id === sourceCli);
  return !!(cli && cli.sessions && cli.sessions.fork);
}

// terminalHandoffMenu is the Continue-in submenu for a terminal row or pane,
// or null when the row cannot act (no pin, unread source, no targets).
// Uninstalled targets stay listed with the reason in the label.
export function terminalHandoffMenu(term, clis) {
  if (!sessionFromTerminal(term)) return null;
  const sourceCli = terminalHandoffSourceCli(term);
  if (!sourceCli) return null;
  const targets = handoffTargets(clis, sourceCli);
  if (!targets.length) return null;
  return {
    id: "handoff",
    label: "Continue in…",
    title: "Open this conversation in another CLI.",
    icon: "ask",
    sub: targets.map((t) => {
      const item = {
        id: "handoff:" + t.id,
        label: t.name,
        target: t,
      };
      if (t.landings.length > 1) {
        // Two surfaces: a second menu level picks where it opens. Each
        // child carries its own target copy with the landing baked in, so
        // the handlers stay one shape on every surface.
        item.label = t.name + "…";
        item.title = "Choose where this conversation continues in " + t.name + ".";
        item.sub = t.landings.map((l) => ({
          id: "handoff:" + t.id + ":" + l,
          label: landingLabel(t, l) + (l === "terminal" && !t.installed ? " (not installed)" : ""),
          title: l === "agent"
            ? "Continue as a stopped " + t.name + " agent in this app."
            : t.installed ? "Continue in a " + t.name + " terminal." : t.name + " is not installed.",
          disabled: l === "terminal" && !t.installed,
          target: { ...t, landing: l },
        }));
        return item;
      }
      const only = t.landings[0] || "terminal";
      item.label = t.installed ? t.name : t.name + " (not installed)";
      item.title = t.installed ? "Continue this conversation in " + t.name + "." : t.name + " is not installed.";
      item.disabled = !t.installed;
      item.target = { ...t, landing: only };
      return item;
    }),
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
