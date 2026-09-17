// Quick launch settings (2026-09-17, docs/benchmarks/2026-09-17-cli-launch-quick-presets.md).
// Per-CLI controls for the choices people actually type as raw flags:
// model, autonomy/approvals, reasoning effort, sandbox. Each control is a
// pure patch over the draft's argument array — generated flags are
// replaced, every argument the user typed stays exactly where it was.
// Values are individual argv entries, never a shell string (ADR-0069).
//
// Flags verified against the vendor docs on 2026-09-17 (pi README CLI
// reference, code.claude.com/docs/en/cli, developers.openai.com/codex/cli/
// reference, opencode.ai/docs/cli). Grok, Hermes, Omp, Muse Code and
// Antigravity stay advanced-only until their installed versions verify the
// same flags — quickSettingsFor returns null and LaunchFields renders
// exactly as before.

const option = (value, label, warn) => (warn ? { value, label, warn } : { value, label });

const CLAUDE_MODES = [
  option("default", "Ask before edits"),
  option("acceptEdits", "Edit automatically"),
  option("plan", "Plan only"),
  option("auto", "Auto"),
  option("dontAsk", "Deny unless allowed"),
  option("bypassPermissions", "Skip all prompts", "Runs without permission prompts. For containers or disposable VMs."),
];

const CODEX_SANDBOX = [
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

export const CLI_QUICK_SETTINGS = {
  pi: [
    { key: "model", label: "Model", type: "text", flags: ["--model"], placeholder: "sonnet:high · openai/gpt-4o", hint: "Same model syntax as the pi CLI." },
    { key: "thinking", label: "Thinking level", type: "select", flags: ["--thinking"], options: PI_THINKING },
  ],
  "claude-code": [
    { key: "model", label: "Model", type: "text", flags: ["--model"], suggestions: ["sonnet", "opus", "haiku", "fable"], placeholder: "sonnet", hint: "Alias or full model id." },
    { key: "permissionMode", label: "Approvals", type: "select", flags: ["--permission-mode"], options: CLAUDE_MODES },
  ],
  codex: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "gpt-5.3-codex", hint: "Overrides the configured model." },
    { key: "reasoning", label: "Reasoning effort", type: "select", flags: ["-c"], kv: "model_reasoning_effort", options: CODEX_EFFORT },
    { key: "sandbox", label: "Sandbox", type: "select", flags: ["--sandbox", "-s"], options: CODEX_SANDBOX },
    { key: "approval", label: "Approvals", type: "select", flags: ["--ask-for-approval", "-a"], options: CODEX_APPROVALS },
  ],
  opencode: [
    { key: "model", label: "Model", type: "text", flags: ["--model", "-m"], placeholder: "provider/model", hint: "Model in provider/model form." },
    { key: "auto", label: "Auto-approve actions", type: "boolean", flags: ["--auto"], hint: "Actions not explicitly denied are approved." },
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

// argLine serializes one argv entry back to a textarea line, quoted when
// the entry is empty, starts with a quote, or carries whitespace.
export function argLine(a) {
  return a === "" || /["\s]/.test(a) ? JSON.stringify(a) : a;
}
