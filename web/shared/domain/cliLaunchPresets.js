// Quick launch settings (2026-09-17, docs/benchmarks/2026-09-17-cli-launch-quick-presets.md).
// Per-CLI controls for the choices people actually type as raw flags:
// model, autonomy/approvals, reasoning effort, sandbox. Each control is a
// pure patch over the draft's argument array — generated flags are
// replaced, every argument the user typed stays exactly where it was.
// Values are individual argv entries, never a shell string (ADR-0069).
//
// Flags verified against the installed binaries on 2026-09-17/18 (grok
// 1.0.34 — the official xAI "Grok Build" CLI, reference
// docs.x.ai/build/cli/reference —, Hermes Agent v0.21.3, omp 18.2.4,
// codex-cli 0.154.0 — each exercised with `--help` and a
// `--flag --version` parse check) plus the vendor docs for pi, Claude Code
// and OpenCode. Muse Code and Antigravity stay excluded:
// they have no adapter, so their launch screens are read-only by design.
//
// spec.group marks mutually exclusive controls (a vendor flag conflict):
// applying one clears the others in the group (applyQuickSetting).

const option = (value, label, warn) => (warn ? { value, label, warn } : { value, label });

const PERMISSION_MODES = [
  option("default", "Ask before edits"),
  option("acceptEdits", "Edit automatically"),
  option("plan", "Plan only"),
  option("auto", "Auto"),
  option("dontAsk", "Deny unless allowed"),
  option("bypassPermissions", "Skip all prompts", "Runs without permission prompts. For containers or disposable VMs."),
];

const SANDBOX_MODES = [
  option("read-only", "Read only"),
  option("workspace-write", "Write inside the workspace"),
  option("danger-full-access", "Full access", "No sandbox: full file and network access."),
];

const CODEX_APPROVALS = [
  option("untrusted", "Ask before untrusted commands"),
  option("on-request", "Ask when the agent requests"),
  option("never", "Never ask", "The agent runs commands without asking."),
];

const PI_THINKING = ["off", "minimal", "low", "medium", "high", "xhigh", "max"].map((v) => option(v, v));

const CODEX_EFFORT = ["minimal", "low", "medium", "high", "xhigh"].map((v) => option(v, v));

// Built-in Grok sandbox profiles (docs.x.ai/build/features/sandbox, verified
// 2026-09-18 against the installed 1.0.34 — custom profile names are legal
// too and render as an "as typed" option when present).
const GROK_SANDBOX = [
  option("workspace", "Write inside the workspace"),
  option("devbox", "Cloud devbox"),
  option("read-only", "Read only"),
  option("strict", "Strict (untrusted repositories)"),
];

const HERMES_REASONING = ["none", "minimal", "low", "medium", "high", "xhigh", "max", "ultra"].map((v) => option(v, v));

export const CLI_QUICK_SETTINGS = {
  pi: [
    { key: "model", label: "Model", type: "text", flags: ["--model"], placeholder: "sonnet:high · openai/gpt-4o", hint: "Same model syntax as the pi CLI." },
    { key: "thinking", label: "Thinking level", type: "select", flags: ["--thinking"], options: PI_THINKING },
  ],
  "claude-code": [
    { key: "model", label: "Model", type: "text", flags: ["--model"], suggestions: ["sonnet", "opus", "haiku", "fable"], placeholder: "sonnet", hint: "Alias or full model id." },
    { key: "addDirs", label: "Additional folders", type: "list", flags: ["--add-dir"], placeholder: "/srv/shared", hint: "One directory per line, allowed for tool access." },
    { key: "permissionMode", label: "Approvals", type: "select", flags: ["--permission-mode"], options: PERMISSION_MODES },
  ],
  codex: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "gpt-5.3-codex", hint: "Overrides the configured model." },
    { key: "addDirs", label: "Additional folders", type: "list", flags: ["--add-dir"], placeholder: "/srv/shared", hint: "One directory per line, writable alongside the workspace." },
    { key: "reasoning", label: "Reasoning effort", type: "select", flags: ["-c"], kv: "model_reasoning_effort", options: CODEX_EFFORT },
    { key: "sandbox", label: "Sandbox", type: "select", flags: ["--sandbox", "-s"], options: SANDBOX_MODES, group: "codex" },
    { key: "approval", label: "Approvals", type: "select", flags: ["--ask-for-approval", "-a"], options: CODEX_APPROVALS, group: "codex" },
    { key: "yolo", label: "YOLO", type: "boolean", flags: ["--yolo"], group: "codex", hint: "Skips all approvals and the sandbox (--dangerously-bypass-approvals-and-sandbox)." },
  ],
  grok: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "grok-4.3", hint: "Model id; grok models lists the catalog." },
    { key: "effort", label: "Reasoning effort", type: "text", flags: ["--effort", "--reasoning-effort"], placeholder: "high", hint: "Effort level; depends on the model." },
    { key: "sandbox", label: "Sandbox", type: "select", flags: ["--sandbox"], options: GROK_SANDBOX },
    { key: "permissionMode", label: "Approvals", type: "select", flags: ["--permission-mode"], options: PERMISSION_MODES },
    { key: "alwaysApprove", label: "Auto-approve tools", type: "boolean", flags: ["--always-approve"], hint: "All tool executions are approved without asking (alias --yolo)." },
  ],
  hermes: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "anthropic/claude-sonnet-4.6", hint: "Model override for this invocation." },
    { key: "reasoning", label: "Reasoning effort", type: "select", flags: ["--reasoning"], options: HERMES_REASONING },
    { key: "yolo", label: "YOLO", type: "boolean", flags: ["--yolo"], hint: "Bypasses approval prompts for dangerous commands." },
  ],
  opencode: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "provider/model", hint: "Model in provider/model form." },
    { key: "auto", label: "Auto-approve actions", type: "boolean", flags: ["--auto"], hint: "Actions not explicitly denied are approved." },
  ],
  omp: [
    { key: "model", label: "Model", type: "text", flags: ["--model"], placeholder: "opus · openai/gpt-5.2", hint: "Fuzzy match, same as the omp CLI." },
    { key: "addDirs", label: "Additional folders", type: "list", flags: ["--add-dir"], placeholder: "/srv/shared", hint: "One directory per line, beyond the working directory." },
  ],
};

// quickSettingsFor returns the control specs for a CLI id, or null when the
// CLI has no verified quick controls.
export function quickSettingsFor(cliId) {
  return CLI_QUICK_SETTINGS[cliId] || null;
}

// matchFlag reports how arg matches one of the spec's flags: joined
// ("--model=x") or bare ("--model" / "-m"). The short form never joins.
function matchFlag(arg, flags) {
  if (flags.includes(arg)) return { bare: true };
  const eq = arg.indexOf("=");
  if (eq > 1 && flags.includes(arg.slice(0, eq))) return { joined: arg.slice(eq + 1) };
  return null;
}

const kvMatch = (arg, next, spec) => !!spec.kv && spec.flags.includes(arg) && next !== undefined && next.startsWith(spec.kv + "=");

// readQuickValue reads the control's current value out of argv. A flag at
// the end with no value reads as set-but-empty; an unknown select value
// comes back verbatim so the UI can show it honestly. A kv spec only
// matches its own key — another `-c key=value` stays untouched.
export function readQuickValue(args, spec) {
  for (let i = 0; i < args.length; i++) {
    if (spec.kv) {
      if (kvMatch(args[i], args[i + 1], spec)) return args[i + 1].slice(spec.kv.length + 1);
      continue;
    }
    const m = matchFlag(args[i], spec.flags);
    if (!m) continue;
    if (spec.type === "boolean") return "on";
    if (m.joined !== undefined) return m.joined;
    const next = args[i + 1];
    if (next !== undefined && next.startsWith("-")) return "";
    return next ?? "";
  }
  return "";
}

// applyQuickValue returns argv with the spec's generated flag replaced by
// value ("" or undefined removes it). The replacement takes the old flag's
// position, so surrounding arguments keep their order; an absent flag
// appends at the end.
export function applyQuickValue(args, spec, value) {
  const out = [];
  let insertAt = -1;
  for (let i = 0; i < args.length; i++) {
    if (kvMatch(args[i], args[i + 1], spec)) { if (insertAt < 0) insertAt = out.length; i++; continue; }
    const m = matchFlag(args[i], spec.flags);
    if (!m) { out.push(args[i]); continue; }
    if (insertAt < 0) insertAt = out.length;
    if (spec.kv) { out.push(args[i]); continue; }
    if (spec.type !== "boolean" && m.joined === undefined) {
      const next = args[i + 1];
      if (next !== undefined && !next.startsWith("-")) i++;
    }
  }
  if (value) {
    const pair = [spec.flags[0]];
    if (spec.type !== "boolean") pair.push(spec.kv ? `${spec.kv}=${value}` : value);
    out.splice(insertAt < 0 ? out.length : insertAt, 0, ...pair);
  }
  return out;
}

// applyQuickSetting applies one control's value, first clearing the other
// controls in its exclusivity group — a documented vendor flag conflict,
// e.g. codex --yolo vs --sandbox / --ask-for-approval. An emptied control
// never clears its group (removal is not a choice that conflicts).
export function applyQuickSetting(args, specs, spec, value) {
  let next = args;
  if (spec.group && value) {
    for (const other of specs) {
      if (other !== spec && other.group === spec.group) next = applyQuickValue(next, other, "");
    }
  }
  return applyQuickValue(next, spec, value);
}

// argLine serializes one argv entry back to a textarea line, quoted when
// the entry is empty, starts with a quote, or carries whitespace.
export function argLine(a) {
  return a === "" || /["\s]/.test(a) ? JSON.stringify(a) : a;
}

// readQuickList returns every value of a repeatable flag (spec.type
// "list"), in argv order. Joined forms count; a bare flag with no value
// contributes nothing.
export function readQuickList(args, spec) {
  const out = [];
  for (let i = 0; i < args.length; i++) {
    const m = matchFlag(args[i], spec.flags);
    if (!m) continue;
    if (m.joined !== undefined) { out.push(m.joined); continue; }
    const next = args[i + 1];
    if (next !== undefined && !next.startsWith("-")) { out.push(next); i++; }
  }
  return out;
}

// applyQuickList returns argv with every pair of the repeatable flag
// replaced by values (one pair per value, in order). The block takes the
// first old pair's position, so surrounding arguments keep their order; a
// fresh list appends at the end and an empty list removes the flag.
export function applyQuickList(args, spec, values) {
  const out = [];
  let insertAt = -1;
  for (let i = 0; i < args.length; i++) {
    const m = matchFlag(args[i], spec.flags);
    if (!m) { out.push(args[i]); continue; }
    if (insertAt < 0) insertAt = out.length;
    if (m.joined === undefined) {
      const next = args[i + 1];
      if (next !== undefined && !next.startsWith("-")) i++;
    }
  }
  const pairs = [];
  for (const v of values) {
    if (!v) continue;
    pairs.push(spec.flags[0], v);
  }
  if (pairs.length) out.splice(insertAt < 0 ? out.length : insertAt, 0, ...pairs);
  return out;
}
